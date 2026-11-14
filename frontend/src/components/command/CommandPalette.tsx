import React, { useEffect } from 'react';
import { Command } from 'cmdk';
import {
  Play,
  RefreshCw,
  Table2,
  Layers,
  AlertTriangle,
  UploadCloud,
  FileSpreadsheet,
  FileText,
  Activity,
  CheckCircle2,
  X,
  Trash2,
  History,
  Scale,
} from 'lucide-react';
import { useReconStore } from '../../store/useReconStore';
import { apiClient } from '../../api/client';

export const CommandPalette: React.FC = () => {
  const {
    isCommandPaletteOpen,
    setCommandPaletteOpen,
    setActiveView,
    setStatusFilter,
    triggerReconcile,
    resetData,
    seedData,
    toggleTelemetryDock,
  } = useReconStore();

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setCommandPaletteOpen(!isCommandPaletteOpen);
      }
      if (e.key === 'Escape' && isCommandPaletteOpen) {
        setCommandPaletteOpen(false);
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [isCommandPaletteOpen, setCommandPaletteOpen]);

  if (!isCommandPaletteOpen) return null;

  return (
    <div className="fixed inset-0 z-50 bg-black/70 backdrop-blur-sm flex items-start justify-center pt-24 p-4 animate-fade-in">
      <div className="w-full max-w-xl bg-ledger-850 border border-ledger-700 rounded-xl shadow-2xl overflow-hidden flex flex-col">
        <Command className="w-full">
          {/* Search Header */}
          <div className="flex items-center px-4 border-b border-ledger-700">
            <Command.Input
              autoFocus
              placeholder="Type a command, query an order, or run reconciliation..."
              className="w-full py-3 bg-transparent text-sm text-white placeholder-ledger-400 focus:outline-none font-mono"
            />
            <button
              onClick={() => setCommandPaletteOpen(false)}
              className="p-1 text-ledger-400 hover:text-white"
            >
              <X className="w-4 h-4" />
            </button>
          </div>

          {/* List Options */}
          <Command.List className="max-h-80 overflow-y-auto p-2 space-y-1 text-xs">
            <Command.Empty className="py-6 text-center text-ledger-400">
              No matching commands found.
            </Command.Empty>

            {/* Quick Actions */}
            <Command.Group heading="Engine Actions" className="text-[10px] font-mono uppercase text-ledger-400 px-2 py-1.5">
              <Command.Item
                onSelect={() => {
                  triggerReconcile();
                  setCommandPaletteOpen(false);
                }}
                className="flex items-center gap-2.5 px-3 py-2 rounded-md text-ledger-200 hover:text-white hover:bg-ledger-800 cursor-pointer"
              >
                <Play className="w-4 h-4 text-recon-emerald fill-current" />
                <span>Run 3-Way Reconciliation Engine</span>
              </Command.Item>

              <Command.Item
                onSelect={() => {
                  resetData();
                  setCommandPaletteOpen(false);
                }}
                className="flex items-center gap-2.5 px-3 py-2 rounded-md text-recon-crimson hover:text-white hover:bg-recon-crimson/20 cursor-pointer"
              >
                <Trash2 className="w-4 h-4 text-recon-crimson" />
                <span>Reset & Erase All Ledger Data</span>
              </Command.Item>

              <Command.Item
                onSelect={() => {
                  seedData();
                  setCommandPaletteOpen(false);
                }}
                className="flex items-center gap-2.5 px-3 py-2 rounded-md text-ledger-200 hover:text-white hover:bg-ledger-800 cursor-pointer"
              >
                <RefreshCw className="w-4 h-4 text-recon-indigo" />
                <span>Reset & Seed Demo Financial Ledger (INR/USD)</span>
              </Command.Item>

              <Command.Item
                onSelect={() => {
                  toggleTelemetryDock();
                  setCommandPaletteOpen(false);
                }}
                className="flex items-center gap-2.5 px-3 py-2 rounded-md text-ledger-200 hover:text-white hover:bg-ledger-800 cursor-pointer"
              >
                <Activity className="w-4 h-4 text-recon-cyan" />
                <span>Toggle Live Pipeline Telemetry Stream</span>
              </Command.Item>
            </Command.Group>

            {/* Navigation Views */}
            <Command.Group heading="Navigation" className="text-[10px] font-mono uppercase text-ledger-400 px-2 py-1.5 mt-2">
              <Command.Item
                onSelect={() => {
                  setActiveView('explorer');
                  setCommandPaletteOpen(false);
                }}
                className="flex items-center gap-2.5 px-3 py-2 rounded-md text-ledger-200 hover:text-white hover:bg-ledger-800 cursor-pointer"
              >
                <Table2 className="w-4 h-4 text-ledger-400" />
                <span>3-Way Trace Ledger Explorer</span>
              </Command.Item>

              <Command.Item
                onSelect={() => {
                  setActiveView('batches');
                  setCommandPaletteOpen(false);
                }}
                className="flex items-center gap-2.5 px-3 py-2 rounded-md text-ledger-200 hover:text-white hover:bg-ledger-800 cursor-pointer"
              >
                <Layers className="w-4 h-4 text-recon-cyan" />
                <span>Level-2 Payout Batches & UTR Clearing</span>
              </Command.Item>

              <Command.Item
                onSelect={() => {
                  setActiveView('discrepancies');
                  setCommandPaletteOpen(false);
                }}
                className="flex items-center gap-2.5 px-3 py-2 rounded-md text-ledger-200 hover:text-white hover:bg-ledger-800 cursor-pointer"
              >
                <AlertTriangle className="w-4 h-4 text-recon-crimson" />
                <span>Discrepancy Audit Dossiers & AI Tickets</span>
              </Command.Item>

              <Command.Item
                onSelect={() => {
                  setActiveView('upload');
                  setCommandPaletteOpen(false);
                }}
                className="flex items-center gap-2.5 px-3 py-2 rounded-md text-ledger-200 hover:text-white hover:bg-ledger-800 cursor-pointer"
              >
                <UploadCloud className="w-4 h-4 text-recon-indigo" />
                <span>Ingest Custom Settlement CSV Files</span>
              </Command.Item>

              <Command.Item
                onSelect={() => {
                  setActiveView('history');
                  setCommandPaletteOpen(false);
                }}
                className="flex items-center gap-2.5 px-3 py-2 rounded-md text-ledger-200 hover:text-white hover:bg-ledger-800 cursor-pointer"
              >
                <History className="w-4 h-4 text-indigo-400" />
                <span>Upload History & Ingestion Archive</span>
              </Command.Item>

              <Command.Item
                onSelect={() => {
                  setActiveView('rules');
                  setCommandPaletteOpen(false);
                }}
                className="flex items-center gap-2.5 px-3 py-2 rounded-md text-ledger-200 hover:text-white hover:bg-ledger-800 cursor-pointer"
              >
                <Scale className="w-4 h-4 text-recon-emerald" />
                <span>Contract Rules, Agreed MDR & GST Rates</span>
              </Command.Item>
            </Command.Group>

            {/* Status Filters */}
            <Command.Group heading="Quick Filters" className="text-[10px] font-mono uppercase text-ledger-400 px-2 py-1.5 mt-2">
              <Command.Item
                onSelect={() => {
                  setActiveView('explorer');
                  setStatusFilter('discrepancy');
                  setCommandPaletteOpen(false);
                }}
                className="flex items-center gap-2.5 px-3 py-2 rounded-md text-ledger-200 hover:text-white hover:bg-ledger-800 cursor-pointer"
              >
                <AlertTriangle className="w-4 h-4 text-recon-crimson" />
                <span>Filter: Discrepancies Only</span>
              </Command.Item>

              <Command.Item
                onSelect={() => {
                  setActiveView('explorer');
                  setStatusFilter('reconciled');
                  setCommandPaletteOpen(false);
                }}
                className="flex items-center gap-2.5 px-3 py-2 rounded-md text-ledger-200 hover:text-white hover:bg-ledger-800 cursor-pointer"
              >
                <CheckCircle2 className="w-4 h-4 text-recon-emerald" />
                <span>Filter: Reconciled Matches Only</span>
              </Command.Item>
            </Command.Group>

            {/* Export Reports */}
            <Command.Group heading="Export Reports" className="text-[10px] font-mono uppercase text-ledger-400 px-2 py-1.5 mt-2">
              <Command.Item
                onSelect={() => {
                  apiClient.exportReport('csv', 'traces');
                  setCommandPaletteOpen(false);
                }}
                className="flex items-center gap-2.5 px-3 py-2 rounded-md text-ledger-200 hover:text-white hover:bg-ledger-800 cursor-pointer"
              >
                <FileSpreadsheet className="w-4 h-4 text-recon-emerald" />
                <span>Download 3-Way Trace Audit (CSV)</span>
              </Command.Item>

              <Command.Item
                onSelect={() => {
                  apiClient.exportReport('csv', 'discrepancies');
                  setCommandPaletteOpen(false);
                }}
                className="flex items-center gap-2.5 px-3 py-2 rounded-md text-ledger-200 hover:text-white hover:bg-ledger-800 cursor-pointer"
              >
                <FileSpreadsheet className="w-4 h-4 text-recon-crimson" />
                <span>Download Discrepancies Dispute Report (CSV)</span>
              </Command.Item>

              <Command.Item
                onSelect={() => {
                  apiClient.exportReport('json', 'traces');
                  setCommandPaletteOpen(false);
                }}
                className="flex items-center gap-2.5 px-3 py-2 rounded-md text-ledger-200 hover:text-white hover:bg-ledger-800 cursor-pointer"
              >
                <FileText className="w-4 h-4 text-recon-indigo" />
                <span>Download Full Reconciled Ledger (JSON)</span>
              </Command.Item>
            </Command.Group>
          </Command.List>

          {/* Footer Helper */}
          <div className="p-2 border-t border-ledger-700 bg-ledger-900/60 flex items-center justify-between text-[11px] font-mono text-ledger-400 px-3">
            <span>Navigation: ↑ ↓ | Select: ↵</span>
            <span>ESC to close</span>
          </div>
        </Command>
      </div>
    </div>
  );
};

