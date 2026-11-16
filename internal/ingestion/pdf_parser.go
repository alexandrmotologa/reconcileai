package ingestion

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/alexandrmotologa/reconcileai/internal/config"
	"github.com/ledongthuc/pdf"
)

// ExtractedWord represents a grouped word with geometric coordinates.
type ExtractedWord struct {
	Text     string
	Page     int
	X        float64
	Y        float64
	Width    float64
	Height   float64
	FontSize float64
}

// ExtractedLine represents a horizontal line of text fragments.
type ExtractedLine struct {
	Page   int
	Y      float64
	Words  []ExtractedWord
	String string
	MinX   float64
	MaxX   float64
	Height float64
}

// ParsePDFDocument parses a PDF file and reconstructs headers and line-item tables.
func ParsePDFDocument(filePath string, docType config.DocumentType) (*config.Document, error) {
	file, reader, err := pdf.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open pdf failed: %w", err)
	}
	defer file.Close()

	numPages := reader.NumPage()
	var allLines []ExtractedLine

	for pageIdx := 1; pageIdx <= numPages; pageIdx++ {
		page := reader.Page(pageIdx)
		if page.V.IsNull() {
			continue
		}

		texts := page.Content().Text
		if len(texts) == 0 {
			continue
		}

		pageLines := clusterTextIntoLines(texts, pageIdx)
		allLines = append(allLines, pageLines...)
	}

	header := extractHeader(allLines, docType)
	lines := extractLineItems(allLines, docType)

	// If subtotal/total were not parsed explicitly from header labels, calculate from lines
	if header.Subtotal == 0 && len(lines) > 0 {
		var sum float64
		for _, l := range lines {
			sum += l.TotalPrice
		}
		header.Subtotal = math.Round(sum*100) / 100
	}
	if header.TotalAmount == 0 {
		header.TotalAmount = math.Round((header.Subtotal+header.TaxTotal+header.FreightTotal)*100) / 100
	}

	doc := &config.Document{
		Header:     header,
		Lines:      lines,
		RawPDFPath: filePath,
	}

	return doc, nil
}

// clusterTextIntoLines groups raw PDF text fragments by vertical position.
func clusterTextIntoLines(texts []pdf.Text, page int) []ExtractedLine {
	// Sort primarily by Y descending (top to bottom in PDF coordinates), then X ascending
	sort.Slice(texts, func(i, j int) bool {
		if math.Abs(texts[i].Y-texts[j].Y) < 2.0 {
			return texts[i].X < texts[j].X
		}
		return texts[i].Y > texts[j].Y
	})

	var lines []ExtractedLine
	var currentWords []ExtractedWord
	var currentY float64 = -1
	var currentHeight float64 = 10

	flushLine := func() {
		if len(currentWords) == 0 {
			return
		}
		sort.Slice(currentWords, func(i, j int) bool {
			return currentWords[i].X < currentWords[j].X
		})

		var sb strings.Builder
		minX := currentWords[0].X
		maxX := currentWords[0].X + currentWords[0].Width

		for i, w := range currentWords {
			if i > 0 {
				sb.WriteString(" ")
			}
			sb.WriteString(w.Text)
			endX := w.X + w.Width
			if endX > maxX {
				maxX = endX
			}
		}

		lines = append(lines, ExtractedLine{
			Page:   page,
			Y:      currentY,
			Words:  currentWords,
			String: strings.TrimSpace(sb.String()),
			MinX:   minX,
			MaxX:   maxX,
			Height: currentHeight,
		})
		currentWords = nil
	}

	for _, t := range texts {
		cleanText := strings.TrimSpace(t.S)
		if cleanText == "" {
			continue
		}

		if currentY < 0 {
			currentY = t.Y
			currentHeight = t.FontSize
			if currentHeight <= 0 {
				currentHeight = 10
			}
		}

		if math.Abs(t.Y-currentY) <= 3.0 {
			currentWords = append(currentWords, ExtractedWord{
				Text:     cleanText,
				Page:     page,
				X:        t.X,
				Y:        t.Y,
				Width:    t.W,
				Height:   t.FontSize,
				FontSize: t.FontSize,
			})
		} else {
			flushLine()
			currentY = t.Y
			currentHeight = t.FontSize
			if currentHeight <= 0 {
				currentHeight = 10
			}
			currentWords = append(currentWords, ExtractedWord{
				Text:     cleanText,
				Page:     page,
				X:        t.X,
				Y:        t.Y,
				Width:    t.W,
				Height:   t.FontSize,
				FontSize: t.FontSize,
			})
		}
	}
	flushLine()

	return lines
}

var (
	docNumRegex   = regexp.MustCompile(`(?i)(?:PO|INV|GRN|Invoice|Order|Receipt|Document)\s*#?[:\s]*([A-Z0-9\-_]+)`)
	refPoRegex    = regexp.MustCompile(`(?i)(?:PO\s*Ref|Purchase\s*Order|PO\s*#|Order\s*Ref)[:\s]*([A-Z0-9\-_]+)`)
	dateRegex     = regexp.MustCompile(`\b(\d{4}[-/.]\d{1,2}[-/.]\d{1,2}|\d{1,2}[-/.]\d{1,2}[-/.]\d{4})\b`)
	moneyRegex    = regexp.MustCompile(`\$?\s*([0-9]{1,3}(?:,[0-9]{3})*(?:\.[0-9]{2}))`)
	vendorRegex   = regexp.MustCompile(`(?i)(?:Vendor|Supplier|From|Issued By)[:\s]*(.+)`)
	taxIdRegex    = regexp.MustCompile(`(?i)(?:Tax ID|VAT|EIN)[:\s]*([A-Z0-9\-]+)`)
	currencyRegex = regexp.MustCompile(`\b(USD|EUR|GBP|CAD|AUD|JPY)\b|(\$|€|£)`)
)

func extractHeader(lines []ExtractedLine, docType config.DocumentType) config.DocumentHeader {
	header := config.DocumentHeader{
		DocumentType: docType,
		Currency:     "USD",
		PaymentTerms: "Net 30",
	}

	for _, l := range lines {
		raw := l.String

		// Extract currency
		if match := currencyRegex.FindStringSubmatch(raw); len(match) > 0 {
			switch {
			case match[1] != "":
				header.Currency = match[1]
			case match[2] == "$":
				header.Currency = "USD"
			case match[2] == "€":
				header.Currency = "EUR"
			case match[2] == "£":
				header.Currency = "GBP"
			}
		}

		// Extract document ID
		if header.DocumentID == "" {
			if match := docNumRegex.FindStringSubmatch(raw); len(match) > 1 {
				header.DocumentID = match[1]
			}
		}

		// Extract reference PO
		if header.ReferenceID == "" && docType != config.DocTypePO {
			if match := refPoRegex.FindStringSubmatch(raw); len(match) > 1 {
				header.ReferenceID = match[1]
			}
		}

		// Extract Issue Date
		if header.IssueDate == "" {
			if match := dateRegex.FindStringSubmatch(raw); len(match) > 1 {
				header.IssueDate = match[1]
			}
		}

		// Extract Vendor name
		if header.VendorName == "" {
			if match := vendorRegex.FindStringSubmatch(raw); len(match) > 1 {
				header.VendorName = strings.TrimSpace(match[1])
			}
		}

		// Extract Tax ID
		if header.VendorTaxID == "" {
			if match := taxIdRegex.FindStringSubmatch(raw); len(match) > 1 {
				header.VendorTaxID = match[1]
			}
		}

		// Extract Totals
		lower := strings.ToLower(raw)
		if strings.Contains(lower, "subtotal") {
			if amt := parseMoney(raw); amt > 0 {
				header.Subtotal = amt
			}
		} else if strings.Contains(lower, "tax") || strings.Contains(lower, "vat") {
			if amt := parseMoney(raw); amt > 0 {
				header.TaxTotal = amt
			}
		} else if strings.Contains(lower, "freight") || strings.Contains(lower, "shipping") {
			if amt := parseMoney(raw); amt > 0 {
				header.FreightTotal = amt
			}
		} else if strings.Contains(lower, "total") && !strings.Contains(lower, "subtotal") {
			if amt := parseMoney(raw); amt > 0 {
				header.TotalAmount = amt
			}
		}
	}

	if header.DocumentID == "" {
		header.DocumentID = fmt.Sprintf("%s-AUTO-01", docType)
	}

	return header
}

func parseMoney(str string) float64 {
	matches := moneyRegex.FindAllStringSubmatch(str, -1)
	if len(matches) == 0 {
		return 0
	}
	last := matches[len(matches)-1][1]
	clean := strings.ReplaceAll(last, ",", "")
	val, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0
	}
	return val
}
