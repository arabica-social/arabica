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

func TestRegisterRecordBehaviorDuplicatePanics(t *testing.T) {
	testType := lexicons.RecordType("_test_behavior_duplicate_sentinel")
	RegisterRecordBehavior(testType, &RecordBehavior{})
	defer delete(behaviorRegistry, testType)

	assert.Panics(t, func() {
		RegisterRecordBehavior(testType, &RecordBehavior{})
	})
}

func TestBehaviorByNSID(t *testing.T) {
	testType := lexicons.RecordType("_test_behavior_by_nsid")
	Register(&Descriptor{Type: testType, NSID: "test.behavior.nsid"})
	RegisterRecordBehavior(testType, &RecordBehavior{})
	defer delete(registry, testType)
	defer delete(nsidIndex, "test.behavior.nsid")
	defer delete(behaviorRegistry, testType)

	assert.NotNil(t, BehaviorByNSID("test.behavior.nsid"))
	assert.Nil(t, BehaviorByNSID("test.behavior.unknown"))
}
