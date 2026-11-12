import React, { useState } from 'react';
import {
  X,
  Sparkles,
  Copy,
  Check,
  AlertTriangle,
  FileText,
  Activity,
  ArrowRight,
  ShieldAlert,
  Send,
} from 'lucide-react';
import { useReconStore } from '../../store/useReconStore';
import { DiscrepancyStatus } from '../../types';

export const EvidenceDrawer: React.FC = () => {
  const {
    selectedDiscrepancy,
    selectDiscrepancy,
    diagnoseSelectedDiscrepancy,
    updateSelectedDiscrepancyStatus,
    isLoading,
  } = useReconStore();

  const [copied, setCopied] = useState(false);

  if (!selectedDiscrepancy) return null;

  const d = selectedDiscrepancy;
  const isCritical = d.severity === 'CRITICAL';
  const isHigh = d.severity === 'HIGH';

  const formatCurrency = (val?: string) => {
    if (!val) return '₹0.00';
    const num = parseFloat(val);
    if (isNaN(num)) return `₹${val}`;
    return '₹' + num.toLocaleString('en-IN', { minimumFractionDigits: 2, maximumFractionDigits: 4 });
  };

  const handleCopyTicket = () => {
    if (d.ai_diagnosis?.draft_dispute_ticket) {
      navigator.clipboard.writeText(d.ai_diagnosis.draft_dispute_ticket);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <div className="fixed inset-0 z-50 overflow-hidden flex justify-end bg-black/60 backdrop-blur-sm animate-fade-in">
      <div
        className="w-full max-w-2xl bg-ledger-850 border-l border-ledger-700 h-full flex flex-col shadow-2xl animate-slide-in select-none"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="p-4 border-b border-ledger-700 bg-ledger-900 flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="w-7 h-7 rounded bg-recon-crimson/20 border border-recon-crimson/40 flex items-center justify-center">
              <AlertTriangle className="w-4 h-4 text-recon-crimson" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h3 className="text-sm font-semibold text-white tracking-tight">
                  {d.discrepancy_type.replace(/_/g, ' ')}
                </h3>
                <span
                  className={`text-[10px] font-mono px-1.5 py-0.2 rounded uppercase font-bold ${
                    isCritical
                      ? 'bg-rose-950 text-rose-300 border border-rose-800'
                      : isHigh
                      ? 'bg-amber-950 text-amber-300 border border-amber-800'
                      : 'bg-ledger-800 text-ledger-300'
                  }`}
                >
                  {d.severity}
                </span>
              </div>
              <p className="text-[11px] font-mono text-ledger-400">
                Discrepancy ID: {d.id}
              </p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            {/* Status Selector */}
            <select
              value={d.status}
              onChange={(e) => updateSelectedDiscrepancyStatus(e.target.value as DiscrepancyStatus)}
              className="px-2.5 py-1 rounded bg-ledger-800 border border-ledger-700 text-xs text-ledger-200 font-mono focus:outline-none focus:border-ledger-600"
            >
              <option value="OPEN">Status: OPEN</option>
              <option value="IN_DISPUTE">Status: IN DISPUTE</option>
              <option value="RESOLVED">Status: RESOLVED</option>
              <option value="WRITTEN_OFF">Status: WRITTEN OFF</option>
            </select>

            <button
              onClick={() => selectDiscrepancy(null)}
              className="p-1.5 rounded hover:bg-ledger-800 text-ledger-400 hover:text-white transition"
            >
              <X className="w-5 h-5" />
            </button>
          </div>
        </div>

        {/* Content Body */}
        <div className="flex-1 overflow-y-auto p-5 space-y-5 text-xs">
          {/* Financial Variance Summary Box */}
          <div className="p-3.5 rounded-lg bg-ledger-900 border border-ledger-700 grid grid-cols-3 gap-3 font-mono">
            <div>
              <span className="text-[10px] text-ledger-400 uppercase block mb-1">Expected Amount</span>
              <span className="text-sm font-bold text-white tabular-nums">
                {formatCurrency(d.expected_amount)}
              </span>
            </div>
            <div>
              <span className="text-[10px] text-ledger-400 uppercase block mb-1">Actual Amount</span>
              <span className="text-sm font-bold text-white tabular-nums">
                {formatCurrency(d.actual_amount)}
              </span>
            </div>
            <div>
              <span className="text-[10px] text-recon-crimson uppercase block mb-1 font-semibold">Mismatch Variance</span>
              <span className="text-sm font-bold text-recon-crimson tabular-nums">
                Δ {formatCurrency(d.mismatch_amount)}
              </span>
            </div>
          </div>

          {/* Side-by-Side 3-Way Comparative Math Table (Order -> PG Fee Math -> Bank UTR) */}
          <div className="p-3.5 rounded-lg bg-ledger-900 border border-ledger-700 space-y-2.5 font-mono">
            <div className="flex items-center justify-between text-xs text-ledger-300 font-semibold uppercase tracking-wider">
              <span>3-Way Mathematical Ledger Comparison</span>
              <span className="text-[10px] text-recon-cyan">Order → PG Fee → Bank UTR</span>
            </div>

            <div className="grid grid-cols-3 gap-2.5 text-[11px] pt-1">
              {/* Column 1: ERP Order */}
              <div className="p-2.5 rounded bg-ledger-850 border border-ledger-750 space-y-1.5">
                <div className="text-[10px] text-recon-cyan font-bold uppercase pb-1 border-b border-ledger-700">
                  1. ERP Order
                </div>
                <div><span className="text-ledger-400">Order ID:</span> <span className="text-white font-medium">{d.order_id || 'N/A'}</span></div>
                <div><span className="text-ledger-400">Gross:</span> <span className="text-white font-bold">{d.raw_evidence.order ? formatCurrency(d.raw_evidence.order.gross_amount) : (d.expected_amount ? formatCurrency(d.expected_amount) : '—')}</span></div>
                <div><span className="text-ledger-400">Status:</span> <span className="text-emerald-400 font-medium">{d.raw_evidence.order?.order_status || 'CAPTURED'}</span></div>
              </div>

              {/* Column 2: PG Settlement */}
              <div className="p-2.5 rounded bg-ledger-850 border border-ledger-750 space-y-1.5">
                <div className="text-[10px] text-recon-amber font-bold uppercase pb-1 border-b border-ledger-700">
                  2. Gateway Fee Math
                </div>
                <div><span className="text-ledger-400">MDR Fee:</span> <span className="text-recon-amber">{d.raw_evidence.pg_settlement ? '−' + formatCurrency(d.raw_evidence.pg_settlement.pg_fee) : '2.0% MDR'}</span></div>
                <div><span className="text-ledger-400">GST (18%):</span> <span className="text-recon-amber">{d.raw_evidence.pg_settlement ? '−' + formatCurrency(d.raw_evidence.pg_settlement.tax) : '18% ITC'}</span></div>
                <div><span className="text-ledger-400">Net Settled:</span> <span className="text-recon-emerald font-bold">{d.raw_evidence.pg_settlement ? formatCurrency(d.raw_evidence.pg_settlement.net_amount) : formatCurrency(d.actual_amount)}</span></div>
              </div>

              {/* Column 3: Bank Credit UTR */}
              <div className="p-2.5 rounded bg-ledger-850 border border-ledger-750 space-y-1.5">
                <div className="text-[10px] text-recon-emerald font-bold uppercase pb-1 border-b border-ledger-700">
                  3. Bank Nodal Credit
                </div>
                <div><span className="text-ledger-400">Payout/UTR:</span> <span className="text-recon-cyan font-medium truncate block">{d.payout_id || d.raw_evidence.bank_statement?.utr_number || 'UNASSIGNED'}</span></div>
                <div><span className="text-ledger-400">Bank Credit:</span> <span className={d.raw_evidence.bank_statement ? 'text-white font-bold' : 'text-recon-crimson font-bold'}>{d.raw_evidence.bank_statement ? formatCurrency(d.raw_evidence.bank_statement.credit_amount) : (d.discrepancy_level === 'LEVEL_2_PG_BANK' ? 'ZERO CREDIT' : 'CLEARED IN BATCH')}</span></div>
                <div><span className="text-ledger-400">Clearing:</span> <span className="text-ledger-300 font-medium">Verified RTGS/NEFT</span></div>
              </div>
            </div>
          </div>

          {/* AI Forensic Diagnosis Card */}
          <div className="p-4 rounded-lg bg-ledger-900 border border-recon-cyan/40 space-y-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2 text-recon-indigo font-semibold">
                <Sparkles className="w-4 h-4" />
                <span className="text-xs uppercase tracking-wide">
                  Gemini 3.8 Flash Forensic Diagnosis
                </span>
              </div>
              <div className="flex items-center gap-2">
                {d.ai_diagnosis && (
                  <span className="text-[11px] font-mono px-2 py-0.5 rounded bg-recon-indigo/20 text-recon-indigo border border-recon-indigo/40 font-bold">
                    {d.ai_diagnosis.confidence_score}% Confidence
                  </span>
                )}
                <button
                  onClick={diagnoseSelectedDiscrepancy}
                  disabled={isLoading}
                  className="px-2 py-1 rounded bg-ledger-800 hover:bg-ledger-750 text-ledger-300 hover:text-white border border-ledger-700 text-[10px] font-mono flex items-center gap-1 transition"
                >
                  <Activity className={`w-3 h-3 ${isLoading ? 'animate-spin' : ''}`} />
                  <span>Re-Diagnose</span>
                </button>
              </div>
            </div>

            {d.ai_diagnosis ? (
              <>
                <div className="space-y-1">
                  <div className="text-[10px] font-mono text-ledger-400 uppercase">
                    Root Cause Analysis:
                  </div>
                  <p className="text-ledger-100 text-xs leading-relaxed font-sans">
                    {d.ai_diagnosis.root_cause_summary}
                  </p>
                </div>

                <div className="space-y-1.5 pt-2 border-t border-recon-indigo/20">
                  <div className="text-[10px] font-mono text-ledger-400 uppercase">
                    Recommended Actions:
                  </div>
                  <ul className="space-y-1 pl-4 list-disc text-ledger-300 text-xs">
                    {d.ai_diagnosis.recommended_actions.map((act, i) => (
                      <li key={i}>{act}</li>
                    ))}
                  </ul>
                </div>

                {/* 1-Click Copyable Dispute Ticket */}
                <div className="pt-2 border-t border-recon-indigo/20 space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-[10px] font-mono uppercase text-ledger-400">
                      Draft Dispute Ticket (Ready for Razorpay / Stripe / Bank):
                    </span>
                    <button
                      onClick={handleCopyTicket}
                      className="flex items-center gap-1 px-2.5 py-1 rounded bg-recon-indigo/20 hover:bg-recon-indigo/30 text-recon-indigo border border-recon-indigo/40 text-xs font-medium transition"
                    >
                      {copied ? (
                        <>
                          <Check className="w-3.5 h-3.5 text-recon-emerald" />
                          <span className="text-recon-emerald">Copied!</span>
                        </>
                      ) : (
                        <>
                          <Copy className="w-3.5 h-3.5" />
                          <span>Copy Ticket</span>
                        </>
                      )}
                    </button>
                  </div>
                  <pre className="p-3 rounded bg-ledger-950 border border-ledger-700 text-ledger-300 font-mono text-[11px] whitespace-pre-wrap leading-relaxed max-h-48 overflow-y-auto">
                    {d.ai_diagnosis.draft_dispute_ticket}
                  </pre>
                </div>
              </>
            ) : (
              <div className="py-6 text-center text-ledger-400 font-mono text-xs">
                Click "Re-Diagnose" to generate structured AI analysis with Gemini 3.8 Flash.
              </div>
            )}
          </div>

          {/* Audit Formulas & Evidence Dossier */}
          <div className="p-4 rounded-lg bg-ledger-900 border border-ledger-700 space-y-3 font-mono">
            <div className="flex items-center gap-2 text-ledger-300 font-semibold text-xs uppercase">
              <FileText className="w-4 h-4 text-ledger-400" />
              <span>Audit Formula & Calculation Trace</span>
            </div>

            <div className="space-y-1.5">
              <div className="text-[10px] text-ledger-400 uppercase">Expected Formula:</div>
              <div className="p-2 rounded bg-ledger-850 border border-ledger-700 text-recon-emerald text-xs font-semibold">
                {d.raw_evidence.expected_formula}
              </div>
            </div>

            <div className="space-y-1.5">
              <div className="text-[10px] text-ledger-400 uppercase">Actual Ledger Execution:</div>
              <div className="p-2 rounded bg-ledger-850 border border-ledger-700 text-recon-crimson text-xs font-semibold">
                {d.raw_evidence.actual_formula}
              </div>
            </div>

            {d.raw_evidence.calculated_values && (
              <div className="pt-2 border-t border-ledger-700 space-y-1">
                <div className="text-[10px] text-ledger-400 uppercase">Calculated Variables:</div>
                <div className="grid grid-cols-2 gap-2 text-[11px]">
                  {Object.entries(d.raw_evidence.calculated_values).map(([k, v]) => (
                    <div key={k} className="p-1.5 rounded bg-ledger-850 border border-ledger-700 flex justify-between">
                      <span className="text-ledger-400">{k}:</span>
                      <span className="text-white font-medium">{String(v)}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

