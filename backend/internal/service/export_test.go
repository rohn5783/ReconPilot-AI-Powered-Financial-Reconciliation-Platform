package service

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"reconpilot/internal/domain"
)

func TestExportServiceMarkdownAndJSON(t *testing.T) {
	exporter := NewExportService()

	scope := &domain.Scope{
		ID:         uuid.New(),
		RootDomain: "example.com",
	}

	job := &domain.ScanJob{
		ID:      uuid.New(),
		ScopeID: scope.ID,
		Profile: domain.ProfileStandard,
		Metrics: domain.ScanMetrics{
			TotalHosts:      2,
			DurationSeconds: 15.2,
		},
	}

	assets := []*domain.Asset{
		{
			ID:          uuid.New(),
			FQDN:        "api.example.com",
			HTTPStatus:  200,
			IPAddresses: []string{"104.21.10.5"},
			OpenPorts:   []int{80, 443},
			TechStack:   []string{"Cloudflare", "Go"},
		},
	}

	findings := []*domain.Finding{
		{
			ID:        uuid.New(),
			FQDN:      "api.example.com",
			Title:     "Permissive CORS",
			Severity:  domain.SeverityMedium,
			CVSSScore: 6.5,
			RuleID:    "cors-wildcard",
			Tool:      "nuclei",
		},
	}

	report := &domain.AIReport{
		ID:               uuid.New(),
		ScanJobID:        job.ID,
		ExecutiveSummary: "Overall posture is nominal.",
		OverallRiskScore: 35,
		RemediationPlaybook: []domain.RemediationItem{
			{
				Title:          "Tighten CORS Whitelist",
				TargetCategory: "Nginx",
				ConfigSnippet:  "add_header Access-Control-Allow-Origin $cors_origin;",
			},
		},
		GeneratedAt: time.Now(),
	}

	// 1. Test Markdown Export
	md, err := exporter.ExportMarkdown(scope, job, assets, nil, findings, report)
	if err != nil {
		t.Fatalf("unexpected error exporting markdown: %v", err)
	}

	if !strings.Contains(md, "example.com") {
		t.Errorf("expected markdown to contain root domain example.com")
	}
	if !strings.Contains(md, "Permissive CORS") {
		t.Errorf("expected markdown to contain finding title")
	}
	if !strings.Contains(md, "Tighten CORS Whitelist") {
		t.Errorf("expected markdown to contain playbook item")
	}

	// 2. Test JSON Export
	jsonBytes, err := exporter.ExportJSON(scope, job, assets, nil, findings, report)
	if err != nil {
		t.Fatalf("unexpected error exporting JSON: %v", err)
	}

	if len(jsonBytes) == 0 {
		t.Fatalf("expected non-empty JSON bytes")
	}
	if !strings.Contains(string(jsonBytes), `"target_domain": ""`) && !strings.Contains(string(jsonBytes), `"root_domain": "example.com"`) {
		t.Errorf("expected JSON to contain root_domain example.com")
	}
}

