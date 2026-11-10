package prober

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"reconpilot/internal/domain"
	"reconpilot/internal/modules"
	"reconpilot/pkg/runner"
)

func TestProberModuleMockMode(t *testing.T) {
	prob := NewProberModule("httpx")

	scope := &domain.Scope{
		ID:         uuid.New(),
		RootDomain: "example.com",
	}

	job := &domain.ScanJob{
		ID:      uuid.New(),
		ScopeID: scope.ID,
		DryRun:  true,
	}

	hosts := []string{
		"example.com",
		"api.example.com",
		"auth.example.com",
		"admin.example.com",
	}

	telemetryChan := make(chan domain.TelemetryEvent, 64)
	go func() {
		for range telemetryChan {
		}
	}()

	stageCtx := &modules.StageContext{
		Context:         context.Background(),
		ScanJob:         job,
		Scope:           scope,
		Runner:          runner.NewSafeRunner(5*time.Second, 128),
		Telemetry:       telemetryChan,
		DryRun:          true,
		DiscoveredHosts: hosts,
	}

	res, err := prob.Execute(stageCtx)
	if err != nil {
		t.Fatalf("unexpected prober error in dry run: %v", err)
	}

	if !res.Success {
		t.Fatalf("expected prober to succeed")
	}

	if len(res.Assets) != len(hosts) {
		t.Fatalf("expected %d assets, got %d", len(hosts), len(res.Assets))
	}

	for _, a := range res.Assets {
		if a.HTTPStatus == 0 {
			t.Errorf("asset %s has unpopulated HTTP status", a.FQDN)
		}
		if len(a.TechStack) == 0 {
			t.Errorf("asset %s has empty tech stack", a.FQDN)
		}
	}
}

