package ai

import (
	"testing"

	"github.com/google/uuid"

	"reconpilot/internal/domain"
)

func TestFallbackGeneratorRiskScoreAndPlaybook(t *testing.T) {
	generator := NewFallbackGenerator()

	payload := &CompactPayload{
		TargetDomain: "example.com",
		TotalAssets:  5,
		Assets: []CompactAsset{
			{FQDN: "staging.example.com", OpenPorts: []int{80, 6379}},
		},
		SeverityStats: map[string]int{
			"critical": 1,
			"high":     1,
			"medium":   0,
			"low":      1,
		},
		Findings: []CompactFinding{
			{
				RuleID:        "redis-unauthenticated-access",
				Title:         "Exposed Redis",
				Severity:      domain.SeverityCritical,
				AffectedHosts: []string{"staging.example.com"},
				CVSSScore:     9.8,
			},
			{
				RuleID:        "git-config-disclosure",
				Title:         "Exposed Git Config",
				Severity:      domain.SeverityHigh,
				AffectedHosts: []string{"staging.example.com"},
				CVSSScore:     7.5,
			},
		},
	}

	report := generator.Generate(uuid.New(), payload)

	if report == nil {
		t.Fatalf("expected non-nil report from fallback generator")
	}

	if !report.IsFallback {
		t.Errorf("expected IsFallback to be true")
	}

	if report.OverallRiskScore < 50 {
		t.Errorf("expected high risk score >= 50 given critical + high findings, got %d", report.OverallRiskScore)
	}

	if len(report.PrioritizedFindings) != 2 {
		t.Errorf("expected 2 prioritized findings, got %d", len(report.PrioritizedFindings))
	}

	if len(report.RemediationPlaybook) == 0 {
		t.Errorf("expected remediation playbook items to be generated")
	}

	// Verify Redis config snippet generated
	foundRedisFix := false
	for _, item := range report.RemediationPlaybook {
		if item.TargetCategory == "AWS_SG / Linux Firewall" {
			foundRedisFix = true
		}
	}
	if !foundRedisFix {
		t.Errorf("expected Redis defensive remediation item in playbook")
	}
}

