package service

import (
	"context"
	"testing"

	"reconpilot/internal/ai"
	"reconpilot/internal/domain"
	"reconpilot/internal/matching"
	"reconpilot/internal/repository"
	"reconpilot/internal/transport/stream"
	"reconpilot/pkg/money"
)

func TestReconciliationServiceLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMockRepository(true)
	engine := matching.NewEngine(matching.DefaultEngineConfig())
	aiClient, err := ai.NewClient(ctx, "", "gemini-2.5-flash")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	hub := stream.NewHub()

	svc := NewReconciliationService(repo, engine, aiClient, hub)

	// Summary check
	summary, err := svc.GetSummary(ctx)
	if err != nil {
		t.Fatalf("GetSummary failed: %v", err)
	}
	if summary.TotalOrders < 5 {
		t.Errorf("Expected at least 5 orders in summary, got %d", summary.TotalOrders)
	}

	// Traces check
	traces, total, err := svc.GetTraces(ctx, "", "", 10, 0)
	if err != nil {
		t.Fatalf("GetTraces failed: %v", err)
	}
	if total < 5 || len(traces) == 0 {
		t.Errorf("Expected traces, got %d", total)
	}

	// Search filter
	filteredTraces, _, err := svc.GetTraces(ctx, "ORD-IN-1006", "", 10, 0)
	if err != nil {
		t.Fatalf("GetTraces with search failed: %v", err)
	}
	if len(filteredTraces) == 0 || filteredTraces[0].OrderID != "ORD-IN-1006" {
		t.Errorf("Expected to find ORD-IN-1006, got %+v", filteredTraces)
	}

	// Payout batches check
	batches, err := svc.GetPayoutBatches(ctx)
	if err != nil {
		t.Fatalf("GetPayoutBatches failed: %v", err)
	}
	if len(batches) == 0 {
		t.Errorf("Expected at least 1 payout batch group, got 0")
	}

	// Discrepancies check
	discs, err := svc.GetDiscrepancies(ctx, repository.DiscrepancyFilter{})
	if err != nil {
		t.Fatalf("GetDiscrepancies failed: %v", err)
	}
	if len(discs) == 0 {
		t.Errorf("Expected detected discrepancies, got 0")
	}

	// Update status
	firstDisc := discs[0]
	err = svc.UpdateDiscrepancyStatus(ctx, firstDisc.ID, domain.StatusInDispute)
	if err != nil {
		t.Fatalf("UpdateDiscrepancyStatus failed: %v", err)
	}

	updated, err := svc.GetDiscrepancyByID(ctx, firstDisc.ID)
	if err != nil || updated == nil {
		t.Fatalf("GetDiscrepancyByID failed: %v", err)
	}
	if updated.Status != domain.StatusInDispute {
		t.Errorf("Expected status IN_DISPUTE, got %s", updated.Status)
	}

	// Diagnose discrepancy
	diag, err := svc.DiagnoseDiscrepancy(ctx, firstDisc.ID)
	if err != nil {
		t.Fatalf("DiagnoseDiscrepancy failed: %v", err)
	}
	if diag == nil || diag.RootCauseSummary == "" {
		t.Errorf("Expected non-empty diagnosis, got %+v", diag)
	}
}

func TestExportService(t *testing.T) {
	exportSvc := NewExportService()

	orderGross := money.FromInt(10000)
	orderStatus := domain.OrderStatusCaptured
	traces := []domain.ThreeWayTraceItem{
		{
			OrderID:      "ORD-EXP-01",
			OrderGross:   &orderGross,
			OrderStatus:  &orderStatus,
			IsReconciled: true,
		},
	}

	csvData, err := exportSvc.ExportTracesCSV(traces)
	if err != nil {
		t.Fatalf("ExportTracesCSV failed: %v", err)
	}
	if len(csvData) == 0 {
		t.Errorf("Expected non-empty CSV output")
	}

	jsonData, err := exportSvc.ExportJSON(traces)
	if err != nil {
		t.Fatalf("ExportJSON failed: %v", err)
	}
	if len(jsonData) == 0 {
		t.Errorf("Expected non-empty JSON output")
	}
}

