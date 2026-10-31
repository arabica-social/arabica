package ai

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestAPIKeyValues(t *testing.T) {
	t.Setenv("AI_KEY_PART", "secret-part")
	t.Setenv("AI_KEY_OTHER", `quoted"\part`)
	t.Setenv("AI_KEY_NESTED", "${AI_KEY_PART}")
	for _, tc := range []struct{ name, input, want string }{
		{"literal", "literal-secret", "literal-secret"},
		{"no auth", "", ""},
		{"bare dollar literal", "$AI_KEY_PART", "$AI_KEY_PART"},
		{"environment", "${AI_KEY_PART}", "secret-part"},
		{"combined", "prefix-${AI_KEY_PART}-${AI_KEY_OTHER}", `prefix-secret-part-quoted"\part`},
		{"not recursive", "${AI_KEY_NESTED}", "${AI_KEY_PART}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := testConfig("http://localhost:8080/v1")
			p := cfg.Providers["test"]
			p.APIKey = tc.input
			cfg.Providers["test"] = p
			// Parse JSON first: environment quotes/backslashes cannot change its shape.
			data, err := json.Marshal(cfg)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := ParseConfig(strings.NewReader(string(data)))
			if err != nil {
				t.Fatal(err)
			}
			client, err := NewClient(parsed)
			if err != nil {
				t.Fatal(err)
			}
			if client.key != tc.want {
				t.Fatal("resolved key does not match expected value")
			}
			if client.provider.APIKey != "" {
				t.Fatal("provider metadata retained the key template")
			}
		})
	}
}

func TestAPIKeyResolutionOnce(t *testing.T) {
	t.Setenv("AI_KEY_ONCE", "first-secret")
	cfg := testConfig("http://localhost:8080/v1")
	p := cfg.Providers["test"]
	p.APIKey = "${AI_KEY_ONCE}"
	cfg.Providers["test"] = p
	client := testClient(t, cfg)
	t.Setenv("AI_KEY_ONCE", "changed-secret")
	if client.key != "first-secret" {
		t.Fatal("key changed after startup")
	}
}

func TestAPIKeyInterpolationFailures(t *testing.T) {
	t.Setenv("AI_KEY_MISSING", "")
	if err := os.Unsetenv("AI_KEY_MISSING"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AI_KEY_EMPTY", "")
	t.Setenv("AI_KEY_BAD", "secret\r\nheader")
	t.Setenv("AI_KEY_HUGE", strings.Repeat("x", 8193))
	t.Setenv("AI_KEY_BOUND", strings.Repeat("x", 8192))
	for _, value := range []string{
		"${AI_KEY_MISSING}", "prefix-${AI_KEY_EMPTY}", "${}", "${UNCLOSED",
		"${1INVALID}", "${AI-KEY}", "${AI_KEY_PART:-fallback}", "${AI_KEY_BAD}",
		"${AI_KEY_HUGE}", "prefix-${AI_KEY_BOUND}", "secret\x00value", "secret\tvalue",
	} {
		cfg := testConfig("http://localhost:8080/v1")
		p := cfg.Providers["test"]
		p.APIKey = value
		cfg.Providers["test"] = p
		_, err := NewClient(cfg)
		if err == nil {
			t.Fatal("invalid credential template accepted")
		}
		if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), value) {
			t.Fatal("credential error exposed its value")
		}
	}
}
