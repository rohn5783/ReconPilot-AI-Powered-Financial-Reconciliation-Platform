package portscan

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"reconpilot/internal/domain"
	"reconpilot/internal/modules"
	"reconpilot/pkg/runner"
)

func TestPortScanModuleMockMode(t *testing.T) {
	scanner := NewPortScanModule("nmap")

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
			FQDN:      "api.example.com",
		},
		{
			ID:        uuid.New(),
			ScanJobID: job.ID,
			ScopeID:   scope.ID,
			FQDN:      "staging.example.com",
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

	res, err := scanner.Execute(stageCtx)
	if err != nil {
		t.Fatalf("unexpected portscan error: %v", err)
	}

	if !res.Success {
		t.Fatalf("expected portscan to succeed")
	}

	// Verify ports were attached
	foundRedis := false
	for _, a := range res.Assets {
		if len(a.OpenPorts) == 0 {
			t.Errorf("asset %s has no open ports assigned", a.FQDN)
		}
		if a.FQDN == "staging.example.com" {
			for _, p := range a.OpenPorts {
				if p == 6379 {
					foundRedis = true
				}
			}
		}
	}

	if !foundRedis {
		t.Errorf("expected staging.example.com to have port 6379 open in mock simulation")
	}
}

