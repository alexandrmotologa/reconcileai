import React, { useState, useEffect } from 'react';
import { AuditSummary } from '../types';
import { Download, Copy, Check, FileCode, FileSpreadsheet, FileText, Shield } from 'lucide-react';

interface ErpExportDrawerProps {
  summary: AuditSummary;
}

export const ErpExportDrawer: React.FC<ErpExportDrawerProps> = ({ summary }) => {
  const [activeTab, setActiveTab] = useState<'SAP' | 'NETSUITE' | 'QUICKBOOKS' | 'DISPUTE'>('SAP');
  const [content, setContent] = useState<string>('');
  const [loading, setLoading] = useState<boolean>(false);
  const [copied, setCopied] = useState<boolean>(false);

  useEffect(() => {
    fetchExportContent(activeTab);
  }, [activeTab, summary.dossier_id]);

  const fetchExportContent = async (tab: string) => {
    setLoading(true);
    let endpoint = '';
    switch (tab) {
      case 'SAP':
        endpoint = `/api/export/sap/${summary.dossier_id}`;
        break;
      case 'NETSUITE':
        endpoint = `/api/export/netsuite/${summary.dossier_id}`;
        break;
      case 'QUICKBOOKS':
        endpoint = `/api/export/quickbooks/${summary.dossier_id}`;
        break;
      case 'DISPUTE':
        endpoint = `/api/export/dispute/${summary.dossier_id}`;
        break;
    }

    try {
      const res = await fetch(endpoint);
      if (res.ok) {
        const text = await res.text();
        setContent(text);
      } else {
        setContent(`// Export failed or server returned ${res.status}`);
      }
    } catch (err) {
      setContent(`// Failed to fetch export: ${err}`);
    } finally {
      setLoading(false);
    }
  };

  const handleCopy = () => {
    navigator.clipboard.writeText(content);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleDownload = () => {
    const blob = new Blob([content], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    let ext = 'txt';
    if (activeTab === 'SAP') ext = 'xml';
    else if (activeTab === 'NETSUITE') ext = 'csv';
    else if (activeTab === 'QUICKBOOKS') ext = 'json';
    else if (activeTab === 'DISPUTE') ext = 'md';

    link.href = url;
    link.download = `${summary.dossier_id}_${activeTab.toLowerCase()}.${ext}`;
    link.click();
    URL.revokeObjectURL(url);
  };

  const handleDownloadPDFEvidence = () => {
    window.open(`/api/export/evidence/${summary.dossier_id}`, '_blank');
  };

  return (
    <div className="bg-slate-900 border border-slate-800 rounded-xl overflow-hidden shadow-xl">
      <div className="p-4 border-b border-slate-800 flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <FileCode className="h-5 w-5 text-sky-400" />
          <h3 className="text-sm font-bold text-white">ERP Settlement & Dispute Exports</h3>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={handleDownloadPDFEvidence}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-rose-600/20 text-rose-300 border border-rose-500/30 hover:bg-rose-600/30 text-xs font-semibold transition"
          >
            <Shield className="h-3.5 w-3.5" />
            <span>Download Audit PDF</span>
          </button>

          <button
            onClick={handleCopy}
            className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-300 text-xs transition"
            title="Copy content"
          >
            {copied ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
            <span>Copy</span>
          </button>

          <button
            onClick={handleDownload}
            className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-sky-600 hover:bg-sky-500 text-white text-xs font-semibold transition"
          >
            <Download className="h-3.5 w-3.5" />
            <span>Download</span>
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="bg-slate-950/80 px-4 pt-2 border-b border-slate-800 flex gap-2">
        <button
          onClick={() => setActiveTab('SAP')}
          className={`px-3 py-2 text-xs font-medium border-b-2 flex items-center gap-1.5 transition ${
            activeTab === 'SAP'
              ? 'border-sky-500 text-sky-400 font-semibold'
              : 'border-transparent text-slate-400 hover:text-slate-200'
          }`}
        >
          <FileCode className="h-3.5 w-3.5" />
          <span>SAP S/4HANA (IDoc XML)</span>
        </button>

        <button
          onClick={() => setActiveTab('NETSUITE')}
          className={`px-3 py-2 text-xs font-medium border-b-2 flex items-center gap-1.5 transition ${
            activeTab === 'NETSUITE'
              ? 'border-sky-500 text-sky-400 font-semibold'
              : 'border-transparent text-slate-400 hover:text-slate-200'
          }`}
        >
          <FileSpreadsheet className="h-3.5 w-3.5" />
          <span>NetSuite (CSV)</span>
        </button>

        <button
          onClick={() => setActiveTab('QUICKBOOKS')}
          className={`px-3 py-2 text-xs font-medium border-b-2 flex items-center gap-1.5 transition ${
            activeTab === 'QUICKBOOKS'
              ? 'border-sky-500 text-sky-400 font-semibold'
              : 'border-transparent text-slate-400 hover:text-slate-200'
          }`}
        >
          <FileCode className="h-3.5 w-3.5" />
          <span>QuickBooks (JSON)</span>
        </button>

        <button
          onClick={() => setActiveTab('DISPUTE')}
          className={`px-3 py-2 text-xs font-medium border-b-2 flex items-center gap-1.5 transition ${
            activeTab === 'DISPUTE'
              ? 'border-rose-500 text-rose-400 font-semibold'
              : 'border-transparent text-slate-400 hover:text-slate-200'
          }`}
        >
          <FileText className="h-3.5 w-3.5" />
          <span>Vendor Dispute Memo (MD)</span>
        </button>
      </div>

      {/* Code / Content Viewer */}
      <div className="p-4 bg-slate-950 font-mono text-xs max-h-64 overflow-y-auto text-slate-300">
        {loading ? (
          <div className="py-8 text-center text-slate-500">Generating payload...</div>
        ) : (
          <pre className="whitespace-pre-wrap">{content}</pre>
        )}
      </div>
    </div>
  );
};
