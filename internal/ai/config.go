// Package ai provides app-agnostic, bounded OpenAI-compatible chat transport.
package ai

import (
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// Config selects a server-configured provider. Browser input must never supply it.
type Config struct {
	DefaultProvider string                    `json:"default_provider"`
	Providers       map[string]ProviderConfig `json:"providers"`
}

// ProviderConfig is server-only. APIKey is literal text with optional ${VAR}
// interpolation, resolved once by NewClient; never log this configuration.
// BaseURL is the API root (for example https://api.openai.com/v1).
// Plain HTTP is intentionally allowed for self-hosted providers.
type ProviderConfig struct {
	BaseURL          string `json:"base_url"`
	Model            string `json:"model"`
	APIKey           string `json:"api_key,omitempty"`
	TimeoutSeconds   int    `json:"timeout_seconds,omitempty"`
	MaxRequestBytes  int64  `json:"max_request_bytes,omitempty"`
	MaxResponseBytes int64  `json:"max_response_bytes,omitempty"`
	MaxTokens        int    `json:"max_tokens,omitempty"`
}

// LoadConfig reads an explicitly configured file. An unset feature flag is the
// bootstrap caller's responsibility, not a silent fallback here.
func LoadConfig(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, errors.New("ai: cannot open configuration")
	}
	defer f.Close()
	return ParseConfig(f)
}

// ParseConfig rejects unknown fields, trailing JSON, oversized input and invalid
// providers, including unused providers, so deployment mistakes fail explicitly.
func ParseConfig(r io.Reader) (Config, error) {
	b, err := io.ReadAll(io.LimitReader(r, 65537))
	if err != nil || len(b) > 65536 {
		return Config{}, errors.New("ai: cannot read configuration within limit")
	}
	d := json.NewDecoder(strings.NewReader(string(b)))
	d.DisallowUnknownFields()
	var c Config
	if err := d.Decode(&c); err != nil {
		return Config{}, errors.New("ai: invalid configuration JSON")
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return Config{}, errors.New("ai: trailing configuration data")
	}
	return normalize(c)
}

func normalize(c Config) (Config, error) {
	if strings.TrimSpace(c.DefaultProvider) == "" || len(c.Providers) == 0 || len(c.Providers) > 16 {
		return Config{}, errors.New("ai: invalid provider selection")
	}
	if _, ok := c.Providers[c.DefaultProvider]; !ok {
		return Config{}, errors.New("ai: default provider not found")
	}
	out := Config{DefaultProvider: c.DefaultProvider, Providers: make(map[string]ProviderConfig, len(c.Providers))}
	for name, p := range c.Providers {
		if strings.TrimSpace(name) == "" {
			return Config{}, errors.New("ai: empty provider name")
		}
		u, err := url.Parse(p.BaseURL)
		if err != nil || u.Hostname() == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || u.ForceQuery || strings.Contains(p.BaseURL, "#") || u.Fragment != "" || u.Opaque != "" || strings.ContainsAny(p.BaseURL, " \t\r\n") {
			return Config{}, errors.New("ai: invalid provider base URL")
		}
		if strings.HasSuffix(u.Host, ":") {
			return Config{}, errors.New("ai: invalid provider port")
		}
		if u.Port() != "" {
			port, err := strconv.Atoi(u.Port())
			if err != nil || port < 1 || port > 65535 {
				return Config{}, errors.New("ai: invalid provider port")
			}
		}
		if strings.TrimSpace(p.Model) == "" || len(p.Model) > 256 || strings.ContainsAny(p.Model, "\r\n") {
			return Config{}, errors.New("ai: invalid provider model")
		}
		if len(p.APIKey) > 8192 || (p.APIKey != "" && !validAPIKey(p.APIKey)) {
			return Config{}, errors.New("ai: invalid API key")
		}
		if p.TimeoutSeconds == 0 {
			p.TimeoutSeconds = 30
		}
		if p.MaxRequestBytes == 0 {
			p.MaxRequestBytes = 6 * 1024 * 1024
		}
		if p.MaxResponseBytes == 0 {
			p.MaxResponseBytes = 1024 * 1024
		}
		if p.MaxTokens == 0 {
			p.MaxTokens = 2048
		}
		if p.TimeoutSeconds < 1 || p.TimeoutSeconds > 120 || p.MaxRequestBytes < 1 || p.MaxRequestBytes > 8*1024*1024 || p.MaxResponseBytes < 1 || p.MaxResponseBytes > 4*1024*1024 || p.MaxTokens < 1 || p.MaxTokens > 16384 {
			return Config{}, errors.New("ai: provider limits outside allowed range")
		}
		out.Providers[name] = p
	}
	return out, nil
}
