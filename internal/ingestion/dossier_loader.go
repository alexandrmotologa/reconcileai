package ingestion

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alexandrmotologa/reconcileai/internal/config"
)

// LoadDossierFromJSON reads a structured JSON dossier from disk.
func LoadDossierFromJSON(path string) (*config.Dossier, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dossier json: %w", err)
	}

	var dossier config.Dossier
	if err := json.Unmarshal(data, &dossier); err != nil {
		return nil, fmt.Errorf("unmarshal dossier json: %w", err)
	}

	if dossier.ID == "" {
		dossier.ID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	return &dossier, nil
}

// LoadDossierFromDirectory inspects a directory for PO, GRN, and INV files (PDF or JSON).
func LoadDossierFromDirectory(dirPath string) (*config.Dossier, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("read directory %s: %w", dirPath, err)
	}

	// Check if a single manifest dossier.json exists
	manifestPath := filepath.Join(dirPath, "dossier.json")
	if _, err := os.Stat(manifestPath); err == nil {
		return LoadDossierFromJSON(manifestPath)
	}

	var poFile string
	var invFile string
	var grnFiles []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		lower := strings.ToLower(name)
		ext := filepath.Ext(lower)

		if ext != ".pdf" && ext != ".json" {
			continue
		}

		fullPath := filepath.Join(dirPath, name)

		switch {
		case strings.HasPrefix(lower, "po") || strings.Contains(lower, "purchase_order"):
			poFile = fullPath
		case strings.HasPrefix(lower, "inv") || strings.Contains(lower, "invoice"):
			invFile = fullPath
		case strings.HasPrefix(lower, "grn") || strings.Contains(lower, "receipt") || strings.Contains(lower, "delivery"):
			grnFiles = append(grnFiles, fullPath)
		}
	}

	if poFile == "" {
		return nil, fmt.Errorf("no purchase order document found in %s", dirPath)
	}
	if invFile == "" {
		return nil, fmt.Errorf("no invoice document found in %s", dirPath)
	}
	if len(grnFiles) == 0 {
		return nil, fmt.Errorf("no goods receipt document found in %s", dirPath)
	}

	poDoc, err := ParsePDFDocument(poFile, config.DocTypePO)
	if err != nil {
		return nil, fmt.Errorf("parse po pdf: %w", err)
	}

	invDoc, err := ParsePDFDocument(invFile, config.DocTypeInvoice)
	if err != nil {
		return nil, fmt.Errorf("parse invoice pdf: %w", err)
	}

	var grnDocs []config.Document
	for _, grnPath := range grnFiles {
		grnDoc, err := ParsePDFDocument(grnPath, config.DocTypeGRN)
		if err != nil {
			return nil, fmt.Errorf("parse grn pdf (%s): %w", grnPath, err)
		}
		grnDocs = append(grnDocs, *grnDoc)
	}

	dossierID := filepath.Base(dirPath)
	return &config.Dossier{
		ID:            dossierID,
		Title:         fmt.Sprintf("Dossier %s - %s", dossierID, invDoc.Header.VendorName),
		PurchaseOrder: *poDoc,
		GoodsReceipts: grnDocs,
		Invoice:       *invDoc,
	}, nil
}

// ComputeDocumentChecksum returns a SHA-256 fingerprint for document immutability.
func ComputeDocumentChecksum(doc *config.Document) string {
	h := sha256.New()
	h.Write([]byte(doc.Header.DocumentID))
	h.Write([]byte(doc.Header.VendorID))
	h.Write([]byte(doc.Header.IssueDate))
	for _, l := range doc.Lines {
		h.Write([]byte(fmt.Sprintf("%s:%s:%.4f:%.4f", l.SKU, l.Description, l.Quantity, l.UnitPrice)))
	}
	return hex.EncodeToString(h.Sum(nil))
}
