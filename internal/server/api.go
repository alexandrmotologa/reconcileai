package server

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/alexandrmotologa/reconcileai/internal/auditor"
	"github.com/alexandrmotologa/reconcileai/internal/config"
	"github.com/alexandrmotologa/reconcileai/internal/erp"
	"github.com/alexandrmotologa/reconcileai/internal/ingestion"
	"github.com/alexandrmotologa/reconcileai/internal/report"
	"github.com/alexandrmotologa/reconcileai/web"
)

// Server coordinates HTTP API endpoints and embedded web cockpit.
type Server struct {
	Config    config.AppConfig
	SSE       *SSEHub
	ledger    *auditor.DuplicateLedger
	mu        sync.RWMutex
	dossiers  map[string]*config.Dossier
	summaries map[string]*config.AuditSummary
	history   []*config.AuditSummary
}

// NewServer creates a new API and Cockpit server instance.
func NewServer(cfg config.AppConfig) *Server {
	return &Server{
		Config:    cfg,
		SSE:       NewSSEHub(),
		ledger:    auditor.NewDuplicateLedger(),
		dossiers:  make(map[string]*config.Dossier),
		summaries: make(map[string]*config.AuditSummary),
		history:   make([]*config.AuditSummary, 0),
	}
}

// RegisterDossier adds an active transaction dossier to the in-memory index.
func (s *Server) RegisterDossier(d *config.Dossier) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dossiers[d.ID] = d
}

// Handler configures HTTP routes and returns the top-level handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/dossiers", s.handleListDossiers)
	mux.HandleFunc("/api/dossiers/", s.handleGetDossier)
	mux.HandleFunc("/api/reconcile/", s.handleReconcile)
	mux.HandleFunc("/api/history", s.handleHistory)
	mux.HandleFunc("/api/metrics", s.handleMetrics)
	mux.HandleFunc("/api/vendors/scorecard", s.handleVendorScorecard)
	mux.HandleFunc("/api/batch", s.handleBatch)
	mux.HandleFunc("/api/export/sap/", s.handleExportSAP)
	mux.HandleFunc("/api/export/netsuite/", s.handleExportNetSuite)
	mux.HandleFunc("/api/export/quickbooks/", s.handleExportQuickBooks)
	mux.HandleFunc("/api/export/dispute/", s.handleExportDispute)
	mux.HandleFunc("/api/export/evidence/", s.handleExportEvidencePDF)
	mux.HandleFunc("/api/export/csv/", s.handleExportCSV)
	mux.HandleFunc("/api/upload", s.handleUpload)
	mux.HandleFunc("/api/stream", s.SSE.HandleSSE)

	// Embedded React 19 Cockpit
	mux.Handle("/", web.Handler())

	return withCORS(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"service":   "reconcileai",
		"version":   "1.0.0",
		"dossiers":  len(s.dossiers),
		"timestamp": os.Getenv("ENV"),
	})
}

func (s *Server) handleListDossiers(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*config.Dossier
	for _, d := range s.dossiers {
		list = append(list, d)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (s *Server) handleGetDossier(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/dossiers/")
	s.mu.RLock()
	dossier, found := s.dossiers[id]
	s.mu.RUnlock()

	if !found {
		http.Error(w, "dossier not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dossier)
}

func (s *Server) handleReconcile(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/reconcile/")
	s.mu.RLock()
	dossier, found := s.dossiers[id]
	s.mu.RUnlock()

	if !found {
		http.Error(w, "dossier not found", http.StatusNotFound)
		return
	}

	summary := auditor.ReconcileDossierWithLedger(dossier, s.Config, s.ledger)

	s.mu.Lock()
	s.summaries[id] = summary
	s.history = append(s.history, summary)
	s.mu.Unlock()

	// Broadcast SSE event
	s.SSE.Broadcast("AUDIT_COMPLETE", summary)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func (s *Server) handleExportSAP(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/export/sap/")
	summary := s.getOrCreateSummary(id)
	if summary == nil {
		http.Error(w, "dossier not found", http.StatusNotFound)
		return
	}

	xmlData, err := erp.GenerateSAPIDocXML(summary, "1000")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"sap_%s.xml\"", id))
	w.Write(xmlData)
}

func (s *Server) handleExportNetSuite(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/export/netsuite/")
	summary := s.getOrCreateSummary(id)
	if summary == nil {
		http.Error(w, "dossier not found", http.StatusNotFound)
		return
	}

	csvData, err := erp.GenerateNetSuiteCSV(summary)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"netsuite_%s.csv\"", id))
	w.Write(csvData)
}

func (s *Server) handleExportQuickBooks(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/export/quickbooks/")
	summary := s.getOrCreateSummary(id)
	if summary == nil {
		http.Error(w, "dossier not found", http.StatusNotFound)
		return
	}

	jsonData, err := erp.GenerateQuickBooksJSON(summary)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"quickbooks_%s.json\"", id))
	w.Write(jsonData)
}

func (s *Server) handleExportDispute(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/export/dispute/")
	summary := s.getOrCreateSummary(id)
	if summary == nil {
		http.Error(w, "dossier not found", http.StatusNotFound)
		return
	}

	memo := report.GenerateVendorDisputeMemo(summary, "Enterprise Accounts Payable")
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"dispute_%s.md\"", id))
	w.Write([]byte(memo))
}

func (s *Server) handleExportEvidencePDF(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/export/evidence/")
	summary := s.getOrCreateSummary(id)
	if summary == nil {
		http.Error(w, "dossier not found", http.StatusNotFound)
		return
	}

	s.mu.RLock()
	dossier := s.dossiers[id]
	s.mu.RUnlock()

	tmpPDF := filepath.Join(os.TempDir(), fmt.Sprintf("evidence_%s.pdf", id))
	if err := report.GenerateAuditEvidencePDF(summary, &dossier.Invoice, tmpPDF); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpPDF)

	data, err := os.ReadFile(tmpPDF)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=\"evidence_%s.pdf\"", id))
	w.Write(data)
}

func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "parse multipart form error", http.StatusBadRequest)
		return
	}

	tmpDir, err := os.MkdirTemp("", "reconcileai_upload_*")
	if err != nil {
		http.Error(w, "create temp dir failed", http.StatusInternalServerError)
		return
	}

	saveFile := func(field, defaultName string) (string, error) {
		f, handler, err := r.FormFile(field)
		if err != nil {
			return "", err
		}
		defer f.Close()

		filename := handler.Filename
		if filename == "" {
			filename = defaultName
		}
		destPath := filepath.Join(tmpDir, filename)
		out, err := os.Create(destPath)
		if err != nil {
			return "", err
		}
		defer out.Close()

		if _, err := io.Copy(out, f); err != nil {
			return "", err
		}
		return destPath, nil
	}

	poPath, err := saveFile("po", "po.pdf")
	if err != nil {
		http.Error(w, "po file missing: "+err.Error(), http.StatusBadRequest)
		return
	}

	grnPath, err := saveFile("grn", "grn.pdf")
	if err != nil {
		http.Error(w, "grn file missing: "+err.Error(), http.StatusBadRequest)
		return
	}

	invPath, err := saveFile("invoice", "invoice.pdf")
	if err != nil {
		http.Error(w, "invoice file missing: "+err.Error(), http.StatusBadRequest)
		return
	}

	poDoc, err := ingestion.ParsePDFDocument(poPath, config.DocTypePO)
	if err != nil {
		http.Error(w, "parse PO failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	grnDoc, err := ingestion.ParsePDFDocument(grnPath, config.DocTypeGRN)
	if err != nil {
		http.Error(w, "parse GRN failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	invDoc, err := ingestion.ParsePDFDocument(invPath, config.DocTypeInvoice)
	if err != nil {
		http.Error(w, "parse Invoice failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	uploadID := fmt.Sprintf("upload_%d", os.Getpid())
	dossier := &config.Dossier{
		ID:            uploadID,
		Title:         fmt.Sprintf("Uploaded Dossier (%s)", invDoc.Header.VendorName),
		PurchaseOrder: *poDoc,
		GoodsReceipts: []config.Document{*grnDoc},
		Invoice:       *invDoc,
	}

	s.RegisterDossier(dossier)
	s.SSE.Broadcast("DOSSIER_UPLOADED", dossier)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dossier)
}

func (s *Server) getOrCreateSummary(id string) *config.AuditSummary {
	s.mu.RLock()
	summary, found := s.summaries[id]
	dossier, dFound := s.dossiers[id]
	s.mu.RUnlock()

	if found {
		return summary
	}
	if !dFound {
		return nil
	}

	sum := auditor.ReconcileDossierWithLedger(dossier, s.Config, s.ledger)
	s.mu.Lock()
	s.summaries[id] = sum
	s.history = append(s.history, sum)
	s.mu.Unlock()
	return sum
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make([]*config.AuditSummary, len(s.history))
	for i, item := range s.history {
		res[len(s.history)-1-i] = item
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

// MetricsSummary contains aggregated AP audit performance figures.
type MetricsSummary struct {
	TotalDossiers            int            `json:"total_dossiers"`
	TotalAudited             int            `json:"total_audited"`
	ApprovedCount            int            `json:"approved_count"`
	FlaggedCount             int            `json:"flagged_count"`
	TotalInvoicedSpend       float64        `json:"total_invoiced_spend"`
	TotalAuthorizedSpend     float64        `json:"total_authorized_spend"`
	TotalPreventedOvercharge float64        `json:"total_prevented_overcharge"`
	EarlyDiscountPotential   float64        `json:"early_discount_potential"`
	DiscrepancyBreakdown     map[string]int `json:"discrepancy_breakdown"`
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics := MetricsSummary{
		TotalDossiers:        len(s.dossiers),
		TotalAudited:         len(s.summaries),
		DiscrepancyBreakdown: make(map[string]int),
	}

	for _, sum := range s.summaries {
		metrics.TotalInvoicedSpend += sum.TotalInvoiced
		metrics.TotalAuthorizedSpend += sum.TotalAuthorized
		metrics.TotalPreventedOvercharge += sum.PreventedOvercharge
		if sum.MatchStatus == "APPROVED" {
			metrics.ApprovedCount++
		} else {
			metrics.FlaggedCount++
		}
		if sum.EarlyDiscount != nil && sum.EarlyDiscount.IsEligible {
			metrics.EarlyDiscountPotential += sum.EarlyDiscount.DiscountAmount
		}
		for _, d := range sum.Discrepancies {
			metrics.DiscrepancyBreakdown[string(d.Type)]++
		}
	}

	metrics.TotalInvoicedSpend = math.Round(metrics.TotalInvoicedSpend*100) / 100
	metrics.TotalAuthorizedSpend = math.Round(metrics.TotalAuthorizedSpend*100) / 100
	metrics.TotalPreventedOvercharge = math.Round(metrics.TotalPreventedOvercharge*100) / 100
	metrics.EarlyDiscountPotential = math.Round(metrics.EarlyDiscountPotential*100) / 100

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// VendorScorecard summarizes historical billing compliance per supplier.
type VendorScorecard struct {
	VendorName           string  `json:"vendor_name"`
	TotalAuditedInvoices int     `json:"total_audited_invoices"`
	CleanInvoices        int     `json:"clean_invoices"`
	FlaggedInvoices      int     `json:"flagged_invoices"`
	TotalInvoiced        float64 `json:"total_invoiced"`
	TotalAuthorized      float64 `json:"total_authorized"`
	PreventedOvercharge  float64 `json:"prevented_overcharge"`
	AccuracyRate         float64 `json:"accuracy_rate"`
	RiskTier             string  `json:"risk_tier"`
}

func (s *Server) handleVendorScorecard(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	vendorMap := make(map[string]*VendorScorecard)

	for _, sum := range s.summaries {
		vName := sum.VendorName
		if vName == "" {
			vName = "Unknown Vendor"
		}
		sc, exists := vendorMap[vName]
		if !exists {
			sc = &VendorScorecard{VendorName: vName}
			vendorMap[vName] = sc
		}

		sc.TotalAuditedInvoices++
		sc.TotalInvoiced += sum.TotalInvoiced
		sc.TotalAuthorized += sum.TotalAuthorized
		sc.PreventedOvercharge += sum.PreventedOvercharge
		if sum.MatchStatus == "APPROVED" {
			sc.CleanInvoices++
		} else {
			sc.FlaggedInvoices++
		}
	}

	var scorecards []VendorScorecard
	for _, sc := range vendorMap {
		if sc.TotalAuditedInvoices > 0 {
			sc.AccuracyRate = math.Round((float64(sc.CleanInvoices)/float64(sc.TotalAuditedInvoices))*1000) / 10.0
		}
		if sc.AccuracyRate >= 95.0 {
			sc.RiskTier = "LOW"
		} else if sc.AccuracyRate >= 75.0 {
			sc.RiskTier = "MEDIUM"
		} else {
			sc.RiskTier = "HIGH"
		}
		sc.TotalInvoiced = math.Round(sc.TotalInvoiced*100) / 100
		sc.TotalAuthorized = math.Round(sc.TotalAuthorized*100) / 100
		sc.PreventedOvercharge = math.Round(sc.PreventedOvercharge*100) / 100
		scorecards = append(scorecards, *sc)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(scorecards)
}

// BatchRequest specifies target dossiers to reconcile in one transaction.
type BatchRequest struct {
	DossierIDs []string `json:"dossier_ids"`
}

// BatchResponse returns consolidated multi-dossier reconciliation results.
type BatchResponse struct {
	TotalProcessed      int                    `json:"total_processed"`
	ApprovedCount       int                    `json:"approved_count"`
	FlaggedCount        int                    `json:"flagged_count"`
	TotalInvoiced       float64                `json:"total_invoiced"`
	TotalAuthorized     float64                `json:"total_authorized"`
	PreventedOvercharge float64                `json:"prevented_overcharge"`
	Summaries           []*config.AuditSummary `json:"summaries"`
}

func (s *Server) handleBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req BatchRequest
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&req)
	}

	s.mu.RLock()
	var targets []*config.Dossier
	if len(req.DossierIDs) > 0 {
		for _, id := range req.DossierIDs {
			if d, ok := s.dossiers[id]; ok {
				targets = append(targets, d)
			}
		}
	} else {
		for _, d := range s.dossiers {
			targets = append(targets, d)
		}
	}
	s.mu.RUnlock()

	var summaries []*config.AuditSummary
	var totalInvoiced, totalAuth, prevented float64
	var approved, flagged int

	for _, d := range targets {
		sum := auditor.ReconcileDossierWithLedger(d, s.Config, s.ledger)
		summaries = append(summaries, sum)
		totalInvoiced += sum.TotalInvoiced
		totalAuth += sum.TotalAuthorized
		prevented += sum.PreventedOvercharge
		if sum.MatchStatus == "APPROVED" {
			approved++
		} else {
			flagged++
		}

		s.mu.Lock()
		s.summaries[d.ID] = sum
		s.history = append(s.history, sum)
		s.mu.Unlock()
	}

	resp := BatchResponse{
		TotalProcessed:      len(summaries),
		ApprovedCount:       approved,
		FlaggedCount:        flagged,
		TotalInvoiced:       math.Round(totalInvoiced*100) / 100,
		TotalAuthorized:     math.Round(totalAuth*100) / 100,
		PreventedOvercharge: math.Round(prevented*100) / 100,
		Summaries:           summaries,
	}

	s.SSE.Broadcast("BATCH_COMPLETE", resp)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleExportCSV(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/export/csv/")
	summary := s.getOrCreateSummary(id)
	if summary == nil {
		http.Error(w, "dossier not found", http.StatusNotFound)
		return
	}

	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	writer.Write([]string{
		"DossierID", "Vendor", "InvoiceNumber", "PONumber", "SKU", "Description",
		"InvoicedQty", "ReceivedQty", "OrderedQty", "InvoicedPrice", "POPrice",
		"PriceVariance", "DiscrepancyStatus", "MatchConfidence", "MatchReason",
	})

	for _, p := range summary.MatchedPairs {
		status := "MATCH"
		if p.HasDiscrepancy {
			status = "DISCREPANCY"
		}
		writer.Write([]string{
			summary.DossierID,
			summary.VendorName,
			summary.InvoiceNumber,
			summary.PONumber,
			p.ItemSKU,
			p.Description,
			fmt.Sprintf("%.2f", p.InvoicedQty),
			fmt.Sprintf("%.2f", p.ReceivedQty),
			fmt.Sprintf("%.2f", p.OrderedQty),
			fmt.Sprintf("%.2f", p.InvoicedPrice),
			fmt.Sprintf("%.2f", p.POPrice),
			fmt.Sprintf("%.2f", p.PriceVariance),
			status,
			fmt.Sprintf("%.2f", p.Confidence),
			p.MatchReason,
		})
	}
	writer.Flush()

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"diff_%s.csv\"", id))
	w.Write(buf.Bytes())
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
