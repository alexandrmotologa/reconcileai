package auditor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"time"

	"github.com/alexandrmotologa/reconcileai/internal/config"
	"github.com/alexandrmotologa/reconcileai/internal/matching"
)

// ReconcileDossier executes full tripartite 3-way matching and discrepancy auditing.
func ReconcileDossier(dossier *config.Dossier, cfg config.AppConfig) *config.AuditSummary {
	startTime := time.Now()

	// 1. Semantic alignment and bipartite matching
	pairs := matching.MatchDossierLines(dossier, cfg.Matching.MinDescriptionSimilarity)

	// Map invoice lines for fast coordinate lookup
	invLineMap := make(map[string]*config.LineItem)
	for i := range dossier.Invoice.Lines {
		l := &dossier.Invoice.Lines[i]
		invLineMap[l.ID] = l
	}

	var discrepancies []config.Discrepancy

	// 2. Audit line item pairs
	var authorizedLineSubtotal float64

	for i := range pairs {
		pair := &pairs[i]
		invLine := invLineMap[pair.InvoiceLineID]

		// Check for unapproved / unmatched line
		if pair.POLineID == "" {
			pair.HasDiscrepancy = true
			disc := config.Discrepancy{
				ID:             fmt.Sprintf("DISC-UNMATCHED-%s", pair.InvoiceLineID),
				Type:           config.DiscrepancyUnmatchedItem,
				Severity:       config.SeverityCritical,
				Title:          fmt.Sprintf("Unmatched Invoiced SKU %s", pair.ItemSKU),
				Description:    fmt.Sprintf("Item '%s' (SKU: %s) does not match any authorized line item on Purchase Order %s.", pair.Description, pair.ItemSKU, dossier.PurchaseOrder.Header.DocumentID),
				ItemSKU:        pair.ItemSKU,
				ExpectedValue:  0.0,
				ActualValue:    pair.InvoicedQty * pair.InvoicedPrice,
				VarianceAmount: pair.InvoicedQty * pair.InvoicedPrice,
				Currency:       dossier.Invoice.Header.Currency,
				TargetDocument: config.DocTypeInvoice,
			}
			if invLine != nil {
				disc.PageNumber = invLine.BoundingBox.Page
				disc.BoundingBox = invLine.BoundingBox
			}
			discrepancies = append(discrepancies, disc)
			continue
		}

		// Quantity check
		if qtyDisc := AuditQuantity(pair, invLine, cfg.Tolerance.MaxQuantityVariance); qtyDisc != nil {
			pair.HasDiscrepancy = true
			discrepancies = append(discrepancies, *qtyDisc)
		}

		// Price variance check
		if priceDisc := AuditPrice(pair, invLine, cfg.Tolerance); priceDisc != nil {
			pair.HasDiscrepancy = true
			discrepancies = append(discrepancies, *priceDisc)
		}

		// Compute authorized line amount: payable quantity is min(invoiced, received) at authorized PO price
		payableQty := math.Min(pair.InvoicedQty, pair.ReceivedQty)
		authorizedLineSubtotal += payableQty * pair.POPrice
	}

	// 3. Audit arithmetic integrity
	arithDiscs := AuditFaceArithmetic(&dossier.Invoice)
	for _, d := range arithDiscs {
		discrepancies = append(discrepancies, *d)
	}

	// 4. Audit unauthorized fees and taxes
	surcharges := AuditSurcharges(dossier)
	for _, d := range surcharges {
		discrepancies = append(discrepancies, *d)
	}

	if taxDisc := AuditTax(dossier, cfg.Tolerance.MaxTaxVarianceAbsolute); taxDisc != nil {
		discrepancies = append(discrepancies, *taxDisc)
	}

	// 5. Compute authorized totals and prevented overcharges
	totalInvoiced := dossier.Invoice.Header.TotalAmount
	if totalInvoiced <= 0 {
		totalInvoiced = dossier.Invoice.Header.Subtotal + dossier.Invoice.Header.TaxTotal + dossier.Invoice.Header.FreightTotal
	}

	// Calculate authorized tax rate based on authorized line subtotal
	var authorizedTax float64
	if dossier.Invoice.Header.Subtotal > 0 && dossier.Invoice.Header.TaxTotal > 0 {
		effectiveRate := dossier.Invoice.Header.TaxTotal / dossier.Invoice.Header.Subtotal
		authorizedTax = math.Round(authorizedLineSubtotal*effectiveRate*100) / 100
	}

	var authorizedFreight float64
	if len(surcharges) == 0 {
		authorizedFreight = dossier.Invoice.Header.FreightTotal
	}

	totalAuthorized := math.Round((authorizedLineSubtotal+authorizedTax+authorizedFreight)*100) / 100
	preventedOvercharge := math.Round((totalInvoiced-totalAuthorized)*100) / 100
	if preventedOvercharge < 0 {
		preventedOvercharge = 0.0
	}

	// 6. Formulate reconciliation verdict
	matchStatus := "APPROVED"
	for _, d := range discrepancies {
		if d.Severity == config.SeverityCritical {
			matchStatus = "FLAGGED_DISCREPANCY"
			break
		}
	}

	elapsed := time.Since(startTime).Milliseconds()

	summary := &config.AuditSummary{
		DossierID:          dossier.ID,
		VendorName:         dossier.Invoice.Header.VendorName,
		InvoiceNumber:      dossier.Invoice.Header.DocumentID,
		PONumber:           dossier.PurchaseOrder.Header.DocumentID,
		MatchStatus:        matchStatus,
		TotalInvoiced:      totalInvoiced,
		TotalAuthorized:    totalAuthorized,
		PreventedOvercharge: preventedOvercharge,
		Currency:           dossier.Invoice.Header.Currency,
		Discrepancies:      discrepancies,
		MatchedPairs:       pairs,
		AuditedAt:          time.Now().UTC().Format(time.RFC3339),
		ExecutionTimeMs:    elapsed,
	}

	summary.AuditHash = computeAuditHash(summary)
	return summary
}

func computeAuditHash(s *config.AuditSummary) string {
	h := sha256.New()
	h.Write([]byte(s.DossierID))
	h.Write([]byte(s.VendorName))
	h.Write([]byte(s.InvoiceNumber))
	h.Write([]byte(s.PONumber))
	h.Write([]byte(fmt.Sprintf("%.2f:%.2f:%.2f:%d", s.TotalInvoiced, s.TotalAuthorized, s.PreventedOvercharge, len(s.Discrepancies))))
	for _, d := range s.Discrepancies {
		h.Write([]byte(fmt.Sprintf("%s:%s:%.2f", d.ID, d.Type, d.VarianceAmount)))
	}
	return hex.EncodeToString(h.Sum(nil))
}
