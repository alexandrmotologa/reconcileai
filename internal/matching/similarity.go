package matching

import (
	"math"
	"strings"
	"unicode"
)

// JaroWinkler calculates the Jaro-Winkler distance between two strings (0.0 to 1.0).
func JaroWinkler(s1, s2 string) float64 {
	jaro := jaroDistance(s1, s2)
	if jaro < 0.7 {
		return jaro
	}

	// Prefix scale (up to 4 chars)
	prefixLen := 0
	maxPrefix := 4
	minLen := len(s1)
	if len(s2) < minLen {
		minLen = len(s2)
	}
	if maxPrefix > minLen {
		maxPrefix = minLen
	}

	for i := 0; i < maxPrefix; i++ {
		if s1[i] == s2[i] {
			prefixLen++
		} else {
			break
		}
	}

	// Standard scaling factor is 0.1
	return jaro + (float64(prefixLen) * 0.1 * (1.0 - jaro))
}

func jaroDistance(s1, s2 string) float64 {
	l1 := len(s1)
	l2 := len(s2)
	if l1 == 0 && l2 == 0 {
		return 1.0
	}
	if l1 == 0 || l2 == 0 {
		return 0.0
	}

	matchDistance := (int(math.Max(float64(l1), float64(l2))) / 2) - 1
	if matchDistance < 0 {
		matchDistance = 0
	}

	s1Matches := make([]bool, l1)
	s2Matches := make([]bool, l2)

	matches := 0
	for i := 0; i < l1; i++ {
		start := int(math.Max(0, float64(i-matchDistance)))
		end := int(math.Min(float64(i+matchDistance+1), float64(l2)))

		for j := start; j < end; j++ {
			if s2Matches[j] || s1[i] != s2[j] {
				continue
			}
			s1Matches[i] = true
			s2Matches[j] = true
			matches++
			break
		}
	}

	if matches == 0 {
		return 0.0
	}

	transpositions := 0
	k := 0
	for i := 0; i < l1; i++ {
		if !s1Matches[i] {
			continue
		}
		for !s2Matches[k] {
			k++
		}
		if s1[i] != s2[k] {
			transpositions++
		}
		k++
	}

	m := float64(matches)
	return (m/float64(l1) + m/float64(l2) + (m-float64(transpositions/2))/m) / 3.0
}

// TokenJaccard calculates Jaccard similarity across word token sets.
func TokenJaccard(s1, s2 string) float64 {
	t1 := tokenize(s1)
	t2 := tokenize(s2)

	if len(t1) == 0 && len(t2) == 0 {
		return 1.0
	}
	if len(t1) == 0 || len(t2) == 0 {
		return 0.0
	}

	set1 := make(map[string]struct{}, len(t1))
	for _, tok := range t1 {
		set1[tok] = struct{}{}
	}

	set2 := make(map[string]struct{}, len(t2))
	for _, tok := range t2 {
		set2[tok] = struct{}{}
	}

	intersection := 0
	matchedSet2 := make(map[string]bool)

	for k1 := range set1 {
		if _, ok := set2[k1]; ok {
			intersection++
			matchedSet2[k1] = true
			continue
		}
		// Soft match fallback
		for k2 := range set2 {
			if !matchedSet2[k2] && JaroWinkler(k1, k2) >= 0.88 {
				intersection++
				matchedSet2[k2] = true
				break
			}
		}
	}

	union := len(set1) + len(set2) - intersection
	if union <= 0 {
		return 0.0
	}
	return float64(intersection) / float64(union)
}

func tokenize(s string) []string {
	f := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	}
	raw := strings.FieldsFunc(strings.ToLower(s), f)
	var tokens []string
	for _, tok := range raw {
		// Stem plural 's' or 'es' for robust word matching
		stemmed := tok
		if len(stemmed) > 4 && strings.HasSuffix(stemmed, "es") {
			stemmed = strings.TrimSuffix(stemmed, "es")
		} else if len(stemmed) > 3 && strings.HasSuffix(stemmed, "s") {
			stemmed = strings.TrimSuffix(stemmed, "s")
		}

		if len(stemmed) > 1 && !isStopword(stemmed) {
			tokens = append(tokens, stemmed)
		}
	}
	return tokens
}

var commonStopwords = map[string]struct{}{
	"the": {}, "and": {}, "for": {}, "with": {}, "item": {},
	"part": {}, "per": {}, "unit": {}, "type": {}, "grade": {},
}

func isStopword(w string) bool {
	_, ok := commonStopwords[w]
	return ok
}

// ComputeItemSimilarity evaluates cross-field alignment between a PO item and Invoice item.
func ComputeItemSimilarity(poSKU, poDesc, invSKU, invDesc string) (float64, string) {
	poSKUClean := strings.ToUpper(strings.TrimSpace(poSKU))
	invSKUClean := strings.ToUpper(strings.TrimSpace(invSKU))

	// Exact SKU match
	if poSKUClean != "" && poSKUClean == invSKUClean {
		return 1.0, "Exact SKU match"
	}

	// Partial SKU match (e.g. vendor added prefix or suffix)
	if poSKUClean != "" && invSKUClean != "" {
		if strings.Contains(invSKUClean, poSKUClean) || strings.Contains(poSKUClean, invSKUClean) {
			return 0.92, "Sub-string SKU alignment"
		}
	}

	// Description semantic similarity
	poDescLower := strings.ToLower(strings.TrimSpace(poDesc))
	invDescLower := strings.ToLower(strings.TrimSpace(invDesc))

	jw := JaroWinkler(poDescLower, invDescLower)
	jaccard := TokenJaccard(poDescLower, invDescLower)

	combined := (jw * 0.45) + (jaccard * 0.55)

	reason := "Fuzzy text and token match"
	if combined > 0.85 {
		reason = "High confidence description match"
	} else if combined > 0.65 {
		reason = "Moderate confidence token overlap"
	}

	return combined, reason
}
