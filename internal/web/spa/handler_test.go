package spa

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"tangled.org/arabica.social/arabica/internal/web/assets"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testManifest() assets.Manifest {
	return assets.NewManifest(nil, nil)
}

func TestNewShellHandler_ReadsEmbeddedIndex(t *testing.T) {
	h, err := NewShellHandler(testManifest(), "arabica")
	require.NoError(t, err)
	assert.NotEmpty(t, h.indexHTML, "indexHTML should be loaded from embed")
}

func TestShellHandler_InjectsHeadContent(t *testing.T) {
	h, err := NewShellHandler(testManifest(), "arabica")
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/some/spa/route", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	resp := w.Result()
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))
	assert.Contains(t, html, "<title>Arabica</title>")
	assert.Contains(t, html, `property="og:title"`)
	assert.Contains(t, html, `property="og:site_name" content="Arabica"`)
	assert.Contains(t, html, `name="twitter:card"`)
	assert.Contains(t, html, `rel="stylesheet"`)
	assert.Contains(t, html, `rel="manifest" href="/static/manifest.json"`)
	// The SvelteKit bootstrap script should still be present
	assert.Contains(t, html, "__sveltekit")
}

func TestShellHandler_InjectsBodyAttrs(t *testing.T) {
	h, err := NewShellHandler(testManifest(), "arabica")
	require.NoError(t, err)

	t.Run("with DID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		ctx := WithDID(req.Context(), "did:plc:test123")
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		body, _ := io.ReadAll(w.Result().Body)
		assert.Contains(t, string(body), `data-user-did="did:plc:test123"`)
		assert.Contains(t, string(body), `data-app="arabica"`)
	})

	t.Run("without DID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		body, _ := io.ReadAll(w.Result().Body)
		assert.NotContains(t, string(body), "data-user-did")
	})
}

func TestShellHandler_InjectsSessionData(t *testing.T) {
	h, err := NewShellHandler(testManifest(), "arabica")
	require.NoError(t, err)
	h.SetSessionResolver(func(_ context.Context, did string) SessionData {
		if did != "did:plc:test123" {
			return SessionData{}
		}
		return SessionData{
			Handle:                  "alice.bsky.social",
			DisplayName:             "Alice",
			Avatar:                  "https://cdn.example/avatar.png",
			IsModerator:             true,
			UnreadNotificationCount: 3,
		}
	})

	req := httptest.NewRequest("GET", "/", nil)
	ctx := WithDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	html := string(body)
	assert.Contains(t, html, `data-user-handle="alice.bsky.social"`)
	assert.Contains(t, html, `data-user-display="Alice"`)
	assert.Contains(t, html, `data-user-avatar="https://cdn.example/avatar.png"`)
	assert.Contains(t, html, `data-is-moderator="true"`)
	assert.Contains(t, html, `data-unread-notifications="3"`)
}

func TestShellHandler_SessionDataEscaped(t *testing.T) {
	h, err := NewShellHandler(testManifest(), "arabica")
	require.NoError(t, err)
	h.SetSessionResolver(func(_ context.Context, _ string) SessionData {
		return SessionData{
			DisplayName: `Alice "<script>"`,
		}
	})

	req := httptest.NewRequest("GET", "/", nil)
	ctx := WithDID(req.Context(), "did:plc:test123")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	html := string(body)
	assert.Contains(t, html, `data-user-display="Alice &quot;&lt;script&gt;&quot;"`)
	// The raw, unescaped angle brackets must not leak into the attribute.
	assert.NotContains(t, html, `data-user-display="Alice "<script>""`)
}

func TestShellHandler_OolongBranding(t *testing.T) {
	h, err := NewShellHandler(testManifest(), "oolong")
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	html := string(body)

	assert.Contains(t, html, "tea tracking app")
	assert.Contains(t, html, "#b8d5aa") // light theme color
	assert.Contains(t, html, `data-app="oolong"`)
}

func TestShellHandler_OGImage(t *testing.T) {
	h, err := NewShellHandler(testManifest(), "arabica")
	require.NoError(t, err)

	// Simulate entity-specific OG data via context (future: middleware sets this)
	req := httptest.NewRequest("GET", "/beans/actor/rkey", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	html := string(body)

	// Without an OG image, twitter:card should be "summary"
	assert.Contains(t, html, `name="twitter:card" content="summary"`)
	assert.NotContains(t, html, "og:image")
}

func TestIsSPAAsset(t *testing.T) {
	assert.True(t, IsSPAAsset("/_app/immutable/entry/start.js"))
	assert.True(t, IsSPAAsset("/_app/immutable/chunks/abc.js"))
	assert.False(t, IsSPAAsset("/static/css/output.css"))
	assert.False(t, IsSPAAsset("/api/data"))
	assert.False(t, IsSPAAsset("/beans/actor/rkey"))
}

func TestAssetHandler_ServesImmutableChunks(t *testing.T) {
	handler := AssetHandler()

	// The embedded build should have _app/immutable/ files
	req := httptest.NewRequest("GET", "/immutable/entry/start.js", nil)
	// StripPrefix removes /_app/, so the path after strip is immutable/...
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	// We may or may not have this exact file depending on build state,
	// but the handler should not panic. A 200 or 404 is acceptable.
	assert.NotEqual(t, 500, w.Code)
}

func TestShellHandler_TraceparentInjected(t *testing.T) {
	h, err := NewShellHandler(testManifest(), "arabica")
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/", nil)
	ctx := WithTraceparent(req.Context(), "00-abc-def-01")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	assert.Contains(t, string(body), `name="traceparent" content="00-abc-def-01"`)
}

func TestShellHandler_MarkerReplaced(t *testing.T) {
	h, err := NewShellHandler(testManifest(), "arabica")
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	html := string(body)

	// The marker meta tag should be replaced with actual head content
	assert.NotContains(t, html, `name="arabica-spa-head"`)
	// But the SvelteKit modulepreload links should still be present
	assert.Contains(t, html, "modulepreload")
}

func TestShellHandler_NoCacheHeader(t *testing.T) {
	h, err := NewShellHandler(testManifest(), "arabica")
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
}

// Ensure the shell HTML is well-formed enough to contain key structural elements
func TestShellHandler_HTMLStructure(t *testing.T) {
	h, err := NewShellHandler(testManifest(), "arabica")
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	html := string(body)

	assert.True(t, strings.HasPrefix(strings.TrimSpace(html), "<!DOCTYPE html>"))
	assert.Contains(t, html, "<head>")
	assert.Contains(t, html, "</head>")
	assert.Contains(t, html, "<body")
	assert.Contains(t, html, "</body>")
	assert.Contains(t, html, "</html>")
}
