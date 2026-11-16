# CLI Usage Guide

The `reconcileai` binary operates as a standalone tool or as a local cockpit server.

## Installation and Build

Prerequisites:
- Go 1.22 or newer
- Node.js 18 or newer (only needed when rebuilding the frontend)

Build the binary:
```bash
# Build frontend
cd web
npm install
npm run build
cd ..

# Build Go executable
go build -o bin/reconcileai.exe ./cmd/reconcileai
```

## Subcommands

### 1. `serve`
Starts the HTTP server and embeds the React 19 cockpit.

```bash
# Start server on default port 7272
./bin/reconcileai.exe serve

# Bind to custom interface and port
./bin/reconcileai.exe serve --host 0.0.0.0 --port 8080 --dossiers ./examples/dossiers
```

Flags:
- `--host`: network interface to bind (default: `127.0.0.1`).
- `--port`: port number (default: `7272`).
- `--dossiers`: folder path containing transaction dossiers (default: `./examples/dossiers`).
- `--config`: optional path to configuration JSON file.

### 2. `audit` (or `match`)
Runs 3-way matching and mathematical verification on a specific dossier directory.

```bash
# Print formatted terminal summary table
./bin/reconcileai.exe audit --dir ./examples/dossiers/dossier_02_qty_overbilling

# Output structured JSON audit summary
./bin/reconcileai.exe audit --dir ./examples/dossiers/dossier_02_qty_overbilling --format json

# Output vendor dispute notice memo in Markdown
./bin/reconcileai.exe audit --dir ./examples/dossiers/dossier_02_qty_overbilling --format memo
```

Flags:
- `--dir`: path to folder holding PO, GRN, and Invoice documents.
- `--format`: output format, choices: `table` (default), `json`, or `memo`.

### 3. `export`
Generates ERP settlement payloads or dispute artifacts from an audited dossier.

```bash
# Generate SAP S/4HANA IDoc XML
./bin/reconcileai.exe export --dir ./examples/dossiers/dossier_01_clean_match --target sap --out ./exports/sap_bill.xml

# Generate NetSuite Bill CSV
./bin/reconcileai.exe export --dir ./examples/dossiers/dossier_01_clean_match --target netsuite --out ./exports/netsuite_bill.csv

# Generate QuickBooks Online JSON
./bin/reconcileai.exe export --dir ./examples/dossiers/dossier_01_clean_match --target quickbooks --out ./exports/qb_bill.json

# Generate Vendor Dispute Memo in Markdown
./bin/reconcileai.exe export --dir ./examples/dossiers/dossier_03_price_variance --target dispute --out ./exports/dispute_memo.md

# Generate Audit Evidence PDF with red bounding box overlays
./bin/reconcileai.exe export --dir ./examples/dossiers/dossier_02_qty_overbilling --target pdf --out ./exports/audit_evidence.pdf
```

Flags:
- `--dir`: path to folder holding PO, GRN, and Invoice documents.
- `--target`: export format, choices: `sap`, `netsuite`, `quickbooks`, `dispute`, `pdf`.
- `--out`: destination path for the exported file (defaults to stdout for text formats).

### 4. `generate-dossier`
Synthesizes reference tripartite PDF dossiers with realistic accounting anomalies.

```bash
./bin/reconcileai.exe generate-dossier --dir ./examples/dossiers
```

Creates five reference scenarios:
1. `dossier_01_clean_match`: full match, zero discrepancies.
2. `dossier_02_qty_overbilling`: warehouse short delivery billed as full.
3. `dossier_03_price_variance`: unit price billed above agreed purchase order rate.
4. `dossier_04_unauthorized_fee`: unexpected freight surcharge on FOB Destination contract.
5. `dossier_05_uom_split_delivery`: split warehouse intake with dozen-to-each packaging conversion.

### 5. `version`
Displays current version, compiler details, and runtime heap allocation.

```bash
./bin/reconcileai.exe version
```
