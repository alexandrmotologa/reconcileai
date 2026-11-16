import React, { useEffect, useState } from 'react';
import { VendorScorecard } from '../types';
import { X, Award, Search } from 'lucide-react';

interface VendorScorecardModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const VendorScorecardModal: React.FC<VendorScorecardModalProps> = ({ isOpen, onClose }) => {
  const [scorecards, setScorecards] = useState<VendorScorecard[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');

  useEffect(() => {
    if (!isOpen) return;
    setLoading(true);
    fetch('/api/vendors/scorecard')
      .then((res) => res.json())
      .then((data) => {
        setScorecards(Array.isArray(data) ? data : []);
        setLoading(false);
      })
      .catch((err) => {
        console.error('Failed to fetch scorecards:', err);
        setLoading(false);
      });
  }, [isOpen]);

  if (!isOpen) return null;

  const filtered = scorecards.filter((sc) =>
    sc.vendor_name.toLowerCase().includes(search.toLowerCase())
  );

  const totalInvoiced = scorecards.reduce((acc, sc) => acc + sc.total_invoiced, 0);
  const totalPrevented = scorecards.reduce((acc, sc) => acc + sc.prevented_overcharge, 0);
  const avgAccuracy =
    scorecards.length > 0
      ? scorecards.reduce((acc, sc) => acc + sc.accuracy_rate, 0) / scorecards.length
      : 100;
  const highRiskCount = scorecards.filter((sc) => sc.risk_tier === 'HIGH').length;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm animate-in fade-in duration-200">
      <div className="bg-slate-900 border border-slate-800 rounded-2xl w-full max-w-4xl max-h-[90vh] flex flex-col shadow-2xl overflow-hidden">
        {/* Modal Header */}
        <div className="p-5 border-b border-slate-800 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="h-10 w-10 rounded-xl bg-sky-500/10 border border-sky-500/30 flex items-center justify-center text-sky-400">
              <Award className="h-5 w-5" />
            </div>
            <div>
              <h2 className="text-lg font-bold text-white">Vendor Compliance & Risk Scorecard</h2>
              <p className="text-xs text-slate-400">
                Audited accuracy metrics, billing variance rates, and ERP settlement risk tiers
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="h-8 w-8 rounded-lg bg-slate-800 hover:bg-slate-700 text-slate-400 hover:text-white flex items-center justify-center transition"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Aggregate KPI Strip */}
        <div className="p-5 bg-slate-950/60 border-b border-slate-800 grid grid-cols-2 md:grid-cols-5 gap-3">
          <div className="bg-slate-900/80 border border-slate-800 rounded-lg p-3">
            <span className="text-[10px] uppercase font-semibold text-slate-400">Tracked Vendors</span>
            <div className="text-xl font-bold text-white mt-0.5">{scorecards.length}</div>
          </div>
          <div className="bg-slate-900/80 border border-slate-800 rounded-lg p-3">
            <span className="text-[10px] uppercase font-semibold text-slate-400">Invoiced Spend</span>
            <div className="text-xl font-bold text-slate-200 mt-0.5">${totalInvoiced.toFixed(0)}</div>
          </div>
          <div className="bg-slate-900/80 border border-slate-800 rounded-lg p-3">
            <span className="text-[10px] uppercase font-semibold text-slate-400">Avg Compliance</span>
            <div className="text-xl font-bold text-emerald-400 mt-0.5">{avgAccuracy.toFixed(1)}%</div>
          </div>
          <div className="bg-slate-900/80 border border-slate-800 rounded-lg p-3">
            <span className="text-[10px] uppercase font-semibold text-slate-400">Prevented Overcharge</span>
            <div className="text-xl font-bold text-rose-400 mt-0.5">${totalPrevented.toFixed(2)}</div>
          </div>
          <div className="bg-slate-900/80 border border-slate-800 rounded-lg p-3">
            <span className="text-[10px] uppercase font-semibold text-slate-400">High Risk Suppliers</span>
            <div className="text-xl font-bold text-amber-400 mt-0.5">{highRiskCount}</div>
          </div>
        </div>

        {/* Filter bar */}
        <div className="p-4 border-b border-slate-800 flex items-center gap-3">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-2.5 h-4 w-4 text-slate-500" />
            <input
              type="text"
              placeholder="Search vendor name..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="w-full bg-slate-950 border border-slate-800 rounded-lg pl-9 pr-3 py-1.5 text-xs text-slate-200 placeholder-slate-500 focus:outline-none focus:border-sky-500 transition"
            />
          </div>
        </div>

        {/* Scorecard Table */}
        <div className="flex-1 overflow-y-auto p-5">
          {loading ? (
            <div className="py-12 text-center text-xs text-slate-500">Calculating vendor audit scorecards...</div>
          ) : filtered.length === 0 ? (
            <div className="py-12 text-center text-xs text-slate-500">No vendor scorecards recorded yet. Run audits to build history.</div>
          ) : (
            <table className="w-full text-left text-xs border-collapse">
              <thead>
                <tr className="bg-slate-950/80 text-slate-400 uppercase tracking-wider font-semibold border-b border-slate-800">
                  <th className="py-3 px-3">Vendor</th>
                  <th className="py-3 px-3 text-center">Audited Invoices</th>
                  <th className="py-3 px-3 text-center">Clean vs Flagged</th>
                  <th className="py-3 px-3">Accuracy</th>
                  <th className="py-3 px-3 text-right">Invoiced Spend</th>
                  <th className="py-3 px-3 text-right">Prevented Loss</th>
                  <th className="py-3 px-3 text-center">Risk Tier</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-800/60 font-mono">
                {filtered.map((sc, idx) => (
                  <tr key={idx} className="hover:bg-slate-800/30 transition">
                    <td className="py-3 px-3 font-sans font-medium text-slate-200">{sc.vendor_name}</td>
                    <td className="py-3 px-3 text-center text-slate-300">{sc.total_audited_invoices}</td>
                    <td className="py-3 px-3 text-center">
                      <span className="text-emerald-400">{sc.clean_invoices}</span> /{' '}
                      <span className="text-rose-400">{sc.flagged_invoices}</span>
                    </td>
                    <td className="py-3 px-3">
                      <div className="flex items-center gap-2">
                        <div className="w-16 bg-slate-800 h-1.5 rounded-full overflow-hidden">
                          <div
                            className={`h-full ${
                              sc.accuracy_rate >= 90
                                ? 'bg-emerald-400'
                                : sc.accuracy_rate >= 70
                                ? 'bg-amber-400'
                                : 'bg-rose-400'
                            }`}
                            style={{ width: `${sc.accuracy_rate}%` }}
                          />
                        </div>
                        <span className="text-xs text-slate-300">{sc.accuracy_rate.toFixed(0)}%</span>
                      </div>
                    </td>
                    <td className="py-3 px-3 text-right text-slate-300">${sc.total_invoiced.toFixed(2)}</td>
                    <td className="py-3 px-3 text-right font-bold text-rose-400">
                      {sc.prevented_overcharge > 0 ? `+$${sc.prevented_overcharge.toFixed(2)}` : '$0.00'}
                    </td>
                    <td className="py-3 px-3 text-center">
                      <span
                        className={`text-[10px] px-2 py-0.5 rounded-full font-bold uppercase ${
                          sc.risk_tier === 'LOW'
                            ? 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                            : sc.risk_tier === 'MEDIUM'
                            ? 'bg-amber-950 text-amber-400 border border-amber-800'
                            : 'bg-rose-950 text-rose-400 border border-rose-800'
                        }`}
                      >
                        {sc.risk_tier} RISK
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        {/* Modal Footer */}
        <div className="p-4 border-t border-slate-800 bg-slate-950 flex justify-end">
          <button
            onClick={onClose}
            className="px-4 py-1.5 bg-slate-800 hover:bg-slate-700 text-slate-200 text-xs font-semibold rounded-lg transition"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};
