import React, { useState } from 'react';
import { Dossier, Discrepancy } from '../types';
import { FileText, Layers, AlertCircle } from 'lucide-react';

interface DocumentViewerProps {
  dossier: Dossier;
  discrepancies: Discrepancy[];
  selectedSKU: string | null;
  onSelectSKU: (sku: string) => void;
}

export const DocumentViewer: React.FC<DocumentViewerProps> = ({
  dossier,
  discrepancies,
  selectedSKU,
  onSelectSKU,
}) => {
  const [activeTab, setActiveTab] = useState<'INVOICE' | 'PO' | 'GRN'>('INVOICE');
  const [activeGRNIndex, setActiveGRNIndex] = useState(0);
  const [hoveredDisc, setHoveredDisc] = useState<Discrepancy | null>(null);

  // Resolve active document
  let activeDoc = dossier.invoice;
  if (activeTab === 'PO') {
    activeDoc = dossier.purchase_order;
  } else if (activeTab === 'GRN') {
    activeDoc = dossier.goods_receipts[activeGRNIndex] || dossier.goods_receipts[0];
  }

  return (
    <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow-xl flex flex-col h-full">
      {/* Header and Document Switcher Tabs */}
      <div className="p-3 bg-slate-950/90 border-b border-slate-800 flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-1.5">
          <button
            onClick={() => setActiveTab('INVOICE')}
            className={`px-3 py-1.5 text-xs font-semibold rounded-lg flex items-center gap-1.5 transition ${
              activeTab === 'INVOICE'
                ? 'bg-amber-500/20 text-amber-300 border border-amber-500/40'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <FileText className="h-3.5 w-3.5" />
            <span>Supplier Invoice</span>
          </button>

          <button
            onClick={() => setActiveTab('PO')}
            className={`px-3 py-1.5 text-xs font-semibold rounded-lg flex items-center gap-1.5 transition ${
              activeTab === 'PO'
                ? 'bg-sky-500/20 text-sky-300 border border-sky-500/40'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <FileText className="h-3.5 w-3.5" />
            <span>Purchase Order</span>
          </button>

          <button
            onClick={() => setActiveTab('GRN')}
            className={`px-3 py-1.5 text-xs font-semibold rounded-lg flex items-center gap-1.5 transition ${
              activeTab === 'GRN'
                ? 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/40'
                : 'text-slate-400 hover:text-slate-200'
            }`}
          >
            <Layers className="h-3.5 w-3.5" />
            <span>Warehouse GRN ({dossier.goods_receipts.length})</span>
          </button>
        </div>

        {/* Multi-GRN Selector if multiple GRNs exist */}
        {activeTab === 'GRN' && dossier.goods_receipts.length > 1 && (
          <div className="flex items-center gap-1">
            {dossier.goods_receipts.map((g, idx) => (
              <button
                key={g.header.document_id}
                onClick={() => setActiveGRNIndex(idx)}
                className={`px-2 py-0.5 text-[11px] rounded font-mono ${
                  activeGRNIndex === idx
                    ? 'bg-cyan-600 text-white'
                    : 'bg-slate-800 text-slate-400 hover:text-slate-200'
                }`}
              >
                Part {idx + 1}
              </button>
            ))}
          </div>
        )}

        <div className="text-xs text-slate-400 font-mono">
          Doc ID: <span className="text-slate-200 font-semibold">{activeDoc.header.document_id}</span>
        </div>
      </div>

      {/* Visual Document Canvas Paper Layout */}
      <div className="flex-1 p-3 sm:p-6 overflow-y-auto bg-slate-950/60 relative">
        <div className="max-w-2xl mx-auto bg-white text-slate-900 rounded-lg shadow-2xl p-4 sm:p-8 relative border border-slate-300 font-sans text-xs">
          {/* Document Header */}
          <div className="flex flex-col sm:flex-row justify-between items-start border-b border-slate-200 pb-4 mb-4 gap-2">
            <div>
              <h3 className="text-lg font-bold text-slate-900 tracking-tight">
                {activeDoc.header.document_type}
              </h3>
              <p className="text-slate-500 font-mono text-[11px]">{activeDoc.header.document_id}</p>
              {activeDoc.header.reference_id && (
                <p className="text-slate-600 text-[11px] mt-0.5">
                  Ref PO: <strong>{activeDoc.header.reference_id}</strong>
                </p>
              )}
            </div>
            <div className="sm:text-right">
              <div className="font-bold text-slate-800">{activeDoc.header.vendor_name}</div>
              <div className="text-slate-500 text-[11px]">Date: {activeDoc.header.issue_date}</div>
              {activeDoc.header.payment_terms && (
                <div className="text-slate-500 text-[11px]">{activeDoc.header.payment_terms}</div>
              )}
            </div>
          </div>

          {/* Line Items Table */}
          <div className="overflow-x-auto my-4">
            <table className="w-full text-left border-collapse min-w-[360px]">
              <thead>
                <tr className="border-b-2 border-slate-300 text-slate-600 font-semibold text-[11px]">
                  <th className="py-1.5 px-2">SKU</th>
                <th className="py-1.5 px-2">Description</th>
                <th className="py-1.5 px-2 text-right">Qty</th>
                <th className="py-1.5 px-2 text-center">UOM</th>
                {activeTab !== 'GRN' && (
                  <>
                    <th className="py-1.5 px-2 text-right">Unit Price</th>
                    <th className="py-1.5 px-2 text-right">Total</th>
                  </>
                )}
              </tr>
            </thead>
            <tbody>
              {activeDoc.lines.map((l, idx) => {
                const lineDisc = (discrepancies || []).find(
                  (d) => d.item_sku === l.sku && activeTab === 'INVOICE'
                );
                const isSelected = selectedSKU === l.sku;

                return (
                  <tr
                    key={l.id || idx}
                    onClick={() => onSelectSKU(l.sku)}
                    onMouseEnter={() => lineDisc && setHoveredDisc(lineDisc)}
                    onMouseLeave={() => setHoveredDisc(null)}
                    className={`relative cursor-pointer transition-all ${
                      lineDisc
                        ? 'bg-rose-50 border-2 border-rose-500 rounded font-medium'
                        : isSelected
                        ? 'bg-sky-50 border border-sky-400'
                        : 'hover:bg-slate-50 border-b border-slate-100'
                    }`}
                  >
                    <td className="py-2 px-2 font-mono font-semibold text-slate-800 flex items-center gap-1.5">
                      {lineDisc && (
                        <span className="h-4 w-4 rounded-full bg-rose-600 text-white text-[10px] flex items-center justify-center font-bold">
                          !
                        </span>
                      )}
                      <span>{l.sku}</span>
                    </td>
                    <td className="py-2 px-2 text-slate-700">{l.description}</td>
                    <td className="py-2 px-2 text-right font-mono">{l.quantity}</td>
                    <td className="py-2 px-2 text-center text-slate-500">{l.uom || 'EA'}</td>
                    {activeTab !== 'GRN' && (
                      <>
                        <td className="py-2 px-2 text-right font-mono">${l.unit_price.toFixed(2)}</td>
                        <td className="py-2 px-2 text-right font-mono font-semibold">
                          ${l.total_price.toFixed(2)}
                        </td>
                      </>
                    )}
                  </tr>
                );
              })}
            </tbody>
          </table>
          </div>

          {/* Surcharges & Grand Totals */}
          {activeTab !== 'GRN' && (
            <div className="border-t border-slate-200 pt-3 mt-4 flex flex-col items-end gap-1 text-[11px]">
              <div className="flex justify-between w-48 text-slate-600">
                <span>Subtotal:</span>
                <span className="font-mono font-semibold">
                  ${activeDoc.header.subtotal.toFixed(2)}
                </span>
              </div>
              {activeDoc.header.tax_total > 0 && (
                <div className="flex justify-between w-48 text-slate-600">
                  <span>Tax:</span>
                  <span className="font-mono">${activeDoc.header.tax_total.toFixed(2)}</span>
                </div>
              )}
              {activeDoc.header.freight_total > 0 && (
                <div className="flex justify-between w-48 text-rose-600 font-semibold bg-rose-50 px-1 rounded">
                  <span>Freight:</span>
                  <span className="font-mono">${activeDoc.header.freight_total.toFixed(2)}</span>
                </div>
              )}
              <div className="flex justify-between w-48 text-slate-900 font-bold text-sm border-t border-slate-300 pt-1 mt-1">
                <span>Total Due:</span>
                <span className="font-mono">${activeDoc.header.total_amount.toFixed(2)}</span>
              </div>
            </div>
          )}

          {/* Interactive Discrepancy Popover Overlay */}
          {hoveredDisc && (
            <div className="absolute bottom-4 left-4 right-4 bg-rose-900/95 backdrop-blur text-white p-3 rounded-lg shadow-2xl border border-rose-500 animate-in fade-in slide-in-from-bottom-2">
              <div className="flex items-start gap-2">
                <AlertCircle className="h-5 w-5 text-rose-300 shrink-0 mt-0.5" />
                <div>
                  <h4 className="font-bold text-sm text-rose-100">{hoveredDisc.title}</h4>
                  <p className="text-xs text-rose-200 mt-0.5">{hoveredDisc.description}</p>
                  <div className="mt-1 flex items-center gap-3 text-[11px] font-mono text-rose-300">
                    <span>Expected: {hoveredDisc.expected_value}</span>
                    <span>•</span>
                    <span>Actual: {hoveredDisc.actual_value}</span>
                    <span>•</span>
                    <span className="font-bold text-white">
                      Variance: +${hoveredDisc.variance_amount.toFixed(2)}
                    </span>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};
