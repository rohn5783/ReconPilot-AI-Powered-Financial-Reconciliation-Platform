import React, { useEffect } from 'react';
import { Command } from 'cmdk';
import { useWorkspaceStore } from '../../store/useWorkspaceStore';
import {
  Search,
  Terminal,
  Play,
  Layers,
  Network,
  Bot,
  Shield,
  Download,
  AlertTriangle,
  Server,
} from 'lucide-react';

export const CommandPalette: React.FC = () => {
  const {
    commandPaletteOpen,
    setCommandPaletteOpen,
    setActiveTab,
    toggleDock,
    setSeverityFilter,
    setSearchQuery,
    selectedScanId,
  } = useWorkspaceStore();

  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if ((e.key === 'k' || e.key === 'K') && (e.metaKey || e.ctrlKey)) {
        e.preventDefault();
        setCommandPaletteOpen(!commandPaletteOpen);
      }
    };
    document.addEventListener('keydown', down);
    return () => document.removeEventListener('keydown', down);
  }, [commandPaletteOpen, setCommandPaletteOpen]);

  if (!commandPaletteOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 bg-black/75 backdrop-blur-sm flex items-start justify-center pt-24 px-4"
      onClick={() => setCommandPaletteOpen(false)}
    >
      <div onClick={(e) => e.stopPropagation()} className="w-full max-w-xl">
        <Command label="Global Command Palette">
          <div className="flex items-center gap-2 px-3 hairline-b">
            <Search className="w-4 h-4 text-typography-muted shrink-0" />
            <Command.Input autoFocus placeholder="Type a command, target domain, or hotkey..." />
          </div>

          <Command.List className="max-h-80 overflow-y-auto p-2">
            <Command.Empty className="p-4 text-xs text-center text-typography-muted">
              No matching commands or assets found.
            </Command.Empty>

            {/* Navigation & Workspace Views */}
            <Command.Group heading="Workspace Views">
              <Command.Item
                onSelect={() => {
                  setActiveTab('inventory');
                  setCommandPaletteOpen(false);
                }}
              >
                <Layers className="w-3.5 h-3.5 mr-2 text-signal-accent" />
                <span>Asset Inventory & Drift Explorer</span>
                <span className="ml-auto text-[10px] font-mono text-typography-muted">1</span>
              </Command.Item>

              <Command.Item
                onSelect={() => {
                  setActiveTab('topology');
                  setCommandPaletteOpen(false);
                }}
              >
                <Network className="w-3.5 h-3.5 mr-2 text-signal-accent" />
                <span>Attack Surface Topology Graph</span>
                <span className="ml-auto text-[10px] font-mono text-typography-muted">2</span>
              </Command.Item>

              <Command.Item
                onSelect={() => {
                  setActiveTab('analyst');
                  setCommandPaletteOpen(false);
                }}
              >
                <Bot className="w-3.5 h-3.5 mr-2 text-signal-accent" />
                <span>AI Security Analyst & Remediation Workstation</span>
                <span className="ml-auto text-[10px] font-mono text-typography-muted">3</span>
              </Command.Item>

              <Command.Item
                onSelect={() => {
                  setActiveTab('scopes');
                  setCommandPaletteOpen(false);
                }}
              >
                <Shield className="w-3.5 h-3.5 mr-2 text-signal-accent" />
                <span>Scope Registry & Ownership Verification</span>
                <span className="ml-auto text-[10px] font-mono text-typography-muted">4</span>
              </Command.Item>
            </Command.Group>

            {/* Tactical Recon Filters */}
            <Command.Group heading="High-Risk Exposure Filters">
              <Command.Item
                onSelect={() => {
                  setSeverityFilter('critical');
                  setActiveTab('inventory');
                  setCommandPaletteOpen(false);
                }}
              >
                <AlertTriangle className="w-3.5 h-3.5 mr-2 text-signal-critical" />
                <span>Filter Critical Severity Findings Only</span>
              </Command.Item>

              <Command.Item
                onSelect={() => {
                  setSearchQuery('6379');
                  setActiveTab('inventory');
                  setCommandPaletteOpen(false);
                }}
              >
                <Server className="w-3.5 h-3.5 mr-2 text-signal-critical" />
                <span>Search Exposed Redis Ports (6379)</span>
              </Command.Item>

              <Command.Item
                onSelect={() => {
                  setSearchQuery('22');
                  setActiveTab('inventory');
                  setCommandPaletteOpen(false);
                }}
              >
                <Server className="w-3.5 h-3.5 mr-2 text-signal-high" />
                <span>Search Exposed SSH Ports (22)</span>
              </Command.Item>
            </Command.Group>

            {/* Pipeline Controls & Exports */}
            <Command.Group heading="Actions & Telemetry">
              <Command.Item
                onSelect={() => {
                  toggleDock();
                  setCommandPaletteOpen(false);
                }}
              >
                <Terminal className="w-3.5 h-3.5 mr-2 text-signal-nominal" />
                <span>Toggle Subprocess Telemetry Dock</span>
                <span className="ml-auto text-[10px] font-mono text-typography-muted">`</span>
              </Command.Item>

              {selectedScanId && (
                <>
                  <Command.Item
                    onSelect={() => {
                      window.open(`/api/v1/scans/${selectedScanId}/export/markdown`, '_blank');
                      setCommandPaletteOpen(false);
                    }}
                  >
                    <Download className="w-3.5 h-3.5 mr-2 text-typography-primary" />
                    <span>Download Assessment Briefing (Markdown)</span>
                  </Command.Item>

                  <Command.Item
                    onSelect={() => {
                      window.open(`/api/v1/scans/${selectedScanId}/export/json`, '_blank');
                      setCommandPaletteOpen(false);
                    }}
                  >
                    <Download className="w-3.5 h-3.5 mr-2 text-typography-primary" />
                    <span>Download Full Scan Data (JSON)</span>
                  </Command.Item>
                </>
              )}
            </Command.Group>
          </Command.List>
        </Command>
      </div>
    </div>
  );
};

