package server

import (
	"encoding/json"
	"fmt"
	"io"
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
	mu        sync.RWMutex
	dossiers  map[string]*config.Dossier
	summaries map[string]*config.AuditSummary
}

// NewServer creates a new API and Cockpit server instance.
func NewServer(cfg config.AppConfig) *Server {
	return &Server{
		Config:    cfg,
		SSE:       NewSSEHub(),
		dossiers:  make(map[string]*config.Dossier),
		summaries: make(map[string]*config.AuditSummary),
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
	mux.HandleFunc("/api/export/sap/", s.handleExportSAP)
	mux.HandleFunc("/api/export/netsuite/", s.handleExportNetSuite)
	mux.HandleFunc("/api/export/quickbooks/", s.handleExportQuickBooks)
	mux.HandleFunc("/api/export/dispute/", s.handleExportDispute)
	mux.HandleFunc("/api/export/evidence/", s.handleExportEvidencePDF)
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

	summary := auditor.ReconcileDossier(dossier, s.Config)

	s.mu.Lock()
	s.summaries[id] = summary
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

	sum := auditor.ReconcileDossier(dossier, s.Config)
	s.mu.Lock()
	s.summaries[id] = sum
	s.mu.Unlock()
	return sum
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
