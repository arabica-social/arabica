package ai

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigParsing(t *testing.T) {
	b, err := os.ReadFile("../../ai.example.json")
	if err != nil {
		t.Fatal(err)
	}
	c, err := ParseConfig(strings.NewReader(string(b)))
	if err != nil {
		t.Fatal(err)
	}
	p := c.Providers[c.DefaultProvider]
	if p.TimeoutSeconds != 30 || p.MaxRequestBytes != 262144 || p.MaxResponseBytes != 1048576 || p.MaxTokens != 2048 {
		t.Fatalf("unexpected defaults %+v", p)
	}
	path := filepath.Join(t.TempDir(), "ai.json")
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path + "missing"); err == nil {
		t.Fatal("missing config accepted")
	}
	for _, input := range []string{"", "null", "{}", string(b) + ` {}`, string(b) + ` null`, `{"unknown":true}`, strings.Repeat(" ", 65537), `{"default_provider":"missing","providers":{"a":{"base_url":"http://localhost:8080/v1","model":"m"}}}`, `{"default_provider":"a","providers":{"a":{"base_url":"http://localhost:8080/v1","model":"m","unknown":true}}}`} {
		if _, err := ParseConfig(strings.NewReader(input)); err == nil {
			t.Errorf("accepted invalid config %.100s", input)
		}
	}
	minimal := `{"default_provider":"a","providers":{"a":{"base_url":"http://localhost:8080/v1","model":"m"}}}`
	c, err = ParseConfig(strings.NewReader(minimal))
	if err != nil {
		t.Fatal(err)
	}
	p = c.Providers["a"]
	if p.TimeoutSeconds != 30 || p.MaxRequestBytes != 262144 || p.MaxResponseBytes != 1048576 || p.MaxTokens != 2048 {
		t.Fatalf("unexpected defaults %+v", p)
	}
}

func TestConfigValidation(t *testing.T) {
	for _, endpoint := range []string{"", "ftp://example.com", "/relative", "https:///path", "https://user:secret@example.com/v1", "https://example.com/v1?key=secret", "https://example.com/v1?", "https://example.com/v1#fragment", "https://exa mple.com", "https://example.com:bad", "https://example.com:99999", "https://example.com:", "https://example.com/v1#", "https://example.com/\n"} {
		if _, err := NewClient(testConfig(endpoint)); err == nil {
			t.Errorf("accepted URL %q", endpoint)
		}
	}
	for _, mutate := range []func(*ProviderConfig){
		func(p *ProviderConfig) { p.Model = "" }, func(p *ProviderConfig) { p.Model = "x\ny" },
		func(p *ProviderConfig) { p.APIKey = "secret\nvalue" }, func(p *ProviderConfig) { p.APIKey = " PAD " },
		func(p *ProviderConfig) { p.TimeoutSeconds = -1 }, func(p *ProviderConfig) { p.TimeoutSeconds = 121 },
		func(p *ProviderConfig) { p.MaxTokens = -1 }, func(p *ProviderConfig) { p.MaxTokens = 16385 },
		func(p *ProviderConfig) { p.MaxRequestBytes = -1 }, func(p *ProviderConfig) { p.MaxRequestBytes = 1024*1024 + 1 },
		func(p *ProviderConfig) { p.MaxResponseBytes = -1 }, func(p *ProviderConfig) { p.MaxResponseBytes = 4*1024*1024 + 1 },
	} {
		cfg := testConfig("http://localhost:8080/v1")
		p := cfg.Providers["test"]
		mutate(&p)
		cfg.Providers["test"] = p
		data, _ := json.Marshal(cfg)
		if _, err := ParseConfig(strings.NewReader(string(data))); err == nil {
			t.Errorf("accepted invalid config %+v", p)
		}
	}
	cfg := testConfig("http://localhost:8080/v1")
	cfg.Providers["unused"] = ProviderConfig{}
	if _, err := NewClient(cfg); err == nil {
		t.Error("invalid unused provider accepted")
	}
}

func TestMissingAndInvalidKeyRedacted(t *testing.T) {
	cfg := testConfig("http://localhost:8080/v1")
	p := cfg.Providers["test"]
	p.APIKey = "${AI_TEST_KEY}"
	cfg.Providers["test"] = p
	for _, key := range []string{"", " ", "secret\nvalue", "secret\rvalue"} {
		t.Setenv("AI_TEST_KEY", key)
		_, err := NewClient(cfg)
		if err == nil || strings.Contains(err.Error(), "secret") {
			t.Fatalf("invalid key error %v", err)
		}
	}
}
