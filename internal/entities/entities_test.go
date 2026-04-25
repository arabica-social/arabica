package entities

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func TestGetKnownTypes(t *testing.T) {
	for _, rt := range []lexicons.RecordType{
		lexicons.RecordTypeBean,
		lexicons.RecordTypeBrew,
		lexicons.RecordTypeBrewer,
		lexicons.RecordTypeGrinder,
		lexicons.RecordTypeRecipe,
		lexicons.RecordTypeRoaster,
	} {
		d := Get(rt)
		assert.NotNil(t, d, "expected descriptor for %s", rt)
		assert.Equal(t, rt, d.Type)
		assert.NotEmpty(t, d.NSID)
		assert.NotEmpty(t, d.DisplayName)
		assert.NotEmpty(t, d.Noun)
		assert.NotEmpty(t, d.URLPath)
	}
}

func TestGetUnknownType(t *testing.T) {
	assert.Nil(t, Get("unknown-type"))
	assert.Nil(t, Get(""))
}

func TestAllReturnsSortedDescriptors(t *testing.T) {
	all := All()
	assert.NotEmpty(t, all)
	for i := 1; i < len(all); i++ {
		assert.Less(t, string(all[i-1].Type), string(all[i].Type))
	}
}

func TestAllContainsAllRegisteredTypes(t *testing.T) {
	all := All()
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

func TestRegisterDuplicatePanics(t *testing.T) {
	testType := lexicons.RecordType("_test_duplicate_sentinel")
	Register(&Descriptor{Type: testType, NSID: "test.nsid"})
	defer delete(registry, testType)

	assert.Panics(t, func() {
		Register(&Descriptor{Type: testType, NSID: "test.nsid"})
	})
}
