package auditor

import (
	"fmt"
	"math"

	"github.com/alexandrmotologa/reconcileai/internal/config"
)

// AuditQuantity checks whether billed quantities exceed verified warehouse goods receipts.
func AuditQuantity(pair *config.MatchedLinePair, invLine *config.LineItem, tolerance float64) *config.Discrepancy {
	if pair.POLineID == "" {
		// Unmatched item handled separately
		return nil
	}

	diff := pair.InvoicedQty - pair.ReceivedQty

	if diff > tolerance {
		unitPrice := pair.InvoicedPrice
		if unitPrice == 0 {
			unitPrice = pair.POPrice
		}
		varianceAmt := math.Round(diff*unitPrice*100) / 100

		box := config.BoundingBox{}
		if invLine != nil {
			box = invLine.BoundingBox
		}

		return &config.Discrepancy{
			ID:             fmt.Sprintf("DISC-QTY-%s", pair.InvoiceLineID),
			Type:           config.DiscrepancyQtyOverbilled,
			Severity:       config.SeverityCritical,
			Title:          fmt.Sprintf("Quantity Overbilled on SKU %s", pair.ItemSKU),
			Description:    fmt.Sprintf("Vendor invoiced %.2f units but warehouse only verified intake of %.2f units (short delivery of %.2f units).", pair.InvoicedQty, pair.ReceivedQty, diff),
			ItemSKU:        pair.ItemSKU,
			ExpectedValue:  pair.ReceivedQty,
			ActualValue:    pair.InvoicedQty,
			VarianceAmount: varianceAmt,
			Currency:       "USD",
			TargetDocument: config.DocTypeInvoice,
			PageNumber:     box.Page,
			BoundingBox:    box,
		}
	}

	return nil
}
