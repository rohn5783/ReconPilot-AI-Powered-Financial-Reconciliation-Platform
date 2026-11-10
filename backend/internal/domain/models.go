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
	ID                  uuid.UUID          `json:"id" db:"id"`
	Name                string             `json:"name" db:"name"`
	RootDomain          string             `json:"root_domain" db:"root_domain"`
	VerificationStatus  VerificationStatus `json:"verification_status" db:"verification_status"`
	VerificationToken   string             `json:"verification_token" db:"verification_token"`
	VerifiedAt          *time.Time         `json:"verified_at,omitempty" db:"verified_at"`
	AllowedCIDRs        []string           `json:"allowed_cidrs" db:"allowed_cidrs"`
	ExcludedCIDRs       []string           `json:"excluded_cidrs" db:"excluded_cidrs"`
	ExcludedSubdomains  []string           `json:"excluded_subdomains" db:"excluded_subdomains"`
	RateLimitRPS        int                `json:"rate_limit_rps" db:"rate_limit_rps"`
	CreatedAt           time.Time          `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time          `json:"updated_at" db:"updated_at"`
}

// StageCheckpoint stores execution status and output stats for pipeline persistence.
type StageCheckpoint struct {
	Stage       StageName   `json:"stage"`
	Status      StageStatus `json:"status"`
	StartedAt   time.Time   `json:"started_at"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
	DurationMs  int64       `json:"duration_ms"`
	OutputCount int         `json:"output_count"`
	Error       *string     `json:"error,omitempty"`
}

// ScanMetrics contains aggregate pipeline performance metrics.
type ScanMetrics struct {
	TotalHosts      int     `json:"total_hosts"`
	LiveHosts       int     `json:"live_hosts"`
	OpenPorts       int     `json:"open_ports"`
	FindingsCount   int     `json:"findings_count"`
	DurationSeconds float64 `json:"duration_seconds"`
	RequestsPerSec  float64 `json:"requests_per_sec"`
}

// ScanJob represents an executed or ongoing scan session.
type ScanJob struct {
	ID               uuid.UUID                   `json:"id" db:"id"`
	ScopeID          uuid.UUID                   `json:"scope_id" db:"scope_id"`
	Status           ScanStatus                  `json:"status" db:"status"`
	Profile          ScanProfile                 `json:"profile" db:"profile"`
	CurrentStage     StageName                   `json:"current_stage" db:"current_stage"`
	StageCheckpoints map[StageName]StageCheckpoint `json:"stage_checkpoints" db:"stage_checkpoints"`
	Metrics          ScanMetrics                 `json:"metrics" db:"metrics"`
	ErrorMessage     *string                     `json:"error_message,omitempty" db:"error_message"`
	DryRun           bool                        `json:"dry_run" db:"dry_run"`
	StartedAt        *time.Time                  `json:"started_at,omitempty" db:"started_at"`
	CompletedAt      *time.Time                  `json:"completed_at,omitempty" db:"completed_at"`
	CreatedAt        time.Time                   `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time                   `json:"updated_at" db:"updated_at"`
}

// TLSDetails holds parsed cryptographic attributes of an endpoint.
type TLSDetails struct {
	Subject     string     `json:"subject,omitempty"`
	Issuer      string     `json:"issuer,omitempty"`
	ValidFrom   *time.Time `json:"valid_from,omitempty"`
	ValidUntil  *time.Time `json:"valid_until,omitempty"`
	SANs        []string   `json:"sans,omitempty"`
	CipherSuite string     `json:"cipher_suite,omitempty"`
	Version     string     `json:"version,omitempty"`
}

// Asset represents a discovered host and its fingerprint.
type Asset struct {
	ID              uuid.UUID         `json:"id" db:"id"`
	ScanJobID       uuid.UUID         `json:"scan_job_id" db:"scan_job_id"`
	ScopeID         uuid.UUID         `json:"scope_id" db:"scope_id"`
	FQDN            string            `json:"fqdn" db:"fqdn"`
	RootDomain      string            `json:"root_domain" db:"root_domain"`
	IPAddresses     []string          `json:"ip_addresses" db:"ip_addresses"`
	ASN             string            `json:"asn,omitempty" db:"asn"`
	ASNOrg          string            `json:"asn_org,omitempty" db:"asn_org"`
	HTTPStatus      int               `json:"http_status,omitempty" db:"http_status"`
	PageTitle       string            `json:"page_title,omitempty" db:"page_title"`
	TechStack       []string          `json:"tech_stack" db:"tech_stack"`
	OpenPorts       []int             `json:"open_ports" db:"open_ports"`
	TLSInfo         TLSDetails        `json:"tls_info" db:"tls_info"`
	ResponseHeaders map[string]string `json:"response_headers,omitempty" db:"response_headers"`
	FirstSeenAt     time.Time         `json:"first_seen_at" db:"first_seen_at"`
	LastSeenAt      time.Time         `json:"last_seen_at" db:"last_seen_at"`
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at" db:"updated_at"`
}

// AssetDrift records delta changes across scans.
type AssetDrift struct {
	ID                 uuid.UUID              `json:"id" db:"id"`
	ScanJobID          uuid.UUID              `json:"scan_job_id" db:"scan_job_id"`
	PreviousScanJobID  *uuid.UUID             `json:"previous_scan_job_id,omitempty" db:"previous_scan_job_id"`
	AssetID            *uuid.UUID             `json:"asset_id,omitempty" db:"asset_id"`
	FQDN               string                 `json:"fqdn" db:"fqdn"`
	ChangeType         DriftChangeType        `json:"change_type" db:"change_type"`
	Details            map[string]interface{} `json:"details" db:"details"`
	Severity           FindingSeverity        `json:"severity" db:"severity"`
	DetectedAt         time.Time              `json:"detected_at" db:"detected_at"`
}

// RawEvidence encapsulates audit verification logs and payloads.
type RawEvidence struct {
	MatchedAt        string            `json:"matched_at,omitempty"`
	Request          string            `json:"request,omitempty"`
	Response         string            `json:"response,omitempty"`
	MatcherName      string            `json:"matcher_name,omitempty"`
	ExtractedResults []string          `json:"extracted_results,omitempty"`
	CurlCommand      string            `json:"curl_command,omitempty"`
	ExtraHeaders     map[string]string `json:"extra_headers,omitempty"`
}

// Finding represents an audited security vulnerability or misconfiguration.
type Finding struct {
	ID            uuid.UUID       `json:"id" db:"id"`
	ScanJobID     uuid.UUID       `json:"scan_job_id" db:"scan_job_id"`
	AssetID       uuid.UUID       `json:"asset_id" db:"asset_id"`
	FQDN          string          `json:"fqdn,omitempty" db:"-"`
	Tool          string          `json:"tool" db:"tool"`
	RuleID        string          `json:"rule_id" db:"rule_id"`
	Title         string          `json:"title" db:"title"`
	Severity      FindingSeverity `json:"severity" db:"severity"`
	Description   string          `json:"description" db:"description"`
	CVEIDs        []string        `json:"cve_ids" db:"cve_ids"`
	CVSSScore     float64         `json:"cvss_score" db:"cvss_score"`
	Remediation   string          `json:"remediation" db:"remediation"`
	RawEvidence   RawEvidence     `json:"raw_evidence" db:"raw_evidence"`
	FalsePositive bool            `json:"false_positive" db:"false_positive"`
	Status        FindingStatus   `json:"status" db:"status"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}

// DriftAnalysis breaks down state changes synthesized for AI review.
type DriftAnalysis struct {
	TotalNewHosts      int      `json:"total_new_hosts"`
	TotalRemovedHosts  int      `json:"total_removed_hosts"`
	TotalOpenedPorts   int      `json:"total_opened_ports"`
	TotalClosedPorts   int      `json:"total_closed_ports"`
	HighRiskMutations  []string `json:"high_risk_mutations"`
	SummaryDescription string   `json:"summary_description"`
}

// PrioritizedFinding groups finding details with urgency ranking.
type PrioritizedFinding struct {
	Rank               int             `json:"rank"`
	Title              string          `json:"title"`
	Severity           FindingSeverity `json:"severity"`
	AffectedAsset      string          `json:"affected_asset"`
	ExploitProbability string          `json:"exploit_probability"`
	ImpactAnalysis     string          `json:"impact_analysis"`
	RootCause          string          `json:"root_cause"`
}

// RemediationItem holds actionable code and config fixes.
type RemediationItem struct {
	Title           string   `json:"title"`
	TargetCategory  string   `json:"target_category"` // "Nginx", "Cloudflare", "AWS_SG", "Docker", "TLS"
	Severity        string   `json:"severity"`
	ConfigSnippet   string   `json:"config_snippet"`
	Explanation     string   `json:"explanation"`
	ValidationSteps []string `json:"validation_steps"`
}

// AIReport defines the structured JSON output synthesized by Google GenAI.
type AIReport struct {
	ID                          uuid.UUID            `json:"id" db:"id"`
	ScanJobID                   uuid.UUID            `json:"scan_job_id" db:"scan_job_id"`
	ExecutiveSummary            string               `json:"executive_summary" db:"executive_summary"`
	OverallRiskScore            int                  `json:"overall_risk_score" db:"overall_risk_score"`
	AttackSurfaceDriftAnalysis  DriftAnalysis        `json:"attack_surface_drift_analysis" db:"attack_surface_drift_analysis"`
	PrioritizedFindings         []PrioritizedFinding `json:"prioritized_findings" db:"prioritized_findings"`
	RemediationPlaybook         []RemediationItem    `json:"remediation_playbook" db:"remediation_playbook"`
	TokensUsed                  int                  `json:"tokens_used" db:"tokens_used"`
	ModelVersion                string               `json:"model_version" db:"model_version"`
	IsFallback                  bool                 `json:"is_fallback" db:"is_fallback"`
	GeneratedAt                 time.Time            `json:"generated_at" db:"generated_at"`
}

// TelemetryEvent carries streaming terminal logs and progress updates to clients.
type TelemetryEvent struct {
	Type      string      `json:"type"` // "log", "stage_update", "metric", "finding", "asset"
	ScanJobID uuid.UUID   `json:"scan_job_id"`
	Stage     StageName   `json:"stage,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

