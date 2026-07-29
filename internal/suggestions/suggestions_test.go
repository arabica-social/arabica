package suggestions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Keep these tests in-package so they can cover unexported normalization helpers.
func TestFuzzyName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Counter Culture Coffee", "counter culture"},
		{"Counter Culture", "counter culture"},
		{"counter culture coffee roasters", "counter culture"},
		{"Stumptown Coffee Roasters", "stumptown"},
		{"Stumptown", "stumptown"},
		{"Black & White Coffee", "black white"},
		{"  Some  Roasting Company  ", "some"},
		{"Heart Coffee Roasters", "heart"},
		{"Heart Roasters", "heart"},
		{"Heart", "heart"},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, FuzzyName(tt.input), "FuzzyName(%q)", tt.input)
	}
}

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

func TestNormalize(t *testing.T) {
	assert.Equal(t, "durham, nc", Normalize("  Durham,  NC  "))
	assert.Equal(t, "oakland ca", Normalize("Oakland CA"))
	assert.Equal(t, "", Normalize(""))
}
