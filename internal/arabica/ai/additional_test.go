package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	provider "tangled.org/arabica.social/arabica/internal/ai"
	"tangled.org/arabica.social/arabica/internal/records"
)

func TestLongBrewAndManyPoursRemainValid(t *testing.T) {
	pours := make([]any, 100)
	for i := range pours {
		pours[i] = map[string]any{"water_amount": 1.0, "time_seconds": 172800.0}
	}
	if err := validateFields("brew", map[string]any{"time_seconds": 172800.0, "pours": pours, "espresso_params.yield_weight": 2000.5}); err != nil {
		t.Fatal("existing long-brew API values rejected", err)
	}
	if err := validateFields("brew", map[string]any{"pours": append(pours, pours[0])}); err == nil {
		t.Fatal("pour cap bypassed")
	}
}

func TestProvisionalKindMeaningfulValues(t *testing.T) {
	output := `{"in_scope":true,"reply":"What roast?","kind":"bean","patch":{"name":"New bag"},"choices":[]}`
	for name, value := range map[string]any{"zero": float64(0), "empty text": " ", "empty array": []any{}, "pours": []any{map[string]any{"water_amount": 10.0, "time_seconds": 5.0}}} {
		t.Run(name, func(t *testing.T) {
			req := validRequest()
			key := "pours"
			if name == "zero" {
				key = "coffee_amount"
			}
			if name == "empty text" {
				key = "tasting_notes"
			}
			req.Draft.Fields[key] = value
			got, err := NewService(outputClient(output)).Chat(context.Background(), ownDID, ownLibrary(), req)
			if err != nil {
				t.Fatal(err)
			}
			expected := "bean"
			if name == "pours" {
				expected = "brew"
			}
			if got.Kind != expected {
				t.Fatalf("kind=%s expected=%s", got.Kind, expected)
			}
		})
	}
}

func TestCompleteBoundedConversationPreservesOlderFacts(t *testing.T) {
	req := validRequest()
	req.Messages[0].Content = "first turn still matters"
	for len(req.Messages) < 40 {
		req.Messages = append(req.Messages, Message{Role: "user", Content: "more details"})
	}
	client := fakeCompleter(func(ctx context.Context, m []provider.Message, tools []provider.Tool, r json.RawMessage) (provider.Completion, error) {
		if !strings.Contains(m[1].Content, "first turn still matters") {
			t.Error("older unresolved dialogue was silently dropped")
		}
		var payload struct {
			Messages []Message `json:"messages"`
		}
		if err := json.Unmarshal([]byte(m[1].Content), &payload); err != nil || len(payload.Messages) != 40 {
			t.Fatalf("missing accepted dialogue: %d %v", len(payload.Messages), err)
		}
		return outputClient(validOutput)(ctx, m, tools, r)
	})
	if _, err := NewService(client).Chat(context.Background(), ownDID, ownLibrary(), req); err != nil {
		t.Fatal(err)
	}
}

func TestMultipleLibraryOptionsRequireChoice(t *testing.T) {
	lib := ownLibrary()
	original := lib.fetch
	lib.fetch = func(ctx context.Context, nsid string) ([]records.RawRecord, error) {
		result, err := original(ctx, nsid)
		if strings.HasSuffix(nsid, ".brewer") {
			result = append(result, records.RawRecord{RKey: "other-brewer", Record: map[string]any{"name": "Other brewer", "brewerType": "espresso"}})
		}
		return result, err
	}
	req := validRequest()
	req.Messages[0].Content = "18 grams coffee with a brewer"
	response, err := NewService(outputClient(validOutput)).Chat(context.Background(), ownDID, lib, req)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := response.Patch["brewer_rkey"]; ok || response.BrewerType != "" || len(response.Choices) != 1 || response.Choices[0].Patch["brewer_rkey"] != "real-brewer" || response.Choices[0].BrewerType != "pourover" {
		t.Fatalf("multi-option library pick auto-applied: %+v", response)
	}
}

func TestGlobalActiveBackpressureAndRelease(t *testing.T) {
	var l limiter
	now := time.Now()
	releases := make([]func(), maxActiveRequests)
	for i := range releases {
		var err error
		releases[i], err = l.acquire(fmt.Sprint(i), now)
		if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := l.acquire("extra", now.Add(2*time.Minute)); !errors.Is(err, ErrLimit) {
		t.Fatal("global backpressure bypassed")
	}
	if len(l.accounts) != maxActiveRequests {
		t.Fatal("active keys were evicted")
	}
	releases[0]()
	release, err := l.acquire("extra", now.Add(2*time.Minute))
	if err != nil {
		t.Fatal("global slot not released", err)
	}
	release()
	for _, r := range releases[1:] {
		r()
	}
	if l.active != 0 {
		t.Fatal("active slots leaked")
	}
}

func TestReplyComparisonNotationIsPlainText(t *testing.T) {
	output := strings.Replace(validOutput, `Which bean?`, `15g -> 250g, under <30s?`, 1)
	response, err := NewService(outputClient(output)).Chat(context.Background(), ownDID, ownLibrary(), validRequest())
	if err != nil || response.Reply != "15g -> 250g, under <30s?" {
		t.Fatalf("plain text notation rejected %+v %v", response, err)
	}
}
