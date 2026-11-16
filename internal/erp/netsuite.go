package erp

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"math"
	"time"

	"github.com/alexandrmotologa/reconcileai/internal/config"
)

// GenerateNetSuiteCSV formats an authorized settlement as a NetSuite Bill import CSV.
func GenerateNetSuiteCSV(summary *config.AuditSummary) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	// CSV Header
	header := []string{
		"ExternalID",
		"Vendor",
		"TransactionDate",
		"DueDate",
		"InvoiceNumber",
		"PurchaseOrderRef",
		"Currency",
		"Terms",
		"Memo",
		"LineNumber",
		"ItemSKU",
		"Description",
		"Quantity",
		"Rate",
		"Amount",
		"AuditHash",
	}
	if err := writer.Write(header); err != nil {
		return nil, err
	}

	externalID := fmt.Sprintf("BILL-%s", summary.InvoiceNumber)
	today := time.Now().Format("2006-01-02")
	dueDate := time.Now().AddDate(0, 1, 0).Format("2006-01-02")
	memo := fmt.Sprintf("reconcileai 3-way verified settlement (PO: %s)", summary.PONumber)

	lineIdx := 1
	for _, pair := range summary.MatchedPairs {
		payableQty := math.Min(pair.InvoicedQty, pair.ReceivedQty)
		if payableQty <= 0 {
			continue
		}

		lineAmount := math.Round(payableQty*pair.POPrice*100) / 100

		row := []string{
			externalID,
			summary.VendorName,
			today,
			dueDate,
			summary.InvoiceNumber,
			summary.PONumber,
			summary.Currency,
			"Net 30",
			memo,
			fmt.Sprintf("%d", lineIdx),
			pair.ItemSKU,
			pair.Description,
			fmt.Sprintf("%.2f", payableQty),
			fmt.Sprintf("%.4f", pair.POPrice),
			fmt.Sprintf("%.2f", lineAmount),
			summary.AuditHash[:16],
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
		lineIdx++
	}

	writer.Flush()
	return buf.Bytes(), writer.Error()
}
