package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompleteMultimodalWire(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []struct {
				Role    string          `json:"role"`
				Content json.RawMessage `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.Messages) != 4 {
			t.Fatal("lost conversation")
		}
		var parts []ContentPart
		if err := json.Unmarshal(body.Messages[1].Content, &parts); err != nil {
			t.Fatal(err)
		}
		if body.Messages[1].Role != "user" || len(parts) != 2 || parts[0].Type != "text" || parts[0].Text != "coffee" || parts[1].Type != "image_url" || parts[1].ImageURL.URL != "data:image/png;base64,cGl4ZWxz" {
			t.Fatalf("bad multimodal wire: %s", body.Messages[1].Content)
		}
		if string(body.Messages[0].Content) != `"context"` || string(body.Messages[2].Content) != `"question"` || string(body.Messages[3].Content) != `"answer"` {
			t.Fatal("text-only shape changed")
		}
		io.WriteString(w, goodResponse)
	}))
	defer server.Close()
	msgs := []Message{{Role: "system", Content: "context"}, {Role: "user", Parts: []ContentPart{{Type: "text", Text: "coffee"}, {Type: "image_url", ImageURL: &ImageURL{URL: "data:image/png;base64,cGl4ZWxz"}}}}, {Role: "assistant", Content: "question"}, {Role: "user", Content: "answer"}}
	if _, err := testClient(t, testConfig(server.URL)).Complete(context.Background(), msgs, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestCompleteLargeImagePayloadDefault(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		io.Copy(io.Discard, r.Body)
		io.WriteString(w, goodResponse)
	}))
	defer server.Close()
	msgs := []Message{{Role: "user", Parts: []ContentPart{{Type: "image_url", ImageURL: &ImageURL{URL: "data:image/jpeg;base64," + strings.Repeat("a", 5*1024*1024)}}}}}
	if _, err := testClient(t, testConfig(server.URL)).Complete(context.Background(), msgs, nil, nil); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatal("wrong number of requests")
	}
}
