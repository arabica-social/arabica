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
		"social.oolong.alpha.comment",
		"social.oolong.alpha.infuser",
		"social.oolong.alpha.like",
		"social.oolong.alpha.tea",
		"social.oolong.alpha.vendor",
		"social.oolong.alpha.vessel",
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
		"repo:social.oolong.alpha.comment",
		"repo:social.oolong.alpha.infuser",
		"repo:social.oolong.alpha.like",
		"repo:social.oolong.alpha.tea",
		"repo:social.oolong.alpha.vendor",
		"repo:social.oolong.alpha.vessel",
	}
	sort.Strings(want)

	assert.Equal(t, want, got)
}
