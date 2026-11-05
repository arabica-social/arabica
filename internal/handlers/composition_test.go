package handlers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArabicaCompositionDefaults(t *testing.T) {
	did, session := CookieNames()
	require.Equal(t, "account_did", did)
	require.Equal(t, "session_id", session)
	h := NewHandler(nil, nil, nil, nil, nil, Config{})
	require.Equal(t, "arabica.social", h.siteCardOpts().Wordmark)
	require.Equal(t, FeedViews(), h.feedViews)
}
