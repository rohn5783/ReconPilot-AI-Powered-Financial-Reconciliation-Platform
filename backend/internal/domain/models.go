package domain

import (
	"time"

	"github.com/google/uuid"
)

// VerificationStatus represents target ownership verification states.
type VerificationStatus string

const (
	VerificationPending  VerificationStatus = "pending_verification"
	VerificationVerified VerificationStatus = "verified"
	VerificationRevoked  VerificationStatus = "revoked"
)

// ScanStatus represents the lifecycle of a scan pipeline.
type ScanStatus string

const (
	ScanQueued    ScanStatus = "queued"
	ScanRunning   ScanStatus = "running"
	ScanCompleted ScanStatus = "completed"
	ScanFailed    ScanStatus = "failed"
	ScanCancelled ScanStatus = "cancelled"
)

// ScanProfile determines pipeline intensity and tool execution depth.
type ScanProfile string

const (
	ProfilePassive  ScanProfile = "passive"
	ProfileStandard ScanProfile = "standard"
	ProfileDeep     ScanProfile = "deep"
)

// StageName defines the sequential stages of the DAG orchestrator.
type StageName string

const (
	StageIdle        StageName = "idle"
	StageDiscovery   StageName = "discovery"
	StageProbing     StageName = "probing"
	StagePortScan    StageName = "portscan"
	StageAuditor     StageName = "auditor"
	StageAISynthesis StageName = "ai_synthesis"
	StageCompleted   StageName = "completed"
)

// StageStatus represents the state of a single pipeline stage.
type StageStatus string

const (
	StageStatusPending StageStatus = "pending"
	StageStatusRunning StageStatus = "running"
	StageStatusDone    StageStatus = "done"
	StageStatusFailed  StageStatus = "failed"
	StageStatusSkipped StageStatus = "skipped"
)

// FindingSeverity categorizes risk levels.
type FindingSeverity string

const (
	SeverityInfo     FindingSeverity = "info"
	SeverityLow      FindingSeverity = "low"
	SeverityMedium   FindingSeverity = "medium"
	SeverityHigh     FindingSeverity = "high"
	SeverityCritical FindingSeverity = "critical"
)

// FindingStatus represents finding triage state.
type FindingStatus string

const (
	FindingActive       FindingStatus = "active"
	FindingRemediated   FindingStatus = "remediated"
	FindingAcknowledged FindingStatus = "acknowledged"
)

// DriftChangeType represents delta mutations in attack surface state.
type DriftChangeType string

const (
	DriftNewHost         DriftChangeType = "new_host"
	DriftRemovedHost     DriftChangeType = "removed_host"
	DriftPortOpened      DriftChangeType = "port_opened"
	DriftPortClosed      DriftChangeType = "port_closed"
	DriftTechChanged     DriftChangeType = "tech_changed"
	DriftIPChanged       DriftChangeType = "ip_changed"
	DriftFindingAdded    DriftChangeType = "finding_added"
	DriftFindingResolved DriftChangeType = "finding_resolved"
)

// Scope represents an authorized root target boundary.
type Scope struct {
	ID                 uuid.UUID          `json:"id" db:"id" bson:"_id"`
	Name               string             `json:"name" db:"name" bson:"name"`
	RootDomain         string             `json:"root_domain" db:"root_domain" bson:"root_domain"`
	VerificationStatus VerificationStatus `json:"verification_status" db:"verification_status" bson:"verification_status"`
	VerificationToken  string             `json:"verification_token" db:"verification_token" bson:"verification_token"`
	VerifiedAt         *time.Time         `json:"verified_at,omitempty" db:"verified_at" bson:"verified_at,omitempty"`
	AllowedCIDRs       []string           `json:"allowed_cidrs" db:"allowed_cidrs" bson:"allowed_cidrs"`
	ExcludedCIDRs      []string           `json:"excluded_cidrs" db:"excluded_cidrs" bson:"excluded_cidrs"`
	ExcludedSubdomains []string           `json:"excluded_subdomains" db:"excluded_subdomains" bson:"excluded_subdomains"`
	RateLimitRPS       int                `json:"rate_limit_rps" db:"rate_limit_rps" bson:"rate_limit_rps"`
	CreatedAt          time.Time          `json:"created_at" db:"created_at" bson:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at" db:"updated_at" bson:"updated_at"`
}

// StageCheckpoint stores execution status and output stats for pipeline persistence.
type StageCheckpoint struct {
	Stage       StageName   `json:"stage" bson:"stage"`
	Status      StageStatus `json:"status" bson:"status"`
	StartedAt   time.Time   `json:"started_at" bson:"started_at"`
	CompletedAt *time.Time  `json:"completed_at,omitempty" bson:"completed_at,omitempty"`
	DurationMs  int64       `json:"duration_ms" bson:"duration_ms"`
	OutputCount int         `json:"output_count" bson:"output_count"`
	Error       *string     `json:"error,omitempty" bson:"error,omitempty"`
}

// ScanMetrics contains aggregate pipeline performance metrics.
type ScanMetrics struct {
	TotalHosts      int     `json:"total_hosts" bson:"total_hosts"`
	LiveHosts       int     `json:"live_hosts" bson:"live_hosts"`
	OpenPorts       int     `json:"open_ports" bson:"open_ports"`
	FindingsCount   int     `json:"findings_count" bson:"findings_count"`
	DurationSeconds float64 `json:"duration_seconds" bson:"duration_seconds"`
	RequestsPerSec  float64 `json:"requests_per_sec" bson:"requests_per_sec"`
}

// ScanJob represents an executed or ongoing scan session.
type ScanJob struct {
	ID               uuid.UUID                     `json:"id" db:"id" bson:"_id"`
	ScopeID          uuid.UUID                     `json:"scope_id" db:"scope_id" bson:"scope_id"`
	Status           ScanStatus                    `json:"status" db:"status" bson:"status"`
	Profile          ScanProfile                   `json:"profile" db:"profile" bson:"profile"`
	CurrentStage     StageName                     `json:"current_stage" db:"current_stage" bson:"current_stage"`
	StageCheckpoints map[StageName]StageCheckpoint `json:"stage_checkpoints" db:"stage_checkpoints" bson:"stage_checkpoints"`
	Metrics          ScanMetrics                   `json:"metrics" db:"metrics" bson:"metrics"`
	ErrorMessage     *string                       `json:"error_message,omitempty" db:"error_message" bson:"error_message,omitempty"`
	DryRun           bool                          `json:"dry_run" db:"dry_run" bson:"dry_run"`
	StartedAt        *time.Time                    `json:"started_at,omitempty" db:"started_at" bson:"started_at,omitempty"`
	CompletedAt      *time.Time                    `json:"completed_at,omitempty" db:"completed_at" bson:"completed_at,omitempty"`
	CreatedAt        time.Time                     `json:"created_at" db:"created_at" bson:"created_at"`
	UpdatedAt        time.Time                     `json:"updated_at" db:"updated_at" bson:"updated_at"`
}

// TLSDetails holds parsed cryptographic attributes of an endpoint.
type TLSDetails struct {
	Subject     string     `json:"subject,omitempty" bson:"subject,omitempty"`
	Issuer      string     `json:"issuer,omitempty" bson:"issuer,omitempty"`
	ValidFrom   *time.Time `json:"valid_from,omitempty" bson:"valid_from,omitempty"`
	ValidUntil  *time.Time `json:"valid_until,omitempty" bson:"valid_until,omitempty"`
	SANs        []string   `json:"sans,omitempty" bson:"sans,omitempty"`
	CipherSuite string     `json:"cipher_suite,omitempty" bson:"cipher_suite,omitempty"`
	Version     string     `json:"version,omitempty" bson:"version,omitempty"`
}

// Asset represents a discovered host and its fingerprint.
type Asset struct {
	ID              uuid.UUID         `json:"id" db:"id" bson:"_id"`
	ScanJobID       uuid.UUID         `json:"scan_job_id" db:"scan_job_id" bson:"scan_job_id"`
	ScopeID         uuid.UUID         `json:"scope_id" db:"scope_id" bson:"scope_id"`
	FQDN            string            `json:"fqdn" db:"fqdn" bson:"fqdn"`
	RootDomain      string            `json:"root_domain" db:"root_domain" bson:"root_domain"`
	IPAddresses     []string          `json:"ip_addresses" db:"ip_addresses" bson:"ip_addresses"`
	ASN             string            `json:"asn,omitempty" db:"asn" bson:"asn,omitempty"`
	ASNOrg          string            `json:"asn_org,omitempty" db:"asn_org" bson:"asn_org,omitempty"`
	HTTPStatus      int               `json:"http_status,omitempty" db:"http_status" bson:"http_status,omitempty"`
	PageTitle       string            `json:"page_title,omitempty" db:"page_title" bson:"page_title,omitempty"`
	TechStack       []string          `json:"tech_stack" db:"tech_stack" bson:"tech_stack"`
	OpenPorts       []int             `json:"open_ports" db:"open_ports" bson:"open_ports"`
	TLSInfo         TLSDetails        `json:"tls_info" db:"tls_info" bson:"tls_info"`
	ResponseHeaders map[string]string `json:"response_headers,omitempty" db:"response_headers" bson:"response_headers,omitempty"`
	FirstSeenAt     time.Time         `json:"first_seen_at" db:"first_seen_at" bson:"first_seen_at"`
	LastSeenAt      time.Time         `json:"last_seen_at" db:"last_seen_at" bson:"last_seen_at"`
	CreatedAt       time.Time         `json:"created_at" db:"created_at" bson:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at" db:"updated_at" bson:"updated_at"`
}

// AssetDrift records delta changes across scans.
type AssetDrift struct {
	ID                uuid.UUID              `json:"id" db:"id" bson:"_id"`
	ScanJobID         uuid.UUID              `json:"scan_job_id" db:"scan_job_id" bson:"scan_job_id"`
	PreviousScanJobID *uuid.UUID             `json:"previous_scan_job_id,omitempty" db:"previous_scan_job_id" bson:"previous_scan_job_id,omitempty"`
	AssetID           *uuid.UUID             `json:"asset_id,omitempty" db:"asset_id" bson:"asset_id,omitempty"`
	FQDN              string                 `json:"fqdn" db:"fqdn" bson:"fqdn"`
	ChangeType        DriftChangeType        `json:"change_type" db:"change_type" bson:"change_type"`
	Details           map[string]interface{} `json:"details" db:"details" bson:"details"`
	Severity          FindingSeverity        `json:"severity" db:"severity" bson:"severity"`
	DetectedAt        time.Time              `json:"detected_at" db:"detected_at" bson:"detected_at"`
}

// RawEvidence encapsulates audit verification logs and payloads.
type RawEvidence struct {
	MatchedAt        string            `json:"matched_at,omitempty" bson:"matched_at,omitempty"`
	Request          string            `json:"request,omitempty" bson:"request,omitempty"`
	Response         string            `json:"response,omitempty" bson:"response,omitempty"`
	MatcherName      string            `json:"matcher_name,omitempty" bson:"matcher_name,omitempty"`
	ExtractedResults []string          `json:"extracted_results,omitempty" bson:"extracted_results,omitempty"`
	CurlCommand      string            `json:"curl_command,omitempty" bson:"curl_command,omitempty"`
	ExtraHeaders     map[string]string `json:"extra_headers,omitempty" bson:"extra_headers,omitempty"`
}

// Finding represents an audited security vulnerability or misconfiguration.
type Finding struct {
	ID            uuid.UUID       `json:"id" db:"id" bson:"_id"`
	ScanJobID     uuid.UUID       `json:"scan_job_id" db:"scan_job_id" bson:"scan_job_id"`
	AssetID       uuid.UUID       `json:"asset_id" db:"asset_id" bson:"asset_id"`
	FQDN          string          `json:"fqdn,omitempty" db:"-" bson:"fqdn,omitempty"`
	Tool          string          `json:"tool" db:"tool" bson:"tool"`
	RuleID        string          `json:"rule_id" db:"rule_id" bson:"rule_id"`
	Title         string          `json:"title" db:"title" bson:"title"`
	Severity      FindingSeverity `json:"severity" db:"severity" bson:"severity"`
	Description   string          `json:"description" db:"description" bson:"description"`
	CVEIDs        []string        `json:"cve_ids" db:"cve_ids" bson:"cve_ids"`
	CVSSScore     float64         `json:"cvss_score" db:"cvss_score" bson:"cvss_score"`
	Remediation   string          `json:"remediation" db:"remediation" bson:"remediation"`
	RawEvidence   RawEvidence     `json:"raw_evidence" db:"raw_evidence" bson:"raw_evidence"`
	FalsePositive bool            `json:"false_positive" db:"false_positive" bson:"false_positive"`
	Status        FindingStatus   `json:"status" db:"status" bson:"status"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at" bson:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at" bson:"updated_at"`
}

// DriftAnalysis breaks down state changes synthesized for AI review.
type DriftAnalysis struct {
	TotalNewHosts      int      `json:"total_new_hosts" bson:"total_new_hosts"`
	TotalRemovedHosts  int      `json:"total_removed_hosts" bson:"total_removed_hosts"`
	TotalOpenedPorts   int      `json:"total_opened_ports" bson:"total_opened_ports"`
	TotalClosedPorts   int      `json:"total_closed_ports" bson:"total_closed_ports"`
	HighRiskMutations  []string `json:"high_risk_mutations" bson:"high_risk_mutations"`
	SummaryDescription string   `json:"summary_description" bson:"summary_description"`
}

// PrioritizedFinding groups finding details with urgency ranking.
type PrioritizedFinding struct {
	Rank               int             `json:"rank" bson:"rank"`
	Title              string          `json:"title" bson:"title"`
	Severity           FindingSeverity `json:"severity" bson:"severity"`
	AffectedAsset      string          `json:"affected_asset" bson:"affected_asset"`
	ExploitProbability string          `json:"exploit_probability" bson:"exploit_probability"`
	ImpactAnalysis     string          `json:"impact_analysis" bson:"impact_analysis"`
	RootCause          string          `json:"root_cause" bson:"root_cause"`
}

// RemediationItem holds actionable code and config fixes.
type RemediationItem struct {
	Title           string   `json:"title" bson:"title"`
	TargetCategory  string   `json:"target_category" bson:"target_category"`
	Severity        string   `json:"severity" bson:"severity"`
	ConfigSnippet   string   `json:"config_snippet" bson:"config_snippet"`
	Explanation     string   `json:"explanation" bson:"explanation"`
	ValidationSteps []string `json:"validation_steps" bson:"validation_steps"`
}

// AIReport defines the structured JSON output synthesized by Google GenAI.
type AIReport struct {
	ID                         uuid.UUID            `json:"id" db:"id" bson:"_id"`
	ScanJobID                  uuid.UUID            `json:"scan_job_id" db:"scan_job_id" bson:"scan_job_id"`
	ExecutiveSummary           string               `json:"executive_summary" db:"executive_summary" bson:"executive_summary"`
	OverallRiskScore           int                  `json:"overall_risk_score" db:"overall_risk_score" bson:"overall_risk_score"`
	AttackSurfaceDriftAnalysis DriftAnalysis        `json:"attack_surface_drift_analysis" db:"attack_surface_drift_analysis" bson:"attack_surface_drift_analysis"`
	PrioritizedFindings        []PrioritizedFinding `json:"prioritized_findings" db:"prioritized_findings" bson:"prioritized_findings"`
	RemediationPlaybook        []RemediationItem    `json:"remediation_playbook" db:"remediation_playbook" bson:"remediation_playbook"`
	TokensUsed                 int                  `json:"tokens_used" db:"tokens_used" bson:"tokens_used"`
	ModelVersion               string               `json:"model_version" db:"model_version" bson:"model_version"`
	IsFallback                 bool                 `json:"is_fallback" db:"is_fallback" bson:"is_fallback"`
	GeneratedAt                time.Time            `json:"generated_at" db:"generated_at" bson:"generated_at"`
}

// TelemetryEvent carries streaming terminal logs and progress updates to clients.
type TelemetryEvent struct {
	Type      string      `json:"type" bson:"type"`
	ScanJobID uuid.UUID   `json:"scan_job_id" bson:"scan_job_id"`
	Stage     StageName   `json:"stage,omitempty" bson:"stage,omitempty"`
	Timestamp time.Time   `json:"timestamp" bson:"timestamp"`
	Payload   interface{} `json:"payload" bson:"payload"`
}
