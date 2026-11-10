import React, { useState } from 'react';
import { Scope, ScanProfile } from '../../types';
import { Play, X, Shield, Sliders, Zap } from 'lucide-react';

interface LaunchScanModalProps {
  isOpen: boolean;
  onClose: () => void;
  scopes: Scope[];
  onLaunch: (scopeId: string, profile: ScanProfile, dryRun: boolean) => Promise<void>;
}

export const LaunchScanModal: React.FC<LaunchScanModalProps> = ({
  isOpen,
  onClose,
  scopes,
  onLaunch,
}) => {
  const [selectedScopeId, setSelectedScopeId] = useState<string>(scopes[0]?.id || '');
  const [profile, setProfile] = useState<ScanProfile>('standard');
  const [dryRun, setDryRun] = useState<boolean>(true);
  const [loading, setLoading] = useState<boolean>(false);

  if (!isOpen) return null;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedScopeId) return;
    setLoading(true);
    try {
      await onLaunch(selectedScopeId, profile, dryRun);
      onClose();
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm flex items-center justify-center p-4">
      <div className="bg-carbon-panel border border-carbon-border rounded-micro w-full max-w-md shadow-2xl overflow-hidden">
        {/* Modal Header */}
        <div className="flex items-center justify-between px-4 py-3 hairline-b">
          <div className="flex items-center gap-2">
            <Zap className="w-4 h-4 text-signal-accent" />
            <span className="font-semibold text-xs tracking-tight text-typography-primary font-mono uppercase">
              Launch Reconnaissance Pipeline
            </span>
          </div>
          <button
            onClick={onClose}
            className="text-typography-muted hover:text-typography-primary transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Modal Body */}
        <form onSubmit={handleSubmit} className="p-4 space-y-4 text-xs">
          {/* Target Scope Selection */}
          <div className="space-y-1.5">
            <label className="block text-[11px] font-mono text-typography-muted uppercase tracking-wider">
              Target Boundary (Root Domain)
            </label>
            <select
              value={selectedScopeId}
              onChange={(e) => setSelectedScopeId(e.target.value)}
              className="w-full bg-carbon-card border border-carbon-border rounded-micro px-3 py-2 text-xs font-mono text-typography-primary focus:outline-none focus:border-signal-accent"
              required
            >
              {scopes.map((s) => (
                <option key={s.id} value={s.id}>
                  {s.root_domain} ({s.name}) [{s.verification_status}]
                </option>
              ))}
            </select>
          </div>

          {/* Profile Selector */}
          <div className="space-y-1.5">
            <label className="block text-[11px] font-mono text-typography-muted uppercase tracking-wider">
              Execution Profile
            </label>
            <div className="grid grid-cols-3 gap-2">
              {(['passive', 'standard', 'deep'] as ScanProfile[]).map((p) => (
                <button
                  type="button"
                  key={p}
                  onClick={() => setProfile(p)}
                  className={`py-2 px-3 rounded-micro border text-center font-mono capitalize transition-all ${
                    profile === p
                      ? 'border-signal-accent bg-signal-accentBg/40 text-signal-accent font-semibold'
                      : 'border-carbon-border bg-carbon-card text-typography-secondary hover:text-typography-primary'
                  }`}
                >
                  {p}
                </button>
              ))}
            </div>
            <p className="text-[10px] text-typography-dim font-mono">
              {profile === 'passive' && 'Subfinder OSINT only. Zero direct network packets sent to target.'}
              {profile === 'standard' && 'Subfinder + Httpx web prober + top-100 TCP port scan + Nuclei misconfig audit.'}
              {profile === 'deep' && 'Comprehensive subdomain brute-force + full TCP port sweep + deep CVE vulnerability templates.'}
            </p>
          </div>

          {/* Dry Run / Simulation Mode Toggle */}
          <div className="p-3 rounded-micro border border-carbon-border bg-carbon-card/50 flex items-center justify-between">
            <div className="flex flex-col">
              <span className="font-semibold text-typography-primary text-[11px]">
                Dry-Run / Mock Telemetry Mode
              </span>
              <span className="text-[10px] text-typography-muted">
                Simulates real-time tool execution without sending external network traffic.
              </span>
            </div>
            <input
              type="checkbox"
              checked={dryRun}
              onChange={(e) => setDryRun(e.target.checked)}
              className="w-4 h-4 rounded-micro accent-signal-accent cursor-pointer"
            />
          </div>

          {/* Footer Controls */}
          <div className="flex items-center justify-end gap-2 pt-2 hairline-t">
            <button
              type="button"
              onClick={onClose}
              className="px-3 py-1.5 rounded-micro border border-carbon-border text-typography-muted hover:text-typography-primary text-xs font-mono transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={loading || !selectedScopeId}
              className="px-4 py-1.5 rounded-micro bg-signal-accent hover:bg-sky-400 text-carbon-canvas font-semibold text-xs font-mono flex items-center gap-1.5 transition-colors shadow-[0_0_12px_rgba(56,189,248,0.2)] disabled:opacity-50"
            >
              <Play className="w-3 h-3 fill-carbon-canvas" />
              <span>{loading ? 'Initializing...' : 'Start Pipeline'}</span>
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

