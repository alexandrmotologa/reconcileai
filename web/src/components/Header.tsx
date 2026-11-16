import React from 'react';
import { Dossier, AuditSummary } from '../types';
import { FileCheck, Play, Upload, RefreshCw, Radio } from 'lucide-react';

interface HeaderProps {
  dossiers: Dossier[];
  selectedDossier: Dossier | null;
  onSelectDossier: (d: Dossier) => void;
  onRunAudit: () => void;
  onOpenUpload: () => void;
  auditSummary?: AuditSummary | null;
  isAuditing: boolean;
  sseConnected: boolean;
}

export const Header: React.FC<HeaderProps> = ({
  dossiers,
  selectedDossier,
  onSelectDossier,
  onRunAudit,
  onOpenUpload,
  isAuditing,
  sseConnected,
}) => {
  return (
    <header className="border-b border-slate-800 bg-slate-900/95 backdrop-blur px-6 py-4 sticky top-0 z-30">
      <div className="max-w-7xl mx-auto flex flex-col md:flex-row items-center justify-between gap-4">
        {/* Brand */}
        <div className="flex items-center gap-3">
          <div className="h-10 w-10 rounded-lg bg-sky-500/10 border border-sky-500/30 flex items-center justify-center text-sky-400">
            <FileCheck className="h-6 w-6" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-xl font-bold tracking-tight text-white">reconcileai</h1>
              <span className="text-xs px-2 py-0.5 rounded-full bg-slate-800 text-slate-300 border border-slate-700">
                v1.0
              </span>
              <span
                className={`inline-flex items-center gap-1 text-[11px] px-2 py-0.5 rounded-full font-mono ${
                  sseConnected
                    ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/30'
                    : 'bg-amber-500/10 text-amber-400 border border-amber-500/30'
                }`}
                title={sseConnected ? 'SSE Live Stream Active' : 'Connecting to Stream'}
              >
                <Radio className={`h-3 w-3 ${sseConnected ? 'animate-pulse' : ''}`} />
                {sseConnected ? 'LIVE' : 'SYNCING'}
              </span>
            </div>
            <p className="text-xs text-slate-400">
              Autonomous 3-Way Matching & Discrepancy Auditor for Accounts Payable
            </p>
          </div>
        </div>

        {/* Action Controls */}
        <div className="flex items-center gap-3 w-full md:w-auto">
          {/* Dossier Selector */}
          <div className="relative flex-1 md:w-72">
            <select
              value={selectedDossier?.id || ''}
              onChange={(e) => {
                const found = dossiers.find((d) => d.id === e.target.value);
                if (found) onSelectDossier(found);
              }}
              className="w-full bg-slate-800 border border-slate-700 rounded-lg px-3 py-2 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-sky-500"
            >
              {dossiers.map((d) => (
                <option key={d.id} value={d.id}>
                  {d.title || d.id}
                </option>
              ))}
            </select>
          </div>

          {/* Upload Button */}
          <button
            onClick={onOpenUpload}
            className="flex items-center gap-1.5 px-3 py-2 text-sm font-medium rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-200 border border-slate-700 transition"
            title="Upload custom PO, GRN, and Invoice PDFs"
          >
            <Upload className="h-4 w-4 text-slate-400" />
            <span>Upload</span>
          </button>

          {/* Reconcile Audit Button */}
          <button
            onClick={onRunAudit}
            disabled={isAuditing}
            className="flex items-center gap-2 px-4 py-2 text-sm font-semibold rounded-lg bg-sky-600 hover:bg-sky-500 disabled:opacity-50 text-white shadow-lg shadow-sky-950 transition"
          >
            {isAuditing ? (
              <>
                <RefreshCw className="h-4 w-4 animate-spin" />
                <span>Auditing...</span>
              </>
            ) : (
              <>
                <Play className="h-4 w-4 fill-white" />
                <span>Reconcile 3-Way</span>
              </>
            )}
          </button>
        </div>
      </div>
    </header>
  );
};
