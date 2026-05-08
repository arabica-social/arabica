package main

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestArabicaApp_OAuthScopes_matchesLegacyList guards against accidental
// scope drift during the multi-tenant refactor. The previous static slice
// in atproto/oauth.go contained these scopes; the App-derived list must
// remain identical until we deliberately add or remove a scope.
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

// TestArabicaApp_NSIDs guards against accidental drift in the arabica
// app's collection set. Every code path that needs the entity list (the
// firehose subscription, witness/PDS export, BackfillUser) reads
// app.NSIDs() — this test pins the expected output for the arabica
// binary specifically.
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
