package main

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestArabicaApp_OAuthScopes_matchesLegacyList guards against accidental
// scope drift during the multi-tenant refactor.
func TestArabicaApp_OAuthScopes_matchesLegacyList(t *testing.T) {
	app := newArabicaApp()
	got := app.OAuthScopes()
	sort.Strings(got)

	want := []string{
		"atproto",
		"repo:social.arabica.alpha.bean",
		"repo:social.arabica.alpha.brew",
		"repo:social.arabica.alpha.brewer",
		"repo:social.arabica.alpha.comment",
		"repo:social.arabica.alpha.grinder",
		"repo:social.arabica.alpha.like",
		"repo:social.arabica.alpha.recipe",
		"repo:social.arabica.alpha.roaster",
	}
	sort.Strings(want)
	assert.Equal(t, want, got)
}

func TestArabicaApp_NSIDs(t *testing.T) {
	app := newArabicaApp()
	got := app.NSIDs()
	sort.Strings(got)

	want := []string{
		"social.arabica.alpha.bean",
		"social.arabica.alpha.brew",
		"social.arabica.alpha.brewer",
		"social.arabica.alpha.comment",
		"social.arabica.alpha.grinder",
		"social.arabica.alpha.like",
		"social.arabica.alpha.recipe",
		"social.arabica.alpha.roaster",
	}
	sort.Strings(want)

	assert.Equal(t, want, got)
}
