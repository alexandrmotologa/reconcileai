package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexandrmotologa/reconcileai/internal/config"
)

func createTestServer() *Server {
	cfg := config.DefaultConfig()
	srv := NewServer(cfg)

	// Register sample dossier
	dossier := &config.Dossier{
		ID:    "test_dossier_01",
		Title: "Test Vendor Dossier",
		PurchaseOrder: config.Document{
			Header: config.DocumentHeader{
				DocumentID:   "PO-001",
				VendorID:     "VEND-1",
				VendorName:   "Acme Corp",
				PaymentTerms: "2/10 Net 30",
				TotalAmount:  1000.0,
				Currency:     "USD",
			},
			Lines: []config.LineItem{
				{ID: "PO-L1", LineNumber: 1, SKU: "SKU-1", Description: "Widget Alpha", Quantity: 10, UnitOfMeasure: "EA", UnitPrice: 100.0, TotalPrice: 1000.0},
			},
		},
		GoodsReceipts: []config.Document{
			{
				Header: config.DocumentHeader{DocumentID: "GRN-001", VendorID: "VEND-1"},
				Lines: []config.LineItem{
					{ID: "GRN-L1", LineNumber: 1, SKU: "SKU-1", Quantity: 10, UnitOfMeasure: "EA"},
				},
			},
		},
		Invoice: config.Document{
			Header: config.DocumentHeader{
				DocumentID:   "INV-001",
				VendorID:     "VEND-1",
				VendorName:   "Acme Corp",
				IssueDate:    "2026-09-01",
				PaymentTerms: "2/10 Net 30",
				Subtotal:     1000.0,
				TotalAmount:  1000.0,
				Currency:     "USD",
			},
			Lines: []config.LineItem{
				{ID: "INV-L1", LineNumber: 1, SKU: "SKU-1", Description: "Widget Alpha", Quantity: 10, UnitOfMeasure: "EA", UnitPrice: 100.0, TotalPrice: 1000.0},
			},
		},
	}
	srv.RegisterDossier(dossier)
	return srv
}

func TestServerEndpoints(t *testing.T) {
	srv := createTestServer()
	handler := srv.Handler()

	// 1. Health check
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Health check failed with code %d", rec.Code)
	}

	// 2. List dossiers
	req = httptest.NewRequest(http.MethodGet, "/api/dossiers", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("List dossiers failed with code %d", rec.Code)
	}

	// 3. Reconcile single dossier
	req = httptest.NewRequest(http.MethodGet, "/api/reconcile/test_dossier_01", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Reconcile failed with code %d", rec.Code)
	}

	var sum config.AuditSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &sum); err != nil {
		t.Fatalf("Unmarshal summary failed: %v", err)
	}
	if sum.MatchStatus != "APPROVED" {
		t.Errorf("Expected APPROVED status, got %s", sum.MatchStatus)
	}
	if sum.EarlyDiscount == nil {
		t.Errorf("Expected EarlyDiscount to be calculated, got nil")
	}

	// 4. Metrics endpoint
	req = httptest.NewRequest(http.MethodGet, "/api/metrics", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Metrics failed with code %d", rec.Code)
	}
	var metrics MetricsSummary
	if err := json.Unmarshal(rec.Body.Bytes(), &metrics); err != nil {
		t.Fatalf("Unmarshal metrics failed: %v", err)
	}
	if metrics.ApprovedCount != 1 {
		t.Errorf("Expected approved count 1, got %d", metrics.ApprovedCount)
	}

	// 5. Vendor Scorecard endpoint
	req = httptest.NewRequest(http.MethodGet, "/api/vendors/scorecard", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Vendor scorecard failed with code %d", rec.Code)
	}
	var scorecards []VendorScorecard
	if err := json.Unmarshal(rec.Body.Bytes(), &scorecards); err != nil {
		t.Fatalf("Unmarshal scorecard failed: %v", err)
	}
	if len(scorecards) != 1 || scorecards[0].AccuracyRate != 100.0 {
		t.Errorf("Unexpected scorecard data: %+v", scorecards)
	}

	// 6. Export CSV
	req = httptest.NewRequest(http.MethodGet, "/api/export/csv/test_dossier_01", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Export CSV failed with code %d", rec.Code)
	}
	csvBody := rec.Body.String()
	if !strings.Contains(csvBody, "SKU-1") || !strings.Contains(csvBody, "Widget Alpha") {
		t.Errorf("CSV output missing line data: %s", csvBody)
	}

	// 7. Batch reconcile
	batchBody, _ := json.Marshal(BatchRequest{DossierIDs: []string{"test_dossier_01"}})
	req = httptest.NewRequest(http.MethodPost, "/api/batch", bytes.NewReader(batchBody))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("Batch reconcile failed with code %d", rec.Code)
	}
	var batchResp BatchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &batchResp); err != nil {
		t.Fatalf("Unmarshal batch resp failed: %v", err)
	}
	if batchResp.TotalProcessed != 1 {
		t.Errorf("Expected 1 processed dossier, got %d", batchResp.TotalProcessed)
	}
}
