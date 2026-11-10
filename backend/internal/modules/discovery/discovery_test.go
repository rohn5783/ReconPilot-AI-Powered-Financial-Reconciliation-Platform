package discovery

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"reconpilot/internal/domain"
	"reconpilot/internal/modules"
	"reconpilot/pkg/runner"
)

func TestDiscoveryModuleMockMode(t *testing.T) {
	disc := NewDiscoveryModule("subfinder")

	scope := &domain.Scope{
		ID:         uuid.New(),
		RootDomain: "example.com",
		ExcludedSubdomains: []string{
			"mail.example.com",
		},
	}

	job := &domain.ScanJob{
		ID:      uuid.New(),
		ScopeID: scope.ID,
		DryRun:  true,
	}

	telemetryChan := make(chan domain.TelemetryEvent, 64)
	go func() {
		for range telemetryChan {
		}
	}()

	stageCtx := &modules.StageContext{
		Context:   context.Background(),
		ScanJob:   job,
		Scope:     scope,
		Runner:    runner.NewSafeRunner(5*time.Second, 128),
		Telemetry: telemetryChan,
		DryRun:    true,
	}

	res, err := disc.Execute(stageCtx)
	if err != nil {
		t.Fatalf("unexpected discovery error in dry run: %v", err)
	}

	if !res.Success {
		t.Fatalf("expected discovery stage to succeed")
	}

	if len(res.DiscoveredHosts) == 0 {
		t.Fatalf("expected discovered hosts to be non-empty")
	}

	// Verify excluded subdomain was filtered out
	for _, h := range res.DiscoveredHosts {
		if h == "mail.example.com" {
			t.Errorf("expected mail.example.com to be excluded by scope guard")
		}
	}
}

