package matching

import (
	"regexp"
	"strings"
)

var legalEntitySuffixes = regexp.MustCompile(`(?i)\b(inc|incorporated|corp|corporation|llc|ltd|limited|gmbh|sarl|sa|co|company)\b|\.`)

// CleanVendorName strips legal entity suffixes, punctuation, and excess spaces.
func CleanVendorName(name string) string {
	cleaned := legalEntitySuffixes.ReplaceAllString(strings.ToLower(name), "")
	return strings.Join(strings.Fields(cleaned), " ")
}

// CompareVendorEntities checks if two vendor names refer to the same corporate entity.
func CompareVendorEntities(poVendor, invVendor string) (float64, bool) {
	c1 := CleanVendorName(poVendor)
	c2 := CleanVendorName(invVendor)

	if c1 == "" || c2 == "" {
		return 0.5, true // Tolerant if missing
	}

	if c1 == c2 {
		return 1.0, true
	}

	jw := JaroWinkler(c1, c2)
	jaccard := TokenJaccard(c1, c2)
	score := (jw * 0.5) + (jaccard * 0.5)

	return score, score >= 0.70
}

// VerifyPOReference checks whether the invoice cites the expected PO number.
func VerifyPOReference(poID, invRefID string) bool {
	cleanPO := strings.ToUpper(strings.TrimSpace(poID))
	cleanRef := strings.ToUpper(strings.TrimSpace(invRefID))

	if cleanPO == "" || cleanRef == "" {
		return false
	}
	return cleanPO == cleanRef || strings.Contains(cleanRef, cleanPO)
}
