package assets

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJSAssetsLoadsEmbeddedFiles(t *testing.T) {
	a := NewJSAssets(JSConfig{})
	a.MustBuild()

	// Spot-check a file we know exists.
	bytes, hash, ok := a.lookup("combo-select.js")
	assert.True(t, ok)
	assert.NotEmpty(t, bytes)
	assert.Len(t, hash, 16)
}

func TestJSAssetsHrefIncludesContentHash(t *testing.T) {
	a := NewJSAssets(JSConfig{})
	href := a.Href("combo-select.js")
	assert.True(t, strings.HasPrefix(href, "/static/js/combo-select.js?h="), "got %q", href)
	hash := strings.TrimPrefix(href, "/static/js/combo-select.js?h=")
	assert.Len(t, hash, 16)
}

func TestJSAssetsHrefEmptyForUnknownFile(t *testing.T) {
	a := NewJSAssets(JSConfig{})
	assert.Equal(t, "", a.Href("does-not-exist.js"))
}

func TestJSAssetsHandlerServesContentWithCacheHeaders(t *testing.T) {
	a := NewJSAssets(JSConfig{})

	req := httptest.NewRequest(http.MethodGet, "/static/js/combo-select.js", nil)
	req.SetPathValue("name", "combo-select.js")
	rec := httptest.NewRecorder()
	a.Handler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/javascript; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Equal(t, "public, max-age=31536000, immutable", rec.Header().Get("Cache-Control"))
	assert.NotEmpty(t, rec.Header().Get("ETag"))
	assert.NotEmpty(t, rec.Body.Bytes())
}

func TestJSAssetsHandlerReturns404ForUnknownFile(t *testing.T) {
	a := NewJSAssets(JSConfig{})

	req := httptest.NewRequest(http.MethodGet, "/static/js/missing.js", nil)
	req.SetPathValue("name", "missing.js")
	rec := httptest.NewRecorder()
	a.Handler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestJSAssetsHandlerReturns304OnMatchingETag(t *testing.T) {
	a := NewJSAssets(JSConfig{})
	hash, ok := a.hashFor("combo-select.js")
	assert.True(t, ok)

	req := httptest.NewRequest(http.MethodGet, "/static/js/combo-select.js", nil)
	req.SetPathValue("name", "combo-select.js")
	req.Header.Set("If-None-Match", `"`+hash+`"`)
	rec := httptest.NewRecorder()
	a.Handler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotModified, rec.Code)
	assert.Empty(t, rec.Body.Bytes())
}

func TestJSHrefForFallsBackWhenUnregistered(t *testing.T) {
	// Clear registry for this test.
	jsRegistryMu.Lock()
	saved := jsRegistered
	jsRegistered = nil
	jsRegistryMu.Unlock()
	defer func() {
		jsRegistryMu.Lock()
		jsRegistered = saved
		jsRegistryMu.Unlock()
	}()

	assert.Equal(t, "/static/js/combo-select.js", JSHrefFor("combo-select.js"))
}

func TestJSHrefForUsesRegisteredAssets(t *testing.T) {
	a := NewJSAssets(JSConfig{})
	RegisterJS(a)
	defer func() {
		jsRegistryMu.Lock()
		jsRegistered = nil
		jsRegistryMu.Unlock()
	}()

	href := JSHrefFor("combo-select.js")
	assert.True(t, strings.HasPrefix(href, "/static/js/combo-select.js?h="), "got %q", href)
}
