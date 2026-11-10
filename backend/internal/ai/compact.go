package ai

import (
	"fmt"
	"sort"
	"strings"

	"reconpilot/internal/domain"
)

// CompactFinding represents a deduplicated, token-minimized security issue.
type CompactFinding struct {
	RuleID         string                 `json:"rule_id"`
	Title          string                 `json:"title"`
	Severity       domain.FindingSeverity `json:"severity"`
	CVSSScore      float64                `json:"cvss_score"`
	AffectedHosts  []string               `json:"affected_hosts"`
	Count          int                    `json:"occurrence_count"`
	Description    string                 `json:"description"`
	CVEIDs         []string               `json:"cve_ids,omitempty"`
	SampleEvidence string                 `json:"sample_evidence,omitempty"`
}

// CompactAsset represents an asset with redundant data stripped for AI ingestion.
type CompactAsset struct {
	FQDN       string   `json:"fqdn"`
	IPs        []string `json:"ips,omitempty"`
	OpenPorts  []int    `json:"open_ports,omitempty"`
	TechStack  []string `json:"tech_stack,omitempty"`
	HTTPStatus int      `json:"http_status,omitempty"`
}

// TokenOptimizer compacts assets, drift mutations, and findings to minimize GenAI token consumption.
type TokenOptimizer struct{}

// NewTokenOptimizer creates a new TokenOptimizer.
func NewTokenOptimizer() *TokenOptimizer {
	return &TokenOptimizer{}
}

// CompactPayload packages scan outputs into a high-density, low-token structure.
type CompactPayload struct {
	TargetDomain  string                 `json:"target_domain"`
	TotalAssets   int                    `json:"total_assets"`
	Assets        []CompactAsset         `json:"assets"`
	DriftSummary  *domain.DriftAnalysis  `json:"drift_summary,omitempty"`
	Findings      []CompactFinding       `json:"findings"`
	SeverityStats map[string]int         `json:"severity_stats"`
}

// Optimize reduces findings and assets into their most salient representation.
func (o *TokenOptimizer) Optimize(
	targetDomain string,
	assets []*domain.Asset,
	findings []*domain.Finding,
	drift *domain.DriftAnalysis,
) *CompactPayload {
	// 1. Compact and Deduplicate Findings by RuleID
	findingMap := make(map[string]*CompactFinding)
	severityStats := map[string]int{
		"critical": 0,
		"high":     0,
		"medium":   0,
		"low":      0,
		"info":     0,
	}

	for _, f := range findings {
		sevKey := strings.ToLower(string(f.Severity))
		severityStats[sevKey]++

		key := fmt.Sprintf("%s:%s", f.RuleID, f.Severity)
		existing, ok := findingMap[key]
		if !ok {
			sample := f.RawEvidence.MatchedAt
			if f.RawEvidence.Request != "" {
				// Keep first 200 chars of request
				reqSnippet := f.RawEvidence.Request
				if len(reqSnippet) > 200 {
					reqSnippet = reqSnippet[:200] + "..."
				}
				sample = fmt.Sprintf("%s | Req: %s", sample, reqSnippet)
			}

			findingMap[key] = &CompactFinding{
				RuleID:         f.RuleID,
				Title:          f.Title,
				Severity:       f.Severity,
				CVSSScore:      f.CVSSScore,
				AffectedHosts:  []string{f.FQDN},
				Count:          1,
				Description:    f.Description,
				CVEIDs:         f.CVEIDs,
				SampleEvidence: sample,
			}
		} else {
			existing.Count++
			if !containsString(existing.AffectedHosts, f.FQDN) && len(existing.AffectedHosts) < 5 {
				existing.AffectedHosts = append(existing.AffectedHosts, f.FQDN)
			}
		}
	}

	var compactFindings []CompactFinding
	for _, cf := range findingMap {
		compactFindings = append(compactFindings, *cf)
	}

	// Sort findings by severity and CVSS
	sort.Slice(compactFindings, func(i, j int) bool {
		if severityWeight(compactFindings[i].Severity) != severityWeight(compactFindings[j].Severity) {
			return severityWeight(compactFindings[i].Severity) > severityWeight(compactFindings[j].Severity)
		}
		return compactFindings[i].CVSSScore > compactFindings[j].CVSSScore
	})

	// 2. Compact Assets
	var compactAssets []CompactAsset
	for _, a := range assets {
		compactAssets = append(compactAssets, CompactAsset{
			FQDN:       a.FQDN,
			IPs:        a.IPAddresses,
			OpenPorts:  a.OpenPorts,
			TechStack:  a.TechStack,
			HTTPStatus: a.HTTPStatus,
		})
	}

	return &CompactPayload{
		TargetDomain:  targetDomain,
		TotalAssets:   len(assets),
		Assets:        compactAssets,
		DriftSummary:  drift,
		Findings:      compactFindings,
		SeverityStats: severityStats,
	}
}

func containsString(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func severityWeight(s domain.FindingSeverity) int {
	switch s {
	case domain.SeverityCritical:
		return 5
	case domain.SeverityHigh:
		return 4
	case domain.SeverityMedium:
		return 3
	case domain.SeverityLow:
		return 2
	default:
		return 1
	}
}

