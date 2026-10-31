// Command arabica runs the Arabica coffee tracker.
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	arabicaai "tangled.org/arabica.social/arabica/internal/arabica/ai"
	arabicaapp "tangled.org/arabica.social/arabica/internal/arabica/app"
	"tangled.org/arabica.social/arabica/internal/arabica/handlers"
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
	aiService, aiErr := arabicaai.FromEnv()
	if aiErr != nil {
		log.Fatal().Err(aiErr).Msg("Invalid assistant configuration")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Info().Msg("Signal received, shutting down")
		cancel()
	}()

	app := arabicaapp.New()
	log.Info().Str("app", app.Name).Msg("Starting app")
	err := server.Run(ctx, app, server.Options{
		KnownDIDsPath:      *knownDIDsFile,
		DefaultPort:        defaultPort,
		DefaultMetricsPort: defaultMetricsPort,
		AppRoutes:          handlers.Routes{AI: aiService},
	})
	if err != nil {
		log.Fatal().Err(err).Str("app", app.Name).Msg("App exited with error")
	}
	log.Info().Msg("Stopped")
}
