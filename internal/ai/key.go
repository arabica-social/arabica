package ai

import (
	"errors"
	"os"
	"strings"
)

func validAPIKey(key string) bool {
	for _, char := range key {
		if char <= ' ' || char == 127 {
			return false
		}
	}
	return true
}

// resolveAPIKey expands only ${NAME}, after JSON decoding. Bare dollar signs
// remain literal, and values are never expanded recursively or logged.
func resolveAPIKey(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	invalid := errors.New("ai: API key is invalid or an environment variable is missing or empty")
	var result strings.Builder
	for len(raw) > 0 {
		start := strings.Index(raw, "${")
		if start < 0 {
			result.WriteString(raw)
			break
		}
		result.WriteString(raw[:start])
		raw = raw[start+2:]
		end := strings.IndexByte(raw, '}')
		if end < 0 {
			return "", invalid
		}
		name := raw[:end]
		if name == "" {
			return "", invalid
		}
		for i, char := range name {
			if !(char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || char == '_' || i > 0 && char >= '0' && char <= '9') {
				return "", invalid
			}
		}
		value, exists := os.LookupEnv(name)
		if !exists || value == "" || len(value) > 8192 || result.Len()+len(value) > 8192 {
			return "", invalid
		}
		result.WriteString(value)
		raw = raw[end+1:]
	}
	key := result.String()
	if len(key) > 8192 || !validAPIKey(key) {
		return "", invalid
	}
	return key, nil
}
