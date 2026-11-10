package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"reconpilot/internal/domain"
)

// ExportService produces formatted Markdown and machine-readable JSON exports.
type ExportService struct{}

// NewExportService creates a new ExportService.
func NewExportService() *ExportService {
	return &ExportService{}
}

// FullScanExport wraps all scan artifacts for JSON export.
type FullScanExport struct {
	Scope       *domain.Scope        `json:"scope"`
	ScanJob     *domain.ScanJob      `json:"scan_job"`
	Metrics     domain.ScanMetrics   `json:"metrics"`
	Assets      []*domain.Asset      `json:"assets"`
	Drifts      []*domain.AssetDrift `json:"drifts"`
	Findings    []*domain.Finding    `json:"findings"`
	AIReport    *domain.AIReport     `json:"ai_report,omitempty"`
	ExportedAt  time.Time            `json:"exported_at"`
}

// ExportJSON generates a complete, formatted JSON dump of the scan results.
func (s *ExportService) ExportJSON(
	scope *domain.Scope,
	job *domain.ScanJob,
	assets []*domain.Asset,
	drifts []*domain.AssetDrift,
	findings []*domain.Finding,
	report *domain.AIReport,
) ([]byte, error) {
	dump := FullScanExport{
		Scope:      scope,
		ScanJob:    job,
		Metrics:    job.Metrics,
		Assets:     assets,
		Drifts:     drifts,
		Findings:   findings,
		AIReport:   report,
		ExportedAt: time.Now(),
	}

	return json.MarshalIndent(dump, "", "  ")
}

// ExportMarkdown generates a production-grade industrial engineering briefing in Markdown.
func (s *ExportService) ExportMarkdown(
	scope *domain.Scope,
	job *domain.ScanJob,
	assets []*domain.Asset,
	drifts []*domain.AssetDrift,
	findings []*domain.Finding,
	report *domain.AIReport,
) (string, error) {
	var sb strings.Builder

	rootDomain := "Unknown"
	if scope != nil {
		rootDomain = scope.RootDomain
	}

	scanDate := time.Now().Format("2006-01-02 15:04:05 UTC")
	if job != nil && job.CompletedAt != nil {
		scanDate = job.CompletedAt.Format("2006-01-02 15:04:05 UTC")
	}

	riskScore := 0
	execSummary := "No executive summary generated."
	if report != nil {
		riskScore = report.OverallRiskScore
		execSummary = report.ExecutiveSummary
	}

	// 1. Header & Posture
	sb.WriteString(fmt.Sprintf("# ReconPilot External Attack Surface Assessment\n\n"))
	sb.WriteString(fmt.Sprintf("**Target Boundary:** `%s`  \n", rootDomain))
	sb.WriteString(fmt.Sprintf("**Scan ID:** `%s`  \n", job.ID))
	sb.WriteString(fmt.Sprintf("**Execution Profile:** `%s` (DryRun: %v)  \n", job.Profile, job.DryRun))
	sb.WriteString(fmt.Sprintf("**Completed At:** `%s`  \n", scanDate))
	sb.WriteString(fmt.Sprintf("**Calculated Risk Score:** `%d / 100`  \n\n", riskScore))
	sb.WriteString("---\n\n")

	// 2. Executive Posture Summary
	sb.WriteString("## 1. Executive Posture Summary\n\n")
	sb.WriteString(execSummary + "\n\n")

	// Telemetry Metrics Grid
	sb.WriteString("### Perimeter Telemetry Metrics\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("| :--- | :--- |\n")
	sb.WriteString(fmt.Sprintf("| Discovered Hosts | `%d` |\n", len(assets)))
	sb.WriteString(fmt.Sprintf("| Total Security Findings | `%d` |\n", len(findings)))
	sb.WriteString(fmt.Sprintf("| Detected State Drifts | `%d` |\n", len(drifts)))
	sb.WriteString(fmt.Sprintf("| Total Scan Duration | `%.1fs` |\n\n", job.Metrics.DurationSeconds))

	// 3. Attack Surface Drift Breakdown
	if len(drifts) > 0 {
		sb.WriteString("## 2. Attack Surface Historical Drift Analysis\n\n")
		sb.WriteString("| Change Type | Affected Target | Severity | Details |\n")
		sb.WriteString("| :--- | :--- | :--- | :--- |\n")
		for _, d := range drifts {
			detailsJSON, _ := json.Marshal(d.Details)
			sb.WriteString(fmt.Sprintf("| `%s` | `%s` | **%s** | `%s` |\n",
				d.ChangeType, d.FQDN, strings.ToUpper(string(d.Severity)), string(detailsJSON)))
		}
		sb.WriteString("\n")
	}

	// 4. Prioritized Vulnerability Findings
	sb.WriteString("## 3. Prioritized Security Findings\n\n")
	if len(findings) == 0 {
		sb.WriteString("_No critical or high-risk misconfigurations were identified._\n\n")
	} else {
		for i, f := range findings {
			cveStr := "None"
			if len(f.CVEIDs) > 0 {
				cveStr = strings.Join(f.CVEIDs, ", ")
			}
			sb.WriteString(fmt.Sprintf("### 3.%d [%s] %s\n\n", i+1, strings.ToUpper(string(f.Severity)), f.Title))
			sb.WriteString(fmt.Sprintf("- **Target Asset:** `%s`\n", f.FQDN))
			sb.WriteString(fmt.Sprintf("- **CVSS Score:** `%.1f` (CVEs: %s)\n", f.CVSSScore, cveStr))
			sb.WriteString(fmt.Sprintf("- **Rule / Template:** `%s` (Engine: %s)\n", f.RuleID, f.Tool))
			sb.WriteString(fmt.Sprintf("- **Description:** %s\n\n", f.Description))

			if f.RawEvidence.MatchedAt != "" {
				sb.WriteString(fmt.Sprintf("**Evidence Matched:** `%s`\n\n", f.RawEvidence.MatchedAt))
			}
			if f.RawEvidence.Request != "" {
				sb.WriteString("```http\n" + f.RawEvidence.Request + "\n```\n\n")
			}
			if f.Remediation != "" {
				sb.WriteString(fmt.Sprintf("**Recommended Action:** %s\n\n", f.Remediation))
			}
			sb.WriteString("---\n\n")
		}
	}

	// 5. Remediation Playbook (Defensive Hardening Snippets)
	if report != nil && len(report.RemediationPlaybook) > 0 {
		sb.WriteString("## 4. Defensive Remediation Playbook\n\n")
		for i, item := range report.RemediationPlaybook {
			sb.WriteString(fmt.Sprintf("### 4.%d %s\n\n", i+1, item.Title))
			sb.WriteString(fmt.Sprintf("**Category:** `%s` | **Severity:** `%s`  \n\n", item.TargetCategory, item.Severity))
			sb.WriteString(item.Explanation + "\n\n")

			sb.WriteString("```bash\n" + item.ConfigSnippet + "\n```\n\n")

			if len(item.ValidationSteps) > 0 {
				sb.WriteString("**Verification Steps:**\n")
				for _, step := range item.ValidationSteps {
					sb.WriteString(fmt.Sprintf("- %s\n", step))
				}
				sb.WriteString("\n")
			}
		}
	}

	// 6. Complete Discovered Asset Inventory
	sb.WriteString("## 5. Discovered Asset Inventory\n\n")
	sb.WriteString("| FQDN | HTTP Status | Resolved IPs | Open Ports | Tech Stack |\n")
	sb.WriteString("| :--- | :--- | :--- | :--- | :--- |\n")
	for _, a := range assets {
		ips := strings.Join(a.IPAddresses, ", ")
		if ips == "" {
			ips = "—"
		}
		var portsStr []string
		for _, p := range a.OpenPorts {
			portsStr = append(portsStr, fmt.Sprintf("%d", p))
		}
		ports := strings.Join(portsStr, ", ")
		if ports == "" {
			ports = "80, 443"
		}
		tech := strings.Join(a.TechStack, ", ")
		if tech == "" {
			tech = "—"
		}
		sb.WriteString(fmt.Sprintf("| `%s` | `%d` | %s | `%s` | %s |\n",
			a.FQDN, a.HTTPStatus, ips, ports, tech))
	}
	sb.WriteString("\n")

	sb.WriteString("---\n")
	sb.WriteString("_Generated autonomously by ReconPilot EASM Pipeline._\n")

	return sb.String(), nil
}

