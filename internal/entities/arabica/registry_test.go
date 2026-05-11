// Tests that the arabica package's init() correctly populates the shared
// entities registry with all expected record types. Lives here (not in
// internal/entities) so the registrations have actually run by import time.
package arabica_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/entities/arabica"
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

func TestArabicaRegistry_RKey(t *testing.T) {
	cases := []struct {
		rt   lexicons.RecordType
		rec  any
		want string
	}{
		{lexicons.RecordTypeBean, &arabica.Bean{RKey: "b1"}, "b1"},
		{lexicons.RecordTypeRoaster, &arabica.Roaster{RKey: "r1"}, "r1"},
		{lexicons.RecordTypeGrinder, &arabica.Grinder{RKey: "g1"}, "g1"},
		{lexicons.RecordTypeBrewer, &arabica.Brewer{RKey: "br1"}, "br1"},
		{lexicons.RecordTypeRecipe, &arabica.Recipe{RKey: "re1"}, "re1"},
		{lexicons.RecordTypeBrew, &arabica.Brew{RKey: "bw1"}, "bw1"},
	}
	for _, c := range cases {
		d := entities.Get(c.rt)
		assert.NotNil(t, d.RKey, "RKey not wired for %s", c.rt)
		assert.Equal(t, c.want, d.RKey(c.rec), c.rt)
		// Wrong type yields "".
		assert.Equal(t, "", d.RKey(struct{}{}), "%s should reject wrong type", c.rt)
	}
}

func TestArabicaRegistry_DisplayTitle(t *testing.T) {
	cases := []struct {
		name string
		rt   lexicons.RecordType
		rec  any
		want string
	}{
		{"bean name", lexicons.RecordTypeBean, &arabica.Bean{Name: "Geisha"}, "Geisha"},
		{"bean fallback to origin", lexicons.RecordTypeBean, &arabica.Bean{Origin: "Ethiopia"}, "Ethiopia"},
		{"roaster name", lexicons.RecordTypeRoaster, &arabica.Roaster{Name: "Onyx"}, "Onyx"},
		{"recipe name", lexicons.RecordTypeRecipe, &arabica.Recipe{Name: "V60"}, "V60"},
		{"brew uses bean name", lexicons.RecordTypeBrew, &arabica.Brew{Bean: &arabica.Bean{Name: "Geisha"}}, "Geisha"},
		{"brew no bean falls back", lexicons.RecordTypeBrew, &arabica.Brew{}, "Coffee Brew"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := entities.Get(c.rt)
			assert.NotNil(t, d.DisplayTitle, "DisplayTitle not wired for %s", c.rt)
			assert.Equal(t, c.want, d.DisplayTitle(c.rec))
		})
	}
}
