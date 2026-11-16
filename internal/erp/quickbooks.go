package erp

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/alexandrmotologa/reconcileai/internal/config"
)

// QuickBooksBill represents the Intuit QuickBooks Online REST Bill schema.
type QuickBooksBill struct {
	DocNumber    string               `json:"DocNumber"`
	TxnDate      string               `json:"TxnDate"`
	DueDate      string               `json:"DueDate"`
	VendorRef    QBReference          `json:"VendorRef"`
	APAccountRef QBReference          `json:"APAccountRef"`
	TotalAmt     float64              `json:"TotalAmt"`
	CurrencyRef  QBReference          `json:"CurrencyRef"`
	PrivateNote  string               `json:"PrivateNote"`
	Line         []QuickBooksBillLine `json:"Line"`
}

type QBReference struct {
	Value string `json:"value"`
	Name  string `json:"name,omitempty"`
}

type QuickBooksBillLine struct {
	Id                      string             `json:"Id,omitempty"`
	LineNum                 int                `json:"LineNum"`
	Description             string             `json:"Description"`
	Amount                  float64            `json:"Amount"`
	DetailType              string             `json:"DetailType"`
	ItemBasedExpenseLineDetail QBItemDetail    `json:"ItemBasedExpenseLineDetail"`
}

type QBItemDetail struct {
	ItemRef   QBReference `json:"ItemRef"`
	UnitPrice float64     `json:"UnitPrice"`
	Qty       float64     `json:"Qty"`
}

// GenerateQuickBooksJSON formats an authorized settlement as QuickBooks Online Bill JSON.
func GenerateQuickBooksJSON(summary *config.AuditSummary) ([]byte, error) {
	today := time.Now().Format("2006-01-02")
	dueDate := time.Now().AddDate(0, 1, 0).Format("2006-01-02")

	bill := QuickBooksBill{
		DocNumber: summary.InvoiceNumber,
		TxnDate:   today,
		DueDate:   dueDate,
		VendorRef: QBReference{
			Value: "VENDOR-01",
			Name:  summary.VendorName,
		},
		APAccountRef: QBReference{
			Value: "AP-ACCOUNT-3001",
			Name:  "Accounts Payable",
		},
		TotalAmt:    summary.TotalAuthorized,
		CurrencyRef: QBReference{Value: summary.Currency},
		PrivateNote: fmt.Sprintf("reconcileai authorized bill (PO: %s | Prevented overcharge: %s %.2f)",
			summary.PONumber, summary.Currency, summary.PreventedOvercharge),
	}

	lineIdx := 1
	for _, pair := range summary.MatchedPairs {
		payableQty := math.Min(pair.InvoicedQty, pair.ReceivedQty)
		if payableQty <= 0 {
			continue
		}

		lineAmount := math.Round(payableQty*pair.POPrice*100) / 100

		bill.Line = append(bill.Line, QuickBooksBillLine{
			Id:          fmt.Sprintf("%d", lineIdx),
			LineNum:     lineIdx,
			Description: pair.Description,
			Amount:      lineAmount,
			DetailType:  "ItemBasedExpenseLineDetail",
			ItemBasedExpenseLineDetail: QBItemDetail{
				ItemRef: QBReference{
					Value: pair.ItemSKU,
					Name:  pair.ItemSKU,
				},
				UnitPrice: pair.POPrice,
				Qty:       payableQty,
			},
		})
		lineIdx++
	}

	return json.MarshalIndent(bill, "", "  ")
}
