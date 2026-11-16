import React, { useState } from 'react';
import { AuditSummary } from '../types';
import { ShieldCheck, ShieldAlert, DollarSign, Clock, Hash, Check } from 'lucide-react';

interface SavingsSummaryProps {
  summary: AuditSummary;
}

export const SavingsSummary: React.FC<SavingsSummaryProps> = ({ summary }) => {
  const [copied, setCopied] = useState(false);

  const handleCopyHash = () => {
    navigator.clipboard.writeText(summary.audit_hash);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const isApproved = summary.match_status === 'APPROVED';

  return (
    <div className="bg-slate-900 border border-slate-800 rounded-xl p-5 shadow-xl">
      <div className="flex flex-col lg:flex-row items-start lg:items-center justify-between gap-4 mb-4 pb-4 border-b border-slate-800/80">
        <div className="flex items-center gap-3">
          <div
            className={`h-12 w-12 rounded-xl flex items-center justify-center ${
              isApproved
                ? 'bg-emerald-500/10 border border-emerald-500/30 text-emerald-400'
                : 'bg-rose-500/10 border border-rose-500/30 text-rose-400'
            }`}
          >
            {isApproved ? <ShieldCheck className="h-7 w-7" /> : <ShieldAlert className="h-7 w-7" />}
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span
                className={`text-xs px-2.5 py-0.5 rounded-full font-bold uppercase tracking-wider ${
                  isApproved
                    ? 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                    : 'bg-rose-950 text-rose-400 border border-rose-800'
                }`}
              >
                {summary.match_status.replace('_', ' ')}
              </span>
              <span className="text-sm text-slate-400">
                Vendor: <strong className="text-slate-200">{summary.vendor_name}</strong>
              </span>
            </div>
            <div className="text-xs text-slate-400 mt-1 flex items-center gap-3">
              <span>PO: <span className="font-mono text-slate-300">{summary.po_number}</span></span>
              <span>•</span>
              <span>Invoice: <span className="font-mono text-slate-300">{summary.invoice_number}</span></span>
            </div>
          </div>
        </div>

        {/* Audit Hash & Execution Speed */}
        <div className="flex items-center gap-3 self-end lg:self-center">
          <div className="flex items-center gap-1.5 text-xs text-slate-400 bg-slate-800/80 px-2.5 py-1.5 rounded-lg border border-slate-700/60 font-mono">
            <Clock className="h-3.5 w-3.5 text-sky-400" />
            <span>{summary.execution_time_ms}ms</span>
          </div>

          <button
            onClick={handleCopyHash}
            className="flex items-center gap-1.5 text-xs text-slate-400 bg-slate-800/80 hover:bg-slate-700 px-2.5 py-1.5 rounded-lg border border-slate-700/60 transition"
            title="Click to copy SHA-256 Audit Verification Hash"
          >
            <Hash className="h-3.5 w-3.5 text-slate-400" />
            <span className="font-mono">{summary.audit_hash.slice(0, 10)}...</span>
            {copied ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : null}
          </button>
        </div>
      </div>

      {/* KPI Stats Grid */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {/* Total Invoiced */}
        <div className="bg-slate-950/60 border border-slate-800/80 rounded-lg p-3.5">
          <span className="text-[11px] font-semibold tracking-wider uppercase text-slate-400">
            Total Invoiced
          </span>
          <div className="text-xl font-bold text-slate-100 mt-1">
            {summary.currency} {summary.total_invoiced.toLocaleString(undefined, { minimumFractionDigits: 2 })}
          </div>
        </div>

        {/* Authorized Settlement */}
        <div className="bg-slate-950/60 border border-slate-800/80 rounded-lg p-3.5">
          <span className="text-[11px] font-semibold tracking-wider uppercase text-emerald-400">
            Authorized Settlement
          </span>
          <div className="text-xl font-bold text-emerald-400 mt-1">
            {summary.currency} {summary.total_authorized.toLocaleString(undefined, { minimumFractionDigits: 2 })}
          </div>
        </div>

        {/* Prevented Overcharge */}
        <div className="bg-slate-950/60 border border-slate-800/80 rounded-lg p-3.5">
          <span className="text-[11px] font-semibold tracking-wider uppercase text-rose-400">
            Prevented Overcharge
          </span>
          <div className="text-xl font-bold text-rose-400 mt-1 flex items-center gap-1">
            <DollarSign className="h-5 w-5 text-rose-400" />
            {summary.prevented_overcharge.toLocaleString(undefined, { minimumFractionDigits: 2 })}
          </div>
        </div>

        {/* Discrepancy Count */}
        <div className="bg-slate-950/60 border border-slate-800/80 rounded-lg p-3.5">
          <span className="text-[11px] font-semibold tracking-wider uppercase text-slate-400">
            Discrepancies Found
          </span>
          <div className="text-xl font-bold text-amber-400 mt-1">
            {summary.discrepancies.length} {summary.discrepancies.length === 1 ? 'item' : 'items'}
          </div>
        </div>
      </div>
    </div>
  );
};
