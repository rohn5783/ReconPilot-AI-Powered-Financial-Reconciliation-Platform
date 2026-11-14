import React, { useState, useMemo } from 'react';
import {
  History,
  FileSpreadsheet,
  Download,
  Eye,
  Search,
  Filter,
  RefreshCw,
  Clock,
  Calendar,
  Database,
  CheckCircle2,
  AlertCircle,
  FileText,
  User,
  HardDrive,
  Copy,
  Check,
  X,
  UploadCloud,
  Sparkles,
} from 'lucide-react';
import { useReconStore } from '../../store/useReconStore';
import { UploadRecord } from '../../types';

export const UploadHistoryView: React.FC = () => {
  const {
    uploadHistory,
    isLoadingUploads,
    fetchUploadHistory,
    downloadUploadFile,
    setActiveView,
    seedData,
  } = useReconStore();

  const [search, setSearch] = useState('');
  const [typeFilter, setTypeFilter] = useState<'ALL' | 'ORDERS' | 'PG_SETTLEMENTS' | 'BANK_STATEMENTS'>('ALL');
  const [selectedRecord, setSelectedRecord] = useState<UploadRecord | null>(null);
  const [copied, setCopied] = useState(false);

  // Format bytes into human readable format
  const formatBytes = (bytes: number) => {
    if (!bytes || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`;
  };

  // Format date and time
  const formatDateTime = (isoString: string) => {
    try {
      const d = new Date(isoString);
      const year = d.getFullYear();
      const month = String(d.getMonth() + 1).padStart(2, '0');
      const day = String(d.getDate()).padStart(2, '0');
      const hours = String(d.getHours()).padStart(2, '0');
      const mins = String(d.getMinutes()).padStart(2, '0');
      const secs = String(d.getSeconds()).padStart(2, '0');
      return {
        date: `${year}-${month}-${day}`,
        time: `${hours}:${mins}:${secs}`,
        full: `${year}-${month}-${day} ${hours}:${mins}:${secs} UTC`,
      };
    } catch {
      return { date: isoString, time: '', full: isoString };
    }
  };

  // Filter records
  const filteredRecords = useMemo(() => {
    return (uploadHistory || []).filter((rec) => {
      const matchesSearch =
        rec.file_name.toLowerCase().includes(search.toLowerCase()) ||
        rec.uploaded_by.toLowerCase().includes(search.toLowerCase()) ||
        rec.summary.toLowerCase().includes(search.toLowerCase()) ||
        rec.file_type.toLowerCase().includes(search.toLowerCase());

      const matchesType = typeFilter === 'ALL' || rec.file_type === typeFilter;
      return matchesSearch && matchesType;
    });
  }, [uploadHistory, search, typeFilter]);

  // Aggregate KPI stats
  const totalRecordsCount = useMemo(() => {
    return (uploadHistory || []).reduce((sum, r) => sum + (r.record_count || 0), 0);
  }, [uploadHistory]);

  const totalBytesSize = useMemo(() => {
    return (uploadHistory || []).reduce((sum, r) => sum + (r.file_size || 0), 0);
  }, [uploadHistory]);

  const latestUpload = useMemo(() => {
    if (!uploadHistory || uploadHistory.length === 0) return 'None';
    return formatDateTime(uploadHistory[0].uploaded_at).full;
  }, [uploadHistory]);

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const getTypeBadge = (type: string) => {
    switch (type) {
      case 'ORDERS':
        return (
          <span className="px-2 py-0.5 rounded text-[10px] font-mono font-semibold bg-indigo-500/15 border border-indigo-500/30 text-indigo-300">
            ORDERS (L1)
          </span>
        );
      case 'PG_SETTLEMENTS':
        return (
          <span className="px-2 py-0.5 rounded text-[10px] font-mono font-semibold bg-recon-cyan/15 border border-recon-cyan/30 text-recon-cyan">
            PG SETTLEMENTS
          </span>
        );
      case 'BANK_STATEMENTS':
        return (
          <span className="px-2 py-0.5 rounded text-[10px] font-mono font-semibold bg-recon-emerald/15 border border-recon-emerald/30 text-recon-emerald">
            BANK UTR
          </span>
        );
      default:
        return (
          <span className="px-2 py-0.5 rounded text-[10px] font-mono font-semibold bg-ledger-800 border border-ledger-700 text-ledger-300">
            {type}
          </span>
        );
    }
  };

  return (
    <div className="flex-1 flex flex-col h-full overflow-hidden bg-ledger-900 select-none">
      {/* Top Header Bar */}
      <div className="px-6 py-4 border-b border-ledger-700/80 bg-ledger-850 flex items-center justify-between">
        <div>
          <div className="flex items-center gap-2">
            <History className="w-5 h-5 text-indigo-400" />
            <h1 className="text-base font-semibold text-white tracking-wide">
              Data Ingestion & Upload Archive
            </h1>
          </div>
          <p className="text-xs text-ledger-400 mt-0.5">
            Date-stamped audit ledger of all internal orders, settlement batches, and bank remittances ingested into ReconPilot.
          </p>
        </div>

        <div className="flex items-center gap-2.5">
          <button
            onClick={() => setActiveView('upload')}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-indigo-600/20 hover:bg-indigo-600/30 border border-indigo-500/40 text-indigo-300 text-xs font-mono font-medium transition"
          >
            <UploadCloud className="w-3.5 h-3.5 text-indigo-400" />
            <span>Upload New Files</span>
          </button>
          <button
            onClick={() => fetchUploadHistory()}
            disabled={isLoadingUploads}
            title="Refresh upload records"
            className="p-1.5 rounded-lg bg-ledger-800 hover:bg-ledger-750 border border-ledger-700 text-ledger-300 hover:text-white transition disabled:opacity-50"
          >
            <RefreshCw className={`w-4 h-4 ${isLoadingUploads ? 'animate-spin text-indigo-400' : ''}`} />
          </button>
        </div>
      </div>

      {/* KPI Cards Strip */}
      <div className="grid grid-cols-4 gap-4 px-6 py-4 border-b border-ledger-800 bg-ledger-950/40">
        <div className="p-3.5 rounded-xl bg-ledger-850/80 border border-ledger-750/70">
          <div className="flex items-center justify-between text-ledger-400 text-xs mb-1">
            <span className="font-mono uppercase text-[10px] tracking-wider">Total Ingestions</span>
            <FileSpreadsheet className="w-4 h-4 text-indigo-400" />
          </div>
          <div className="text-xl font-bold font-mono text-white">
            {(uploadHistory || []).length}
          </div>
          <div className="text-[11px] text-ledger-400 mt-0.5">Historical ingestion batches</div>
        </div>

        <div className="p-3.5 rounded-xl bg-ledger-850/80 border border-ledger-750/70">
          <div className="flex items-center justify-between text-ledger-400 text-xs mb-1">
            <span className="font-mono uppercase text-[10px] tracking-wider">Total Rows / Records</span>
            <Database className="w-4 h-4 text-recon-cyan" />
          </div>
          <div className="text-xl font-bold font-mono text-recon-cyan">
            {totalRecordsCount.toLocaleString()}
          </div>
          <div className="text-[11px] text-ledger-400 mt-0.5">Parsed financial transactions</div>
        </div>

        <div className="p-3.5 rounded-xl bg-ledger-850/80 border border-ledger-750/70">
          <div className="flex items-center justify-between text-ledger-400 text-xs mb-1">
            <span className="font-mono uppercase text-[10px] tracking-wider">Storage Volume</span>
            <HardDrive className="w-4 h-4 text-recon-emerald" />
          </div>
          <div className="text-xl font-bold font-mono text-recon-emerald">
            {formatBytes(totalBytesSize)}
          </div>
          <div className="text-[11px] text-ledger-400 mt-0.5">Raw archival payload size</div>
        </div>

        <div className="p-3.5 rounded-xl bg-ledger-850/80 border border-ledger-750/70">
          <div className="flex items-center justify-between text-ledger-400 text-xs mb-1">
            <span className="font-mono uppercase text-[10px] tracking-wider">Latest Timestamp</span>
            <Clock className="w-4 h-4 text-amber-400" />
          </div>
          <div className="text-xs font-bold font-mono text-amber-300 truncate mt-1.5" title={latestUpload}>
            {latestUpload}
          </div>
          <div className="text-[11px] text-ledger-400 mt-0.5">Most recent dataset run</div>
        </div>
      </div>

      {/* Filter and Search Bar */}
      <div className="px-6 py-3 border-b border-ledger-800 bg-ledger-850/40 flex items-center justify-between gap-4">
        <div className="flex-1 max-w-sm relative">
          <Search className="w-3.5 h-3.5 text-ledger-400 absolute left-3 top-2.5" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search by file name, uploader, summary..."
            className="w-full pl-9 pr-3 py-1.5 rounded-lg bg-ledger-900 border border-ledger-700 text-xs font-mono text-slate-100 placeholder-ledger-400 focus:outline-none focus:border-indigo-500 transition"
          />
        </div>

        <div className="flex items-center gap-1.5 bg-ledger-900 p-1 rounded-lg border border-ledger-750">
          <span className="text-[10px] font-mono uppercase text-ledger-400 px-2 flex items-center gap-1">
            <Filter className="w-3 h-3" />
            <span>Type:</span>
          </span>
          {(['ALL', 'ORDERS', 'PG_SETTLEMENTS', 'BANK_STATEMENTS'] as const).map((t) => (
            <button
              key={t}
              onClick={() => setTypeFilter(t)}
              className={`px-2.5 py-1 rounded text-[11px] font-mono transition ${
                typeFilter === t
                  ? 'bg-indigo-600 text-white font-medium shadow-sm'
                  : 'text-ledger-400 hover:text-white hover:bg-ledger-800'
              }`}
            >
              {t === 'ALL' ? 'All' : t.replace(/_/g, ' ')}
            </button>
          ))}
        </div>
      </div>

      {/* Main Table / Ledger List */}
      <div className="flex-1 overflow-y-auto px-6 py-4">
        {filteredRecords.length === 0 ? (
          <div className="h-64 flex flex-col items-center justify-center border border-dashed border-ledger-750 rounded-xl bg-ledger-850/30 text-center p-6">
            <div className="w-12 h-12 rounded-xl bg-ledger-800 flex items-center justify-center text-ledger-400 mb-3">
              <History className="w-6 h-6 text-indigo-400" />
            </div>
            <h3 className="text-sm font-semibold text-white">No Ingestion Records Found</h3>
            <p className="text-xs text-ledger-400 mt-1 max-w-sm">
              Upload orders, settlement batches, or bank remittances, or reseed the demo financial dataset to populate this date-wise archive.
            </p>
            <div className="flex items-center gap-3 mt-4">
              <button
                onClick={() => setActiveView('upload')}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-mono font-medium transition"
              >
                <UploadCloud className="w-3.5 h-3.5" />
                <span>Upload CSVs</span>
              </button>
              <button
                onClick={() => seedData()}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded bg-ledger-800 hover:bg-ledger-750 border border-ledger-700 text-ledger-300 hover:text-white text-xs font-mono font-medium transition"
              >
                <Sparkles className="w-3.5 h-3.5 text-recon-emerald" />
                <span>Seed Realistic Demo Data</span>
              </button>
            </div>
          </div>
        ) : (
          <div className="border border-ledger-750 rounded-xl overflow-hidden bg-ledger-850/60 shadow-lg">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="border-b border-ledger-750 bg-ledger-900/90 text-[10px] font-mono uppercase text-ledger-400 tracking-wider">
                  <th className="py-3 px-4">Date & Time (UTC)</th>
                  <th className="py-3 px-4">File Name</th>
                  <th className="py-3 px-4">Type</th>
                  <th className="py-3 px-4 text-right">Records</th>
                  <th className="py-3 px-4 text-right">Size</th>
                  <th className="py-3 px-4">Uploaded By</th>
                  <th className="py-3 px-4">Status</th>
                  <th className="py-3 px-4 text-right">Actions</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-ledger-800/80 text-xs">
                {filteredRecords.map((item) => {
                  const dt = formatDateTime(item.uploaded_at);
                  return (
                    <tr
                      key={item.id}
                      className="hover:bg-ledger-800/40 transition group"
                    >
                      {/* Date & Time */}
                      <td className="py-3 px-4 whitespace-nowrap">
                        <div className="font-mono text-slate-200 font-semibold flex items-center gap-1.5">
                          <Calendar className="w-3.5 h-3.5 text-ledger-400" />
                          <span>{dt.date}</span>
                        </div>
                        <div className="font-mono text-[11px] text-ledger-400 pl-5">
                          {dt.time} UTC
                        </div>
                      </td>

                      {/* File Name */}
                      <td className="py-3 px-4 font-mono">
                        <div className="flex items-center gap-2">
                          <FileSpreadsheet className="w-4 h-4 text-indigo-400 shrink-0" />
                          <span className="text-white font-medium truncate max-w-[220px]" title={item.file_name}>
                            {item.file_name}
                          </span>
                        </div>
                        {item.summary && (
                          <div className="text-[11px] text-ledger-400 truncate max-w-[240px] pl-6 mt-0.5" title={item.summary}>
                            {item.summary}
                          </div>
                        )}
                      </td>

                      {/* Type Badge */}
                      <td className="py-3 px-4 whitespace-nowrap">
                        {getTypeBadge(item.file_type)}
                      </td>

                      {/* Record Count */}
                      <td className="py-3 px-4 text-right font-mono text-slate-200 whitespace-nowrap">
                        <span className="font-semibold">{item.record_count.toLocaleString()}</span>
                        <span className="text-ledger-400 text-[10px] ml-1">rows</span>
                      </td>

                      {/* File Size */}
                      <td className="py-3 px-4 text-right font-mono text-ledger-300 whitespace-nowrap">
                        {formatBytes(item.file_size)}
                      </td>

                      {/* Uploaded By */}
                      <td className="py-3 px-4 whitespace-nowrap">
                        <div className="flex items-center gap-1.5 text-slate-300 font-mono text-[11px]">
                          <User className="w-3 h-3 text-ledger-400" />
                          <span className="truncate max-w-[150px]" title={item.uploaded_by}>
                            {item.uploaded_by}
                          </span>
                        </div>
                      </td>

                      {/* Status */}
                      <td className="py-3 px-4 whitespace-nowrap">
                        <div className="flex items-center gap-1.5">
                          <CheckCircle2 className="w-3.5 h-3.5 text-recon-emerald" />
                          <span className="font-mono text-[11px] text-recon-emerald font-semibold uppercase">
                            {item.status || 'SUCCESS'}
                          </span>
                        </div>
                      </td>

                      {/* Actions */}
                      <td className="py-3 px-4 text-right whitespace-nowrap">
                        <div className="flex items-center justify-end gap-1.5">
                          {/* Inspect Modal Trigger */}
                          {item.raw_data && (
                            <button
                              onClick={() => setSelectedRecord(item)}
                              title="Inspect raw CSV payload"
                              className="p-1.5 rounded-lg bg-ledger-800 hover:bg-ledger-700 text-ledger-300 hover:text-white transition"
                            >
                              <Eye className="w-3.5 h-3.5" />
                            </button>
                          )}

                          {/* Download Button */}
                          <button
                            onClick={() => downloadUploadFile(item.id, item.file_name)}
                            title={`Download ${item.file_name}`}
                            className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-indigo-600/20 hover:bg-indigo-600/30 border border-indigo-500/40 text-indigo-300 hover:text-white font-mono text-[11px] font-semibold transition"
                          >
                            <Download className="w-3.5 h-3.5 text-indigo-400" />
                            <span>Download CSV</span>
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Modal: Raw CSV Content Preview */}
      {selectedRecord && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-sm select-text">
          <div
            className="w-full max-w-3xl bg-ledger-900 border border-ledger-700 rounded-xl shadow-2xl flex flex-col max-h-[85vh] overflow-hidden"
            onClick={(e) => e.stopPropagation()}
          >
            {/* Modal Header */}
            <div className="px-6 py-4 border-b border-ledger-750 bg-ledger-950 flex items-center justify-between">
              <div className="flex items-center gap-2.5">
                <FileSpreadsheet className="w-5 h-5 text-indigo-400" />
                <div>
                  <h3 className="text-sm font-semibold text-white font-mono">{selectedRecord.file_name}</h3>
                  <p className="text-xs text-ledger-400">
                    Uploaded on {formatDateTime(selectedRecord.uploaded_at).full} • {selectedRecord.record_count} rows • {formatBytes(selectedRecord.file_size)}
                  </p>
                </div>
              </div>

              <div className="flex items-center gap-2">
                <button
                  onClick={() => copyToClipboard(selectedRecord.raw_data || '')}
                  className="flex items-center gap-1 px-2.5 py-1.5 rounded bg-ledger-800 hover:bg-ledger-750 text-ledger-300 hover:text-white text-xs font-mono transition"
                >
                  {copied ? <Check className="w-3.5 h-3.5 text-recon-emerald" /> : <Copy className="w-3.5 h-3.5" />}
                  <span>{copied ? 'Copied' : 'Copy CSV'}</span>
                </button>
                <button
                  onClick={() => downloadUploadFile(selectedRecord.id, selectedRecord.file_name)}
                  className="flex items-center gap-1 px-2.5 py-1.5 rounded bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-mono transition font-medium"
                >
                  <Download className="w-3.5 h-3.5" />
                  <span>Download</span>
                </button>
                <button
                  onClick={() => setSelectedRecord(null)}
                  className="p-1.5 rounded text-ledger-400 hover:text-white hover:bg-ledger-800 transition"
                >
                  <X className="w-4 h-4" />
                </button>
              </div>
            </div>

            {/* Modal Body */}
            <div className="flex-1 overflow-auto p-4 bg-ledger-950/70 font-mono text-[11px] text-slate-200 leading-relaxed">
              <pre className="whitespace-pre overflow-x-auto select-text font-mono">
                {selectedRecord.raw_data}
              </pre>
            </div>

            {/* Modal Footer */}
            <div className="px-6 py-2.5 border-t border-ledger-800 bg-ledger-950 flex items-center justify-between text-[11px] text-ledger-400 font-mono">
              <span>Type: {selectedRecord.file_type}</span>
              <span>Uploader: {selectedRecord.uploaded_by}</span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
