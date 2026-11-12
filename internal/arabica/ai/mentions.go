package ai

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Context relevance ranks provider retention. Identity matching below is a separate,
// request-scoped policy checked against the complete own library.
func contextTerms(text string) map[string]bool {
	terms := map[string]bool{}
	stop := map[string]bool{"coffee": true, "bean": true, "brew": true, "brewing": true, "recipe": true, "with": true, "from": true, "for": true, "the": true, "a": true, "an": true, "and": true, "use": true, "my": true, "please": true}
	for _, word := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) }) {
		word = strings.TrimSuffix(word, "s")
		if word != "" && !stop[word] {
			terms[word] = true
		}
	}
	return terms
}
func contextRelevance(terms map[string]bool, labels ...string) int {
	seen := map[string]bool{}
	for _, label := range labels {
		for word := range contextTerms(label) {
			if terms[word] {
				seen[word] = true
				if len(seen) == 4 {
					return 4
				}
			}
		}
	}
	return len(seen)
}

func normalized(s string) string { return strings.Join(strings.Fields(strings.ToLower(s)), " ") }
func wordRune(r rune) bool       { return unicode.IsLetter(r) || unicode.IsNumber(r) || r == '_' }

// exactMention is literal normalized matching with word boundaries, not fuzzy
// matching or model confidence. Authored text is not rewritten by this check.
func exactMention(message, label string) bool {
	message = normalized(message)
	label = normalized(label)
	if label == "" {
		return false
	}
	start := 0
	for start <= len(message)-len(label) {
		offset := strings.Index(message[start:], label)
		if offset < 0 {
			return false
		}
		offset += start
		end := offset + len(label)
		beforeOK, afterOK := true, true
		if offset > 0 {
			r, _ := utf8.DecodeLastRuneInString(message[:offset])
			beforeOK = !wordRune(r)
		}
		if end < len(message) {
			r, _ := utf8.DecodeRuneInString(message[end:])
			afterOK = !wordRune(r)
		}
		if beforeOK && afterOK {
			return true
		}
		start = offset + 1
	}
	return false
}
func (c catalog) explicitlyNamed(key, rkey, message string) bool {
	for _, item := range c {
		if item.Kind == referenceKinds[key] && item.rkey == rkey {
			// A bean named exactly after its roaster is not independent name
			// evidence: a roaster-only request must still honor bag ambiguity/status.
			named := item.uniqueLabel && exactMention(message, item.Label)
			if item.Kind == "bean" && item.RoasterLabel != "" && normalized(item.Label) == normalized(item.RoasterLabel) {
				named = false
			}
			return named ||
				(item.Kind == "bean" && item.Closed != nil && !*item.Closed && item.OpenRoasterMatches == 1 && exactMention(message, item.RoasterLabel))
		}
	}
	return false
}

// identityScore permits meaningful short names and one small typo, not arbitrary
// substrings. Exact names outrank partial overlap; tiny/generic tokens carry no identity.
func identityScore(message, label string) int {
	if exactMention(message, label) {
		return 100
	}
	query, names := identityTerms(message), identityTerms(label)
	score := 0
	for name := range names {
		if query[name] {
			score += 2
			continue
		}
		if len(name) < 5 {
			continue
		}
		for word := range query {
			if len(word) >= 5 && editOne(word, name) {
				score++
				break
			}
		}
	}
	return score
}
func identityTerms(text string) map[string]bool {
	terms := contextTerms(compact(text, 16000))
	for word := range terms {
		switch word {
		case "own", "brewer", "grinder", "paper", "process", "bag", "open", "closed", "saved", "filter", "new", "old", "use", "water", "coffee", "bean", "brew":
			delete(terms, word)
		default:
			if len(word) < 3 {
				delete(terms, word)
			}
		}
	}
	return terms
}

// editOne checks insertion, deletion, substitution or adjacent transposition in
// linear space/time; tokens/labels are bounded before reaching this helper.
func editOne(a, b string) bool {
	x, y := []rune(a), []rune(b)
	if len(x) > len(y) {
		x, y = y, x
	}
	if len(y)-len(x) > 1 {
		return false
	}
	for i := 0; i < len(x); i++ {
		if x[i] == y[i] {
			continue
		}
		if len(x) == len(y) {
			if string(x[i+1:]) == string(y[i+1:]) {
				return true
			}
			return i+1 < len(x) && x[i] == y[i+1] && x[i+1] == y[i] && string(x[i+2:]) == string(y[i+2:])
		}
		return string(x[i:]) == string(y[i+1:])
	}
	return true
}

// safeGroundMatch is computed from the latest user turn against full rows, not
// the retained subset, provider prose, confidence, or the number of choices.
func (c catalog) safeGroundMatch(key, rkey, message string, current map[string]any) bool {
	for _, item := range c {
		if item.Kind != referenceKinds[key] || item.rkey != rkey {
			continue
		}
		if item.pourCountMismatch {
			return false
		}
		if item.recentGround && (current[key] == nil || current[key] == "") {
			return true
		}
		if item.RequestMatch {
			return true
		}
		if item.SoleOpenDefault && (current[key] == nil || current[key] == "") {
			return true
		}
		return c.explicitlyNamed(key, rkey, message)
	}
	return false
}

// recipeIdentityScore requires a distinctive recipe-name word, not just its
// equipment. Once anchored, "60" may distinguish "Fast V60" from "Fast AeroPress".
func recipeIdentityScore(message, label, brewer string) int {
	if exactMention(message, label) {
		return 100
	}
	equipment := identityTerms(brewer)
	anchored := false
	for word := range identityTerms(label) {
		if equipment[word] {
			continue
		}
		letters := true
		for _, r := range word {
			if !unicode.IsLetter(r) {
				letters = false
				break
			}
		}
		if letters && identityScore(message, word) > 0 {
			anchored = true
			break
		}
	}
	if !anchored {
		return 0
	}
	score := identityScore(message, label)
	query := contextTerms(message)
	for word := range contextTerms(label) {
		if len(word) < 3 || word[0] != 'v' || query[word] || !query[word[1:]] {
			continue
		}
		digits := true
		for _, r := range word[1:] {
			if !unicode.IsDigit(r) {
				digits = false
				break
			}
		}
		if digits {
			score += 2
		}
	}
	return score
}

// specifiedPourCount reads identity phrases, not arbitrary numbers: "3-pour",
// "3 pours" and "three pour" agree; "1 bloom", "15g" and "Orea 01" do not count.
// Conflicting count phrases leave interpretation to the model and user.
func specifiedPourCount(text string) int {
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) })
	numbers := map[string]int{"one": 1, "two": 2, "three": 3, "four": 4, "five": 5, "six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10}
	count := 0
	for i := 0; i+1 < len(words); i++ {
		if strings.TrimSuffix(words[i+1], "s") != "pour" {
			continue
		}
		n := numbers[words[i]]
		if n == 0 {
			n, _ = strconv.Atoi(words[i])
		}
		if n < 1 || n > 100 {
			continue
		}
		if count != 0 && count != n {
			return 0
		}
		count = n
	}
	return count
}
