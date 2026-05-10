package atproto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeHandle(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain ascii unchanged", "alice.example.com", "alice.example.com"},
		{"strips at prefix", "@alice.example.com", "alice.example.com"},
		{"lowercases", "Alice.Example.COM", "alice.example.com"},
		{"trims whitespace", "  alice.example.com  ", "alice.example.com"},
		{"empty stays empty", "", ""},
		{"unicode to punycode", "café.example.com", "xn--caf-dma.example.com"},
		{"punycode idempotent", "xn--caf-dma.example.com", "xn--caf-dma.example.com"},
		{"mixed case unicode", "Café.Example.com", "xn--caf-dma.example.com"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, NormalizeHandle(tc.in))
		})
	}
}

func TestDisplayHandle(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain ascii unchanged", "alice.example.com", "alice.example.com"},
		{"empty stays empty", "", ""},
		{"punycode to unicode", "xn--caf-dma.example.com", "café.example.com"},
		{"unicode idempotent", "café.example.com", "café.example.com"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, DisplayHandle(tc.in))
		})
	}
}

func TestNormalizeDisplayRoundTrip(t *testing.T) {
	inputs := []string{
		"alice.example.com",
		"café.example.com",
		"xn--caf-dma.example.com",
		"Café.Example.COM",
	}
	for _, in := range inputs {
		t.Run(in, func(t *testing.T) {
			ascii := NormalizeHandle(in)
			display := DisplayHandle(ascii)
			// Round-tripping back through normalize should land at the same ASCII form.
			assert.Equal(t, ascii, NormalizeHandle(display))
		})
	}
}
