import React from 'react';
import {
  Layers,
  CheckCircle2,
  AlertTriangle,
  ArrowRight,
  Landmark,
  ShieldAlert,
} from 'lucide-react';
import { useReconStore } from '../../store/useReconStore';

export const PayoutBatchesView: React.FC = () => {
  const { payoutBatches, selectDiscrepancy } = useReconStore();

  const formatCurrency = (val?: string) => {
    if (!val) return '—';
    const num = parseFloat(val);
    if (isNaN(num)) return `₹${val}`;
    return '₹' + num.toLocaleString('en-IN', { minimumFractionDigits: 2, maximumFractionDigits: 4 });
  };

  return (
    <div className="flex-1 flex flex-col h-full overflow-hidden bg-ledger-900">
      {/* Header bar */}
      <div className="p-4 border-b border-ledger-700 bg-ledger-850 flex items-center justify-between">
        <div>
          <div className="flex items-center gap-2">
            <Layers className="w-4 h-4 text-recon-cyan" />
            <h2 className="text-sm font-semibold text-white">Level-2 Payout Batch & Bank UTR Reconciliation</h2>
          </div>
          <p className="text-xs text-ledger-400 mt-0.5">
            Aggregates Payment Gateway Net settlements by Payout ID and reconciles against Bank Statement inward UTR remittances.
          </p>
        </div>
        <div className="text-xs font-mono text-ledger-300 bg-ledger-900 px-3 py-1.5 rounded border border-ledger-700">
          Total Batches: <span className="text-white font-bold">{payoutBatches.length}</span>
        </div>
      </div>

      {/* Grid list of payout batches */}
      <div className="flex-1 overflow-auto p-4 space-y-3">
        {payoutBatches.length === 0 ? (
          <div className="py-16 text-center text-ledger-400 font-mono text-xs">
            No payout batches detected. Run reconciliation or seed demo data.
          </div>
        ) : (
          payoutBatches.map((batch) => {
            const isMatched = batch.status === 'MATCHED';
            const isMissing = batch.status === 'BANK_MISSING';
            const isShortfall = batch.status === 'SHORTFALL';

            return (
              <div
                key={batch.payout_id}
                onClick={() => {
                  if (batch.discrepancy) {
                    selectDiscrepancy(batch.discrepancy);
                  }
                }}
                className={`p-4 rounded-lg border bg-ledger-850 transition cursor-pointer hover:border-ledger-600 ${
                  isMatched
                    ? 'border-ledger-700'
                    : 'border-recon-crimson/50 bg-recon-crimson/5 hover:bg-recon-crimson/10'
                }`}
              >
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-3">
                    <span className="font-mono text-sm font-bold text-white tracking-tight">
                      {batch.payout_id}
                    </span>
                    <span className="text-[11px] font-mono px-2 py-0.5 rounded bg-ledger-800 text-ledger-300 border border-ledger-700">
                      {batch.settlements_count} settlements
                    </span>
                  </div>

                  {/* Status badge */}
                  <div>
                    {isMatched ? (
                      <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded bg-emerald-950/80 text-emerald-400 border border-emerald-800/60 text-xs font-medium">
                        <CheckCircle2 className="w-3.5 h-3.5" />
                        <span>BANK UTR MATCHED</span>
                      </span>
                    ) : isMissing ? (
                      <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded bg-rose-950/90 text-rose-300 border border-rose-800/80 text-xs font-bold animate-pulse-subtle">
                        <ShieldAlert className="w-3.5 h-3.5 text-recon-crimson" />
                        <span>BANK CREDIT MISSING</span>
                      </span>
                    ) : isShortfall ? (
                      <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded bg-rose-950/90 text-rose-300 border border-rose-800/80 text-xs font-bold">
                        <AlertTriangle className="w-3.5 h-3.5 text-recon-crimson" />
                        <span>PAYOUT SHORTFALL</span>
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded bg-amber-950/80 text-amber-300 border border-amber-800/60 text-xs font-medium">
                        <AlertTriangle className="w-3.5 h-3.5 text-recon-amber" />
                        <span>SURPLUS CREDIT</span>
                      </span>
                    )}
                  </div>
                </div>

                {/* Arithmetic breakdown strip */}
                <div className="grid grid-cols-5 gap-3 mt-3 pt-3 border-t border-ledger-700/60 text-xs font-mono">
                  <div>
                    <span className="text-ledger-400 block text-[10px] uppercase">Gross Total</span>
                    <span className="font-semibold text-white">{formatCurrency(batch.total_gross)}</span>
                  </div>
                  <div>
                    <span className="text-ledger-400 block text-[10px] uppercase">Total Fees & Tax</span>
                    <span className="text-recon-amber">
                      −{formatCurrency(batch.total_fee)} (Tax: {formatCurrency(batch.total_tax)})
                    </span>
                  </div>
                  <div>
                    <span className="text-ledger-400 block text-[10px] uppercase">Expected Net Payout</span>
                    <span className="font-bold text-recon-emerald">{formatCurrency(batch.total_expected_net)}</span>
                  </div>
                  <div>
                    <span className="text-ledger-400 block text-[10px] uppercase">Bank Credited</span>
                    <div className="flex items-center gap-1 mt-0.5">
                      <Landmark className="w-3 h-3 text-recon-cyan" />
                      <span className={batch.bank_credit ? 'font-bold text-white' : 'text-recon-crimson font-bold'}>
                        {batch.bank_credit ? formatCurrency(batch.bank_credit) : '₹0.00'}
                      </span>
                    </div>
                  </div>
                  <div>
                    <span className="text-ledger-400 block text-[10px] uppercase">Variance (Delta)</span>
                    <span className={`font-bold ${isMatched ? 'text-recon-emerald' : 'text-recon-crimson'}`}>
                      {batch.variance ? formatCurrency(batch.variance) : (isMatched ? '₹0.00' : '−' + formatCurrency(batch.total_expected_net))}
                    </span>
                  </div>
                </div>

                {/* Bank reference info */}
                {batch.bank_utr && (
                  <div className="mt-2 text-[11px] font-mono text-ledger-400 flex items-center gap-2">
                    <span>Nodal Bank Reference UTR:</span>
                    <span className="text-recon-cyan font-semibold">{batch.bank_utr}</span>
                  </div>
                )}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
};

