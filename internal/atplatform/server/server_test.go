package server

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateAppName(t *testing.T) {
	cases := []struct {
		name    string
		wantErr bool
	}{
		{"arabica", false},
		{"matcha", false},
		{"app2", false},
		{"", true},
		{"Arabica", true},   // uppercase
		{"arabica-dev", true}, // hyphen
		{"arabica_dev", true}, // underscore
		{"arabica.dev", true}, // dot
		{"2arabica", true},    // leading digit
		{"arabica/x", true},   // slash
	}
	for _, c := range cases {
		err := validateAppName(c.name)
		if c.wantErr {
			assert.Error(t, err, "expected error for %q", c.name)
		} else {
			assert.NoError(t, err, "unexpected error for %q", c.name)
		}
	}
}

func TestResolveDataDir_Override(t *testing.T) {
	t.Setenv("ARABICA_DATA_DIR", "/srv/arabica")
	got, err := resolveDataDir("ARABICA", "arabica")
	assert.NoError(t, err)
	assert.Equal(t, "/srv/arabica", got)
}

func TestResolveDataDir_XDGFallback(t *testing.T) {
	t.Setenv("ARABICA_DATA_DIR", "")
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg")
	got, err := resolveDataDir("ARABICA", "arabica")
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join("/tmp/xdg", "arabica"), got)
}

func TestResolveDataDir_PerApp(t *testing.T) {
	// Both apps under the same XDG root get isolated dirs.
	t.Setenv("ARABICA_DATA_DIR", "")
	t.Setenv("MATCHA_DATA_DIR", "")
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg")
	a, _ := resolveDataDir("ARABICA", "arabica")
	m, _ := resolveDataDir("MATCHA", "matcha")
	assert.NotEqual(t, a, m)
}
