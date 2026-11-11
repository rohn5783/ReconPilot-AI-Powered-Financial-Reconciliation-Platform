package seed

import (
	"context"
	"testing"

	"reconpilot/internal/repository"
)

func TestBuildRealisticDataset(t *testing.T) {
	data := BuildRealisticDataset()

	if len(data.Orders) < 60 {
		t.Fatalf("Expected at least 60 orders, got %d", len(data.Orders))
	}
	if len(data.PGSettlements) < 60 {
		t.Fatalf("Expected at least 60 PG settlements, got %d", len(data.PGSettlements))
	}
	if len(data.BankStatements) < 4 {
		t.Fatalf("Expected at least 4 bank statements, got %d", len(data.BankStatements))
	}

	// Verify Payout Batches
	payouts := make(map[string]int)
	for _, pg := range data.PGSettlements {
		payouts[pg.PayoutID]++
	}

	if len(payouts) != 6 {
		t.Errorf("Expected 6 distinct payout batches, got %d", len(payouts))
	}

	// Verify all money amounts are valid and non-zero
	for _, o := range data.Orders {
		if o.GrossAmount.IsZero() {
			t.Errorf("Order %s has zero gross amount", o.OrderID)
		}
	}
}

func TestSeedRepositoryAndBootstrap(t *testing.T) {
	mockRepo := repository.NewMockRepository(false)
	ctx := context.Background()

	count, err := mockRepo.Orders().Count(ctx)
	if err != nil || count != 0 {
		t.Fatalf("Expected 0 orders before seed, got %d (err: %v)", count, err)
	}

	reconciled := false
	reconcileFn := func(c context.Context) error {
		reconciled = true
		return nil
	}

	err = BootstrapIfEmpty(ctx, mockRepo, true, reconcileFn)
	if err != nil {
		t.Fatalf("BootstrapIfEmpty failed: %v", err)
	}

	countAfter, err := mockRepo.Orders().Count(ctx)
	if err != nil || countAfter < 60 {
		t.Fatalf("Expected 60+ orders after bootstrap, got %d (err: %v)", countAfter, err)
	}

	if !reconciled {
		t.Errorf("Expected reconcileFn to be invoked during bootstrap")
	}
}
