import React, { useState } from 'react';
import { X, UploadCloud, AlertCircle } from 'lucide-react';

interface UploadModalProps {
  isOpen: boolean;
  onClose: () => void;
  onUploadSuccess: () => void;
}

export const UploadModal: React.FC<UploadModalProps> = ({
  isOpen,
  onClose,
  onUploadSuccess,
}) => {
  const [poFile, setPoFile] = useState<File | null>(null);
  const [grnFile, setGrnFile] = useState<File | null>(null);
  const [invFile, setInvFile] = useState<File | null>(null);
  const [isUploading, setIsUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!poFile || !grnFile || !invFile) {
      setError('Please select all three required documents: Purchase Order, Goods Receipt, and Invoice.');
      return;
    }

    setIsUploading(true);
    setError(null);

    const formData = new FormData();
    formData.append('po', poFile);
    formData.append('grn', grnFile);
    formData.append('invoice', invFile);

    try {
      const res = await fetch('/api/upload', {
        method: 'POST',
        body: formData,
      });

      if (!res.ok) {
        const errText = await res.text();
        throw new Error(errText || 'Upload failed');
      }

      onUploadSuccess();
      onClose();
    } catch (err: any) {
      setError(err.message || 'Failed to upload and parse documents');
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 bg-slate-950/80 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl max-w-md w-full p-6 shadow-2xl animate-in fade-in zoom-in-95">
        <div className="flex items-center justify-between pb-3 border-b border-slate-800">
          <div className="flex items-center gap-2">
            <UploadCloud className="h-5 w-5 text-sky-400" />
            <h3 className="text-base font-bold text-white">Upload Custom Dossier</h3>
          </div>
          <button onClick={onClose} className="text-slate-400 hover:text-white p-1">
            <X className="h-4 w-4" />
          </button>
        </div>

        {error && (
          <div className="mt-4 p-3 bg-rose-950/50 border border-rose-800 text-rose-300 text-xs rounded-lg flex items-center gap-2">
            <AlertCircle className="h-4 w-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="mt-4 space-y-4 text-xs">
          {/* PO Upload */}
          <div>
            <label className="block font-semibold text-slate-300 mb-1">
              1. Purchase Order (PDF or JSON)
            </label>
            <input
              type="file"
              accept=".pdf,.json"
              onChange={(e) => setPoFile(e.target.files?.[0] || null)}
              className="w-full text-slate-400 file:mr-3 file:py-1.5 file:px-3 file:rounded-lg file:border-0 file:text-xs file:font-semibold file:bg-sky-600/20 file:text-sky-300 hover:file:bg-sky-600/30"
            />
          </div>

          {/* GRN Upload */}
          <div>
            <label className="block font-semibold text-slate-300 mb-1">
              2. Goods Receipt Note (PDF or JSON)
            </label>
            <input
              type="file"
              accept=".pdf,.json"
              onChange={(e) => setGrnFile(e.target.files?.[0] || null)}
              className="w-full text-slate-400 file:mr-3 file:py-1.5 file:px-3 file:rounded-lg file:border-0 file:text-xs file:font-semibold file:bg-cyan-600/20 file:text-cyan-300 hover:file:bg-cyan-600/30"
            />
          </div>

          {/* Invoice Upload */}
          <div>
            <label className="block font-semibold text-slate-300 mb-1">
              3. Supplier Invoice (PDF or JSON)
            </label>
            <input
              type="file"
              accept=".pdf,.json"
              onChange={(e) => setInvFile(e.target.files?.[0] || null)}
              className="w-full text-slate-400 file:mr-3 file:py-1.5 file:px-3 file:rounded-lg file:border-0 file:text-xs file:font-semibold file:bg-amber-600/20 file:text-amber-300 hover:file:bg-amber-600/30"
            />
          </div>

          <div className="pt-3 border-t border-slate-800 flex justify-end gap-2">
            <button
              type="button"
              onClick={onClose}
              className="px-3 py-1.5 rounded-lg text-slate-300 hover:bg-slate-800"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isUploading}
              className="px-4 py-1.5 rounded-lg bg-sky-600 hover:bg-sky-500 font-semibold text-white disabled:opacity-50"
            >
              {isUploading ? 'Parsing & Ingesting...' : 'Upload & Reconcile'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};
