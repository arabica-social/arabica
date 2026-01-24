package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"arabica/internal/atproto"
	"arabica/internal/database/boltstore"
	"arabica/internal/feed"
	"arabica/internal/firehose"
	"arabica/internal/handlers"
	"arabica/internal/routing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Parse command-line flags
	useFirehose := flag.Bool("firehose", false, "Enable firehose-based feed (Jetstream consumer)")
	knownDIDsFile := flag.String("known-dids", "", "Path to file containing DIDs to backfill on startup (one per line)")
	flag.Parse()

	// Configure zerolog
	// Set log level from environment (default: info)
	logLevel := os.Getenv("LOG_LEVEL")
	switch logLevel {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info", "":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Use pretty console logging in development, JSON in production
	if os.Getenv("LOG_FORMAT") == "json" {
		// Production: JSON logs
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		// Development: pretty console logs
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		})
	}

	log.Info().Bool("firehose", *useFirehose).Msg("Starting Arabica Coffee Tracker")

	// Get port from env or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "18910"
	}

	// Get public root URL for reverse proxy deployments
	// This allows the server to be accessed via a different URL than it's running on
	// e.g., SERVER_PUBLIC_URL=https://arabica.example.com when behind a reverse proxy
	publicURL := os.Getenv("SERVER_PUBLIC_URL")

	// Initialize BoltDB store for persistent sessions and feed registry
	dbPath := os.Getenv("ARABICA_DB_PATH")
	if dbPath == "" {
		// Default to XDG data directory or home directory for development
		// This avoids issues when running from read-only locations (e.g., nix run)
		dataDir := os.Getenv("XDG_DATA_HOME")
		if dataDir == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				log.Fatal().Err(err).Msg("Failed to get home directory")
			}
			dataDir = filepath.Join(home, ".local", "share")
		}
		dbPath = filepath.Join(dataDir, "arabica", "arabica.db")
	}

	store, err := boltstore.Open(boltstore.Options{
		Path: dbPath,
	})
	if err != nil {
		log.Fatal().Err(err).Str("path", dbPath).Msg("Failed to open database")
	}
	defer store.Close()

	log.Info().Str("path", dbPath).Msg("Database opened")

	// Get specialized stores
	sessionStore := store.SessionStore()
	feedStore := store.FeedStore()

	// Initialize OAuth manager with persistent session store
	// For local development, localhost URLs trigger special localhost mode in indigo
	clientID := os.Getenv("OAUTH_CLIENT_ID")
	redirectURI := os.Getenv("OAUTH_REDIRECT_URI")

	if clientID == "" && redirectURI == "" {
		// Use public URL if set, otherwise localhost defaults for development
		if publicURL != "" {
			redirectURI = publicURL + "/oauth/callback"
			clientID = publicURL + "/oauth-client-metadata.json"
			log.Info().
				Str("public_url", publicURL).
				Msg("Using public URL for OAuth (reverse proxy mode)")
		} else {
			redirectURI = fmt.Sprintf("http://127.0.0.1:%s/oauth/callback", port)
			clientID = "" // Empty triggers localhost mode
			log.Info().Msg("Using localhost OAuth mode (for development)")
		}
	}

	oauthManager, err := atproto.NewOAuthManager(clientID, redirectURI, sessionStore)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize OAuth")
	}

	// Initialize feed registry with persistent store
	// This loads existing registered DIDs from the database
	feedRegistry := feed.NewPersistentRegistry(feedStore)
	feedService := feed.NewService(feedRegistry)

	log.Info().
		Int("registered_users", feedRegistry.Count()).
		Msg("Feed service initialized with persistent registry")

	// Setup context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Initialize firehose consumer if enabled
	var firehoseConsumer *firehose.Consumer
	if *useFirehose {
		// Determine feed index path
		feedIndexPath := os.Getenv("ARABICA_FEED_INDEX_PATH")
		if feedIndexPath == "" {
			dataDir := os.Getenv("XDG_DATA_HOME")
			if dataDir == "" {
				home, err := os.UserHomeDir()
				if err != nil {
					log.Fatal().Err(err).Msg("Failed to get home directory for feed index")
				}
				dataDir = filepath.Join(home, ".local", "share")
			}
			feedIndexPath = filepath.Join(dataDir, "arabica", "feed-index.db")
		}

		// Create firehose config
		firehoseConfig := firehose.DefaultConfig()
		firehoseConfig.IndexPath = feedIndexPath

		// Parse profile cache TTL from env if set
		if ttlStr := os.Getenv("ARABICA_PROFILE_CACHE_TTL"); ttlStr != "" {
			if ttl, err := time.ParseDuration(ttlStr); err == nil {
				firehoseConfig.ProfileCacheTTL = int64(ttl.Seconds())
			}
		}

		// Create feed index
		feedIndex, err := firehose.NewFeedIndex(feedIndexPath, time.Duration(firehoseConfig.ProfileCacheTTL)*time.Second)
		if err != nil {
			log.Fatal().Err(err).Str("path", feedIndexPath).Msg("Failed to create feed index")
		}

		log.Info().Str("path", feedIndexPath).Msg("Feed index opened")

		// Create and start consumer
		firehoseConsumer = firehose.NewConsumer(firehoseConfig, feedIndex)
		firehoseConsumer.Start(ctx)

		// Wire up the feed service to use the firehose index
		adapter := firehose.NewFeedIndexAdapter(feedIndex)
		feedService.SetFirehoseIndex(adapter)

		log.Info().Msg("Firehose consumer started")

		// Log known DIDs from database (DIDs discovered via firehose)
		if knownDIDsFromDB, err := feedIndex.GetKnownDIDs(); err == nil {
			if len(knownDIDsFromDB) > 0 {
				log.Info().
					Int("count", len(knownDIDsFromDB)).
					Strs("dids", knownDIDsFromDB).
					Msg("Known DIDs from firehose index")
			} else {
				log.Info().Msg("No known DIDs in firehose index yet (will populate as events arrive)")
			}
		} else {
			log.Warn().Err(err).Msg("Failed to retrieve known DIDs from firehose index")
		}

		// Backfill registered users and known DIDs in background
		go func() {
			time.Sleep(5 * time.Second) // Wait for initial connection

			// Collect all DIDs to backfill
			didsToBackfill := make(map[string]struct{})

			// Add registered users
			for _, did := range feedRegistry.List() {
				didsToBackfill[did] = struct{}{}
			}

			// Add DIDs from known-dids file if provided
			if *knownDIDsFile != "" {
				knownDIDs, err := loadKnownDIDs(*knownDIDsFile)
				if err != nil {
					log.Warn().Err(err).Str("file", *knownDIDsFile).Msg("Failed to load known DIDs file")
				} else {
					for _, did := range knownDIDs {
						didsToBackfill[did] = struct{}{}
					}
					log.Info().
						Int("count", len(knownDIDs)).
						Str("file", *knownDIDsFile).
						Strs("dids", knownDIDs).
						Msg("Loaded known DIDs from file")
				}
			}

			// Backfill all collected DIDs
			successCount := 0
			for did := range didsToBackfill {
				if err := firehoseConsumer.BackfillDID(ctx, did); err != nil {
					log.Warn().Err(err).Str("did", did).Msg("Failed to backfill user")
				} else {
					successCount++
				}
			}
			log.Info().Int("total", len(didsToBackfill)).Int("success", successCount).Msg("Backfill complete")
		}()
	} else {
		// Firehose not enabled, log registered users from feed registry
		registeredDIDs := feedRegistry.List()
		if len(registeredDIDs) > 0 {
			log.Info().
				Int("count", len(registeredDIDs)).
				Strs("dids", registeredDIDs).
				Msg("Registered users in feed registry (polling mode)")
		} else {
			log.Info().Msg("No registered users in feed registry yet (users register on first login)")
		}
	}

	// Register users in the feed when they authenticate
	// This ensures users are added to the feed even if they had an existing session
	oauthManager.SetOnAuthSuccess(func(did string) {
		feedRegistry.Register(did)
		// If firehose is enabled, backfill the user's records
		if firehoseConsumer != nil {
			go func() {
				if err := firehoseConsumer.BackfillDID(context.Background(), did); err != nil {
					log.Warn().Err(err).Str("did", did).Msg("Failed to backfill new user")
				}
			}()
		}
	})

	if clientID == "" {
		log.Info().
			Str("mode", "localhost development").
			Str("redirect_uri", redirectURI).
			Msg("OAuth configured")
	} else {
		log.Info().
			Str("mode", "public").
			Str("client_id", clientID).
			Str("redirect_uri", redirectURI).
			Msg("OAuth configured")
	}

	// Initialize atproto client
	atprotoClient := atproto.NewClient(oauthManager)
	log.Info().Msg("ATProto client initialized")

	// Initialize session cache for in-memory caching of user data
	sessionCache := atproto.NewSessionCache()
	stopCacheCleanup := sessionCache.StartCleanupRoutine(10 * time.Minute)
	defer stopCacheCleanup()
	log.Info().Msg("Session cache initialized with background cleanup")

	// Determine if we should use secure cookies (default: false for development)
	// Set SECURE_COOKIES=true in production with HTTPS
	secureCookies := os.Getenv("SECURE_COOKIES") == "true"

	// Initialize handlers with all dependencies via constructor injection
	h := handlers.NewHandler(
		oauthManager,
		atprotoClient,
		sessionCache,
		feedService,
		feedRegistry,
		handlers.Config{
			SecureCookies: secureCookies,
		},
	)

	// Setup router with middleware
	handler := routing.SetupRouter(routing.Config{
		Handlers:     h,
		OAuthManager: oauthManager,
		Logger:       log.Logger,
	})

	// Create HTTP server
	server := &http.Server{
		Addr:    "0.0.0.0:" + port,
		Handler: handler,
	}

	// Start HTTP server in goroutine
	go func() {
		log.Info().
			Str("address", "0.0.0.0:"+port).
			Str("url", "http://localhost:"+port).
			Bool("secure_cookies", secureCookies).
			Bool("firehose", *useFirehose).
			Str("database", dbPath).
			Msg("Starting HTTP server")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	// Wait for shutdown signal
	<-sigCh
	log.Info().Msg("Shutdown signal received")

	// Stop firehose consumer first
	if firehoseConsumer != nil {
		log.Info().Msg("Stopping firehose consumer...")
		firehoseConsumer.Stop()
	}

	// Graceful shutdown of HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server shutdown error")
	}

	log.Info().Msg("Server stopped")
}

// loadKnownDIDs reads a file containing DIDs (one per line) and returns them as a slice.
// Lines starting with # are treated as comments and ignored.
// Empty lines and whitespace are trimmed.
func loadKnownDIDs(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var dids []string
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Basic DID validation (must start with "did:")
		if !strings.HasPrefix(line, "did:") {
			log.Warn().
				Str("file", filePath).
				Int("line", lineNum).
				Str("content", line).
				Msg("Skipping invalid DID (must start with 'did:')")
			continue
		}

		dids = append(dids, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return dids, nil
}
