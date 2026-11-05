package suggestions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://www.counterculturecoffee.com/shop", "counterculturecoffee.com"},
		{"http://example.com", "example.com"},
		{"https://example.com:8080/path", "example.com"},
		{"www.example.com", "example.com"},
		{"example.com", "example.com"},
		{"", ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, extractDomain(tt.input), "extractDomain(%q)", tt.input)
	}
}
