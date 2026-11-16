package auditor

import (
	"testing"

	"github.com/alexandrmotologa/reconcileai/internal/config"
)

func TestDuplicateLedger(t *testing.T) {
	ledger := NewDuplicateLedger()

	dossier1 := &config.Dossier{
		ID: "dossier_original",
		Invoice: config.Document{
			Header: config.DocumentHeader{
				DocumentID:   "INV-99901",
				VendorID:     "VEND-ABC",
				VendorName:   "ABC Supply Co",
				IssueDate:    "2026-09-01",
				TotalAmount:  1500.00,
				Currency:     "USD",
			},
		},
	}

	// 1. Initial check - should not be duplicate
	if disc := ledger.CheckDuplicate(dossier1); disc != nil {
		t.Fatalf("Expected no duplicate for new dossier, got %v", disc)
	}

	// Record dossier1
	ledger.Record(dossier1, "hash_original_1234567890")
	if ledger.Count() != 1 {
		t.Fatalf("Expected ledger count 1, got %d", ledger.Count())
	}

	// Same dossier ID re-check should not flag itself
	if disc := ledger.CheckDuplicate(dossier1); disc != nil {
		t.Fatalf("Expected self check to not flag as duplicate, got %v", disc)
	}

	// 2. Exact duplicate check (different dossier ID, same vendor + invoice number)
	dossierExactDup := &config.Dossier{
		ID: "dossier_duplicate_submission",
		Invoice: config.Document{
			Header: config.DocumentHeader{
				DocumentID:   "INV-99901",
				VendorID:     "VEND-ABC",
				VendorName:   "ABC Supply Co",
				IssueDate:    "2026-09-01",
				TotalAmount:  1500.00,
				Currency:     "USD",
			},
		},
	}

	discExact := ledger.CheckDuplicate(dossierExactDup)
	if discExact == nil {
		t.Fatalf("Expected duplicate discrepancy for exact match, got nil")
	}
	if discExact.Type != config.DiscrepancyDuplicateInvoice {
		t.Errorf("Expected type %s, got %s", config.DiscrepancyDuplicateInvoice, discExact.Type)
	}
	if discExact.VarianceAmount != 1500.00 {
		t.Errorf("Expected variance amount 1500.00, got %.2f", discExact.VarianceAmount)
	}

	// 3. Fuzzy duplicate check (different invoice number, but same vendor + amount + issue date)
	dossierFuzzyDup := &config.Dossier{
		ID: "dossier_reissued_invoice",
		Invoice: config.Document{
			Header: config.DocumentHeader{
				DocumentID:   "INV-99901-REV",
				VendorID:     "VEND-ABC",
				VendorName:   "ABC Supply Co",
				IssueDate:    "2026-09-01",
				TotalAmount:  1500.00,
				Currency:     "USD",
			},
		},
	}

	discFuzzy := ledger.CheckDuplicate(dossierFuzzyDup)
	if discFuzzy == nil {
		t.Fatalf("Expected fuzzy duplicate discrepancy, got nil")
	}
	if discFuzzy.Type != config.DiscrepancyDuplicateInvoice {
		t.Errorf("Expected type %s, got %s", config.DiscrepancyDuplicateInvoice, discFuzzy.Type)
	}

	// 4. Distinct invoice from same vendor (different date, different amount)
	dossierDistinct := &config.Dossier{
		ID: "dossier_next_month",
		Invoice: config.Document{
			Header: config.DocumentHeader{
				DocumentID:   "INV-99902",
				VendorID:     "VEND-ABC",
				VendorName:   "ABC Supply Co",
				IssueDate:    "2026-10-01",
				TotalAmount:  2200.00,
				Currency:     "USD",
			},
		},
	}

	if disc := ledger.CheckDuplicate(dossierDistinct); disc != nil {
		t.Fatalf("Expected distinct invoice to not flag as duplicate, got %v", disc)
	}
}
