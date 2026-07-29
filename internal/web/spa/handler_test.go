package spa

import (
	"context"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/middleware"
	"tangled.org/arabica.social/arabica/internal/web/assets"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testManifest() assets.Manifest {
	return assets.NewManifest(nil)
}

// arabicaBrand mirrors the BrandConfig set by the arabica app constructor
// (internal/arabica/app/app.go). Keeping a local copy lets the spa tests run
// without importing the app package.
func arabicaBrand() domain.BrandConfig {
	return domain.BrandConfig{
		DisplayName:     "Arabica",
		Tagline:         "Your brew, your data",
		SiteDescription: "Arabica is a coffee brew tracking app built on AT Protocol. Your brewing data is stored in your own Personal Data Server, giving you full ownership and portability.",
		LightThemeColor: "#4a2c2a",
		DarkThemeColor:  "#0F0A08",
	}
}

func TestNewShellHandler_ReadsEmbeddedIndex(t *testing.T) {
	h, err := NewShellHandler(testManifest(), "arabica", arabicaBrand())
	require.NoError(t, err)
	assert.NotEmpty(t, h.indexHTML, "indexHTML should be loaded from embed")
}

func TestShellHandler_InjectsHeadContent(t *testing.T) {
	h, err := NewShellHandler(testManifest(), "arabica", arabicaBrand())
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
	assert.Contains(t, html, "__sveltekit")
}

func TestShellHandler_InjectsCSPNonce(t *testing.T) {
	h, err := NewShellHandler(testManifest(), "arabica", arabicaBrand())
	require.NoError(t, err)

	// Simulate the security middleware adding a nonce to context.
	req := httptest.NewRequest("GET", "/", nil)
	ctx := middleware.WithCSPNonce(req.Context(), "test-nonce-xyz")
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	html := string(body)

	// Theme script in the injected head fragment.
	assert.Contains(t, html, `<script nonce="test-nonce-xyz">`)
	// SvelteKit's bootstrap inline script.
	assert.Contains(t, html, `<script nonce="test-nonce-xyz">`)
	assert.Contains(t, html, "__sveltekit")
	// The nonce should only appear on script tags (not elsewhere).
	assert.Equal(t, 2, strings.Count(html, `nonce="test-nonce-xyz"`))
}

func TestShellHandler_InjectsBodyAttrs(t *testing.T) {
	h, err := NewShellHandler(testManifest(), "arabica", arabicaBrand())
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
		assert.Contains(t, string(body), `data-frontend="sveltekit"`)
	})

	t.Run("without DID", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)

		body, _ := io.ReadAll(w.Result().Body)
		assert.NotContains(t, string(body), "data-user-did")
		assert.Contains(t, string(body), `data-frontend="sveltekit"`)
	})
}

func TestShellHandler_InjectsSessionData(t *testing.T) {
	h, err := NewShellHandler(testManifest(), "arabica", arabicaBrand())
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
	h, err := NewShellHandler(testManifest(), "arabica", arabicaBrand())
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

func TestShellHandler_OGImage(t *testing.T) {
	h, err := NewShellHandler(testManifest(), "arabica", arabicaBrand())
	require.NoError(t, err)

	// Simulate entity-specific OG data via context (future: middleware sets this)
	req := httptest.NewRequest("GET", "/beans/actor/rkey", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	html := string(body)

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

func TestAssetHandler_SetsJavaScriptContentType(t *testing.T) {
	handler := AssetHandler()

	entries, err := fs.Glob(EmbeddedFS(), "_app/immutable/entry/start.*.js")
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	req := httptest.NewRequest("GET", "/"+entries[0], nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/javascript; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestAssetHandler_ServesImmutableChunks(t *testing.T) {
	handler := AssetHandler()

	entries, err := fs.Glob(EmbeddedFS(), "_app/immutable/entry/app.*.js")
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	req := httptest.NewRequest("GET", "/"+entries[0], nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestContentTypeLockingResponseWriter(t *testing.T) {
	// Simulate a downstream handler that tries to overwrite an explicitly
	// set Content-Type with text/plain (mimicking http.FileServer on a
	// system with a broken MIME database).
	downstream := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("// js"))
	})

	handler := contentTypeMiddleware(downstream)
	req := httptest.NewRequest("GET", "/immutable/chunk.js", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/javascript; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestShellHandler_TraceparentInjected(t *testing.T) {
	h, err := NewShellHandler(testManifest(), "arabica", arabicaBrand())
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
	h, err := NewShellHandler(testManifest(), "arabica", arabicaBrand())
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	html := string(body)

	assert.NotContains(t, html, `name="arabica-spa-head"`)
	assert.Contains(t, html, "modulepreload")
}

func TestShellHandler_NoCacheHeader(t *testing.T) {
	h, err := NewShellHandler(testManifest(), "arabica", arabicaBrand())
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
}

func TestShellHandler_HTMLStructure(t *testing.T) {
	h, err := NewShellHandler(testManifest(), "arabica", arabicaBrand())
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

func TestShellHandler_DevDir_ReadsIndexFromDisk(t *testing.T) {
	devDir := t.TempDir()
	// Mirror the shape SvelteKit produces: the head marker that
	// ServeHTTP replaces, plus a unique body marker we can assert on.
	devIndex := "<!DOCTYPE html><html><head>" + headMarker +
		"</head><body data-dev-source=\"disk\">dev shell</body></html>"
	require.NoError(t, os.WriteFile(filepath.Join(devDir, "index.html"), []byte(devIndex), 0o644))

	h, err := NewShellHandler(testManifest(), "arabica", arabicaBrand())
	require.NoError(t, err)
	h.SetDevDir(devDir)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	body, _ := io.ReadAll(w.Result().Body)
	html := string(body)
	assert.Contains(t, html, `data-dev-source="disk"`, "dev dir should override embedded index.html")
	assert.Contains(t, html, "dev shell")
	assert.NotContains(t, html, `name="arabica-spa-head"`)
	assert.Contains(t, html, `<title>`)
}

func TestShellHandler_DevDir_FallsBackToEmbedded(t *testing.T) {
	devDir := t.TempDir()

	h, err := NewShellHandler(testManifest(), "arabica", arabicaBrand())
	require.NoError(t, err)
	h.SetDevDir(devDir)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "missing dev file should fall back to embedded")
	body, _ := io.ReadAll(w.Result().Body)
	assert.NotContains(t, string(body), `name="arabica-spa-head"`, "embedded marker should still be replaced")
}

func TestAssetHandlerWithDevDir_ServesFromDisk(t *testing.T) {
	devDir := t.TempDir()
	// Mirror the embedded layout: build/_app/immutable/entry/start.<hash>.js
	chunkRel := filepath.Join("immutable", "entry", "start.dev.js")
	require.NoError(t, os.MkdirAll(filepath.Join(devDir, "_app", "immutable", "entry"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(devDir, "_app", chunkRel), []byte("// dev chunk"), 0o644))

	handler := AssetHandlerWithDevDir(devDir)
	req := httptest.NewRequest("GET", "/_app/"+chunkRel, nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/javascript; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"), "dev assets should not be cached")
	body, _ := io.ReadAll(w.Result().Body)
	assert.Equal(t, "// dev chunk", string(body))
}
