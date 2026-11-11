package matching

import (
	"context"
	"testing"
	"time"

	"reconpilot/internal/domain"
	"reconpilot/pkg/money"
)

func TestEnginePerfectThreeWayMatch(t *testing.T) {
	engine := NewEngine(DefaultEngineConfig())
	ctx := context.Background()
	now := time.Now().UTC()

	// 1 Order, 1 PG Settlement (2.0% fee + 18% GST), 1 Bank Statement
	gross := money.FromInt(10000)
	fee, tax, net := money.CalculateMDRAndGST(gross, money.FromFloat(0.02), money.FromFloat(0.18))

	orders := []domain.Order{
		{
			OrderID:     "ORD-PERFECT-01",
			CustomerID:  "CUST-01",
			GrossAmount: gross,
			Currency:    "INR",
			OrderStatus: domain.OrderStatusCaptured,
			CreatedAt:   now,
		},
	}

	settlements := []domain.PGSettlement{
		{
			PGTxnID:      "pay_rzp_01",
			OrderID:      "ORD-PERFECT-01",
			GrossAmount:  gross,
			PGFee:        fee,
			Tax:          tax,
			NetAmount:    net,
			PayoutID:     "PO-BATCH-01",
			PayoutStatus: domain.PayoutStatusSettled,
			SettledAt:    now.Add(24 * time.Hour),
		},
	}

	bankStatements := []domain.BankStatement{
		{
			BankTxnID:       "BNK-HDFC-01",
			UTRNumber:       "HDFCR52026110190123",
			CreditAmount:    net,
			Description:     "CMS/RAZORPAY/PO-BATCH-01/HDFCR52026110190123",
			TransactionDate: now.Add(26 * time.Hour),
		},
	}

	var progressEvents []domain.PipelineTelemetryEvent
	res, err := engine.Reconcile(ctx, orders, settlements, bankStatements, func(evt domain.PipelineTelemetryEvent) {
		progressEvents = append(progressEvents, evt)
	})
	if err != nil {
		t.Fatalf("Reconcile returned unexpected error: %v", err)
	}

	if len(res.Discrepancies) != 0 {
		t.Errorf("Expected 0 discrepancies for perfect match, got %d", len(res.Discrepancies))
	}
	if len(res.TraceItems) != 1 {
		t.Fatalf("Expected 1 trace item, got %d", len(res.TraceItems))
	}
	if !res.TraceItems[0].IsReconciled {
		t.Errorf("Expected trace item to be marked reconciled")
	}
	if !res.Summary.ThreeWayMatchRate.Equal(money.FromInt(100)) {
		t.Errorf("Expected 100%% match rate, got %s", res.Summary.ThreeWayMatchRate.StringFixed(2))
	}
	if len(progressEvents) < 3 {
		t.Errorf("Expected progress events to be emitted, got %d", len(progressEvents))
	}
}

func TestEngineLevel1Discrepancies(t *testing.T) {
	engine := NewEngine(DefaultEngineConfig())
	ctx := context.Background()
	now := time.Now().UTC()

	orders := []domain.Order{
		// 1. Fee overcharge
		{
			OrderID:     "ORD-FEE-01",
			GrossAmount: money.FromInt(50000),
			Currency:    "INR",
			OrderStatus: domain.OrderStatusCaptured,
			CreatedAt:   now,
		},
		// 2. Order Missing in PG
		{
			OrderID:     "ORD-MISSING-01",
			GrossAmount: money.FromInt(20000),
			Currency:    "INR",
			OrderStatus: domain.OrderStatusCaptured,
			CreatedAt:   now,
		},
		// 3. Status mismatch (Order Refunded, PG Settled)
		{
			OrderID:     "ORD-STATUS-01",
			GrossAmount: money.FromInt(10000),
			Currency:    "INR",
			OrderStatus: domain.OrderStatusRefunded,
			CreatedAt:   now,
		},
		// 4. Gross amount mismatch
		{
			OrderID:     "ORD-AMOUNT-01",
			GrossAmount: money.FromStringOrZero("35000.0000"),
			Currency:    "INR",
			OrderStatus: domain.OrderStatusCaptured,
			CreatedAt:   now,
		},
	}

	settlements := []domain.PGSettlement{
		// Fee overcharge: contract rate is 2% (1000 fee), but PG charged 3% (1500 fee)
		{
			PGTxnID:      "pay_rzp_fee",
			OrderID:      "ORD-FEE-01",
			GrossAmount:  money.FromInt(50000),
			PGFee:        money.FromInt(1500),
			Tax:          money.FromInt(270),
			NetAmount:    money.FromInt(48230),
			PayoutID:     "PO-01",
			PayoutStatus: domain.PayoutStatusSettled,
			SettledAt:    now,
		},
		// ORD-MISSING-01 is deliberately omitted from settlements
		// Status mismatch
		{
			PGTxnID:      "pay_rzp_status",
			OrderID:      "ORD-STATUS-01",
			GrossAmount:  money.FromInt(10000),
			PGFee:        money.FromInt(200),
			Tax:          money.FromInt(36),
			NetAmount:    money.FromInt(9764),
			PayoutID:     "PO-01",
			PayoutStatus: domain.PayoutStatusSettled,
			SettledAt:    now,
		},
		// Gross amount mismatch: PG recorded 30,000 instead of 35,000
		{
			PGTxnID:      "pay_rzp_amount",
			OrderID:      "ORD-AMOUNT-01",
			GrossAmount:  money.FromStringOrZero("30000.0000"),
			PGFee:        money.FromInt(600),
			Tax:          money.FromInt(108),
			NetAmount:    money.FromInt(29292),
			PayoutID:     "PO-01",
			PayoutStatus: domain.PayoutStatusSettled,
			SettledAt:    now,
		},
	}

	// Supply matching bank statement for PO-01 (48230 + 9764 + 29292 = 87286) so Level 2 matches
	bankStatements := []domain.BankStatement{
		{
			BankTxnID:       "BNK-PO-01",
			UTRNumber:       "HDFCR52026110287286",
			CreditAmount:    money.FromStringOrZero("87286.0000"),
			Description:     "CMS/RAZORPAY/PO-01",
			TransactionDate: now.Add(24 * time.Hour),
		},
	}

	res, err := engine.Reconcile(ctx, orders, settlements, bankStatements, nil)
	if err != nil {
		t.Fatalf("Reconcile returned unexpected error: %v", err)
	}

	if len(res.Discrepancies) != 4 {
		t.Fatalf("Expected 4 discrepancies, got %d", len(res.Discrepancies))
	}

	foundTypes := make(map[domain.DiscrepancyType]bool)
	for _, d := range res.Discrepancies {
		foundTypes[d.DiscrepancyType] = true
	}

	expectedTypes := []domain.DiscrepancyType{
		domain.TypeFeeOvercharge,
		domain.TypeOrderMissingInPG,
		domain.TypeCaptureStatusMismatch,
		domain.TypeGrossAmountMismatch,
	}

	for _, exp := range expectedTypes {
		if !foundTypes[exp] {
			t.Errorf("Expected discrepancy type %s not found in detected discrepancies", exp)
		}
	}
}

func TestEngineLevel2Discrepancies(t *testing.T) {
	engine := NewEngine(DefaultEngineConfig())
	ctx := context.Background()
	now := time.Now().UTC()

	orders := []domain.Order{
		{
			OrderID:     "ORD-L2-01",
			GrossAmount: money.FromInt(100000),
			Currency:    "INR",
			OrderStatus: domain.OrderStatusCaptured,
			CreatedAt:   now,
		},
		{
			OrderID:     "ORD-L2-02",
			GrossAmount: money.FromInt(50000),
			Currency:    "INR",
			OrderStatus: domain.OrderStatusCaptured,
			CreatedAt:   now,
		},
	}

	settlements := []domain.PGSettlement{
		// Payout 1: Net 97,640.0000 -> Bank missing
		{
			PGTxnID:      "pay_rzp_l2_01",
			OrderID:      "ORD-L2-01",
			GrossAmount:  money.FromInt(100000),
			PGFee:        money.FromInt(2000),
			Tax:          money.FromInt(360),
			NetAmount:    money.FromStringOrZero("97640.0000"),
			PayoutID:     "PO-BATCH-MISSING",
			PayoutStatus: domain.PayoutStatusSettled,
			SettledAt:    now,
		},
		// Payout 2: Net 48,820.0000 -> Bank received 45,000.0000 (Shortfall)
		{
			PGTxnID:      "pay_rzp_l2_02",
			OrderID:      "ORD-L2-02",
			GrossAmount:  money.FromInt(50000),
			PGFee:        money.FromInt(1000),
			Tax:          money.FromInt(180),
			NetAmount:    money.FromStringOrZero("48820.0000"),
			PayoutID:     "PO-BATCH-SHORTFALL",
			PayoutStatus: domain.PayoutStatusSettled,
			SettledAt:    now,
		},
	}

	bankStatements := []domain.BankStatement{
		{
			BankTxnID:       "BNK-SHORTFALL",
			UTRNumber:       "ICICR52026110299999",
			CreditAmount:    money.FromStringOrZero("45000.0000"), // 3820 shortfall
			Description:     "CMS/RAZORPAY/PO-BATCH-SHORTFALL",
			TransactionDate: now.Add(24 * time.Hour),
		},
		// PO-BATCH-MISSING has NO corresponding bank statement
	}

	res, err := engine.Reconcile(ctx, orders, settlements, bankStatements, nil)
	if err != nil {
		t.Fatalf("Reconcile returned unexpected error: %v", err)
	}

	foundBankMissing := false
	foundShortfall := false

	for _, d := range res.Discrepancies {
		if d.DiscrepancyType == domain.TypeBankMissing && *d.PayoutID == "PO-BATCH-MISSING" {
			foundBankMissing = true
			if !d.MismatchAmount.Equal(money.FromStringOrZero("97640.0000")) {
				t.Errorf("Expected BankMissing mismatch amount 97640.0000, got %s", d.MismatchAmount.StringFixed(4))
			}
		}
		if d.DiscrepancyType == domain.TypePayoutShortfall && *d.PayoutID == "PO-BATCH-SHORTFALL" {
			foundShortfall = true
			expectedShortfall := money.FromStringOrZero("3820.0000")
			if !d.MismatchAmount.Equal(expectedShortfall) {
				t.Errorf("Expected Shortfall mismatch amount %s, got %s", expectedShortfall.StringFixed(4), d.MismatchAmount.StringFixed(4))
			}
		}
	}

	if !foundBankMissing {
		t.Errorf("Expected TypeBankMissing discrepancy for PO-BATCH-MISSING")
	}
	if !foundShortfall {
		t.Errorf("Expected TypePayoutShortfall discrepancy for PO-BATCH-SHORTFALL")
	}
}
