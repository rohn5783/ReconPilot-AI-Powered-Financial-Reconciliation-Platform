import { create } from 'zustand';
import { Asset, Finding, FindingSeverity } from '../types';

export type WorkspaceTab = 'inventory' | 'topology' | 'analyst' | 'scopes';

interface WorkspaceState {
  activeTab: WorkspaceTab;
  setActiveTab: (tab: WorkspaceTab) => void;

  selectedScanId: string | null;
  setSelectedScanId: (id: string | null) => void;

  selectedAsset: Asset | null;
  setSelectedAsset: (asset: Asset | null) => void;

  selectedFinding: Finding | null;
  setSelectedFinding: (finding: Finding | null) => void;

  isDockOpen: boolean;
  setIsDockOpen: (open: boolean) => void;
  toggleDock: () => void;

  dockHeight: number;
  setDockHeight: (height: number) => void;

  searchQuery: string;
  setSearchQuery: (q: string) => void;

  severityFilter: FindingSeverity | 'all';
  setSeverityFilter: (s: FindingSeverity | 'all') => void;

  commandPaletteOpen: boolean;
  setCommandPaletteOpen: (open: boolean) => void;

  activeInspectorTab: 'headers' | 'evidence' | 'ports' | 'remediation';
  setActiveInspectorTab: (tab: 'headers' | 'evidence' | 'ports' | 'remediation') => void;
}

export const useWorkspaceStore = create<WorkspaceState>((set) => ({
  activeTab: 'inventory',
  setActiveTab: (tab) => set({ activeTab: tab }),

  selectedScanId: null,
  setSelectedScanId: (id) => set({ selectedScanId: id }),

  selectedAsset: null,
  setSelectedAsset: (asset) => set({ selectedAsset: asset }),

  selectedFinding: null,
  setSelectedFinding: (finding) => set({ selectedFinding: finding }),

  isDockOpen: true,
  setIsDockOpen: (open) => set({ isDockOpen: open }),
  toggleDock: () => set((state) => ({ isDockOpen: !state.isDockOpen })),

  dockHeight: 260,
  setDockHeight: (height) => set({ dockHeight: height }),

  searchQuery: '',
  setSearchQuery: (q) => set({ searchQuery: q }),

  severityFilter: 'all',
  setSeverityFilter: (s) => set({ severityFilter: s }),

  commandPaletteOpen: false,
  setCommandPaletteOpen: (open) => set({ commandPaletteOpen: open }),

  activeInspectorTab: 'headers',
  setActiveInspectorTab: (tab) => set({ activeInspectorTab: tab }),
}));

