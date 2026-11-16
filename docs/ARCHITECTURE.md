# System Architecture

reconcileai automates 3-way document matching for accounts payable departments. The application reconciles purchase orders, warehouse goods receipts, and supplier invoices without cloud dependencies.

## Data Flow

```
1. Document Ingestion
   [Purchase Order PDF/JSON]   [Goods Receipt Notes (1..N)]   [Supplier Invoice PDF/JSON]
               \                           |                           /
                +--------------------------+--------------------------+
                                           |
                                           v
2. Normalization & Spatial Layout Extraction
   - Extract text tokens and bounding coordinates (X, Y, W, H)
   - Convert packaging units (Dozen, Case, Box to Each; Lbs to Kg)
   - Strip legal suffixes from entity names (Inc, LLC, Corp)
                                           |
                                           v
3. Bipartite Line Matching
   - Exact SKU match (1.0 weight)
   - Substring SKU alignment (0.92 weight)
   - Token Jaccard overlap and Jaro-Winkler string distance
   - Aggregate split-shipment quantities across multiple receipts
                                           |
                                           v
4. Mathematical Verification Engine
   - Invariant 1: Invoiced quantity <= received quantity
   - Invariant 2: Unit price variance <= contract tolerance
   - Invariant 3: Line items sum check equals billed subtotal
   - Invariant 4: Freight and surcharge authorization check
   - Invariant 5: SHA-256 cryptographic audit trail computation
                                           |
                   +-----------------------+-----------------------+
                   |                                               |
                   v                                               v
5a. Zero Discrepancy                            5b. Discrepancy Detected
   - Certified settlement status                   - Flagged status
   - SAP S/4HANA IDoc XML generation              - Red bounding box PDF annotation
   - NetSuite CSV bill payload                    - Itemized vendor dispute memo
   - QuickBooks Online JSON payload                - React 19 cockpit human review
```

## Core Modules

### internal/ingestion
Reads digital PDF files and structured JSON files.
- `pdf_parser.go`: clusters raw PDF text chunks into horizontal lines by Y coordinate and extracts document metadata.
- `table_extractor.go`: identifies table header rows and parses SKU, description, quantity, UOM, and price columns.
- `uom.go`: normalizes units of measure across mass, length, volume, and count categories.
- `dossier_loader.go`: loads directories containing tripartite document sets.

### internal/matching
Pairs invoice line items with purchase order line items.
- `similarity.go`: implements Jaro-Winkler string distance, token Jaccard similarity, and plural stemming.
- `entity_aligner.go`: normalizes vendor names and validates cross-referenced purchase order identifiers.
- `bipartite_matcher.go`: finds candidate pairings and normalizes quantities and unit prices when units differ.

### internal/auditor
Evaluates mathematical business rules.
- `quantity.go`: checks whether billed quantities exceed verified goods receipts.
- `price.go`: verifies unit prices against contract rates within absolute or percentage tolerances.
- `tax_surcharge.go`: audits sales tax calculations and detects unapproved freight charges.
- `arithmetic.go`: verifies that row multiplications sum to subtotal and grand total.
- `engine.go`: coordinates the audit workflow, calculates total authorized amounts, and generates SHA-256 audit hashes.

### internal/report
Formats dispute notices and annotates document pages.
- `memo.go`: formats plain text and markdown dispute letters citing contract terms, variances, and requested actions.
- `pdf_annotator.go`: creates audit evidence packets with red rectangular bounding boxes and numbered discrepancy markers.

### internal/erp
Exports approved settlements.
- `sap_idoc.go`: formats SAP S/4HANA INVOIC02 XML documents for accounts payable posting.
- `netsuite.go`: formats NetSuite Bill import CSV records.
- `quickbooks.go`: formats QuickBooks Online REST Bill JSON payloads.

### internal/server
Serves HTTP API endpoints and the embedded web cockpit.
- `api.go`: REST endpoints for listing, auditing, exporting, and uploading dossiers.
- `sse.go`: server-sent events broadcaster for real-time audit notifications.

### web
Single-page application cockpit built with React 19, Vite, and Tailwind CSS. Static assets embed directly into the Go binary.
