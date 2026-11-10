package service

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"reconpilot/internal/domain"
)

func TestDriftEngineComputeDiff(t *testing.T) {
	engine := NewDriftEngine()

	currentJobID := uuid.New()
	prevJobID := uuid.New()

	baseAssetID := uuid.New()
	curAsset1ID := baseAssetID
	curAsset2ID := uuid.New()

	now := time.Now()

	// Baseline Scan:
	// - api.example.com (ports 80, 443; tech: Nginx, PHP; IP: 1.1.1.1)
	// - legacy.example.com (ports 80; tech: Apache)
	// - Finding on api.example.com: "cve-2021-1234"
	baselineAssets := []*domain.Asset{
		{
			ID:          baseAssetID,
			ScanJobID:   prevJobID,
			FQDN:        "api.example.com",
			OpenPorts:   []int{80, 443},
			TechStack:   []string{"Nginx", "PHP"},
			IPAddresses: []string{"1.1.1.1"},
			HTTPStatus:  200,
		},
		{
			ID:          uuid.New(),
			ScanJobID:   prevJobID,
			FQDN:        "legacy.example.com",
			OpenPorts:   []int{80},
			TechStack:   []string{"Apache"},
			IPAddresses: []string{"1.1.1.2"},
			HTTPStatus:  200,
		},
	}

	baselineFindings := []*domain.Finding{
		{
			ID:        uuid.New(),
			ScanJobID: prevJobID,
			AssetID:   baseAssetID,
			FQDN:      "api.example.com",
			RuleID:    "cve-2021-1234",
			Title:     "Old Resolved Bug",
			Severity:  domain.SeverityMedium,
		},
	}

	// Current Scan:
	// - api.example.com: port 6379 opened! (Critical), port 80 closed, tech changed to Go + Nginx, IP changed to 1.1.1.99
	// - staging.example.com: NEW HOST!
	// - legacy.example.com: REMOVED!
	// - Finding "cve-2021-1234" resolved!
	// - New finding on staging: "unauth-redis" (Critical)
	currentAssets := []*domain.Asset{
		{
			ID:          curAsset1ID,
			ScanJobID:   currentJobID,
			FQDN:        "api.example.com",
			OpenPorts:   []int{443, 6379}, // 80 closed, 6379 opened
			TechStack:   []string{"Nginx", "Golang"}, // PHP removed, Golang added
			IPAddresses: []string{"1.1.1.99"},
			HTTPStatus:  200,
		},
		{
			ID:          curAsset2ID,
			ScanJobID:   currentJobID,
			FQDN:        "staging.example.com",
			OpenPorts:   []int{80, 443},
			TechStack:   []string{"Docker"},
			IPAddresses: []string{"1.1.1.50"},
			HTTPStatus:  200,
		},
	}

	currentFindings := []*domain.Finding{
		{
			ID:        uuid.New(),
			ScanJobID: currentJobID,
			AssetID:   curAsset2ID,
			FQDN:      "staging.example.com",
			RuleID:    "unauth-redis",
			Title:     "Unauthenticated Redis",
			Severity:  domain.SeverityCritical,
			CreatedAt: now,
		},
	}

	drifts, analysis := engine.ComputeDrift(
		currentJobID, &prevJobID, currentAssets, baselineAssets, currentFindings, baselineFindings,
	)

	if len(drifts) == 0 {
		t.Fatalf("expected drift records, got 0")
	}

	if analysis.TotalNewHosts != 1 {
		t.Errorf("expected 1 new host, got %d", analysis.TotalNewHosts)
	}

	if analysis.TotalRemovedHosts != 1 {
		t.Errorf("expected 1 removed host, got %d", analysis.TotalRemovedHosts)
	}

	if analysis.TotalOpenedPorts != 1 {
		t.Errorf("expected 1 opened port (6379), got %d", analysis.TotalOpenedPorts)
	}

	if analysis.TotalClosedPorts != 1 {
		t.Errorf("expected 1 closed port (80), got %d", analysis.TotalClosedPorts)
	}

	// Verify specific mutation types
	changeTypeCounts := make(map[domain.DriftChangeType]int)
	for _, d := range drifts {
		changeTypeCounts[d.ChangeType]++
	}

	if changeTypeCounts[domain.DriftNewHost] != 1 {
		t.Errorf("expected 1 DriftNewHost, got %d", changeTypeCounts[domain.DriftNewHost])
	}
	if changeTypeCounts[domain.DriftRemovedHost] != 1 {
		t.Errorf("expected 1 DriftRemovedHost, got %d", changeTypeCounts[domain.DriftRemovedHost])
	}
	if changeTypeCounts[domain.DriftPortOpened] != 1 {
		t.Errorf("expected 1 DriftPortOpened, got %d", changeTypeCounts[domain.DriftPortOpened])
	}
	if changeTypeCounts[domain.DriftPortClosed] != 1 {
		t.Errorf("expected 1 DriftPortClosed, got %d", changeTypeCounts[domain.DriftPortClosed])
	}
	if changeTypeCounts[domain.DriftTechChanged] != 1 {
		t.Errorf("expected 1 DriftTechChanged, got %d", changeTypeCounts[domain.DriftTechChanged])
	}
	if changeTypeCounts[domain.DriftIPChanged] != 1 {
		t.Errorf("expected 1 DriftIPChanged, got %d", changeTypeCounts[domain.DriftIPChanged])
	}
	if changeTypeCounts[domain.DriftFindingAdded] != 1 {
		t.Errorf("expected 1 DriftFindingAdded, got %d", changeTypeCounts[domain.DriftFindingAdded])
	}
	if changeTypeCounts[domain.DriftFindingResolved] != 1 {
		t.Errorf("expected 1 DriftFindingResolved, got %d", changeTypeCounts[domain.DriftFindingResolved])
	}

	// Verify critical mutation flag triggered for port 6379
	if len(analysis.HighRiskMutations) == 0 {
		t.Errorf("expected HighRiskMutations to capture port 6379 opening")
	}
}

