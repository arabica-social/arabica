package coffeehandlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoutesSPAOwnedRoutes(t *testing.T) {
	routes := Routes{}.SPAOwnedRoutes()

	for _, pattern := range []string{
		"GET /{$}",
		"GET /about",
		"GET /my-coffee",
		"GET /roasters/{actor}/{id}",
		"GET /roasters/new",
		"GET /roasters/{id}/edit",
		"GET /brews/new",
		"GET /recipes",
		"GET /recipes/{actor}/{id}",
	} {
		assert.Contains(t, routes, pattern)
	}

	for _, pattern := range []string{
		"GET /manage",
		"GET /brews",
		"GET /roasters/{actor}/{id}/backlinks",
		"GET /teas/{actor}/{id}",
	} {
		assert.NotContains(t, routes, pattern)
	}
}
