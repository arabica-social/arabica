package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	provider "tangled.org/arabica.social/arabica/internal/ai"
	"testing"
)

func TestImageChatRealProviderWire(t *testing.T) {
	img := testImage(t, "png", 2, 2)
	req := validRequest()
	req.Messages = []Message{{Role: "user", ImagesUnavailable: 1}, {Role: "assistant", Content: "Please reattach"}, {Role: "user", Images: []Image{img}}, {Role: "assistant", Content: "What dose?"}, {Role: "user", Content: "18g"}}
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body struct {
			Messages []struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if len(body.Messages) != 7 {
			t.Error("wrong transcript size")
			return
		}
		if strings.Contains(string(body.Messages[1].Content), img.Data) {
			t.Error("pixels in grounding JSON")
		}
		if !strings.Contains(string(body.Messages[2].Content), "earlier image(s) unavailable") {
			t.Error("missing unavailable marker")
		}
		if body.Messages[4].Role != "user" {
			t.Error("image association changed")
		}
		var parts []provider.ContentPart
		if err := json.Unmarshal(body.Messages[4].Content, &parts); err != nil {
			t.Error(err)
			return
		}
		if len(parts) != 1 || parts[0].Type != "image_url" || parts[0].ImageURL.URL != "data:image/png;base64,"+img.Data {
			t.Error("bad image wire")
		}
		if string(body.Messages[6].Content) != `"18g"` {
			t.Error("lost follow-up")
		}
		content, _ := json.Marshal(validOutput)
		io.WriteString(w, `{"choices":[{"message":{"role":"assistant","content":`+string(content)+`},"finish_reason":"stop"}]}`)
	}))
	defer server.Close()
	cfg := provider.Config{DefaultProvider: "test", Providers: map[string]provider.ProviderConfig{"test": {BaseURL: server.URL, Model: "vision"}}}
	client, err := provider.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewService(client).Chat(context.Background(), ownDID, ownLibrary(), req); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("unexpected retry")
	}
	p := cfg.Providers["test"]
	p.MaxRequestBytes = 100
	cfg.Providers["test"] = p
	client, err = provider.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewService(client).Chat(context.Background(), ownDID, ownLibrary(), req); !errors.Is(err, ErrProviderRequestTooLarge) {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("oversize request sent upstream")
	}
}

func TestUnavailableImageMarkers(t *testing.T) {
	for _, tt := range []struct {
		name    string
		message Message
		valid   bool
	}{
		{"image only restored", Message{Role: "user", ImagesUnavailable: 1}, true},
		{"four", Message{Role: "user", ImagesUnavailable: 4}, true},
		{"too many", Message{Role: "user", ImagesUnavailable: 5}, false},
		{"negative", Message{Role: "user", ImagesUnavailable: -1}, false},
		{"assistant", Message{Role: "assistant", Content: "text", ImagesUnavailable: 1}, false},
		{"both", Message{Role: "user", ImagesUnavailable: 1, Images: []Image{testImage(t, "png", 1, 1)}}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := validRequest()
			req.Messages = []Message{tt.message, {Role: "user", Content: "followup"}}
			if got := validateRequest(req); (got == nil) != tt.valid {
				t.Fatalf("valid=%v err=%v", tt.valid, got)
			}
		})
	}
}

func TestImageTextBudgetRemainsBounded(t *testing.T) {
	req := validRequest()
	req.Messages = nil
	for i := 0; i < 40; i++ {
		req.Messages = append(req.Messages, Message{Role: "user", Content: strings.Repeat("界", 4000)})
	}
	req.Messages[0].Images = []Image{testImage(t, "png", 1, 1)}
	called := false
	client := fakeCompleter(func(context.Context, []provider.Message, []provider.Tool, json.RawMessage) (provider.Completion, error) {
		called = true
		return provider.Completion{}, nil
	})
	_, err := NewService(client).Chat(context.Background(), ownDID, ownLibrary(), req)
	if !errors.Is(err, ErrInvalidRequest) || called {
		t.Fatal("text context budget bypassed")
	}
}
