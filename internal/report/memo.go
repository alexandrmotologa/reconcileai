package report

import (
	"fmt"
	"strings"
	"time"

	"github.com/alexandrmotologa/reconcileai/internal/config"
)

// GenerateVendorDisputeMemo formats an official, humanized dispute notice for vendor accounts receivable.
func GenerateVendorDisputeMemo(summary *config.AuditSummary, companyName string) string {
	if companyName == "" {
		companyName = "Enterprise Procurement & Accounts Payable Operations"
	}

	var sb strings.Builder

	sb.WriteString("# Vendor Invoice Discrepancy Notice\n\n")
	sb.WriteString(fmt.Sprintf("**Date:** %s\n", time.Now().Format("2006-01-02")))
	sb.WriteString(fmt.Sprintf("**To:** %s (Accounts Receivable)\n", summary.VendorName))
	sb.WriteString(fmt.Sprintf("**From:** %s\n", companyName))
	sb.WriteString(fmt.Sprintf("**Dispute Reference:** DISP-%s-%s\n\n", summary.DossierID, time.Now().Format("060102")))

	sb.WriteString("## Transaction Reference\n")
	sb.WriteString(fmt.Sprintf("- **Vendor Invoice Number:** %s\n", summary.InvoiceNumber))
	sb.WriteString(fmt.Sprintf("- **Authorized Purchase Order:** %s\n", summary.PONumber))
	sb.WriteString(fmt.Sprintf("- **Audit Cryptographic Hash:** `%s`\n\n", summary.AuditHash))

	sb.WriteString("## Financial Summary\n")
	sb.WriteString("| Field | Amount |\n")
	sb.WriteString("| :--- | :--- |\n")
	sb.WriteString(fmt.Sprintf("| Invoiced Amount | %s %.2f |\n", summary.Currency, summary.TotalInvoiced))
	sb.WriteString(fmt.Sprintf("| Authorized Settlement Amount | %s %.2f |\n", summary.Currency, summary.TotalAuthorized))
	sb.WriteString(fmt.Sprintf("| **Amount in Dispute** | **%s %.2f** |\n\n", summary.Currency, summary.PreventedOvercharge))

	sb.WriteString("## Itemized Variances\n\n")
	sb.WriteString("| Item / SKU | Discrepancy Type | Invoiced | Authorized | Variance |\n")
	sb.WriteString("| :--- | :--- | :--- | :--- | :--- |\n")

	for _, d := range summary.Discrepancies {
		sb.WriteString(fmt.Sprintf("| %s | %s | %.2f | %.2f | +%.2f %s |\n",
			d.ItemSKU,
			formatDiscrepancyType(d.Type),
			d.ActualValue,
			d.ExpectedValue,
			d.VarianceAmount,
			summary.Currency,
		))
	}

	sb.WriteString("\n## Variance Details and Explanations\n\n")
	for i, d := range summary.Discrepancies {
		sb.WriteString(fmt.Sprintf("### %d. %s\n", i+1, d.Title))
		sb.WriteString(fmt.Sprintf("%s\n\n", d.Description))
		if d.BoundingBox.Width > 0 {
			sb.WriteString(fmt.Sprintf("- **Document Evidence:** Page %d, coordinates (X: %.1f, Y: %.1f, W: %.1f, H: %.1f)\n\n",
				d.PageNumber, d.BoundingBox.X, d.BoundingBox.Y, d.BoundingBox.Width, d.BoundingBox.Height))
		}
	}

	sb.WriteString("## Requested Remediation\n\n")
	sb.WriteString("To ensure timely payment processing, please provide one of the following within 5 business days:\n")
	sb.WriteString(fmt.Sprintf("1. A revised invoice in the authorized amount of **%s %.2f** citing PO %s.\n", summary.Currency, summary.TotalAuthorized, summary.PONumber))
	sb.WriteString(fmt.Sprintf("2. A credit note in the amount of **%s %.2f** referencing Invoice %s.\n\n", summary.Currency, summary.PreventedOvercharge, summary.InvoiceNumber))
	sb.WriteString("Payments for undisputed items will proceed according to agreed payment terms.\n")

	return sb.String()
}

func formatDiscrepancyType(t config.DiscrepancyType) string {
	switch t {
	case config.DiscrepancyQtyOverbilled:
		return "Quantity Overbilled"
	case config.DiscrepancyPriceVariance:
		return "Price Rate Variance"
	case config.DiscrepancyUnauthorizedFee:
		return "Unauthorized Surcharge"
	case config.DiscrepancyArithmeticMismatch:
		return "Arithmetic Mismatch"
	case config.DiscrepancyUnmatchedItem:
		return "Unmatched Item"
	default:
		return string(t)
	}
}
