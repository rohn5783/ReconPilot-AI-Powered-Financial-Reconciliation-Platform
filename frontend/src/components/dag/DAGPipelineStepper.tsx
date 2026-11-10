import React from 'react';
import { ScanJob, StageName } from '../../types';
import { CheckCircle2, Clock, PlayCircle, XCircle, ArrowRight, Activity, ShieldAlert, Cpu } from 'lucide-react';

interface DAGPipelineStepperProps {
  scan: ScanJob | null;
}

interface StageStep {
  id: StageName;
  label: string;
  sublabel: string;
  icon: React.ComponentType<{ className?: string }>;
}

const STAGES: StageStep[] = [
  { id: 'discovery', label: '1. Discovery', sublabel: 'Subfinder', icon: PlayCircle },
  { id: 'probing', label: '2. Probing', sublabel: 'Httpx', icon: Activity },
  { id: 'portscan', label: '3. PortScan', sublabel: 'Nmap', icon: Cpu },
  { id: 'auditor', label: '4. Auditor', sublabel: 'Nuclei', icon: ShieldAlert },
  { id: 'ai_synthesis', label: '5. AI Synthesis', sublabel: 'Gemini SDK', icon: Cpu },
];

export const DAGPipelineStepper: React.FC<DAGPipelineStepperProps> = ({ scan }) => {
  if (!scan) return null;

  const currentStage = scan.current_stage;
  const checkpoints = scan.stage_checkpoints || {};

  return (
    <div className="flex items-center justify-between px-4 py-2 bg-carbon-panel hairline-b overflow-x-auto text-xs">
      {/* Target & Global Status Pill */}
      <div className="flex items-center gap-3 shrink-0 pr-4 hairline-r">
        <div className="flex flex-col">
          <span className="text-[10px] uppercase font-mono tracking-wider text-typography-muted">Target Boundary</span>
          <span className="font-mono font-semibold text-typography-primary tracking-tight">
            {scan.scope_id ? 'acme.org' : 'Target Host'}
          </span>
        </div>

        <span
          className={`px-2 py-0.5 rounded-micro border font-mono text-[11px] uppercase tracking-wider font-semibold ${
            scan.status === 'running'
              ? 'bg-signal-accent/10 border-signal-accent/30 text-signal-accent animate-pulse'
              : scan.status === 'completed'
              ? 'bg-signal-nominalBg border-signal-nominal/40 text-signal-nominal'
              : scan.status === 'failed'
              ? 'bg-signal-criticalBg border-signal-critical/40 text-signal-critical'
              : 'bg-carbon-card border-carbon-border text-typography-muted'
          }`}
        >
          {scan.status}
        </span>
      </div>

      {/* DAG Stage Horizontal Flow */}
      <div className="flex items-center gap-2 px-3 overflow-x-auto">
        {STAGES.map((st, idx) => {
          const cp = checkpoints[st.id];
          const isDone = cp?.status === 'done' || (scan.status === 'completed' && currentStage === 'completed');
          const isRunning = currentStage === st.id && scan.status === 'running';
          const isFailed = cp?.status === 'failed';
          const isPending = !isDone && !isRunning && !isFailed;

          return (
            <React.Fragment key={st.id}>
              <div
                className={`flex items-center gap-2 px-2.5 py-1.5 rounded-micro border transition-all ${
                  isRunning
                    ? 'border-signal-accent bg-signal-accentBg/40 shadow-[0_0_12px_rgba(56,189,248,0.15)]'
                    : isDone
                    ? 'border-carbon-border bg-carbon-card text-typography-primary'
                    : isFailed
                    ? 'border-signal-critical/50 bg-signal-criticalBg text-signal-critical'
                    : 'border-carbon-border/50 bg-carbon-canvas/60 text-typography-muted'
                }`}
              >
                {/* Status Indicator Icon */}
                {isDone ? (
                  <CheckCircle2 className="w-3.5 h-3.5 text-signal-nominal" />
                ) : isRunning ? (
                  <div className="w-3.5 h-3.5 rounded-full border-2 border-signal-accent border-t-transparent animate-spin" />
                ) : isFailed ? (
                  <XCircle className="w-3.5 h-3.5 text-signal-critical" />
                ) : (
                  <Clock className="w-3.5 h-3.5 text-typography-dim" />
                )}

                <div className="flex flex-col">
                  <div className="flex items-center gap-1.5">
                    <span className={`font-semibold tracking-tight text-[11px] ${isRunning ? 'text-signal-accent' : ''}`}>
                      {st.label}
                    </span>
                    <span className="text-[10px] font-mono text-typography-muted">
                      ({st.sublabel})
                    </span>
                  </div>

                  <div className="flex items-center gap-2 text-[10px] font-mono text-typography-muted">
                    {cp?.duration_ms ? (
                      <span>{(cp.duration_ms / 1000).toFixed(1)}s</span>
                    ) : isRunning ? (
                      <span className="text-signal-accent">Active...</span>
                    ) : (
                      <span>—</span>
                    )}

                    {cp?.output_count !== undefined && cp.output_count > 0 && (
                      <span className="text-typography-secondary font-medium">
                        • {cp.output_count} items
                      </span>
                    )}
                  </div>
                </div>
              </div>

              {idx < STAGES.length - 1 && (
                <ArrowRight className="w-3 h-3 text-carbon-border shrink-0" />
              )}
            </React.Fragment>
          );
        })}
      </div>

      {/* Global Performance Telemetry */}
      <div className="flex items-center gap-4 shrink-0 pl-4 hairline-l">
        <div className="flex flex-col text-right">
          <span className="text-[10px] uppercase font-mono tracking-wider text-typography-muted">Elapsed</span>
          <span className="font-mono font-semibold text-typography-primary">
            {scan.metrics?.duration_seconds ? `${scan.metrics.duration_seconds.toFixed(1)}s` : '0.0s'}
          </span>
        </div>

        <div className="flex flex-col text-right">
          <span className="text-[10px] uppercase font-mono tracking-wider text-typography-muted">Throughput</span>
          <span className="font-mono font-semibold text-typography-primary">
            {scan.metrics?.requests_per_sec ? `${scan.metrics.requests_per_sec.toFixed(1)} req/s` : '—'}
          </span>
        </div>
      </div>
    </div>
  );
};

