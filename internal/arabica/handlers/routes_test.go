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
		"GET /manage",
		"GET /brews",
		"GET /my-coffee",
		"GET /roasters/{actor}/{id}",
		"GET /roasters/{actor}/{id}/backlinks",
		"GET /roasters/new",
		"GET /roasters/{id}/edit",
		"GET /brews/new",
		"GET /recipes",
		"GET /recipes/{actor}/{id}",
		"GET /recipes/{actor}/{id}/backlinks",
		"GET /beans/{actor}/{id}/backlinks",
		"GET /grinders/{actor}/{id}/backlinks",
		"GET /brewers/{actor}/{id}/backlinks",
	} {
		assert.Contains(t, routes, pattern)
	}

	for _, pattern := range []string{
		"GET /teas/{actor}/{id}",
	} {
		assert.NotContains(t, routes, pattern)
	}
}
