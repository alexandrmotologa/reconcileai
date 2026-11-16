package matching

import (
	"fmt"
	"math"
	"sort"

	"github.com/alexandrmotologa/reconcileai/internal/config"
	"github.com/alexandrmotologa/reconcileai/internal/ingestion"
)

// MatchDossierLines pairs invoice lines with corresponding PO and GRN lines.
func MatchDossierLines(dossier *config.Dossier, minSimilarity float64) []config.MatchedLinePair {
	poLines := dossier.PurchaseOrder.Lines
	invLines := dossier.Invoice.Lines

	type candidateMatch struct {
		invIdx     int
		poIdx      int
		score      float64
		reason     string
	}

	var candidates []candidateMatch

	for i, invItem := range invLines {
		for p, poItem := range poLines {
			score, reason := ComputeItemSimilarity(poItem.SKU, poItem.Description, invItem.SKU, invItem.Description)
			if score >= minSimilarity {
				candidates = append(candidates, candidateMatch{
					invIdx: i,
					poIdx:  p,
					score:  score,
					reason: reason,
				})
			}
		}
	}

	// Sort candidates by score descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	matchedPO := make(map[int]bool)
	matchedINV := make(map[int]bool)
	var pairs []config.MatchedLinePair

	for _, c := range candidates {
		if matchedINV[c.invIdx] || matchedPO[c.poIdx] {
			continue
		}

		matchedINV[c.invIdx] = true
		matchedPO[c.poIdx] = true

		invItem := invLines[c.invIdx]
		poItem := poLines[c.poIdx]

		// Aggregate received quantities from GRNs
		receivedQty, grnLineIDs := aggregateGRNReceipts(poItem, invItem, dossier.GoodsReceipts)

		// Normalize invoiced quantity and unit price if UOM differs from PO
		invoicedQtyNorm := invItem.Quantity
		invoicedPriceNorm := invItem.UnitPrice
		if invItem.UnitOfMeasure != "" && poItem.UnitOfMeasure != "" && invItem.UnitOfMeasure != poItem.UnitOfMeasure {
			if converted, ok := ingestion.ConvertQuantity(invItem.Quantity, invItem.UnitOfMeasure, poItem.UnitOfMeasure); ok && invItem.Quantity > 0 {
				factor := converted / invItem.Quantity
				invoicedQtyNorm = converted
				if factor > 0 {
					invoicedPriceNorm = math.Round((invItem.UnitPrice/factor)*10000) / 10000
				}
			}
		}

		priceVariance := math.Round((invoicedPriceNorm-poItem.UnitPrice)*1000) / 1000

		pairs = append(pairs, config.MatchedLinePair{
			InvoiceLineID: invItem.ID,
			POLineID:      poItem.ID,
			GRNLineIDs:    grnLineIDs,
			ItemSKU:       poItem.SKU,
			Description:   poItem.Description,
			Confidence:    math.Round(c.score*100) / 100,
			MatchReason:   c.reason,
			InvoicedQty:   invoicedQtyNorm,
			ReceivedQty:   receivedQty,
			OrderedQty:    poItem.Quantity,
			InvoicedPrice: invoicedPriceNorm,
			POPrice:       poItem.UnitPrice,
			PriceVariance: priceVariance,
		})
	}

	// Handle unmatched invoice lines (potential unapproved items or rogue billings)
	for i, invItem := range invLines {
		if matchedINV[i] {
			continue
		}

		pairs = append(pairs, config.MatchedLinePair{
			InvoiceLineID:  invItem.ID,
			POLineID:       "",
			GRNLineIDs:     nil,
			ItemSKU:        invItem.SKU,
			Description:    invItem.Description,
			Confidence:     0.0,
			MatchReason:    "Unmatched invoice item not present on Purchase Order",
			InvoicedQty:    invItem.Quantity,
			ReceivedQty:    0.0,
			OrderedQty:     0.0,
			InvoicedPrice:  invItem.UnitPrice,
			POPrice:        0.0,
			PriceVariance:  invItem.UnitPrice,
			HasDiscrepancy: true,
		})
	}

	return pairs
}

// aggregateGRNReceipts sums actual received quantities across all GRN documents for a line item.
func aggregateGRNReceipts(poItem, invItem config.LineItem, grns []config.Document) (float64, []string) {
	var totalReceived float64
	var grnLineIDs []string

	for _, grn := range grns {
		for _, grnItem := range grn.Lines {
			match := false

			// Match by SKU
			if grnItem.SKU != "" && (grnItem.SKU == poItem.SKU || grnItem.SKU == invItem.SKU) {
				match = true
			} else {
				// Match by description similarity
				score, _ := ComputeItemSimilarity(poItem.SKU, poItem.Description, grnItem.SKU, grnItem.Description)
				if score >= 0.75 {
					match = true
				}
			}

			if match {
				qty := grnItem.Quantity
				// Normalize UOM to PO unit if needed
				if grnItem.UnitOfMeasure != "" && poItem.UnitOfMeasure != "" && grnItem.UnitOfMeasure != poItem.UnitOfMeasure {
					if converted, ok := ingestion.ConvertQuantity(qty, grnItem.UnitOfMeasure, poItem.UnitOfMeasure); ok {
						qty = converted
					}
				}
				totalReceived += qty
				grnLineIDs = append(grnLineIDs, fmt.Sprintf("%s:%s", grn.Header.DocumentID, grnItem.ID))
			}
		}
	}

	return totalReceived, grnLineIDs
}
