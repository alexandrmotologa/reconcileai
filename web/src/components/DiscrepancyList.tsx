import React, { useState } from 'react';
import { Discrepancy } from '../types';
import { AlertCircle, AlertTriangle, Info, Copy, Check, Eye } from 'lucide-react';

interface DiscrepancyListProps {
  discrepancies: Discrepancy[];
  selectedSKU: string | null;
  onSelectSKU: (sku: string) => void;
}

export const DiscrepancyList: React.FC<DiscrepancyListProps> = ({
  discrepancies,
  selectedSKU,
  onSelectSKU,
}) => {
  const [copiedID, setCopiedID] = useState<string | null>(null);

  const handleCopy = (d: Discrepancy) => {
    const text = `[DISCREPANCY: ${d.title}]\nSKU: ${d.item_sku}\nExpected: ${d.expected_value} | Billed: ${d.actual_value}\nVariance: +$${d.variance_amount.toFixed(2)}\nReason: ${d.description}`;
    navigator.clipboard.writeText(text);
    setCopiedID(d.id);
    setTimeout(() => setCopiedID(null), 2000);
  };

  if (discrepancies.length === 0) {
    return (
      <div className="bg-slate-900 border border-slate-800 rounded-xl p-6 text-center text-slate-400">
        <div className="inline-flex h-12 w-12 rounded-full bg-emerald-500/10 border border-emerald-500/30 text-emerald-400 items-center justify-center mb-3">
          <Check className="h-6 w-6" />
        </div>
        <h3 className="text-sm font-semibold text-slate-200">Zero Discrepancies Detected</h3>
        <p className="text-xs text-slate-400 mt-1">
          All line items, quantities, and prices match contract purchase order invariants.
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between px-1">
        <h3 className="text-xs font-bold uppercase tracking-wider text-slate-400">
          Detected Discrepancies ({discrepancies.length})
        </h3>
        <span className="text-[11px] text-rose-400 font-medium">Requires Resolution</span>
      </div>

      <div className="space-y-2.5">
        {discrepancies.map((d) => {
          const isSelected = selectedSKU === d.item_sku;

          return (
            <div
              key={d.id}
              onClick={() => onSelectSKU(d.item_sku)}
              className={`p-3.5 rounded-xl border transition-all cursor-pointer ${
                isSelected
                  ? 'bg-rose-950/40 border-rose-500 shadow-lg shadow-rose-950/50'
                  : 'bg-slate-900 border-slate-800 hover:border-slate-700'
              }`}
            >
              <div className="flex items-start justify-between gap-2">
                <div className="flex items-center gap-2">
                  {d.severity === 'CRITICAL' ? (
                    <AlertCircle className="h-4 w-4 text-rose-400 shrink-0" />
                  ) : d.severity === 'WARNING' ? (
                    <AlertTriangle className="h-4 w-4 text-amber-400 shrink-0" />
                  ) : (
                    <Info className="h-4 w-4 text-sky-400 shrink-0" />
                  )}
                  <h4 className="text-xs font-bold text-slate-200 line-clamp-1">{d.title}</h4>
                </div>

                <span className="text-xs font-mono font-bold text-rose-400 bg-rose-500/10 px-2 py-0.5 rounded border border-rose-500/20 shrink-0">
                  +${d.variance_amount.toFixed(2)}
                </span>
              </div>

              <p className="text-[11px] text-slate-400 mt-1.5 leading-relaxed font-sans">
                {d.description}
              </p>

              <div className="mt-2.5 pt-2 border-t border-slate-800/80 flex items-center justify-between text-[11px]">
                <div className="flex items-center gap-2 text-slate-400 font-mono">
                  <span>Exp: <strong className="text-slate-300">{d.expected_value}</strong></span>
                  <span>•</span>
                  <span>Act: <strong className="text-slate-300">{d.actual_value}</strong></span>
                </div>

                <div className="flex items-center gap-1.5">
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      handleCopy(d);
                    }}
                    className="p-1 rounded text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition"
                    title="Copy dispute note text"
                  >
                    {copiedID === d.id ? (
                      <Check className="h-3.5 w-3.5 text-emerald-400" />
                    ) : (
                      <Copy className="h-3.5 w-3.5" />
                    )}
                  </button>

                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      onSelectSKU(d.item_sku);
                    }}
                    className="p-1 rounded text-slate-400 hover:text-slate-200 hover:bg-slate-800 transition"
                    title="Focus on document"
                  >
                    <Eye className="h-3.5 w-3.5" />
                  </button>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};
