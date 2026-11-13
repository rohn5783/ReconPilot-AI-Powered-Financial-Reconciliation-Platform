import React from 'react';
import {
  Table2,
  Layers,
  AlertTriangle,
  UploadCloud,
  Cpu,
  ShieldCheck,
  Scale,
} from 'lucide-react';
import { useReconStore, WorkstationView } from '../../store/useReconStore';

export const Sidebar: React.FC = () => {
  const {
    activeView,
    setActiveView,
    summary,
    discrepancies,
    payoutBatches,
  } = useReconStore();

  const navItems: Array<{
    id: WorkstationView;
    label: string;
    icon: React.ReactNode;
    badge?: number | string;
    badgeColor?: string;
  }> = [
    {
      id: 'explorer',
      label: '3-Way Trace Ledger',
      icon: <Table2 className="w-4 h-4" />,
      badge: summary?.total_orders,
      badgeColor: 'bg-ledger-800 text-ledger-300',
    },
    {
      id: 'batches',
      label: 'Payout Batches (L2)',
      icon: <Layers className="w-4 h-4" />,
      badge: (payoutBatches || []).length,
      badgeColor: 'bg-ledger-800 text-recon-cyan',
    },
    {
      id: 'discrepancies',
      label: 'Discrepancy Audits',
      icon: <AlertTriangle className="w-4 h-4" />,
      badge: (discrepancies || []).length,
      badgeColor: 'bg-recon-crimson/20 text-recon-crimson border border-recon-crimson/30',
    },
    {
      id: 'upload',
      label: 'Streaming Ingestion',
      icon: <UploadCloud className="w-4 h-4" />,
    },
  ];

  return (
    <aside className="w-60 border-r border-ledger-700 bg-ledger-850 flex flex-col justify-between select-none">
      {/* Navigation Links */}
      <div className="p-3 space-y-1">
        <div className="px-3 py-2 text-[10px] font-mono uppercase tracking-wider text-ledger-400">
          Reconciliation Modules
        </div>

        {navItems.map((item) => {
          const isActive = activeView === item.id;
          return (
            <button
              key={item.id}
              onClick={() => setActiveView(item.id)}
              className={`w-full flex items-center justify-between px-3 py-2 rounded-md text-xs font-medium transition ${
                isActive
                  ? 'bg-ledger-750 text-white border border-ledger-600 shadow-sm'
                  : 'text-ledger-300 hover:text-white hover:bg-ledger-800'
              }`}
            >
              <div className="flex items-center gap-2.5">
                <span className={isActive ? 'text-recon-emerald' : 'text-ledger-400'}>
                  {item.icon}
                </span>
                <span>{item.label}</span>
              </div>
              {item.badge !== undefined && (
                <span
                  className={`px-1.5 py-0.2 rounded font-mono text-[10px] ${
                    item.badgeColor || 'bg-ledger-800 text-ledger-400'
                  }`}
                >
                  {item.badge}
                </span>
              )}
            </button>
          );
        })}
      </div>

      {/* Contract Invariants & Engine Parameters Widget */}
      <div className="p-3 border-t border-ledger-700 bg-ledger-900/50 space-y-2.5 text-[11px] font-mono">
        <div className="flex items-center gap-2 text-ledger-400">
          <Scale className="w-3.5 h-3.5 text-ledger-400" />
          <span className="uppercase tracking-wider text-[10px]">Contract Invariants</span>
        </div>

        <div className="bg-ledger-850 rounded p-2 border border-ledger-700 space-y-1.5 text-ledger-300 text-[10px]">
          <div className="flex justify-between items-center">
            <span className="text-ledger-400">Agreed MDR:</span>
            <span className="text-white font-semibold">2.00%</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-ledger-400">Statutory GST:</span>
            <span className="text-white font-semibold">18.00%</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-ledger-400">Fee Tolerance:</span>
            <span className="text-recon-emerald font-semibold">±₹0.0100</span>
          </div>
          <div className="flex justify-between items-center">
            <span className="text-ledger-400">Rounding:</span>
            <span className="text-white">Half-Even</span>
          </div>
        </div>

        {/* AI Engine Status */}
        <div className="flex items-center justify-between px-2 py-1.5 rounded bg-ledger-800 border border-ledger-700">
          <div className="flex items-center gap-1.5 text-ledger-300">
            <Cpu className="w-3.5 h-3.5 text-recon-indigo" />
            <span className="text-[10px]">Gemini 3.8 Flash</span>
          </div>
          <ShieldCheck className="w-3.5 h-3.5 text-recon-emerald" />
        </div>
      </div>
    </aside>
  );
};

