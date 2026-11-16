package auditor

import (
	"fmt"
	"math"

	"github.com/alexandrmotologa/reconcileai/internal/config"
)

// AuditFaceArithmetic validates that line item totals equal subtotal and grand total.
func AuditFaceArithmetic(invoice *config.Document) []*config.Discrepancy {
	var discrepancies []*config.Discrepancy

	var computedSubtotal float64
	for _, line := range invoice.Lines {
		computedLineTotal := math.Round(line.Quantity*line.UnitPrice*100) / 100
		if math.Abs(computedLineTotal-line.TotalPrice) > 0.05 && line.TotalPrice > 0 {
			discrepancies = append(discrepancies, &config.Discrepancy{
				ID:             fmt.Sprintf("DISC-ARITH-LINE-%s", line.ID),
				Type:           config.DiscrepancyArithmeticMismatch,
				Severity:       config.SeverityCritical,
				Title:          fmt.Sprintf("Line Arithmetic Error on SKU %s", line.SKU),
				Description:    fmt.Sprintf("Line multiplication (%.2f @ $%.4f = $%.2f) does not equal billed line total of $%.2f.", line.Quantity, line.UnitPrice, computedLineTotal, line.TotalPrice),
				ItemSKU:        line.SKU,
				ExpectedValue:  computedLineTotal,
				ActualValue:    line.TotalPrice,
				VarianceAmount: math.Abs(computedLineTotal - line.TotalPrice),
				Currency:       invoice.Header.Currency,
				TargetDocument: config.DocTypeInvoice,
				PageNumber:     line.BoundingBox.Page,
				BoundingBox:    line.BoundingBox,
			})
		}
		computedSubtotal += line.TotalPrice
	}

	computedSubtotal = math.Round(computedSubtotal*100) / 100

	// Check subtotal against header subtotal
	if invoice.Header.Subtotal > 0 {
		diff := math.Abs(invoice.Header.Subtotal - computedSubtotal)
		if diff > 0.05 {
			discrepancies = append(discrepancies, &config.Discrepancy{
				ID:             "DISC-ARITH-SUBTOTAL",
				Type:           config.DiscrepancyArithmeticMismatch,
				Severity:       config.SeverityCritical,
				Title:          "Subtotal Summation Deviation",
				Description:    fmt.Sprintf("Invoice subtotal ($%.2f) does not equal the sum of lines ($%.2f).", invoice.Header.Subtotal, computedSubtotal),
				ExpectedValue:  computedSubtotal,
				ActualValue:    invoice.Header.Subtotal,
				VarianceAmount: diff,
				Currency:       invoice.Header.Currency,
				TargetDocument: config.DocTypeInvoice,
				PageNumber:     1,
			})
		}
	}

	// Check grand total
	expectedGrandTotal := math.Round((invoice.Header.Subtotal+invoice.Header.TaxTotal+invoice.Header.FreightTotal)*100) / 100
	if invoice.Header.TotalAmount > 0 {
		diff := math.Abs(invoice.Header.TotalAmount - expectedGrandTotal)
		if diff > 0.05 {
			discrepancies = append(discrepancies, &config.Discrepancy{
				ID:             "DISC-ARITH-GRANDTOTAL",
				Type:           config.DiscrepancyArithmeticMismatch,
				Severity:       config.SeverityCritical,
				Title:          "Invoice Grand Total Calculation Error",
				Description:    fmt.Sprintf("Billed grand total ($%.2f) diverges from calculated total ($%.2f = subtotal + tax + freight).", invoice.Header.TotalAmount, expectedGrandTotal),
				ExpectedValue:  expectedGrandTotal,
				ActualValue:    invoice.Header.TotalAmount,
				VarianceAmount: diff,
				Currency:       invoice.Header.Currency,
				TargetDocument: config.DocTypeInvoice,
				PageNumber:     1,
			})
		}
	}

	return discrepancies
}
