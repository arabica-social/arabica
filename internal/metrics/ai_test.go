package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	dto "github.com/prometheus/client_model/go"
	provider "tangled.org/arabica.social/arabica/internal/ai"
)

func TestAITransportMetrics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"secret reply"}}],"model":"untrusted-model","usage":{"prompt_tokens":100,"completion_tokens":20,"prompt_tokens_details":{"cached_tokens":40}}}`))
	}))
	defer server.Close()
	config := provider.Config{DefaultProvider: "metrics-test", Providers: map[string]provider.ProviderConfig{"metrics-test": {Model: "configured-model", BaseURL: server.URL, APIKey: "secret-key"}}}
	client, err := provider.NewClientWithObserver(config, ObserveAI)
	if err != nil {
		t.Fatal(err)
	}
	counter := aiRequests.WithLabelValues("metrics-test", "configured-model", "success", "200")
	before := counterValue(counter)
	tokenBefore := map[string]float64{}
	for _, kind := range []string{"input", "output", "cached_input"} {
		tokenBefore[kind] = counterValue(aiTokens.WithLabelValues("metrics-test", "configured-model", kind))
	}
	usageBefore := counterValue(aiUsage.WithLabelValues("metrics-test", "configured-model", "reported"))
	if _, err := client.Complete(context.Background(), []provider.Message{{Role: "user", Content: "secret prompt"}}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got := counterValue(counter) - before; got != 1 {
		t.Fatalf("requests: %v", got)
	}
	for kind, want := range map[string]float64{"input": 100, "output": 20, "cached_input": 40} {
		if got := counterValue(aiTokens.WithLabelValues("metrics-test", "configured-model", kind)) - tokenBefore[kind]; got != want {
			t.Fatalf("%s tokens: %v", kind, got)
		}
	}
	registry := prometheus.NewRegistry()
	registry.MustRegister(aiRequests, aiDuration, aiTokens, aiUsage)
	recorder := httptest.NewRecorder()
	promhttp.HandlerFor(registry, promhttp.HandlerOpts{}).ServeHTTP(recorder, httptest.NewRequest("GET", "/metrics", nil))
	body := recorder.Body.String()
	for _, secret := range []string{"secret prompt", "secret reply", "secret-key", server.URL, "untrusted-model"} {
		if strings.Contains(body, secret) {
			t.Fatalf("metrics leaked %q", secret)
		}
	}
	if counterValue(aiUsage.WithLabelValues("metrics-test", "configured-model", "reported"))-usageBefore != 1 || !strings.Contains(body, `arabica_ai_usage_reports_total{model="configured-model",provider="metrics-test",state="reported"}`) || !strings.Contains(body, "arabica_ai_request_duration_seconds_count") {
		t.Fatal(body)
	}
}

func counterValue(counter prometheus.Counter) float64 {
	var metric dto.Metric
	if err := counter.Write(&metric); err != nil {
		panic(err)
	}
	return metric.GetCounter().GetValue()
}
