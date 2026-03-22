package matching

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatch_ExactName(t *testing.T) {
	candidates := []Candidate{
		{RKey: "abc", Name: "Hario V60 02", Type: "Pour-Over"},
		{RKey: "def", Name: "Chemex", Type: "Pour-Over"},
	}

	result := Match("Hario V60 02", "Pour-Over", candidates)
	assert.NotNil(t, result)
	assert.Equal(t, "abc", result.RKey)
	assert.Equal(t, 1.0, result.Score)
}

func TestMatch_ExactNameCaseInsensitive(t *testing.T) {
	candidates := []Candidate{
		{RKey: "abc", Name: "hario v60 02", Type: "Pour-Over"},
	}

	result := Match("Hario V60 02", "", candidates)
	assert.NotNil(t, result)
	assert.Equal(t, "abc", result.RKey)
	assert.Equal(t, 1.0, result.Score)
}

func TestMatch_SingleTypeMatch(t *testing.T) {
	candidates := []Candidate{
		{RKey: "abc", Name: "My French Press", Type: "French Press"},
		{RKey: "def", Name: "Chemex", Type: "Pour-Over"},
	}

	result := Match("Some Other Press", "French Press", candidates)
	assert.NotNil(t, result)
	assert.Equal(t, "abc", result.RKey)
	assert.Equal(t, 0.5, result.Score)
}

func TestMatch_AmbiguousType_ReturnsNil(t *testing.T) {
	candidates := []Candidate{
		{RKey: "abc", Name: "V60", Type: "Pour-Over"},
		{RKey: "def", Name: "Chemex", Type: "Pour-Over"},
	}

	result := Match("Kalita Wave", "Pour-Over", candidates)
	assert.Nil(t, result)
}

func TestMatch_NoCandidates(t *testing.T) {
	result := Match("V60", "Pour-Over", nil)
	assert.Nil(t, result)
}

func TestMatch_NoMatchAtAll(t *testing.T) {
	candidates := []Candidate{
		{RKey: "abc", Name: "AeroPress", Type: "Immersion"},
	}

	result := Match("V60", "Pour-Over", candidates)
	assert.Nil(t, result)
}

func TestMatch_EmptySourceName_FallsToType(t *testing.T) {
	candidates := []Candidate{
		{RKey: "abc", Name: "My V60", Type: "Pour-Over"},
	}

	result := Match("", "Pour-Over", candidates)
	assert.NotNil(t, result)
	assert.Equal(t, "abc", result.RKey)
	assert.Equal(t, 0.5, result.Score)
}

func TestMatch_EmptySourceType_NameOnly(t *testing.T) {
	candidates := []Candidate{
		{RKey: "abc", Name: "V60", Type: "Pour-Over"},
	}

	result := Match("V60", "", candidates)
	assert.NotNil(t, result)
	assert.Equal(t, "abc", result.RKey)
	assert.Equal(t, 1.0, result.Score)
}
