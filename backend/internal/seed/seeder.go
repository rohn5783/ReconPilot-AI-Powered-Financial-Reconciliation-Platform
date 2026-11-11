package seed

import (
	"context"
	"fmt"
	"time"

	"reconpilot/internal/domain"
	"reconpilot/internal/repository"
	"reconpilot/pkg/money"
)

// Dataset encapsulates orders, PG settlements, and bank statements for bootstrapping.
type Dataset struct {
	Orders         []domain.Order
	PGSettlements  []domain.PGSettlement
	BankStatements []domain.BankStatement
}

// BuildRealisticDataset constructs 65+ multi-currency financial records across 6 Payout Batches
// covering every production discrepancy scenario.
func BuildRealisticDataset() Dataset {
	baseTime := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	var orders []domain.Order
	var pgSettlements []domain.PGSettlement
	var bankStatements []domain.BankStatement

	// -------------------------------------------------------------------------
	// BATCH 1: Perfect 3-Way Match (INR / Razorpay / HDFC Bank)
	// Payout ID: PO-IN-2026-001 | UTR: HDFCR52026110290101
	// 10 orders, 2.0% MDR + 18% GST, exact match at Level 1 & Level 2
	// -------------------------------------------------------------------------
	payout1ID := "PO-IN-2026-001"
	utr1 := "HDFCR52026110290101"
	var p1NetTotal money.Decimal

	for i := 1; i <= 10; i++ {
		orderID := fmt.Sprintf("ORD-IN-10%02d", i)
		gross := money.FromInt(int64(10000 + i*3500))
		fee, tax, net := money.CalculateMDRAndGST(gross, money.FromFloat(0.02), money.FromFloat(0.18))
		p1NetTotal = p1NetTotal.Add(net)

		orders = append(orders, domain.Order{
			OrderID:     orderID,
			CustomerID:  fmt.Sprintf("CUST-IN-90%02d", i),
			GrossAmount: gross,
			Currency:    "INR",
			OrderStatus: domain.OrderStatusCaptured,
			CreatedAt:   baseTime.Add(time.Duration(i*30) * time.Minute),
		})

		pgSettlements = append(pgSettlements, domain.PGSettlement{
			PGTxnID:      fmt.Sprintf("pay_rzp_10%02d", i),
			OrderID:      orderID,
			GrossAmount:  gross,
			PGFee:        fee,
			Tax:          tax,
			NetAmount:    net,
			PayoutID:     payout1ID,
			PayoutStatus: domain.PayoutStatusSettled,
			SettledAt:    baseTime.Add(time.Duration(i*30+15) * time.Minute),
		})
	}

	bankStatements = append(bankStatements, domain.BankStatement{
		BankTxnID:       "BNK-HDFC-20261102-001",
		UTRNumber:       utr1,
		CreditAmount:    p1NetTotal,
		Description:     fmt.Sprintf("CMS CR-RAZORPAY NODAL-%s-MERCHANT CORP", payout1ID),
		TransactionDate: baseTime.Add(24 * time.Hour),
	})

	// -------------------------------------------------------------------------
	// BATCH 2: Fee Overcharge & Tax Calculation Error (INR / ICICI Bank)
	// Payout ID: PO-IN-2026-002 | UTR: ICICR52026110390202
	// 12 orders: 2 fee overcharges (2.85% MDR), 2 tax calculation errors (12% GST)
	// -------------------------------------------------------------------------
	payout2ID := "PO-IN-2026-002"
	utr2 := "ICICR52026110390202"
	var p2NetTotal money.Decimal

	for i := 1; i <= 12; i++ {
		orderID := fmt.Sprintf("ORD-IN-20%02d", i)
		gross := money.FromInt(int64(25000 + i*4000))

		var fee, tax, net money.Decimal
		if i == 1 || i == 2 {
			// Scenario: FEE_OVERCHARGE (Contract 2.0%, Gateway charged 2.85%)
			fee = money.Round4(gross.Mul(money.FromFloat(0.0285)))
			tax = money.Round4(fee.Mul(money.FromFloat(0.18)))
			net = gross.Sub(fee).Sub(tax)
		} else if i == 3 || i == 4 {
			// Scenario: TAX_CALCULATION_ERROR (Contracted GST 18%, Gateway charged 12%)
			fee = money.Round4(gross.Mul(money.FromFloat(0.02)))
			tax = money.Round4(fee.Mul(money.FromFloat(0.12))) // 12% instead of 18%
			net = gross.Sub(fee).Sub(tax)
		} else {
			// Clean orders
			fee, tax, net = money.CalculateMDRAndGST(gross, money.FromFloat(0.02), money.FromFloat(0.18))
		}
		p2NetTotal = p2NetTotal.Add(net)

		orders = append(orders, domain.Order{
			OrderID:     orderID,
			CustomerID:  fmt.Sprintf("CUST-IN-80%02d", i),
			GrossAmount: gross,
			Currency:    "INR",
			OrderStatus: domain.OrderStatusCaptured,
			CreatedAt:   baseTime.Add(time.Duration(24+i) * time.Hour),
		})

		pgSettlements = append(pgSettlements, domain.PGSettlement{
			PGTxnID:      fmt.Sprintf("pay_rzp_20%02d", i),
			OrderID:      orderID,
			GrossAmount:  gross,
			PGFee:        fee,
			Tax:          tax,
			NetAmount:    net,
			PayoutID:     payout2ID,
			PayoutStatus: domain.PayoutStatusSettled,
			SettledAt:    baseTime.Add(time.Duration(25+i) * time.Hour),
		})
	}

	bankStatements = append(bankStatements, domain.BankStatement{
		BankTxnID:       "BNK-ICICI-20261103-002",
		UTRNumber:       utr2,
		CreditAmount:    p2NetTotal,
		Description:     fmt.Sprintf("NEFT CR-ICIC0000001-RAZORPAY-%s", payout2ID),
		TransactionDate: baseTime.Add(48 * time.Hour),
	})

	// -------------------------------------------------------------------------
	// BATCH 3: Order Missing in PG, PG Orphan, Status Mismatch (USD / Stripe / JPMC)
	// Payout ID: PO-US-2026-003 | UTR: JPMCUS3320261104803
	// 12 transactions: 2 missing in PG, 2 orphans, 2 status mismatches, 6 clean
	// -------------------------------------------------------------------------
	payout3ID := "PO-US-2026-003"
	utr3 := "JPMCUS3320261104803"
	var p3NetTotal money.Decimal

	// 2 Missing in PG: Orders exist internally as CAPTURED, but no PG record
	orders = append(orders, domain.Order{
		OrderID:     "ORD-US-3001",
		CustomerID:  "CUST-US-101",
		GrossAmount: money.FromStringOrZero("2400.0000"),
		Currency:    "USD",
		OrderStatus: domain.OrderStatusCaptured,
		CreatedAt:   baseTime.Add(50 * time.Hour),
	})
	orders = append(orders, domain.Order{
		OrderID:     "ORD-US-3002",
		CustomerID:  "CUST-US-102",
		GrossAmount: money.FromStringOrZero("4850.0000"),
		Currency:    "USD",
		OrderStatus: domain.OrderStatusCaptured,
		CreatedAt:   baseTime.Add(51 * time.Hour),
	})

	// 2 Orphan PG Transactions: Settlements exist at gateway, but no order in ERP
	orphanGross1 := money.FromStringOrZero("1500.0000")
	orphanFee1 := money.Round4(orphanGross1.Mul(money.FromFloat(0.029)))
	orphanNet1 := orphanGross1.Sub(orphanFee1)
	p3NetTotal = p3NetTotal.Add(orphanNet1)
	pgSettlements = append(pgSettlements, domain.PGSettlement{
		PGTxnID:      "ch_stripe_orphan_3003",
		OrderID:      "ORD-GHOST-3003",
		GrossAmount:  orphanGross1,
		PGFee:        orphanFee1,
		Tax:          money.Zero(),
		NetAmount:    orphanNet1,
		PayoutID:     payout3ID,
		PayoutStatus: domain.PayoutStatusSettled,
		SettledAt:    baseTime.Add(52 * time.Hour),
	})

	orphanGross2 := money.FromStringOrZero("3200.0000")
	orphanFee2 := money.Round4(orphanGross2.Mul(money.FromFloat(0.029)))
	orphanNet2 := orphanGross2.Sub(orphanFee2)
	p3NetTotal = p3NetTotal.Add(orphanNet2)
	pgSettlements = append(pgSettlements, domain.PGSettlement{
		PGTxnID:      "ch_stripe_orphan_3004",
		OrderID:      "ORD-GHOST-3004",
		GrossAmount:  orphanGross2,
		PGFee:        orphanFee2,
		Tax:          money.Zero(),
		NetAmount:    orphanNet2,
		PayoutID:     payout3ID,
		PayoutStatus: domain.PayoutStatusSettled,
		SettledAt:    baseTime.Add(53 * time.Hour),
	})

	// 2 Status Mismatch: Order is FAILED or REFUNDED in ERP, but SETTLED in PG
	mismatchGross1 := money.FromStringOrZero("890.0000")
	mismatchFee1 := money.Round4(mismatchGross1.Mul(money.FromFloat(0.029)))
	mismatchNet1 := mismatchGross1.Sub(mismatchFee1)
	p3NetTotal = p3NetTotal.Add(mismatchNet1)
	orders = append(orders, domain.Order{
		OrderID:     "ORD-US-3005",
		CustomerID:  "CUST-US-105",
		GrossAmount: mismatchGross1,
		Currency:    "USD",
		OrderStatus: domain.OrderStatusFailed,
		CreatedAt:   baseTime.Add(54 * time.Hour),
	})
	pgSettlements = append(pgSettlements, domain.PGSettlement{
		PGTxnID:      "ch_stripe_mismatch_3005",
		OrderID:      "ORD-US-3005",
		GrossAmount:  mismatchGross1,
		PGFee:        mismatchFee1,
		Tax:          money.Zero(),
		NetAmount:    mismatchNet1,
		PayoutID:     payout3ID,
		PayoutStatus: domain.PayoutStatusSettled,
		SettledAt:    baseTime.Add(55 * time.Hour),
	})

	mismatchGross2 := money.FromStringOrZero("1240.0000")
	mismatchFee2 := money.Round4(mismatchGross2.Mul(money.FromFloat(0.029)))
	mismatchNet2 := mismatchGross2.Sub(mismatchFee2)
	p3NetTotal = p3NetTotal.Add(mismatchNet2)
	orders = append(orders, domain.Order{
		OrderID:     "ORD-US-3006",
		CustomerID:  "CUST-US-106",
		GrossAmount: mismatchGross2,
		Currency:    "USD",
		OrderStatus: domain.OrderStatusRefunded,
		CreatedAt:   baseTime.Add(55 * time.Hour),
	})
	pgSettlements = append(pgSettlements, domain.PGSettlement{
		PGTxnID:      "ch_stripe_mismatch_3006",
		OrderID:      "ORD-US-3006",
		GrossAmount:  mismatchGross2,
		PGFee:        mismatchFee2,
		Tax:          money.Zero(),
		NetAmount:    mismatchNet2,
		PayoutID:     payout3ID,
		PayoutStatus: domain.PayoutStatusSettled,
		SettledAt:    baseTime.Add(56 * time.Hour),
	})

	// 6 Clean Matched US Orders
	for i := 7; i <= 12; i++ {
		orderID := fmt.Sprintf("ORD-US-30%02d", i)
		gross := money.FromInt(int64(1200 + i*450))
		fee := money.Round4(gross.Mul(money.FromFloat(0.029)))
		net := gross.Sub(fee)
		p3NetTotal = p3NetTotal.Add(net)

		orders = append(orders, domain.Order{
			OrderID:     orderID,
			CustomerID:  fmt.Sprintf("CUST-US-1%02d", i),
			GrossAmount: gross,
			Currency:    "USD",
			OrderStatus: domain.OrderStatusCaptured,
			CreatedAt:   baseTime.Add(time.Duration(50+i) * time.Hour),
		})

		pgSettlements = append(pgSettlements, domain.PGSettlement{
			PGTxnID:      fmt.Sprintf("ch_stripe_30%02d", i),
			OrderID:      orderID,
			GrossAmount:  gross,
			PGFee:        fee,
			Tax:          money.Zero(),
			NetAmount:    net,
			PayoutID:     payout3ID,
			PayoutStatus: domain.PayoutStatusSettled,
			SettledAt:    baseTime.Add(time.Duration(51+i) * time.Hour),
		})
	}

	bankStatements = append(bankStatements, domain.BankStatement{
		BankTxnID:       "BNK-JPMC-20261104-003",
		UTRNumber:       utr3,
		CreditAmount:    p3NetTotal,
		Description:     fmt.Sprintf("JPMC WIRE TRANSFER STRIPE PAYOUT %s", payout3ID),
		TransactionDate: baseTime.Add(72 * time.Hour),
	})

	// -------------------------------------------------------------------------
	// BATCH 4: Payout Shortfall / Haircut Discrepancy (INR / Axis Bank)
	// Payout ID: PO-IN-2026-004 | UTR: AXISR52026110590404
	// 10 orders. PG expected net sum = ₹555,280.00, Bank credited ₹546,280.00 (₹9,000 variance)
	// -------------------------------------------------------------------------
	payout4ID := "PO-IN-2026-004"
	utr4 := "AXISR52026110590404"
	var p4NetTotal money.Decimal

	for i := 1; i <= 10; i++ {
		orderID := fmt.Sprintf("ORD-IN-40%02d", i)
		gross := money.FromInt(int64(45000 + i*5000))
		fee, tax, net := money.CalculateMDRAndGST(gross, money.FromFloat(0.02), money.FromFloat(0.18))
		p4NetTotal = p4NetTotal.Add(net)

		orders = append(orders, domain.Order{
			OrderID:     orderID,
			CustomerID:  fmt.Sprintf("CUST-IN-70%02d", i),
			GrossAmount: gross,
			Currency:    "INR",
			OrderStatus: domain.OrderStatusCaptured,
			CreatedAt:   baseTime.Add(time.Duration(72+i) * time.Hour),
		})

		pgSettlements = append(pgSettlements, domain.PGSettlement{
			PGTxnID:      fmt.Sprintf("pay_rzp_40%02d", i),
			OrderID:      orderID,
			GrossAmount:  gross,
			PGFee:        fee,
			Tax:          tax,
			NetAmount:    net,
			PayoutID:     payout4ID,
			PayoutStatus: domain.PayoutStatusSettled,
			SettledAt:    baseTime.Add(time.Duration(73+i) * time.Hour),
		})
	}

	// Bank credit has ₹9,000.00 shortfall (haircut / bank deduction)
	shortfall := money.FromStringOrZero("9000.0000")
	bankStatements = append(bankStatements, domain.BankStatement{
		BankTxnID:       "BNK-AXIS-20261105-004",
		UTRNumber:       utr4,
		CreditAmount:    p4NetTotal.Sub(shortfall),
		Description:     fmt.Sprintf("AXIS BANK RTGS RAZORPAY %s LESS ADJ", payout4ID),
		TransactionDate: baseTime.Add(96 * time.Hour),
	})

	// -------------------------------------------------------------------------
	// BATCH 5: Unsettled Payout / Bank Missing (USD / Silicon Valley Bank)
	// Payout ID: PO-US-2026-005 | NO matching Bank statement (Uncredited funds)
	// 8 orders settled in PG, $68,400 uncredited
	// -------------------------------------------------------------------------
	payout5ID := "PO-US-2026-005"
	for i := 1; i <= 8; i++ {
		orderID := fmt.Sprintf("ORD-US-50%02d", i)
		gross := money.FromInt(int64(8000 + i*1200))
		fee := money.Round4(gross.Mul(money.FromFloat(0.029)))
		net := gross.Sub(fee)

		orders = append(orders, domain.Order{
			OrderID:     orderID,
			CustomerID:  fmt.Sprintf("CUST-US-5%02d", i),
			GrossAmount: gross,
			Currency:    "USD",
			OrderStatus: domain.OrderStatusCaptured,
			CreatedAt:   baseTime.Add(time.Duration(96+i) * time.Hour),
		})

		pgSettlements = append(pgSettlements, domain.PGSettlement{
			PGTxnID:      fmt.Sprintf("ch_stripe_50%02d", i),
			OrderID:      orderID,
			GrossAmount:  gross,
			PGFee:        fee,
			Tax:          money.Zero(),
			NetAmount:    net,
			PayoutID:     payout5ID,
			PayoutStatus: domain.PayoutStatusSettled,
			SettledAt:    baseTime.Add(time.Duration(97+i) * time.Hour),
		})
	}
	// Note: Bank statement is intentionally missing for payout5ID to model UNSETTLED_PAYOUT / BANK_MISSING

	// -------------------------------------------------------------------------
	// BATCH 6: Enterprise High-Volume Clean Batch (INR / State Bank of India)
	// Payout ID: PO-IN-2026-006 | UTR: SBINR52026110790606
	// 15 enterprise transactions, 100% matched
	// -------------------------------------------------------------------------
	payout6ID := "PO-IN-2026-006"
	utr6 := "SBINR52026110790606"
	var p6NetTotal money.Decimal

	for i := 1; i <= 15; i++ {
		orderID := fmt.Sprintf("ORD-IN-60%02d", i)
		gross := money.FromInt(int64(50000 + i*15000))
		fee, tax, net := money.CalculateMDRAndGST(gross, money.FromFloat(0.02), money.FromFloat(0.18))
		p6NetTotal = p6NetTotal.Add(net)

		orders = append(orders, domain.Order{
			OrderID:     orderID,
			CustomerID:  fmt.Sprintf("CUST-CORP-6%02d", i),
			GrossAmount: gross,
			Currency:    "INR",
			OrderStatus: domain.OrderStatusCaptured,
			CreatedAt:   baseTime.Add(time.Duration(120+i) * time.Hour),
		})

		pgSettlements = append(pgSettlements, domain.PGSettlement{
			PGTxnID:      fmt.Sprintf("pay_rzp_60%02d", i),
			OrderID:      orderID,
			GrossAmount:  gross,
			PGFee:        fee,
			Tax:          tax,
			NetAmount:    net,
			PayoutID:     payout6ID,
			PayoutStatus: domain.PayoutStatusSettled,
			SettledAt:    baseTime.Add(time.Duration(121+i) * time.Hour),
		})
	}

	bankStatements = append(bankStatements, domain.BankStatement{
		BankTxnID:       "BNK-SBI-20261107-006",
		UTRNumber:       utr6,
		CreditAmount:    p6NetTotal,
		Description:     fmt.Sprintf("SBI E-SETTLEMENT RAZORPAY NODAL %s", payout6ID),
		TransactionDate: baseTime.Add(144 * time.Hour),
	})

	return Dataset{
		Orders:         orders,
		PGSettlements:  pgSettlements,
		BankStatements: bankStatements,
	}
}

// SeedRepository clears and inserts the realistic dataset into any storage implementation.
func SeedRepository(ctx context.Context, repo repository.ReconciliationRepository) error {
	if err := repo.ResetAll(ctx); err != nil {
		return fmt.Errorf("reset repository: %w", err)
	}

	data := BuildRealisticDataset()

	if err := repo.Orders().InsertBatch(ctx, data.Orders); err != nil {
		return fmt.Errorf("seed orders: %w", err)
	}
	if err := repo.PG().InsertBatch(ctx, data.PGSettlements); err != nil {
		return fmt.Errorf("seed pg settlements: %w", err)
	}
	if err := repo.Bank().InsertBatch(ctx, data.BankStatements); err != nil {
		return fmt.Errorf("seed bank statements: %w", err)
	}

	return nil
}

// BootstrapIfEmpty checks if the repository is empty, seeds the dataset if needed,
// and immediately runs the reconciliation pipeline so endpoints are pre-computed.
type ReconcileRunner interface {
	RunReconciliation(ctx context.Context, customOrders []domain.Order, customPG []domain.PGSettlement, customBank []domain.BankStatement) (interface{}, error)
}

func BootstrapIfEmpty(
	ctx context.Context,
	repo repository.ReconciliationRepository,
	autoSeed bool,
	runReconcile func(ctx context.Context) error,
) error {
	count, err := repo.Orders().Count(ctx)
	if err != nil {
		return fmt.Errorf("check orders count: %w", err)
	}

	if count == 0 && autoSeed {
		if err := SeedRepository(ctx, repo); err != nil {
			return fmt.Errorf("auto seed repository: %w", err)
		}
	}

	// Always trigger initial reconciliation to prime cache and summary
	if runReconcile != nil {
		if err := runReconcile(ctx); err != nil {
			return fmt.Errorf("prime reconciliation: %w", err)
		}
	}

	return nil
}
