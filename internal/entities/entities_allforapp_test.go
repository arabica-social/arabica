package entities_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"tangled.org/arabica.social/arabica/internal/entities"
	_ "tangled.org/arabica.social/arabica/internal/entities/arabica"
	_ "tangled.org/arabica.social/arabica/internal/entities/oolong"
)

func TestAllForApp_filtersByNSIDPrefix(t *testing.T) {
	arab := entities.AllForApp("social.arabica.alpha")
	for _, d := range arab {
		assert.True(t, strings.HasPrefix(d.NSID, "social.arabica.alpha."),
			"arabica filter leaked NSID %s", d.NSID)
	}
	assert.NotEmpty(t, arab, "expected arabica descriptors")

	tea := entities.AllForApp("social.oolong.alpha")
	for _, d := range tea {
		assert.True(t, strings.HasPrefix(d.NSID, "social.oolong.alpha."),
			"oolong filter leaked NSID %s", d.NSID)
	}
	assert.NotEmpty(t, tea, "expected oolong descriptors")

	for _, a := range arab {
		for _, o := range tea {
			assert.NotEqual(t, a.NSID, o.NSID)
		}
	}
}
