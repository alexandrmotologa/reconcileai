package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/alexandrmotologa/reconcileai/internal/auditor"
	"github.com/alexandrmotologa/reconcileai/internal/config"
	"github.com/alexandrmotologa/reconcileai/internal/erp"
	"github.com/alexandrmotologa/reconcileai/internal/ingestion"
	"github.com/alexandrmotologa/reconcileai/internal/report"
	"github.com/alexandrmotologa/reconcileai/internal/server"
	"github.com/alexandrmotologa/reconcileai/internal/synth"
)

const version = "1.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	command := os.Args[1]

	switch command {
	case "serve":
		cmdServe(os.Args[2:])
	case "audit", "match":
		cmdAudit(os.Args[2:])
	case "batch":
		cmdBatch(os.Args[2:])
	case "watch":
		cmdWatch(os.Args[2:])
	case "scorecard":
		cmdScorecard(os.Args[2:])
	case "export":
		cmdExport(os.Args[2:])
	case "generate-dossier":
		cmdGenerateDossiers(os.Args[2:])
	case "version":
		printVersion()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command '%s'\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf("reconcileai v%s - Autonomous 3-Way Document Matching & AP Discrepancy Auditor\n\n", version)
	fmt.Println("Usage:")
	fmt.Println("  reconcileai <command> [flags]")
	fmt.Println("\nCommands:")
	fmt.Println("  serve             Start HTTP API and embedded React 19 Discrepancy Cockpit")
	fmt.Println("  audit             Run 3-way matching and discrepancy audit on a single dossier directory")
	fmt.Println("  batch             Run reconciliation across multiple dossier directories with consolidated reporting")
	fmt.Println("  watch             Monitor an inbox directory for new dossiers and auto-process them")
	fmt.Println("  scorecard         Analyze vendor billing compliance rates and generate risk scorecards")
	fmt.Println("  export            Generate ERP settlement payload (SAP IDoc, NetSuite, QuickBooks) or dispute memo")
	fmt.Println("  generate-dossier  Synthesize realistic reference PDF dossiers with controlled anomalies")
	fmt.Println("  version           Display engine version and memory footprint stats")
	fmt.Println("\nRun 'reconcileai <command> --help' for details on each subcommand.")
}

func printVersion() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	allocMB := float64(m.Alloc) / 1024 / 1024

	fmt.Printf("reconcileai version %s\n", version)
	fmt.Printf("Go runtime: %s (%s/%s)\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Current memory allocation: %.2f MB (Target: < 25 MB)\n", allocMB)
}

func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	host := fs.String("host", "127.0.0.1", "Host interface to bind")
	port := fs.Int("port", 7272, "HTTP port to listen on")
	dossiersDir := fs.String("dossiers", "./examples/dossiers", "Path to dossiers directory")
	configPath := fs.String("config", "", "Path to optional config JSON")
	fs.Parse(args)

	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Printf("Warning: failed to load config %s, using defaults: %v", *configPath, err)
		cfg = config.DefaultConfig()
	}

	cfg.Server.Host = *host
	cfg.Server.Port = *port

	srv := server.NewServer(cfg)

	// Ensure reference dossiers exist or synthesize them
	if _, err := os.Stat(*dossiersDir); os.IsNotExist(err) {
		fmt.Println("Synthesizing reference test dossiers in", *dossiersDir, "...")
		if _, err := synth.GenerateAllReferenceDossiers(*dossiersDir); err != nil {
			log.Fatalf("Failed to generate test dossiers: %v", err)
		}
	}

	// Load dossiers from directory
	entries, err := os.ReadDir(*dossiersDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				dossierPath := filepath.Join(*dossiersDir, e.Name())
				dossier, err := ingestion.LoadDossierFromDirectory(dossierPath)
				if err == nil {
					srv.RegisterDossier(dossier)
					fmt.Printf("  Loaded dossier: %s (%s)\n", dossier.ID, dossier.Invoice.Header.VendorName)
				}
			}
		}
	}

	addr := fmt.Sprintf("%s:%d", *host, *port)
	fmt.Printf("\n=== reconcileai Cockpit Server running at http://%s ===\n", addr)
	fmt.Println("Press Ctrl+C to terminate.")

	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func cmdAudit(args []string) {
	fs := flag.NewFlagSet("audit", flag.ExitOnError)
	dir := fs.String("dir", "", "Path to dossier directory containing PO, GRN, and Invoice")
	format := fs.String("format", "table", "Output format: table, json, or memo")
	fs.Parse(args)

	if *dir == "" {
		fmt.Fprintln(os.Stderr, "Error: --dir is required")
		fs.Usage()
		os.Exit(1)
	}

	dossier, err := ingestion.LoadDossierFromDirectory(*dir)
	if err != nil {
		log.Fatalf("Failed to load dossier from %s: %v", *dir, err)
	}

	cfg := config.DefaultConfig()
	summary := auditor.ReconcileDossier(dossier, cfg)

	switch strings.ToLower(*format) {
	case "json":
		data, _ := json.MarshalIndent(summary, "", "  ")
		fmt.Println(string(data))

	case "memo":
		memo := report.GenerateVendorDisputeMemo(summary, "Enterprise Accounts Payable")
		fmt.Println(memo)

	default:
		// Terminal Table
		fmt.Println("================================================================================")
		fmt.Printf("RECONCILEAI AUDIT REPORT: %s\n", summary.DossierID)
		fmt.Printf("Vendor: %s | PO: %s | Invoice: %s\n", summary.VendorName, summary.PONumber, summary.InvoiceNumber)
		fmt.Printf("Status: %s | Exec Time: %dms | Hash: %s\n", summary.MatchStatus, summary.ExecutionTimeMs, summary.AuditHash[:16])
		fmt.Println("--------------------------------------------------------------------------------")
		fmt.Printf("Total Invoiced:       %s %.2f\n", summary.Currency, summary.TotalInvoiced)
		fmt.Printf("Authorized Amount:    %s %.2f\n", summary.Currency, summary.TotalAuthorized)
		fmt.Printf("Prevented Overcharge: %s %.2f\n", summary.Currency, summary.PreventedOvercharge)
		fmt.Println("================================================================================")
		fmt.Println("\nLINE RECONCILIATION SUMMARY:")
		fmt.Printf("%-14s %-24s %-8s %-8s %-10s %-10s %s\n", "SKU", "Description", "InvQty", "RecQty", "InvPrice", "POPrice", "Status")
		fmt.Println(strings.Repeat("-", 80))
		for _, p := range summary.MatchedPairs {
			status := "MATCH"
			if p.HasDiscrepancy {
				status = "DISCREPANCY"
			}
			desc := p.Description
			if len(desc) > 23 {
				desc = desc[:20] + "..."
			}
			fmt.Printf("%-14s %-24s %-8.1f %-8.1f $%-9.2f $%-9.2f %s\n",
				p.ItemSKU, desc, p.InvoicedQty, p.ReceivedQty, p.InvoicedPrice, p.POPrice, status)
		}

		if len(summary.Discrepancies) > 0 {
			fmt.Println("\nDETECTED DISCREPANCIES:")
			for i, d := range summary.Discrepancies {
				fmt.Printf("  %d. [%s] %s\n     %s (Impact: +$%.2f)\n", i+1, d.Type, d.Title, d.Description, d.VarianceAmount)
			}
		}

		if summary.EarlyDiscount != nil {
			fmt.Printf("\nEARLY PAYMENT DISCOUNT TERMS: %s\n", summary.EarlyDiscount.Terms)
			fmt.Printf("  Potential Savings: $%.2f (Deadline: %s, Annualized ROI: %.2f%%)\n",
				summary.EarlyDiscount.DiscountAmount, summary.EarlyDiscount.DiscountDeadline, summary.EarlyDiscount.AnnualizedROI)
		}
		fmt.Println()
	}
}

func cmdBatch(args []string) {
	fs := flag.NewFlagSet("batch", flag.ExitOnError)
	dir := fs.String("dir", "./examples/dossiers", "Path to base directory containing dossier subdirectories")
	format := fs.String("format", "table", "Output format: table, json, or csv")
	outDir := fs.String("out", "", "Optional directory to write JSON audit reports per dossier")
	fs.Parse(args)

	entries, err := os.ReadDir(*dir)
	if err != nil {
		log.Fatalf("Failed to read dossiers directory %s: %v", *dir, err)
	}

	cfg := config.DefaultConfig()
	ledger := auditor.NewDuplicateLedger()

	var summaries []*config.AuditSummary
	var totalInvoiced, totalAuthorized, totalPrevented float64
	var approvedCount, flaggedCount int

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dossierPath := filepath.Join(*dir, e.Name())
		dossier, err := ingestion.LoadDossierFromDirectory(dossierPath)
		if err != nil {
			continue
		}

		summary := auditor.ReconcileDossierWithLedger(dossier, cfg, ledger)
		summaries = append(summaries, summary)
		totalInvoiced += summary.TotalInvoiced
		totalAuthorized += summary.TotalAuthorized
		totalPrevented += summary.PreventedOvercharge

		if summary.MatchStatus == "APPROVED" {
			approvedCount++
		} else {
			flaggedCount++
		}

		if *outDir != "" {
			_ = os.MkdirAll(*outDir, 0755)
			data, _ := json.MarshalIndent(summary, "", "  ")
			_ = os.WriteFile(filepath.Join(*outDir, fmt.Sprintf("audit_%s.json", summary.DossierID)), data, 0644)
		}
	}

	if *outDir != "" {
		batchSummary := map[string]interface{}{
			"total_dossiers":       len(summaries),
			"approved_count":       approvedCount,
			"flagged_count":        flaggedCount,
			"total_invoiced":       totalInvoiced,
			"total_authorized":     totalAuthorized,
			"prevented_overcharge": totalPrevented,
		}
		data, _ := json.MarshalIndent(batchSummary, "", "  ")
		_ = os.WriteFile(filepath.Join(*outDir, "batch_summary.json"), data, 0644)
	}

	switch strings.ToLower(*format) {
	case "json":
		data, _ := json.MarshalIndent(summaries, "", "  ")
		fmt.Println(string(data))

	case "csv":
		writer := csv.NewWriter(os.Stdout)
		writer.Write([]string{"DossierID", "Vendor", "InvoiceNumber", "PONumber", "Status", "TotalInvoiced", "TotalAuthorized", "PreventedOvercharge", "DiscrepancyCount"})
		for _, s := range summaries {
			writer.Write([]string{
				s.DossierID, s.VendorName, s.InvoiceNumber, s.PONumber, s.MatchStatus,
				fmt.Sprintf("%.2f", s.TotalInvoiced),
				fmt.Sprintf("%.2f", s.TotalAuthorized),
				fmt.Sprintf("%.2f", s.PreventedOvercharge),
				fmt.Sprintf("%d", len(s.Discrepancies)),
			})
		}
		writer.Flush()

	default:
		fmt.Println("=========================================================================================")
		fmt.Println("RECONCILEAI BATCH AUDIT REGISTER")
		fmt.Println("=========================================================================================")
		fmt.Printf("%-32s %-22s %-12s %-12s %-12s %s\n", "Dossier ID", "Vendor", "Invoiced", "Authorized", "Overcharge", "Status")
		fmt.Println(strings.Repeat("-", 105))
		for _, s := range summaries {
			vendor := s.VendorName
			if len(vendor) > 20 {
				vendor = vendor[:18] + ".."
			}
			fmt.Printf("%-32s %-22s $%-11.2f $%-11.2f $%-11.2f %s\n",
				s.DossierID, vendor, s.TotalInvoiced, s.TotalAuthorized, s.PreventedOvercharge, s.MatchStatus)
		}
		fmt.Println(strings.Repeat("=", 105))
		fmt.Printf("BATCH TOTALS: %d Invoices (%d Approved, %d Flagged)\n", len(summaries), approvedCount, flaggedCount)
		fmt.Printf("Total Invoiced Spend:    $%.2f\n", totalInvoiced)
		fmt.Printf("Total Authorized Amount: $%.2f\n", totalAuthorized)
		fmt.Printf("Total Prevented Losses:  $%.2f\n", totalPrevented)
		fmt.Println("=========================================================================================")
	}
}

func cmdWatch(args []string) {
	fs := flag.NewFlagSet("watch", flag.ExitOnError)
	dir := fs.String("dir", "./inbox", "Directory to watch for incoming dossier folders")
	pollSec := fs.Int("poll", 3, "Polling interval in seconds")
	fs.Parse(args)

	if err := os.MkdirAll(*dir, 0755); err != nil {
		log.Fatalf("Failed to create watch directory %s: %v", *dir, err)
	}

	fmt.Printf("Watching %s for incoming accounting dossiers (polling every %ds)...\n", *dir, *pollSec)
	fmt.Println("Press Ctrl+C to stop.")

	cfg := config.DefaultConfig()
	ledger := auditor.NewDuplicateLedger()
	processed := make(map[string]bool)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(time.Duration(*pollSec) * time.Second)
	defer ticker.Stop()

	scan := func() {
		entries, err := os.ReadDir(*dir)
		if err != nil {
			return
		}

		for _, e := range entries {
			if !e.IsDir() || processed[e.Name()] {
				continue
			}

			dossierPath := filepath.Join(*dir, e.Name())
			dossier, err := ingestion.LoadDossierFromDirectory(dossierPath)
			if err != nil {
				continue
			}

			processed[e.Name()] = true
			summary := auditor.ReconcileDossierWithLedger(dossier, cfg, ledger)

			fmt.Printf("\n[DETECTED NEW DOSSIER] %s\n", e.Name())
			fmt.Printf("  Vendor: %s | PO: %s | Invoice: %s\n", summary.VendorName, summary.PONumber, summary.InvoiceNumber)
			fmt.Printf("  Result: %s | Prevented Overcharge: %s %.2f\n", summary.MatchStatus, summary.Currency, summary.PreventedOvercharge)
			if len(summary.Discrepancies) > 0 {
				for i, d := range summary.Discrepancies {
					fmt.Printf("    * Discrepancy %d: [%s] %s (+$%.2f)\n", i+1, d.Type, d.Title, d.VarianceAmount)
				}
			}
		}
	}

	// Initial scan
	scan()

	for {
		select {
		case <-sigChan:
			fmt.Println("\nStopping dossier watch mode.")
			return
		case <-ticker.C:
			scan()
		}
	}
}

func cmdScorecard(args []string) {
	fs := flag.NewFlagSet("scorecard", flag.ExitOnError)
	dir := fs.String("dir", "./examples/dossiers", "Directory containing dossiers to analyze")
	format := fs.String("format", "table", "Output format: table or json")
	fs.Parse(args)

	entries, err := os.ReadDir(*dir)
	if err != nil {
		log.Fatalf("Failed to read directory %s: %v", *dir, err)
	}

	cfg := config.DefaultConfig()
	ledger := auditor.NewDuplicateLedger()

	vendorMap := make(map[string]*server.VendorScorecard)

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dossierPath := filepath.Join(*dir, e.Name())
		dossier, err := ingestion.LoadDossierFromDirectory(dossierPath)
		if err != nil {
			continue
		}

		sum := auditor.ReconcileDossierWithLedger(dossier, cfg, ledger)
		vName := sum.VendorName
		if vName == "" {
			vName = "Unknown Vendor"
		}
		sc, exists := vendorMap[vName]
		if !exists {
			sc = &server.VendorScorecard{VendorName: vName}
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

	var scorecards []server.VendorScorecard
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

	if strings.ToLower(*format) == "json" {
		data, _ := json.MarshalIndent(scorecards, "", "  ")
		fmt.Println(string(data))
		return
	}

	fmt.Println("=========================================================================================")
	fmt.Println("RECONCILEAI VENDOR COMPLIANCE & RISK SCORECARD")
	fmt.Println("=========================================================================================")
	fmt.Printf("%-32s %-8s %-8s %-12s %-14s %-12s %s\n", "Vendor Name", "Audited", "Clean", "Accuracy", "Invoiced", "Overcharge", "Risk Tier")
	fmt.Println(strings.Repeat("-", 100))
	for _, sc := range scorecards {
		vName := sc.VendorName
		if len(vName) > 30 {
			vName = vName[:28] + ".."
		}
		fmt.Printf("%-32s %-8d %-8d %-11.1f%% $%-13.2f $%-11.2f %s\n",
			vName, sc.TotalAuditedInvoices, sc.CleanInvoices, sc.AccuracyRate, sc.TotalInvoiced, sc.PreventedOvercharge, sc.RiskTier)
	}
	fmt.Println("=========================================================================================")
}

func cmdExport(args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	dir := fs.String("dir", "", "Path to dossier directory")
	target := fs.String("target", "sap", "Export target: sap, netsuite, quickbooks, dispute, pdf")
	outPath := fs.String("out", "", "Output file destination path (default stdout)")
	fs.Parse(args)

	if *dir == "" {
		fmt.Fprintln(os.Stderr, "Error: --dir is required")
		fs.Usage()
		os.Exit(1)
	}

	dossier, err := ingestion.LoadDossierFromDirectory(*dir)
	if err != nil {
		log.Fatalf("Failed to load dossier from %s: %v", *dir, err)
	}

	cfg := config.DefaultConfig()
	summary := auditor.ReconcileDossier(dossier, cfg)

	var outputBytes []byte
	switch strings.ToLower(*target) {
	case "sap", "idoc":
		outputBytes, err = erp.GenerateSAPIDocXML(summary, "1000")
	case "netsuite", "csv":
		outputBytes, err = erp.GenerateNetSuiteCSV(summary)
	case "quickbooks", "qb", "json":
		outputBytes, err = erp.GenerateQuickBooksJSON(summary)
	case "dispute", "memo", "md":
		outputBytes = []byte(report.GenerateVendorDisputeMemo(summary, "Enterprise Accounts Payable"))
	case "pdf", "evidence":
		if *outPath == "" {
			*outPath = fmt.Sprintf("evidence_%s.pdf", summary.DossierID)
		}
		err = report.GenerateAuditEvidencePDF(summary, &dossier.Invoice, *outPath)
		if err == nil {
			fmt.Printf("Exported audit evidence PDF to: %s\n", *outPath)
			return
		}
	default:
		log.Fatalf("Unknown export target '%s'. Choose sap, netsuite, quickbooks, dispute, or pdf", *target)
	}

	if err != nil {
		log.Fatalf("Export error: %v", err)
	}

	if *outPath != "" {
		if err := os.WriteFile(*outPath, outputBytes, 0644); err != nil {
			log.Fatalf("Failed to write to %s: %v", *outPath, err)
		}
		fmt.Printf("Successfully exported to: %s\n", *outPath)
	} else {
		fmt.Println(string(outputBytes))
	}
}

func cmdGenerateDossiers(args []string) {
	fs := flag.NewFlagSet("generate-dossier", flag.ExitOnError)
	outDir := fs.String("dir", "./examples/dossiers", "Directory to write synthesized dossiers")
	fs.Parse(args)

	fmt.Printf("Synthesizing 5 reference accounting dossiers in %s ...\n", *outDir)
	created, err := synth.GenerateAllReferenceDossiers(*outDir)
	if err != nil {
		log.Fatalf("Error synthesizing dossiers: %v", err)
	}

	for _, c := range created {
		fmt.Printf("  Created: %s\n", c)
	}
	fmt.Println("Dossier generation complete.")
}
