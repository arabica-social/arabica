// Command arabica is the coffee-tracking binary. It wires arabica's
// App value (descriptors + brand) and hands off to the shared platform
// server in internal/atplatform/server.
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

func main() {
	knownDIDsFile := flag.String("known-dids", "", "Path to file containing DIDs to backfill on startup (one per line)")
	flag.Parse()

	configureLogging()

	log.Info().Msg("Starting Arabica Coffee Tracker")

	app := newArabicaApp()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	if err := server.Run(ctx, app, server.Options{KnownDIDsPath: *knownDIDsFile}); err != nil {
		log.Fatal().Err(err).Msg("Server failed")
	}
}

// configureLogging sets the global zerolog level and output format
// based on LOG_LEVEL/LOG_FORMAT env vars and bridges slog so indigo
// (and any other slog user) routes through zerolog.
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
