package routing_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	provider "tangled.org/arabica.social/arabica/internal/ai"
	arabicaai "tangled.org/arabica.social/arabica/internal/arabica/ai"
	arabicaapp "tangled.org/arabica.social/arabica/internal/arabica/app"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/moderation"
	"tangled.org/arabica.social/arabica/internal/records"
	"tangled.org/arabica.social/arabica/internal/routing"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"
)

const bodyLimitDID = "did:plc:bodylimittest"

type bodyLimitStore struct{ records.Store }

func (bodyLimitStore) DID() string { return bodyLimitDID }
func (bodyLimitStore) FetchAllRecords(context.Context, string) ([]records.RawRecord, error) {
	return nil, nil
}

type bodyLimitCompleter struct{ calls int }

func (c *bodyLimitCompleter) Complete(context.Context, []provider.Message, []provider.Tool, json.RawMessage) (provider.Completion, error) {
	c.calls++
	return provider.Completion{Message: provider.Message{Content: `{"in_scope":true,"reply":"Which bean?","kind":"brew","patch":{},"choices":[]}`}, FinishReason: "stop"}, nil
}

func TestAIRouterBodyLimits(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "moderation.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{"roles":{"admin":{"permissions":[]}},"users":[{"did":"`+bodyLimitDID+`","role":"admin"}]}`), 0600))
	mod, err := moderation.NewService(configPath)
	require.NoError(t, err)
	app := arabicaapp.New()
	h := handlers.NewHandler(nil, nil, nil, nil, nil, handlers.Config{})
	h.SetApp(app)
	h.SetModeration(mod, nil)
	h.SetStoreOverrideForTest(bodyLimitStore{})
	completer := &bodyLimitCompleter{}
	router := routing.SetupRouter(routing.Config{App: app, Handlers: h, AI: arabicaai.NewService(completer), ModerationService: mod, Logger: zerolog.Nop(), DisableRateLimit: true})

	img := image.NewNRGBA(image.Rect(0, 0, 450, 450))
	rng := rand.New(rand.NewPCG(1, 2))
	for i := range img.Pix {
		img.Pix[i] = byte(rng.Uint32())
	}
	var encoded bytes.Buffer
	require.NoError(t, (&png.Encoder{CompressionLevel: png.NoCompression}).Encode(&encoded, img))
	require.Less(t, encoded.Len(), 1<<20)
	data := base64.StdEncoding.EncodeToString(encoded.Bytes())
	images := make([]arabicaai.Image, 4)
	for i := range images {
		images[i] = arabicaai.Image{MIMEType: "image/png", Data: data}
	}
	body, err := json.Marshal(map[string]any{
		"session_id": "session", "request_id": "request", "revision": 0,
		"messages": []arabicaai.Message{{Role: "user", Content: "18g coffee", Images: images}},
		"draft":    map[string]any{"kind": "brew", "fields": map[string]any{}},
	})
	require.NoError(t, err)
	require.Greater(t, len(body), 1<<20)
	const limit = 6 << 20
	require.Less(t, len(body), limit)
	for _, tc := range []struct {
		name, body     string
		status         int
		providerCalled bool
	}{
		{"four images above default limit", string(body), 200, true},
		{"exact HTTP boundary", string(body) + strings.Repeat(" ", limit-len(body)), 200, true},
		{"above HTTP boundary", string(body) + strings.Repeat(" ", limit-len(body)+1), 413, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/ai/chat", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(atpmiddleware.ContextWithAuth(req.Context(), bodyLimitDID, "session"))
			rec := httptest.NewRecorder()
			before := completer.calls
			router.ServeHTTP(rec, req)
			require.Equal(t, tc.status, rec.Code, rec.Body.String())
			if tc.providerCalled {
				require.Equal(t, before+1, completer.calls)
			} else {
				require.Equal(t, before, completer.calls)
				require.Contains(t, rec.Body.String(), "request_too_large")
			}
		})
	}
}
