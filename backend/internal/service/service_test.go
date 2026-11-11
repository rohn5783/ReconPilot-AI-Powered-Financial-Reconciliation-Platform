package service

import (
	"context"
	"testing"
	"time"

	"reconpilot/internal/ai"
	"reconpilot/internal/domain"
	"reconpilot/internal/matching"
	"reconpilot/internal/repository"
	"reconpilot/internal/transport/stream"
	"reconpilot/pkg/money"
)

func TestReconciliationServiceLifecycle(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewMockRepository(false)
	engine := matching.NewEngine(matching.DefaultEngineConfig())
	aiClient, err := ai.NewClient(ctx, "", "gemini-2.5-flash")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	hub := stream.NewHub()

	svc := NewReconciliationService(repo, engine, aiClient, hub)

	// Feed dynamic test records (no hardcoded pre-seeded data)
	testOrders := []domain.Order{
		{
			OrderID:     "ORD-TEST-001",
			CustomerID:  "CUST-001",
			GrossAmount: money.FromInt(10000),
			Currency:    "INR",
			OrderStatus: domain.OrderStatusCaptured,
			CreatedAt:   time.Now().UTC(),
		},
		{
			OrderID:     "ORD-TEST-002",
			CustomerID:  "CUST-002",
			GrossAmount: money.FromInt(20000),
			Currency:    "INR",
			OrderStatus: domain.OrderStatusCaptured,
			CreatedAt:   time.Now().UTC(),
		},
	}
	testPG := []domain.PGSettlement{
		{
			PGTxnID:      "pay_test_001",
			OrderID:      "ORD-TEST-001",
			GrossAmount:  money.FromInt(10000),
			PGFee:        money.FromInt(200),
			Tax:          money.FromStringOrZero("36.0000"),
			NetAmount:    money.FromStringOrZero("9764.0000"),
			PayoutID:     "PO-TEST-01",
			PayoutStatus: domain.PayoutStatusSettled,
			SettledAt:    time.Now().UTC(),
		},
		{
			// Discrepancy: fee overcharge on ORD-TEST-002
			PGTxnID:      "pay_test_002",
			OrderID:      "ORD-TEST-002",
			GrossAmount:  money.FromInt(20000),
			PGFee:        money.FromInt(600), // higher fee
			Tax:          money.FromStringOrZero("108.0000"),
			NetAmount:    money.FromStringOrZero("19292.0000"),
			PayoutID:     "PO-TEST-01",
			PayoutStatus: domain.PayoutStatusSettled,
			SettledAt:    time.Now().UTC(),
		},
	}
	testBank := []domain.BankStatement{
		{
			BankTxnID:       "BNK-TEST-001",
			UTRNumber:       "UTRTEST001",
			CreditAmount:    money.FromStringOrZero("29056.0000"),
			Description:     "CMS CREDIT PO-TEST-01",
			TransactionDate: time.Now().UTC(),
		},
	}

	_, err = svc.RunReconciliation(ctx, testOrders, testPG, testBank)
	if err != nil {
		t.Fatalf("RunReconciliation failed: %v", err)
	}

	// Summary check
	summary, err := svc.GetSummary(ctx)
	if err != nil {
		t.Fatalf("GetSummary failed: %v", err)
	}
	if summary.TotalOrders != 2 {
		t.Errorf("Expected 2 orders in summary, got %d", summary.TotalOrders)
	}

	// Traces check
	traces, total, err := svc.GetTraces(ctx, "", "", 10, 0)
	if err != nil {
		t.Fatalf("GetTraces failed: %v", err)
	}
	if total != 2 || len(traces) != 2 {
		t.Errorf("Expected 2 traces, got %d", total)
	}

	// Search filter
	filteredTraces, _, err := svc.GetTraces(ctx, "ORD-TEST-001", "", 10, 0)
	if err != nil {
		t.Fatalf("GetTraces with search failed: %v", err)
	}
	if len(filteredTraces) == 0 || filteredTraces[0].OrderID != "ORD-TEST-001" {
		t.Errorf("Expected to find ORD-TEST-001, got %+v", filteredTraces)
	}

	// Payout batches check
	batches, err := svc.GetPayoutBatches(ctx)
	if err != nil {
		t.Fatalf("GetPayoutBatches failed: %v", err)
	}
	if len(batches) != 1 {
		t.Errorf("Expected 1 payout batch group, got %d", len(batches))
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
