package main

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"tangled.org/arabica.social/arabica/internal/firehose"
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

// TestArabicaApp_NSIDs_matchesFirehoseCollections guards against drift
// between the runtime firehose subscription (driven by app.NSIDs()) and
// firehose.ArabicaCollections, which is still read by BackfillUser and
// admin export. Once those readers migrate to App-driven sources in later
// phases, ArabicaCollections (and this test) can be removed.
func TestArabicaApp_NSIDs_matchesFirehoseCollections(t *testing.T) {
	app := newArabicaApp()
	got := app.NSIDs()
	sort.Strings(got)

	want := append([]string(nil), firehose.ArabicaCollections...)
	sort.Strings(want)

	assert.Equal(t, want, got)
}
