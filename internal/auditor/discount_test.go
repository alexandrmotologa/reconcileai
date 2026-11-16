package auditor

import (
	"testing"
)

func TestParsePaymentTerms(t *testing.T) {
	tests := []struct {
		terms       string
		wantPct     float64
		wantDays    int
		wantNet     int
		wantFound   bool
	}{
		{"2/10 Net 30", 2.0, 10, 30, true},
		{"2/10 Net 30 FOB Destination", 2.0, 10, 30, true},
		{"1.5/15 Net 45", 1.5, 15, 45, true},
		{"2% 10 Net 30", 2.0, 10, 30, true},
		{"Net 30", 0, 0, 0, false},
		{"Due on receipt", 0, 0, 0, false},
		{"", 0, 0, 0, false},
	}

	for _, tt := range tests {
		pct, discDays, netDays, ok := ParsePaymentTerms(tt.terms)
		if ok != tt.wantFound {
			t.Errorf("ParsePaymentTerms(%q) found = %v, want %v", tt.terms, ok, tt.wantFound)
			continue
		}
		if ok {
			if pct != tt.wantPct || discDays != tt.wantDays || netDays != tt.wantNet {
				t.Errorf("ParsePaymentTerms(%q) = (%v, %v, %v), want (%v, %v, %v)",
					tt.terms, pct, discDays, netDays, tt.wantPct, tt.wantDays, tt.wantNet)
			}
		}
	}
}

func TestCalculateEarlyDiscount(t *testing.T) {
	disc := CalculateEarlyDiscount("2/10 Net 30", "2026-09-01", 5238.00)
	if disc == nil {
		t.Fatalf("Expected EarlyDiscount, got nil")
	}

	// 2% of 5238.00 = 104.76
	if disc.DiscountAmount != 104.76 {
		t.Errorf("Expected discount amount 104.76, got %.2f", disc.DiscountAmount)
	}

	// Deadline: 2026-09-01 + 10 days = 2026-09-11
	if disc.DiscountDeadline != "2026-09-11" {
		t.Errorf("Expected deadline 2026-09-11, got %s", disc.DiscountDeadline)
	}

	// Annualized ROI for 2/10 Net 30: (2/98) * (365/20) * 100 = 37.24%
	if disc.AnnualizedROI < 37.0 || disc.AnnualizedROI > 37.5 {
		t.Errorf("Expected ROI ~37.24%%, got %.2f%%", disc.AnnualizedROI)
	}
}
