package auditor

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/alexandrmotologa/reconcileai/internal/config"
)

// LedgerEntry stores key fingerprint data for an ingested invoice.
type LedgerEntry struct {
	DossierID     string  `json:"dossier_id"`
	VendorID      string  `json:"vendor_id"`
	VendorName    string  `json:"vendor_name"`
	InvoiceNumber string  `json:"invoice_number"`
	IssueDate     string  `json:"issue_date"`
	TotalAmount   float64 `json:"total_amount"`
	AuditHash     string  `json:"audit_hash"`
}

// DuplicateLedger maintains thread-safe historical indexes to prevent double billing.
type DuplicateLedger struct {
	mu           sync.RWMutex
	exactIndex   map[string]LedgerEntry // key: vendor_id + ":" + invoice_number
	amountIndex  map[string]LedgerEntry // key: vendor_id + ":" + issue_date + ":" + formatted_amount
}

// NewDuplicateLedger creates an initialized duplicate tracking ledger.
func NewDuplicateLedger() *DuplicateLedger {
	return &DuplicateLedger{
		exactIndex:  make(map[string]LedgerEntry),
		amountIndex: make(map[string]LedgerEntry),
	}
}

// NormalizeKey normalizes strings for consistent hashing.
func normalizeKey(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func makeExactKey(vendorID, invoiceNum string) string {
	raw := fmt.Sprintf("%s:%s", normalizeKey(vendorID), normalizeKey(invoiceNum))
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

func makeAmountKey(vendorID, issueDate string, amount float64) string {
	raw := fmt.Sprintf("%s:%s:%.2f", normalizeKey(vendorID), normalizeKey(issueDate), amount)
	hash := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(hash[:])
}

// CheckDuplicate evaluates if the invoice has already been audited or exhibits duplicate patterns.
func (l *DuplicateLedger) CheckDuplicate(dossier *config.Dossier) *config.Discrepancy {
	if l == nil || dossier == nil {
		return nil
	}

	inv := &dossier.Invoice
	vendorID := inv.Header.VendorID
	if vendorID == "" {
		vendorID = inv.Header.VendorName
	}
	invoiceNum := inv.Header.DocumentID
	issueDate := inv.Header.IssueDate
	totalAmount := inv.Header.TotalAmount
	if totalAmount <= 0 {
		totalAmount = inv.Header.Subtotal + inv.Header.TaxTotal + inv.Header.FreightTotal
	}

	exactKey := makeExactKey(vendorID, invoiceNum)
	amountKey := makeAmountKey(vendorID, issueDate, totalAmount)

	l.mu.RLock()
	defer l.mu.RUnlock()

	// Check 1: Exact duplicate (same vendor + invoice number)
	if prior, exists := l.exactIndex[exactKey]; exists && prior.DossierID != dossier.ID {
		return &config.Discrepancy{
			ID:             fmt.Sprintf("DISC-DUP-EXACT-%s", invoiceNum),
			Type:           config.DiscrepancyDuplicateInvoice,
			Severity:       config.SeverityCritical,
			Title:          fmt.Sprintf("Duplicate Invoice Number: %s", invoiceNum),
			Description:    fmt.Sprintf("Invoice %s from vendor '%s' was previously processed and settled in dossier %s (Audit Hash: %s). Duplicate submissions must not be re-authorized.", invoiceNum, inv.Header.VendorName, prior.DossierID, prior.AuditHash[:12]),
			ExpectedValue:  0.0,
			ActualValue:    totalAmount,
			VarianceAmount: totalAmount,
			Currency:       inv.Header.Currency,
			TargetDocument: config.DocTypeInvoice,
			PageNumber:     1,
		}
	}

	// Check 2: Fuzzy duplicate (same vendor + same amount + same issue date, but different invoice number)
	if prior, exists := l.amountIndex[amountKey]; exists && prior.DossierID != dossier.ID && normalizeKey(prior.InvoiceNumber) != normalizeKey(invoiceNum) {
		return &config.Discrepancy{
			ID:             fmt.Sprintf("DISC-DUP-FUZZY-%s", invoiceNum),
			Type:           config.DiscrepancyDuplicateInvoice,
			Severity:       config.SeverityCritical,
			Title:          fmt.Sprintf("Suspected Duplicate Billing: %.2f %s", totalAmount, inv.Header.Currency),
			Description:    fmt.Sprintf("An identical invoice total of %.2f %s dated %s was previously processed for vendor '%s' under invoice number %s (dossier %s). Verify if this is a reissued or duplicate claim.", totalAmount, inv.Header.Currency, issueDate, inv.Header.VendorName, prior.InvoiceNumber, prior.DossierID),
			ExpectedValue:  0.0,
			ActualValue:    totalAmount,
			VarianceAmount: totalAmount,
			Currency:       inv.Header.Currency,
			TargetDocument: config.DocTypeInvoice,
			PageNumber:     1,
		}
	}

	return nil
}

// Record registers an audited dossier in the ledger.
func (l *DuplicateLedger) Record(dossier *config.Dossier, auditHash string) {
	if l == nil || dossier == nil {
		return
	}

	inv := &dossier.Invoice
	vendorID := inv.Header.VendorID
	if vendorID == "" {
		vendorID = inv.Header.VendorName
	}
	invoiceNum := inv.Header.DocumentID
	issueDate := inv.Header.IssueDate
	totalAmount := inv.Header.TotalAmount
	if totalAmount <= 0 {
		totalAmount = inv.Header.Subtotal + inv.Header.TaxTotal + inv.Header.FreightTotal
	}

	entry := LedgerEntry{
		DossierID:     dossier.ID,
		VendorID:      vendorID,
		VendorName:    inv.Header.VendorName,
		InvoiceNumber: invoiceNum,
		IssueDate:     issueDate,
		TotalAmount:   totalAmount,
		AuditHash:     auditHash,
	}

	exactKey := makeExactKey(vendorID, invoiceNum)
	amountKey := makeAmountKey(vendorID, issueDate, totalAmount)

	l.mu.Lock()
	defer l.mu.Unlock()

	l.exactIndex[exactKey] = entry
	l.amountIndex[amountKey] = entry
}

// Count returns the number of registered unique invoices.
func (l *DuplicateLedger) Count() int {
	if l == nil {
		return 0
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.exactIndex)
}

// Clear resets all historical entries.
func (l *DuplicateLedger) Clear() {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.exactIndex = make(map[string]LedgerEntry)
	l.amountIndex = make(map[string]LedgerEntry)
}
