package logging

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestZerologHandler_Handle(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	slogLogger := slog.New(NewZerologHandler(logger))

	slogLogger.Warn("auth server request failed", "request", "token-refresh", "statusCode", 400)

	out := buf.String()
	assert.Contains(t, out, `"level":"warn"`)
	assert.Contains(t, out, `"message":"auth server request failed"`)
	assert.Contains(t, out, `"request":"token-refresh"`)
	assert.Contains(t, out, `"statusCode":400`)
}

func TestZerologHandler_Enabled(t *testing.T) {
	logger := zerolog.New(nil).Level(zerolog.WarnLevel)
	h := NewZerologHandler(logger)

	assert.True(t, h.Enabled(nil, slog.LevelWarn))
	assert.True(t, h.Enabled(nil, slog.LevelError))
	assert.False(t, h.Enabled(nil, slog.LevelInfo))
	assert.False(t, h.Enabled(nil, slog.LevelDebug))
}

func TestZerologHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	slogLogger := slog.New(NewZerologHandler(logger)).With("component", "oauth")

	slogLogger.Info("test message")

	out := buf.String()
	assert.Contains(t, out, `"component":"oauth"`)
	assert.Contains(t, out, `"message":"test message"`)
}

func TestZerologHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	logger := zerolog.New(&buf)
	slogLogger := slog.New(NewZerologHandler(logger)).WithGroup("auth")

	slogLogger.Info("test", "method", "dpop")

	out := buf.String()
	assert.Contains(t, out, `"auth.method":"dpop"`)
}
