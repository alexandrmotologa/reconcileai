package erp

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"time"

	"github.com/alexandrmotologa/reconcileai/internal/config"
)

// SAP IDoc INVOIC02 structures for automated MM/FI posting.
type IDoc struct {
	XMLName xml.Name  `xml:"INVOIC02"`
	EDI_DC  EDI_DC40  `xml:"EDI_DC40"`
	Header  E1EDK01   `xml:"E1EDK01"`
	Refs    []E1EDK02 `xml:"E1EDK02"`
	Lines   []E1EDP01 `xml:"E1EDP01"`
	Totals  []E1EDS01 `xml:"E1EDS01"`
}

type EDI_DC40 struct {
	TABNAM string `xml:"TABNAM"`
	MANDT  string `xml:"MANDT"`
	DOCNUM string `xml:"DOCNUM"`
	DIRECT string `xml:"DIRECT"`
	IDOCTP string `xml:"IDOCTYP"`
	MESTYP string `xml:"MESTYP"`
	SNDPRT string `xml:"SNDPRT"`
	SNDPRN string `xml:"SNDPRN"`
	RCVPRT string `xml:"RCVPRT"`
	RCVPRN string `xml:"RCVPRN"`
	CREDAT string `xml:"CREDAT"`
	CRETIM string `xml:"CRETIM"`
}

type E1EDK01 struct {
	CURCY string `xml:"CURCY"`
	ZTERM string `xml:"ZTERM"`
	BELNR string `xml:"BELNR"`
}

type E1EDK02 struct {
	QUALF string `xml:"QUALF"` // 001 = PO, 002 = Invoice, 004 = Delivery Note
	BELNR string `xml:"BELNR"`
}

type E1EDP01 struct {
	POSEX string  `xml:"POSEX"` // Line number
	MENGE float64 `xml:"MENGE"` // Quantity
	MENEE string  `xml:"MENEE"` // UOM
	VPREI float64 `xml:"VPREI"` // Unit Price
	NETWR float64 `xml:"NETWR"` // Net Value
	MATNR string  `xml:"E1EDP19>IDTNR"` // Material SKU
	KTEXT string  `xml:"E1EDP19>KTEXT"` // Description
}

type E1EDS01 struct {
	SUMID string  `xml:"SUMID"` // 001 = Net, 002 = Tax, 003 = Gross
	SUMAM float64 `xml:"SUMAM"`
	WAERQ string  `xml:"WAERQ"`
}

// GenerateSAPIDocXML generates an official SAP S/4HANA INVOIC02 settlement payload.
func GenerateSAPIDocXML(summary *config.AuditSummary, companyCode string) ([]byte, error) {
	if companyCode == "" {
		companyCode = "1000"
	}

	now := time.Now()
	docNum := fmt.Sprintf("IDOC%s%04d", now.Format("0601021504"), now.Nanosecond()%10000)

	idoc := IDoc{
		EDI_DC: EDI_DC40{
			TABNAM: "EDI_DC40",
			MANDT:  "100",
			DOCNUM: docNum,
			DIRECT: "2", // Inbound
			IDOCTP: "INVOIC02",
			MESTYP: "INVOIC",
			SNDPRT: "LF", // Vendor
			SNDPRN: summary.VendorName,
			RCVPRT: "LS",
			RCVPRN: companyCode,
			CREDAT: now.Format("20060102"),
			CRETIM: now.Format("150405"),
		},
		Header: E1EDK01{
			CURCY: summary.Currency,
			ZTERM: "NT30",
			BELNR: summary.InvoiceNumber,
		},
		Refs: []E1EDK02{
			{QUALF: "001", BELNR: summary.PONumber},
			{QUALF: "002", BELNR: summary.InvoiceNumber},
		},
	}

	var netTotal float64
	for i, pair := range summary.MatchedPairs {
		// Only settle authorized items
		payableQty := math.Min(pair.InvoicedQty, pair.ReceivedQty)
		if payableQty <= 0 {
			continue
		}

		lineNet := math.Round(payableQty*pair.POPrice*100) / 100
		netTotal += lineNet

		idoc.Lines = append(idoc.Lines, E1EDP01{
			POSEX: fmt.Sprintf("%06d", (i+1)*10),
			MENGE: payableQty,
			MENEE: "EA",
			VPREI: pair.POPrice,
			NETWR: lineNet,
			MATNR: pair.ItemSKU,
			KTEXT: pair.Description,
		})
	}

	netTotal = math.Round(netTotal*100) / 100
	taxTotal := math.Round((summary.TotalAuthorized-netTotal)*100) / 100
	if taxTotal < 0 {
		taxTotal = 0
	}

	idoc.Totals = []E1EDS01{
		{SUMID: "001", SUMAM: netTotal, WAERQ: summary.Currency},
		{SUMID: "002", SUMAM: taxTotal, WAERQ: summary.Currency},
		{SUMID: "003", SUMAM: summary.TotalAuthorized, WAERQ: summary.Currency},
	}

	var buf bytes.Buffer
	buf.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(idoc); err != nil {
		return nil, fmt.Errorf("encode sap idoc xml: %w", err)
	}

	return buf.Bytes(), nil
}
