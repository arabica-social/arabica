package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	provider "tangled.org/arabica.social/arabica/internal/ai"
	arabicaai "tangled.org/arabica.social/arabica/internal/arabica/ai"
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/records"
)

func TestAIChatScreenshotShorthandHTTP(t *testing.T) {
	const did = "did:plc:test123456789"
	var calls, writes atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stage := calls.Add(1)
		var input struct {
			Messages []provider.Message `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || len(input.Messages) != 2 {
			t.Errorf("invalid provider request: %v", err)
			http.Error(w, "bad request", 500)
			return
		}
		for _, secret := range []string{did, "private-mango", "private-fast", "private-v60", "private-zp6", "at://"} {
			if strings.Contains(input.Messages[1].Content, secret) {
				t.Errorf("private reference leaked: %s", secret)
			}
		}
		var payload struct {
			Library []struct{ ID, Kind string } `json:"library"`
		}
		if err := json.Unmarshal([]byte(input.Messages[1].Content), &payload); err != nil {
			t.Error(err)
			http.Error(w, "bad payload", 500)
			return
		}
		ids := map[string]string{}
		for _, item := range payload.Library {
			ids[item.Kind] = item.ID
		}
		patch := map[string]any{}
		choices := []any{}
		reply := "Using El Mango Natural and Fast V60. Review the draft."
		if stage == 1 {
			choices = append(choices, map[string]any{"label": "Confirm bean", "patch": map[string]any{"bean_rkey": ids["bean"]}}, map[string]any{"label": "Confirm recipe", "patch": map[string]any{"recipe_rkey": ids["recipe"]}})
		} else {
			patch["rating"] = 8
			choices = append(choices, map[string]any{"label": "Confirm grinder", "patch": map[string]any{"grinder_rkey": ids["grinder"]}})
			reply = "Using 1ZPresso ZP6, rating 8."
		}
		content, _ := json.Marshal(map[string]any{"in_scope": true, "reply": reply, "kind": "brew", "patch": patch, "choices": choices})
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": string(content)}, "finish_reason": "stop"}}})
	}))
	defer upstream.Close()
	client, err := provider.NewClient(provider.Config{DefaultProvider: "test", Providers: map[string]provider.ProviderConfig{"test": {BaseURL: upstream.URL + "/v1", Model: "screenshot-test"}}})
	if err != nil {
		t.Fatal(err)
	}
	h, ctx := setupAI(t)
	h.Service = arabicaai.NewService(client)
	ctx.MockStore.PutRecordFunc = func(context.Context, string, string, any) (string, string, error) { writes.Add(1); return "", "", nil }
	ctx.MockStore.RemoveRecordFunc = func(context.Context, string, string) error { writes.Add(1); return nil }
	ref := func(kind, key string) string { return "at://" + did + "/" + arabica.NSIDBase + "." + kind + "/" + key }
	ctx.MockStore.FetchAllRecordsFunc = func(_ context.Context, nsid string) ([]records.RawRecord, error) {
		kind := strings.TrimPrefix(nsid, arabica.NSIDBase+".")
		var key string
		var data map[string]any
		switch kind {
		case "bean":
			key = "private-mango"
			data = map[string]any{"name": "El Mango Natural", "closed": false}
		case "brewer":
			key = "private-v60"
			data = map[string]any{"name": "Hario V60", "brewerType": "pourover"}
		case "grinder":
			key = "private-zp6"
			data = map[string]any{"name": "1ZPresso ZP6"}
		case "recipe":
			key = "private-fast"
			data = map[string]any{"name": "Fast V60", "brewerRef": ref("brewer", "private-v60"), "coffeeAmount": 150.0, "waterAmount": 2500.0, "pours": []any{map[string]any{"waterAmount": 40.0, "timeSeconds": 20.0}, map[string]any{"waterAmount": 120.0, "timeSeconds": 45.0}, map[string]any{"waterAmount": 90.0, "timeSeconds": 0.0}}}
		default:
			return nil, nil
		}
		return []records.RawRecord{{RKey: key, URI: ref(kind, key), Record: data}}, nil
	}
	current := map[string]any{}
	messages := []any{}
	var originalPours string
	for i, message := range []string{"brew, el mango, fast 60", "8 rating. used zp6 grinder"} {
		messages = append(messages, map[string]any{"role": "user", "content": message})
		body, _ := json.Marshal(map[string]any{"session_id": "screenshot", "request_id": fmt.Sprintf("request-%d", i), "revision": i, "messages": messages, "draft": map[string]any{"kind": "brew", "fields": current}})
		recorder := httptest.NewRecorder()
		h.HandleChat(recorder, aiRequest(string(body)))
		if recorder.Code != 200 {
			t.Fatalf("turn %d: status %d %s", i, recorder.Code, recorder.Body.String())
		}
		var response arabicaai.Response
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatal(err)
		}
		if len(response.Choices) != 0 {
			t.Fatalf("turn %d still requires confirmation buttons: %+v", i, response)
		}
		for key, value := range response.Patch {
			current[key] = value
		}
		for key, value := range map[string]any{"bean_rkey": "private-mango", "recipe_rkey": "private-fast", "brewer_rkey": "private-v60", "coffee_amount": 15.0, "water_amount": 250.0} {
			if current[key] != value {
				t.Errorf("turn %d: %s=%v want %v", i, key, current[key], value)
			}
		}
		rawPours, _ := json.Marshal(current["pours"])
		if i == 0 {
			originalPours = string(rawPours)
			pours, ok := current["pours"].([]any)
			if !ok || len(pours) != 3 {
				t.Fatalf("saved pours missing: %+v", current)
			}
			for n, amount := range []float64{40, 120, 90} {
				if pours[n].(map[string]any)["water_amount"] != amount || pours[n].(map[string]any)["time_seconds"] != ([]float64{20, 45, 0})[n] {
					t.Errorf("pour order/amount changed: %s", rawPours)
				}
			}
		} else {
			if current["rating"] != 8.0 || current["grinder_rkey"] != "private-zp6" || string(rawPours) != originalPours {
				t.Fatalf("follow-up failed or reset recipe: %+v", current)
			}
		}
		messages = append(messages, map[string]any{"role": "assistant", "content": response.Reply})
	}
	if calls.Load() != 2 || writes.Load() != 0 {
		t.Fatalf("provider calls=%d record writes=%d", calls.Load(), writes.Load())
	}
}
