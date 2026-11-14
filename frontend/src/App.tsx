import React, { useEffect } from 'react';
import { useReconStore } from './store/useReconStore';
import { apiClient } from './api/client';
import { TopHeader } from './components/layout/TopHeader';
import { Sidebar } from './components/layout/Sidebar';
import { KPIStrip } from './components/layout/KPIStrip';
import { ThreeWayExplorerView } from './components/views/ThreeWayExplorerView';
import { PayoutBatchesView } from './components/views/PayoutBatchesView';
import { DiscrepanciesView } from './components/views/DiscrepanciesView';
import { UploadView } from './components/views/UploadView';
import { UploadHistoryView } from './components/views/UploadHistoryView';
import { ContractRulesView } from './components/views/ContractRulesView';
import { EvidenceDrawer } from './components/views/EvidenceDrawer';
import { BottomTelemetryDock } from './components/layout/BottomTelemetryDock';
import { CommandPalette } from './components/command/CommandPalette';
import { AuthModal } from './components/auth/AuthModal';

export const App: React.FC = () => {
  const activeView = useReconStore((state) => state.activeView);

  useEffect(() => {
    // 0. Restore user authentication session if token exists
    useReconStore.getState().checkAuth();

    // 1. Initial data fetch on mount
    useReconStore.getState().fetchAllData();

    // 2. Connect to live real-time persistent SSE stream (single connection, zero reconnect loops)
    const unsubscribe = apiClient.subscribeTelemetry((event) => {
      useReconStore.getState().addTelemetryEvent(event);
    });

    return () => {
      unsubscribe();
    };
  }, []);

  return (
    <div className="flex flex-col h-screen w-screen overflow-hidden bg-ledger-900 text-ledger-100 font-sans">
      {/* Top Header */}
      <TopHeader />

      {/* KPI Strip */}
      <KPIStrip />

      {/* Main Body */}
      <div className="flex-1 flex overflow-hidden">
        {/* Left Sidebar */}
        <Sidebar />

        {/* Center Dynamic Workstation View */}
        <main className="flex-1 flex flex-col overflow-hidden bg-ledger-900">
          {activeView === 'explorer' && <ThreeWayExplorerView />}
          {activeView === 'batches' && <PayoutBatchesView />}
          {activeView === 'discrepancies' && <DiscrepanciesView />}
          {activeView === 'upload' && <UploadView />}
          {activeView === 'history' && <UploadHistoryView />}
          {activeView === 'rules' && (
            <div className="flex-1 overflow-y-auto p-6">
              <ContractRulesView />
            </div>
          )}
        </main>
      </div>

      {/* Bottom Telemetry Dock */}
      <BottomTelemetryDock />

      {/* Slide-over Discrepancy & AI Forensic Audit Drawer */}
      <EvidenceDrawer />

      {/* Command Palette (Cmd+K) */}
      <CommandPalette />

      {/* Operator Authentication & Registration Modal */}
      <AuthModal />
    </div>
  );
};

export default App;

