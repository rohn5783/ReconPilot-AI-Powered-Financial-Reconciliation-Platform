import {
  ReconciliationSummary,
  ThreeWayTraceItem,
  PayoutBatchGroup,
  Discrepancy,
  AIDiagnosis,
  DiscrepancyStatus,
  PipelineTelemetryEvent,
  UserProfile,
  RegisterPayload,
  LoginPayload,
  AuthResponse,
  UploadRecord,
  ContractRule,
} from '../types';

export const LIVE_BACKEND_URL = 'https://reconpilot-backend-latest.onrender.com';

export function getApiBase(): string {
  if (typeof window !== 'undefined') {
    const custom = localStorage.getItem('RECONPILOT_BACKEND_URL');
    if (custom && custom.trim()) {
      return custom.trim().replace(/\/+$/, '');
    }
  }

  const envUrl = import.meta.env.VITE_API_URL;
  if (envUrl && envUrl.trim()) {
    // When running inside local Docker (served by Nginx at localhost:3000),
    // always use relative '/api/v1' so Nginx proxies directly to the backend container
    if (
      typeof window !== 'undefined' &&
      (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') &&
      envUrl.startsWith('http')
    ) {
      return '/api/v1';
    }
    return envUrl.trim().replace(/\/+$/, '');
  }

  if (typeof window !== 'undefined') {
    return '/api/v1';
  }

  return `${LIVE_BACKEND_URL}/api/v1`;
}

export const API_BASE = {
  toString() {
    return getApiBase();
  },
};

export function getApiKey(): string {
  return (
    import.meta.env.VITE_AUTH_API_KEY ||
    localStorage.getItem('RECONPILOT_API_KEY') ||
    'c6295744f18aba25b56fbe51834593ce23d29b6e89abd354ec5ce5724fec13dd'
  );
}

export function setApiKey(key: string): void {
  if (key) {
    localStorage.setItem('RECONPILOT_API_KEY', key.trim());
  } else {
    localStorage.removeItem('RECONPILOT_API_KEY');
  }
}

export function getAuthToken(): string | null {
  return localStorage.getItem('RECONPILOT_AUTH_TOKEN');
}

export function setAuthToken(token: string | null): void {
  if (token) {
    localStorage.setItem('RECONPILOT_AUTH_TOKEN', token.trim());
  } else {
    localStorage.removeItem('RECONPILOT_AUTH_TOKEN');
  }
}

async function extractErrorMessage(res: Response): Promise<string> {
  let msg = `HTTP ${res.status}: ${res.statusText}`;
  try {
    const body = await res.json();
    if (body.details) {
      msg = `${body.error}: ${body.details}`;
    } else if (body.message) {
      msg = body.message;
    } else if (body.error) {
      msg = body.error;
    }
  } catch {
    // ignore
  }
  return msg;
}

function getHeaders(extraHeaders: Record<string, string> = {}): HeadersInit {
  const headers: Record<string, string> = { ...extraHeaders };
  const token = getAuthToken();
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  const key = getApiKey();
  if (key) {
    headers['X-API-Key'] = key;
  }
  return headers;
}

export const apiClient = {
  async getSummary(): Promise<ReconciliationSummary> {
    const res = await fetch(`${API_BASE}/summary`, { headers: getHeaders() });
    if (!res.ok) throw new Error(await extractErrorMessage(res));
    return res.json();
  },

  async getTraces(
    search = '',
    status = '',
    limit = 50,
    offset = 0
  ): Promise<{ data: ThreeWayTraceItem[]; total: number; limit: number; offset: number }> {
    const params = new URLSearchParams();
    if (search) params.append('search', search);
    if (status) params.append('status', status);
    params.append('limit', limit.toString());
    params.append('offset', offset.toString());

    const res = await fetch(`${API_BASE}/traces?${params.toString()}`, { headers: getHeaders() });
    if (!res.ok) throw new Error(await extractErrorMessage(res));
    return res.json();
  },

  async getPayoutBatches(): Promise<{ data: PayoutBatchGroup[]; total: number }> {
    const res = await fetch(`${API_BASE}/payout-batches`, { headers: getHeaders() });
    if (!res.ok) throw new Error(await extractErrorMessage(res));
    return res.json();
  },

  async getDiscrepancies(filters: Record<string, string> = {}): Promise<{ data: Discrepancy[]; total: number }> {
    const params = new URLSearchParams(filters);
    const res = await fetch(`${API_BASE}/discrepancies?${params.toString()}`, { headers: getHeaders() });
    if (!res.ok) throw new Error(await extractErrorMessage(res));
    return res.json();
  },

  async getDiscrepancy(id: string): Promise<Discrepancy> {
    const res = await fetch(`${API_BASE}/discrepancies/${id}`, { headers: getHeaders() });
    if (!res.ok) throw new Error(await extractErrorMessage(res));
    return res.json();
  },

  async diagnoseDiscrepancy(id: string): Promise<AIDiagnosis> {
    const res = await fetch(`${API_BASE}/discrepancies/${id}/diagnose`, {
      method: 'POST',
      headers: getHeaders(),
    });
    if (!res.ok) throw new Error(await extractErrorMessage(res));
    return res.json();
  },

  async updateDiscrepancyStatus(id: string, status: DiscrepancyStatus): Promise<void> {
    const res = await fetch(`${API_BASE}/discrepancies/${id}/status`, {
      method: 'PATCH',
      headers: getHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify({ status }),
    });
    if (!res.ok) throw new Error(await extractErrorMessage(res));
  },

  async triggerReconcile(formData?: FormData): Promise<any> {
    const res = await fetch(`${API_BASE}/reconcile`, {
      method: 'POST',
      headers: getHeaders(),
      body: formData,
    });
    if (!res.ok) throw new Error(await extractErrorMessage(res));
    return res.json();
  },

  async resetData(): Promise<any> {
    const res = await fetch(`${API_BASE}/reset`, {
      method: 'POST',
      headers: getHeaders(),
    });
    if (!res.ok && res.status === 404) {
      // Fallback for deployments before /reset route was added
      const fallback = await fetch(`${API_BASE}/seed?mode=clear`, {
        method: 'POST',
        headers: getHeaders(),
      });
      if (!fallback.ok) throw new Error(await extractErrorMessage(fallback));
      return fallback.json();
    }
    if (!res.ok) throw new Error(await extractErrorMessage(res));
    return res.json();
  },

  async seedData(): Promise<any> {
    const res = await fetch(`${API_BASE}/seed`, {
      method: 'POST',
      headers: getHeaders(),
    });
    if (!res.ok) throw new Error(await extractErrorMessage(res));
    return res.json();
  },

  async register(payload: RegisterPayload): Promise<AuthResponse> {
    const res = await fetch(`${API_BASE}/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (!res.ok) throw new Error(await extractErrorMessage(res));
    const data: AuthResponse = await res.json();
    if (data?.token) {
      setAuthToken(data.token);
    }
    return data;
  },

  async login(payload: LoginPayload): Promise<AuthResponse> {
    const res = await fetch(`${API_BASE}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload),
    });
    if (!res.ok) throw new Error(await extractErrorMessage(res));
    const data: AuthResponse = await res.json();
    if (data?.token) {
      setAuthToken(data.token);
    }
    return data;
  },

  async getMe(): Promise<UserProfile> {
    const res = await fetch(`${API_BASE}/auth/me`, {
      headers: getHeaders(),
    });
    if (!res.ok) throw new Error(await extractErrorMessage(res));
    return res.json();
  },

  logout(): void {
    setAuthToken(null);
  },

  async getUploadHistory(): Promise<{ data: UploadRecord[]; total: number }> {
    const res = await fetch(`${API_BASE}/uploads`, { headers: getHeaders() });
    if (!res.ok) throw new Error(await extractErrorMessage(res));
    return res.json();
  },

  async getUploadRecord(id: string): Promise<UploadRecord> {
    const res = await fetch(`${API_BASE}/uploads/${encodeURIComponent(id)}`, { headers: getHeaders() });
    if (!res.ok) throw new Error(await extractErrorMessage(res));
    return res.json();
  },

  downloadUploadedFile(id: string, fileName: string): void {
    const token = getAuthToken();
    const key = getApiKey();
    const tokenParam = token ? `api_key=${encodeURIComponent(token)}` : (key ? `api_key=${encodeURIComponent(key)}` : '');
    const query = tokenParam ? `?${tokenParam}` : '';
    const url = `${API_BASE}/uploads/${encodeURIComponent(id)}/download${query}`;
    const link = document.createElement('a');
    link.href = url;
    link.setAttribute('download', fileName);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  },

  async getContractRules(): Promise<{ data: ContractRule[]; total: number }> {
    const res = await fetch(`${API_BASE}/contract-rules`, { headers: getHeaders() });
    if (!res.ok) throw new Error(await extractErrorMessage(res));
    return res.json();
  },

  async createContractRule(payload: Partial<ContractRule>): Promise<ContractRule> {
    const res = await fetch(`${API_BASE}/contract-rules`, {
      method: 'POST',
      headers: getHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify(payload),
    });
    if (!res.ok) throw new Error(await extractErrorMessage(res));
    return res.json();
  },

  async updateContractRule(id: string, payload: Partial<ContractRule>): Promise<ContractRule> {
    const res = await fetch(`${API_BASE}/contract-rules/${encodeURIComponent(id)}`, {
      method: 'PUT',
      headers: getHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify(payload),
    });
    if (!res.ok) throw new Error(await extractErrorMessage(res));
    return res.json();
  },

  async activateContractRule(id: string): Promise<{ status: string; rule: ContractRule }> {
    const res = await fetch(`${API_BASE}/contract-rules/${encodeURIComponent(id)}/activate`, {
      method: 'POST',
      headers: getHeaders(),
    });
    if (!res.ok) throw new Error(await extractErrorMessage(res));
    return res.json();
  },

  async deleteContractRule(id: string): Promise<void> {
    const res = await fetch(`${API_BASE}/contract-rules/${encodeURIComponent(id)}`, {
      method: 'DELETE',
      headers: getHeaders(),
    });
    if (!res.ok) throw new Error(await extractErrorMessage(res));
  },

  exportReport(format: 'csv' | 'json', target: 'traces' | 'discrepancies' = 'traces'): void {
    const key = getApiKey();
    const query = key ? `&api_key=${encodeURIComponent(key)}` : '';
    const url = `${API_BASE}/export?format=${format}&target=${target}${query}`;
    const link = document.createElement('a');
    link.href = url;
    link.setAttribute('download', `reconpilot-${target}.${format}`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  },

  subscribeTelemetry(onEvent: (event: PipelineTelemetryEvent) => void): () => void {
    const key = getApiKey();
    const query = key ? `?api_key=${encodeURIComponent(key)}` : '';
    const sseUrl = `${API_BASE}/reconcile/stream${query}`;
    const eventSource = new EventSource(sseUrl);

    eventSource.onmessage = (event) => {
      try {
        const parsed = JSON.parse(event.data);
        onEvent(parsed);
      } catch (err) {
        console.error('Failed to parse SSE telemetry event:', err);
      }
    };

    eventSource.onerror = (err) => {
      console.warn('SSE connection interrupted, retrying...', err);
    };

    return () => {
      eventSource.close();
    };
  },
};

