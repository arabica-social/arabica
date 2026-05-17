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
		{"oolong", false},
		{"app2", false},
		{"", true},
		{"Arabica", true},     // uppercase
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
	got, source, err := resolveDataDir("ARABICA", "arabica")
	assert.NoError(t, err)
	assert.Equal(t, "/srv/arabica", got)
	assert.Equal(t, "env:ARABICA_DATA_DIR", source)
}

func TestResolveDataDir_XDGFallback(t *testing.T) {
	t.Setenv("ARABICA_DATA_DIR", "")
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg")
	got, source, err := resolveDataDir("ARABICA", "arabica")
	assert.NoError(t, err)
	assert.Equal(t, filepath.Join("/tmp/xdg", "arabica"), got)
	assert.Equal(t, "env:XDG_DATA_HOME", source)
}

func TestResolveDataDir_PerApp(t *testing.T) {
	// Both apps under the same XDG root get isolated dirs.
	t.Setenv("ARABICA_DATA_DIR", "")
	t.Setenv("OOLONG_DATA_DIR", "")
	t.Setenv("XDG_DATA_HOME", "/tmp/xdg")
	a, _, _ := resolveDataDir("ARABICA", "arabica")
	m, _, _ := resolveDataDir("OOLONG", "oolong")
	assert.NotEqual(t, a, m)
}
