export type VerificationStatus = 'pending_verification' | 'verified' | 'revoked';
export type ScanStatus = 'queued' | 'running' | 'completed' | 'failed' | 'cancelled';
export type ScanProfile = 'passive' | 'standard' | 'deep';
export type StageName = 'idle' | 'discovery' | 'probing' | 'portscan' | 'auditor' | 'ai_synthesis' | 'completed';
export type StageStatus = 'pending' | 'running' | 'done' | 'failed' | 'skipped';
export type FindingSeverity = 'info' | 'low' | 'medium' | 'high' | 'critical';
export type FindingStatus = 'active' | 'remediated' | 'acknowledged';
export type DriftChangeType =
  | 'new_host'
  | 'removed_host'
  | 'port_opened'
  | 'port_closed'
  | 'tech_changed'
  | 'ip_changed'
  | 'finding_added'
  | 'finding_resolved';

export interface Scope {
  id: string;
  name: string;
  root_domain: string;
  verification_status: VerificationStatus;
  verification_token: string;
  verified_at?: string;
  allowed_cidrs: string[];
  excluded_cidrs: string[];
  excluded_subdomains: string[];
  rate_limit_rps: number;
  created_at: string;
  updated_at: string;
}

export interface StageCheckpoint {
  stage: StageName;
  status: StageStatus;
  started_at: string;
  completed_at?: string;
  duration_ms: number;
  output_count: number;
  error?: string;
}

export interface ScanMetrics {
  total_hosts: number;
  live_hosts: number;
  open_ports: number;
  findings_count: number;
  duration_seconds: number;
  requests_per_sec: number;
}

export interface ScanJob {
  id: string;
  scope_id: string;
  status: ScanStatus;
  profile: ScanProfile;
  current_stage: StageName;
  stage_checkpoints: Record<StageName, StageCheckpoint>;
  metrics: ScanMetrics;
  error_message?: string;
  dry_run: boolean;
  started_at?: string;
  completed_at?: string;
  created_at: string;
  updated_at: string;
}

export interface TLSDetails {
  subject?: string;
  issuer?: string;
  valid_from?: string;
  valid_until?: string;
  sans?: string[];
  cipher_suite?: string;
  version?: string;
}

export interface Asset {
  id: string;
  scan_job_id: string;
  scope_id: string;
  fqdn: string;
  root_domain: string;
  ip_addresses: string[];
  asn?: string;
  asn_org?: string;
  http_status: number;
  page_title: string;
  tech_stack: string[];
  open_ports: number[];
  tls_info?: TLSDetails;
  response_headers?: Record<string, string>;
  first_seen_at: string;
  last_seen_at: string;
  created_at: string;
  updated_at: string;
}

export interface AssetDrift {
  id: string;
  scan_job_id: string;
  previous_scan_job_id?: string;
  asset_id?: string;
  fqdn: string;
  change_type: DriftChangeType;
  details: Record<string, any>;
  severity: FindingSeverity;
  detected_at: string;
}

export interface RawEvidence {
  matched_at?: string;
  request?: string;
  response?: string;
  matcher_name?: string;
  extracted_results?: string[];
  curl_command?: string;
}

export interface Finding {
  id: string;
  scan_job_id: string;
  asset_id: string;
  fqdn: string;
  tool: string;
  rule_id: string;
  title: string;
  severity: FindingSeverity;
  description: string;
  cve_ids: string[];
  cvss_score: number;
  remediation?: string;
  raw_evidence: RawEvidence;
  false_positive: boolean;
  status: FindingStatus;
  created_at: string;
  updated_at: string;
}

export interface DriftAnalysis {
  total_new_hosts: number;
  total_removed_hosts: number;
  total_opened_ports: number;
  total_closed_ports: number;
  high_risk_mutations: string[];
  summary_description: string;
}

export interface PrioritizedFinding {
  rank: number;
  title: string;
  severity: FindingSeverity;
  affected_asset: string;
  exploit_probability: string;
  impact_analysis: string;
  root_cause: string;
}

export interface RemediationItem {
  title: string;
  target_category: string;
  severity: string;
  config_snippet: string;
  explanation: string;
  validation_steps: string[];
}

export interface AIReport {
  id: string;
  scan_job_id: string;
  executive_summary: string;
  overall_risk_score: number;
  attack_surface_drift_analysis: DriftAnalysis;
  prioritized_findings: PrioritizedFinding[];
  remediation_playbook: RemediationItem[];
  tokens_used: number;
  model_version: string;
  is_fallback: boolean;
  generated_at: string;
}

export interface TelemetryEvent {
  type: string;
  scan_job_id: string;
  stage?: StageName;
  timestamp: string;
  payload: any;
}

