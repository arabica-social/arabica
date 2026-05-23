// Command arabica is the entrypoint for the arabica coffee tracker.
// It runs a single app; the tea-tracking sister app lives in
// ./cmd/oolong. Both binaries share the same internal/atplatform/server
// bootstrap but get their own listener, metrics endpoint, data
// directory, and firehose consumer.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	coffeehandlers "tangled.org/arabica.social/arabica/internal/arabica/handlers"
	"tangled.org/arabica.social/arabica/internal/atplatform/apps"
	"tangled.org/arabica.social/arabica/internal/atplatform/server"
	"tangled.org/arabica.social/arabica/internal/logging"

	"github.com/rs/zerolog/log"
)

const (
	defaultPort        = "18910"
	defaultMetricsPort = "9101"
)

func main() {
	knownDIDsFile := flag.String("known-dids", "", "Path to file containing DIDs to backfill on startup (one per line)")
	flag.Parse()

	logging.ConfigureFromEnv(os.Stdout)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Info().Msg("Signal received, shutting down")
		cancel()
	}()

	app := apps.NewArabica()
	log.Info().Str("app", app.Name).Msg("Starting app")
	err := server.Run(ctx, app, server.Options{
		KnownDIDsPath:      *knownDIDsFile,
		DefaultPort:        defaultPort,
		DefaultMetricsPort: defaultMetricsPort,
		AppRoutes:          coffeehandlers.Routes{},
	})
	if err != nil {
		log.Fatal().Err(err).Str("app", app.Name).Msg("App exited with error")
	}
	log.Info().Msg("Stopped")
}
