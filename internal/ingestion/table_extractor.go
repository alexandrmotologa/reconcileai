package ingestion

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/alexandrmotologa/reconcileai/internal/config"
)

var (
	skuRegex     = regexp.MustCompile(`^[A-Z0-9]{2,}[A-Z0-9\-_.]*$`)
	numOnlyRegex = regexp.MustCompile(`^-?\d+(?:\.\d+)?$`)
)

// extractLineItems locates line item rows and parses structured item columns.
func extractLineItems(lines []ExtractedLine, docType config.DocumentType) []config.LineItem {
	var items []config.LineItem

	tableStarted := false
	lineNum := 1

	for _, l := range lines {
		raw := strings.TrimSpace(l.String)
		lower := strings.ToLower(raw)

		// Check for table header
		if !tableStarted {
			if (strings.Contains(lower, "item") || strings.Contains(lower, "sku") || strings.Contains(lower, "description")) &&
				(strings.Contains(lower, "qty") || strings.Contains(lower, "quantity") || strings.Contains(lower, "price") || strings.Contains(lower, "amount")) {
				tableStarted = true
				continue
			}
		}

		if !tableStarted {
			continue
		}

		// Check for table terminator
		if strings.HasPrefix(lower, "subtotal") || strings.HasPrefix(lower, "total") ||
			strings.HasPrefix(lower, "notes") || strings.HasPrefix(lower, "payment terms") ||
			strings.HasPrefix(lower, "authorized by") || strings.HasPrefix(lower, "received by") {
			break
		}

		// Try parsing row into LineItem
		item, ok := parseLineRow(l, lineNum, docType)
		if ok {
			items = append(items, item)
			lineNum++
		}
	}

	return items
}

func parseLineRow(line ExtractedLine, lineNum int, docType config.DocumentType) (config.LineItem, bool) {
	words := line.Words
	if len(words) < 2 {
		return config.LineItem{}, false
	}

	// We expect tokens: [SKU, Description..., Qty, (UOM), UnitPrice, (TotalPrice)]
	// For Goods Receipt Note (GRN), UnitPrice might be omitted.
	var tokens []string
	for _, w := range words {
		clean := strings.Trim(w.Text, "$, \t")
		if clean != "" {
			tokens = append(tokens, clean)
		}
	}

	if len(tokens) < 2 {
		return config.LineItem{}, false
	}

	var sku string
	var uom = "EA"
	var qty float64
	var unitPrice float64
	var totalPrice float64
	var descTokens []string

	firstToken := tokens[0]
	if skuRegex.MatchString(firstToken) {
		sku = firstToken
		tokens = tokens[1:]
	} else {
		sku = fmt.Sprintf("SKU-%03d", lineNum)
	}

	// Parse numeric values from the end of tokens
	var numbers []float64
	var nonNumbers []string

	for i := len(tokens) - 1; i >= 0; i-- {
		tok := strings.ReplaceAll(tokens[i], ",", "")
		if val, err := strconv.ParseFloat(tok, 64); err == nil {
			numbers = append([]float64{val}, numbers...)
		} else {
			nonNumbers = append([]string{tokens[i]}, nonNumbers...)
		}
	}

	// Inspect non-numbers for UOM
	for _, tok := range nonNumbers {
		norm := NormalizeUOM(tok)
		if norm != tok || isKnownUOM(tok) {
			uom = norm
		} else {
			descTokens = append(descTokens, tok)
		}
	}

	// Determine Qty, UnitPrice, TotalPrice from parsed numbers
	switch len(numbers) {
	case 1:
		qty = numbers[0]
		unitPrice = 0
		totalPrice = 0
	case 2:
		qty = numbers[0]
		unitPrice = numbers[1]
		totalPrice = qty * unitPrice
	case 3:
		qty = numbers[0]
		unitPrice = numbers[1]
		totalPrice = numbers[2]
	default:
		if len(numbers) > 3 {
			// e.g. [lineNo, qty, unitPrice, totalPrice]
			qty = numbers[len(numbers)-3]
			unitPrice = numbers[len(numbers)-2]
			totalPrice = numbers[len(numbers)-1]
		} else {
			return config.LineItem{}, false
		}
	}

	if qty <= 0 {
		return config.LineItem{}, false
	}

	desc := strings.Join(descTokens, " ")
	if desc == "" {
		desc = sku
	}

	box := config.BoundingBox{
		Page:   line.Page,
		X:      line.MinX,
		Y:      line.Y,
		Width:  line.MaxX - line.MinX,
		Height: line.Height,
	}

	return config.LineItem{
		ID:            fmt.Sprintf("%s-L%02d", docType, lineNum),
		LineNumber:    lineNum,
		SKU:           sku,
		Description:   desc,
		Quantity:      qty,
		UnitOfMeasure: uom,
		UnitPrice:     unitPrice,
		TotalPrice:    totalPrice,
		BoundingBox:   box,
	}, true
}

func isKnownUOM(tok string) bool {
	upper := strings.ToUpper(strings.TrimSpace(tok))
	_, exists := unitRegistry[upper]
	return exists
}
