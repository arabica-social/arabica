// Command server is the unified entrypoint for the arabica coffee
// tracker and its tea-tracking sister app. By default it boots both
// apps in one process on distinct ports; APPS=arabica or APPS=oolong
// (or whatever teaAppName is set to in apps.go) selects a single app.
//
// Both apps share one tracing pipeline, one logging stream, and one
// signal handler — but each gets its own listener, metrics endpoint,
// data directory, and firehose consumer. Per-app env vars
// (<APP>_PORT, <APP>_PUBLIC_URL, etc.) prevent collisions in combined
// mode; bare-env fallbacks (PORT, SERVER_PUBLIC_URL, OAUTH_*) keep
// single-app deploys identical to the prior cmd/arabica binary.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/atplatform/server"
	"tangled.org/arabica.social/arabica/internal/logging"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	knownDIDsFile := flag.String("known-dids", "", "Path to file containing DIDs to backfill on startup (one per line)")
	flag.Parse()

	configureLogging()

	selected, err := selectApps(os.Getenv("APPS"))
	if err != nil {
		log.Fatal().Err(err).Msg("Invalid APPS env var")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Info().Msg("Signal received, shutting down all apps")
		cancel()
	}()

	var wg sync.WaitGroup
	for _, entry := range selected {
		wg.Go(
			func() {
				defer wg.Done()
				log.Info().Str("app", entry.App.Name).Msg("Starting app")
				err := server.Run(ctx, entry.App, server.Options{
					KnownDIDsPath:      *knownDIDsFile,
					DefaultPort:        entry.DefaultPort,
					DefaultMetricsPort: entry.DefaultMetricsPort,
				})
				if err != nil {
					log.Error().Err(err).Str("app", entry.App.Name).Msg("App exited with error")
					cancel() // bring the other app down too
				}
			},
		)
	}
	wg.Wait()
	log.Info().Msg("All apps stopped")
}

// appEntry pairs a constructed *domain.App with its default port set.
// Defaults are per-app so combined-mode boots don't collide on 18910.
type appEntry struct {
	App                *domain.App
	DefaultPort        string
	DefaultMetricsPort string
}

// selectApps resolves the APPS env var into the list of apps to boot.
// Empty or "all" boots every registered app. A comma-separated list
// (e.g. "arabica" or "arabica,oolong") boots just those.
func selectApps(envValue string) ([]appEntry, error) {
	all := []appEntry{
		{App: newArabicaApp(), DefaultPort: "18910", DefaultMetricsPort: "9101"},
		{App: newTeaApp(), DefaultPort: "18920", DefaultMetricsPort: "9102"},
	}

	envValue = strings.TrimSpace(envValue)
	// TODO: switch to "all" once the tea app is ready
	if envValue == "" || strings.EqualFold(envValue, "arabica") {
		return all, nil
	}

	wanted := map[string]bool{}
	for _, name := range strings.Split(envValue, ",") {
		name = strings.TrimSpace(strings.ToLower(name))
		if name != "" {
			wanted[name] = true
		}
	}

	var selected []appEntry
	for _, entry := range all {
		if wanted[entry.App.Name] {
			selected = append(selected, entry)
			delete(wanted, entry.App.Name)
		}
	}
	if len(wanted) > 0 {
		var unknown []string
		for k := range wanted {
			unknown = append(unknown, k)
		}
		return nil, fmt.Errorf("unknown app names in APPS=%q: %v", envValue, unknown)
	}
	if len(selected) == 0 {
		return nil, fmt.Errorf("APPS=%q resolved to no apps", envValue)
	}
	return selected, nil
}

func configureLogging() {
	switch os.Getenv("LOG_LEVEL") {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	if os.Getenv("LOG_FORMAT") == "json" {
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	} else {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		})
	}

	slog.SetDefault(slog.New(logging.NewZerologHandler(log.Logger)))
}
