package ai

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	provider "tangled.org/arabica.social/arabica/internal/ai"
	"testing"
)

func TestSafeProviderDiagnostics(t *testing.T) {
	for _, tc := range []struct{ name, body, reason, field string }{
		{"invalid JSON", "secret-response", "invalid_json_envelope", ""},
		{"missing scope", `{"reply":"secret-response","kind":"brew","patch":{},"choices":[]}`, "missing_in_scope", ""},
		{"missing choices", `{"in_scope":true,"reply":"secret-response","kind":"brew","patch":{}}`, "missing_choices", ""},
		{"invalid measurement", `{"in_scope":true,"reply":"secret-response","kind":"brew","patch":{"temperature":999},"choices":[]}`, "invalid_draft_fields", "temperature"},
		{"unknown field", `{"in_scope":true,"reply":"secret-response","kind":"brew","patch":{"secret-key":1},"choices":[]}`, "invalid_draft_fields", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewService(outputClient(tc.body)).Chat(context.Background(), ownDID, ownLibrary(), validRequest())
			detail := ProviderDiagnostic(err)
			if !errors.Is(err, ErrProvider) || detail.Reason != tc.reason || detail.Field != tc.field {
				t.Fatalf("wrong safe diagnostic: %+v", detail)
			}
			encoded, _ := json.Marshal(detail)
			if strings.Contains(string(encoded), "secret") || strings.Contains(err.Error(), "secret") {
				t.Fatal("diagnostic leaked model content")
			}
		})
	}
	t.Run("opaque transport error", func(t *testing.T) {
		service := NewService(fakeCompleter(func(context.Context, []provider.Message, []provider.Tool, json.RawMessage) (provider.Completion, error) {
			return provider.Completion{}, errors.New("secret-key")
		}))
		_, err := service.Chat(context.Background(), ownDID, ownLibrary(), validRequest())
		if strings.Contains(err.Error(), "secret") || ProviderDiagnostic(err).Transport.Category != "unknown" {
			t.Fatal("opaque transport error leaked")
		}
	})
	t.Run("truncated completion", func(t *testing.T) {
		service := NewService(fakeCompleter(func(context.Context, []provider.Message, []provider.Tool, json.RawMessage) (provider.Completion, error) {
			return provider.Completion{Message: provider.Message{Content: "secret-response"}, FinishReason: "length"}, nil
		}))
		_, err := service.Chat(context.Background(), ownDID, ownLibrary(), validRequest())
		if d := ProviderDiagnostic(err); d.Reason != "incomplete_completion" || d.FinishReason != "length" {
			t.Fatal("truncation diagnostic missing")
		}
	})
}
