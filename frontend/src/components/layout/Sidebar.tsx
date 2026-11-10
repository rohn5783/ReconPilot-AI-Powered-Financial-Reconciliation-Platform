import React from 'react';
import { useWorkspaceStore, WorkspaceTab } from '../../store/useWorkspaceStore';
import {
  Layers,
  Network,
  Bot,
  Shield,
  Activity,
  Cpu,
  Database,
  Radio,
} from 'lucide-react';

interface SidebarProps {
  assetCount?: number;
  findingCount?: number;
  riskScore?: number;
}

export const Sidebar: React.FC<SidebarProps> = ({
  assetCount = 15,
  findingCount = 7,
  riskScore = 100,
}) => {
  const { activeTab, setActiveTab } = useWorkspaceStore();

  const navItems: { id: WorkspaceTab; label: string; icon: React.ComponentType<{ className?: string }>; badge?: string; badgeColor?: string }[] = [
    {
      id: 'inventory',
      label: 'Asset Inventory',
      icon: Layers,
      badge: `${assetCount}`,
      badgeColor: 'text-typography-secondary bg-carbon-card border-carbon-border',
    },
    {
      id: 'topology',
      label: 'Attack Topology',
      icon: Network,
    },
    {
      id: 'analyst',
      label: 'AI Analyst & Fixes',
      icon: Bot,
      badge: `${riskScore}/100`,
      badgeColor: riskScore >= 75 ? 'text-signal-critical bg-signal-criticalBg border-signal-critical/40' : 'text-signal-nominal bg-signal-nominalBg border-signal-nominal/40',
    },
    {
      id: 'scopes',
      label: 'Scope Registry',
      icon: Shield,
    },
  ];

  return (
    <aside className="w-56 bg-carbon-panel hairline-r flex flex-col justify-between shrink-0 select-none z-20">
      {/* Top Navigation Links */}
      <div className="p-2 space-y-1">
        <div className="px-2 py-1.5 text-[10px] font-mono uppercase tracking-widest text-typography-muted font-semibold">
          Workstation Panes
        </div>

        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = activeTab === item.id;

          return (
            <button
              key={item.id}
              onClick={() => setActiveTab(item.id)}
              className={`w-full flex items-center justify-between px-2.5 py-2 rounded-micro text-xs font-medium transition-all ${
                isActive
                  ? 'bg-carbon-active border border-signal-accent/30 text-signal-accent shadow-sm'
                  : 'text-typography-secondary hover:text-typography-primary hover:bg-carbon-hover border border-transparent'
              }`}
            >
              <div className="flex items-center gap-2.5">
                <Icon className={`w-4 h-4 ${isActive ? 'text-signal-accent' : 'text-typography-muted'}`} />
                <span className="tracking-tight">{item.label}</span>
              </div>

              {item.badge && (
                <span
                  className={`px-1.5 py-0.2 rounded-micro border text-[10px] font-mono font-medium ${item.badgeColor}`}
                >
                  {item.badge}
                </span>
              )}
            </button>
          );
        })}
      </div>

      {/* Bottom Telemetry Health & Daemon Monitor */}
      <div className="p-3 hairline-t bg-carbon-canvas/50 text-[11px] font-mono space-y-2">
        <div className="flex items-center justify-between text-typography-muted">
          <div className="flex items-center gap-1.5">
            <Radio className="w-3 h-3 text-signal-nominal animate-pulse" />
            <span className="text-[10px] uppercase tracking-wider">Engine Daemon</span>
          </div>
          <span className="text-signal-nominal text-[10px]">ONLINE</span>
        </div>

        <div className="space-y-1 text-typography-dim text-[10px]">
          <div className="flex justify-between">
            <span>Worker Pool:</span>
            <span className="text-typography-secondary">4 / 4 Slots</span>
          </div>
          <div className="flex justify-between">
            <span>Ring Buffer:</span>
            <span className="text-typography-secondary">1024 KB OK</span>
          </div>
          <div className="flex justify-between">
            <span>SSRF Guard:</span>
            <span className="text-signal-nominal">RFC1918 Enforced</span>
          </div>
        </div>
      </div>
    </aside>
  );
};

