# reconcileai Architecture and Implementation Plan
An autonomous 3-way document matching and discrepancy auditor for enterprise accounts payable.
Single-binary Go engine with embedded React 19 cockpit, visual PDF bounding box annotation, and ERP settlement export.

---

## 1. Problem Overview

Accounts payable teams spend hours manually comparing tripartite procurement documents:
1. **Purchase Order (PO)**: Internal contract specifying authorized items, quantities, agreed unit prices, and payment terms.
2. **Goods Receipt Note (GRN)**: Warehouse intake record specifying physical quantities received, damaged goods, and receipt timestamps.
3. **Supplier Invoice (INV)**: Vendor payment demand with applied unit prices, taxes, freight fees, and payment instructions.

Manual review across non-standard vendor layouts causes three main operational issues:
- **Silent overbilling**: Minor unit price increases, billing for partial delivery as full delivery, or adding unauthorized freight and packaging surcharges.
- **Divergent naming**: Suppliers rarely use internal customer part numbers. String lookups fail when PO lists `SKU-8820-BRG` and Invoice lists `Heavy Duty Flange Bearing 25mm`.
- **Audit lag**: Delayed approval cycles sacrifice vendor discounts (for example 2/10 Net 30) and tie up back-office resources.

`reconcileai` provides an automated, local Go engine (< 25MB RAM footprint) coupled with an embedded React 19 cockpit to parse, align, audit, annotate, and dispatch settlement files.

---

## 2. Invariant Auditing Rules

```
+-----------------------------------+-----------------------------------------------------------------+
| Verification Rule                 | Mathematical Formulation and Invariant                          |
+-----------------------------------+-----------------------------------------------------------------+
| 1. Quantity Invariant             | Q_inv <= sum(Q_rec) <= Q_po                                     |
|                                   | Invoiced quantity must not exceed total verified receipt.       |
| 2. Price Tolerance Bound          | |P_inv - P_po| <= max(epsilon_abs, epsilon_rel * P_po)          |
|                                   | Billed unit price must match PO price within tolerance.         |
| 3. Line Item Mathematical Sum     | sum(Q_i * P_i) + Tax + Freight - Discount = Total_inv           |
|                                   | Internal invoice arithmetic must be mathematically consistent.  |
| 4. Surcharge Authorization        | forall s in Surcharges_inv: s in PO_authorized_terms            |
|                                   | Unauthorized freight, fuel, or handling fees trigger flags.     |
| 5. Multi-GRN Aggregation          | sum_{k=1}^m Q_{rec,k} >= Q_inv                                  |
|                                   | Split warehouse receipts are aggregated across multiple GRNs.   |
| 6. Unit of Measure Normalization  | Q_norm = Q * Factor(UOM, TargetUOM)                             |
|                                   | Cases, dozens, gross, kilograms, and pounds normalize cleanly.  |
| 7. Multi-Currency Triangulation   | |P_inv * FX_rate - P_po| <= FX_tolerance                        |
|                                   | Audits cross-currency pricing against daily benchmark rates.    |
| 8. Duplicate Invoicing Check      | Hash(VendorID, InvoiceNumber) not in SettledLedger              |
|                                   | Protects against duplicate invoice submissions.                 |
+-----------------------------------+-----------------------------------------------------------------+
```

---

## 3. System Architecture and Component Layout

```
                                  +------------------------------------+
                                  |     Document Ingestion Layer       |
                                  |  - PDF Vector Text & Layout Parser |
                                  |  - Scanned OCR Fallback Engine     |
                                  |  - Structured JSON/EDI Dossiers    |
                                  +------------------------------------+
                                                     |
                                                     v
                                  +------------------------------------+
                                  |      Semantic Alignment Layer      |
                                  |  - Token & Jaro-Winkler Similarity |
                                  |  - Unit of Measure (UOM) Converter |
                                  |  - Vendor Entity Normalizer        |
                                  |  - Bipartite Minimum-Cost Matcher  |
                                  +------------------------------------+
                                                     |
                                                     v
                                  +------------------------------------+
                                  |     Mathematical Audit Engine      |
                                  |  - Quantity & Split-GRN Auditor    |
                                  |  - Price Variance Bound Auditor    |
                                  |  - Tax & Surcharge Validator       |
                                  |  - Invoice Face Arithmetic Check   |
                                  +------------------------------------+
                                                     |
                         +---------------------------+---------------------------+
                         |                                                       |
                         v                                                       v
        +---------------------------------+                     +---------------------------------+
        |     Zero Discrepancy Path       |                     |    Discrepancy Detected Path    |
        |  - Certified Settlement Pack    |                     |  - Red PDF Bounding Annotator   |
        |  - SAP S/4HANA IDoc XML Export  |                     |  - Itemized Vendor Dispute Memo |
        |  - NetSuite CSV Bill Export     |                     |  - React 19 Discrepancy Cockpit |
        |  - QuickBooks Online JSON Bill  |                     |  - Human-in-the-loop Resolution |
        +---------------------------------+                     +---------------------------------+
```

---

## 4. Repository Structure

```
reconcileai/
├── cmd/
│   └── reconcileai/
│       └── main.go                 # CLI entry point: match, audit, export, serve, generate-dossier
├── internal/
│   ├── config/
│   │   ├── config.go               # YAML configuration, tolerance bounds, currency settings
│   │   └── schema.go               # Domain types: PO, GRN, Invoice, LineItem, Discrepancy
│   ├── ingestion/
│   │   ├── pdf_parser.go           # PDF layout extractor with text coordinates
│   │   ├── table_extractor.go      # Line-item table detection and column boundary resolution
│   │   ├── uom.go                  # Unit of measure conversion table (EA, DZ, CS, KG, LBS, etc.)
│   │   └── dossier_loader.go       # Multi-document dossier assembler and validator
│   ├── matching/
│   │   ├── similarity.go           # Jaro-Winkler, token overlap, and Levenshtein algorithms
│   │   ├── entity_aligner.go       # Vendor name, Tax ID, and address normalizer
│   │   └── bipartite_matcher.go    # Kuhn-Munkres/greedy bipartite line-item matcher
│   ├── auditor/
│   │   ├── engine.go               # Master 3-way matching rule coordinator
│   │   ├── quantity.go             # Warehouse intake vs billed quantity auditor
│   │   ├── price.go                # Unit price and contract tolerance auditor
│   │   ├── tax_surcharge.go        # Tax rate calculation and unauthorized fee detector
│   │   ├── fx_triangulation.go     # Multi-currency exchange rate auditor
│   │   └── arithmetic.go           # Line-item sum and invoice face arithmetic checker
│   ├── report/
│   │   ├── memo.go                 # Vendor dispute notice memo generator (Markdown/Text)
│   │   └── pdf_annotator.go        # Pure Go PDF annotator drawing red bounding boxes and badges
│   ├── erp/
│   │   ├── sap_idoc.go             # SAP S/4HANA INVOIC02 XML generator
│   │   ├── netsuite.go             # NetSuite Bill import CSV formatter
│   │   └── quickbooks.go           # QuickBooks Online Bill JSON payload builder
│   ├── synth/
│   │   └── generator.go            # Synthetic PDF dossier generator with controlled anomalies
│   └── server/
│       ├── api.go                  # REST API for dossier upload, audit execution, and export
│       ├── embed.go                # Embedded React 19 assets via go:embed
│       └── sse.go                  # Server-Sent Events for real-time audit streaming
├── web/
│   ├── src/
│   │   ├── App.tsx                 # React 19 root application
│   │   ├── components/
│   │   │   ├── Header.tsx          # Navigation, status indicators, and dossier selector
│   │   │   ├── ThreeWayDiff.tsx    # Synchronized 3-way line item comparison table
│   │   │   ├── DocumentViewer.tsx  # Interactive visual document viewer with bounding overlays
│   │   │   ├── DiscrepancyList.tsx # Detailed cards for each flagged anomaly
│   │   │   ├── ResolutionModal.tsx # Approval, dispute, and note attachment actions
│   │   │   ├── SavingsSummary.tsx  # Metrics on overbilling prevented
│   │   │   └── ErpExportDrawer.tsx # Live preview and download of SAP IDoc, NetSuite, QuickBooks
│   │   ├── types/
│   │   │   └── index.ts            # Frontend TypeScript definitions matching Go schema
│   │   └── main.tsx                # Application mount point
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   └── tailwind.config.js
├── examples/
│   ├── reconcileai.yaml            # Standard enterprise AP configuration
│   └── dossiers/                   # Reference test dossiers
│       ├── dossier_01_clean_match/
│       ├── dossier_02_qty_overbilling/
│       ├── dossier_03_price_variance/
│       ├── dossier_04_unauthorized_fee/
│       └── dossier_05_uom_mismatch/
├── docs/
│   ├── ARCHITECTURE.md             # Detailed engineering specifications
│   ├── MATCHING_ALGORITHMS.md      # Semantic alignment and bipartite matching reference
│   ├── ERP_INTEGRATIONS.md         # SAP, NetSuite, and QuickBooks schemas and mappings
│   └── CLI_USAGE.md                # Command line flags and automation recipes
├── Makefile
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

---

## 5. Implementation Steps

### Step 1: Core Configuration and Domain Models
- Define `internal/config` and `internal/config/schema.go` with strict data structures.
- Support YAML configuration for tolerance bounds (relative percent, absolute dollar threshold), enabled ERP targets, and UOM mapping tables.

### Step 2: Document Ingestion and PDF Parsing
- Implement PDF layout and text chunk extractor in pure Go.
- Implement structured table detection identifying column boundaries (SKU, Description, Qty, Unit Price, Total).
- Support Unit of Measure (UOM) normalization in `internal/ingestion/uom.go`.

### Step 3: Semantic Alignment and Bipartite Matching
- Build string distance algorithms in `internal/matching/similarity.go` (Jaro-Winkler, Levenshtein, token overlap).
- Implement bipartite matching to find optimal global pairings between PO lines and Invoice lines even with disparate wording.

### Step 4: Mathematical 3-Way Audit Engine
- Implement quantity checks against single or aggregated Goods Receipt Notes.
- Implement unit price variance auditing against agreed purchase order terms.
- Implement line arithmetic and unauthorized surcharge detection.
- Implement foreign exchange tolerance triangulation.

### Step 5: Visual PDF Annotator and Dispute Memo Generator
- Build pure Go PDF stream annotator adding red rectangular bounding boxes and numbered discrepancy markers.
- Implement automated vendor dispute notice generator detailing line variances and contract violations.

### Step 6: ERP Settlement Dispatchers
- Build SAP S/4HANA INVOIC02 XML builder.
- Build NetSuite Bill CSV exporter.
- Build QuickBooks Online Bill JSON exporter.

### Step 7: Synthetic Dossier Generator
- Build `internal/synth/generator.go` producing genuine tripartite test PDF sets:
  - Clean match (no variances).
  - Quantity discrepancy (warehouse short delivery billed as full).
  - Price creep (supplier bumped unit price beyond tolerance).
  - Unauthorized surcharge (freight fee tacked on contrary to Free on Board terms).
  - Unit of measure confusion (case vs each).

### Step 8: React 19 Discrepancy Cockpit
- Build modern responsive cockpit with Vite, React 19, and Tailwind CSS.
- Implement 3-way synchronized comparison grid.
- Implement interactive visual PDF overlay renderer.
- Implement dispute action workflows and ERP export viewer.
- Embed frontend build into Go binary (`//go:embed`).

### Step 9: HTTP REST Server, SSE and CLI Commands
- CLI commands: `reconcileai match`, `reconcileai audit`, `reconcileai export`, `reconcileai serve`, `reconcileai generate-dossier`.
- HTTP REST API with Server-Sent Events for real-time audit feedback.

### Step 10: Verification, Benchmarking, and Humanized Documentation
- Unit test suite for matching, auditing, and ERP generation.
- Performance benchmark verifying execution under 25MB RAM and sub-250ms per 3-document dossier.
- Author humanized `README.md` and documentation files.
