package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"arabica/internal/atproto"
	"arabica/internal/database/boltstore"
	"arabica/internal/feed"
	"arabica/internal/handlers"
	"arabica/internal/routing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
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

	log.Info().Msg("Starting Arabica Coffee Tracker")

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

	// Register users in the feed when they authenticate
	// This ensures users are added to the feed even if they had an existing session
	oauthManager.SetOnAuthSuccess(func(did string) {
		feedRegistry.Register(did)
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

	// Start HTTP server
	log.Info().
		Str("address", "0.0.0.0:"+port).
		Str("url", "http://localhost:"+port).
		Bool("secure_cookies", secureCookies).
		Str("database", dbPath).
		Msg("Starting HTTP server")

	if err := http.ListenAndServe("0.0.0.0:"+port, handler); err != nil {
		log.Fatal().Err(err).Msg("Server failed to start")
	}
}
