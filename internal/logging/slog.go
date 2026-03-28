// Package logging provides log integration utilities.
package logging

import (
	"context"
	"log/slog"

	"github.com/rs/zerolog"
)

// ZerologHandler is a slog.Handler that routes slog output through zerolog,
// ensuring libraries that use slog (e.g. indigo OAuth) produce output
// consistent with the rest of the application.
type ZerologHandler struct {
	logger zerolog.Logger
	attrs  []slog.Attr
	group  string
}

// NewZerologHandler returns a slog.Handler backed by the given zerolog.Logger.
func NewZerologHandler(logger zerolog.Logger) *ZerologHandler {
	return &ZerologHandler{logger: logger}
}

func (h *ZerologHandler) Enabled(_ context.Context, level slog.Level) bool {
	return h.logger.GetLevel() <= slogToZerolog(level)
}

func (h *ZerologHandler) Handle(_ context.Context, r slog.Record) error {
	ev := h.logger.WithLevel(slogToZerolog(r.Level))
	if ev == nil {
		return nil
	}

	// Add any pre-set attrs from WithAttrs.
	for _, a := range h.attrs {
		ev = addAttr(ev, h.group, a)
	}

	// Add attrs from the record itself.
	r.Attrs(func(a slog.Attr) bool {
		ev = addAttr(ev, h.group, a)
		return true
	})

	ev.Msg(r.Message)
	return nil
}

func (h *ZerologHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs), len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	newAttrs = append(newAttrs, attrs...)
	return &ZerologHandler{logger: h.logger, attrs: newAttrs, group: h.group}
}

func (h *ZerologHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	prefix := name
	if h.group != "" {
		prefix = h.group + "." + name
	}
	newAttrs := make([]slog.Attr, len(h.attrs))
	copy(newAttrs, h.attrs)
	return &ZerologHandler{logger: h.logger, attrs: newAttrs, group: prefix}
}

func slogToZerolog(level slog.Level) zerolog.Level {
	switch {
	case level >= slog.LevelError:
		return zerolog.ErrorLevel
	case level >= slog.LevelWarn:
		return zerolog.WarnLevel
	case level >= slog.LevelInfo:
		return zerolog.InfoLevel
	default:
		return zerolog.DebugLevel
	}
}

func addAttr(ev *zerolog.Event, group string, a slog.Attr) *zerolog.Event {
	key := a.Key
	if group != "" {
		key = group + "." + key
	}

	val := a.Value.Resolve()
	switch val.Kind() {
	case slog.KindString:
		return ev.Str(key, val.String())
	case slog.KindInt64:
		return ev.Int64(key, val.Int64())
	case slog.KindUint64:
		return ev.Uint64(key, val.Uint64())
	case slog.KindFloat64:
		return ev.Float64(key, val.Float64())
	case slog.KindBool:
		return ev.Bool(key, val.Bool())
	case slog.KindDuration:
		return ev.Dur(key, val.Duration())
	case slog.KindTime:
		return ev.Time(key, val.Time())
	case slog.KindGroup:
		for _, ga := range val.Group() {
			ev = addAttr(ev, key, ga)
		}
		return ev
	default:
		return ev.Interface(key, val.Any())
	}
}
