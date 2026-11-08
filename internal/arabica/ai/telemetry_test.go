package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	provider "tangled.org/arabica.social/arabica/internal/ai"
)

func TestFromEnvWiresTelemetry(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hello"}}],"usage":{"prompt_tokens":11,"completion_tokens":4}}`))
	}))
	defer upstream.Close()
	config := provider.Config{DefaultProvider: "bootstrap-telemetry", Providers: map[string]provider.ProviderConfig{"bootstrap-telemetry": {BaseURL: upstream.URL, Model: "bootstrap-model"}}}
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "ai.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ARABICA_AI_CONFIG", path)
	service, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	before := bootstrapRequestCount(t)
	if _, err := service.client.Complete(context.Background(), []provider.Message{{Role: "user", Content: "hello"}}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := bootstrapRequestCount(t) - before; got != 1 {
		t.Fatalf("bootstrap telemetry calls: %v", got)
	}
}

func bootstrapRequestCount(t *testing.T) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() != "arabica_ai_requests_total" {
			continue
		}
		for _, metric := range family.Metric {
			for _, label := range metric.Label {
				if label.GetName() == "provider" && label.GetValue() == "bootstrap-telemetry" {
					return metric.GetCounter().GetValue()
				}
			}
		}
	}
	return 0
}
