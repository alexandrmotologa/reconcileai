import React, { useState, useEffect } from 'react';
import { Dossier, AuditSummary } from './types';
import { Header } from './components/Header';
import { SavingsSummary } from './components/SavingsSummary';
import { ThreeWayDiff } from './components/ThreeWayDiff';
import { DocumentViewer } from './components/DocumentViewer';
import { DiscrepancyList } from './components/DiscrepancyList';
import { ErpExportDrawer } from './components/ErpExportDrawer';
import { UploadModal } from './components/UploadModal';

export const App: React.FC = () => {
  const [dossiers, setDossiers] = useState<Dossier[]>([]);
  const [selectedDossier, setSelectedDossier] = useState<Dossier | null>(null);
  const [auditSummary, setAuditSummary] = useState<AuditSummary | null>(null);
  const [selectedSKU, setSelectedSKU] = useState<string | null>(null);
  const [isAuditing, setIsAuditing] = useState<boolean>(false);
  const [isUploadOpen, setIsUploadOpen] = useState<boolean>(false);
  const [sseConnected, setSseConnected] = useState<boolean>(false);

  // 1. Initial Dossier Fetch
  useEffect(() => {
    fetchDossiers();
  }, []);

  // 2. Connect to SSE Stream
  useEffect(() => {
    const sse = new EventSource('/api/stream');
    sse.onopen = () => setSseConnected(true);
    sse.onerror = () => setSseConnected(false);

    sse.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.type === 'AUDIT_COMPLETE' && data.payload) {
          setAuditSummary(data.payload);
        } else if (data.type === 'DOSSIER_UPLOADED') {
          fetchDossiers();
        }
      } catch (err) {
        // Ignored
      }
    };

    return () => sse.close();
  }, []);

  const fetchDossiers = async () => {
    try {
      const res = await fetch('/api/dossiers');
      if (res.ok) {
        const list: Dossier[] = await res.json();
        setDossiers(list);
        if (list.length > 0) {
          const current = selectedDossier ? list.find((d) => d.id === selectedDossier.id) || list[0] : list[0];
          setSelectedDossier(current);
          runAudit(current.id);
        }
      }
    } catch (err) {
      console.error('Fetch dossiers error:', err);
    }
  };

  const runAudit = async (dossierId: string) => {
    setIsAuditing(true);
    try {
      const res = await fetch(`/api/reconcile/${dossierId}`, { method: 'POST' });
      if (res.ok) {
        const summary: AuditSummary = await res.json();
        setAuditSummary(summary);
      }
    } catch (err) {
      console.error('Reconcile audit error:', err);
    } finally {
      setIsAuditing(false);
    }
  };

  const handleSelectDossier = (d: Dossier) => {
    setSelectedDossier(d);
    setSelectedSKU(null);
    runAudit(d.id);
  };

  return (
    <div className="min-h-screen bg-slate-950 text-slate-100 flex flex-col font-sans">
      <Header
        dossiers={dossiers}
        selectedDossier={selectedDossier}
        onSelectDossier={handleSelectDossier}
        onRunAudit={() => selectedDossier && runAudit(selectedDossier.id)}
        onOpenUpload={() => setIsUploadOpen(true)}
        auditSummary={auditSummary}
        isAuditing={isAuditing}
        sseConnected={sseConnected}
      />

      <main className="flex-1 max-w-7xl w-full mx-auto p-6 space-y-6">
        {/* KPI Summary Banner */}
        {auditSummary && <SavingsSummary summary={auditSummary} />}

        {/* Core Cockpit Workspace */}
        {selectedDossier && auditSummary && (
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
            {/* Left Column: 3-Way Diff Table & ERP Exports (7 cols) */}
            <div className="lg:col-span-7 space-y-6">
              <ThreeWayDiff
                pairs={auditSummary.matched_pairs}
                currency={auditSummary.currency}
                selectedSKU={selectedSKU}
                onSelectLine={(sku) => setSelectedSKU(sku === selectedSKU ? null : sku)}
              />

              <ErpExportDrawer summary={auditSummary} />
            </div>

            {/* Right Column: Visual Document Viewer & Discrepancy Breakdown (5 cols) */}
            <div className="lg:col-span-5 space-y-6">
              <DocumentViewer
                dossier={selectedDossier}
                discrepancies={auditSummary.discrepancies}
                selectedSKU={selectedSKU}
                onSelectSKU={(sku) => setSelectedSKU(sku === selectedSKU ? null : sku)}
              />

              <DiscrepancyList
                discrepancies={auditSummary.discrepancies}
                selectedSKU={selectedSKU}
                onSelectSKU={(sku) => setSelectedSKU(sku === selectedSKU ? null : sku)}
              />
            </div>
          </div>
        )}
      </main>

      <UploadModal
        isOpen={isUploadOpen}
        onClose={() => setIsUploadOpen(false)}
        onUploadSuccess={() => fetchDossiers()}
      />
    </div>
  );
};

export default App;
