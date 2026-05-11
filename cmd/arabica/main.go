// Command arabica is the entrypoint for the arabica coffee tracker.
// It runs a single app; the tea-tracking sister app lives in
// ./cmd/oolong. Both binaries share the same internal/atplatform/server
// bootstrap but get their own listener, metrics endpoint, data
// directory, and firehose consumer.
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tangled.org/arabica.social/arabica/internal/atplatform/server"
	"tangled.org/arabica.social/arabica/internal/logging"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	defaultPort        = "18910"
	defaultMetricsPort = "9101"
)

func main() {
	knownDIDsFile := flag.String("known-dids", "", "Path to file containing DIDs to backfill on startup (one per line)")
	flag.Parse()

	configureLogging()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Info().Msg("Signal received, shutting down")
		cancel()
	}()

	app := newArabicaApp()
	log.Info().Str("app", app.Name).Msg("Starting app")
	err := server.Run(ctx, app, server.Options{
		KnownDIDsPath:      *knownDIDsFile,
		DefaultPort:        defaultPort,
		DefaultMetricsPort: defaultMetricsPort,
	})
	if err != nil {
		log.Fatal().Err(err).Str("app", app.Name).Msg("App exited with error")
	}
	log.Info().Msg("Stopped")
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
