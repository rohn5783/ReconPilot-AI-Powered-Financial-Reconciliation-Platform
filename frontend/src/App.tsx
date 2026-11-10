import React, { useState, useEffect } from 'react';
import { QueryClient, QueryClientProvider, useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from './api/client';
import { useWorkspaceStore } from './store/useWorkspaceStore';
import { WorkspaceLayout } from './components/layout/WorkspaceLayout';
import { AssetInventoryView } from './components/views/AssetInventoryView';
import { TopologyView } from './components/views/TopologyView';
import { AIAnalystView } from './components/views/AIAnalystView';
import { ScopeRegistryView } from './components/views/ScopeRegistryView';
import { EvidenceDrawer } from './components/views/EvidenceDrawer';
import { LaunchScanModal } from './components/modals/LaunchScanModal';
import { Scope, ScanProfile } from './types';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5000,
      refetchOnWindowFocus: false,
    },
  },
});

const ReconPilotDashboard: React.FC = () => {
  const qc = useQueryClient();
  const {
    activeTab,
    selectedScanId,
    setSelectedScanId,
    selectedAsset,
    setSelectedAsset,
    selectedFinding,
    setSelectedFinding,
  } = useWorkspaceStore();

  const [isLaunchModalOpen, setIsLaunchModalOpen] = useState(false);

  // Load scopes
  const { data: scopes = [], refetch: refetchScopes } = useQuery({
    queryKey: ['scopes'],
    queryFn: () => api.getScopes(),
  });

  // Load scans
  const { data: scans = [], refetch: refetchScans } = useQuery({
    queryKey: ['scans'],
    queryFn: () => api.getScans(),
    refetchInterval: 3000,
  });

  // Default to first scan if not selected
  useEffect(() => {
    if (!selectedScanId && scans.length > 0) {
      setSelectedScanId(scans[0].id);
    }
  }, [scans, selectedScanId, setSelectedScanId]);

  const activeScan = scans.find((s) => s.id === selectedScanId) || scans[0] || null;
  const currentScanId = activeScan?.id;

  // Load assets for active scan
  const { data: assets = [] } = useQuery({
    queryKey: ['assets', currentScanId],
    queryFn: () => (currentScanId ? api.getScanAssets(currentScanId) : Promise.resolve([])),
    enabled: !!currentScanId,
  });

  // Load drifts for active scan
  const { data: drifts = [] } = useQuery({
    queryKey: ['drifts', currentScanId],
    queryFn: () => (currentScanId ? api.getScanDrifts(currentScanId) : Promise.resolve([])),
    enabled: !!currentScanId,
  });

  // Load findings for active scan
  const { data: findings = [] } = useQuery({
    queryKey: ['findings', currentScanId],
    queryFn: () => (currentScanId ? api.getScanFindings(currentScanId) : Promise.resolve([])),
    enabled: !!currentScanId,
  });

  // Load AI report for active scan
  const { data: report = null, refetch: refetchReport } = useQuery({
    queryKey: ['report', currentScanId],
    queryFn: () => (currentScanId ? api.getScanReport(currentScanId) : Promise.resolve(null)),
    enabled: !!currentScanId,
  });

  const launchScanMutation = useMutation({
    mutationFn: async ({
      scopeId,
      profile,
      dryRun,
    }: {
      scopeId: string;
      profile: ScanProfile;
      dryRun: boolean;
    }) => {
      return await api.createScan({
        scope_id: scopeId,
        profile,
        dry_run: dryRun,
      });
    },
    onSuccess: (newScan) => {
      setSelectedScanId(newScan.id);
      qc.invalidateQueries({ queryKey: ['scans'] });
      setIsLaunchModalOpen(false);
    },
    onError: (err: any) => {
      alert(`Failed to launch scan: ${err.message}`);
    },
  });

  const handleLaunchScan = async (scopeId: string, profile: ScanProfile, dryRun: boolean) => {
    await launchScanMutation.mutateAsync({ scopeId, profile, dryRun });
  };

  const handleSelectScopeToScan = (_scope: Scope) => {
    setIsLaunchModalOpen(true);
  };

  return (
    <WorkspaceLayout
      scopes={scopes}
      scans={scans}
      currentScan={activeScan}
      onNewScan={() => setIsLaunchModalOpen(true)}
      assetCount={assets.length}
      findingCount={findings.length}
      riskScore={report?.overall_risk_score}
    >
      {/* Dynamic View Canvas */}
      {activeTab === 'inventory' && (
        <AssetInventoryView assets={assets} drifts={drifts} findings={findings} />
      )}

      {activeTab === 'topology' && (
        <TopologyView
          rootDomain={scopes.find((s) => s.id === activeScan?.scope_id)?.root_domain || 'acme.org'}
          assets={assets}
          findings={findings}
        />
      )}

      {activeTab === 'analyst' && (
        <AIAnalystView
          report={report}
          scanJob={activeScan}
          onRefreshReport={() => refetchReport()}
        />
      )}

      {activeTab === 'scopes' && (
        <ScopeRegistryView
          scopes={scopes}
          onScopesUpdated={() => refetchScopes()}
          onSelectScopeToScan={handleSelectScopeToScan}
        />
      )}

      {/* Slide-over Evidence and Technical Inspector */}
      <EvidenceDrawer
        asset={selectedAsset}
        finding={selectedFinding}
        onClose={() => {
          setSelectedAsset(null);
          setSelectedFinding(null);
        }}
      />

      {/* Launch Scan Modal */}
      <LaunchScanModal
        isOpen={isLaunchModalOpen}
        onClose={() => setIsLaunchModalOpen(false)}
        scopes={scopes}
        onLaunch={handleLaunchScan}
      />
    </WorkspaceLayout>
  );
};

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <ReconPilotDashboard />
    </QueryClientProvider>
  );
}
