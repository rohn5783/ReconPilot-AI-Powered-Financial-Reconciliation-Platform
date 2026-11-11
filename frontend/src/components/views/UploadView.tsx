import React, { useState } from 'react';
import {
  UploadCloud,
  FileSpreadsheet,
  CheckCircle2,
  Play,
  Download,
  AlertCircle,
  FileCheck,
} from 'lucide-react';
import { useReconStore } from '../../store/useReconStore';

export const UploadView: React.FC = () => {
  const { triggerReconcile, isReconciling, setActiveView, lastError } = useReconStore();

  const [ordersFile, setOrdersFile] = useState<File | null>(null);
  const [pgFile, setPGFile] = useState<File | null>(null);
  const [bankFile, setBankFile] = useState<File | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErrorMsg(null);
    setSuccessMsg(null);

    const formData = new FormData();
    if (ordersFile) formData.append('orders_file', ordersFile);
    if (pgFile) formData.append('pg_file', pgFile);
    if (bankFile) formData.append('bank_file', bankFile);

    const ok = await triggerReconcile(formData);
    if (ok) {
      setSuccessMsg('CSVs parsed and 3-way reconciliation completed successfully!');
      setTimeout(() => setActiveView('explorer'), 800);
    } else {
      setErrorMsg(useReconStore.getState().lastError || 'Reconciliation failed. Check file format or API credentials.');
    }
  };

  return (
    <div className="flex-1 flex flex-col h-full overflow-y-auto bg-ledger-900 p-6 select-none">
      <div className="max-w-3xl mx-auto w-full space-y-6">
        <div>
          <div className="flex items-center gap-2">
            <UploadCloud className="w-5 h-5 text-recon-indigo" />
            <h2 className="text-base font-semibold text-white">Streaming Settlement File Ingestion</h2>
          </div>
          <p className="text-xs text-ledger-400 mt-1">
            Ingest custom settlement files processed through bounded 2048 KB ring buffers for zero-OOM high-throughput parsing.
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Status & Error Feedback Banners */}
          {errorMsg && (
            <div className="p-3.5 rounded-lg bg-recon-crimson/15 border border-recon-crimson/50 text-recon-crimson text-xs flex items-start gap-2.5">
              <AlertCircle className="w-4 h-4 mt-0.5 shrink-0" />
              <div className="flex-1 font-mono break-all leading-relaxed">
                <span className="font-semibold uppercase tracking-wider">Ingestion / Auth Failure:</span> {errorMsg}
                {errorMsg.toLowerCase().includes('unauthorized') && (
                  <p className="mt-1 text-[11px] text-ledger-300">
                    Your backend requires an API Key. Click the <span className="text-white font-semibold">Key/Shield</span> icon in the top header to configure your <code className="bg-ledger-800 px-1 py-0.5 rounded text-recon-cyan">AUTH_API_KEY</code>, or remove it from <code className="bg-ledger-800 px-1 py-0.5 rounded">.env</code>.
                  </p>
                )}
              </div>
            </div>
          )}

          {successMsg && (
            <div className="p-3.5 rounded-lg bg-recon-emerald/15 border border-recon-emerald/50 text-recon-emerald text-xs flex items-center gap-2.5">
              <CheckCircle2 className="w-4 h-4 shrink-0" />
              <span className="font-mono font-medium">{successMsg}</span>
            </div>
          )}

          {/* 1. Orders File */}
          <div className="bg-ledger-850 p-4 rounded-lg border border-ledger-700 hover:border-ledger-600 transition">
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2">
                <FileSpreadsheet className="w-4 h-4 text-recon-emerald" />
                <span className="text-xs font-semibold text-white">1. Internal Orders CSV</span>
              </div>
              <span className="text-[10px] font-mono text-ledger-400">Headers: order_id, gross_amount, currency...</span>
            </div>
            <label className="flex flex-col items-center justify-center border-2 border-dashed border-ledger-700 hover:border-recon-emerald/60 rounded-lg p-4 cursor-pointer transition bg-ledger-900/40">
              <input
                type="file"
                accept=".csv"
                onChange={(e) => setOrdersFile(e.target.files?.[0] || null)}
                className="hidden"
              />
              {ordersFile ? (
                <div className="flex items-center gap-2 text-xs text-recon-emerald font-mono">
                  <FileCheck className="w-4 h-4" />
                  <span>{ordersFile.name} ({(ordersFile.size / 1024).toFixed(1)} KB)</span>
                </div>
              ) : (
                <div className="text-center text-xs text-ledger-400 font-mono">
                  <span>Click to select or drag & drop Internal Orders CSV</span>
                </div>
              )}
            </label>
          </div>

          {/* 2. PG Settlements File */}
          <div className="bg-ledger-850 p-4 rounded-lg border border-ledger-700 hover:border-ledger-600 transition">
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2">
                <FileSpreadsheet className="w-4 h-4 text-recon-amber" />
                <span className="text-xs font-semibold text-white">2. Payment Gateway Settlements CSV</span>
              </div>
              <span className="text-[10px] font-mono text-ledger-400">Headers: pg_txn_id, order_id, pg_fee, net_amount...</span>
            </div>
            <label className="flex flex-col items-center justify-center border-2 border-dashed border-ledger-700 hover:border-recon-amber/60 rounded-lg p-4 cursor-pointer transition bg-ledger-900/40">
              <input
                type="file"
                accept=".csv"
                onChange={(e) => setPGFile(e.target.files?.[0] || null)}
                className="hidden"
              />
              {pgFile ? (
                <div className="flex items-center gap-2 text-xs text-recon-amber font-mono">
                  <FileCheck className="w-4 h-4" />
                  <span>{pgFile.name} ({(pgFile.size / 1024).toFixed(1)} KB)</span>
                </div>
              ) : (
                <div className="text-center text-xs text-ledger-400 font-mono">
                  <span>Click to select or drag & drop Razorpay / Stripe Settlements CSV</span>
                </div>
              )}
            </label>
          </div>

          {/* 3. Bank Statements File */}
          <div className="bg-ledger-850 p-4 rounded-lg border border-ledger-700 hover:border-ledger-600 transition">
            <div className="flex items-center justify-between mb-2">
              <div className="flex items-center gap-2">
                <FileSpreadsheet className="w-4 h-4 text-recon-cyan" />
                <span className="text-xs font-semibold text-white">3. Bank Statement Remittance CSV</span>
              </div>
              <span className="text-[10px] font-mono text-ledger-400">Headers: utr_number, credit_amount, description...</span>
            </div>
            <label className="flex flex-col items-center justify-center border-2 border-dashed border-ledger-700 hover:border-recon-cyan/60 rounded-lg p-4 cursor-pointer transition bg-ledger-900/40">
              <input
                type="file"
                accept=".csv"
                onChange={(e) => setBankFile(e.target.files?.[0] || null)}
                className="hidden"
              />
              {bankFile ? (
                <div className="flex items-center gap-2 text-xs text-recon-cyan font-mono">
                  <FileCheck className="w-4 h-4" />
                  <span>{bankFile.name} ({(bankFile.size / 1024).toFixed(1)} KB)</span>
                </div>
              ) : (
                <div className="text-center text-xs text-ledger-400 font-mono">
                  <span>Click to select or drag & drop Bank Statement (HDFC/ICICI) CSV</span>
                </div>
              )}
            </label>
          </div>

          {/* Trigger button */}
          <div className="pt-2 flex justify-end">
            <button
              type="submit"
              disabled={isReconciling || (!ordersFile && !pgFile && !bankFile)}
              className="flex items-center gap-2 px-5 py-2.5 rounded-lg bg-gradient-to-r from-emerald-600 to-teal-600 hover:from-emerald-500 hover:to-teal-500 text-white text-xs font-semibold shadow-lg shadow-emerald-950/40 transition disabled:opacity-50"
            >
              <Play className="w-4 h-4 fill-current" />
              <span>{isReconciling ? 'Streaming & Matching...' : 'Start 3-Way Reconciliation'}</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

