import React, { ReactNode } from 'react';
import { TopHeader } from './TopHeader';
import { Sidebar } from './Sidebar';
import { BottomTelemetryDock } from './BottomTelemetryDock';
import { DAGPipelineStepper } from '../dag/DAGPipelineStepper';
import { CommandPalette } from '../command/CommandPalette';
import { Scope, ScanJob } from '../../types';

interface WorkspaceLayoutProps {
  children: ReactNode;
  scopes: Scope[];
  scans: ScanJob[];
  currentScan: ScanJob | null;
  onNewScan: () => void;
  assetCount?: number;
  findingCount?: number;
  riskScore?: number;
}

export const WorkspaceLayout: React.FC<WorkspaceLayoutProps> = ({
  children,
  scopes,
  scans,
  currentScan,
  onNewScan,
  assetCount,
  findingCount,
  riskScore,
}) => {
  return (
    <div className="flex flex-col h-screen w-screen overflow-hidden bg-carbon-canvas text-typography-primary">
      {/* Top Application Bar */}
      <TopHeader scopes={scopes} scans={scans} onNewScan={onNewScan} />

      {/* Real-time DAG Execution Pipeline Stepper */}
      <DAGPipelineStepper scan={currentScan} />

      {/* Main Workspace Body */}
      <div className="flex flex-1 overflow-hidden relative">
        {/* Collapsible Industrial Left Rail */}
        <Sidebar assetCount={assetCount} findingCount={findingCount} riskScore={riskScore} />

        {/* Central Workstation Canvas */}
        <main className="flex-1 flex flex-col overflow-hidden bg-carbon-canvas relative">
          {children}
        </main>
      </div>

      {/* Bottom Live Subprocess Telemetry Dock */}
      <BottomTelemetryDock scanId={currentScan?.id || null} dryRun={currentScan?.dry_run} />

      {/* Global Cmd+K Command Palette */}
      <CommandPalette />
    </div>
  );
};

