import React from 'react';
import {
  TrendingUp,
  Percent,
  CheckCircle2,
  AlertCircle,
  Landmark,
  ArrowRight,
} from 'lucide-react';
import { useReconStore } from '../../store/useReconStore';

export const KPIStrip: React.FC = () => {
  const { summary } = useReconStore();

  const formatCurrency = (val?: string) => {
    if (!val) return '₹0.00';
    const num = parseFloat(val);
    if (isNaN(num)) return `₹${val}`;
    return '₹' + num.toLocaleString('en-IN', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
  };

  const matchRateNum = summary ? parseFloat(summary.three_way_match_rate) : 0;
  const isHealthy = matchRateNum >= 95;

  return (
    <div className="grid grid-cols-6 gap-3 p-4 bg-ledger-900 border-b border-ledger-700 select-none">
      {/* 1. Gross Volume */}
      <div className="bg-ledger-850 p-3 rounded-lg border border-ledger-700 hover:border-ledger-600 transition shadow-sm">
        <div className="flex items-center justify-between text-ledger-400 text-xs mb-1.5">
          <span className="font-medium">Gross Order Volume</span>
          <TrendingUp className="w-3.5 h-3.5 text-ledger-400" />
        </div>
        <div className="text-lg font-mono font-bold text-white tabular-nums tracking-tight">
          {formatCurrency(summary?.total_gross_order_volume)}
        </div>
        <div className="text-[11px] font-mono text-ledger-400 mt-1 flex items-center gap-1.5">
          <span className="text-white font-medium">{summary?.total_orders || 0}</span> orders captured
        </div>
      </div>

      {/* 2. Gateway Deductions (Fee + Tax) */}
      <div className="bg-ledger-850 p-3 rounded-lg border border-ledger-700 hover:border-ledger-600 transition shadow-sm">
        <div className="flex items-center justify-between text-ledger-400 text-xs mb-1.5">
          <span className="font-medium">Gateway Fees & Tax</span>
          <Percent className="w-3.5 h-3.5 text-recon-amber" />
        </div>
        <div className="text-lg font-mono font-bold text-white tabular-nums tracking-tight">
          {formatCurrency(summary?.total_pg_fees)}
        </div>
        <div className="text-[11px] font-mono text-recon-amber mt-1 flex items-center justify-between">
          <span>GST: {formatCurrency(summary?.total_pg_tax)}</span>
          <span className="text-ledger-400">18% ITC</span>
        </div>
      </div>

      {/* 3. Expected Net Remittance */}
      <div className="bg-ledger-850 p-3 rounded-lg border border-ledger-700 hover:border-ledger-600 transition shadow-sm">
        <div className="flex items-center justify-between text-ledger-400 text-xs mb-1.5">
          <span className="font-medium">Expected Net Payout</span>
          <ArrowRight className="w-3.5 h-3.5 text-recon-cyan" />
        </div>
        <div className="text-lg font-mono font-bold text-white tabular-nums tracking-tight">
          {formatCurrency(summary?.expected_net_payout)}
        </div>
        <div className="text-[11px] font-mono text-ledger-400 mt-1">
          Gross − Fee − Statutory GST
        </div>
      </div>

      {/* 4. Bank Settled Credits */}
      <div className="bg-ledger-850 p-3 rounded-lg border border-ledger-700 hover:border-ledger-600 transition shadow-sm">
        <div className="flex items-center justify-between text-ledger-400 text-xs mb-1.5">
          <span className="font-medium">Bank Settled Credits</span>
          <Landmark className="w-3.5 h-3.5 text-recon-emerald" />
        </div>
        <div className="text-lg font-mono font-bold text-white tabular-nums tracking-tight">
          {formatCurrency(summary?.total_bank_credits)}
        </div>
        <div className="text-[11px] font-mono text-recon-emerald mt-1 flex items-center gap-1">
          <CheckCircle2 className="w-3 h-3" />
          <span>Cleared in Nodal Accounts</span>
        </div>
      </div>

      {/* 5. Net Discrepancy Variance */}
      <div className="bg-ledger-850 p-3 rounded-lg border border-ledger-700 hover:border-ledger-600 transition shadow-sm relative overflow-hidden">
        {summary && parseFloat(summary.net_discrepancy_variance) > 0 && (
          <div className="absolute top-0 right-0 w-2 h-full bg-recon-crimson" />
        )}
        <div className="flex items-center justify-between text-ledger-400 text-xs mb-1.5">
          <span className="font-medium">Discrepancy Variance</span>
          <AlertCircle className="w-3.5 h-3.5 text-recon-crimson" />
        </div>
        <div className="text-lg font-mono font-bold text-recon-crimson tabular-nums tracking-tight">
          {formatCurrency(summary?.net_discrepancy_variance)}
        </div>
        <div className="text-[11px] font-mono text-ledger-400 mt-1 flex items-center justify-between">
          <span>L1: {summary?.level_1_discrepancies_count || 0}</span>
          <span>L2: {summary?.level_2_discrepancies_count || 0}</span>
        </div>
      </div>

      {/* 6. 3-Way Match Rate */}
      <div className="bg-ledger-850 p-3 rounded-lg border border-ledger-700 hover:border-ledger-600 transition shadow-sm">
        <div className="flex items-center justify-between text-ledger-400 text-xs mb-1.5">
          <span className="font-medium">3-Way Match Rate</span>
          <span
            className={`w-2 h-2 rounded-full ${
              isHealthy ? 'bg-recon-emerald' : 'bg-recon-crimson animate-ping'
            }`}
          />
        </div>
        <div className="flex items-baseline gap-2">
          <span className="text-2xl font-mono font-bold text-white tabular-nums tracking-tight">
            {summary ? parseFloat(summary.three_way_match_rate).toFixed(2) : '0.00'}%
          </span>
          <span className="text-[10px] font-mono text-ledger-400">
            {summary?.reconciled_orders_count || 0} / {summary?.total_orders || 0}
          </span>
        </div>
        {/* Progress bar */}
        <div className="w-full bg-ledger-800 rounded-full h-1.5 mt-2 overflow-hidden border border-ledger-700">
          <div
            className={`h-full transition-all duration-500 ${
              isHealthy ? 'bg-recon-emerald' : 'bg-recon-crimson'
            }`}
            style={{ width: `${Math.min(matchRateNum, 100)}%` }}
          />
        </div>
      </div>
    </div>
  );
};

