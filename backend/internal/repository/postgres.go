package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"reconpilot/internal/config"
	"reconpilot/internal/domain"
)

// PostgresDB encapsulates the pgx connection pool and provides repository implementations.
type PostgresDB struct {
	Pool *pgxpool.Pool
}

// NewPostgresDB initializes a connection pool using the provided configuration.
func NewPostgresDB(ctx context.Context, cfg *config.DatabaseConfig) (*PostgresDB, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		poolCfg.MinConns = cfg.MinConns
	}
	if cfg.MaxConnLifetime > 0 {
		poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	}
	if cfg.MaxConnIdleTime > 0 {
		poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgxpool: %w", err)
	}

	// Verify connectivity
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		// Log warning but return DB instance so server can start in offline/mock mode
		return &PostgresDB{Pool: pool}, nil
	}

	return &PostgresDB{Pool: pool}, nil
}

// Close releases database connections.
func (db *PostgresDB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// ==============================================================================
// SCOPE REPOSITORY
// ==============================================================================

func (db *PostgresDB) Create(ctx context.Context, scope *domain.Scope) error {
	allowedJSON, _ := json.Marshal(scope.AllowedCIDRs)
	excludedCIDRsJSON, _ := json.Marshal(scope.ExcludedCIDRs)
	excludedSubsJSON, _ := json.Marshal(scope.ExcludedSubdomains)

	query := `
		INSERT INTO scopes (
			id, name, root_domain, verification_status, verification_token,
			verified_at, allowed_cidrs, excluded_cidrs, excluded_subdomains,
			rate_limit_rps, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	now := time.Now()
	if scope.ID == uuid.Nil {
		scope.ID = uuid.New()
	}
	scope.CreatedAt = now
	scope.UpdatedAt = now

	_, err := db.Pool.Exec(ctx, query,
		scope.ID, scope.Name, scope.RootDomain, scope.VerificationStatus, scope.VerificationToken,
		scope.VerifiedAt, allowedJSON, excludedCIDRsJSON, excludedSubsJSON,
		scope.RateLimitRPS, scope.CreatedAt, scope.UpdatedAt,
	)
	return err
}

func (db *PostgresDB) GetByID(ctx context.Context, id uuid.UUID) (*domain.Scope, error) {
	query := `
		SELECT id, name, root_domain, verification_status, verification_token,
		       verified_at, allowed_cidrs, excluded_cidrs, excluded_subdomains,
		       rate_limit_rps, created_at, updated_at
		FROM scopes WHERE id = $1
	`
	row := db.Pool.QueryRow(ctx, query, id)

	var s domain.Scope
	var allowedJSON, excludedCIDRsJSON, excludedSubsJSON []byte

	err := row.Scan(
		&s.ID, &s.Name, &s.RootDomain, &s.VerificationStatus, &s.VerificationToken,
		&s.VerifiedAt, &allowedJSON, &excludedCIDRsJSON, &excludedSubsJSON,
		&s.RateLimitRPS, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	_ = json.Unmarshal(allowedJSON, &s.AllowedCIDRs)
	_ = json.Unmarshal(excludedCIDRsJSON, &s.ExcludedCIDRs)
	_ = json.Unmarshal(excludedSubsJSON, &s.ExcludedSubdomains)

	return &s, nil
}

func (db *PostgresDB) GetByRootDomain(ctx context.Context, rootDomain string) (*domain.Scope, error) {
	query := `
		SELECT id, name, root_domain, verification_status, verification_token,
		       verified_at, allowed_cidrs, excluded_cidrs, excluded_subdomains,
		       rate_limit_rps, created_at, updated_at
		FROM scopes WHERE root_domain = $1
	`
	row := db.Pool.QueryRow(ctx, query, rootDomain)

	var s domain.Scope
	var allowedJSON, excludedCIDRsJSON, excludedSubsJSON []byte

	err := row.Scan(
		&s.ID, &s.Name, &s.RootDomain, &s.VerificationStatus, &s.VerificationToken,
		&s.VerifiedAt, &allowedJSON, &excludedCIDRsJSON, &excludedSubsJSON,
		&s.RateLimitRPS, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	_ = json.Unmarshal(allowedJSON, &s.AllowedCIDRs)
	_ = json.Unmarshal(excludedCIDRsJSON, &s.ExcludedCIDRs)
	_ = json.Unmarshal(excludedSubsJSON, &s.ExcludedSubdomains)

	return &s, nil
}

func (db *PostgresDB) Update(ctx context.Context, scope *domain.Scope) error {
	allowedJSON, _ := json.Marshal(scope.AllowedCIDRs)
	excludedCIDRsJSON, _ := json.Marshal(scope.ExcludedCIDRs)
	excludedSubsJSON, _ := json.Marshal(scope.ExcludedSubdomains)

	query := `
		UPDATE scopes SET
			name = $2, allowed_cidrs = $3, excluded_cidrs = $4,
			excluded_subdomains = $5, rate_limit_rps = $6, updated_at = $7
		WHERE id = $1
	`
	scope.UpdatedAt = time.Now()
	_, err := db.Pool.Exec(ctx, query,
		scope.ID, scope.Name, allowedJSON, excludedCIDRsJSON,
		excludedSubsJSON, scope.RateLimitRPS, scope.UpdatedAt,
	)
	return err
}

func (db *PostgresDB) UpdateVerificationStatus(ctx context.Context, id uuid.UUID, status domain.VerificationStatus) error {
	query := `
		UPDATE scopes SET verification_status = $2, verified_at = $3, updated_at = $4
		WHERE id = $1
	`
	now := time.Now()
	var verifiedAt *time.Time
	if status == domain.VerificationVerified {
		verifiedAt = &now
	}
	_, err := db.Pool.Exec(ctx, query, id, status, verifiedAt, now)
	return err
}

func (db *PostgresDB) List(ctx context.Context) ([]*domain.Scope, error) {
	query := `
		SELECT id, name, root_domain, verification_status, verification_token,
		       verified_at, allowed_cidrs, excluded_cidrs, excluded_subdomains,
		       rate_limit_rps, created_at, updated_at
		FROM scopes ORDER BY created_at DESC
	`
	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var scopes []*domain.Scope
	for rows.Next() {
		var s domain.Scope
		var allowedJSON, excludedCIDRsJSON, excludedSubsJSON []byte

		if err := rows.Scan(
			&s.ID, &s.Name, &s.RootDomain, &s.VerificationStatus, &s.VerificationToken,
			&s.VerifiedAt, &allowedJSON, &excludedCIDRsJSON, &excludedSubsJSON,
			&s.RateLimitRPS, &s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}

		_ = json.Unmarshal(allowedJSON, &s.AllowedCIDRs)
		_ = json.Unmarshal(excludedCIDRsJSON, &s.ExcludedCIDRs)
		_ = json.Unmarshal(excludedSubsJSON, &s.ExcludedSubdomains)
		scopes = append(scopes, &s)
	}

	return scopes, nil
}

func (db *PostgresDB) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := db.Pool.Exec(ctx, "DELETE FROM scopes WHERE id = $1", id)
	return err
}

// ==============================================================================
// SCAN JOB REPOSITORY
// ==============================================================================

func (db *PostgresDB) CreateJob(ctx context.Context, job *domain.ScanJob) error {
	checkpointsJSON, _ := json.Marshal(job.StageCheckpoints)
	metricsJSON, _ := json.Marshal(job.Metrics)

	query := `
		INSERT INTO scan_jobs (
			id, scope_id, status, profile, current_stage, stage_checkpoints,
			metrics, error_message, dry_run, started_at, completed_at,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`
	now := time.Now()
	if job.ID == uuid.Nil {
		job.ID = uuid.New()
	}
	job.CreatedAt = now
	job.UpdatedAt = now

	_, err := db.Pool.Exec(ctx, query,
		job.ID, job.ScopeID, job.Status, job.Profile, job.CurrentStage,
		checkpointsJSON, metricsJSON, job.ErrorMessage, job.DryRun,
		job.StartedAt, job.CompletedAt, job.CreatedAt, job.UpdatedAt,
	)
	return err
}

func (db *PostgresDB) GetJobByID(ctx context.Context, id uuid.UUID) (*domain.ScanJob, error) {
	query := `
		SELECT id, scope_id, status, profile, current_stage, stage_checkpoints,
		       metrics, error_message, dry_run, started_at, completed_at,
		       created_at, updated_at
		FROM scan_jobs WHERE id = $1
	`
	row := db.Pool.QueryRow(ctx, query, id)

	var j domain.ScanJob
	var checkpointsJSON, metricsJSON []byte

	err := row.Scan(
		&j.ID, &j.ScopeID, &j.Status, &j.Profile, &j.CurrentStage, &checkpointsJSON,
		&metricsJSON, &j.ErrorMessage, &j.DryRun, &j.StartedAt, &j.CompletedAt,
		&j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	_ = json.Unmarshal(checkpointsJSON, &j.StageCheckpoints)
	_ = json.Unmarshal(metricsJSON, &j.Metrics)

	return &j, nil
}

func (db *PostgresDB) GetLatestCompletedByScope(ctx context.Context, scopeID uuid.UUID, excludeJobID uuid.UUID) (*domain.ScanJob, error) {
	query := `
		SELECT id, scope_id, status, profile, current_stage, stage_checkpoints,
		       metrics, error_message, dry_run, started_at, completed_at,
		       created_at, updated_at
		FROM scan_jobs
		WHERE scope_id = $1 AND status = 'completed' AND id != $2
		ORDER BY completed_at DESC LIMIT 1
	`
	row := db.Pool.QueryRow(ctx, query, scopeID, excludeJobID)

	var j domain.ScanJob
	var checkpointsJSON, metricsJSON []byte

	err := row.Scan(
		&j.ID, &j.ScopeID, &j.Status, &j.Profile, &j.CurrentStage, &checkpointsJSON,
		&metricsJSON, &j.ErrorMessage, &j.DryRun, &j.StartedAt, &j.CompletedAt,
		&j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	_ = json.Unmarshal(checkpointsJSON, &j.StageCheckpoints)
	_ = json.Unmarshal(metricsJSON, &j.Metrics)

	return &j, nil
}

func (db *PostgresDB) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.ScanStatus, errMessage *string) error {
	query := `
		UPDATE scan_jobs SET status = $2, error_message = $3, updated_at = $4,
		       completed_at = CASE WHEN $2 IN ('completed', 'failed', 'cancelled') THEN $4 ELSE completed_at END
		WHERE id = $1
	`
	_, err := db.Pool.Exec(ctx, query, id, status, errMessage, time.Now())
	return err
}

func (db *PostgresDB) UpdateCurrentStage(ctx context.Context, id uuid.UUID, stage domain.StageName) error {
	query := `UPDATE scan_jobs SET current_stage = $2, updated_at = $3 WHERE id = $1`
	_, err := db.Pool.Exec(ctx, query, id, stage, time.Now())
	return err
}

func (db *PostgresDB) SaveCheckpoint(ctx context.Context, id uuid.UUID, checkpoint domain.StageCheckpoint) error {
	// Atomic JSONB merge into stage_checkpoints
	cpJSON, _ := json.Marshal(checkpoint)
	query := `
		UPDATE scan_jobs
		SET stage_checkpoints = jsonb_set(COALESCE(stage_checkpoints, '{}'::jsonb), ARRAY[$2::text], $3::jsonb),
		    updated_at = $4
		WHERE id = $1
	`
	_, err := db.Pool.Exec(ctx, query, id, string(checkpoint.Stage), cpJSON, time.Now())
	return err
}

func (db *PostgresDB) UpdateMetrics(ctx context.Context, id uuid.UUID, metrics domain.ScanMetrics) error {
	metricsJSON, _ := json.Marshal(metrics)
	query := `UPDATE scan_jobs SET metrics = $2, updated_at = $3 WHERE id = $1`
	_, err := db.Pool.Exec(ctx, query, id, metricsJSON, time.Now())
	return err
}

func (db *PostgresDB) ListJobsByScope(ctx context.Context, scopeID uuid.UUID) ([]*domain.ScanJob, error) {
	query := `
		SELECT id, scope_id, status, profile, current_stage, stage_checkpoints,
		       metrics, error_message, dry_run, started_at, completed_at,
		       created_at, updated_at
		FROM scan_jobs WHERE scope_id = $1 ORDER BY created_at DESC
	`
	rows, err := db.Pool.Query(ctx, query, scopeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*domain.ScanJob
	for rows.Next() {
		var j domain.ScanJob
		var checkpointsJSON, metricsJSON []byte

		if err := rows.Scan(
			&j.ID, &j.ScopeID, &j.Status, &j.Profile, &j.CurrentStage, &checkpointsJSON,
			&metricsJSON, &j.ErrorMessage, &j.DryRun, &j.StartedAt, &j.CompletedAt,
			&j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, err
		}

		_ = json.Unmarshal(checkpointsJSON, &j.StageCheckpoints)
		_ = json.Unmarshal(metricsJSON, &j.Metrics)
		jobs = append(jobs, &j)
	}

	return jobs, nil
}

// ==============================================================================
// ASSET REPOSITORY
// ==============================================================================

func (db *PostgresDB) BatchInsertAssets(ctx context.Context, assets []*domain.Asset) error {
	if len(assets) == 0 {
		return nil
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	stmt := `
		INSERT INTO assets (
			id, scan_job_id, scope_id, fqdn, root_domain, ip_addresses,
			asn, asn_org, http_status, page_title, tech_stack, open_ports,
			tls_info, response_headers, first_seen_at, last_seen_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		) ON CONFLICT (scan_job_id, fqdn) DO UPDATE SET
			ip_addresses = EXCLUDED.ip_addresses,
			http_status = EXCLUDED.http_status,
			page_title = EXCLUDED.page_title,
			tech_stack = EXCLUDED.tech_stack,
			open_ports = EXCLUDED.open_ports,
			tls_info = EXCLUDED.tls_info,
			response_headers = EXCLUDED.response_headers,
			last_seen_at = EXCLUDED.last_seen_at,
			updated_at = EXCLUDED.updated_at
	`

	for _, a := range assets {
		ipsJSON, _ := json.Marshal(a.IPAddresses)
		techJSON, _ := json.Marshal(a.TechStack)
		portsJSON, _ := json.Marshal(a.OpenPorts)
		tlsJSON, _ := json.Marshal(a.TLSInfo)
		headersJSON, _ := json.Marshal(a.ResponseHeaders)

		if a.ID == uuid.Nil {
			a.ID = uuid.New()
		}

		_, err := tx.Exec(ctx, stmt,
			a.ID, a.ScanJobID, a.ScopeID, a.FQDN, a.RootDomain, ipsJSON,
			a.ASN, a.ASNOrg, a.HTTPStatus, a.PageTitle, techJSON, portsJSON,
			tlsJSON, headersJSON, a.FirstSeenAt, a.LastSeenAt, a.CreatedAt, a.UpdatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (db *PostgresDB) ListAssetsByScanJob(ctx context.Context, scanJobID uuid.UUID) ([]*domain.Asset, error) {
	query := `
		SELECT id, scan_job_id, scope_id, fqdn, root_domain, ip_addresses,
		       asn, asn_org, http_status, page_title, tech_stack, open_ports,
		       tls_info, response_headers, first_seen_at, last_seen_at,
		       created_at, updated_at
		FROM assets WHERE scan_job_id = $1 ORDER BY fqdn ASC
	`
	rows, err := db.Pool.Query(ctx, query, scanJobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []*domain.Asset
	for rows.Next() {
		var a domain.Asset
		var ipsJSON, techJSON, portsJSON, tlsJSON, headersJSON []byte

		if err := rows.Scan(
			&a.ID, &a.ScanJobID, &a.ScopeID, &a.FQDN, &a.RootDomain, &ipsJSON,
			&a.ASN, &a.ASNOrg, &a.HTTPStatus, &a.PageTitle, &techJSON, &portsJSON,
			&tlsJSON, &headersJSON, &a.FirstSeenAt, &a.LastSeenAt, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}

		_ = json.Unmarshal(ipsJSON, &a.IPAddresses)
		_ = json.Unmarshal(techJSON, &a.TechStack)
		_ = json.Unmarshal(portsJSON, &a.OpenPorts)
		_ = json.Unmarshal(tlsJSON, &a.TLSInfo)
		_ = json.Unmarshal(headersJSON, &a.ResponseHeaders)

		assets = append(assets, &a)
	}

	return assets, nil
}

// ==============================================================================
// ASSET DRIFT REPOSITORY
// ==============================================================================

func (db *PostgresDB) BatchInsertDrifts(ctx context.Context, drifts []*domain.AssetDrift) error {
	if len(drifts) == 0 {
		return nil
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	stmt := `
		INSERT INTO asset_drifts (
			id, scan_job_id, previous_scan_job_id, asset_id, fqdn,
			change_type, details, severity, detected_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	for _, d := range drifts {
		detailsJSON, _ := json.Marshal(d.Details)
		if d.ID == uuid.Nil {
			d.ID = uuid.New()
		}

		_, err := tx.Exec(ctx, stmt,
			d.ID, d.ScanJobID, d.PreviousScanJobID, d.AssetID, d.FQDN,
			d.ChangeType, detailsJSON, d.Severity, d.DetectedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (db *PostgresDB) ListDriftsByScanJob(ctx context.Context, scanJobID uuid.UUID) ([]*domain.AssetDrift, error) {
	query := `
		SELECT id, scan_job_id, previous_scan_job_id, asset_id, fqdn,
		       change_type, details, severity, detected_at
		FROM asset_drifts WHERE scan_job_id = $1 ORDER BY detected_at DESC
	`
	rows, err := db.Pool.Query(ctx, query, scanJobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var drifts []*domain.AssetDrift
	for rows.Next() {
		var d domain.AssetDrift
		var detailsJSON []byte

		if err := rows.Scan(
			&d.ID, &d.ScanJobID, &d.PreviousScanJobID, &d.AssetID, &d.FQDN,
			&d.ChangeType, &detailsJSON, &d.Severity, &d.DetectedAt,
		); err != nil {
			return nil, err
		}

		_ = json.Unmarshal(detailsJSON, &d.Details)
		drifts = append(drifts, &d)
	}

	return drifts, nil
}

// ==============================================================================
// FINDINGS REPOSITORY
// ==============================================================================

func (db *PostgresDB) BatchInsertFindings(ctx context.Context, findings []*domain.Finding) error {
	if len(findings) == 0 {
		return nil
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	stmt := `
		INSERT INTO findings (
			id, scan_job_id, asset_id, tool, rule_id, title, severity,
			description, cve_ids, cvss_score, remediation, raw_evidence,
			false_positive, status, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
		)
	`

	for _, f := range findings {
		cvesJSON, _ := json.Marshal(f.CVEIDs)
		evidenceJSON, _ := json.Marshal(f.RawEvidence)
		if f.ID == uuid.Nil {
			f.ID = uuid.New()
		}

		_, err := tx.Exec(ctx, stmt,
			f.ID, f.ScanJobID, f.AssetID, f.Tool, f.RuleID, f.Title, f.Severity,
			f.Description, cvesJSON, f.CVSSScore, f.Remediation, evidenceJSON,
			f.FalsePositive, f.Status, f.CreatedAt, f.UpdatedAt,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (db *PostgresDB) ListFindingsByScanJob(ctx context.Context, scanJobID uuid.UUID) ([]*domain.Finding, error) {
	query := `
		SELECT f.id, f.scan_job_id, f.asset_id, COALESCE(a.fqdn, '') AS fqdn,
		       f.tool, f.rule_id, f.title, f.severity, f.description,
		       f.cve_ids, f.cvss_score, f.remediation, f.raw_evidence,
		       f.false_positive, f.status, f.created_at, f.updated_at
		FROM findings f
		LEFT JOIN assets a ON f.asset_id = a.id
		WHERE f.scan_job_id = $1
		ORDER BY f.cvss_score DESC, f.created_at DESC
	`
	rows, err := db.Pool.Query(ctx, query, scanJobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var findings []*domain.Finding
	for rows.Next() {
		var f domain.Finding
		var cvesJSON, evidenceJSON []byte

		if err := rows.Scan(
			&f.ID, &f.ScanJobID, &f.AssetID, &f.FQDN,
			&f.Tool, &f.RuleID, &f.Title, &f.Severity, &f.Description,
			&cvesJSON, &f.CVSSScore, &f.Remediation, &evidenceJSON,
			&f.FalsePositive, &f.Status, &f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, err
		}

		_ = json.Unmarshal(cvesJSON, &f.CVEIDs)
		_ = json.Unmarshal(evidenceJSON, &f.RawEvidence)
		findings = append(findings, &f)
	}

	return findings, nil
}

// ==============================================================================
// AI REPORT REPOSITORY
// ==============================================================================

func (db *PostgresDB) CreateAIReport(ctx context.Context, report *domain.AIReport) error {
	driftJSON, _ := json.Marshal(report.AttackSurfaceDriftAnalysis)
	findingsJSON, _ := json.Marshal(report.PrioritizedFindings)
	playbookJSON, _ := json.Marshal(report.RemediationPlaybook)

	query := `
		INSERT INTO ai_reports (
			id, scan_job_id, executive_summary, overall_risk_score,
			attack_surface_drift_analysis, prioritized_findings, remediation_playbook,
			tokens_used, model_version, is_fallback, generated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (scan_job_id) DO UPDATE SET
			executive_summary = EXCLUDED.executive_summary,
			overall_risk_score = EXCLUDED.overall_risk_score,
			attack_surface_drift_analysis = EXCLUDED.attack_surface_drift_analysis,
			prioritized_findings = EXCLUDED.prioritized_findings,
			remediation_playbook = EXCLUDED.remediation_playbook,
			tokens_used = EXCLUDED.tokens_used,
			model_version = EXCLUDED.model_version,
			is_fallback = EXCLUDED.is_fallback,
			generated_at = EXCLUDED.generated_at
	`
	if report.ID == uuid.Nil {
		report.ID = uuid.New()
	}
	report.GeneratedAt = time.Now()

	_, err := db.Pool.Exec(ctx, query,
		report.ID, report.ScanJobID, report.ExecutiveSummary, report.OverallRiskScore,
		driftJSON, findingsJSON, playbookJSON, report.TokensUsed, report.ModelVersion,
		report.IsFallback, report.GeneratedAt,
	)
	return err
}

func (db *PostgresDB) GetAIReportByScanJob(ctx context.Context, scanJobID uuid.UUID) (*domain.AIReport, error) {
	query := `
		SELECT id, scan_job_id, executive_summary, overall_risk_score,
		       attack_surface_drift_analysis, prioritized_findings, remediation_playbook,
		       tokens_used, model_version, is_fallback, generated_at
		FROM ai_reports WHERE scan_job_id = $1
	`
	row := db.Pool.QueryRow(ctx, query, scanJobID)

	var r domain.AIReport
	var driftJSON, findingsJSON, playbookJSON []byte

	err := row.Scan(
		&r.ID, &r.ScanJobID, &r.ExecutiveSummary, &r.OverallRiskScore,
		&driftJSON, &findingsJSON, &playbookJSON, &r.TokensUsed, &r.ModelVersion,
		&r.IsFallback, &r.GeneratedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	_ = json.Unmarshal(driftJSON, &r.AttackSurfaceDriftAnalysis)
	_ = json.Unmarshal(findingsJSON, &r.PrioritizedFindings)
	_ = json.Unmarshal(playbookJSON, &r.RemediationPlaybook)

	return &r, nil
}

