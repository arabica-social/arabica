// Tests that the arabica package's init() correctly populates the shared
// entities registry with all expected record types. Lives here (not in
// internal/entities) so the registrations have actually run by import time.
package arabica_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"tangled.org/arabica.social/arabica/internal/entities"
	_ "tangled.org/arabica.social/arabica/internal/entities/arabica"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func TestArabicaRegistry_KnownTypes(t *testing.T) {
	for _, rt := range []lexicons.RecordType{
		lexicons.RecordTypeBean,
		lexicons.RecordTypeBrew,
		lexicons.RecordTypeBrewer,
		lexicons.RecordTypeGrinder,
		lexicons.RecordTypeRecipe,
		lexicons.RecordTypeRoaster,
	} {
		d := entities.Get(rt)
		assert.NotNil(t, d, "expected descriptor for %s", rt)
		if d == nil {
			continue
		}
		assert.Equal(t, rt, d.Type)
		assert.NotEmpty(t, d.NSID)
		assert.NotEmpty(t, d.DisplayName)
		assert.NotEmpty(t, d.Noun)
		assert.NotEmpty(t, d.URLPath)
	}
}

func TestArabicaRegistry_AllSorted(t *testing.T) {
	all := entities.All()
	assert.NotEmpty(t, all)
	for i := 1; i < len(all); i++ {
		assert.Less(t, string(all[i-1].Type), string(all[i].Type))
	}
}

func TestArabicaRegistry_AllContainsAllTypes(t *testing.T) {
	all := entities.All()
	types := make(map[lexicons.RecordType]bool, len(all))
	for _, d := range all {
		types[d.Type] = true
	}
	for _, rt := range []lexicons.RecordType{
		lexicons.RecordTypeBean,
		lexicons.RecordTypeBrew,
		lexicons.RecordTypeBrewer,
		lexicons.RecordTypeGrinder,
		lexicons.RecordTypeRecipe,
		lexicons.RecordTypeRoaster,
	} {
		assert.True(t, types[rt], "expected %s in All()", rt)
	}
}
