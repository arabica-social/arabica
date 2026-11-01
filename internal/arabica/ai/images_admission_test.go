package ai

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	provider "tangled.org/arabica.social/arabica/internal/ai"
	"tangled.org/arabica.social/arabica/internal/records"
	"testing"
	"time"
)

func TestImageAdmissionPrecedesDecodingAndCatalog(t *testing.T) {
	service := NewService(fakeCompleter(func(context.Context, []provider.Message, []provider.Tool, json.RawMessage) (provider.Completion, error) {
		t.Fatal("provider called")
		return provider.Completion{}, nil
	}))
	store := fakeLibrary{did: ownDID, fetch: func(context.Context, string) ([]records.RawRecord, error) { t.Fatal("catalog read"); return nil, nil }}
	release, err := service.limiter.acquire(ownDID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	req := validRequest()
	req.Messages[0].Images = []Image{{MIMEType: "image/png", Data: "bad!base64"}}
	_, err = service.Chat(context.Background(), ownDID, store, req)
	if !errors.Is(err, ErrLimit) {
		t.Fatalf("decoded bad image before admission: %v", err)
	}
	release()
	_, err = service.Chat(context.Background(), ownDID, store, req)
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("admitted invalid image not rejected: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = service.Chat(ctx, ownDID, store, req)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("decoded after canceled deadline: %v", err)
	}
}

func TestImageTurnsDoNotAuthorizeSoleOpenDefault(t *testing.T) {
	for _, tt := range []struct {
		name, content                 string
		unavailable, historical, fill bool
	}{
		{name: "image only"}, {name: "unspecified text", content: "15g coffee, 250g water"},
		{name: "missing image", unavailable: true}, {name: "explicit text", content: "Use El Mango Natural", fill: true},
		{name: "retained visual followup", content: "15g coffee, 250g water", historical: true},
		{name: "unavailable visual followup", content: "15g coffee, 250g water", historical: true, unavailable: true},
		{name: "explicit text after image", content: "Use El Mango Natural", historical: true, fill: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := validRequest()
			req.Messages[0].Content = tt.content
			if tt.unavailable {
				req.Messages[0].ImagesUnavailable = 1
			} else {
				req.Messages[0].Images = []Image{testImage(t, "png", 1, 1)}
			}
			if tt.historical {
				req.Messages[0].Content = ""
				req.Messages = append(req.Messages, Message{Role: "assistant", Content: "What dose and water?"}, Message{Role: "user", Content: tt.content})
			}
			client := fakeCompleter(func(ctx context.Context, m []provider.Message, tools []provider.Tool, format json.RawMessage) (provider.Completion, error) {
				if strings.Contains(m[1].Content, `"sole_open_default":true`) {
					t.Fatal("visual identity treated as unspecified coffee")
				}
				return outputClient(`{"in_scope":true,"reply":"Coffee bag","kind":"brew","patch":{"bean_rkey":"bean-1"},"choices":[]}`)(ctx, m, tools, format)
			})
			response, err := NewService(client).Chat(context.Background(), ownDID, forgivingLibrary(map[string]any{"name": "El Mango Natural", "closed": false}), req)
			if err != nil {
				t.Fatal(err)
			}
			if (response.Patch["bean_rkey"] == "bag-0") != tt.fill {
				t.Fatalf("incorrect visual grounding: %+v", response)
			}
			if !tt.fill && len(response.Choices) != 1 {
				t.Fatalf("guess not held for review: %+v", response)
			}
		})
	}
}
