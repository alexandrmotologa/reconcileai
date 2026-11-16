package matching

import (
	"testing"
)

func TestItemSimilarity(t *testing.T) {
	tests := []struct {
		name       string
		poSKU      string
		poDesc     string
		invSKU     string
		invDesc    string
		minScore   float64
		wantReason string
	}{
		{
			name:     "Exact SKU Match",
			poSKU:    "HYD-VALVE-50",
			poDesc:   "High Pressure Hydraulic Solenoid Valve",
			invSKU:   "HYD-VALVE-50",
			invDesc:  "Solenoid Valve 24V",
			minScore: 0.99,
		},
		{
			name:     "Substring SKU Match",
			poSKU:    "VALVE-50",
			poDesc:   "Hydraulic Solenoid Valve",
			invSKU:   "HYD-VALVE-50-A",
			invDesc:  "Solenoid Valve",
			minScore: 0.90,
		},
		{
			name:     "Description Token Match with Different SKU",
			poSKU:    "INTERNAL-901",
			poDesc:   "Industrial Thermoplastic Resin Pellet A-Grade",
			invSKU:   "POLY-RESIN-01",
			invDesc:  "Resin Pellets Thermoplastic Grade-A",
			minScore: 0.65,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score, _ := ComputeItemSimilarity(tc.poSKU, tc.poDesc, tc.invSKU, tc.invDesc)
			if score < tc.minScore {
				t.Errorf("expected score >= %.2f, got %.2f", tc.minScore, score)
			}
		})
	}
}

func TestVendorEntityAlignment(t *testing.T) {
	v1 := "Logix Hydraulic Systems, LLC."
	v2 := "LOGIX HYDRAULIC SYSTEMS INC"

	score, matched := CompareVendorEntities(v1, v2)
	if !matched {
		t.Errorf("expected vendor entities to match, score=%.2f", score)
	}
}
