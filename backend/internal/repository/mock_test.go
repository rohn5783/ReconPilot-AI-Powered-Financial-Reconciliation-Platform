package repository

import (
	"context"
	"testing"
	"time"

	"reconpilot/internal/domain"
	"reconpilot/pkg/money"
)

func TestMockRepositorySeeding(t *testing.T) {
	repo := NewMockRepository(true)
	ctx := context.Background()

	// Check orders
	orders, err := repo.Orders().FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll orders failed: %v", err)
	}
	if len(orders) < 5 {
		t.Errorf("Expected at least 5 seeded orders, got %d", len(orders))
	}

	// Check PG settlements
	pgs, err := repo.PG().FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll PG settlements failed: %v", err)
	}
	if len(pgs) < 5 {
		t.Errorf("Expected at least 5 seeded PG settlements, got %d", len(pgs))
	}

	// Check Bank statements
	banks, err := repo.Bank().FindAll(ctx)
	if err != nil {
		t.Fatalf("FindAll bank statements failed: %v", err)
	}
	if len(banks) < 2 {
		t.Errorf("Expected at least 2 bank statement records, got %d", len(banks))
	}

	// Check Discrepancies
	discs, err := repo.Discrepancies().FindAll(ctx, DiscrepancyFilter{})
	if err != nil {
		t.Fatalf("FindAll discrepancies failed: %v", err)
	}
	if len(discs) < 5 {
		t.Errorf("Expected at least 5 discrepancies seeded, got %d", len(discs))
	}
}

func TestMockRepositoryDiscrepancyFiltering(t *testing.T) {
	repo := NewMockRepository(true)
	ctx := context.Background()

	// Filter by Level 1
	l1 := domain.DiscrepancyLevel1OrderPG
	l1Discs, err := repo.Discrepancies().FindAll(ctx, DiscrepancyFilter{Level: &l1})
	if err != nil {
		t.Fatalf("FindAll level 1 failed: %v", err)
	}
	for _, d := range l1Discs {
		if d.DiscrepancyLevel != domain.DiscrepancyLevel1OrderPG {
			t.Errorf("Expected level 1, got %s", d.DiscrepancyLevel)
		}
	}

	// Filter by Severity Critical
	sevCrit := domain.SeverityCritical
	critDiscs, err := repo.Discrepancies().FindAll(ctx, DiscrepancyFilter{Severity: &sevCrit})
	if err != nil {
		t.Fatalf("FindAll critical failed: %v", err)
	}
	for _, d := range critDiscs {
		if d.Severity != domain.SeverityCritical {
			t.Errorf("Expected critical, got %s", d.Severity)
		}
	}

	// Update Status
	if len(critDiscs) > 0 {
		target := critDiscs[0]
		err := repo.Discrepancies().UpdateStatus(ctx, target.ID, domain.StatusInDispute)
		if err != nil {
			t.Fatalf("UpdateStatus failed: %v", err)
		}

		updated, err := repo.Discrepancies().FindByID(ctx, target.ID)
		if err != nil || updated == nil {
			t.Fatalf("FindByID failed: %v", err)
		}
		if updated.Status != domain.StatusInDispute {
			t.Errorf("Expected IN_DISPUTE status, got %s", updated.Status)
		}
	}
}

func TestMockRepositoryInsertBatchAndReset(t *testing.T) {
	repo := NewMockRepository(false)
	ctx := context.Background()

	// Insert custom order
	o := domain.Order{
		OrderID:     "TEST-ORD-01",
		CustomerID:  "CUST-01",
		GrossAmount: money.FromInt(5000),
		Currency:    "INR",
		OrderStatus: domain.OrderStatusCaptured,
		CreatedAt:   time.Now(),
	}
	if err := repo.Orders().InsertBatch(ctx, []domain.Order{o}); err != nil {
		t.Fatalf("InsertBatch orders failed: %v", err)
	}

	found, err := repo.Orders().FindByID(ctx, "TEST-ORD-01")
	if err != nil || found == nil {
		t.Fatalf("FindByID failed: %v", err)
	}
	if !found.GrossAmount.Equal(o.GrossAmount) {
		t.Errorf("Expected gross amount %s, got %s", o.GrossAmount.StringFixed(4), found.GrossAmount.StringFixed(4))
	}

	// Reset
	if err := repo.ResetAll(ctx); err != nil {
		t.Fatalf("ResetAll failed: %v", err)
	}

	count, _ := repo.Orders().Count(ctx)
	if count != 0 {
		t.Errorf("Expected 0 orders after reset, got %d", count)
	}
}
