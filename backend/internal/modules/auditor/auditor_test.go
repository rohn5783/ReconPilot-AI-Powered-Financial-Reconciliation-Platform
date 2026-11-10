package auditor

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"reconpilot/internal/domain"
	"reconpilot/internal/modules"
	"reconpilot/pkg/runner"
)

func TestAuditorModuleMockMode(t *testing.T) {
	auditor := NewAuditorModule("nuclei")

	scope := &domain.Scope{
		ID:         uuid.New(),
		RootDomain: "example.com",
	}

	job := &domain.ScanJob{
		ID:      uuid.New(),
		ScopeID: scope.ID,
		DryRun:  true,
	}

	assets := []*domain.Asset{
		{
			ID:        uuid.New(),
			ScanJobID: job.ID,
			ScopeID:   scope.ID,
			FQDN:      "staging.example.com",
		},
		{
			ID:        uuid.New(),
			ScanJobID: job.ID,
			ScopeID:   scope.ID,
			FQDN:      "api.example.com",
		},
	}

	telemetryChan := make(chan domain.TelemetryEvent, 64)
	go func() {
		for range telemetryChan {
		}
	}()

	stageCtx := &modules.StageContext{
		Context:    context.Background(),
		ScanJob:    job,
		Scope:      scope,
		Runner:     runner.NewSafeRunner(5*time.Second, 128),
		Telemetry:  telemetryChan,
		DryRun:     true,
		LiveAssets: assets,
	}

	res, err := auditor.Execute(stageCtx)
	if err != nil {
		t.Fatalf("unexpected auditor error: %v", err)
	}

	if !res.Success {
		t.Fatalf("expected auditor to succeed")
	}

	if len(res.Findings) == 0 {
		t.Fatalf("expected simulated findings, got 0")
	}

	var hasCritical bool
	for _, f := range res.Findings {
		if f.Severity == domain.SeverityCritical {
			hasCritical = true
		}
		if f.RawEvidence.MatchedAt == "" {
			t.Errorf("finding %s has empty raw evidence matched_at", f.Title)
		}
	}

	if !hasCritical {
		t.Errorf("expected at least one critical finding in mock audit")
	}
}

