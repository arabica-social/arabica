package main

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOolongApp_NSIDs(t *testing.T) {
	app := newOolongApp()
	got := app.NSIDs()
	sort.Strings(got)

	want := []string{
		"social.oolong.alpha.brew",
		"social.oolong.alpha.brewer",
		"social.oolong.alpha.cafe",
		"social.oolong.alpha.comment",
		"social.oolong.alpha.drink",
		"social.oolong.alpha.like",
		"social.oolong.alpha.recipe",
		"social.oolong.alpha.tea",
		"social.oolong.alpha.vendor",
	}
	sort.Strings(want)

	assert.Equal(t, want, got)
}

func TestOolongApp_OAuthScopes(t *testing.T) {
	app := newOolongApp()
	got := app.OAuthScopes()
	sort.Strings(got)

	want := []string{
		"atproto",
		"repo:social.oolong.alpha.brew",
		"repo:social.oolong.alpha.brewer",
		"repo:social.oolong.alpha.cafe",
		"repo:social.oolong.alpha.comment",
		"repo:social.oolong.alpha.drink",
		"repo:social.oolong.alpha.like",
		"repo:social.oolong.alpha.recipe",
		"repo:social.oolong.alpha.tea",
		"repo:social.oolong.alpha.vendor",
	}
	sort.Strings(want)

	assert.Equal(t, want, got)
}
