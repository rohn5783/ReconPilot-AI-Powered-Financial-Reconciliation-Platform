package ai

import (
	"testing"

	"reconpilot/internal/domain"
)

func TestTokenOptimizerDeduplication(t *testing.T) {
	optimizer := NewTokenOptimizer()

	assets := []*domain.Asset{
		{FQDN: "api.example.com", OpenPorts: []int{80, 443}, TechStack: []string{"Nginx"}},
		{FQDN: "staging.example.com", OpenPorts: []int{80, 6379}, TechStack: []string{"Docker"}},
	}

	// 3 duplicate findings for the same rule on different hosts
	findings := []*domain.Finding{
		{
			RuleID:      "missing-hsts",
			Title:       "Missing HSTS",
			Severity:    domain.SeverityLow,
			FQDN:        "api.example.com",
			Description: "Header absent",
		},
		{
			RuleID:      "missing-hsts",
			Title:       "Missing HSTS",
			Severity:    domain.SeverityLow,
			FQDN:        "staging.example.com",
			Description: "Header absent",
		},
		{
			RuleID:      "exposed-redis",
			Title:       "Exposed Redis",
			Severity:    domain.SeverityCritical,
			FQDN:        "staging.example.com",
			Description: "Port 6379 accessible",
			CVSSScore:   9.8,
		},
	}

	compact := optimizer.Optimize("example.com", assets, findings, nil)

	if compact.TotalAssets != 2 {
		t.Errorf("expected 2 assets, got %d", compact.TotalAssets)
	}

	// The 2 duplicate missing-hsts findings should be compacted into 1 record with count 2
	if len(compact.Findings) != 2 {
		t.Fatalf("expected 2 unique compact findings, got %d", len(compact.Findings))
	}

	// First finding should be the Critical one (sorted by severity)
	if compact.Findings[0].Severity != domain.SeverityCritical {
		t.Errorf("expected first finding to be Critical, got %s", compact.Findings[0].Severity)
	}

	// Second finding should have count 2 and both hosts
	if compact.Findings[1].Count != 2 {
		t.Errorf("expected occurrence count 2 for missing-hsts, got %d", compact.Findings[1].Count)
	}
	if len(compact.Findings[1].AffectedHosts) != 2 {
		t.Errorf("expected 2 affected hosts for missing-hsts, got %d", len(compact.Findings[1].AffectedHosts))
	}
}

