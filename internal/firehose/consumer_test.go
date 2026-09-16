package firehose

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricOperation(t *testing.T) {
	assert.Equal(t, "create", metricOperation("create"))
	assert.Equal(t, "update", metricOperation("update"))
	assert.Equal(t, "delete", metricOperation("delete"))
	assert.Equal(t, "unknown", metricOperation("surprise"))
}
