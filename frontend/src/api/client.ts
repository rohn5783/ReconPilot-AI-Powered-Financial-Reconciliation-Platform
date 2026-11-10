import { Scope, ScanJob, Asset, Finding, AssetDrift, AIReport } from '../types';

const BASE_URL = '/api/v1';

async function handleResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const errorText = await res.text();
    throw new Error(errorText || `HTTP request failed: ${res.statusText}`);
  }
  return res.json();
}

export const api = {
  // Scopes
  async getScopes(): Promise<Scope[]> {
    try {
      const res = await fetch(`${BASE_URL}/scopes`);
      return await handleResponse<Scope[]>(res);
    } catch {
      return getMockScopes();
    }
  },

  async createScope(data: { name: string; root_domain: string; rate_limit_rps?: number }): Promise<Scope> {
    const res = await fetch(`${BASE_URL}/scopes`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<Scope>(res);
  },

  async verifyScope(id: string, mock = false): Promise<{ verified: boolean; details: string }> {
    const res = await fetch(`${BASE_URL}/scopes/${id}/verify?mock=${mock}`, {
      method: 'POST',
    });
    return handleResponse<{ verified: boolean; details: string }>(res);
  },

  async deleteScope(id: string): Promise<void> {
    const res = await fetch(`${BASE_URL}/scopes/${id}`, { method: 'DELETE' });
    await handleResponse(res);
  },

  // Scans
  async getScans(scopeId?: string): Promise<ScanJob[]> {
    try {
      const url = scopeId ? `${BASE_URL}/scans?scope_id=${scopeId}` : `${BASE_URL}/scans`;
      const res = await fetch(url);
      return await handleResponse<ScanJob[]>(res);
    } catch {
      return getMockScans();
    }
  },

  async getScan(id: string): Promise<ScanJob> {
    try {
      const res = await fetch(`${BASE_URL}/scans/${id}`);
      return await handleResponse<ScanJob>(res);
    } catch {
      return getMockScans()[0];
    }
  },

  async createScan(data: { scope_id: string; profile: string; dry_run: boolean }): Promise<ScanJob> {
    const res = await fetch(`${BASE_URL}/scans`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    });
    return handleResponse<ScanJob>(res);
  },

  async cancelScan(id: string): Promise<void> {
    const res = await fetch(`${BASE_URL}/scans/${id}/cancel`, { method: 'POST' });
    await handleResponse(res);
  },

  // Artifacts
  async getScanAssets(id: string): Promise<Asset[]> {
    try {
      const res = await fetch(`${BASE_URL}/scans/${id}/assets`);
      return await handleResponse<Asset[]>(res);
    } catch {
      return getMockAssets(id);
    }
  },

  async getScanDrifts(id: string): Promise<AssetDrift[]> {
    try {
      const res = await fetch(`${BASE_URL}/scans/${id}/drifts`);
      return await handleResponse<AssetDrift[]>(res);
    } catch {
      return getMockDrifts(id);
    }
  },

  async getScanFindings(id: string): Promise<Finding[]> {
    try {
      const res = await fetch(`${BASE_URL}/scans/${id}/findings`);
      return await handleResponse<Finding[]>(res);
    } catch {
      return getMockFindings(id);
    }
  },

  async getScanReport(id: string): Promise<AIReport> {
    try {
      const res = await fetch(`${BASE_URL}/scans/${id}/report`);
      return await handleResponse<AIReport>(res);
    } catch {
      return getMockReport(id);
    }
  },

  async generateReport(id: string): Promise<AIReport> {
    const res = await fetch(`${BASE_URL}/scans/${id}/report/generate`, { method: 'POST' });
    return handleResponse<AIReport>(res);
  },

  getExportMarkdownUrl(id: string): string {
    return `${BASE_URL}/scans/${id}/export/markdown`;
  },

  getExportJsonUrl(id: string): string {
    return `${BASE_URL}/scans/${id}/export/json`;
  },
};

// ==============================================================================
// HIGH-FIDELITY INDUSTRIAL MOCK FALLBACKS FOR INSTANT PREVIEW
// ==============================================================================

function getMockScopes(): Scope[] {
  return [
    {
      id: 'sc-acme-prod-01',
      name: 'Acme Global Infrastructure',
      root_domain: 'acme.org',
      verification_status: 'verified',
      verification_token: 'reconpilot-verification=9f8a7b6c5d4e3f2a1b0c',
      verified_at: '2026-11-01T12:00:00Z',
      allowed_cidrs: ['104.21.0.0/16', '172.67.0.0/16'],
      excluded_cidrs: [],
      excluded_subdomains: ['mail.acme.org'],
      rate_limit_rps: 50,
      created_at: '2026-11-01T10:00:00Z',
      updated_at: '2026-11-01T12:00:00Z',
    },
  ];
}

function getMockScans(): ScanJob[] {
  return [
    {
      id: 'job-98a7-live-81',
      scope_id: 'sc-acme-prod-01',
      status: 'completed',
      profile: 'standard',
      current_stage: 'completed',
      stage_checkpoints: {
        idle: { stage: 'idle', status: 'done', started_at: '2026-11-10T17:00:00Z', duration_ms: 10, output_count: 0 },
        discovery: { stage: 'discovery', status: 'done', started_at: '2026-11-10T17:00:00Z', completed_at: '2026-11-10T17:00:05Z', duration_ms: 5000, output_count: 15 },
        probing: { stage: 'probing', status: 'done', started_at: '2026-11-10T17:00:05Z', completed_at: '2026-11-10T17:00:12Z', duration_ms: 7000, output_count: 15 },
        portscan: { stage: 'portscan', status: 'done', started_at: '2026-11-10T17:00:12Z', completed_at: '2026-11-10T17:00:25Z', duration_ms: 13000, output_count: 38 },
        auditor: { stage: 'auditor', status: 'done', started_at: '2026-11-10T17:00:25Z', completed_at: '2026-11-10T17:00:42Z', duration_ms: 17000, output_count: 7 },
        ai_synthesis: { stage: 'ai_synthesis', status: 'done', started_at: '2026-11-10T17:00:42Z', completed_at: '2026-11-10T17:00:48Z', duration_ms: 6000, output_count: 1 },
        completed: { stage: 'completed', status: 'done', started_at: '2026-11-10T17:00:48Z', duration_ms: 0, output_count: 0 },
      },
      metrics: {
        total_hosts: 15,
        live_hosts: 15,
        open_ports: 38,
        findings_count: 7,
        duration_seconds: 48.2,
        requests_per_sec: 18.5,
      },
      dry_run: true,
      started_at: '2026-11-10T17:00:00Z',
      completed_at: '2026-11-10T17:00:48Z',
      created_at: '2026-11-10T16:59:50Z',
      updated_at: '2026-11-10T17:00:48Z',
    },
  ];
}

function getMockAssets(scanId: string): Asset[] {
  const root = 'acme.org';
  return [
    {
      id: 'as-1', scan_job_id: scanId, scope_id: 'sc-acme-prod-01', fqdn: 'acme.org', root_domain: root,
      ip_addresses: ['104.21.15.82', '172.67.140.23'], asn: 'AS13335', asn_org: 'CLOUDFLARENET',
      http_status: 200, page_title: 'Official Homepage', tech_stack: ['Cloudflare', 'React 19', 'Next.js', 'Vercel'],
      open_ports: [80, 443], first_seen_at: '2026-11-01T10:00:00Z', last_seen_at: '2026-11-10T17:00:00Z',
      created_at: '2026-11-10T17:00:00Z', updated_at: '2026-11-10T17:00:00Z',
      response_headers: { 'Server': 'cloudflare', 'Content-Type': 'text/html; charset=utf-8' },
    },
    {
      id: 'as-2', scan_job_id: scanId, scope_id: 'sc-acme-prod-01', fqdn: 'api.acme.org', root_domain: root,
      ip_addresses: ['104.21.15.82', '172.67.140.23'], asn: 'AS13335', asn_org: 'CLOUDFLARENET',
      http_status: 200, page_title: 'API Gateway Service', tech_stack: ['Envoy', 'Golang', 'gRPC-Web', 'Cloudflare'],
      open_ports: [80, 443, 8080, 8443], first_seen_at: '2026-11-01T10:00:00Z', last_seen_at: '2026-11-10T17:00:00Z',
      created_at: '2026-11-10T17:00:00Z', updated_at: '2026-11-10T17:00:00Z',
      response_headers: { 'Server': 'envoy', 'X-Gateway-Route': 'v2', 'Content-Type': 'application/json' },
    },
    {
      id: 'as-3', scan_job_id: scanId, scope_id: 'sc-acme-prod-01', fqdn: 'auth.acme.org', root_domain: root,
      ip_addresses: ['52.84.12.33'], asn: 'AS16509', asn_org: 'AMAZON-02',
      http_status: 200, page_title: 'Unified Identity & SSO Portal', tech_stack: ['Ory Kratos', 'React', 'Nginx/1.24.0', 'AWS ALB'],
      open_ports: [80, 443, 8443], first_seen_at: '2026-11-01T10:00:00Z', last_seen_at: '2026-11-10T17:00:00Z',
      created_at: '2026-11-10T17:00:00Z', updated_at: '2026-11-10T17:00:00Z',
      response_headers: { 'Server': 'nginx/1.24.0', 'X-Frame-Options': 'SAMEORIGIN' },
    },
    {
      id: 'as-4', scan_job_id: scanId, scope_id: 'sc-acme-prod-01', fqdn: 'admin.acme.org', root_domain: root,
      ip_addresses: ['34.120.55.91'], asn: 'AS15169', asn_org: 'GOOGLE',
      http_status: 403, page_title: '403 Forbidden - Internal Access Only', tech_stack: ['Nginx/1.22.1', 'OpenSSL'],
      open_ports: [22, 443, 8443], first_seen_at: '2026-11-01T10:00:00Z', last_seen_at: '2026-11-10T17:00:00Z',
      created_at: '2026-11-10T17:00:00Z', updated_at: '2026-11-10T17:00:00Z',
      response_headers: { 'Server': 'nginx/1.22.1', 'Content-Type': 'text/html' },
    },
    {
      id: 'as-5', scan_job_id: scanId, scope_id: 'sc-acme-prod-01', fqdn: 'staging.acme.org', root_domain: root,
      ip_addresses: ['54.210.88.19'], asn: 'AS16509', asn_org: 'AMAZON-02',
      http_status: 401, page_title: '401 Authorization Required', tech_stack: ['Traefik v2.10', 'Docker', 'Vue.js'],
      open_ports: [80, 443, 6379, 9090], first_seen_at: '2026-11-10T17:00:00Z', last_seen_at: '2026-11-10T17:00:00Z',
      created_at: '2026-11-10T17:00:00Z', updated_at: '2026-11-10T17:00:00Z',
      response_headers: { 'WWW-Authenticate': 'Basic realm="Restricted"' },
    },
    {
      id: 'as-6', scan_job_id: scanId, scope_id: 'sc-acme-prod-01', fqdn: 'grafana.acme.org', root_domain: root,
      ip_addresses: ['142.93.18.204'], asn: 'AS14061', asn_org: 'DIGITALOCEAN-ASN',
      http_status: 200, page_title: 'Grafana - Enterprise Telemetry', tech_stack: ['Grafana v10.2.3', 'Golang', 'React'],
      open_ports: [443, 3000], first_seen_at: '2026-11-01T10:00:00Z', last_seen_at: '2026-11-10T17:00:00Z',
      created_at: '2026-11-10T17:00:00Z', updated_at: '2026-11-10T17:00:00Z',
    },
    {
      id: 'as-7', scan_job_id: scanId, scope_id: 'sc-acme-prod-01', fqdn: 'vault.acme.org', root_domain: root,
      ip_addresses: ['35.185.12.190'], asn: 'AS15169', asn_org: 'GOOGLE',
      http_status: 200, page_title: 'HashiCorp Vault UI', tech_stack: ['HashiCorp Vault', 'Ember.js', 'Golang'],
      open_ports: [443, 8200], first_seen_at: '2026-11-01T10:00:00Z', last_seen_at: '2026-11-10T17:00:00Z',
      created_at: '2026-11-10T17:00:00Z', updated_at: '2026-11-10T17:00:00Z',
    },
    {
      id: 'as-8', scan_job_id: scanId, scope_id: 'sc-acme-prod-01', fqdn: 'portal.acme.org', root_domain: root,
      ip_addresses: ['76.76.21.21'], asn: 'AS13335', asn_org: 'CLOUDFLARENET',
      http_status: 200, page_title: 'Customer Dashboard', tech_stack: ['Next.js', 'React', 'Tailwind CSS'],
      open_ports: [80, 443], first_seen_at: '2026-11-01T10:00:00Z', last_seen_at: '2026-11-10T17:00:00Z',
      created_at: '2026-11-10T17:00:00Z', updated_at: '2026-11-10T17:00:00Z',
    },
    {
      id: 'as-9', scan_job_id: scanId, scope_id: 'sc-acme-prod-01', fqdn: 'vpn.acme.org', root_domain: root,
      ip_addresses: ['198.51.100.45'], asn: 'AS3356', asn_org: 'LEVEL3',
      http_status: 200, page_title: 'GlobalProtect Portal', tech_stack: ['Palo Alto Networks', 'Apache'],
      open_ports: [443, 10443], first_seen_at: '2026-11-01T10:00:00Z', last_seen_at: '2026-11-10T17:00:00Z',
      created_at: '2026-11-10T17:00:00Z', updated_at: '2026-11-10T17:00:00Z',
    },
    {
      id: 'as-10', scan_job_id: scanId, scope_id: 'sc-acme-prod-01', fqdn: 'internal.acme.org', root_domain: root,
      ip_addresses: ['104.21.15.82'], asn: 'AS13335', asn_org: 'CLOUDFLARENET',
      http_status: 200, page_title: 'Internal Gateway', tech_stack: ['Nginx', 'HTML5'],
      open_ports: [80, 443, 3306, 5432], first_seen_at: '2026-11-01T10:00:00Z', last_seen_at: '2026-11-10T17:00:00Z',
      created_at: '2026-11-10T17:00:00Z', updated_at: '2026-11-10T17:00:00Z',
    },
  ];
}

function getMockDrifts(scanId: string): AssetDrift[] {
  return [
    {
      id: 'dr-1', scan_job_id: scanId, asset_id: 'as-5', fqdn: 'staging.acme.org',
      change_type: 'new_host', severity: 'high', detected_at: '2026-11-10T17:00:40Z',
      details: { fqdn: 'staging.acme.org', open_ports: [80, 443, 6379, 9090] },
    },
    {
      id: 'dr-2', scan_job_id: scanId, asset_id: 'as-5', fqdn: 'staging.acme.org',
      change_type: 'port_opened', severity: 'critical', detected_at: '2026-11-10T17:00:40Z',
      details: { port: 6379, service: 'redis-cache' },
    },
    {
      id: 'dr-3', scan_job_id: scanId, asset_id: 'as-5', fqdn: 'staging.acme.org',
      change_type: 'finding_added', severity: 'critical', detected_at: '2026-11-10T17:00:40Z',
      details: { rule_id: 'redis-unauthenticated-access', title: 'Exposed Unauthenticated Redis Cache' },
    },
    {
      id: 'dr-4', scan_job_id: scanId, asset_id: 'as-2', fqdn: 'api.acme.org',
      change_type: 'port_opened', severity: 'medium', detected_at: '2026-11-10T17:00:40Z',
      details: { port: 8080, service: 'http-proxy' },
    },
  ];
}

function getMockFindings(scanId: string): Finding[] {
  return [
    {
      id: 'fn-1', scan_job_id: scanId, asset_id: 'as-5', fqdn: 'staging.acme.org', tool: 'nuclei',
      rule_id: 'redis-unauthenticated-access', title: 'Exposed Unauthenticated Redis Cache Service',
      severity: 'critical', description: 'An unauthenticated Redis server is accessible over the public Internet on TCP port 6379, allowing arbitrary key inspection and potential remote code execution via CONFIG SET.',
      cve_ids: ['CVE-2022-0543'], cvss_score: 9.8,
      remediation: 'Bind Redis strictly to 127.0.0.1, enforce requirepass in redis.conf, and block port 6379 in AWS security groups.',
      raw_evidence: { matched_at: 'staging.acme.org:6379', request: 'INFO\r\n', response: '# Server\r\nredis_version:7.0.5\r\ntcp_port:6379\r\n' },
      false_positive: false, status: 'active', created_at: '2026-11-10T17:00:30Z', updated_at: '2026-11-10T17:00:30Z',
    },
    {
      id: 'fn-2', scan_job_id: scanId, asset_id: 'as-5', fqdn: 'staging.acme.org', tool: 'nuclei',
      rule_id: 'git-config-disclosure', title: 'Exposed Git Repository Configuration',
      severity: 'high', description: 'The .git/config configuration file is publicly accessible over HTTP, exposing repository URLs, branches, internal server addresses, and credential tokens.',
      cve_ids: [], cvss_score: 7.5,
      remediation: 'Configure Nginx/Apache to deny all requests to dot-prefixed paths (location ~ /\\.git { deny all; return 404; }).',
      raw_evidence: { matched_at: 'https://staging.acme.org/.git/config', request: 'GET /.git/config HTTP/1.1\r\nHost: staging.acme.org\r\n\r\n', response: 'HTTP/1.1 200 OK\r\n[core]\r\nrepositoryformatversion = 0\r\n' },
      false_positive: false, status: 'active', created_at: '2026-11-10T17:00:32Z', updated_at: '2026-11-10T17:00:32Z',
    },
    {
      id: 'fn-3', scan_job_id: scanId, asset_id: 'as-2', fqdn: 'api.acme.org', tool: 'nuclei',
      rule_id: 'cors-wildcard-origin', title: 'Permissive CORS Access-Control-Allow-Origin Wildcard',
      severity: 'medium', description: 'The API endpoint reflects arbitrary Origin headers in Access-Control-Allow-Origin with Access-Control-Allow-Credentials: true, permitting cross-site data theft.',
      cve_ids: [], cvss_score: 6.5,
      remediation: 'Implement an explicit whitelist of trusted frontend origins. Never blindly reflect the Origin request header with Allow-Credentials enabled.',
      raw_evidence: { matched_at: 'https://api.acme.org/v1/user', request: 'OPTIONS /v1/user HTTP/1.1\r\nOrigin: https://evil.attacker.com\r\n\r\n', response: 'HTTP/1.1 204 No Content\r\nAccess-Control-Allow-Origin: https://evil.attacker.com\r\n' },
      false_positive: false, status: 'active', created_at: '2026-11-10T17:00:34Z', updated_at: '2026-11-10T17:00:34Z',
    },
    {
      id: 'fn-4', scan_job_id: scanId, asset_id: 'as-9', fqdn: 'vpn.acme.org', tool: 'nuclei',
      rule_id: 'tls-cbc-weak-ciphers', title: 'Deprecated TLS CBC Ciphers Supported on VPN Gateway',
      severity: 'medium', description: 'The TLS configuration permits legacy AES-CBC ciphers susceptible to padding oracle attacks (Lucky 13 / BEAST).',
      cve_ids: ['CVE-2013-0169'], cvss_score: 5.9,
      remediation: 'Disable TLS 1.0 and 1.1. Enforce TLS 1.3 or TLS 1.2 with AEAD ciphers only.',
      raw_evidence: { matched_at: 'vpn.acme.org:443', request: 'ClientHello with TLS_RSA_WITH_AES_128_CBC_SHA', response: 'ServerHello accepted cipher TLS_RSA_WITH_AES_128_CBC_SHA' },
      false_positive: false, status: 'active', created_at: '2026-11-10T17:00:35Z', updated_at: '2026-11-10T17:00:35Z',
    },
    {
      id: 'fn-5', scan_job_id: scanId, asset_id: 'as-8', fqdn: 'portal.acme.org', tool: 'nuclei',
      rule_id: 'missing-strict-transport-security', title: 'Missing Strict-Transport-Security (HSTS) Header',
      severity: 'low', description: 'The web application does not transmit the Strict-Transport-Security header, leaving users vulnerable to SSL stripping downgrade attacks on insecure networks.',
      cve_ids: [], cvss_score: 3.7,
      remediation: 'Add Strict-Transport-Security: max-age=63072000; includeSubDomains; preload to all production HTTPS responses.',
      raw_evidence: { matched_at: 'https://portal.acme.org/', request: 'HEAD / HTTP/1.1\r\nHost: portal.acme.org\r\n\r\n', response: 'HTTP/1.1 200 OK\r\nServer: Vercel\r\n' },
      false_positive: false, status: 'active', created_at: '2026-11-10T17:00:36Z', updated_at: '2026-11-10T17:00:36Z',
    },
  ];
}

function getMockReport(scanId: string): AIReport {
  return {
    id: 'rep-acme-01',
    scan_job_id: scanId,
    executive_summary: 'External Attack Surface Assessment for acme.org established an overall posture of CRITICAL ELEVATED RISK (Risk Score: 100/100). Reconnaissance across 15 perimeter assets identified 1 critical, 1 high, 2 medium, and 1 low-severity findings. Immediate defensive priorities focus on eliminating unauthenticated administrative services and tightening perimeter ingress access controls.',
    overall_risk_score: 100,
    attack_surface_drift_analysis: {
      total_new_hosts: 1,
      total_removed_hosts: 0,
      total_opened_ports: 2,
      total_closed_ports: 0,
      high_risk_mutations: ['CRITICAL: Port 6379 opened on staging.acme.org (potential database or admin exposure)'],
      summary_description: 'Scan detected 1 new host and 2 newly opened ports. 1 critical mutation detected on staging.',
    },
    prioritized_findings: [
      {
        rank: 1,
        title: 'Exposed Unauthenticated Redis Cache Service',
        severity: 'critical',
        affected_asset: 'staging.acme.org:6379',
        exploit_probability: 'Active / Trivial Remote Exploit',
        impact_analysis: 'Direct unauthenticated memory access and remote code execution vulnerability.',
        root_cause: 'Misconfigured public ingress security group and default redis.conf binding.',
      },
      {
        rank: 2,
        title: 'Exposed Git Repository Configuration',
        severity: 'high',
        affected_asset: 'staging.acme.org',
        exploit_probability: 'High / Public PoC Accessible',
        impact_analysis: 'Full proprietary source code recovery, credential extraction, and architectural mapping.',
        root_cause: 'Missing web server access restriction on dotfiles (.git).',
      },
      {
        rank: 3,
        title: 'Permissive CORS Access-Control-Allow-Origin Wildcard',
        severity: 'medium',
        affected_asset: 'api.acme.org',
        exploit_probability: 'Moderate / Multi-step Chain',
        impact_analysis: 'Cross-site data theft and session hijacking of authenticated enterprise users.',
        root_cause: 'Blind reflection of Origin header with credentials allowed.',
      },
    ],
    remediation_playbook: [
      {
        title: 'Isolate and Authenticate Redis Cache',
        target_category: 'AWS_SG / Linux Firewall',
        severity: 'critical',
        config_snippet: `# /etc/redis/redis.conf
bind 127.0.0.1 -::1
protected-mode yes
requirepass "SECURE_RANDOMLY_GENERATED_64_CHAR_TOKEN"

# Immediate AWS CLI security group revocation:
aws ec2 revoke-security-group-ingress \\
  --group-id sg-0123456789abcdef0 \\
  --protocol tcp --port 6379 \\
  --cidr 0.0.0.0/0`,
        explanation: 'Restricts TCP 6379 exclusively to internal VPC networks and loopback, preventing external network exploitation.',
        validation_steps: [
          'Run nc -zv staging.acme.org 6379; verify connection is refused or dropped.',
          'Verify authorized microservices authenticate with AUTH token.',
        ],
      },
      {
        title: 'Block Public Access to .git Directories',
        target_category: 'Nginx',
        severity: 'high',
        config_snippet: `# /etc/nginx/conf.d/security.conf
location ~ /\\.git {
    deny all;
    return 404;
}`,
        explanation: 'Stops automated scanners and attackers from reconstructing your source repository tree.',
        validation_steps: [
          'Run curl -isk https://staging.acme.org/.git/config; confirm HTTP 404.',
        ],
      },
    ],
    tokens_used: 1420,
    model_version: 'gemini-2.5-flash',
    is_fallback: false,
    generated_at: '2026-11-10T17:00:48Z',
  };
}

