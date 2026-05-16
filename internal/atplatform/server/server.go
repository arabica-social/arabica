// Package server holds the bootstrap that constructs every shared
// dependency (database, OAuth, firehose, handlers, router) and serves
// HTTP until shutdown. cmd/arabica/main.go and cmd/oolong/main.go both
// call Run after building their respective *domain.App, so a bug fix
// in the boot sequence benefits both binaries with no duplication.
package server

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/backup"
	"tangled.org/arabica.social/arabica/internal/database/boltstore"
	"tangled.org/arabica.social/arabica/internal/database/sqlitestore"
	"tangled.org/arabica.social/arabica/internal/email"
	"tangled.org/arabica.social/arabica/internal/feed"
	"tangled.org/arabica.social/arabica/internal/firehose"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/metrics"
	"tangled.org/arabica.social/arabica/internal/moderation"
	"tangled.org/arabica.social/arabica/internal/routing"
	"tangled.org/arabica.social/arabica/internal/tracing"
	"tangled.org/arabica.social/arabica/internal/web/assets"
	"tangled.org/pdewey.com/atp"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
)

// Options collects the per-invocation knobs that don't live in the App
// or in environment variables. CLI flags map into this; everything else
// (paths, ports, SMTP creds) reads from env in Run so each binary can
// override without forcing main to enumerate every knob.
type Options struct {
	// KnownDIDsPath is the optional file of DIDs to backfill at startup
	// (one per line, # comments allowed). Empty skips file-based backfill.
	KnownDIDsPath string

	// DefaultPort is the public HTTP port used when <APP>_PORT is unset.
	// Empty falls back to "18910".
	DefaultPort string

	// DefaultMetricsPort is the localhost metrics port used when
	// <APP>_METRICS_PORT is unset. Empty falls back to "9101".
	DefaultMetricsPort string
}

// tracingOnce ensures the global OpenTelemetry provider is initialised
// at most once per process, even when Run is invoked concurrently for
// multiple apps from the unified server entrypoint.
var tracingOnce sync.Once

// Run constructs the full server stack for app and serves until ctx is
// cancelled or a SIGINT/SIGTERM arrives at the signal handler the
// caller wires up. Returns nil on graceful shutdown, non-nil on a
// fatal startup error or an HTTP server failure.
//
// Environment variables are scoped by app.Name uppercased: the arabica
// binary reads ARABICA_DATA_DIR, ARABICA_MODERATORS_CONFIG,
// ARABICA_PROFILE_CACHE_TTL. oolong reads OOLONG_*. Universal env vars
// (PORT, SERVER_PUBLIC_URL, SECURE_COOKIES, OAUTH_*, SMTP_*, ADMIN_EMAIL,
// PDS_ADMIN_*, METRICS_PORT) are app-agnostic.
//
// All persistent files (BoltDB, feed-index SQLite, backups) live under
// a single per-app data directory: <APP>_DATA_DIR if set, otherwise
// <XDG_DATA_HOME or ~/.local/share>/<app.Name>. This keeps multi-tenant
// hosts (one box running both arabica and oolong) collision-free with a
// single env-var to point at the right disk in production.
func Run(ctx context.Context, app *domain.App, opts Options) error {
	if err := validateAppName(app.Name); err != nil {
		return err
	}
	envPrefix := strings.ToUpper(app.Name)

	dataDir, err := resolveDataDir(envPrefix, app.Name)
	if err != nil {
		return fmt.Errorf("resolve data dir: %w", err)
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir %s: %w", dataDir, err)
	}

	log.Info().
		Str("app", app.Name).
		Int("descriptors", len(app.Descriptors)).
		Str("data_dir", dataDir).
		Msg("Constructed app config")

	// Initialize OpenTelemetry tracing once per process. Multi-app boot
	// (cmd/server running both arabica and oolong) calls Run twice; the
	// tracer provider is global, so init must not race or double-register.
	tracingOnce.Do(func() {
		tp, err := tracing.Init(context.Background())
		if err != nil {
			log.Warn().Err(err).Msg("Failed to initialize tracing, continuing without it")
			return
		}
		// Shutdown is intentionally not deferred here: the provider is
		// process-global and outlives any single Run invocation.
		_ = tp
		log.Info().Msg("OpenTelemetry tracing initialized")
	})

	port := lookupAppEnv(envPrefix, "PORT")
	if port == "" {
		port = opts.DefaultPort
	}
	if port == "" {
		port = "18910"
	}

	bindAddr := lookupAppEnv(envPrefix, "BIND_ADDR")
	if bindAddr == "" {
		bindAddr = "0.0.0.0"
	}

	// Public URL for reverse proxy deployments
	publicURL := lookupAppEnv(envPrefix, "PUBLIC_URL")
	if publicURL == "" {
		publicURL = os.Getenv("SERVER_PUBLIC_URL")
	}

	// All persistent files live under dataDir (per-app, see resolveDataDir).
	dbPath := filepath.Join(dataDir, app.Name+".db")
	store, err := boltstore.Open(boltstore.Options{Path: dbPath})
	if err != nil {
		return fmt.Errorf("open database at %s: %w", dbPath, err)
	}
	defer store.Close()
	log.Info().Str("path", dbPath).Msg("Database opened")

	sessionStore := store.SessionStore()

	// OAuth manager
	clientID := lookupAppEnv(envPrefix, "OAUTH_CLIENT_ID")
	redirectURI := lookupAppEnv(envPrefix, "OAUTH_REDIRECT_URI")
	if clientID == "" && redirectURI == "" {
		if publicURL != "" {
			redirectURI = publicURL + "/oauth/callback"
			clientID = publicURL + "/.well-known/oauth-client-metadata.json"
			log.Info().Str("public_url", publicURL).Msg("Using public URL for OAuth (reverse proxy mode)")
		} else {
			redirectURI = fmt.Sprintf("http://127.0.0.1:%s/oauth/callback", port)
			clientID = "" // Empty triggers indigo localhost mode
			log.Info().Msg("Using localhost OAuth mode (for development)")
		}
	}
	// Declare the superset of scopes in client metadata so the auth server
	// accepts both the default login (base scopes) and the elevated
	// scope-upgrade flow (base + Bluesky profile scopes) when a user opts in
	// to editing their Bluesky profile from settings. The default
	// HandleLoginSubmit requests only the base subset.
	oauthApp, err := atp.NewOAuthApp(atp.OAuthConfig{
		ClientID:    clientID,
		RedirectURI: redirectURI,
		Scopes:      app.OAuthScopesWithProfile(),
		Store:       sessionStore,
		AppName:     app.Brand.DisplayName,
	})
	if err != nil {
		return fmt.Errorf("initialize OAuth: %w", err)
	}

	feedIndexPath := filepath.Join(dataDir, "feed-index.sqlite")

	// Firehose config — wantedCollections come from app.NSIDs() so the
	// jetstream subscription tracks the running app's entity set.
	firehoseConfig := firehose.DefaultConfig()
	firehoseConfig.IndexPath = feedIndexPath
	firehoseConfig.WantedCollections = app.NSIDs()
	if ttlStr := os.Getenv(envPrefix + "_PROFILE_CACHE_TTL"); ttlStr != "" {
		if ttl, err := time.ParseDuration(ttlStr); err == nil {
			firehoseConfig.ProfileCacheTTL = int64(ttl.Seconds())
		}
	}

	feedIndex, err := firehose.NewFeedIndex(feedIndexPath, time.Duration(firehoseConfig.ProfileCacheTTL)*time.Second)
	if err != nil {
		return fmt.Errorf("create feed index at %s: %w", feedIndexPath, err)
	}
	// Tell the index which comment collection this binary owns so it can
	// reconstruct comment AT-URIs for moderation / like-count lookups.
	feedIndex.SetCommentNSID(app.CommentNSID())
	log.Info().Str("path", feedIndexPath).Msg("Feed index opened")

	// One-time seed from legacy BoltDB feed_registry bucket.
	if legacyDIDs := store.LegacyFeedDIDs(); len(legacyDIDs) > 0 {
		added, err := feedIndex.SeedRegisteredDIDs(legacyDIDs)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to seed registered DIDs from legacy feed registry")
		} else if added > 0 {
			log.Info().Int("seeded", added).Msg("Seeded registered_dids from legacy BoltDB feed registry")
		}
	}

	feedRegistry := feed.NewPersistentRegistry(feedIndex)
	feedService := feed.NewService(feedRegistry)
	log.Info().Int("registered_users", feedRegistry.Count()).Msg("Feed service initialised")

	firehoseConsumer := firehose.NewConsumer(firehoseConfig, feedIndex)
	firehoseConsumer.Start(ctx)

	profileWatcher := firehose.NewProfileWatcher(firehoseConfig, feedIndex)
	profileWatcher.Start(ctx)

	adapter := firehose.NewFeedIndexAdapter(feedIndex)
	feedService.SetFirehoseIndex(adapter)

	moderationStore := sqlitestore.NewModerationStore(feedIndex.DB())
	feedService.SetModerationFilter(moderationStore)
	log.Info().Msg("Firehose consumer started")

	// Periodic gauge collector
	metrics.StartCollector(ctx, metrics.StatsSource{
		KnownDIDCount:   feedIndex.KnownDIDCount,
		RegisteredCount: feedRegistry.Count,
		RecordCount:     feedIndex.RecordCount,
		PendingJoinCount: func() int {
			joinStore := store.JoinStore()
			if reqs, err := joinStore.ListRequests(context.Background()); err == nil {
				return len(reqs)
			}
			return 0
		},
		LikeCount:               feedIndex.TotalLikeCount,
		CommentCount:            feedIndex.TotalCommentCount,
		RecordCountByCollection: feedIndex.RecordCountByCollection,
		FirehoseConnected:       firehoseConsumer.IsConnected,
		BoltStats:               store.Stats,
		SQLiteStats:             feedIndex.DB().Stats,
	}, 60*time.Second)

	// Log known DIDs already in the index.
	if knownDIDsFromDB, err := feedIndex.GetKnownDIDs(context.Background()); err == nil {
		if len(knownDIDsFromDB) > 0 {
			log.Info().Int("count", len(knownDIDsFromDB)).Strs("dids", knownDIDsFromDB).Msg("Known DIDs from firehose index")
		} else {
			log.Info().Msg("No known DIDs in firehose index yet (will populate as events arrive)")
		}
	} else {
		log.Warn().Err(err).Msg("Failed to retrieve known DIDs from firehose index")
	}

	// Background backfill of registered + known DIDs
	go runBackfill(ctx, firehoseConsumer, feedRegistry, opts.KnownDIDsPath)

	// onAuth is called by the CookieAuth middleware when a valid session is found.
	onAuth := func(did string) {
		feedRegistry.Register(did)
		profileWatcher.Watch(did)
		go func() {
			if err := firehoseConsumer.BackfillDID(context.Background(), did); err != nil {
				log.Warn().Err(err).Str("did", did).Msg("Failed to backfill new user")
			}
		}()
	}

	if clientID == "" {
		log.Info().Str("mode", "localhost development").Str("redirect_uri", redirectURI).Msg("OAuth configured")
	} else {
		log.Info().Str("mode", "public").Str("client_id", clientID).Str("redirect_uri", redirectURI).Msg("OAuth configured")
	}

	atprotoClient := atproto.NewClient(oauthApp)
	log.Info().Msg("ATProto client initialized")

	sessionCache := atproto.NewSessionCache()
	stopCacheCleanup := sessionCache.StartCleanupRoutine(10 * time.Minute)
	defer stopCacheCleanup()
	log.Info().Msg("Session cache initialized with background cleanup")

	secureCookies := os.Getenv("SECURE_COOKIES") == "true"

	h := handlers.NewHandler(
		oauthApp,
		atprotoClient,
		sessionCache,
		feedService,
		feedRegistry,
		handlers.Config{
			SecureCookies: secureCookies,
			PublicURL:     publicURL,
		},
	)
	h.SetFeedIndex(feedIndex)
	h.SetWitnessCache(feedIndex)
	h.SetBrand(app.Brand)
	h.SetApp(app)

	// Moderation
	moderatorsConfigPath := os.Getenv(envPrefix + "_MODERATORS_CONFIG")
	moderationSvc, err := moderation.NewService(moderatorsConfigPath)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize moderation service, moderation disabled")
	} else {
		h.SetModeration(moderationSvc, moderationStore)
	}

	// Periodic cleanup of expired moderation labels
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if n, err := moderationStore.CleanExpiredLabels(ctx); err != nil {
					log.Error().Err(err).Msg("Failed to clean expired labels")
				} else if n > 0 {
					log.Info().Int("count", n).Msg("Cleaned expired moderation labels")
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Automated backups land under the per-app data dir.
	backupDir := filepath.Join(dataDir, "backups")
	backupDest, err := backup.NewLocalDestination(backupDir)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create backup destination, backups disabled")
	} else {
		backupSvc := backup.NewService(backup.Config{
			ScheduleHour: 11, // 11:00 UTC = 3:00 AM PST
			Retain:       2,
			Dest:         backupDest,
		})
		backupSvc.AddSource(backup.NewSQLiteSource("feed-index", feedIndex.DB()))
		backupSvc.Start(ctx)
		log.Info().Str("dir", backupDir).Msg("Automated backups enabled")
	}

	// Join request handling
	smtpPort := 587
	if portStr := os.Getenv("SMTP_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil {
			smtpPort = p
		}
	}
	emailSender := email.NewSender(email.Config{
		Host:       os.Getenv("SMTP_HOST"),
		Port:       smtpPort,
		User:       os.Getenv("SMTP_USER"),
		Pass:       os.Getenv("SMTP_PASS"),
		From:       os.Getenv("SMTP_FROM"),
		AdminEmail: os.Getenv("ADMIN_EMAIL"),
	})
	joinStore := store.JoinStore()
	pdsAdminURL := os.Getenv("PDS_ADMIN_URL")
	pdsAdminToken := os.Getenv("PDS_ADMIN_PASSWORD")
	h.SetJoin(emailSender, joinStore, pdsAdminURL, pdsAdminToken)
	if emailSender.Enabled() {
		log.Info().Str("host", os.Getenv("SMTP_HOST")).Msg("Email notifications enabled for join requests")
	} else {
		log.Info().Msg("Email notifications disabled (SMTP_HOST not set), join requests will be saved to database only")
	}

	// Static assets: CSS bundle + per-file JS. Embedded at build time, or
	// re-read from disk per-request when <APP>_HOTRELOAD is set. The hash-
	// based URLs replace the manually-bumped ?v= query params.
	hotReload := hotReloadEnabled(envPrefix)
	cssDevDir := ""
	jsDevDir := ""
	if hotReload {
		cssDevDir = "internal/web/assets/css"
		jsDevDir = "internal/web/assets/js"
		log.Info().Msg("CSS+JS hot-reload enabled — assets re-read from disk on each request")
	}
	cssBundle := assets.New(assets.Config{AppName: app.Name, DevDir: cssDevDir})
	cssBundle.MustBuild()
	assets.Register(cssBundle)
	jsAssets := assets.NewJSAssets(assets.JSConfig{DevDir: jsDevDir})
	jsAssets.MustBuild()
	assets.RegisterJS(jsAssets)

	// Router
	handler := routing.SetupRouter(routing.Config{
		App:               app,
		Handlers:          h,
		OAuthApp:          oauthApp,
		OnAuth:            onAuth,
		Logger:            log.Logger,
		ModerationService: moderationSvc,
		FirehoseConsumer:  firehoseConsumer,
		CSSBundle:         cssBundle,
		JSAssets:          jsAssets,
	})

	// Internal metrics server (localhost-only)
	metricsPort := lookupAppEnv(envPrefix, "METRICS_PORT")
	if metricsPort == "" {
		metricsPort = opts.DefaultMetricsPort
	}
	if metricsPort == "" {
		metricsPort = "9101"
	}
	metricsMux := http.NewServeMux()
	metricsMux.Handle("GET /metrics", promhttp.Handler())
	metricsServer := &http.Server{
		Addr:    "127.0.0.1:" + metricsPort,
		Handler: metricsMux,
	}
	go func() {
		log.Info().Str("address", "127.0.0.1:"+metricsPort).Msg("Starting metrics server (localhost only)")
		if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("Metrics server failed to start")
		}
	}()

	// Public HTTP server
	httpServer := &http.Server{
		Addr:    bindAddr + ":" + port,
		Handler: handler,
	}
	serverErr := make(chan error, 1)
	go func() {
		log.Info().
			Str("app", app.Name).
			Str("address", bindAddr+":"+port).
			Str("url", "http://localhost:"+port).
			Bool("secure_cookies", secureCookies).
			Str("database", dbPath).
			Msg("Starting HTTP server")

		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		serverErr <- nil
	}()

	// Wait for ctx cancellation (caller wires SIGINT/SIGTERM) or fatal HTTP error.
	select {
	case <-ctx.Done():
		log.Info().Msg("Shutdown signal received")
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("HTTP server: %w", err)
		}
	}

	// Stop firehose first so no new events land while HTTP drains.
	log.Info().Msg("Stopping firehose consumer...")
	firehoseConsumer.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("Metrics server shutdown error")
	}
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server shutdown error")
	}

	log.Info().Msg("Server stopped")
	return nil
}

// resolveDataDir returns the per-app data directory: <APP>_DATA_DIR if
// set (used to point production at /var/lib/<app> or similar), otherwise
// <XDG_DATA_HOME or ~/.local/share>/<appName>. Both arabica and oolong
// running on the same host get isolated dirs by default.
func resolveDataDir(envPrefix, appName string) (string, error) {
	if d := os.Getenv(envPrefix + "_DATA_DIR"); d != "" {
		return d, nil
	}
	root := os.Getenv("XDG_DATA_HOME")
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		root = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(root, appName), nil
}

// hotReloadEnabled reports whether <APP>_HOTRELOAD is set to a truthy
// value, switching CSS+JS asset handlers from the embedded bytes to
// re-reading from disk on every request.
func hotReloadEnabled(envPrefix string) bool {
	v := lookupAppEnv(envPrefix, "HOTRELOAD")
	switch v {
	case "", "0", "false", "off", "no":
		return false
	default:
		return true
	}
}

// lookupAppEnv returns os.Getenv("<envPrefix>_<key>") if set, falling
// back to os.Getenv(key). This lets a single binary running multiple
// apps (cmd/server) keep per-app overrides like ARABICA_PORT and
// OOLONG_PORT distinct, while a one-app deploy that only sets the
// shared key continues to work unchanged.
func lookupAppEnv(envPrefix, key string) string {
	if v := os.Getenv(envPrefix + "_" + key); v != "" {
		return v
	}
	return os.Getenv(key)
}

// validateAppName ensures app.Name is safe for use as an env-var prefix
// and a path component. Allowed: lowercase letters and digits, starting
// with a letter. Rejects empty, hyphens, underscores, dots, slashes —
// all of which would silently break envPrefix construction or path joins.
func validateAppName(name string) error {
	if name == "" {
		return fmt.Errorf("app.Name must not be empty")
	}
	for i, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9' && i > 0:
		default:
			return fmt.Errorf("app.Name %q invalid at index %d: must match [a-z][a-z0-9]*", name, i)
		}
	}
	return nil
}

// runBackfill collects DIDs from the registry and the known-dids file,
// removes already-backfilled ones, and indexes the rest. Runs once at
// startup (after a 5s delay for the firehose to connect first).
func runBackfill(ctx context.Context, firehoseConsumer *firehose.Consumer, feedRegistry *feed.Registry, knownDIDsFile string) {
	select {
	case <-time.After(5 * time.Second):
	case <-ctx.Done():
		return
	}

	backfillCtx, backfillSpan := tracing.HandlerSpan(ctx, "backfill.startup")
	defer backfillSpan.End()

	collectCtx, collectSpan := tracing.HandlerSpan(backfillCtx, "backfill.collect_dids")
	didsToBackfill := make(map[string]struct{})

	for _, did := range feedRegistry.List() {
		didsToBackfill[did] = struct{}{}
	}

	if knownDIDsFile != "" {
		knownDIDs, err := loadKnownDIDs(knownDIDsFile)
		if err != nil {
			log.Warn().Err(err).Str("file", knownDIDsFile).Msg("Failed to load known DIDs file")
		} else {
			for _, did := range knownDIDs {
				didsToBackfill[did] = struct{}{}
			}
			log.Info().Int("count", len(knownDIDs)).Str("file", knownDIDsFile).Strs("dids", knownDIDs).Msg("Loaded known DIDs from file")
		}
	}
	collectSpan.SetAttributes(attribute.Int("backfill.total_dids", len(didsToBackfill)))
	collectSpan.End()

	filterCtx, filterSpan := tracing.HandlerSpan(collectCtx, "backfill.filter_backfilled")
	alreadyBackfilled, err := firehoseConsumer.BackfilledDIDs(filterCtx)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load backfilled DIDs, will check individually")
		alreadyBackfilled = make(map[string]struct{})
	}
	for did := range alreadyBackfilled {
		delete(didsToBackfill, did)
	}
	filterSpan.SetAttributes(
		attribute.Int("backfill.already_backfilled", len(alreadyBackfilled)),
		attribute.Int("backfill.remaining", len(didsToBackfill)),
	)
	filterSpan.End()

	_, execSpan := tracing.HandlerSpan(filterCtx, "backfill.execute",
		attribute.Int("backfill.count", len(didsToBackfill)),
	)
	successCount := 0
	for did := range didsToBackfill {
		if err := firehoseConsumer.BackfillDID(backfillCtx, did); err != nil {
			log.Warn().Err(err).Str("did", did).Msg("Failed to backfill user")
		} else {
			successCount++
		}
	}
	execSpan.SetAttributes(
		attribute.Int("backfill.success", successCount),
		attribute.Int("backfill.failed", len(didsToBackfill)-successCount),
	)
	execSpan.End()

	log.Info().
		Int("skipped", len(alreadyBackfilled)).
		Int("backfilled", successCount).
		Int("failed", len(didsToBackfill)-successCount).
		Msg("Backfill complete")
}

// loadKnownDIDs reads a file containing DIDs (one per line) and returns
// them as a slice. Lines starting with # are comments; whitespace is
// trimmed; entries not starting with "did:" are skipped with a warning.
func loadKnownDIDs(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer file.Close()

	var dids []string
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !strings.HasPrefix(line, "did:") {
			log.Warn().Str("file", filePath).Int("line", lineNum).Str("content", line).Msg("Skipping invalid DID (must start with 'did:')")
			continue
		}
		dids = append(dids, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	return dids, nil
}
