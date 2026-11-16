package erp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/alexandrmotologa/reconcileai/internal/config"
	"github.com/alexandrmotologa/reconcileai/internal/synth"
	"github.com/alexandrmotologa/reconcileai/internal/auditor"
)

func TestERPExports(t *testing.T) {
	dossier, err := synth.BuildCleanMatchDossier(t.TempDir())
	if err != nil {
		t.Fatalf("failed to build clean dossier: %v", err)
	}

	summary := auditor.ReconcileDossier(dossier, config.DefaultConfig())

	// 1. SAP IDoc XML
	xmlBytes, err := GenerateSAPIDocXML(summary, "1000")
	if err != nil {
		t.Fatalf("failed to generate sap idoc xml: %v", err)
	}
	xmlStr := string(xmlBytes)
	if !strings.Contains(xmlStr, "<INVOIC02>") || !strings.Contains(xmlStr, "Apex Industrial Fasteners Corp") {
		t.Errorf("invalid sap idoc xml output: %s", xmlStr)
	}

	// 2. NetSuite CSV
	csvBytes, err := GenerateNetSuiteCSV(summary)
	if err != nil {
		t.Fatalf("failed to generate netsuite csv: %v", err)
	}
	csvStr := string(csvBytes)
	if !strings.Contains(csvStr, "ExternalID,Vendor") || !strings.Contains(csvStr, "PO-2026-9011") {
		t.Errorf("invalid netsuite csv output: %s", csvStr)
	}

	// 3. QuickBooks JSON
	jsonBytes, err := GenerateQuickBooksJSON(summary)
	if err != nil {
		t.Fatalf("failed to generate quickbooks json: %v", err)
	}
	var qbMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &qbMap); err != nil {
		t.Fatalf("failed to parse qb json: %v", err)
	}
	if qbMap["DocNumber"] != "INV-APEX-8812" {
		t.Errorf("expected DocNumber INV-APEX-8812, got %v", qbMap["DocNumber"])
	}
}
