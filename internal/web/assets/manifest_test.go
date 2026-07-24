package assets

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestManifestZeroValueFallsBack(t *testing.T) {
	var manifest Manifest

	assert.Equal(t, "/static/css/output.css", manifest.StylesheetHref(""))
}

func TestManifestUsesConfiguredAssets(t *testing.T) {
	css := New(Config{})
	css.MustBuild()

	manifest := NewManifest(css)

	assert.True(t, strings.HasPrefix(manifest.StylesheetHref(""), "/static/css/output.css?h="))
}
