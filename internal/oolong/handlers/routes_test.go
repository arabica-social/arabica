package teahandlers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoutesSPAOwnedRoutesStartsEmpty(t *testing.T) {
	routes := Routes{}.SPAOwnedRoutes()

	assert.Empty(t, routes)
	for _, pattern := range []string{
		"GET /vendors/{actor}/{id}",
		"GET /vendors/{actor}/{id}/backlinks",
		"GET /teas/{actor}/{id}",
		"GET /my-tea",
	} {
		assert.NotContains(t, routes, pattern)
	}
}

func TestEntityRouteBundlesExposeJSONViews(t *testing.T) {
	bundles := (&Handlers{}).EntityRouteBundles()

	assert.Len(t, bundles, 5)
	for _, bundle := range bundles {
		assert.NotNil(t, bundle.JSONView, "%s JSON view", bundle.RecordType)
		assert.NotNil(t, bundle.JSONBacklinks, "%s JSON backlinks", bundle.RecordType)
	}
}
