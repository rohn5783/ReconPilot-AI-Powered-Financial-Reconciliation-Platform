import React from 'react';
import {
  Activity,
  ChevronDown,
  ChevronUp,
  Terminal,
  CheckCircle2,
  AlertTriangle,
} from 'lucide-react';
import { useReconStore } from '../../store/useReconStore';

export const BottomTelemetryDock: React.FC = () => {
  const {
    isTelemetryDockOpen,
    toggleTelemetryDock,
    currentProgress,
    currentStage,
    currentMessage,
    telemetryLogs,
    isReconciling,
  } = useReconStore();

  return (
    <div className="border-t border-ledger-700 bg-ledger-850 select-none">
      {/* Strip bar */}
      <div
        onClick={toggleTelemetryDock}
        className="h-9 px-4 flex items-center justify-between cursor-pointer hover:bg-ledger-800 transition"
      >
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-1.5 text-xs font-mono">
            <span
              className={`w-2 h-2 rounded-full ${
                isReconciling ? 'bg-recon-cyan animate-ping' : 'bg-recon-emerald'
              }`}
            />
            <span className="font-semibold text-white uppercase">{currentStage}</span>
          </div>

          <span className="text-ledger-400 text-xs truncate max-w-xl font-mono">
            {currentMessage}
          </span>
        </div>

        <div className="flex items-center gap-4">
          {/* Progress bar pill */}
          <div className="flex items-center gap-2">
            <div className="w-24 bg-ledger-900 rounded-full h-1.5 overflow-hidden border border-ledger-700">
              <div
                className="h-full bg-recon-emerald transition-all duration-300"
                style={{ width: `${currentProgress}%` }}
              />
            </div>
            <span className="text-[11px] font-mono text-ledger-300 w-9 text-right tabular-nums">
              {currentProgress.toFixed(0)}%
            </span>
          </div>

          <button className="text-ledger-400 hover:text-white">
            {isTelemetryDockOpen ? (
              <ChevronDown className="w-4 h-4" />
            ) : (
              <ChevronUp className="w-4 h-4" />
            )}
          </button>
        </div>
      </div>

      {/* Expanded Terminal Log */}
      {isTelemetryDockOpen && (
        <div className="h-44 border-t border-ledger-700 bg-ledger-950 p-3 font-mono text-[11px] overflow-y-auto space-y-1">
          <div className="flex items-center gap-2 text-ledger-400 mb-2 border-b border-ledger-800 pb-1">
            <Terminal className="w-3.5 h-3.5 text-recon-cyan" />
            <span className="uppercase text-[10px] tracking-wider">Live Pipeline Telemetry & Event Log</span>
          </div>

          {telemetryLogs.length === 0 ? (
            <div className="text-ledger-400 py-4 text-center">
              No recent stream telemetry events. Run reconciliation to stream logs in real-time.
            </div>
          ) : (
            telemetryLogs.map((evt, idx) => (
              <div key={idx} className="flex items-start gap-2.5 leading-relaxed">
                <span className="text-ledger-400 shrink-0">
                  {new Date(evt.timestamp).toLocaleTimeString()}
                </span>
                <span
                  className={`px-1 py-0.2 rounded text-[10px] uppercase shrink-0 font-semibold ${
                    evt.stage === 'COMPLETED'
                      ? 'bg-emerald-950 text-emerald-400 border border-emerald-800'
                      : evt.stage === 'AI_DIAGNOSIS'
                      ? 'bg-indigo-950 text-indigo-400 border border-indigo-800'
                      : 'bg-ledger-800 text-ledger-300'
                  }`}
                >
                  {evt.stage}
                </span>
                <span className="text-ledger-200">{evt.message}</span>
              </div>
            ))
          )}
        </div>
      )}
    </div>
  );
};

