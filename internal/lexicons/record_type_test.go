package lexicons

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecordTypeString(t *testing.T) {
	tests := []struct {
		rt       RecordType
		expected string
	}{
		{RecordTypeBean, "bean"},
		{RecordTypeBrew, "brew"},
		{RecordTypeBrewer, "brewer"},
		{RecordTypeGrinder, "grinder"},
		{RecordTypeLike, "like"},
		{RecordTypeRoaster, "roaster"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.rt.String())
		})
	}
}

func TestRecordTypeDisplayName(t *testing.T) {
	tests := []struct {
		rt       RecordType
		expected string
	}{
		{RecordTypeBean, "Bean"},
		{RecordTypeBrew, "Brew"},
		{RecordTypeBrewer, "Brewer"},
		{RecordTypeGrinder, "Grinder"},
		{RecordTypeLike, "Like"},
		{RecordTypeRoaster, "Roaster"},
		{RecordType("unknown"), "unknown"}, // Fallback to string value
	}

	for _, tt := range tests {
		t.Run(string(tt.rt), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.rt.DisplayName())
		})
	}
}
