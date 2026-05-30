package oolongapp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewWiresEntityRoutesForDescriptors(t *testing.T) {
	app := New()

	for _, d := range app.Descriptors {
		route, ok := app.EntityRouteByType(d.Type)
		assert.True(t, ok, "missing route metadata for %s", d.Type)
		assert.NotEmpty(t, route.Path, "route path for %s", d.Type)
		assert.NotEmpty(t, route.Noun, "route noun for %s", d.Type)
	}
}
