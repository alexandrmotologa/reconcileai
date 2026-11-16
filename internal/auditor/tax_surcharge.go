package auditor

import (
	"fmt"
	"math"
	"strings"

	"github.com/alexandrmotologa/reconcileai/internal/config"
)

// AuditSurcharges inspects unapproved freight or handling surcharges.
func AuditSurcharges(dossier *config.Dossier) []*config.Discrepancy {
	var discrepancies []*config.Discrepancy

	invHeader := dossier.Invoice.Header
	poHeader := dossier.PurchaseOrder.Header

	// Check freight authorization
	if invHeader.FreightTotal > 0 {
		poAllowsFreight := poHeader.FreightTotal > 0 ||
			strings.Contains(strings.ToLower(poHeader.PaymentTerms), "prepay and add") ||
			strings.Contains(strings.ToLower(poHeader.PaymentTerms), "fob origin")

		if !poAllowsFreight {
			discrepancies = append(discrepancies, &config.Discrepancy{
				ID:             "DISC-SURCHARGE-FREIGHT",
				Type:           config.DiscrepancyUnauthorizedFee,
				Severity:       config.SeverityCritical,
				Title:          "Unauthorized Freight / Shipping Surcharge",
				Description:    fmt.Sprintf("Invoice added $%.2f freight surcharge contrary to PO terms (agreed FOB Destination / freight prepaid).", invHeader.FreightTotal),
				ExpectedValue:  0.0,
				ActualValue:    invHeader.FreightTotal,
				VarianceAmount: invHeader.FreightTotal,
				Currency:       invHeader.Currency,
				TargetDocument: config.DocTypeInvoice,
				PageNumber:     1,
				BoundingBox: config.BoundingBox{
					Page:   1,
					X:      350,
					Y:      100,
					Width:  200,
					Height: 18,
				},
			})
		}
	}

	return discrepancies
}

// AuditTax checks for sales tax or VAT calculation deviations.
func AuditTax(dossier *config.Dossier, tolerance float64) *config.Discrepancy {
	invHeader := dossier.Invoice.Header
	if invHeader.TaxTotal <= 0 {
		return nil
	}

	// Calculate weighted expected tax from invoice line items
	var expectedTax float64
	for _, line := range dossier.Invoice.Lines {
		if line.TaxRate > 0 {
			expectedTax += line.TotalPrice * (line.TaxRate / 100.0)
		}
	}

	if expectedTax > 0 {
		diff := math.Abs(invHeader.TaxTotal - expectedTax)
		if diff > tolerance {
			return &config.Discrepancy{
				ID:             "DISC-TAX-VARIANCE",
				Type:           config.DiscrepancyArithmeticMismatch,
				Severity:       config.SeverityWarning,
				Title:          "Tax Amount Calculation Mismatch",
				Description:    fmt.Sprintf("Billed tax ($%.2f) diverges from calculated rate-based tax ($%.2f) by $%.2f.", invHeader.TaxTotal, expectedTax, diff),
				ExpectedValue:  expectedTax,
				ActualValue:    invHeader.TaxTotal,
				VarianceAmount: diff,
				Currency:       invHeader.Currency,
				TargetDocument: config.DocTypeInvoice,
				PageNumber:     1,
			}
		}
	}

	return nil
}
