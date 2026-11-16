package config

// DocumentType represents the category of AP accounting document.
type DocumentType string

const (
	DocTypePO      DocumentType = "PO"
	DocTypeGRN     DocumentType = "GRN"
	DocTypeInvoice DocumentType = "INVOICE"
)

// BoundingBox defines coordinate bounds on a document page for visual audit grounding.
type BoundingBox struct {
	Page   int     `json:"page"`
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// LineItem represents an individual goods or service entry.
type LineItem struct {
	ID            string      `json:"id"`
	LineNumber    int         `json:"line_number"`
	SKU           string      `json:"sku"`
	Description   string      `json:"description"`
	Quantity      float64     `json:"quantity"`
	UnitOfMeasure string      `json:"uom"`
	UnitPrice     float64     `json:"unit_price"`
	TotalPrice    float64     `json:"total_price"`
	TaxRate       float64     `json:"tax_rate"`
	TaxAmount     float64     `json:"tax_amount"`
	BoundingBox   BoundingBox `json:"bounding_box"`
}

// DocumentHeader carries metadata for an AP document.
type DocumentHeader struct {
	DocumentID   string       `json:"document_id"`
	DocumentType DocumentType `json:"document_type"`
	ReferenceID  string       `json:"reference_id"` // Cross-reference PO number
	VendorID     string       `json:"vendor_id"`
	VendorName   string       `json:"vendor_name"`
	VendorTaxID  string       `json:"vendor_tax_id"`
	IssueDate    string       `json:"issue_date"`
	Currency     string       `json:"currency"`
	Subtotal     float64      `json:"subtotal"`
	TaxTotal     float64      `json:"tax_total"`
	FreightTotal float64      `json:"freight_total"`
	TotalAmount  float64      `json:"total_amount"`
	PaymentTerms string       `json:"payment_terms"`
}

// Document represents an ingested accounting document.
type Document struct {
	Header      DocumentHeader `json:"header"`
	Lines       []LineItem     `json:"lines"`
	RawPDFPath  string         `json:"raw_pdf_path,omitempty"`
	RawChecksum string         `json:"raw_checksum,omitempty"`
}

// Dossier encapsulates the tripartite records for one transaction.
type Dossier struct {
	ID            string     `json:"id"`
	Title         string     `json:"title"`
	PurchaseOrder Document   `json:"purchase_order"`
	GoodsReceipts []Document `json:"goods_receipts"`
	Invoice       Document   `json:"invoice"`
}

// DiscrepancyType classifies detected accounting anomalies.
type DiscrepancyType string

const (
	DiscrepancyQtyOverbilled     DiscrepancyType = "QUANTITY_OVERBILLED"
	DiscrepancyPriceVariance     DiscrepancyType = "PRICE_VARIANCE_EXCEEDED"
	DiscrepancyUnauthorizedFee   DiscrepancyType = "UNAUTHORIZED_SURCHARGE"
	DiscrepancyArithmeticMismatch DiscrepancyType = "ARITHMETIC_MISMATCH"
	DiscrepancyUnmatchedItem     DiscrepancyType = "UNMATCHED_INVOICE_ITEM"
	DiscrepancyUOMMismatch       DiscrepancyType = "UOM_CONVERSION_ERROR"
	DiscrepancyFXVariance        DiscrepancyType = "CURRENCY_FX_VARIANCE"
	DiscrepancyDuplicateInvoice  DiscrepancyType = "DUPLICATE_INVOICE"
)

// DiscrepancySeverity denotes risk level.
type DiscrepancySeverity string

const (
	SeverityCritical DiscrepancySeverity = "CRITICAL"
	SeverityWarning  DiscrepancySeverity = "WARNING"
	SeverityInfo     DiscrepancySeverity = "INFO"
)

// Discrepancy details an audited variance between documents.
type Discrepancy struct {
	ID             string              `json:"id"`
	Type           DiscrepancyType     `json:"type"`
	Severity       DiscrepancySeverity `json:"severity"`
	Title          string              `json:"title"`
	Description    string              `json:"description"`
	ItemSKU        string              `json:"item_sku"`
	ExpectedValue  float64             `json:"expected_value"`
	ActualValue    float64             `json:"actual_value"`
	VarianceAmount float64             `json:"variance_amount"`
	Currency       string              `json:"currency"`
	TargetDocument DocumentType        `json:"target_document"`
	PageNumber     int                 `json:"page_number"`
	BoundingBox    BoundingBox         `json:"bounding_box"`
}

// MatchedLinePair represents the bipartite association of invoice line to PO line.
type MatchedLinePair struct {
	InvoiceLineID  string   `json:"invoice_line_id"`
	POLineID       string   `json:"po_line_id"`
	GRNLineIDs     []string `json:"grn_line_ids"`
	ItemSKU        string   `json:"item_sku"`
	Description    string   `json:"description"`
	Confidence     float64  `json:"confidence"`
	MatchReason    string   `json:"match_reason"`
	InvoicedQty    float64  `json:"invoiced_qty"`
	ReceivedQty    float64  `json:"received_qty"`
	OrderedQty     float64  `json:"ordered_qty"`
	InvoicedPrice  float64  `json:"invoiced_price"`
	POPrice        float64  `json:"po_price"`
	PriceVariance  float64  `json:"price_variance"`
	HasDiscrepancy bool     `json:"has_discrepancy"`
}

// EarlyDiscount specifies early payment discount terms, captured savings, and annualized ROI.
type EarlyDiscount struct {
	Terms            string  `json:"terms"`
	DiscountPercent  float64 `json:"discount_percent"`
	DiscountDays     int     `json:"discount_days"`
	NetDays          int     `json:"net_days"`
	DiscountAmount   float64 `json:"discount_amount"`
	DiscountDeadline string  `json:"discount_deadline"`
	AnnualizedROI    float64 `json:"annualized_roi"`
	IsEligible       bool    `json:"is_eligible"`
}

// AuditSummary contains the final reconciliation verdict.
type AuditSummary struct {
	DossierID           string            `json:"dossier_id"`
	VendorName          string            `json:"vendor_name"`
	InvoiceNumber       string            `json:"invoice_number"`
	PONumber            string            `json:"po_number"`
	MatchStatus         string            `json:"match_status"` // APPROVED, FLAGGED_DISCREPANCY, REJECTED
	TotalInvoiced       float64           `json:"total_invoiced"`
	TotalAuthorized     float64           `json:"total_authorized"`
	PreventedOvercharge float64           `json:"prevented_overcharge"`
	Currency            string            `json:"currency"`
	EarlyDiscount       *EarlyDiscount    `json:"early_discount,omitempty"`
	Discrepancies       []Discrepancy     `json:"discrepancies"`
	MatchedPairs        []MatchedLinePair `json:"matched_pairs"`
	AuditedAt           string            `json:"audited_at"`
	ExecutionTimeMs     int64             `json:"execution_time_ms"`
	AuditHash           string            `json:"audit_hash"`
}
