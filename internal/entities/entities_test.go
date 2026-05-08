package entities

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func TestGetUnknownType(t *testing.T) {
	assert.Nil(t, Get("unknown-type"))
	assert.Nil(t, Get(""))
}

func TestRegisterDuplicatePanics(t *testing.T) {
	testType := lexicons.RecordType("_test_duplicate_sentinel")
	Register(&Descriptor{Type: testType, NSID: "test.nsid"})
	defer delete(registry, testType)

	assert.Panics(t, func() {
		Register(&Descriptor{Type: testType, NSID: "test.nsid"})
	})
}
