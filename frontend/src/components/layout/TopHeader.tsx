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
  Upload,
  Key,
  ShieldAlert,
  Check,
  X,
  Trash2,
  LogIn,
  LogOut,
  User,
} from 'lucide-react';
import { useReconStore } from '../../store/useReconStore';
import { apiClient } from '../../api/client';

export const TopHeader: React.FC = () => {
  const {
    isReconciling,
    triggerReconcile,
    resetData,
    seedData,
    setCommandPaletteOpen,
    toggleTelemetryDock,
    currentStage,
    setActiveView,
    apiKey,
    saveApiKey,
    lastError,
    currentUser,
    openAuthModal,
    logout,
  } = useReconStore();

  const [exportOpen, setExportOpen] = useState(false);
  const [keyModalOpen, setKeyModalOpen] = useState(false);
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [inputKey, setInputKey] = useState(apiKey || '');

  const isAuthError = Boolean(lastError && (lastError.toLowerCase().includes('unauthorized') || lastError.includes('401')));

  const getInitials = (name?: string, email?: string) => {
    if (name) {
      const parts = name.trim().split(/\s+/);
      if (parts.length >= 2) return `${parts[0][0]}${parts[1][0]}`.toUpperCase();
      if (parts.length === 1 && parts[0].length > 0) return parts[0].substring(0, 2).toUpperCase();
    }
    if (email) return email.substring(0, 2).toUpperCase();
    return 'OP';
  };

  const handleSaveKey = (e: React.FormEvent) => {
    e.preventDefault();
    saveApiKey(inputKey.trim());
    setKeyModalOpen(false);
  };

  return (
    <header className="h-14 border-b border-ledger-700 bg-ledger-850 px-4 flex items-center justify-between select-none">
      {/* Brand & Environment */}
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-2">
          <div className="w-7 h-7 rounded bg-ledger-800 border border-recon-emerald/40 flex items-center justify-center">
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
          <Database className="w-3.5 h-3.5 text-ledger-400" />
          <span className="text-ledger-300 font-mono text-[11px]">LEDGER ENGINE:</span>
          <span className="inline-flex items-center gap-1.5 text-recon-emerald font-medium text-[11px]">
            <span className="w-1.5 h-1.5 rounded-full bg-recon-emerald animate-ping" />
            ONLINE
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
        {/* API Key Modal Button */}
        <div className="relative">
          <button
            onClick={() => {
              setInputKey(apiKey || '');
              setKeyModalOpen(!keyModalOpen);
            }}
            title="Configure backend AUTH_API_KEY"
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded border text-xs font-medium transition ${
              isAuthError
                ? 'bg-recon-crimson/20 border-recon-crimson text-recon-crimson animate-pulse'
                : apiKey
                ? 'bg-recon-emerald/10 border-recon-emerald/40 text-recon-emerald hover:border-recon-emerald'
                : 'bg-ledger-800 border-ledger-700 text-ledger-300 hover:text-white'
            }`}
          >
            {isAuthError ? (
              <ShieldAlert className="w-3.5 h-3.5 text-recon-crimson" />
            ) : (
              <Key className="w-3.5 h-3.5" />
            )}
            <span>{isAuthError ? 'Auth Required' : apiKey ? 'Key Active' : 'API Key'}</span>
          </button>

          {keyModalOpen && (
            <div className="absolute right-0 mt-2 w-80 bg-ledger-850 border border-ledger-700 rounded-lg shadow-2xl p-4 z-50 text-xs select-text">
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-1.5 font-semibold text-white">
                  <Key className="w-4 h-4 text-recon-indigo" />
                  <span>API Authentication Key</span>
                </div>
                <button
                  type="button"
                  onClick={() => setKeyModalOpen(false)}
                  className="text-ledger-400 hover:text-white"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
              <p className="text-[11px] text-ledger-400 mb-3 leading-relaxed">
                Matches <code className="text-recon-cyan font-mono">AUTH_API_KEY</code> from your <code className="text-white">.env</code>. Saved securely in local browser storage.
              </p>
              <form onSubmit={handleSaveKey} className="space-y-3">
                <input
                  type="text"
                  placeholder="Paste AUTH_API_KEY here..."
                  value={inputKey}
                  onChange={(e) => setInputKey(e.target.value)}
                  className="w-full px-2.5 py-1.5 rounded bg-ledger-900 border border-ledger-700 text-white font-mono text-[11px] focus:outline-none focus:border-recon-indigo"
                />
                <div className="flex items-center justify-between pt-1">
                  <button
                    type="button"
                    onClick={() => {
                      setInputKey('');
                      saveApiKey('');
                      setKeyModalOpen(false);
                    }}
                    className="text-[11px] text-ledger-400 hover:text-recon-crimson underline"
                  >
                    Clear Key
                  </button>
                  <button
                    type="submit"
                    className="px-3 py-1.5 rounded bg-recon-indigo hover:bg-indigo-500 text-white font-semibold text-[11px] transition shadow"
                  >
                    Save & Reconnect
                  </button>
                </div>
              </form>
            </div>
          )}
        </div>

        {/* Upload CSVs Button */}
        <button
          onClick={() => setActiveView('upload')}
          title="Upload custom Order, PG Settlement, or Bank Statement CSV files"
          className="flex items-center gap-1.5 px-3 py-1.5 rounded bg-ledger-800 hover:bg-ledger-750 text-ledger-300 hover:text-white border border-ledger-700 text-xs font-medium transition"
        >
          <Upload className="w-3.5 h-3.5" />
          <span>Upload CSVs</span>
        </button>

        {/* Reset & Erase All Ledger Data Button */}
        <button
          onClick={() => resetData()}
          disabled={isReconciling}
          title="Erase all orders, settlements, bank remittances, and discrepancies from UI and database"
          className="flex items-center gap-1.5 px-3 py-1.5 rounded bg-ledger-800 hover:bg-recon-crimson/20 hover:border-recon-crimson/50 text-ledger-300 hover:text-recon-crimson border border-ledger-700 text-xs font-medium transition disabled:opacity-50"
        >
          <Trash2 className={`w-3.5 h-3.5 ${isReconciling ? 'animate-pulse text-recon-crimson' : 'text-recon-crimson/80'}`} />
          <span>Reset & Erase All Data</span>
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
          className="flex items-center gap-1.5 px-3.5 py-1.5 rounded bg-recon-emerald hover:bg-emerald-400 text-ledger-900 text-xs font-semibold border border-emerald-500 transition disabled:opacity-50"
        >
          {isReconciling ? (
            <>
              <RefreshCw className="w-3.5 h-3.5 animate-spin" />
              <span>Reconciling...</span>
            </>
          ) : (
            <>
              <Play className="w-3.5 h-3.5 fill-current" />
              <span>Run 3-Way Reconciliation</span>
            </>
          )}
        </button>

        {/* User Account / Sign In */}
        {currentUser ? (
          <div className="relative">
            <button
              onClick={() => setUserMenuOpen(!userMenuOpen)}
              className="flex items-center gap-2 px-2.5 py-1.5 rounded bg-ledger-800 hover:bg-ledger-750 border border-ledger-700 hover:border-ledger-600 text-xs transition"
            >
              <div className="w-5 h-5 rounded-full bg-indigo-500/20 border border-indigo-500/40 text-indigo-300 flex items-center justify-center text-[10px] font-mono font-semibold">
                {getInitials(currentUser.full_name, currentUser.email)}
              </div>
              <span className="font-mono text-xs text-slate-200 max-w-[110px] truncate">
                {currentUser.full_name || currentUser.email}
              </span>
              <ChevronDown className="w-3 h-3 text-ledger-400" />
            </button>

            {userMenuOpen && (
              <div className="absolute right-0 mt-1.5 w-64 bg-ledger-900 border border-ledger-750 rounded-xl shadow-2xl p-3 z-50 text-xs select-text">
                <div className="pb-2.5 mb-2.5 border-b border-ledger-800">
                  <div className="text-slate-100 font-semibold truncate">{currentUser.full_name}</div>
                  <div className="text-[11px] font-mono text-slate-400 truncate">{currentUser.email}</div>
                  {currentUser.company_name && (
                    <div className="text-[10px] text-slate-400 mt-0.5 truncate">{currentUser.company_name}</div>
                  )}
                  <div className="mt-2 inline-block px-2 py-0.5 rounded text-[10px] font-mono uppercase font-semibold bg-indigo-500/15 border border-indigo-500/30 text-indigo-300">
                    {currentUser.role.replace(/_/g, ' ')}
                  </div>
                </div>

                <button
                  type="button"
                  onClick={() => {
                    logout();
                    setUserMenuOpen(false);
                  }}
                  className="w-full flex items-center gap-2 px-2.5 py-2 rounded-lg text-rose-400 hover:bg-rose-500/10 hover:text-rose-300 transition text-left"
                >
                  <LogOut className="w-3.5 h-3.5" />
                  <span className="font-mono text-xs">Sign Out</span>
                </button>
              </div>
            )}
          </div>
        ) : (
          <button
            onClick={() => openAuthModal('login')}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded bg-indigo-600/20 hover:bg-indigo-600/30 text-indigo-300 border border-indigo-500/40 text-xs font-mono font-medium transition"
          >
            <LogIn className="w-3.5 h-3.5 text-indigo-400" />
            <span>Sign In</span>
          </button>
        )}
      </div>
    </header>
  );
};

