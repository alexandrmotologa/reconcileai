package auditor

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/alexandrmotologa/reconcileai/internal/config"
)

var (
	// Matches terms like "2/10 Net 30", "1.5/15 Net 45", "2/10, Net 30"
	termPatternSlash = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*/\s*(\d+)(?:[,\s]+net\s+(\d+))?`)
	// Matches terms like "2% 10 Net 30"
	termPatternPercent = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*%\s*(\d+)(?:[,\s]+net\s+(\d+))?`)
)

// ParsePaymentTerms extracts discount percentage, discount window days, and full payment net days.
func ParsePaymentTerms(terms string) (discountPercent float64, discountDays int, netDays int, found bool) {
	terms = strings.TrimSpace(terms)
	if terms == "" {
		return 0, 0, 0, false
	}

	// Try standard 2/10 Net 30 pattern
	if m := termPatternSlash.FindStringSubmatch(terms); len(m) >= 3 {
		pct, err1 := strconv.ParseFloat(m[1], 64)
		days, err2 := strconv.Atoi(m[2])
		if err1 == nil && err2 == nil && pct > 0 && days > 0 {
			net := 30 // default standard net days
			if len(m) >= 4 && m[3] != "" {
				if n, err3 := strconv.Atoi(m[3]); err3 == nil && n > days {
					net = n
				}
			}
			return pct, days, net, true
		}
	}

	// Try 2% 10 Net 30 pattern
	if m := termPatternPercent.FindStringSubmatch(terms); len(m) >= 3 {
		pct, err1 := strconv.ParseFloat(m[1], 64)
		days, err2 := strconv.Atoi(m[2])
		if err1 == nil && err2 == nil && pct > 0 && days > 0 {
			net := 30
			if len(m) >= 4 && m[3] != "" {
				if n, err3 := strconv.Atoi(m[3]); err3 == nil && n > days {
					net = n
				}
			}
			return pct, days, net, true
		}
	}

	return 0, 0, 0, false
}

// CalculateEarlyDiscount analyzes available vendor discount terms and computes cash savings & ROI.
func CalculateEarlyDiscount(terms, issueDateStr string, authorizedAmount float64) *config.EarlyDiscount {
	if authorizedAmount <= 0 {
		return nil
	}

	pct, discDays, netDays, ok := ParsePaymentTerms(terms)
	if !ok {
		return nil
	}

	discountAmount := math.Round(authorizedAmount*(pct/100.0)*100) / 100

	// Calculate deadline date
	var deadlineStr string
	var isEligible bool = true

	// Parse issue date (try YYYY-MM-DD first, then RFC3339)
	parsedDate, err := time.Parse("2006-01-02", strings.TrimSpace(issueDateStr))
	if err != nil {
		parsedDate, err = time.Parse(time.RFC3339, strings.TrimSpace(issueDateStr))
	}

	if err == nil {
		deadline := parsedDate.AddDate(0, 0, discDays)
		deadlineStr = deadline.Format("2006-01-02")
		// Check eligibility against current time or nominal window
		isEligible = time.Now().Before(deadline.AddDate(0, 0, 1))
	} else {
		deadlineStr = fmt.Sprintf("+%d days from invoice date", discDays)
	}

	// Calculate Annualized Equivalent Rate (APR / ROI):
	// Formula: (Discount% / (100 - Discount%)) * (365 / (NetDays - DiscountDays)) * 100
	var annualizedROI float64
	if netDays > discDays && (100.0-pct) > 0 {
		rawROI := (pct / (100.0 - pct)) * (365.0 / float64(netDays-discDays)) * 100.0
		annualizedROI = math.Round(rawROI*100) / 100
	}

	return &config.EarlyDiscount{
		Terms:            terms,
		DiscountPercent:  pct,
		DiscountDays:     discDays,
		NetDays:          netDays,
		DiscountAmount:   discountAmount,
		DiscountDeadline: deadlineStr,
		AnnualizedROI:    annualizedROI,
		IsEligible:       isEligible,
	}
}
