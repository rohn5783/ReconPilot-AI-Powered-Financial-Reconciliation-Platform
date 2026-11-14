import { create } from 'zustand';
import {
  ReconciliationSummary,
  ThreeWayTraceItem,
  PayoutBatchGroup,
  Discrepancy,
  DiscrepancyStatus,
  PipelineTelemetryEvent,
  UserProfile,
  RegisterPayload,
  LoginPayload,
  UploadRecord,
} from '../types';
import { apiClient, getApiKey, setApiKey, getAuthToken } from '../api/client';

export type WorkstationView = 'explorer' | 'batches' | 'discrepancies' | 'upload' | 'history';

interface ReconState {
  summary: ReconciliationSummary | null;
  traces: ThreeWayTraceItem[];
  tracesTotal: number;
  payoutBatches: PayoutBatchGroup[];
  discrepancies: Discrepancy[];
  selectedDiscrepancy: Discrepancy | null;
  activeView: WorkstationView;

  // Upload History & Audit Ingestion State
  uploadHistory: UploadRecord[];
  isLoadingUploads: boolean;

  // Authentication State
  currentUser: UserProfile | null;
  authToken: string | null;
  isAuthModalOpen: boolean;
  authModalTab: 'login' | 'register';
  authError: string | null;
  isAuthenticating: boolean;

  // Search & Filters
  searchQuery: string;
  statusFilter: 'all' | 'reconciled' | 'discrepancy';
  levelFilter: 'all' | 'LEVEL_1_ORDER_PG' | 'LEVEL_2_PG_BANK';
  severityFilter: 'all' | 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW';
  discrepancyStatusFilter: 'all' | 'OPEN' | 'IN_DISPUTE' | 'RESOLVED';
  discrepancyTypeFilter: string;

  // Real-time Pipeline & Telemetry
  isLoading: boolean;
  isReconciling: boolean;
  telemetryLogs: PipelineTelemetryEvent[];
  currentProgress: number;
  currentStage: string;
  currentMessage: string;
  isTelemetryDockOpen: boolean;
  isCommandPaletteOpen: boolean;
  lastError: string | null;
  apiKey: string;

  // Actions
  fetchSummary: () => Promise<void>;
  fetchTraces: () => Promise<void>;
  fetchPayoutBatches: () => Promise<void>;
  fetchDiscrepancies: () => Promise<void>;
  fetchAllData: () => Promise<void>;
  selectDiscrepancy: (d: Discrepancy | null) => void;
  diagnoseSelectedDiscrepancy: () => Promise<void>;
  updateSelectedDiscrepancyStatus: (status: DiscrepancyStatus) => Promise<void>;
  triggerReconcile: (formData?: FormData) => Promise<boolean>;
  resetData: () => Promise<void>;
  seedData: () => Promise<void>;
  addTelemetryEvent: (evt: PipelineTelemetryEvent) => void;
  setSearchQuery: (q: string) => void;
  setStatusFilter: (f: 'all' | 'reconciled' | 'discrepancy') => void;
  setLevelFilter: (l: 'all' | 'LEVEL_1_ORDER_PG' | 'LEVEL_2_PG_BANK') => void;
  setSeverityFilter: (s: 'all' | 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW') => void;
  setDiscrepancyStatusFilter: (st: 'all' | 'OPEN' | 'IN_DISPUTE' | 'RESOLVED') => void;
  setDiscrepancyTypeFilter: (t: string) => void;
  setActiveView: (v: WorkstationView) => void;
  setCommandPaletteOpen: (open: boolean) => void;
  toggleTelemetryDock: () => void;
  setLastError: (err: string | null) => void;
  saveApiKey: (key: string) => void;

  // Upload History Actions
  fetchUploadHistory: () => Promise<void>;
  downloadUploadFile: (id: string, fileName: string) => void;

  // Authentication Actions
  openAuthModal: (tab?: 'login' | 'register') => void;
  closeAuthModal: () => void;
  login: (payload: LoginPayload) => Promise<boolean>;
  register: (payload: RegisterPayload) => Promise<boolean>;
  logout: () => void;
  checkAuth: () => Promise<void>;
}

export const useReconStore = create<ReconState>((set, get) => ({
  summary: null,
  traces: [],
  tracesTotal: 0,
  payoutBatches: [],
  discrepancies: [],
  selectedDiscrepancy: null,
  activeView: 'explorer',

  // Upload History & Audit Ingestion State
  uploadHistory: [],
  isLoadingUploads: false,

  // Authentication State
  currentUser: null,
  authToken: getAuthToken(),
  isAuthModalOpen: false,
  authModalTab: 'login',
  authError: null,
  isAuthenticating: false,

  searchQuery: '',
  statusFilter: 'all',
  levelFilter: 'all',
  severityFilter: 'all',
  discrepancyStatusFilter: 'all',
  discrepancyTypeFilter: 'all',

  isLoading: false,
  isReconciling: false,
  telemetryLogs: [],
  currentProgress: 100,
  currentStage: 'COMPLETED',
  currentMessage: 'Platform idle. Ready for reconciliation runs.',
  isTelemetryDockOpen: false,
  isCommandPaletteOpen: false,
  lastError: null,
  apiKey: getApiKey(),

  fetchSummary: async () => {
    try {
      const summary = await apiClient.getSummary();
      set({ summary });
    } catch (err: any) {
      console.error('Failed to fetch summary:', err);
      set({ lastError: err?.message || 'Failed to fetch summary' });
    }
  },

  fetchTraces: async () => {
    const { searchQuery, statusFilter } = get();
    try {
      const statusParam = statusFilter === 'all' ? '' : statusFilter;
      const res = await apiClient.getTraces(searchQuery, statusParam, 100, 0);
      set({ traces: res?.data || [], tracesTotal: res?.total || 0 });
    } catch (err: any) {
      console.error('Failed to fetch traces:', err);
      set({ lastError: err?.message || 'Failed to fetch traces' });
    }
  },

  fetchPayoutBatches: async () => {
    try {
      const res = await apiClient.getPayoutBatches();
      set({ payoutBatches: res?.data || [] });
    } catch (err: any) {
      console.error('Failed to fetch payout batches:', err);
    }
  },

  fetchDiscrepancies: async () => {
    const { levelFilter, severityFilter, discrepancyStatusFilter, discrepancyTypeFilter } = get();
    try {
      const filters: Record<string, string> = {};
      if (levelFilter !== 'all') filters.level = levelFilter;
      if (severityFilter !== 'all') filters.severity = severityFilter;
      if (discrepancyStatusFilter !== 'all') filters.status = discrepancyStatusFilter;
      if (discrepancyTypeFilter !== 'all') filters.type = discrepancyTypeFilter;
      const res = await apiClient.getDiscrepancies(filters);
      set({ discrepancies: res?.data || [] });
    } catch (err: any) {
      console.error('Failed to fetch discrepancies:', err);
    }
  },

  fetchAllData: async () => {
    set({ isLoading: true });
    await Promise.all([
      get().fetchSummary(),
      get().fetchTraces(),
      get().fetchPayoutBatches(),
      get().fetchDiscrepancies(),
      get().fetchUploadHistory(),
    ]);
    set({ isLoading: false });
  },

  selectDiscrepancy: (d) => {
    set({ selectedDiscrepancy: d });
  },

  diagnoseSelectedDiscrepancy: async () => {
    const { selectedDiscrepancy } = get();
    if (!selectedDiscrepancy) return;
    try {
      set({ isLoading: true });
      const diagnosis = await apiClient.diagnoseDiscrepancy(selectedDiscrepancy.id);
      const updated: Discrepancy = { ...selectedDiscrepancy, ai_diagnosis: diagnosis };
      set((state) => ({
        selectedDiscrepancy: updated,
        discrepancies: state.discrepancies.map((d) => (d.id === updated.id ? updated : d)),
      }));
    } catch (err) {
      console.error('Diagnosis failed:', err);
    } finally {
      set({ isLoading: false });
    }
  },

  updateSelectedDiscrepancyStatus: async (status: DiscrepancyStatus) => {
    const { selectedDiscrepancy } = get();
    if (!selectedDiscrepancy) return;
    try {
      await apiClient.updateDiscrepancyStatus(selectedDiscrepancy.id, status);
      const updated: Discrepancy = { ...selectedDiscrepancy, status };
      set((state) => ({
        selectedDiscrepancy: updated,
        discrepancies: state.discrepancies.map((d) => (d.id === updated.id ? updated : d)),
      }));
    } catch (err) {
      console.error('Update status failed:', err);
    }
  },

  triggerReconcile: async (formData?: FormData): Promise<boolean> => {
    set({
      isReconciling: true,
      currentProgress: 5,
      currentStage: 'INGESTION',
      currentMessage: 'Initiating reconciliation pipeline...',
      isTelemetryDockOpen: true,
      lastError: null,
    });
    try {
      await apiClient.triggerReconcile(formData);
      await get().fetchAllData();
      return true;
    } catch (err: any) {
      console.error('Trigger reconcile failed:', err);
      set({ lastError: err?.message || 'Reconciliation execution failed' });
      return false;
    } finally {
      set({ isReconciling: false });
    }
  },

  resetData: async () => {
    set({
      isReconciling: true,
      currentProgress: 20,
      currentStage: 'INGESTION',
      currentMessage: 'Erasing all financial transactions, traces, and discrepancies...',
      isTelemetryDockOpen: true,
      lastError: null,
    });
    try {
      await apiClient.resetData();
      // Immediately wipe and zero out all UI state
      set({
        summary: {
          total_orders: 0,
          total_gross_order_volume: '0',
          total_pg_settlements: 0,
          total_pg_fees: '0',
          total_pg_tax: '0',
          expected_net_payout: '0',
          total_bank_credits: '0',
          net_discrepancy_variance: '0',
          level_1_discrepancies_count: 0,
          level_2_discrepancies_count: 0,
          three_way_match_rate: '100',
          reconciled_orders_count: 0,
          unreconciled_orders_count: 0,
        },
        traces: [],
        tracesTotal: 0,
        payoutBatches: [],
        discrepancies: [],
        uploadHistory: [],
        selectedDiscrepancy: null,
        currentProgress: 100,
        currentStage: 'COMPLETED',
        currentMessage: 'Ledger erased. All transactions cleared from UI and storage.',
      });
      await get().fetchAllData();
    } catch (err: any) {
      console.error('Reset data failed:', err);
      set({ lastError: err?.message || 'Failed to erase data' });
    } finally {
      set({ isReconciling: false });
    }
  },

  seedData: async () => {
    set({
      isReconciling: true,
      currentProgress: 10,
      currentStage: 'INGESTION',
      currentMessage: 'Resetting and seeding institutional demo ledger...',
      isTelemetryDockOpen: true,
    });
    try {
      await apiClient.seedData();
      await get().fetchAllData();
    } catch (err) {
      console.error('Seed demo data failed:', err);
    } finally {
      set({ isReconciling: false });
    }
  },

  addTelemetryEvent: (evt: PipelineTelemetryEvent) => {
    set((state) => {
      const logs = [...(state.telemetryLogs || []), evt];
      if (logs.length > 200) logs.shift();
      return {
        telemetryLogs: logs,
        currentProgress: evt.progress_percentage || state.currentProgress,
        currentStage: evt.stage || state.currentStage,
        currentMessage: evt.message || state.currentMessage,
        isReconciling: evt.stage !== 'COMPLETED',
      };
    });
    if (evt.type === 'completed') {
      get().fetchAllData();
    }
  },

  setSearchQuery: (q: string) => {
    set({ searchQuery: q });
    get().fetchTraces();
  },

  setStatusFilter: (f) => {
    set({ statusFilter: f });
    get().fetchTraces();
  },

  setLevelFilter: (l) => {
    set({ levelFilter: l });
    get().fetchDiscrepancies();
  },

  setSeverityFilter: (s) => {
    set({ severityFilter: s });
    get().fetchDiscrepancies();
  },

  setDiscrepancyStatusFilter: (st) => {
    set({ discrepancyStatusFilter: st });
    get().fetchDiscrepancies();
  },

  setDiscrepancyTypeFilter: (t) => {
    set({ discrepancyTypeFilter: t });
    get().fetchDiscrepancies();
  },

  setActiveView: (v: WorkstationView) => set({ activeView: v }),
  setCommandPaletteOpen: (open: boolean) => set({ isCommandPaletteOpen: open }),
  toggleTelemetryDock: () => set((state) => ({ isTelemetryDockOpen: !state.isTelemetryDockOpen })),
  setLastError: (err: string | null) => set({ lastError: err }),
  saveApiKey: (key: string) => {
    setApiKey(key);
    set({ apiKey: key, lastError: null });
    get().fetchAllData();
  },

  fetchUploadHistory: async () => {
    set({ isLoadingUploads: true });
    try {
      const res = await apiClient.getUploadHistory();
      set({ uploadHistory: res?.data || [], isLoadingUploads: false });
    } catch (err: any) {
      console.error('Failed to fetch upload history:', err);
      set({ isLoadingUploads: false });
    }
  },

  downloadUploadFile: (id: string, fileName: string) => {
    apiClient.downloadUploadedFile(id, fileName);
  },

  openAuthModal: (tab = 'login') => set({ isAuthModalOpen: true, authModalTab: tab, authError: null }),
  closeAuthModal: () => set({ isAuthModalOpen: false, authError: null }),

  login: async (payload: LoginPayload) => {
    set({ isAuthenticating: true, authError: null });
    try {
      const resp = await apiClient.login(payload);
      set({
        currentUser: resp.user,
        authToken: resp.token,
        isAuthModalOpen: false,
        isAuthenticating: false,
        authError: null,
      });
      get().fetchAllData();
      return true;
    } catch (err: any) {
      set({
        authError: err?.message || 'Login failed. Please verify credentials.',
        isAuthenticating: false,
      });
      return false;
    }
  },

  register: async (payload: RegisterPayload) => {
    set({ isAuthenticating: true, authError: null });
    try {
      const resp = await apiClient.register(payload);
      set({
        currentUser: resp.user,
        authToken: resp.token,
        isAuthModalOpen: false,
        isAuthenticating: false,
        authError: null,
      });
      get().fetchAllData();
      return true;
    } catch (err: any) {
      set({
        authError: err?.message || 'Registration failed.',
        isAuthenticating: false,
      });
      return false;
    }
  },

  logout: () => {
    apiClient.logout();
    set({
      currentUser: null,
      authToken: null,
    });
  },

  checkAuth: async () => {
    const token = getAuthToken();
    if (!token) return;
    try {
      const user = await apiClient.getMe();
      set({ currentUser: user });
    } catch (err) {
      console.warn('Session expired or invalid:', err);
      apiClient.logout();
      set({ currentUser: null, authToken: null });
    }
  },
}));

