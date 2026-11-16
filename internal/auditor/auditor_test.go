package auditor

import (
	"testing"

	"github.com/alexandrmotologa/reconcileai/internal/config"
	"github.com/alexandrmotologa/reconcileai/internal/synth"
)

func TestReconcileCleanMatch(t *testing.T) {
	dossier, err := synth.BuildCleanMatchDossier(t.TempDir())
	if err != nil {
		t.Fatalf("failed to build clean match: %v", err)
	}

	cfg := config.DefaultConfig()
	summary := ReconcileDossier(dossier, cfg)

	if summary.MatchStatus != "APPROVED" {
		t.Errorf("expected APPROVED, got %s", summary.MatchStatus)
	}
	if summary.PreventedOvercharge != 0 {
		t.Errorf("expected 0 prevented overcharge, got %.2f", summary.PreventedOvercharge)
	}
	if len(summary.Discrepancies) != 0 {
		t.Errorf("expected 0 discrepancies, got %d", len(summary.Discrepancies))
	}
}

func TestReconcileQuantityOverbilling(t *testing.T) {
	dossier, err := synth.BuildQtyOverbillingDossier(t.TempDir())
	if err != nil {
		t.Fatalf("failed to build qty overbilling dossier: %v", err)
	}

	cfg := config.DefaultConfig()
	summary := ReconcileDossier(dossier, cfg)

	if summary.MatchStatus != "FLAGGED_DISCREPANCY" {
		t.Errorf("expected FLAGGED_DISCREPANCY, got %s", summary.MatchStatus)
	}
	if summary.PreventedOvercharge <= 0 {
		t.Errorf("expected positive prevented overcharge, got %.2f", summary.PreventedOvercharge)
	}

	foundQtyDisc := false
	for _, d := range summary.Discrepancies {
		if d.Type == config.DiscrepancyQtyOverbilled {
			foundQtyDisc = true
			if d.VarianceAmount != 2500.00 {
				t.Errorf("expected variance 2500.00, got %.2f", d.VarianceAmount)
			}
		}
	}

	if !foundQtyDisc {
		t.Errorf("expected to find DiscrepancyQtyOverbilled")
	}
}

func TestReconcilePriceVariance(t *testing.T) {
	dossier, err := synth.BuildPriceVarianceDossier(t.TempDir())
	if err != nil {
		t.Fatalf("failed to build price variance dossier: %v", err)
	}

	cfg := config.DefaultConfig()
	summary := ReconcileDossier(dossier, cfg)

	if summary.MatchStatus != "FLAGGED_DISCREPANCY" {
		t.Errorf("expected FLAGGED_DISCREPANCY, got %s", summary.MatchStatus)
	}

	foundPriceDisc := false
	for _, d := range summary.Discrepancies {
		if d.Type == config.DiscrepancyPriceVariance {
			foundPriceDisc = true
		}
	}

	if !foundPriceDisc {
		t.Errorf("expected to find DiscrepancyPriceVariance")
	}
}

func TestReconcileUnauthorizedSurcharge(t *testing.T) {
	dossier, err := synth.BuildUnauthorizedFeeDossier(t.TempDir())
	if err != nil {
		t.Fatalf("failed to build unauthorized fee dossier: %v", err)
	}

	cfg := config.DefaultConfig()
	summary := ReconcileDossier(dossier, cfg)

	if summary.MatchStatus != "FLAGGED_DISCREPANCY" {
		t.Errorf("expected FLAGGED_DISCREPANCY, got %s", summary.MatchStatus)
	}

	foundFee := false
	for _, d := range summary.Discrepancies {
		if d.Type == config.DiscrepancyUnauthorizedFee {
			foundFee = true
			if d.VarianceAmount != 450.00 {
				t.Errorf("expected fee variance 450.00, got %.2f", d.VarianceAmount)
			}
		}
	}

	if !foundFee {
		t.Errorf("expected to find DiscrepancyUnauthorizedFee")
	}
}
