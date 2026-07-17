package entities_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	_ "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/entities"
)

func TestAllForApp_filtersByNSIDPrefix(t *testing.T) {
	arab := entities.AllForApp("social.arabica.alpha")
	for _, d := range arab {
		assert.True(t, strings.HasPrefix(d.NSID, "social.arabica.alpha."),
			"arabica filter leaked NSID %s", d.NSID)
	}
	assert.NotEmpty(t, arab, "expected arabica descriptors")

	// A non-matching base returns no descriptors.
	none := entities.AllForApp("social.other.alpha")
	assert.Empty(t, none, "expected no descriptors for a non-matching base")
}
