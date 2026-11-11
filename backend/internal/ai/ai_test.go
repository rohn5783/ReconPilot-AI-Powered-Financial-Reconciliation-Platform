package ai

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"reconpilot/internal/domain"
	"reconpilot/pkg/money"
)

func TestRuleBasedDiagnoseAllTypes(t *testing.T) {
	testCases := []domain.DiscrepancyType{
		domain.TypeFeeOvercharge,
		domain.TypeTaxCalculationError,
		domain.TypeOrderMissingInPG,
		domain.TypePGOrphanTxn,
		domain.TypeCaptureStatusMismatch,
		domain.TypeGrossAmountMismatch,
		domain.TypeBankMissing,
		domain.TypePayoutShortfall,
		domain.TypePayoutSurplus,
	}

	orderID := "ORD-TEST-01"
	payoutID := "PO-TEST-01"

	for _, dt := range testCases {
		t.Run(string(dt), func(t *testing.T) {
			d := &domain.Discrepancy{
				ID:               uuid.New(),
				OrderID:          &orderID,
				PayoutID:         &payoutID,
				DiscrepancyLevel: domain.DiscrepancyLevel1OrderPG,
				DiscrepancyType:  dt,
				ExpectedAmount:   money.FromInt(1000),
				ActualAmount:     money.FromInt(950),
				MismatchAmount:   money.FromInt(50),
				Severity:         domain.SeverityHigh,
			}

			diag := RuleBasedDiagnose(d)
			if diag == nil {
				t.Fatalf("Expected non-nil diagnosis for %s", dt)
			}
			if diag.RootCauseSummary == "" {
				t.Errorf("Expected root cause summary for %s", dt)
			}
			if diag.ConfidenceScore < 80 || diag.ConfidenceScore > 100 {
				t.Errorf("Confidence score out of range: %d", diag.ConfidenceScore)
			}
			if len(diag.RecommendedActions) == 0 {
				t.Errorf("Expected recommended actions for %s", dt)
			}
			if diag.DraftDisputeTicket == "" {
				t.Errorf("Expected draft dispute ticket for %s", dt)
			}
		})
	}
}

func TestClientOfflineFallback(t *testing.T) {
	ctx := context.Background()
	// Client without live key falls back gracefully
	client, err := NewClient(ctx, "", "gemini-2.5-flash")
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	orderID := "ORD-OFFLINE-01"
	d := &domain.Discrepancy{
		ID:               uuid.New(),
		OrderID:          &orderID,
		DiscrepancyLevel: domain.DiscrepancyLevel1OrderPG,
		DiscrepancyType:  domain.TypeFeeOvercharge,
		ExpectedAmount:   money.FromInt(10000),
		ActualAmount:     money.FromInt(9700),
		MismatchAmount:   money.FromInt(300),
		Severity:         domain.SeverityHigh,
	}

	diag, err := client.Diagnose(ctx, d)
	if err != nil {
		t.Fatalf("Diagnose failed: %v", err)
	}
	if diag == nil {
		t.Fatalf("Expected non-nil fallback diagnosis")
	}
	if diag.ConfidenceScore != 98 {
		t.Errorf("Expected 98 confidence score, got %d", diag.ConfidenceScore)
	}
}

