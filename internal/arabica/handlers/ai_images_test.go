package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"net/http/httptest"
	"strings"
	provider "tangled.org/arabica.social/arabica/internal/ai"
	arabicaai "tangled.org/arabica.social/arabica/internal/arabica/ai"
	"tangled.org/arabica.social/arabica/internal/records"
	"testing"
)

func TestAIImageRequest(t *testing.T) {
	var b bytes.Buffer
	if err := png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	data := base64.StdEncoding.EncodeToString(b.Bytes())
	for _, tt := range []struct {
		name, mime, data string
		status           int
	}{
		{"image only", "image/png", data, 200}, {"mismatched mime", "image/jpeg", data, 400},
		{"remote url", "image/png", "https://example.com/image.png", 400}, {"invalid bytes", "image/png", "not!base64!!", 400},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h, tc := setupAI(t)
			tc.MockStore.FetchAllRecordsFunc = func(context.Context, string) ([]records.RawRecord, error) { return nil, nil }
			msg, _ := json.Marshal(arabicaai.Message{Role: "user", Images: []arabicaai.Image{{MIMEType: tt.mime, Data: tt.data}}})
			body := strings.Replace(chatRequest, `{"role":"user","content":"18 grams coffee"}`, string(msg), 1)
			rec := httptest.NewRecorder()
			h.HandleChat(rec, aiRequest(body))
			if rec.Code != tt.status {
				t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), tt.data) {
				t.Fatal("image echoed in response")
			}
		})
	}
}

func TestAIProviderRequestLimitError(t *testing.T) {
	h, tc := setupAI(t)
	tc.MockStore.FetchAllRecordsFunc = func(context.Context, string) ([]records.RawRecord, error) { return nil, nil }
	h.Service = arabicaai.NewService(aiFakeCompleter(func(context.Context, []provider.Message, []provider.Tool, json.RawMessage) (provider.Completion, error) {
		return provider.Completion{}, provider.ErrRequestTooLarge
	}))
	rec := httptest.NewRecorder()
	h.HandleChat(rec, aiRequest(chatRequest))
	if rec.Code != 413 || !strings.Contains(rec.Body.String(), "ai_provider_request_too_large") || !strings.Contains(rec.Body.String(), "max_request_bytes") {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}
