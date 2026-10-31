package handlers

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	provider "tangled.org/arabica.social/arabica/internal/ai"
	arabicaai "tangled.org/arabica.social/arabica/internal/arabica/ai"
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/records"
)

func TestAIChatThreePourOreaDoesNotAskToChooseFourPour(t *testing.T) {
	h, ctx := setupAI(t)
	var writes atomic.Int32
	ctx.MockStore.PutRecordFunc = func(context.Context, string, string, any) (string, string, error) { writes.Add(1); return "", "", nil }
	ctx.MockStore.RemoveRecordFunc = func(context.Context, string, string) error { writes.Add(1); return nil }
	const did = "did:plc:test123456789"
	ctx.MockStore.FetchAllRecordsFunc = func(_ context.Context, nsid string) ([]records.RawRecord, error) {
		switch strings.TrimPrefix(nsid, arabica.NSIDBase+".") {
		case "bean":
			return []records.RawRecord{{RKey: "private-fix", Record: map[string]any{"name": "The Fix – October 2026", "closed": false}}}, nil
		case "brewer":
			return []records.RawRecord{{RKey: "private-orea", Record: map[string]any{"name": "Orea 01", "brewerType": "pourover"}}}, nil
		case "recipe":
			rows := []records.RawRecord{}
			for i, name := range []string{"4-Pour Orea 01", "3-Pour (1 bloom) Orea 01"} {
				key := "private-four"
				count := 4
				coffee := 200.0
				if i == 1 {
					key = "private-three"
					count = 3
					coffee = 150.0
				}
				pours := []any{}
				for j := 0; j < count; j++ {
					pours = append(pours, map[string]any{"waterAmount": 50.0, "timeSeconds": float64(j * 20)})
				}
				rows = append(rows, records.RawRecord{RKey: key, Record: map[string]any{"name": name, "coffeeAmount": coffee, "waterAmount": 2500.0, "brewerRef": "at://" + did + "/" + arabica.NSIDBase + ".brewer/private-orea", "pours": pours}})
			}
			return rows, nil
		}
		return nil, nil
	}
	h.Service = arabicaai.NewService(aiFakeCompleter(func(_ context.Context, messages []provider.Message, _ []provider.Tool, _ json.RawMessage) (provider.Completion, error) {
		if strings.Contains(messages[1].Content, "private-three") || strings.Contains(messages[1].Content, did) {
			t.Error("private reference forwarded")
		}
		var payload struct {
			Library []struct {
				Kind, Label string
				Match       bool `json:"request_match"`
			} `json:"library"`
		}
		if err := json.Unmarshal([]byte(messages[1].Content), &payload); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, item := range payload.Library {
			if item.Kind == "recipe" && item.Label == "3-Pour (1 bloom) Orea 01" {
				found = true
				if !item.Match {
					t.Error("three pours not marked as clear match")
				}
			}
			if item.Kind == "recipe" && item.Label == "4-Pour Orea 01" && item.Match {
				t.Error("four pours falsely marked as match")
			}
		}
		if !found {
			t.Error("matching recipe absent")
		}
		return provider.Completion{Message: provider.Message{Role: "assistant", Content: `{"in_scope":true,"kind":"brew","reply":"Using the saved three-pour Orea recipe.","patch":{"bean_rkey":"bean-1"},"choices":[{"label":"four","patch":{"recipe_rkey":"recipe-1"}},{"label":"three","patch":{"recipe_rkey":"recipe-2"}}]}`}, FinishReason: "stop"}, nil
	}))
	body := strings.Replace(chatRequest, "18 grams coffee", "new brew with the fix from this month, 3 pour orea recipe", 1)
	recorder := httptest.NewRecorder()
	h.HandleChat(recorder, aiRequest(body))
	if recorder.Code != 200 {
		t.Fatalf("%d: %s", recorder.Code, recorder.Body.String())
	}
	var response arabicaai.Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Patch["recipe_rkey"] != "private-three" || response.Patch["brewer_rkey"] != "private-orea" || response.Patch["coffee_amount"] != 15.0 || len(response.Choices) != 0 || strings.Contains(response.Reply, "Choose a saved recipe") {
		t.Fatalf("unnecessary clarification or wrong recipe: %+v", response)
	}
	if pours, ok := response.Patch["pours"].([]any); !ok || len(pours) != 3 {
		t.Fatalf("wrong source pours: %+v", response)
	}
	if writes.Load() != 0 {
		t.Fatal("drafting mutated records")
	}
}
