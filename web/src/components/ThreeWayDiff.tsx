import React from 'react';
import { MatchedLinePair } from '../types';
import { CheckCircle2, AlertTriangle, Zap, Download } from 'lucide-react';

interface ThreeWayDiffProps {
  pairs: MatchedLinePair[];
  currency: string;
  selectedSKU: string | null;
  dossierId?: string;
  onSelectLine: (sku: string) => void;
}

export const ThreeWayDiff: React.FC<ThreeWayDiffProps> = ({
  pairs,
  currency,
  selectedSKU,
  dossierId,
  onSelectLine,
}) => {
  return (
    <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow-xl">
      <div className="p-4 border-b border-slate-800 flex items-center justify-between">
        <div>
          <h2 className="text-base font-bold text-white flex items-center gap-2">
            <span>3-Way Line Matching Table</span>
            <span className="text-xs font-normal text-slate-400">
              (PO Contract vs Warehouse GRN vs Supplier Invoice)
            </span>
          </h2>
        </div>
        <div className="flex items-center gap-4 text-xs">
          <div className="flex items-center gap-1.5 text-emerald-400">
            <CheckCircle2 className="h-3.5 w-3.5" />
            <span>Matched</span>
          </div>
          <div className="flex items-center gap-1.5 text-rose-400">
            <AlertTriangle className="h-3.5 w-3.5" />
            <span>Discrepancy</span>
          </div>
          {dossierId && (
            <a
              href={`/api/export/csv/${dossierId}`}
              download={`diff_${dossierId}.csv`}
              className="flex items-center gap-1.5 bg-slate-800 hover:bg-slate-700 text-slate-300 px-2.5 py-1 rounded-md border border-slate-700 transition"
              title="Download line diff as CSV"
            >
              <Download className="h-3.5 w-3.5 text-sky-400" />
              <span>Export CSV</span>
            </a>
          )}
        </div>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs border-collapse">
          <thead>
            <tr className="bg-slate-950/80 text-slate-400 uppercase tracking-wider font-semibold border-b border-slate-800">
              <th className="py-3 px-3 w-10 text-center">Status</th>
              <th className="py-3 px-3">Item / SKU</th>
              <th className="py-3 px-3 text-center">Alignment</th>
              <th className="py-3 px-3 bg-slate-900/50 border-l border-slate-800 text-sky-400">PO Ordered</th>
              <th className="py-3 px-3 bg-slate-900/50 text-sky-400">PO Price</th>
              <th className="py-3 px-3 bg-cyan-950/20 border-l border-slate-800 text-cyan-400">GRN Received</th>
              <th className="py-3 px-3 bg-slate-900/50 border-l border-slate-800 text-amber-400">Invoice Billed</th>
              <th className="py-3 px-3 bg-slate-900/50 text-amber-400">Invoice Price</th>
              <th className="py-3 px-3 border-l border-slate-800 text-right">Variance Impact</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-800/60 font-mono">
            {pairs.map((p) => {
              const isSelected = selectedSKU === p.item_sku;
              const hasQtyMismatch = p.invoiced_qty > p.received_qty;
              const hasPriceMismatch = p.price_variance > 0;
              const isUnmatched = !p.po_line_id;

              return (
                <tr
                  key={p.invoice_line_id}
                  onClick={() => onSelectLine(p.item_sku)}
                  className={`cursor-pointer transition-colors ${
                    isSelected
                      ? 'bg-sky-950/40 border-l-4 border-sky-500'
                      : p.has_discrepancy
                      ? 'bg-rose-950/20 hover:bg-rose-950/30'
                      : 'hover:bg-slate-800/40'
                  }`}
                >
                  {/* Status Icon */}
                  <td className="py-3 px-3 text-center">
                    {p.has_discrepancy ? (
                      <AlertTriangle className="h-4 w-4 text-rose-400 inline" />
                    ) : (
                      <CheckCircle2 className="h-4 w-4 text-emerald-400 inline" />
                    )}
                  </td>

                  {/* SKU & Description */}
                  <td className="py-3 px-3 font-sans">
                    <div className="font-semibold text-slate-200">{p.item_sku}</div>
                    <div className="text-[11px] text-slate-400 line-clamp-1">{p.description}</div>
                  </td>

                  {/* Alignment Confidence */}
                  <td className="py-3 px-3 text-center">
                    <span
                      className={`inline-flex items-center gap-1 text-[10px] px-1.5 py-0.5 rounded font-mono ${
                        p.confidence >= 0.9
                          ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/30'
                          : p.confidence > 0
                          ? 'bg-amber-500/10 text-amber-400 border border-amber-500/30'
                          : 'bg-rose-500/10 text-rose-400 border border-rose-500/30'
                      }`}
                      title={p.match_reason}
                    >
                      <Zap className="h-2.5 w-2.5" />
                      {Math.round(p.confidence * 100)}%
                    </span>
                  </td>

                  {/* PO Ordered */}
                  <td className="py-3 px-3 bg-slate-900/30 border-l border-slate-800 text-slate-300">
                    {p.ordered_qty > 0 ? p.ordered_qty.toFixed(1) : '-'}
                  </td>

                  {/* PO Price */}
                  <td className="py-3 px-3 bg-slate-900/30 text-slate-300">
                    {p.po_price > 0 ? `${currency} ${p.po_price.toFixed(2)}` : '-'}
                  </td>

                  {/* GRN Received */}
                  <td
                    className={`py-3 px-3 bg-cyan-950/10 border-l border-slate-800 ${
                      hasQtyMismatch ? 'text-rose-400 font-bold bg-rose-950/30' : 'text-cyan-300'
                    }`}
                  >
                    {p.received_qty.toFixed(1)}
                  </td>

                  {/* Invoice Billed */}
                  <td
                    className={`py-3 px-3 bg-slate-900/30 border-l border-slate-800 ${
                      hasQtyMismatch ? 'text-rose-400 font-bold bg-rose-950/30' : 'text-slate-300'
                    }`}
                  >
                    {p.invoiced_qty.toFixed(1)}
                  </td>

                  {/* Invoice Price */}
                  <td
                    className={`py-3 px-3 bg-slate-900/30 ${
                      hasPriceMismatch ? 'text-rose-400 font-bold bg-rose-950/30' : 'text-slate-300'
                    }`}
                  >
                    {currency} {p.invoiced_price.toFixed(2)}
                  </td>

                  {/* Variance Impact */}
                  <td className="py-3 px-3 border-l border-slate-800 text-right">
                    {isUnmatched ? (
                      <span className="text-rose-400 font-bold">Unapproved Line</span>
                    ) : p.has_discrepancy ? (
                      <span className="text-rose-400 font-bold">
                        {hasQtyMismatch && `+${(p.invoiced_qty - p.received_qty).toFixed(1)} Qty `}
                        {hasPriceMismatch && `+${currency} ${p.price_variance.toFixed(2)}/unit`}
                      </span>
                    ) : (
                      <span className="text-emerald-400 font-medium">{currency} 0.00</span>
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
};
