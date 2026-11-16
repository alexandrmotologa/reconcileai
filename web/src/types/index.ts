export type DocumentType = 'PO' | 'GRN' | 'INVOICE';

export interface BoundingBox {
  page: number;
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface LineItem {
  id: string;
  line_number: number;
  sku: string;
  description: string;
  quantity: number;
  uom: string;
  unit_price: number;
  total_price: number;
  tax_rate?: number;
  tax_amount?: number;
  bounding_box: BoundingBox;
}

export interface DocumentHeader {
  document_id: string;
  document_type: DocumentType;
  reference_id?: string;
  vendor_id?: string;
  vendor_name: string;
  vendor_tax_id?: string;
  issue_date: string;
  currency: string;
  subtotal: number;
  tax_total: number;
  freight_total: number;
  total_amount: number;
  payment_terms?: string;
}

export interface Document {
  header: DocumentHeader;
  lines: LineItem[];
  raw_pdf_path?: string;
}

export interface Dossier {
  id: string;
  title: string;
  purchase_order: Document;
  goods_receipts: Document[];
  invoice: Document;
}

export type DiscrepancyType =
  | 'QUANTITY_OVERBILLED'
  | 'PRICE_VARIANCE_EXCEEDED'
  | 'UNAUTHORIZED_SURCHARGE'
  | 'ARITHMETIC_MISMATCH'
  | 'UNMATCHED_INVOICE_ITEM'
  | 'UOM_CONVERSION_ERROR'
  | 'CURRENCY_FX_VARIANCE'
  | 'DUPLICATE_INVOICE';

export type DiscrepancySeverity = 'CRITICAL' | 'WARNING' | 'INFO';

export interface EarlyDiscount {
  terms: string;
  discount_percent: number;
  discount_days: number;
  net_days: number;
  discount_amount: number;
  discount_deadline: string;
  annualized_roi: number;
  is_eligible: boolean;
}

export interface VendorScorecard {
  vendor_name: string;
  total_audited_invoices: number;
  clean_invoices: number;
  flagged_invoices: number;
  total_invoiced: number;
  total_authorized: number;
  prevented_overcharge: number;
  accuracy_rate: number;
  risk_tier: 'LOW' | 'MEDIUM' | 'HIGH';
}

export interface MetricsSummary {
  total_dossiers: number;
  total_audited: number;
  approved_count: number;
  flagged_count: number;
  total_invoiced_spend: number;
  total_authorized_spend: number;
  total_prevented_overcharge: number;
  early_discount_potential: number;
  discrepancy_breakdown: Record<string, number>;
}

export interface Discrepancy {
  id: string;
  type: DiscrepancyType;
  severity: DiscrepancySeverity;
  title: string;
  description: string;
  item_sku: string;
  expected_value: number;
  actual_value: number;
  variance_amount: number;
  currency: string;
  target_document: DocumentType;
  page_number: number;
  bounding_box: BoundingBox;
}

export interface MatchedLinePair {
  invoice_line_id: string;
  po_line_id: string;
  grn_line_ids: string[];
  item_sku: string;
  description: string;
  confidence: number;
  match_reason: string;
  invoiced_qty: number;
  received_qty: number;
  ordered_qty: number;
  invoiced_price: number;
  po_price: number;
  price_variance: number;
  has_discrepancy: boolean;
}

export interface AuditSummary {
  dossier_id: string;
  vendor_name: string;
  invoice_number: string;
  po_number: string;
  match_status: 'APPROVED' | 'FLAGGED_DISCREPANCY' | 'REJECTED';
  total_invoiced: number;
  total_authorized: number;
  prevented_overcharge: number;
  currency: string;
  early_discount?: EarlyDiscount;
  discrepancies: Discrepancy[];
  matched_pairs: MatchedLinePair[];
  audited_at: string;
  execution_time_ms: number;
  audit_hash: string;
}
