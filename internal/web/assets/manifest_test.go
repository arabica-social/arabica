package assets

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManifestZeroValueFallsBack(t *testing.T) {
	var manifest Manifest

	assert.Equal(t, "/static/css/output.css", manifest.StylesheetHref(""))
	assert.Equal(t, "/static/js/htmx.min.js", manifest.ScriptHref("htmx.min.js"))
}

func TestManifestUsesConfiguredAssets(t *testing.T) {
	css := New(Config{})
	css.MustBuild()
	js := NewJSAssets(JSConfig{})
	js.MustBuild()

	manifest := NewManifest(css, js)

	assert.True(t, strings.HasPrefix(manifest.StylesheetHref(""), "/static/css/output.css?h="))
	assert.True(t, strings.HasPrefix(manifest.ScriptHref("htmx.min.js"), "/static/js/htmx.min.js?h="))
}
