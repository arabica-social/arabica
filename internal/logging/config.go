package logging

import (
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// ConfigureFromEnv configures zerolog and the standard slog bridge from the
// LOG_LEVEL and LOG_FORMAT environment variables.
func ConfigureFromEnv(out io.Writer) {
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
		log.Logger = zerolog.New(out).With().Timestamp().Logger()
	} else {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        out,
			TimeFormat: time.RFC3339,
		})
	}

	slog.SetDefault(slog.New(NewZerologHandler(log.Logger)))
}
