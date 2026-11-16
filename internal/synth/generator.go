package synth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexandrmotologa/reconcileai/internal/config"
	"github.com/jung-kurt/gofpdf"
)

// GenerateAllReferenceDossiers writes all 5 reference test scenarios to the target directory.
func GenerateAllReferenceDossiers(targetBaseDir string) ([]string, error) {
	dossiers := []struct {
		id      string
		builder func(string) (*config.Dossier, error)
	}{
		{"dossier_01_clean_match", BuildCleanMatchDossier},
		{"dossier_02_qty_overbilling", BuildQtyOverbillingDossier},
		{"dossier_03_price_variance", BuildPriceVarianceDossier},
		{"dossier_04_unauthorized_fee", BuildUnauthorizedFeeDossier},
		{"dossier_05_uom_split_delivery", BuildSplitDeliveryUOMDossier},
	}

	var createdDirs []string
	for _, d := range dossiers {
		dir := filepath.Join(targetBaseDir, d.id)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}

		dossier, err := d.builder(dir)
		if err != nil {
			return nil, fmt.Errorf("build %s: %w", d.id, err)
		}

		// Save manifest json
		manifestPath := filepath.Join(dir, "dossier.json")
		manifestData, err := json.MarshalIndent(dossier, "", "  ")
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
			return nil, err
		}

		createdDirs = append(createdDirs, dir)
	}

	return createdDirs, nil
}

// BuildCleanMatchDossier builds a transaction where all 3 documents match 100%.
func BuildCleanMatchDossier(dir string) (*config.Dossier, error) {
	po := config.Document{
		Header: config.DocumentHeader{
			DocumentID:   "PO-2026-9011",
			DocumentType: config.DocTypePO,
			VendorID:     "VEND-APEX",
			VendorName:   "Apex Industrial Fasteners Corp",
			VendorTaxID:  "US-82-9912041",
			IssueDate:    "2026-09-01",
			Currency:     "USD",
			Subtotal:     4850.00,
			TaxTotal:     388.00,
			FreightTotal: 0.00,
			TotalAmount:  5238.00,
			PaymentTerms: "2/10 Net 30 FOB Destination",
		},
		Lines: []config.LineItem{
			{ID: "PO-L01", LineNumber: 1, SKU: "FAST-HEX-01", Description: "Grade 8 Hex Flange Bolt 3/8-16", Quantity: 500, UnitOfMeasure: "EA", UnitPrice: 4.20, TotalPrice: 2100.00},
			{ID: "PO-L02", LineNumber: 2, SKU: "FAST-NUT-02", Description: "Hardened Grade 8 Lock Nut 3/8-16", Quantity: 500, UnitOfMeasure: "EA", UnitPrice: 2.10, TotalPrice: 1050.00},
			{ID: "PO-L03", LineNumber: 3, SKU: "WASH-FLAT-03", Description: "Zinc Plated Flat Washer 3/8 Inch", Quantity: 1000, UnitOfMeasure: "EA", UnitPrice: 1.70, TotalPrice: 1700.00},
		},
	}

	grn := config.Document{
		Header: config.DocumentHeader{
			DocumentID:   "GRN-2026-4401",
			DocumentType: config.DocTypeGRN,
			ReferenceID:  "PO-2026-9011",
			VendorID:     "VEND-APEX",
			VendorName:   "Apex Industrial Fasteners Corp",
			IssueDate:    "2026-09-05",
			Currency:     "USD",
		},
		Lines: []config.LineItem{
			{ID: "GRN-L01", LineNumber: 1, SKU: "FAST-HEX-01", Description: "Grade 8 Hex Flange Bolt 3/8-16", Quantity: 500, UnitOfMeasure: "EA"},
			{ID: "GRN-L02", LineNumber: 2, SKU: "FAST-NUT-02", Description: "Hardened Grade 8 Lock Nut 3/8-16", Quantity: 500, UnitOfMeasure: "EA"},
			{ID: "GRN-L03", LineNumber: 3, SKU: "WASH-FLAT-03", Description: "Zinc Plated Flat Washer 3/8 Inch", Quantity: 1000, UnitOfMeasure: "EA"},
		},
	}

	inv := config.Document{
		Header: config.DocumentHeader{
			DocumentID:   "INV-APEX-8812",
			DocumentType: config.DocTypeInvoice,
			ReferenceID:  "PO-2026-9011",
			VendorID:     "VEND-APEX",
			VendorName:   "Apex Industrial Fasteners Corp",
			VendorTaxID:  "US-82-9912041",
			IssueDate:    "2026-09-08",
			Currency:     "USD",
			Subtotal:     4850.00,
			TaxTotal:     388.00,
			FreightTotal: 0.00,
			TotalAmount:  5238.00,
			PaymentTerms: "2/10 Net 30",
		},
		Lines: []config.LineItem{
			{ID: "INV-L01", LineNumber: 1, SKU: "FAST-HEX-01", Description: "Hex Flange Bolt 3/8-16 Grade 8", Quantity: 500, UnitOfMeasure: "EA", UnitPrice: 4.20, TotalPrice: 2100.00},
			{ID: "INV-L02", LineNumber: 2, SKU: "FAST-NUT-02", Description: "Lock Nut 3/8-16 Grade 8", Quantity: 500, UnitOfMeasure: "EA", UnitPrice: 2.10, TotalPrice: 1050.00},
			{ID: "INV-L03", LineNumber: 3, SKU: "WASH-FLAT-03", Description: "Flat Washer 3/8 Inch Zinc", Quantity: 1000, UnitOfMeasure: "EA", UnitPrice: 1.70, TotalPrice: 1700.00},
		},
	}

	poPath := filepath.Join(dir, "PO_2026_9011.pdf")
	grnPath := filepath.Join(dir, "GRN_2026_4401.pdf")
	invPath := filepath.Join(dir, "INV_APEX_8812.pdf")

	if err := renderDocumentPDF(&po, poPath); err != nil {
		return nil, err
	}
	if err := renderDocumentPDF(&grn, grnPath); err != nil {
		return nil, err
	}
	if err := renderDocumentPDF(&inv, invPath); err != nil {
		return nil, err
	}

	po.RawPDFPath = poPath
	grn.RawPDFPath = grnPath
	inv.RawPDFPath = invPath

	return &config.Dossier{
		ID:            "dossier_01_clean_match",
		Title:         "Apex Industrial - Perfect 3-Way Match",
		PurchaseOrder: po,
		GoodsReceipts: []config.Document{grn},
		Invoice:       inv,
	}, nil
}

// BuildQtyOverbillingDossier builds a transaction where the vendor billed 100 units, but warehouse only received 80.
func BuildQtyOverbillingDossier(dir string) (*config.Dossier, error) {
	po := config.Document{
		Header: config.DocumentHeader{
			DocumentID:   "PO-2026-9022",
			DocumentType: config.DocTypePO,
			VendorID:     "VEND-LOGIX",
			VendorName:   "Logix Hydraulic Systems LLC",
			IssueDate:    "2026-09-02",
			Currency:     "USD",
			Subtotal:     12500.00,
			TaxTotal:     1000.00,
			FreightTotal: 0.00,
			TotalAmount:  13500.00,
			PaymentTerms: "Net 30 FOB Destination",
		},
		Lines: []config.LineItem{
			{ID: "PO-L01", LineNumber: 1, SKU: "HYD-VALVE-50", Description: "High Pressure Hydraulic Solenoid Valve 24V", Quantity: 100, UnitOfMeasure: "EA", UnitPrice: 125.00, TotalPrice: 12500.00},
		},
	}

	grn := config.Document{
		Header: config.DocumentHeader{
			DocumentID:   "GRN-2026-4420",
			DocumentType: config.DocTypeGRN,
			ReferenceID:  "PO-2026-9022",
			VendorID:     "VEND-LOGIX",
			VendorName:   "Logix Hydraulic Systems LLC",
			IssueDate:    "2026-09-06",
			Currency:     "USD",
		},
		Lines: []config.LineItem{
			// Warehouse intake verified short delivery of 80 units
			{ID: "GRN-L01", LineNumber: 1, SKU: "HYD-VALVE-50", Description: "Hydraulic Solenoid Valve 24V (Short Delivery)", Quantity: 80, UnitOfMeasure: "EA"},
		},
	}

	inv := config.Document{
		Header: config.DocumentHeader{
			DocumentID:   "INV-LOGIX-9921",
			DocumentType: config.DocTypeInvoice,
			ReferenceID:  "PO-2026-9022",
			VendorID:     "VEND-LOGIX",
			VendorName:   "Logix Hydraulic Systems LLC",
			IssueDate:    "2026-09-08",
			Currency:     "USD",
			Subtotal:     12500.00,
			TaxTotal:     1000.00,
			FreightTotal: 0.00,
			TotalAmount:  13500.00, // Billed full 100 units instead of 80
			PaymentTerms: "Net 30",
		},
		Lines: []config.LineItem{
			{ID: "INV-L01", LineNumber: 1, SKU: "HYD-VALVE-50", Description: "Solenoid Valve 24V High Pressure", Quantity: 100, UnitOfMeasure: "EA", UnitPrice: 125.00, TotalPrice: 12500.00},
		},
	}

	poPath := filepath.Join(dir, "PO_2026_9022.pdf")
	grnPath := filepath.Join(dir, "GRN_2026_4420.pdf")
	invPath := filepath.Join(dir, "INV_LOGIX_9921.pdf")

	renderDocumentPDF(&po, poPath)
	renderDocumentPDF(&grn, grnPath)
	renderDocumentPDF(&inv, invPath)

	po.RawPDFPath = poPath
	grn.RawPDFPath = grnPath
	inv.RawPDFPath = invPath

	return &config.Dossier{
		ID:            "dossier_02_qty_overbilling",
		Title:         "Logix Hydraulic - Short Delivery Billed as Full ($2,500 Overcharge)",
		PurchaseOrder: po,
		GoodsReceipts: []config.Document{grn},
		Invoice:       inv,
	}, nil
}

// BuildPriceVarianceDossier builds a transaction where vendor raised unit prices by $2.50 per unit.
func BuildPriceVarianceDossier(dir string) (*config.Dossier, error) {
	po := config.Document{
		Header: config.DocumentHeader{
			DocumentID:   "PO-2026-9033",
			DocumentType: config.DocTypePO,
			VendorID:     "VEND-POLY",
			VendorName:   "Polymer Tech Materials Inc",
			IssueDate:    "2026-09-03",
			Currency:     "USD",
			Subtotal:     6000.00,
			TaxTotal:     480.00,
			FreightTotal: 0.00,
			TotalAmount:  6480.00,
			PaymentTerms: "Net 30",
		},
		Lines: []config.LineItem{
			{ID: "PO-L01", LineNumber: 1, SKU: "POLY-RESIN-01", Description: "Industrial Thermoplastic Resin Pellet A-Grade", Quantity: 500, UnitOfMeasure: "KG", UnitPrice: 12.00, TotalPrice: 6000.00},
		},
	}

	grn := config.Document{
		Header: config.DocumentHeader{
			DocumentID:   "GRN-2026-4433",
			DocumentType: config.DocTypeGRN,
			ReferenceID:  "PO-2026-9033",
			VendorID:     "VEND-POLY",
			VendorName:   "Polymer Tech Materials Inc",
			IssueDate:    "2026-09-07",
			Currency:     "USD",
		},
		Lines: []config.LineItem{
			{ID: "GRN-L01", LineNumber: 1, SKU: "POLY-RESIN-01", Description: "Thermoplastic Resin Pellet A-Grade", Quantity: 500, UnitOfMeasure: "KG"},
		},
	}

	inv := config.Document{
		Header: config.DocumentHeader{
			DocumentID:   "INV-POLY-1044",
			DocumentType: config.DocTypeInvoice,
			ReferenceID:  "PO-2026-9033",
			VendorID:     "VEND-POLY",
			VendorName:   "Polymer Tech Materials Inc",
			IssueDate:    "2026-09-09",
			Currency:     "USD",
			Subtotal:     7250.00, // Raised unit price to $14.50
			TaxTotal:     580.00,
			FreightTotal: 0.00,
			TotalAmount:  7830.00,
			PaymentTerms: "Net 30",
		},
		Lines: []config.LineItem{
			{ID: "INV-L01", LineNumber: 1, SKU: "POLY-RESIN-01", Description: "Resin Pellets Thermoplastic Grade-A", Quantity: 500, UnitOfMeasure: "KG", UnitPrice: 14.50, TotalPrice: 7250.00},
		},
	}

	poPath := filepath.Join(dir, "PO_2026_9033.pdf")
	grnPath := filepath.Join(dir, "GRN_2026_4433.pdf")
	invPath := filepath.Join(dir, "INV_POLY_1044.pdf")

	renderDocumentPDF(&po, poPath)
	renderDocumentPDF(&grn, grnPath)
	renderDocumentPDF(&inv, invPath)

	po.RawPDFPath = poPath
	grn.RawPDFPath = grnPath
	inv.RawPDFPath = invPath

	return &config.Dossier{
		ID:            "dossier_03_price_variance",
		Title:         "Polymer Tech - Unit Price Creep ($1,250 Overcharge)",
		PurchaseOrder: po,
		GoodsReceipts: []config.Document{grn},
		Invoice:       inv,
	}, nil
}

// BuildUnauthorizedFeeDossier builds a transaction with an unauthorized freight fee.
func BuildUnauthorizedFeeDossier(dir string) (*config.Dossier, error) {
	po := config.Document{
		Header: config.DocumentHeader{
			DocumentID:   "PO-2026-9044",
			DocumentType: config.DocTypePO,
			VendorID:     "VEND-DELTA",
			VendorName:   "Delta Precision Instruments Corp",
			IssueDate:    "2026-09-04",
			Currency:     "USD",
			Subtotal:     8400.00,
			TaxTotal:     672.00,
			FreightTotal: 0.00,
			TotalAmount:  9072.00,
			PaymentTerms: "Net 30 FOB Destination Freight Prepaid",
		},
		Lines: []config.LineItem{
			{ID: "PO-L01", LineNumber: 1, SKU: "SENS-OPT-08", Description: "Laser Optical Displacement Sensor 0-50mm", Quantity: 20, UnitOfMeasure: "EA", UnitPrice: 420.00, TotalPrice: 8400.00},
		},
	}

	grn := config.Document{
		Header: config.DocumentHeader{
			DocumentID:   "GRN-2026-4444",
			DocumentType: config.DocTypeGRN,
			ReferenceID:  "PO-2026-9044",
			VendorID:     "VEND-DELTA",
			VendorName:   "Delta Precision Instruments Corp",
			IssueDate:    "2026-09-08",
			Currency:     "USD",
		},
		Lines: []config.LineItem{
			{ID: "GRN-L01", LineNumber: 1, SKU: "SENS-OPT-08", Description: "Optical Displacement Sensor 0-50mm", Quantity: 20, UnitOfMeasure: "EA"},
		},
	}

	inv := config.Document{
		Header: config.DocumentHeader{
			DocumentID:   "INV-DELTA-7731",
			DocumentType: config.DocTypeInvoice,
			ReferenceID:  "PO-2026-9044",
			VendorID:     "VEND-DELTA",
			VendorName:   "Delta Precision Instruments Corp",
			IssueDate:    "2026-09-10",
			Currency:     "USD",
			Subtotal:     8400.00,
			TaxTotal:     672.00,
			FreightTotal: 450.00, // Unauthorized freight charge!
			TotalAmount:  9522.00,
			PaymentTerms: "Net 30",
		},
		Lines: []config.LineItem{
			{ID: "INV-L01", LineNumber: 1, SKU: "SENS-OPT-08", Description: "Optical Displacement Sensor 0-50mm", Quantity: 20, UnitOfMeasure: "EA", UnitPrice: 420.00, TotalPrice: 8400.00},
		},
	}

	poPath := filepath.Join(dir, "PO_2026_9044.pdf")
	grnPath := filepath.Join(dir, "GRN_2026_4444.pdf")
	invPath := filepath.Join(dir, "INV_DELTA_7731.pdf")

	renderDocumentPDF(&po, poPath)
	renderDocumentPDF(&grn, grnPath)
	renderDocumentPDF(&inv, invPath)

	po.RawPDFPath = poPath
	grn.RawPDFPath = grnPath
	inv.RawPDFPath = invPath

	return &config.Dossier{
		ID:            "dossier_04_unauthorized_fee",
		Title:         "Delta Precision - Unauthorized Freight Surcharge ($450 Overcharge)",
		PurchaseOrder: po,
		GoodsReceipts: []config.Document{grn},
		Invoice:       inv,
	}, nil
}

// BuildSplitDeliveryUOMDossier builds a multi-GRN partial delivery with UOM conversion (Dozen to Each).
func BuildSplitDeliveryUOMDossier(dir string) (*config.Dossier, error) {
	po := config.Document{
		Header: config.DocumentHeader{
			DocumentID:   "PO-2026-9055",
			DocumentType: config.DocTypePO,
			VendorID:     "VEND-CHEM",
			VendorName:   "ChemPure Analytical Reagents LLC",
			IssueDate:    "2026-09-05",
			Currency:     "USD",
			Subtotal:     3600.00,
			TaxTotal:     288.00,
			FreightTotal: 0.00,
			TotalAmount:  3888.00,
			PaymentTerms: "Net 30",
		},
		Lines: []config.LineItem{
			{ID: "PO-L01", LineNumber: 1, SKU: "REAG-ACET-01", Description: "Analytical HPLC Grade Acetonitrile 4L", Quantity: 120, UnitOfMeasure: "EA", UnitPrice: 30.00, TotalPrice: 3600.00},
		},
	}

	// First partial delivery: 60 units
	grn1 := config.Document{
		Header: config.DocumentHeader{
			DocumentID:   "GRN-2026-4451",
			DocumentType: config.DocTypeGRN,
			ReferenceID:  "PO-2026-9055",
			VendorID:     "VEND-CHEM",
			VendorName:   "ChemPure Analytical Reagents LLC",
			IssueDate:    "2026-09-07",
			Currency:     "USD",
		},
		Lines: []config.LineItem{
			{ID: "GRN1-L01", LineNumber: 1, SKU: "REAG-ACET-01", Description: "Acetonitrile HPLC Grade 4L (Batch 1)", Quantity: 60, UnitOfMeasure: "EA"},
		},
	}

	// Second partial delivery: 60 units
	grn2 := config.Document{
		Header: config.DocumentHeader{
			DocumentID:   "GRN-2026-4452",
			DocumentType: config.DocTypeGRN,
			ReferenceID:  "PO-2026-9055",
			VendorID:     "VEND-CHEM",
			VendorName:   "ChemPure Analytical Reagents LLC",
			IssueDate:    "2026-09-09",
			Currency:     "USD",
		},
		Lines: []config.LineItem{
			{ID: "GRN2-L01", LineNumber: 1, SKU: "REAG-ACET-01", Description: "Acetonitrile HPLC Grade 4L (Batch 2)", Quantity: 60, UnitOfMeasure: "EA"},
		},
	}

	// Invoice billed 10 Dozen (10 * 12 = 120 Each) at $360/Dozen
	inv := config.Document{
		Header: config.DocumentHeader{
			DocumentID:   "INV-CHEM-6091",
			DocumentType: config.DocTypeInvoice,
			ReferenceID:  "PO-2026-9055",
			VendorID:     "VEND-CHEM",
			VendorName:   "ChemPure Analytical Reagents LLC",
			IssueDate:    "2026-09-11",
			Currency:     "USD",
			Subtotal:     3600.00,
			TaxTotal:     288.00,
			FreightTotal: 0.00,
			TotalAmount:  3888.00,
			PaymentTerms: "Net 30",
		},
		Lines: []config.LineItem{
			{ID: "INV-L01", LineNumber: 1, SKU: "REAG-ACET-01", Description: "Acetonitrile HPLC Grade 4L Case Pack", Quantity: 10, UnitOfMeasure: "DZ", UnitPrice: 360.00, TotalPrice: 3600.00},
		},
	}

	poPath := filepath.Join(dir, "PO_2026_9055.pdf")
	grn1Path := filepath.Join(dir, "GRN_2026_4451.pdf")
	grn2Path := filepath.Join(dir, "GRN_2026_4452.pdf")
	invPath := filepath.Join(dir, "INV_CHEM_6091.pdf")

	renderDocumentPDF(&po, poPath)
	renderDocumentPDF(&grn1, grn1Path)
	renderDocumentPDF(&grn2, grn2Path)
	renderDocumentPDF(&inv, invPath)

	po.RawPDFPath = poPath
	grn1.RawPDFPath = grn1Path
	grn2.RawPDFPath = grn2Path
	inv.RawPDFPath = invPath

	return &config.Dossier{
		ID:            "dossier_05_uom_split_delivery",
		Title:         "ChemPure - Split GRN Delivery with Dozen-to-Each UOM Alignment",
		PurchaseOrder: po,
		GoodsReceipts: []config.Document{grn1, grn2},
		Invoice:       inv,
	}, nil
}

// renderDocumentPDF creates a realistic printed PDF document layout.
func renderDocumentPDF(doc *config.Document, filePath string) error {
	pdf := gofpdf.New("P", "pt", "A4", "")
	pdf.SetMargins(40, 40, 40)
	pdf.AddPage()

	// Header banner
	headerColor := []int{30, 41, 59} // Slate 800
	if doc.Header.DocumentType == config.DocTypeInvoice {
		headerColor = []int{15, 23, 42} // Darker for invoice
	} else if doc.Header.DocumentType == config.DocTypeGRN {
		headerColor = []int{14, 116, 144} // Cyan 700 for warehouse GRN
	}

	pdf.SetFillColor(headerColor[0], headerColor[1], headerColor[2])
	pdf.Rect(0, 0, 595, 55, "F")

	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetTextColor(255, 255, 255)
	title := fmt.Sprintf("%s - %s", doc.Header.DocumentType, doc.Header.DocumentID)
	pdf.Text(40, 36, title)

	// Subheader info
	pdf.SetY(70)
	pdf.SetFont("Helvetica", "B", 11)
	pdf.SetTextColor(15, 23, 42)
	pdf.CellFormat(300, 16, fmt.Sprintf("Vendor: %s", doc.Header.VendorName), "", 0, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 10)
	pdf.SetTextColor(71, 85, 105)
	pdf.CellFormat(215, 16, fmt.Sprintf("Date: %s", doc.Header.IssueDate), "", 1, "R", false, 0, "")

	if doc.Header.ReferenceID != "" {
		pdf.CellFormat(300, 16, fmt.Sprintf("PO Reference: %s", doc.Header.ReferenceID), "", 0, "L", false, 0, "")
	}
	if doc.Header.PaymentTerms != "" {
		pdf.CellFormat(215, 16, fmt.Sprintf("Terms: %s", doc.Header.PaymentTerms), "", 1, "R", false, 0, "")
	}

	// Table Header
	pdf.SetY(120)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(241, 245, 249)
	pdf.SetTextColor(51, 65, 85)

	pdf.CellFormat(80, 22, "SKU", "1", 0, "L", true, 0, "")
	pdf.CellFormat(215, 22, "Description", "1", 0, "L", true, 0, "")
	pdf.CellFormat(50, 22, "Qty", "1", 0, "R", true, 0, "")
	pdf.CellFormat(40, 22, "UOM", "1", 0, "C", true, 0, "")

	if doc.Header.DocumentType != config.DocTypeGRN {
		pdf.CellFormat(65, 22, "Unit Price", "1", 0, "R", true, 0, "")
		pdf.CellFormat(65, 22, "Total", "1", 1, "R", true, 0, "")
	} else {
		pdf.CellFormat(130, 22, "Warehouse Notes", "1", 1, "L", true, 0, "")
	}

	// Table Rows
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetTextColor(30, 41, 59)

	for i := range doc.Lines {
		line := &doc.Lines[i]
		rowY := pdf.GetY()

		// Assign actual PDF coordinates to line item for bounding box tracking
		line.BoundingBox = config.BoundingBox{
			Page:   1,
			X:      40,
			Y:      rowY,
			Width:  515,
			Height: 20,
		}

		pdf.CellFormat(80, 20, line.SKU, "1", 0, "L", false, 0, "")
		pdf.CellFormat(215, 20, line.Description, "1", 0, "L", false, 0, "")
		pdf.CellFormat(50, 20, fmt.Sprintf("%.1f", line.Quantity), "1", 0, "R", false, 0, "")
		pdf.CellFormat(40, 20, line.UnitOfMeasure, "1", 0, "C", false, 0, "")

		if doc.Header.DocumentType != config.DocTypeGRN {
			pdf.CellFormat(65, 20, fmt.Sprintf("%.2f", line.UnitPrice), "1", 0, "R", false, 0, "")
			pdf.CellFormat(65, 20, fmt.Sprintf("%.2f", line.TotalPrice), "1", 1, "R", false, 0, "")
		} else {
			pdf.CellFormat(130, 20, "Intake verified", "1", 1, "L", false, 0, "")
		}
	}

	// Summary Footer if applicable
	if doc.Header.DocumentType != config.DocTypeGRN {
		pdf.SetY(pdf.GetY() + 15)
		pdf.SetFont("Helvetica", "", 10)
		pdf.SetTextColor(71, 85, 105)

		pdf.CellFormat(385, 18, "Subtotal:", "", 0, "R", false, 0, "")
		pdf.CellFormat(130, 18, fmt.Sprintf("%s %.2f", doc.Header.Currency, doc.Header.Subtotal), "", 1, "R", false, 0, "")

		if doc.Header.TaxTotal > 0 {
			pdf.CellFormat(385, 18, "Sales Tax / VAT:", "", 0, "R", false, 0, "")
			pdf.CellFormat(130, 18, fmt.Sprintf("%s %.2f", doc.Header.Currency, doc.Header.TaxTotal), "", 1, "R", false, 0, "")
		}

		if doc.Header.FreightTotal > 0 {
			pdf.CellFormat(385, 18, "Freight / Shipping:", "", 0, "R", false, 0, "")
			pdf.CellFormat(130, 18, fmt.Sprintf("%s %.2f", doc.Header.Currency, doc.Header.FreightTotal), "", 1, "R", false, 0, "")
		}

		pdf.SetFont("Helvetica", "B", 12)
		pdf.SetTextColor(15, 23, 42)
		pdf.CellFormat(385, 24, "Total Amount Due:", "", 0, "R", false, 0, "")
		pdf.CellFormat(130, 24, fmt.Sprintf("%s %.2f", doc.Header.Currency, doc.Header.TotalAmount), "", 1, "R", false, 0, "")
	}

	return pdf.OutputFileAndClose(filePath)
}
