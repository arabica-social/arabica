package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRouteLabel(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{name: "dynamic API route", pattern: "GET /api/brews/{actor}/{id}", want: "/api/brews/{actor}/{id}"},
		{name: "dynamic page route", pattern: "GET /brews/{actor}/{id}", want: "/brews/{actor}/{id}"},
		{name: "exact root", pattern: "GET /{$}", want: "/"},
		{name: "static subtree", pattern: "GET /static/", want: "/static/*"},
		{name: "specific generated static asset", pattern: "GET /static/arabica-abcd.css", want: "/static/*"},
		{name: "Svelte asset subtree", pattern: "GET /_app/", want: "/_app/*"},
		{name: "catch-all", pattern: "/", want: "/{unmatched}"},
		{name: "no selected pattern", pattern: "", want: "/{unmatched}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, RouteLabel(tt.pattern))
		})
	}
}

func TestHTTPSpanName(t *testing.T) {
	assert.Equal(t, "GET /api/brews/{actor}/{id}", HTTPSpanName("GET", "GET /api/brews/{actor}/{id}"))
	assert.Equal(t, "GET /{unmatched}", HTTPSpanName("GET", ""))
}

func TestHTTPMethod(t *testing.T) {
	assert.Equal(t, "GET", HTTPMethod("GET"))
	assert.Equal(t, "OTHER", HTTPMethod("BREW"))
}
