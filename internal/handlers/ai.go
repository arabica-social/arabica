package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"

	arabicaai "tangled.org/arabica.social/arabica/internal/arabica/ai"

	atpmiddleware "tangled.org/pdewey.com/atp/middleware"
)

const MaxAIChatBodySize = 6 << 20

// AIHandlers keeps the optional app service separate from shared platform state.
type AIHandlers struct {
	*Handler
	Service *arabicaai.Service
}

func (h AIHandlers) HandleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	allowed := h.allowed(r)
	WriteJSON(w, struct {
		Enabled bool `json:"enabled"`
		Allowed bool `json:"allowed"`
	}{h.Service.Enabled() && allowed, allowed}, "AI status")
}

func (h AIHandlers) HandleChat(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if !h.allowed(r) {
		http.NotFound(w, r)
		return
	}
	// Authenticate before revealing feature state or decoding user content.
	store, ok := h.GetRecordStore(r)
	if !ok {
		WriteJSONError(w, 401, "authentication_required", "Authentication required")
		return
	}
	did, ok := atpmiddleware.GetDID(r.Context())
	if !ok || store.DID() != did {
		WriteJSONError(w, 401, "authentication_required", "Authentication required")
		return
	}
	if !h.Service.Enabled() {
		WriteJSONError(w, 503, "ai_disabled", "Assistant is unavailable")
		return
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		WriteJSONError(w, 415, "invalid_content_type", "JSON content type required")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxAIChatBodySize)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	var req arabicaai.Request
	if err := d.Decode(&req); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			WriteJSONError(w, 413, "request_too_large", "Assistant request exceeds limit")
		} else {
			WriteJSONError(w, 400, "invalid_request", "Invalid assistant request")
		}
		return
	}
	if err := d.Decode(new(any)); err != io.EOF {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			WriteJSONError(w, 413, "request_too_large", "Assistant request exceeds limit")
		} else {
			WriteJSONError(w, 400, "invalid_request", "Invalid assistant request")
		}
		return
	}
	response, err := h.Service.Chat(r.Context(), did, store, req)
	if err != nil {
		if r.Context().Err() != nil {
			WriteJSONError(w, http.StatusRequestTimeout, "request_cancelled", "Assistant request cancelled")
			return
		}
		switch {
		case errors.Is(err, arabicaai.ErrInvalidRequest):
			WriteJSONError(w, 400, "invalid_request", "Invalid assistant request")
		case errors.Is(err, arabicaai.ErrProviderRequestTooLarge):
			WriteJSONError(w, 413, "ai_provider_request_too_large", "Images and context exceed the configured provider request limit; remove images or ask the operator to increase max_request_bytes")
		case errors.Is(err, arabicaai.ErrLibrary):
			WriteJSONError(w, 503, "ai_library_unavailable", "Own library unavailable; check selected records or try again")
		case errors.Is(err, arabicaai.ErrLimit):
			w.Header().Set("Retry-After", "60")
			WriteJSONError(w, 429, "ai_rate_limited", "Assistant request limit reached; try again later")
		default:
			diagnostic := arabicaai.ProviderDiagnostic(err)
			event := log.Error().Str("stage", diagnostic.Stage).Str("reason", diagnostic.Reason).
				Str("provider_error", diagnostic.Transport.Category).Int("upstream_status", diagnostic.Transport.HTTPStatus).
				Str("finish_reason", diagnostic.FinishReason).Int("response_bytes", diagnostic.ContentBytes).
				Int("tool_calls", diagnostic.ToolCalls).Str("field", diagnostic.Field)
			if span := trace.SpanContextFromContext(r.Context()); span.IsValid() {
				event = event.Str("trace_id", span.TraceID().String())
			}
			event.Msg("AI chat provider failure")
			WriteJSONError(w, 502, "ai_provider_error", "Assistant unavailable; your draft has not changed")
		}
		return
	}
	WriteJSON(w, response, "AI draft response")
}
