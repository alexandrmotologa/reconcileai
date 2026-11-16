package ingestion

import (
	"math"
	"testing"
)

func TestUOMConversions(t *testing.T) {
	tests := []struct {
		name     string
		qty      float64
		fromUOM  string
		toUOM    string
		wantQty  float64
		wantOK   bool
	}{
		{"Dozen to Each", 10, "DZ", "EA", 120, true},
		{"Each to Dozen", 24, "EA", "DZ", 2, true},
		{"Box to Each", 2, "BOX", "EA", 48, true},
		{"Pounds to Kilograms", 10, "LB", "KG", 4.5359237, true},
		{"Incompatible categories", 10, "KG", "EA", 10, false},
		{"Unknown unit", 5, "XYZ_UNKNOWN", "EA", 5, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotQty, ok := ConvertQuantity(tc.qty, tc.fromUOM, tc.toUOM)
			if ok != tc.wantOK {
				t.Fatalf("expected ok=%v, got %v", tc.wantOK, ok)
			}
			if ok && math.Abs(gotQty-tc.wantQty) > 0.001 {
				t.Errorf("expected qty=%.4f, got %.4f", tc.wantQty, gotQty)
			}
		})
	}
}
