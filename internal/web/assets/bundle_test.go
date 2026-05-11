package assets

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArabicaBundleFromEmbed(t *testing.T) {
	b := New(Config{AppName: "arabica"})
	b.MustBuild()

	bytes, hash, err := b.current()
	assert.NoError(t, err)
	assert.NotEmpty(t, bytes)
	assert.Len(t, hash, 16)
	assert.True(t, strings.Contains(string(bytes[:300]), "tokens.css"),
		"first bytes should come from tokens.css; got: %q", string(bytes[:300]))
}

func TestOolongBundleAppendsThemeOverlay(t *testing.T) {
	b := New(Config{AppName: "oolong"})
	b.MustBuild()

	bytes, _, err := b.current()
	assert.NoError(t, err)
	assert.NotEmpty(t, bytes)

	arabica := New(Config{AppName: "arabica"})
	arabica.MustBuild()
	arBytes, _, _ := arabica.current()

	assert.Greater(t, len(bytes), len(arBytes),
		"oolong bundle should be larger than arabica by the theme overlay")
}

func TestHrefIncludesContentHash(t *testing.T) {
	b := New(Config{AppName: "arabica"})
	href := b.Href()
	assert.True(t, strings.HasPrefix(href, "/static/css/output.css?h="), "got %q", href)
	assert.Contains(t, href, "?h=")
	// hash should be 16 hex chars after the ?h=
	hash := strings.TrimPrefix(href, "/static/css/output.css?h=")
	assert.Len(t, hash, 16)
}

func TestNonArabicaURLPath(t *testing.T) {
	b := New(Config{AppName: "oolong"})
	assert.Equal(t, "/static/css/output-oolong.css", b.URLPath())
}

func TestHandlerImmutableCacheInProduction(t *testing.T) {
	b := New(Config{AppName: "arabica"})
	req := httptest.NewRequest(http.MethodGet, "/static/css/output.css", nil)
	rec := httptest.NewRecorder()
	b.Handler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "text/css; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Equal(t, "public, max-age=31536000, immutable", rec.Header().Get("Cache-Control"))
	assert.NotEmpty(t, rec.Header().Get("ETag"))
}

func TestHandlerReturns304OnMatchingETag(t *testing.T) {
	b := New(Config{AppName: "arabica"})
	hash, err := b.hashOnly()
	assert.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/static/css/output.css", nil)
	req.Header.Set("If-None-Match", `"`+hash+`"`)
	rec := httptest.NewRecorder()
	b.Handler().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotModified, rec.Code)
	assert.Empty(t, rec.Body.Bytes())
}

func TestHrefForFallsBackToStaticPathWhenUnregistered(t *testing.T) {
	href := HrefFor("nonexistent-app")
	assert.Equal(t, "/static/css/output-nonexistent-app.css", href)
}

func TestHrefForReturnsRegisteredBundle(t *testing.T) {
	b := New(Config{AppName: "arabica"})
	Register(b)
	defer func() {
		registryMu.Lock()
		delete(registry, "arabica")
		registryMu.Unlock()
	}()

	href := HrefFor("arabica")
	assert.True(t, strings.HasPrefix(href, "/static/css/output.css?h="), "got %q", href)
}
