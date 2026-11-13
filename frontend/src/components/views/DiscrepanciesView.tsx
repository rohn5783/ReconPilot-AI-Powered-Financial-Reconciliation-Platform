import React from 'react';
import {
  AlertTriangle,
  Sparkles,
  ExternalLink,
  ShieldAlert,
  ArrowRight,
  Filter,
} from 'lucide-react';
import { useReconStore } from '../../store/useReconStore';

export const DiscrepanciesView: React.FC = () => {
  const {
    discrepancies,
    levelFilter,
    setLevelFilter,
    severityFilter,
    setSeverityFilter,
    discrepancyStatusFilter,
    setDiscrepancyStatusFilter,
    discrepancyTypeFilter,
    setDiscrepancyTypeFilter,
    selectDiscrepancy,
  } = useReconStore();

  const formatCurrency = (val?: string) => {
    if (!val) return '₹0.00';
    const num = parseFloat(val);
    if (isNaN(num)) return `₹${val}`;
    return '₹' + num.toLocaleString('en-IN', { minimumFractionDigits: 2, maximumFractionDigits: 4 });
  };

  return (
    <div className="flex-1 flex flex-col h-full overflow-hidden bg-ledger-900">
      {/* Header & Filter Bar */}
      <div className="p-4 border-b border-ledger-700 bg-ledger-850 flex items-center justify-between">
        <div>
          <div className="flex items-center gap-2">
            <AlertTriangle className="w-4 h-4 text-recon-crimson" />
            <h2 className="text-sm font-semibold text-white">Discrepancy Audit & AI Dispute Dossiers</h2>
          </div>
          <p className="text-xs text-ledger-400 mt-0.5">
            Institutional ledger divergence records analyzed by Google Gemini 3.8 Flash with 1-click dispute ticket generator.
          </p>
        </div>

        {/* Filters */}
        <div className="flex items-center gap-3">
          {/* Level Filter */}
          <div className="flex items-center rounded bg-ledger-900 p-0.5 border border-ledger-700 text-xs">
            <button
              onClick={() => setLevelFilter('all')}
              className={`px-2.5 py-1 rounded text-[11px] font-medium transition ${
                levelFilter === 'all' ? 'bg-ledger-750 text-white shadow-sm' : 'text-ledger-400 hover:text-white'
              }`}
            >
              All Levels
            </button>
            <button
              onClick={() => setLevelFilter('LEVEL_1_ORDER_PG')}
              className={`px-2.5 py-1 rounded text-[11px] font-medium transition ${
                levelFilter === 'LEVEL_1_ORDER_PG' ? 'bg-ledger-750 text-white shadow-sm' : 'text-ledger-400 hover:text-white'
              }`}
            >
              Level 1 (Order ↔ PG)
            </button>
            <button
              onClick={() => setLevelFilter('LEVEL_2_PG_BANK')}
              className={`px-2.5 py-1 rounded text-[11px] font-medium transition ${
                levelFilter === 'LEVEL_2_PG_BANK' ? 'bg-ledger-750 text-white shadow-sm' : 'text-ledger-400 hover:text-white'
              }`}
            >
              Level 2 (PG ↔ Bank)
            </button>
          </div>

          {/* Status Filter */}
          <select
            value={discrepancyStatusFilter}
            onChange={(e) => setDiscrepancyStatusFilter(e.target.value as any)}
            className="px-2.5 py-1 rounded bg-ledger-900 border border-ledger-700 text-xs text-ledger-300 font-mono focus:outline-none focus:border-ledger-600"
          >
            <option value="all">All Statuses</option>
            <option value="OPEN">Open</option>
            <option value="IN_DISPUTE">In Dispute</option>
            <option value="RESOLVED">Resolved</option>
          </select>

          {/* Type Filter */}
          <select
            value={discrepancyTypeFilter}
            onChange={(e) => setDiscrepancyTypeFilter(e.target.value)}
            className="px-2.5 py-1 rounded bg-ledger-900 border border-ledger-700 text-xs text-ledger-300 font-mono focus:outline-none focus:border-ledger-600 max-w-[170px] truncate"
          >
            <option value="all">All Mismatch Types</option>
            <option value="FEE_OVERCHARGE">Fee Overcharge</option>
            <option value="TAX_CALCULATION_ERROR">Tax Calc Error</option>
            <option value="ORDER_MISSING_IN_PG">Order Missing In PG</option>
            <option value="PG_ORPHAN_TXN">PG Orphan Txn</option>
            <option value="CAPTURE_STATUS_MISMATCH">Capture Status Mismatch</option>
            <option value="PAYOUT_SHORTFALL">Payout Shortfall</option>
            <option value="BANK_MISSING">Bank Credit Missing</option>
            <option value="UNSETTLED_PAYOUT">Unsettled Payout</option>
          </select>

          {/* Severity Filter */}
          <select
            value={severityFilter}
            onChange={(e) => setSeverityFilter(e.target.value as any)}
            className="px-2.5 py-1 rounded bg-ledger-900 border border-ledger-700 text-xs text-ledger-300 font-mono focus:outline-none focus:border-ledger-600"
          >
            <option value="all">All Severities</option>
            <option value="CRITICAL">Critical</option>
            <option value="HIGH">High</option>
            <option value="MEDIUM">Medium</option>
            <option value="LOW">Low</option>
          </select>
        </div>
      </div>

      {/* Discrepancy Card List */}
      <div className="flex-1 overflow-auto p-4 space-y-3">
        {(discrepancies || []).length === 0 ? (
          <div className="py-20 text-center text-ledger-400 font-mono text-xs">
            No active discrepancies found. Run reconciliation engine to detect ledger variances.
          </div>
        ) : (
          (discrepancies || []).map((d) => {
            const isCritical = d.severity === 'CRITICAL';
            const isHigh = d.severity === 'HIGH';

            return (
              <div
                key={d.id}
                onClick={() => selectDiscrepancy(d)}
                className="p-4 rounded-lg border border-ledger-700 hover:border-recon-crimson/60 bg-ledger-850 hover:bg-ledger-800/80 transition cursor-pointer shadow-sm group"
              >
                <div className="flex items-start justify-between">
                  <div className="space-y-1">
                    <div className="flex items-center gap-2">
                      <span className="font-mono text-xs font-bold text-recon-crimson tracking-wide">
                        {d.discrepancy_type.replace(/_/g, ' ')}
                      </span>
                      <span
                        className={`text-[10px] font-mono px-1.5 py-0.2 rounded uppercase font-semibold ${
                          isCritical
                            ? 'bg-rose-950 text-rose-300 border border-rose-800'
                            : isHigh
                            ? 'bg-amber-950 text-amber-300 border border-amber-800'
                            : 'bg-ledger-800 text-ledger-300'
                        }`}
                      >
                        {d.severity}
                      </span>
                      <span className="text-[10px] font-mono px-1.5 py-0.2 rounded bg-ledger-800 text-ledger-400 border border-ledger-700 uppercase">
                        {d.status}
                      </span>
                    </div>

                    <div className="text-xs font-mono text-ledger-300 flex items-center gap-3">
                      {d.order_id && <span>Order: <strong className="text-white">{d.order_id}</strong></span>}
                      {d.payout_id && <span>Payout Batch: <strong className="text-recon-cyan">{d.payout_id}</strong></span>}
                    </div>
                  </div>

                  {/* Variance figure */}
                  <div className="text-right">
                    <span className="text-[10px] font-mono text-ledger-400 uppercase block">Mismatch Variance</span>
                    <span className="text-base font-mono font-bold text-recon-crimson tabular-nums">
                      Δ {formatCurrency(d.mismatch_amount)}
                    </span>
                  </div>
                </div>

                {/* AI Root Cause Pill */}
                {d.ai_diagnosis && (
                  <div className="mt-3 p-2.5 rounded bg-recon-indigo/10 border border-recon-indigo/25 text-xs flex items-start gap-2">
                    <Sparkles className="w-4 h-4 text-recon-indigo shrink-0 mt-0.5" />
                    <div>
                      <div className="flex items-center gap-2 mb-0.5">
                        <span className="font-semibold text-recon-indigo text-[11px] uppercase tracking-wide">
                          {d.ai_diagnosis.discrepancy_category}
                        </span>
                        <span className="text-[10px] font-mono px-1.5 py-0.2 rounded bg-recon-indigo/20 text-recon-indigo">
                          {d.ai_diagnosis.confidence_score}% Confidence
                        </span>
                      </div>
                      <p className="text-ledger-200 text-xs leading-relaxed">
                        {d.ai_diagnosis.root_cause_summary}
                      </p>
                    </div>
                  </div>
                )}

                {/* Arithmetic strip */}
                <div className="mt-3 pt-2.5 border-t border-ledger-700/60 flex items-center justify-between text-[11px] font-mono text-ledger-400">
                  <div className="flex items-center gap-4">
                    <span>Expected: <strong className="text-white">{formatCurrency(d.expected_amount)}</strong></span>
                    <span>Actual: <strong className="text-white">{formatCurrency(d.actual_amount)}</strong></span>
                  </div>
                  <div className="flex items-center gap-1 text-ledger-400 group-hover:text-white transition">
                    <span>Open Audit Dossier</span>
                    <ExternalLink className="w-3 h-3" />
                  </div>
                </div>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
};

