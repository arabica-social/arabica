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

func TestOolongRecordTypes(t *testing.T) {
	cases := []struct {
		raw   string
		want  RecordType
		label string
	}{
		{"oolong-tea", RecordTypeOolongTea, "Tea"},
		{"oolong-brew", RecordTypeOolongBrew, "Tea Brew"},
		{"oolong-brewer", RecordTypeOolongBrewer, "Tea Brewer"},
		{"oolong-recipe", RecordTypeOolongRecipe, "Tea Recipe"},
		{"oolong-vendor", RecordTypeOolongVendor, "Tea Vendor"},
		{"oolong-cafe", RecordTypeOolongCafe, "Tea Cafe"},
		{"oolong-drink", RecordTypeOolongDrink, "Tea Drink"},
		{"oolong-comment", RecordTypeOolongComment, "Tea Comment"},
		{"oolong-like", RecordTypeOolongLike, "Tea Like"},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			assert.Equal(t, tc.want, ParseRecordType(tc.raw))
			assert.Equal(t, tc.label, tc.want.DisplayName())
		})
	}
}

func TestArabicaRecordTypesUnchanged(t *testing.T) {
	assert.Equal(t, RecordTypeBean, ParseRecordType("bean"))
	assert.Equal(t, "Bean", RecordTypeBean.DisplayName())
}
