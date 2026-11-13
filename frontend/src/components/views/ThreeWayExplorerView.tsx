import React from 'react';
import {
  Search,
  CheckCircle2,
  AlertTriangle,
  ArrowRight,
  ExternalLink,
  Filter,
} from 'lucide-react';
import { useReconStore } from '../../store/useReconStore';
import { ThreeWayTraceItem } from '../../types';

export const ThreeWayExplorerView: React.FC = () => {
  const {
    traces,
    tracesTotal,
    searchQuery,
    setSearchQuery,
    statusFilter,
    setStatusFilter,
    selectDiscrepancy,
    summary,
  } = useReconStore();

  const formatCurrency = (val?: string) => {
    if (!val) return '—';
    const num = parseFloat(val);
    if (isNaN(num)) return `₹${val}`;
    return '₹' + num.toLocaleString('en-IN', { minimumFractionDigits: 2, maximumFractionDigits: 4 });
  };

  return (
    <div className="flex-1 flex flex-col h-full overflow-hidden bg-ledger-900">
      {/* Table Controls & Filter Toolbar */}
      <div className="p-3 border-b border-ledger-700 bg-ledger-850 flex items-center justify-between gap-4">
        <div className="flex items-center gap-3 flex-1 max-w-lg">
          <div className="relative flex-1">
            <Search className="w-3.5 h-3.5 text-ledger-400 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Filter by Order ID, PG Txn ID, or Bank UTR..."
              className="w-full pl-9 pr-3 py-1.5 rounded bg-ledger-900 border border-ledger-700 text-xs text-white placeholder-ledger-400 focus:outline-none focus:border-recon-emerald font-mono transition"
            />
          </div>

          {/* Quick status filters */}
          <div className="flex items-center rounded bg-ledger-900 p-0.5 border border-ledger-700 text-xs">
            <button
              onClick={() => setStatusFilter('all')}
              className={`px-2.5 py-1 rounded text-[11px] font-medium transition ${
                statusFilter === 'all'
                  ? 'bg-ledger-750 text-white shadow-sm'
                  : 'text-ledger-400 hover:text-white'
              }`}
            >
              All ({summary?.total_orders || tracesTotal})
            </button>
            <button
              onClick={() => setStatusFilter('reconciled')}
              className={`px-2.5 py-1 rounded text-[11px] font-medium transition flex items-center gap-1.5 ${
                statusFilter === 'reconciled'
                  ? 'bg-ledger-750 text-recon-emerald shadow-sm'
                  : 'text-ledger-400 hover:text-recon-emerald'
              }`}
            >
              <CheckCircle2 className="w-3 h-3 text-recon-emerald" />
              <span>Matched ({summary?.reconciled_orders_count || 0})</span>
            </button>
            <button
              onClick={() => setStatusFilter('discrepancy')}
              className={`px-2.5 py-1 rounded text-[11px] font-medium transition flex items-center gap-1.5 ${
                statusFilter === 'discrepancy'
                  ? 'bg-ledger-750 text-recon-crimson shadow-sm'
                  : 'text-ledger-400 hover:text-recon-crimson'
              }`}
            >
              <AlertTriangle className="w-3 h-3 text-recon-crimson" />
              <span>Discrepancies ({summary?.unreconciled_orders_count || 0})</span>
            </button>
          </div>
        </div>

        <div className="text-xs font-mono text-ledger-400 flex items-center gap-2">
          <Filter className="w-3.5 h-3.5" />
          <span>Showing {(traces || []).length} of {tracesTotal || 0} traces</span>
        </div>
      </div>

      {/* Main 3-Way Grid */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-left text-xs border-collapse">
          <thead className="bg-ledger-850/90 backdrop-blur sticky top-0 z-10 border-b border-ledger-700 text-ledger-400 font-mono text-[11px] uppercase tracking-wider select-none">
            <tr>
              <th className="py-2.5 px-4 w-[24%] border-r border-ledger-700">
                1. Internal Merchant Order
              </th>
              <th className="py-2.5 px-4 w-[34%] border-r border-ledger-700">
                2. Gateway Settlement (Gross − Fee − GST = Net)
              </th>
              <th className="py-2.5 px-4 w-[24%] border-r border-ledger-700">
                3. Bank Remittance (UTR Credit)
              </th>
              <th className="py-2.5 px-4 w-[18%] text-right">
                Verdict & Audit
              </th>
            </tr>
          </thead>

          <tbody className="divide-y divide-ledger-700/60 font-mono">
            {(traces || []).length === 0 ? (
              <tr>
                <td colSpan={4} className="py-16 text-center text-ledger-400">
                  <div className="max-w-sm mx-auto space-y-1.5 font-sans">
                    <p className="font-semibold text-white text-xs">No Ledger Transactions Found</p>
                    <p className="text-[11px] text-ledger-400 leading-relaxed">
                      Upload your Orders, Payment Gateway Settlements, or Bank Statement CSV files in Streaming Ingestion to run real-time 3-way reconciliation.
                    </p>
                  </div>
                </td>
              </tr>
            ) : (
              (traces || []).map((trace, idx) => {
                const hasDiscrepancy = !trace.is_reconciled && trace.discrepancy;
                return (
                  <tr
                    key={trace.order_id || idx}
                    className={`hover:bg-ledger-800/60 transition group cursor-pointer ${
                      hasDiscrepancy ? 'bg-recon-crimson/5 hover:bg-recon-crimson/10' : ''
                    }`}
                    onClick={() => {
                      if (trace.discrepancy) {
                        selectDiscrepancy(trace.discrepancy);
                      }
                    }}
                  >
                    {/* Column 1: Internal Order */}
                    <td className="py-2.5 px-4 border-r border-ledger-700/60 align-top">
                      <div className="font-semibold text-white tracking-tight flex items-center gap-2">
                        <span>{trace.order_id}</span>
                        {trace.order_status && (
                          <span
                            className={`text-[9px] px-1 py-0.2 rounded font-sans uppercase font-medium ${
                              trace.order_status === 'CAPTURED'
                                ? 'bg-emerald-950/80 text-emerald-400 border border-emerald-800/40'
                                : trace.order_status === 'REFUNDED'
                                ? 'bg-amber-950/80 text-amber-400 border border-amber-800/40'
                                : 'bg-rose-950/80 text-rose-400 border border-rose-800/40'
                            }`}
                          >
                            {trace.order_status}
                          </span>
                        )}
                      </div>
                      <div className="text-sm font-bold text-ledger-100 tabular-nums mt-0.5">
                        {formatCurrency(trace.order_gross)}
                      </div>
                    </td>

                    {/* Column 2: Payment Gateway Settlement */}
                    <td className="py-2.5 px-4 border-r border-ledger-700/60 align-top">
                      {trace.pg_txn_id ? (
                        <div>
                          <div className="flex items-center justify-between text-ledger-300">
                            <span className="text-ledger-300 text-[11px] font-medium">
                              {trace.pg_txn_id}
                            </span>
                            {trace.pg_payout_id && (
                              <span className="text-[10px] px-1 py-0.2 rounded bg-ledger-800 text-recon-cyan border border-ledger-700">
                                {trace.pg_payout_id}
                              </span>
                            )}
                          </div>

                          <div className="flex items-center gap-1.5 text-[11px] text-ledger-300 mt-1 tabular-nums">
                            <span className="text-white font-medium">{formatCurrency(trace.pg_gross)}</span>
                            <span className="text-ledger-400">−</span>
                            <span className="text-recon-amber" title="Payment Gateway Fee">
                              {formatCurrency(trace.pg_fee)}
                            </span>
                            <span className="text-ledger-400">−</span>
                            <span className="text-ledger-400" title="18% Statutory GST">
                              {formatCurrency(trace.pg_tax)}
                            </span>
                            <ArrowRight className="w-3 h-3 text-ledger-400" />
                            <span className="font-bold text-recon-emerald">
                              {formatCurrency(trace.pg_net)} Net
                            </span>
                          </div>
                        </div>
                      ) : (
                        <div className="text-recon-crimson text-[11px] flex items-center gap-1.5">
                          <AlertTriangle className="w-3.5 h-3.5" />
                          <span>NO GATEWAY SETTLEMENT RECORD FOUND</span>
                        </div>
                      )}
                    </td>

                    {/* Column 3: Bank Remittance */}
                    <td className="py-2.5 px-4 border-r border-ledger-700/60 align-top">
                      {trace.bank_utr ? (
                        <div>
                          <div className="text-[11px] text-recon-cyan font-semibold truncate">
                            {trace.bank_utr}
                          </div>
                          <div className="text-sm font-bold text-recon-emerald tabular-nums mt-0.5">
                            {formatCurrency(trace.bank_credit)}
                          </div>
                        </div>
                      ) : (
                        <div className="text-recon-crimson text-[11px] flex items-center gap-1.5">
                          <AlertTriangle className="w-3.5 h-3.5" />
                          <span>UNCREDITED (NO BANK UTR)</span>
                        </div>
                      )}
                    </td>

                    {/* Column 4: Verdict & Audit Trigger */}
                    <td className="py-2.5 px-4 text-right align-top">
                      {trace.is_reconciled ? (
                        <div className="inline-flex items-center gap-1.5 px-2 py-1 rounded bg-emerald-950/60 text-emerald-400 border border-emerald-800/50 text-[11px] font-sans font-semibold">
                          <CheckCircle2 className="w-3.5 h-3.5" />
                          <span>RECONCILED</span>
                        </div>
                      ) : (
                        <div className="inline-flex flex-col items-end gap-1">
                          <div className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded bg-rose-950/80 text-rose-300 border border-rose-800/60 text-[10px] font-sans font-bold shadow-sm animate-pulse-subtle">
                            <AlertTriangle className="w-3 h-3 text-recon-crimson" />
                            <span>{trace.discrepancy?.discrepancy_type.replace(/_/g, ' ') || 'DISCREPANCY'}</span>
                          </div>
                          {trace.discrepancy && (
                            <div className="flex items-center gap-1 text-[11px] text-recon-crimson font-bold">
                              <span>Δ {formatCurrency(trace.discrepancy.mismatch_amount)}</span>
                              <ExternalLink className="w-3 h-3 text-ledger-400 group-hover:text-white transition" />
                            </div>
                          )}
                        </div>
                      )}
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
};
