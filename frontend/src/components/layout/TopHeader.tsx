import React, { useState } from 'react';
import {
  Play,
  RefreshCw,
  Download,
  Search,
  Database,
  ChevronDown,
  Sparkles,
  FileSpreadsheet,
  FileText,
  Activity,
} from 'lucide-react';
import { useReconStore } from '../../store/useReconStore';
import { apiClient } from '../../api/client';

export const TopHeader: React.FC = () => {
  const {
    isReconciling,
    triggerReconcile,
    seedData,
    setCommandPaletteOpen,
    toggleTelemetryDock,
    currentStage,
  } = useReconStore();

  const [exportOpen, setExportOpen] = useState(false);

  return (
    <header className="h-14 border-b border-ledger-700 bg-ledger-850 px-4 flex items-center justify-between select-none">
      {/* Brand & Environment */}
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-2">
          <div className="w-7 h-7 rounded bg-gradient-to-br from-recon-emerald/20 to-recon-indigo/30 border border-recon-emerald/40 flex items-center justify-center">
            <Activity className="w-4 h-4 text-recon-emerald animate-pulse" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="font-semibold tracking-tight text-white text-sm">ReconPilot</span>
              <span className="text-[10px] font-mono px-1.5 py-0.5 rounded bg-ledger-800 text-recon-emerald border border-ledger-700">
                v1.0
              </span>
            </div>
            <p className="text-[10px] font-mono text-ledger-400">3-Way Financial Reconciliation Engine</p>
          </div>
        </div>

        <div className="h-5 w-px bg-ledger-700 mx-2" />

        {/* Live Database/Engine Indicator */}
        <div className="flex items-center gap-2 px-2.5 py-1 rounded bg-ledger-900 border border-ledger-700 text-xs">
          <Database className="w-3.5 h-3.5 text-recon-indigo" />
          <span className="text-ledger-300 font-mono text-[11px]">HYBRID REPO:</span>
          <span className="inline-flex items-center gap-1.5 text-recon-emerald font-medium text-[11px]">
            <span className="w-1.5 h-1.5 rounded-full bg-recon-emerald animate-ping" />
            ACTIVE
          </span>
        </div>

        {/* Real-time telemetry dock toggle pill */}
        <button
          onClick={toggleTelemetryDock}
          className="flex items-center gap-1.5 px-2.5 py-1 rounded bg-ledger-800 hover:bg-ledger-750 border border-ledger-700 text-ledger-300 hover:text-white text-xs transition"
        >
          <Activity className="w-3.5 h-3.5 text-recon-cyan" />
          <span className="font-mono text-[11px] uppercase">{currentStage}</span>
        </button>
      </div>

      {/* Center Search / Command Trigger */}
      <div className="flex-1 max-w-md mx-6">
        <button
          onClick={() => setCommandPaletteOpen(true)}
          className="w-full flex items-center justify-between px-3 py-1.5 rounded-md bg-ledger-900 border border-ledger-700 text-ledger-400 hover:text-ledger-300 hover:border-ledger-600 text-xs transition shadow-inner"
        >
          <div className="flex items-center gap-2">
            <Search className="w-3.5 h-3.5 text-ledger-400" />
            <span>Search orders, settlements, UTRs, discrepancies...</span>
          </div>
          <kbd className="px-1.5 py-0.5 text-[10px] font-mono bg-ledger-800 border border-ledger-700 rounded text-ledger-400">
            ⌘K
          </kbd>
        </button>
      </div>

      {/* Right Action Bar */}
      <div className="flex items-center gap-2">
        {/* Seed Demo Data Button */}
        <button
          onClick={() => seedData()}
          disabled={isReconciling}
          title="Seed realistic Indian (INR/Razorpay/HDFC) and Global (USD/Stripe/JPMC) demo datasets"
          className="flex items-center gap-1.5 px-3 py-1.5 rounded bg-ledger-800 hover:bg-ledger-750 text-ledger-300 hover:text-white border border-ledger-700 text-xs font-medium transition disabled:opacity-50"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${isReconciling ? 'animate-spin' : ''}`} />
          <span>Seed Demo</span>
        </button>

        {/* Export Dropdown */}
        <div className="relative">
          <button
            onClick={() => setExportOpen(!exportOpen)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded bg-ledger-800 hover:bg-ledger-750 text-ledger-300 hover:text-white border border-ledger-700 text-xs font-medium transition"
          >
            <Download className="w-3.5 h-3.5" />
            <span>Export</span>
            <ChevronDown className="w-3 h-3 text-ledger-400" />
          </button>

          {exportOpen && (
            <div className="absolute right-0 mt-1 w-48 bg-ledger-850 border border-ledger-700 rounded-md shadow-2xl py-1 z-50 text-xs">
              <button
                onClick={() => {
                  apiClient.exportReport('csv', 'traces');
                  setExportOpen(false);
                }}
                className="w-full flex items-center gap-2 px-3 py-2 text-ledger-300 hover:text-white hover:bg-ledger-800 text-left"
              >
                <FileSpreadsheet className="w-4 h-4 text-recon-emerald" />
                <span>3-Way Traces (CSV)</span>
              </button>
              <button
                onClick={() => {
                  apiClient.exportReport('csv', 'discrepancies');
                  setExportOpen(false);
                }}
                className="w-full flex items-center gap-2 px-3 py-2 text-ledger-300 hover:text-white hover:bg-ledger-800 text-left"
              >
                <FileSpreadsheet className="w-4 h-4 text-recon-crimson" />
                <span>Discrepancy Audit (CSV)</span>
              </button>
              <button
                onClick={() => {
                  apiClient.exportReport('json', 'traces');
                  setExportOpen(false);
                }}
                className="w-full flex items-center gap-2 px-3 py-2 text-ledger-300 hover:text-white hover:bg-ledger-800 text-left"
              >
                <FileText className="w-4 h-4 text-recon-indigo" />
                <span>Full Ledger (JSON)</span>
              </button>
            </div>
          )}
        </div>

        {/* Primary Run Reconcile Trigger */}
        <button
          onClick={() => triggerReconcile()}
          disabled={isReconciling}
          className="flex items-center gap-1.5 px-3.5 py-1.5 rounded bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white text-xs font-semibold shadow-lg shadow-emerald-950/40 border border-emerald-400/30 transition disabled:opacity-50"
        >
          {isReconciling ? (
            <>
              <RefreshCw className="w-3.5 h-3.5 animate-spin" />
              <span>Matching...</span>
            </>
          ) : (
            <>
              <Play className="w-3.5 h-3.5 fill-current" />
              <span>Run Reconcile</span>
            </>
          )}
        </button>
      </div>
    </header>
  );
};

