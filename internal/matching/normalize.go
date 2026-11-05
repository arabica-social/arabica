package matching

import (
	"regexp"
	"strings"
)

// Normalize lowercases, trims whitespace, and collapses internal whitespace.
func Normalize(s string) string {
	return CollapseSpaces(strings.ToLower(strings.TrimSpace(s)))
}

// Common suffixes stripped during fuzzy name normalization for roasters/brewers.
// Order matters: longer suffixes first to avoid partial stripping.
var commonSuffixes = []string{
	"coffee roasters",
	"coffee roasting",
	"coffee company",
	"coffee co",
	"roasting company",
	"roasting co",
	"roasters",
	"roasting",
	"coffee",
	"co.",
}

// FuzzyName normalizes a name by lowercasing, stripping common
// coffee-industry suffixes, punctuation, and extra whitespace. Lets
// "Counter Culture Coffee" merge with "Counter Culture" while keeping
// genuinely different names apart.
func FuzzyName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))

	for _, suffix := range commonSuffixes {
		if strings.HasSuffix(s, suffix) {
			s = strings.TrimSpace(s[:len(s)-len(suffix)])
			break // only strip one suffix
		}
	}

	s = stripPunctuation(s)
	return CollapseSpaces(s)
}

var nonAlphanumSpace = regexp.MustCompile(`[^a-z0-9\s]`)

func stripPunctuation(s string) string {
	return nonAlphanumSpace.ReplaceAllString(s, "")
}

var multiSpace = regexp.MustCompile(`\s+`)

// CollapseSpaces trims and collapses runs of internal whitespace.
func CollapseSpaces(s string) string {
	return strings.TrimSpace(multiSpace.ReplaceAllString(s, " "))
}
