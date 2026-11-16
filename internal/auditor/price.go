package auditor

import (
	"fmt"
	"math"

	"github.com/alexandrmotologa/reconcileai/internal/config"
)

// AuditPrice checks if billed unit price exceeds contract rate beyond permitted tolerance.
func AuditPrice(pair *config.MatchedLinePair, invLine *config.LineItem, cfg config.ToleranceConfig) *config.Discrepancy {
	if pair.POLineID == "" || pair.POPrice <= 0 {
		return nil
	}

	deltaPrice := pair.InvoicedPrice - pair.POPrice

	if deltaPrice <= 0 {
		return nil // Supplier billed equal or lower price
	}

	allowedAbsolute := cfg.MaxPriceVarianceAbsolute
	allowedRelative := (cfg.MaxPriceVariancePercent / 100.0) * pair.POPrice
	maxAllowed := math.Max(allowedAbsolute, allowedRelative)

	if deltaPrice > maxAllowed {
		totalLineOvercharge := math.Round(deltaPrice*pair.InvoicedQty*100) / 100

		box := config.BoundingBox{}
		if invLine != nil {
			box = invLine.BoundingBox
		}

		return &config.Discrepancy{
			ID:             fmt.Sprintf("DISC-PRICE-%s", pair.InvoiceLineID),
			Type:           config.DiscrepancyPriceVariance,
			Severity:       config.SeverityCritical,
			Title:          fmt.Sprintf("Unit Price Variance on SKU %s", pair.ItemSKU),
			Description:    fmt.Sprintf("Vendor billed $%.4f/unit vs agreed PO contract rate of $%.4f/unit (variance: +$%.4f/unit beyond tolerance).", pair.InvoicedPrice, pair.POPrice, deltaPrice),
			ItemSKU:        pair.ItemSKU,
			ExpectedValue:  pair.POPrice,
			ActualValue:    pair.InvoicedPrice,
			VarianceAmount: totalLineOvercharge,
			Currency:       "USD",
			TargetDocument: config.DocTypeInvoice,
			PageNumber:     box.Page,
			BoundingBox:    box,
		}
	}

	return nil
}
