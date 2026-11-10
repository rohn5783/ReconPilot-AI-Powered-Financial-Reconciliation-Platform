-- 001_init.sql: ReconPilot Core Relational Schema
-- High-performance schema for EASM assets, drift tracking, findings, and AI reports

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ==============================================================================
-- 1. SCOPES (Target Authorization & Verification Guardrails)
-- ==============================================================================
CREATE TABLE IF NOT EXISTS scopes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    root_domain VARCHAR(255) NOT NULL UNIQUE,
    verification_status VARCHAR(32) NOT NULL DEFAULT 'pending_verification'
        CHECK (verification_status IN ('pending_verification', 'verified', 'revoked')),
    verification_token VARCHAR(128) NOT NULL,
    verified_at TIMESTAMPTZ,
    allowed_cidrs JSONB NOT NULL DEFAULT '[]'::jsonb,
    excluded_cidrs JSONB NOT NULL DEFAULT '[]'::jsonb,
    excluded_subdomains JSONB NOT NULL DEFAULT '[]'::jsonb,
    rate_limit_rps INT NOT NULL DEFAULT 20,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_scopes_root_domain ON scopes (root_domain);
CREATE INDEX IF NOT EXISTS idx_scopes_verification ON scopes (verification_status);

-- ==============================================================================
-- 2. SCAN JOBS (DAG Pipeline Execution & Checkpoints)
-- ==============================================================================
CREATE TABLE IF NOT EXISTS scan_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scope_id UUID NOT NULL REFERENCES scopes(id) ON DELETE CASCADE,
    status VARCHAR(32) NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'running', 'completed', 'failed', 'cancelled')),
    profile VARCHAR(32) NOT NULL DEFAULT 'standard'
        CHECK (profile IN ('passive', 'standard', 'deep')),
    current_stage VARCHAR(64) NOT NULL DEFAULT 'idle',
    stage_checkpoints JSONB NOT NULL DEFAULT '{}'::jsonb,
    metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT,
    dry_run BOOLEAN NOT NULL DEFAULT FALSE,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_scan_jobs_scope_status ON scan_jobs (scope_id, status);
CREATE INDEX IF NOT EXISTS idx_scan_jobs_created_at ON scan_jobs (created_at DESC);

-- ==============================================================================
-- 3. ASSETS (Discovered External Attack Surface Inventory)
-- ==============================================================================
CREATE TABLE IF NOT EXISTS assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_job_id UUID NOT NULL REFERENCES scan_jobs(id) ON DELETE CASCADE,
    scope_id UUID NOT NULL REFERENCES scopes(id) ON DELETE CASCADE,
    fqdn VARCHAR(255) NOT NULL,
    root_domain VARCHAR(255) NOT NULL,
    ip_addresses JSONB NOT NULL DEFAULT '[]'::jsonb,
    asn VARCHAR(64),
    asn_org VARCHAR(255),
    http_status INT,
    page_title TEXT,
    tech_stack JSONB NOT NULL DEFAULT '[]'::jsonb,
    open_ports JSONB NOT NULL DEFAULT '[]'::jsonb,
    tls_info JSONB NOT NULL DEFAULT '{}'::jsonb,
    response_headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_scan_asset UNIQUE (scan_job_id, fqdn)
);

CREATE INDEX IF NOT EXISTS idx_assets_scan_job_id ON assets (scan_job_id);
CREATE INDEX IF NOT EXISTS idx_assets_scope_id ON assets (scope_id);
CREATE INDEX IF NOT EXISTS idx_assets_fqdn ON assets (fqdn);
CREATE INDEX IF NOT EXISTS idx_assets_http_status ON assets (http_status);
CREATE INDEX IF NOT EXISTS idx_assets_tech_stack ON assets USING gin (tech_stack);
CREATE INDEX IF NOT EXISTS idx_assets_open_ports ON assets USING gin (open_ports);

-- ==============================================================================
-- 4. ASSET DRIFTS (Differential State Tracking Between Scans)
-- ==============================================================================
CREATE TABLE IF NOT EXISTS asset_drifts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_job_id UUID NOT NULL REFERENCES scan_jobs(id) ON DELETE CASCADE,
    previous_scan_job_id UUID REFERENCES scan_jobs(id) ON DELETE SET NULL,
    asset_id UUID REFERENCES assets(id) ON DELETE CASCADE,
    fqdn VARCHAR(255) NOT NULL,
    change_type VARCHAR(64) NOT NULL
        CHECK (change_type IN (
            'new_host', 'removed_host', 'port_opened', 'port_closed', 
            'tech_changed', 'ip_changed', 'finding_added', 'finding_resolved'
        )),
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    severity VARCHAR(32) NOT NULL DEFAULT 'medium'
        CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    detected_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_asset_drifts_scan_job ON asset_drifts (scan_job_id);
CREATE INDEX IF NOT EXISTS idx_asset_drifts_change_type ON asset_drifts (change_type);
CREATE INDEX IF NOT EXISTS idx_asset_drifts_fqdn ON asset_drifts (fqdn);

-- ==============================================================================
-- 5. FINDINGS (Security Audits & Identified Vulnerabilities)
-- ==============================================================================
CREATE TABLE IF NOT EXISTS findings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_job_id UUID NOT NULL REFERENCES scan_jobs(id) ON DELETE CASCADE,
    asset_id UUID REFERENCES assets(id) ON DELETE CASCADE,
    tool VARCHAR(64) NOT NULL,
    rule_id VARCHAR(128) NOT NULL,
    title VARCHAR(255) NOT NULL,
    severity VARCHAR(32) NOT NULL
        CHECK (severity IN ('info', 'low', 'medium', 'high', 'critical')),
    description TEXT NOT NULL,
    cve_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    cvss_score NUMERIC(3, 1) DEFAULT 0.0,
    remediation TEXT,
    raw_evidence JSONB NOT NULL DEFAULT '{}'::jsonb,
    false_positive BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(32) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'remediated', 'acknowledged')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_findings_scan_job_id ON findings (scan_job_id);
CREATE INDEX IF NOT EXISTS idx_findings_asset_id ON findings (asset_id);
CREATE INDEX IF NOT EXISTS idx_findings_severity ON findings (severity);
CREATE INDEX IF NOT EXISTS idx_findings_status ON findings (status);
CREATE INDEX IF NOT EXISTS idx_findings_rule_id ON findings (rule_id);

-- ==============================================================================
-- 6. AI REPORTS (Deterministic Google GenAI Structured Synthesis)
-- ==============================================================================
CREATE TABLE IF NOT EXISTS ai_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scan_job_id UUID NOT NULL REFERENCES scan_jobs(id) ON DELETE CASCADE UNIQUE,
    executive_summary TEXT NOT NULL,
    overall_risk_score INT NOT NULL CHECK (overall_risk_score >= 0 AND overall_risk_score <= 100),
    attack_surface_drift_analysis JSONB NOT NULL DEFAULT '{}'::jsonb,
    prioritized_findings JSONB NOT NULL DEFAULT '[]'::jsonb,
    remediation_playbook JSONB NOT NULL DEFAULT '[]'::jsonb,
    tokens_used INT NOT NULL DEFAULT 0,
    model_version VARCHAR(64) NOT NULL,
    is_fallback BOOLEAN NOT NULL DEFAULT FALSE,
    generated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ai_reports_scan_job ON ai_reports (scan_job_id);

