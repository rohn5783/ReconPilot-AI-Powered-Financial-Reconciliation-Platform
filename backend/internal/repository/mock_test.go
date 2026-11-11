package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"reconpilot/internal/domain"
	"reconpilot/pkg/money"
)

func TestMockRepositoryDynamicOperations(t *testing.T) {
	repo := NewMockRepository(false)
	ctx := context.Background()

	// Initial repository must be empty (Zero hardcoded data)
	count, err := repo.Orders().Count(ctx)
	if err != nil || count != 0 {
		t.Fatalf("Expected 0 initial orders, got %d (err: %v)", count, err)
	}

	// 1. Insert dynamic orders
	orders := []domain.Order{
		{
			OrderID:     "ORD-DYN-001",
			CustomerID:  "CUST-001",
			GrossAmount: money.FromInt(5000),
			Currency:    "INR",
			OrderStatus: domain.OrderStatusCaptured,
			CreatedAt:   time.Now().UTC(),
		},
		{
			OrderID:     "ORD-DYN-002",
			CustomerID:  "CUST-002",
			GrossAmount: money.FromInt(12000),
			Currency:    "INR",
			OrderStatus: domain.OrderStatusCaptured,
			CreatedAt:   time.Now().UTC(),
		},
	}
	if err := repo.Orders().InsertBatch(ctx, orders); err != nil {
		t.Fatalf("InsertBatch orders failed: %v", err)
	}

	foundOrders, err := repo.Orders().FindAll(ctx)
	if err != nil || len(foundOrders) != 2 {
		t.Fatalf("Expected 2 orders, got %d", len(foundOrders))
	}

	singleOrder, err := repo.Orders().FindByID(ctx, "ORD-DYN-001")
	if err != nil || singleOrder == nil {
		t.Fatalf("FindByID failed: %v", err)
	}

	// 2. Insert dynamic PG settlements
	pgs := []domain.PGSettlement{
		{
			PGTxnID:      "pg_dyn_001",
			OrderID:      "ORD-DYN-001",
			GrossAmount:  money.FromInt(5000),
			PGFee:        money.FromInt(100),
			Tax:          money.FromInt(18),
			NetAmount:    money.FromInt(4882),
			PayoutID:     "PO-DYN-01",
			PayoutStatus: domain.PayoutStatusSettled,
			SettledAt:    time.Now().UTC(),
		},
	}
	if err := repo.PG().InsertBatch(ctx, pgs); err != nil {
		t.Fatalf("InsertBatch pg failed: %v", err)
	}

	distinctPayouts, err := repo.PG().FindDistinctPayoutIDs(ctx)
	if err != nil || len(distinctPayouts) != 1 || distinctPayouts[0] != "PO-DYN-01" {
		t.Fatalf("FindDistinctPayoutIDs unexpected: %v", distinctPayouts)
	}

	// 3. Insert dynamic Bank statement
	banks := []domain.BankStatement{
		{
			BankTxnID:       "BNK-DYN-001",
			UTRNumber:       "UTRDYN999",
			CreditAmount:    money.FromInt(4882),
			Description:     "CMS CREDIT PO-DYN-01",
			TransactionDate: time.Now().UTC(),
		},
	}
	if err := repo.Bank().InsertBatch(ctx, banks); err != nil {
		t.Fatalf("InsertBatch bank failed: %v", err)
	}

	foundBank, err := repo.Bank().FindByUTR(ctx, "UTRDYN999")
	if err != nil || foundBank == nil {
		t.Fatalf("FindByUTR failed: %v", err)
	}

	// 4. Test Discrepancy persistence and filtering
	discID := uuid.New()
	level1 := domain.DiscrepancyLevel1OrderPG
	disc := domain.Discrepancy{
		ID:               discID,
		OrderID:          stringPtr("ORD-DYN-002"),
		DiscrepancyLevel: level1,
		DiscrepancyType:  domain.TypeOrderMissingInPG,
		ExpectedAmount:   money.FromInt(12000),
		ActualAmount:     money.Zero(),
		MismatchAmount:   money.FromInt(12000),
		Severity:         domain.SeverityHigh,
		Status:           domain.StatusOpen,
		DetectedAt:       time.Now().UTC(),
	}
	if err := repo.Discrepancies().InsertBatch(ctx, []domain.Discrepancy{disc}); err != nil {
		t.Fatalf("InsertBatch discrepancies failed: %v", err)
	}

	filtered, err := repo.Discrepancies().FindAll(ctx, DiscrepancyFilter{Level: &level1})
	if err != nil || len(filtered) != 1 {
		t.Fatalf("Expected 1 filtered discrepancy, got %d", len(filtered))
	}

	// 5. Test ResetAll clears completely
	if err := repo.ResetAll(ctx); err != nil {
		t.Fatalf("ResetAll failed: %v", err)
	}
	countAfter, _ := repo.Orders().Count(ctx)
	if countAfter != 0 {
		t.Fatalf("Expected 0 orders after ResetAll, got %d", countAfter)
	}
}

func stringPtr(s string) *string {
	return &s
}
