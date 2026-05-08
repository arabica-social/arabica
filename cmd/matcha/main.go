// Command matcha is the tea-tracking sister binary planned in
// docs/tea-multitenant-refactor.md. It builds its own *domain.App and
// hands off to internal/atplatform/server.Run, which is the same
// bootstrap arabica uses.
//
// What's missing for full feature parity with arabica:
//   - Tea-specific lexicons (tea, brew session, vessel, etc.) under a
//     matcha-side equivalent of internal/lexicons.
//   - A matcha entities-registration file (the analogue to arabica's
//     internal/entities/register.go) that registers tea descriptors so
//     app.Descriptors is non-empty.
//   - Tea-specific templ pages, modals, OG cards, and entity handlers.
//
// With those in place, this binary serves matcha at port 18910 with
// the same auth/firehose/feed/moderation stack arabica uses. Today,
// running matcha boots the platform but rejects logins (no
// descriptors → no entity routes → nothing useful to do post-login).
package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/atplatform/server"
	"tangled.org/arabica.social/arabica/internal/logging"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// newMatchaApp builds the App value for the matcha binary. Descriptors
// is intentionally empty: matcha's lexicons live in a tree that hasn't
// been authored yet. The empty list demonstrates the App layer accepts
// an arbitrary descriptor set; once tea lexicons exist they slot in
// here without touching shared packages.
func newMatchaApp() *domain.App {
	return &domain.App{
		Name:        "matcha",
		NSIDBase:    "social.matcha.alpha",
		Descriptors: nil,
		Brand: domain.BrandConfig{
			DisplayName: "Matcha",
			Tagline:     "Your tea, your data",
		},
	}
}

func main() {
	knownDIDsFile := flag.String("known-dids", "", "Path to file containing DIDs to backfill on startup (one per line)")
	flag.Parse()

	configureLogging()

	log.Info().Msg("Starting Matcha Tea Tracker")

	app := newMatchaApp()

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

// configureLogging mirrors the arabica binary's setup so both apps
// produce consistent output.
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
