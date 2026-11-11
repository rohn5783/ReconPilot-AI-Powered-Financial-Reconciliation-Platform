import {
  ReconciliationSummary,
  ThreeWayTraceItem,
  PayoutBatchGroup,
  Discrepancy,
  AIDiagnosis,
  DiscrepancyStatus,
  PipelineTelemetryEvent,
} from '../types';

const API_BASE = import.meta.env.VITE_API_URL || '/api/v1';

export function getApiKey(): string {
  return import.meta.env.VITE_AUTH_API_KEY || localStorage.getItem('RECONPILOT_API_KEY') || '';
}

function getHeaders(extraHeaders: Record<string, string> = {}): HeadersInit {
  const headers: Record<string, string> = { ...extraHeaders };
  const key = getApiKey();
  if (key) {
    headers['X-API-Key'] = key;
  }
  return headers;
}

export const apiClient = {
  async getSummary(): Promise<ReconciliationSummary> {
    const res = await fetch(`${API_BASE}/summary`, { headers: getHeaders() });
    if (!res.ok) throw new Error(`Failed to fetch summary: ${res.statusText}`);
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
    if (!res.ok) throw new Error(`Failed to fetch traces: ${res.statusText}`);
    return res.json();
  },

  async getPayoutBatches(): Promise<{ data: PayoutBatchGroup[]; total: number }> {
    const res = await fetch(`${API_BASE}/payout-batches`, { headers: getHeaders() });
    if (!res.ok) throw new Error(`Failed to fetch payout batches: ${res.statusText}`);
    return res.json();
  },

  async getDiscrepancies(filters: Record<string, string> = {}): Promise<{ data: Discrepancy[]; total: number }> {
    const params = new URLSearchParams(filters);
    const res = await fetch(`${API_BASE}/discrepancies?${params.toString()}`, { headers: getHeaders() });
    if (!res.ok) throw new Error(`Failed to fetch discrepancies: ${res.statusText}`);
    return res.json();
  },

  async getDiscrepancy(id: string): Promise<Discrepancy> {
    const res = await fetch(`${API_BASE}/discrepancies/${id}`, { headers: getHeaders() });
    if (!res.ok) throw new Error(`Failed to fetch discrepancy ${id}: ${res.statusText}`);
    return res.json();
  },

  async diagnoseDiscrepancy(id: string): Promise<AIDiagnosis> {
    const res = await fetch(`${API_BASE}/discrepancies/${id}/diagnose`, {
      method: 'POST',
      headers: getHeaders(),
    });
    if (!res.ok) throw new Error(`Diagnosis failed for ${id}: ${res.statusText}`);
    return res.json();
  },

  async updateDiscrepancyStatus(id: string, status: DiscrepancyStatus): Promise<void> {
    const res = await fetch(`${API_BASE}/discrepancies/${id}/status`, {
      method: 'PATCH',
      headers: getHeaders({ 'Content-Type': 'application/json' }),
      body: JSON.stringify({ status }),
    });
    if (!res.ok) throw new Error(`Failed to update status for ${id}: ${res.statusText}`);
  },

  async triggerReconcile(formData?: FormData): Promise<any> {
    const res = await fetch(`${API_BASE}/reconcile`, {
      method: 'POST',
      headers: getHeaders(),
      body: formData,
    });
    if (!res.ok) throw new Error(`Reconciliation trigger failed: ${res.statusText}`);
    return res.json();
  },

  async seedData(): Promise<any> {
    const res = await fetch(`${API_BASE}/seed`, {
      method: 'POST',
      headers: getHeaders(),
    });
    if (!res.ok) throw new Error(`Seed failed: ${res.statusText}`);
    return res.json();
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

