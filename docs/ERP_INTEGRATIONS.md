# ERP Integration Specifications

reconcileai exports verified accounts payable settlements into standard enterprise accounting systems.

## 1. SAP S/4HANA (IDoc XML)

Standard mapping used for automated posting in SAP MM (Materials Management) and FI (Financial Accounting).

- **Document Structure**: `<INVOIC02>`
- **Message Type**: `INVOIC`
- **Control Record**: `<EDI_DC40>` specifies company code, vendor number, and generation timestamps.
- **Header Segment**: `<E1EDK01>` contains payment terms and currency.
- **Reference Segment**: `<E1EDK02>` specifies reference qualifiers:
  - Qualifier `001`: internal purchase order number.
  - Qualifier `002`: vendor invoice number.
- **Line Segment**: `<E1EDP01>` records payable quantity (`MENGE`), unit of measure (`MENEE`), unit price (`VPREI`), and net amount (`NETWR`).
- **Material Segment**: `<E1EDP19>` contains material SKU (`IDTNR`) and line description (`KTEXT`).
- **Total Segment**: `<E1EDS01>` specifies net subtotal, tax amount, and authorized gross total.

Only quantities verified by warehouse intake are settled. Discrepancies are held back pending vendor dispute resolution.

## 2. NetSuite (Bill Import CSV)

Formats settlement entries for standard NetSuite CSV import workflows.

CSV Columns:
- `ExternalID`: unique bill identifier derived from invoice number.
- `Vendor`: vendor corporate name.
- `TransactionDate`: date of audit clearance.
- `DueDate`: net payment due date.
- `InvoiceNumber`: vendor bill number.
- `PurchaseOrderRef`: reference purchase order.
- `Currency`: transaction currency code.
- `Terms`: payment terms (for example Net 30).
- `Memo`: automated reconciliation note citing PO reference.
- `LineNumber`: integer line item sequence.
- `ItemSKU`: item part number.
- `Description`: line description.
- `Quantity`: verified payable quantity.
- `Rate`: contract unit price.
- `Amount`: line item payable total.
- `AuditHash`: first 16 characters of the cryptographic audit hash.

## 3. QuickBooks Online (REST JSON Bill)

Matches the Intuit QuickBooks Online accounting API schema for `Bill` entities.

JSON fields:
- `DocNumber`: vendor invoice number.
- `TxnDate`: document date.
- `DueDate`: payment due date.
- `VendorRef`: vendor identification object.
- `APAccountRef`: standard accounts payable account reference.
- `TotalAmt`: total authorized amount for release.
- `CurrencyRef`: ISO currency code.
- `PrivateNote`: audit clearance memo specifying prevented overcharges.
- `Line`: array of line items with `ItemBasedExpenseLineDetail` containing item reference, quantity, unit price, and extended amount.
