package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

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
	fmt.Println("  audit             Run 3-way matching and discrepancy audit on a dossier directory")
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
		fmt.Println()
	}
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
