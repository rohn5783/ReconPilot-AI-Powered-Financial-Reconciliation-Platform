import React from 'react';
import { useWorkspaceStore } from '../../store/useWorkspaceStore';
import { Scope, ScanJob } from '../../types';
import {
  Shield,
  Search,
  Play,
  Download,
  Terminal,
  Activity,
  Layers,
  ChevronRight,
  Command as CommandIcon,
} from 'lucide-react';

interface TopHeaderProps {
  scopes: Scope[];
  scans: ScanJob[];
  onNewScan: () => void;
}

export const TopHeader: React.FC<TopHeaderProps> = ({ scopes, scans, onNewScan }) => {
  const {
    activeTab,
    selectedScanId,
    setSelectedScanId,
    setCommandPaletteOpen,
    toggleDock,
    isDockOpen,
    searchQuery,
    setSearchQuery,
  } = useWorkspaceStore();

  const selectedScan = scans.find((s) => s.id === selectedScanId) || scans[0] || null;
  const activeScope = scopes.find((s) => s.id === selectedScan?.scope_id) || scopes[0] || null;

  return (
    <header className="h-12 bg-carbon-panel hairline-b flex items-center justify-between px-4 select-none shrink-0 z-30">
      {/* Left: Brand & Breadcrumbs */}
      <div className="flex items-center gap-3">
        <div className="flex items-center gap-2 pr-3 hairline-r">
          <div className="w-6 h-6 rounded-micro bg-signal-accent/10 border border-signal-accent/40 flex items-center justify-center">
            <Shield className="w-3.5 h-3.5 text-signal-accent" />
          </div>
          <span className="font-bold tracking-tight text-typography-primary font-mono text-[13px]">
            RECONPILOT
          </span>
          <span className="px-1.5 py-0.2 rounded-micro border border-carbon-border bg-carbon-card text-[9px] font-mono text-typography-muted">
            v1.0
          </span>
        </div>

        {/* Dynamic Breadcrumbs */}
        <div className="flex items-center gap-1.5 text-xs text-typography-muted font-mono">
          <span className="hover:text-typography-primary cursor-pointer">
            {activeScope ? activeScope.root_domain : 'targets'}
          </span>
          <ChevronRight className="w-3 h-3 text-typography-dim" />
          <span className="text-typography-secondary font-semibold">
            {activeTab.toUpperCase()}
          </span>
        </div>
      </div>

      {/* Center: Search & Command Palette Pill */}
      <div className="flex items-center gap-2 max-w-md w-full mx-4">
        <div className="relative w-full">
          <Search className="w-3.5 h-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-typography-muted" />
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            placeholder="Search FQDNs, open ports, technologies, CVEs..."
            className="w-full bg-carbon-card border border-carbon-border rounded-micro pl-8 pr-16 py-1.5 text-xs text-typography-primary placeholder:text-typography-muted focus:outline-none focus:border-signal-accent transition-colors font-mono"
          />
          <button
            onClick={() => setCommandPaletteOpen(true)}
            className="absolute right-1.5 top-1/2 -translate-y-1/2 px-1.5 py-0.5 rounded-micro border border-carbon-border bg-carbon-canvas text-[10px] font-mono text-typography-muted flex items-center gap-0.5 hover:text-typography-primary transition-colors"
          >
            <CommandIcon className="w-2.5 h-2.5" />
            <span>K</span>
          </button>
        </div>
      </div>

      {/* Right: Actions, Scan Profile & Telemetry Toggle */}
      <div className="flex items-center gap-2">
        {/* Scan Selector dropdown if multiple */}
        {scans.length > 0 && (
          <select
            value={selectedScanId || scans[0]?.id || ''}
            onChange={(e) => setSelectedScanId(e.target.value)}
            className="bg-carbon-card border border-carbon-border rounded-micro px-2 py-1 text-xs font-mono text-typography-secondary focus:outline-none focus:border-signal-accent cursor-pointer"
          >
            {scans.map((s) => (
              <option key={s.id} value={s.id}>
                Scan {s.id.slice(0, 8)} ({s.profile})
              </option>
            ))}
          </select>
        )}

        {/* Telemetry Console Toggle */}
        <button
          onClick={toggleDock}
          className={`px-2.5 py-1 rounded-micro border text-xs font-mono flex items-center gap-1.5 transition-colors ${
            isDockOpen
              ? 'border-signal-accent/40 bg-signal-accent/10 text-signal-accent'
              : 'border-carbon-border bg-carbon-card text-typography-muted hover:text-typography-primary'
          }`}
          title="Toggle Subprocess Telemetry Console (Hotkey: `)"
        >
          <Terminal className="w-3.5 h-3.5" />
          <span className="text-[11px] font-medium">Terminal</span>
        </button>

        {/* New Scan Action Button */}
        <button
          onClick={onNewScan}
          className="px-3 py-1 rounded-micro bg-signal-accent text-carbon-canvas font-semibold text-xs font-mono flex items-center gap-1.5 hover:bg-sky-400 active:bg-sky-500 transition-colors shadow-[0_0_12px_rgba(56,189,248,0.2)]"
        >
          <Play className="w-3 h-3 fill-carbon-canvas" />
          <span>Launch Scan</span>
        </button>
      </div>
    </header>
  );
};

