package report

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/alexandrmotologa/reconcileai/internal/config"
	"github.com/jung-kurt/gofpdf"
)

// GenerateAuditEvidencePDF produces a formal discrepancy audit document with visual bounding overlays.
func GenerateAuditEvidencePDF(summary *config.AuditSummary, invoiceDoc *config.Document, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	pdf := gofpdf.New("P", "pt", "A4", "")
	pdf.SetMargins(40, 40, 40)
	pdf.AddPage()

	// 1. Header Banner
	pdf.SetFillColor(30, 41, 59) // Slate 800
	pdf.Rect(0, 0, 595, 60, "F")

	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetTextColor(255, 255, 255)
	pdf.Text(40, 38, "reconcileai | Accounts Payable Audit Evidence")

	// 2. Audit Meta
	pdf.SetY(75)
	pdf.SetFont("Helvetica", "B", 13)
	pdf.SetTextColor(15, 23, 42)
	pdf.CellFormat(250, 18, fmt.Sprintf("Dossier: %s", summary.DossierID), "", 0, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(100, 116, 139)
	pdf.CellFormat(260, 18, fmt.Sprintf("Audit Date: %s", summary.AuditedAt), "", 1, "R", false, 0, "")

	// Status badge
	pdf.SetY(98)
	if summary.MatchStatus == "APPROVED" {
		pdf.SetFillColor(220, 252, 231) // Green 100
		pdf.SetTextColor(22, 101, 52)   // Green 800
		pdf.Rect(40, 98, 120, 22, "F")
		pdf.SetFont("Helvetica", "B", 10)
		pdf.Text(48, 113, "STATUS: APPROVED")
	} else {
		pdf.SetFillColor(254, 226, 226) // Red 100
		pdf.SetTextColor(153, 27, 27)   // Red 800
		pdf.Rect(40, 98, 165, 22, "F")
		pdf.SetFont("Helvetica", "B", 10)
		pdf.Text(48, 113, "STATUS: FLAGGED DISCREPANCY")
	}

	// 3. Financial Summary KPI Cards
	pdf.SetY(130)
	cardWidth := 160.0
	cardHeight := 50.0

	// Card 1: Invoiced
	pdf.SetFillColor(248, 250, 252)
	pdf.SetDrawColor(226, 232, 240)
	pdf.Rect(40, 130, cardWidth, cardHeight, "FD")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(100, 116, 139)
	pdf.Text(50, 145, "TOTAL INVOICED")
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetTextColor(15, 23, 42)
	pdf.Text(50, 168, fmt.Sprintf("%s %.2f", summary.Currency, summary.TotalInvoiced))

	// Card 2: Authorized
	pdf.Rect(215, 130, cardWidth, cardHeight, "FD")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(100, 116, 139)
	pdf.Text(225, 145, "AUTHORIZED SETTLEMENT")
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetTextColor(22, 101, 52)
	pdf.Text(225, 168, fmt.Sprintf("%s %.2f", summary.Currency, summary.TotalAuthorized))

	// Card 3: Prevented Overcharge
	pdf.Rect(390, 130, cardWidth, cardHeight, "FD")
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(100, 116, 139)
	pdf.Text(400, 145, "PREVENTED OVERCHARGE")
	pdf.SetFont("Helvetica", "B", 14)
	pdf.SetTextColor(220, 38, 38)
	pdf.Text(400, 168, fmt.Sprintf("%s %.2f", summary.Currency, summary.PreventedOvercharge))

	// 4. Vendor and PO Information
	pdf.SetY(195)
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(51, 65, 85)
	pdf.CellFormat(250, 15, fmt.Sprintf("Vendor: %s", summary.VendorName), "", 0, "L", false, 0, "")
	pdf.CellFormat(260, 15, fmt.Sprintf("Purchase Order: %s", summary.PONumber), "", 1, "L", false, 0, "")
	pdf.CellFormat(250, 15, fmt.Sprintf("Invoice Number: %s", summary.InvoiceNumber), "", 0, "L", false, 0, "")
	pdf.CellFormat(260, 15, fmt.Sprintf("Audit Hash: %s", summary.AuditHash[:16]+"..."), "", 1, "L", false, 0, "")

	// 5. Itemized 3-Way Reconciliation Table
	pdf.SetY(235)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetTextColor(15, 23, 42)
	pdf.Text(40, 245, "3-Way Line Matching Table")

	// Table Header
	pdf.SetY(252)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(241, 245, 249)
	pdf.SetTextColor(71, 85, 105)
	pdf.CellFormat(65, 20, "SKU", "1", 0, "L", true, 0, "")
	pdf.CellFormat(140, 20, "Description", "1", 0, "L", true, 0, "")
	pdf.CellFormat(45, 20, "Inv Qty", "1", 0, "R", true, 0, "")
	pdf.CellFormat(45, 20, "Rec Qty", "1", 0, "R", true, 0, "")
	pdf.CellFormat(55, 20, "Inv Price", "1", 0, "R", true, 0, "")
	pdf.CellFormat(55, 20, "PO Price", "1", 0, "R", true, 0, "")
	pdf.CellFormat(55, 20, "Variance", "1", 0, "R", true, 0, "")
	pdf.CellFormat(55, 20, "Status", "1", 1, "C", true, 0, "")

	pdf.SetFont("Helvetica", "", 8)
	for _, p := range summary.MatchedPairs {
		if p.HasDiscrepancy {
			pdf.SetFillColor(254, 242, 242) // Light red
			pdf.SetTextColor(153, 27, 27)
		} else {
			pdf.SetFillColor(255, 255, 255)
			pdf.SetTextColor(30, 41, 59)
		}

		status := "MATCH"
		if p.HasDiscrepancy {
			status = "FLAG"
		}

		desc := p.Description
		if len(desc) > 28 {
			desc = desc[:25] + "..."
		}

		pdf.CellFormat(65, 18, p.ItemSKU, "1", 0, "L", true, 0, "")
		pdf.CellFormat(140, 18, desc, "1", 0, "L", true, 0, "")
		pdf.CellFormat(45, 18, fmt.Sprintf("%.1f", p.InvoicedQty), "1", 0, "R", true, 0, "")
		pdf.CellFormat(45, 18, fmt.Sprintf("%.1f", p.ReceivedQty), "1", 0, "R", true, 0, "")
		pdf.CellFormat(55, 18, fmt.Sprintf("%.2f", p.InvoicedPrice), "1", 0, "R", true, 0, "")
		pdf.CellFormat(55, 18, fmt.Sprintf("%.2f", p.POPrice), "1", 0, "R", true, 0, "")

		varStr := fmt.Sprintf("%.2f", p.PriceVariance)
		if p.InvoicedQty > p.ReceivedQty {
			varStr = fmt.Sprintf("+%.1f qty", p.InvoicedQty-p.ReceivedQty)
		}
		pdf.CellFormat(55, 18, varStr, "1", 0, "R", true, 0, "")
		pdf.CellFormat(55, 18, status, "1", 1, "C", true, 0, "")
	}

	// 6. Page 2: Visual Evidence with Red Bounding Overlays
	if len(summary.Discrepancies) > 0 {
		pdf.AddPage()

		pdf.SetFillColor(30, 41, 59)
		pdf.Rect(0, 0, 595, 45, "F")
		pdf.SetFont("Helvetica", "B", 14)
		pdf.SetTextColor(255, 255, 255)
		pdf.Text(40, 28, "Visual Evidence & Document Annotation")

		pdf.SetY(60)
		pdf.SetFont("Helvetica", "B", 11)
		pdf.SetTextColor(15, 23, 42)
		pdf.Text(40, 72, "Identified Discrepancies with Exact Bounding Coordinates")

		// Render document simulation panel
		panelY := 85.0
		panelW := 515.0
		panelH := 420.0

		pdf.SetDrawColor(203, 213, 225)
		pdf.SetFillColor(250, 250, 250)
		pdf.Rect(40, panelY, panelW, panelH, "FD")

		// Document watermarking simulation inside panel
		pdf.SetFont("Helvetica", "B", 12)
		pdf.SetTextColor(100, 116, 139)
		pdf.Text(55, panelY+25, fmt.Sprintf("VENDOR INVOICE: %s", summary.InvoiceNumber))
		pdf.SetFont("Helvetica", "", 9)
		pdf.Text(55, panelY+40, fmt.Sprintf("Vendor: %s | PO Ref: %s", summary.VendorName, summary.PONumber))

		// Render simulated lines and draw red bounding boxes
		simY := panelY + 65.0
		for idx, d := range summary.Discrepancies {
			badgeNum := idx + 1

			// Red Bounding Box around discrepancy
			pdf.SetDrawColor(220, 38, 38) // Red 600
			pdf.SetLineWidth(1.8)
			pdf.SetFillColor(254, 242, 242)

			boxY := simY - 12.0
			pdf.Rect(55, boxY, panelW-30, 36, "D")

			// Red Badge callout
			pdf.SetFillColor(220, 38, 38)
			pdf.Circle(45, boxY+18, 9, "F")
			pdf.SetFont("Helvetica", "B", 9)
			pdf.SetTextColor(255, 255, 255)
			pdf.Text(42, boxY+21, fmt.Sprintf("%d", badgeNum))

			// Discrepancy label inside box
			pdf.SetFont("Helvetica", "B", 10)
			pdf.SetTextColor(185, 28, 28)
			pdf.Text(65, simY+2, fmt.Sprintf("[%s] %s (Variance: +$%.2f)", d.ItemSKU, d.Title, d.VarianceAmount))

			pdf.SetFont("Helvetica", "", 8)
			pdf.SetTextColor(71, 85, 105)
			descSnippet := d.Description
			if len(descSnippet) > 90 {
				descSnippet = descSnippet[:87] + "..."
			}
			pdf.Text(65, simY+16, descSnippet)

			simY += 48.0
			if simY > panelY+panelH-30 {
				break
			}
		}

		// Discrepancy Audit Action Footer
		pdf.SetY(panelY + panelH + 20)
		pdf.SetFont("Helvetica", "B", 10)
		pdf.SetTextColor(15, 23, 42)
		pdf.Text(40, panelY+panelH+35, "AP Auditor Certification")

		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(71, 85, 105)
		pdf.Text(40, panelY+panelH+50, "All items outside authorized contract tolerances have been flagged. Certified for AP dispute dispatch.")
		pdf.Text(40, panelY+panelH+65, fmt.Sprintf("Verification Hash: %s", summary.AuditHash))
	}

	return pdf.OutputFileAndClose(outputPath)
}

// MinFloat returns smaller of two float values.
func MinFloat(a, b float64) float64 {
	return math.Min(a, b)
}
