// Command oolong is the entrypoint for the oolong tea-tracking app,
// the sister to arabica. It runs a single app; see ./cmd/arabica for
// the coffee binary. Both share the same internal/atplatform/server
// bootstrap but get their own listener, metrics endpoint, data
// directory, and firehose consumer.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"tangled.org/arabica.social/arabica/internal/atplatform/server"
	"tangled.org/arabica.social/arabica/internal/logging"
	oolongapp "tangled.org/arabica.social/arabica/internal/oolong/app"
	teahandlers "tangled.org/arabica.social/arabica/internal/oolong/handlers"

	"github.com/rs/zerolog/log"
)

const (
	defaultPort        = "18920"
	defaultMetricsPort = "9102"
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

	app := oolongapp.New()
	log.Info().Str("app", app.Name).Msg("Starting app")
	err := server.Run(ctx, app, server.Options{
		KnownDIDsPath:      *knownDIDsFile,
		DefaultPort:        defaultPort,
		DefaultMetricsPort: defaultMetricsPort,
		AppRoutes:          teahandlers.Routes{},
		StaticPages:        teahandlers.StaticPages(),
	})
	if err != nil {
		log.Fatal().Err(err).Str("app", app.Name).Msg("App exited with error")
	}
	log.Info().Msg("Stopped")
}
