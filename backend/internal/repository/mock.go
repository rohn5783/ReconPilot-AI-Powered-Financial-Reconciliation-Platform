package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"reconpilot/internal/domain"
	"reconpilot/pkg/money"
)

// MockRepository provides a high-fidelity, thread-safe in-memory database
// pre-seeded with realistic multi-currency financial records and real-world discrepancy scenarios.
type MockRepository struct {
	mu            sync.RWMutex
	orders        *MockOrderRepo
	pg            *MockPGRepo
	bank          *MockBankRepo
	discrepancies *MockDiscrepancyRepo
}

// NewMockRepository constructs and optionally seeds an in-memory repository.
func NewMockRepository(seedData bool) *MockRepository {
	orders := &MockOrderRepo{items: make(map[string]domain.Order)}
	pg := &MockPGRepo{items: make(map[string]domain.PGSettlement)}
	bank := &MockBankRepo{items: make(map[string]domain.BankStatement)}
	discrepancies := &MockDiscrepancyRepo{items: make(map[uuid.UUID]domain.Discrepancy)}

	repo := &MockRepository{
		orders:        orders,
		pg:            pg,
		bank:          bank,
		discrepancies: discrepancies,
	}

	if seedData {
		repo.SeedRealisticData()
	}

	return repo
}

func (r *MockRepository) Orders() OrderRepository             { return r.orders }
func (r *MockRepository) PG() PGRepository                     { return r.pg }
func (r *MockRepository) Bank() BankRepository                 { return r.bank }
func (r *MockRepository) Discrepancies() DiscrepancyRepository { return r.discrepancies }

func (r *MockRepository) ResetAll(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_ = r.orders.Clear(ctx)
	_ = r.pg.Clear(ctx)
	_ = r.bank.Clear(ctx)
	_ = r.discrepancies.Clear(ctx)
	return nil
}

func (r *MockRepository) Close() error {
	return nil
}

// SeedRealisticData populates the mock repository with realistic Indian (INR/Razorpay/HDFC/ICICI)
// and Global (USD/Stripe/JPMorgan) financial ledger records covering every discrepancy type.
func (r *MockRepository) SeedRealisticData() {
	r.mu.Lock()
	defer r.mu.Unlock()

	baseTime := time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)

	// =========================================================================
	// 1. PERFECT 3-WAY MATCHED BATCH (Batch 1: INR / Razorpay / HDFC)
	// Payout ID: PO-IN-2026-001 | UTR: HDFCR52026110290101
	// 5 orders, 2.0% fee + 18% GST, exact match at Level 1 & Level 2
	// =========================================================================
	payout1ID := "PO-IN-2026-001"
	utr1 := "HDFCR52026110290101"
	var p1NetTotal money.Decimal

	for i := 1; i <= 5; i++ {
		orderID := fmt.Sprintf("ORD-IN-100%d", i)
		gross := money.FromInt(int64(10000 * i))
		fee, tax, net := money.CalculateMDRAndGST(gross, money.FromFloat(0.02), money.FromFloat(0.18))
		p1NetTotal = p1NetTotal.Add(net)

		r.orders.items[orderID] = domain.Order{
			OrderID:     orderID,
			CustomerID:  fmt.Sprintf("CUST-IN-90%d", i),
			GrossAmount: gross,
			Currency:    "INR",
			OrderStatus: domain.OrderStatusCaptured,
			CreatedAt:   baseTime.Add(time.Duration(i) * time.Hour),
		}

		pgTxnID := fmt.Sprintf("pay_rzp_900%d", i)
		r.pg.items[pgTxnID] = domain.PGSettlement{
			PGTxnID:      pgTxnID,
			OrderID:      orderID,
			GrossAmount:  gross,
			PGFee:        fee,
			Tax:          tax,
			NetAmount:    net,
			PayoutID:     payout1ID,
			PayoutStatus: domain.PayoutStatusSettled,
			SettledAt:    baseTime.Add(time.Duration(i+24) * time.Hour),
		}
	}

	// Bank statement credit matching exactly p1NetTotal
	r.bank.items["BNK-HDFC-001"] = domain.BankStatement{
		BankTxnID:       "BNK-HDFC-001",
		UTRNumber:       utr1,
		CreditAmount:    p1NetTotal,
		Description:     fmt.Sprintf("CMS/RAZORPAY SOFTWARE/PO-IN-2026-001/%s", utr1),
		TransactionDate: baseTime.Add(26 * time.Hour),
	}

	// =========================================================================
	// 2. DISCREPANCY SCENARIO A: FEE OVERCHARGE (Level 1)
	// Order: ORD-IN-1006 | Customer: CUST-IN-906
	// Agreed fee: 2.0% (900.0000) + 18% GST (162.0000) = Net 43,938.0000
	// Charged fee: 2.8% (1,260.0000) + 18% GST (226.8000) = Net 43,513.2000
	// Mismatch / Overcharge: 424.8000
	// =========================================================================
	ord6Gross := money.FromStringOrZero("45000.0000")
	ord6ExpectedFee, ord6ExpectedTax, ord6ExpectedNet := money.CalculateMDRAndGST(ord6Gross, money.FromFloat(0.02), money.FromFloat(0.18))
	ord6ActualFee, ord6ActualTax, ord6ActualNet := money.CalculateMDRAndGST(ord6Gross, money.FromFloat(0.028), money.FromFloat(0.18))
	ord6Mismatch := ord6ExpectedNet.Sub(ord6ActualNet).Abs()

	r.orders.items["ORD-IN-1006"] = domain.Order{
		OrderID:     "ORD-IN-1006",
		CustomerID:  "CUST-IN-906",
		GrossAmount: ord6Gross,
		Currency:    "INR",
		OrderStatus: domain.OrderStatusCaptured,
		CreatedAt:   baseTime.Add(2 * time.Hour),
	}

	ord6PG := domain.PGSettlement{
		PGTxnID:      "pay_rzp_9006_overcharge",
		OrderID:      "ORD-IN-1006",
		GrossAmount:  ord6Gross,
		PGFee:        ord6ActualFee,
		Tax:          ord6ActualTax,
		NetAmount:    ord6ActualNet,
		PayoutID:     "PO-IN-2026-002",
		PayoutStatus: domain.PayoutStatusSettled,
		SettledAt:    baseTime.Add(26 * time.Hour),
	}
	r.pg.items[ord6PG.PGTxnID] = ord6PG

	ord6Item := r.orders.items["ORD-IN-1006"]
	disc1ID := uuid.MustParse("00000000-0000-0000-0001-000000000001")
	ordID6 := "ORD-IN-1006"
	poID2 := "PO-IN-2026-002"
	r.discrepancies.items[disc1ID] = domain.Discrepancy{
		ID:               disc1ID,
		OrderID:          &ordID6,
		PayoutID:         &poID2,
		DiscrepancyLevel: domain.DiscrepancyLevel1OrderPG,
		DiscrepancyType:  domain.TypeFeeOvercharge,
		ExpectedAmount:   ord6ExpectedNet,
		ActualAmount:     ord6ActualNet,
		MismatchAmount:   ord6Mismatch,
		Severity:         domain.SeverityHigh,
		RawEvidence: domain.RawEvidence{
			Order:        &ord6Item,
			PGSettlement: &ord6PG,
			ExpectedFormula: fmt.Sprintf("Net = Gross(%s) - Fee2%%(%s) - GST18%%(%s) = %s",
				ord6Gross.StringFixed(4), ord6ExpectedFee.StringFixed(4), ord6ExpectedTax.StringFixed(4), ord6ExpectedNet.StringFixed(4)),
			ActualFormula: fmt.Sprintf("Net = Gross(%s) - ActualFee(%s) - ActualTax(%s) = %s",
				ord6Gross.StringFixed(4), ord6ActualFee.StringFixed(4), ord6ActualTax.StringFixed(4), ord6ActualNet.StringFixed(4)),
			CalculatedValues: map[string]interface{}{
				"expected_fee": ord6ExpectedFee.StringFixed(4),
				"actual_fee":   ord6ActualFee.StringFixed(4),
				"expected_tax": ord6ExpectedTax.StringFixed(4),
				"actual_tax":   ord6ActualTax.StringFixed(4),
				"overcharge":   ord6Mismatch.StringFixed(4),
			},
		},
		AIDiagnosis: &domain.AIDiagnosis{
			RootCauseSummary:    "Razorpay applied standard card MDR rate of 2.80% instead of contractually negotiated enterprise rate of 2.00% under SLA Annex B.",
			DiscrepancyCategory: "Payment Gateway Overbilling / Contract Rate Drift",
			ConfidenceScore:     98,
			RecommendedActions: []string{
				"File fee reversal claim with Razorpay Merchant Operations referencing Merchant Agreement #RP-2025-IND-88.",
				"Apply credit ledger adjustment of ₹424.8000 against pending payout batch.",
				"Audit pricing webhook rule for merchant account MID_RZP_IND_4491.",
			},
			DraftDisputeTicket: fmt.Sprintf("RE: Discrepancy Ticket #DISC-1006 - MDR Overcharge Dispute\nMerchant ID: MID_RZP_IND_4491\nOrder ID: ORD-IN-1006 | PG Txn: pay_rzp_9006_overcharge\nExpected MDR: 2.00%% (₹%s) + 18%% GST (₹%s) = ₹%s Net\nActual MDR Charged: 2.80%% (₹%s) + 18%% GST (₹%s) = ₹%s Net\nDisputed Overcharge: ₹%s\nPlease credit ₹%s to our settlement ledger within 48 hours.",
				ord6ExpectedFee.StringFixed(4), ord6ExpectedTax.StringFixed(4), ord6ExpectedNet.StringFixed(4),
				ord6ActualFee.StringFixed(4), ord6ActualTax.StringFixed(4), ord6ActualNet.StringFixed(4),
				ord6Mismatch.StringFixed(4), ord6Mismatch.StringFixed(4)),
		},
		Status:     domain.StatusOpen,
		DetectedAt: baseTime.Add(27 * time.Hour),
	}

	// =========================================================================
	// 3. DISCREPANCY SCENARIO B: ORDER MISSING IN PG (Level 1)
	// Order: ORD-IN-1007 | Captured internally for ₹18,750.0000, 0 PG records
	// =========================================================================
	ord7Gross := money.FromStringOrZero("18750.0000")
	r.orders.items["ORD-IN-1007"] = domain.Order{
		OrderID:     "ORD-IN-1007",
		CustomerID:  "CUST-IN-907",
		GrossAmount: ord7Gross,
		Currency:    "INR",
		OrderStatus: domain.OrderStatusCaptured,
		CreatedAt:   baseTime.Add(3 * time.Hour),
	}

	ord7Item := r.orders.items["ORD-IN-1007"]
	disc2ID := uuid.MustParse("00000000-0000-0000-0001-000000000002")
	ordID7 := "ORD-IN-1007"
	r.discrepancies.items[disc2ID] = domain.Discrepancy{
		ID:               disc2ID,
		OrderID:          &ordID7,
		DiscrepancyLevel: domain.DiscrepancyLevel1OrderPG,
		DiscrepancyType:  domain.TypeOrderMissingInPG,
		ExpectedAmount:   ord7Gross,
		ActualAmount:     money.Zero(),
		MismatchAmount:   ord7Gross,
		Severity:         domain.SeverityCritical,
		RawEvidence: domain.RawEvidence{
			Order:            &ord7Item,
			ExpectedFormula:  "PG Gross == Order Gross (18750.0000)",
			ActualFormula:    "PG Settlement record NOT FOUND in ingested settlement ledger",
			CalculatedValues: map[string]interface{}{"missing_amount": ord7Gross.StringFixed(4)},
		},
		AIDiagnosis: &domain.AIDiagnosis{
			RootCauseSummary:    "Order was marked CAPTURED by internal order checkout service, but no corresponding settlement webhook or transaction was received from Razorpay.",
			DiscrepancyCategory: "Unsettled Merchant Captured Volume / Dropped Webhook",
			ConfidenceScore:     95,
			RecommendedActions: []string{
				"Query Razorpay Payments API for payment status matching customer metadata or receipt ID ORD-IN-1007.",
				"If payment failed at PG, update order status to FAILED and cancel fulfillment delivery.",
				"If payment was captured at PG, trigger manual backfill settlement ingestion.",
			},
			DraftDisputeTicket: "URGENT: Reconcile Missing Transaction\nOrder ID: ORD-IN-1007\nCustomer: CUST-IN-907\nAmount: ₹18,750.0000\nInternal Status: CAPTURED at 2026-11-01 12:00 UTC\nNo settlement or authorization found on payment gateway. Please confirm capture or reversal.",
		},
		Status:     domain.StatusOpen,
		DetectedAt: baseTime.Add(28 * time.Hour),
	}

	// =========================================================================
	// 4. DISCREPANCY SCENARIO C: CAPTURE STATUS MISMATCH (Level 1)
	// Order: ORD-IN-1008 | Internal: REFUNDED (₹12,400.0000)
	// PG Settlement: Settled as captured without recording refund
	// =========================================================================
	ord8Gross := money.FromStringOrZero("12400.0000")
	r.orders.items["ORD-IN-1008"] = domain.Order{
		OrderID:     "ORD-IN-1008",
		CustomerID:  "CUST-IN-908",
		GrossAmount: ord8Gross,
		Currency:    "INR",
		OrderStatus: domain.OrderStatusRefunded,
		CreatedAt:   baseTime.Add(4 * time.Hour),
	}

	ord8PG := domain.PGSettlement{
		PGTxnID:      "pay_rzp_9008_mismatch",
		OrderID:      "ORD-IN-1008",
		GrossAmount:  ord8Gross,
		PGFee:        money.FromStringOrZero("248.0000"),
		Tax:          money.FromStringOrZero("44.6400"),
		NetAmount:    money.FromStringOrZero("12107.3600"),
		PayoutID:     "PO-IN-2026-002",
		PayoutStatus: domain.PayoutStatusSettled,
		SettledAt:    baseTime.Add(27 * time.Hour),
	}
	r.pg.items[ord8PG.PGTxnID] = ord8PG

	ord8Item := r.orders.items["ORD-IN-1008"]
	disc3ID := uuid.MustParse("00000000-0000-0000-0001-000000000003")
	ordID8 := "ORD-IN-1008"
	r.discrepancies.items[disc3ID] = domain.Discrepancy{
		ID:               disc3ID,
		OrderID:          &ordID8,
		DiscrepancyLevel: domain.DiscrepancyLevel1OrderPG,
		DiscrepancyType:  domain.TypeCaptureStatusMismatch,
		ExpectedAmount:   money.Zero(),
		ActualAmount:     ord8Gross,
		MismatchAmount:   ord8Gross,
		Severity:         domain.SeverityHigh,
		RawEvidence: domain.RawEvidence{
			Order:           &ord8Item,
			PGSettlement:    &ord8PG,
			ExpectedFormula: "OrderStatus == REFUNDED (0.0000 net settled settlement expected)",
			ActualFormula:   "PGSettlement status == SETTLED with net payout 12107.3600",
			CalculatedValues: map[string]interface{}{
				"order_status": "REFUNDED",
				"pg_status":    "SETTLED",
				"retained_net": ord8PG.NetAmount.StringFixed(4),
			},
		},
		AIDiagnosis: &domain.AIDiagnosis{
			RootCauseSummary:    "Customer requested order cancellation; internal database updated to REFUNDED, but PG refund API call timed out or failed to execute.",
			DiscrepancyCategory: "Unexecuted Gateway Refund / Merchant Double-Retention",
			ConfidenceScore:     92,
			RecommendedActions: []string{
				"Issue immediate refund via Razorpay Dashboard API for payment pay_rzp_9008_mismatch.",
				"Verify customer account has not initiated chargeback dispute.",
			},
			DraftDisputeTicket: "Merchant Operations Notice: Process Pending Refund\nOrder: ORD-IN-1008\nPayment: pay_rzp_9008_mismatch\nAmount: ₹12,400.0000\nAction: Execute refund to customer source account immediately.",
		},
		Status:     domain.StatusOpen,
		DetectedAt: baseTime.Add(29 * time.Hour),
	}

	// =========================================================================
	// 5. DISCREPANCY SCENARIO D: GROSS AMOUNT MISMATCH (Level 1)
	// Order: ORD-IN-1009 | Internal Gross: ₹35,000.0000 | PG Gross: ₹31,500.0000
	// Mismatch: ₹3,500.0000
	// =========================================================================
	ord9Gross := money.FromStringOrZero("35000.0000")
	pg9Gross := money.FromStringOrZero("31500.0000")
	mismatch9 := ord9Gross.Sub(pg9Gross).Abs()

	r.orders.items["ORD-IN-1009"] = domain.Order{
		OrderID:     "ORD-IN-1009",
		CustomerID:  "CUST-IN-909",
		GrossAmount: ord9Gross,
		Currency:    "INR",
		OrderStatus: domain.OrderStatusCaptured,
		CreatedAt:   baseTime.Add(5 * time.Hour),
	}

	ord9PG := domain.PGSettlement{
		PGTxnID:      "pay_rzp_9009_amount_mismatch",
		OrderID:      "ORD-IN-1009",
		GrossAmount:  pg9Gross,
		PGFee:        money.FromStringOrZero("630.0000"),
		Tax:          money.FromStringOrZero("113.4000"),
		NetAmount:    money.FromStringOrZero("30756.6000"),
		PayoutID:     "PO-IN-2026-002",
		PayoutStatus: domain.PayoutStatusSettled,
		SettledAt:    baseTime.Add(28 * time.Hour),
	}
	r.pg.items[ord9PG.PGTxnID] = ord9PG

	ord9Item := r.orders.items["ORD-IN-1009"]
	disc4ID := uuid.MustParse("00000000-0000-0000-0001-000000000004")
	ordID9 := "ORD-IN-1009"
	r.discrepancies.items[disc4ID] = domain.Discrepancy{
		ID:               disc4ID,
		OrderID:          &ordID9,
		DiscrepancyLevel: domain.DiscrepancyLevel1OrderPG,
		DiscrepancyType:  domain.TypeGrossAmountMismatch,
		ExpectedAmount:   ord9Gross,
		ActualAmount:     pg9Gross,
		MismatchAmount:   mismatch9,
		Severity:         domain.SeverityHigh,
		RawEvidence: domain.RawEvidence{
			Order:           &ord9Item,
			PGSettlement:    &ord9PG,
			ExpectedFormula: "PG Gross == Order Gross (35000.0000)",
			ActualFormula:   "PG Gross == 31500.0000 (Shortfall: 3500.0000)",
			CalculatedValues: map[string]interface{}{
				"order_gross": ord9Gross.StringFixed(4),
				"pg_gross":    pg9Gross.StringFixed(4),
				"variance":    mismatch9.StringFixed(4),
			},
		},
		AIDiagnosis: &domain.AIDiagnosis{
			RootCauseSummary:    "Coupon discount of ₹3,500.0000 was applied at checkout but recorded as partial payment instead of line-item promo reduction.",
			DiscrepancyCategory: "Promotional Code Accounting Misallocation",
			ConfidenceScore:     89,
			RecommendedActions: []string{
				"Post ₹3,500.0000 debit note to Marketing Promotional Expense account.",
				"Synchronize order line items to reflect merchant-funded voucher discount.",
			},
			DraftDisputeTicket: "Ledger Rectification Notice\nOrder ID: ORD-IN-1009\nVariance: ₹3,500.0000\nDetail: Ingest promotional ledger entry to balance merchant order against gateway capture.",
		},
		Status:     domain.StatusOpen,
		DetectedAt: baseTime.Add(30 * time.Hour),
	}

	// =========================================================================
	// 6. DISCREPANCY SCENARIO E: BANK MISSING / UNSETTLED PAYOUT (Level 2)
	// Payout ID: PO-IN-2026-081 | UTR: HDFCR52026110398711 | Expected: ₹184,250.0000
	// PG shows SETTLED. Bank statement has NO entry for this UTR.
	// =========================================================================
	payout81ID := "PO-IN-2026-081"
	utr81 := "HDFCR52026110398711"
	p81Net := money.FromStringOrZero("184250.0000")

	// Create 2 PG settlements backing payout 81
	r.pg.items["pay_rzp_8101"] = domain.PGSettlement{
		PGTxnID:      "pay_rzp_8101",
		OrderID:      "ORD-IN-1020",
		GrossAmount:  money.FromStringOrZero("100000.0000"),
		PGFee:        money.FromStringOrZero("2000.0000"),
		Tax:          money.FromStringOrZero("360.0000"),
		NetAmount:    money.FromStringOrZero("97640.0000"),
		PayoutID:     payout81ID,
		PayoutStatus: domain.PayoutStatusSettled,
		SettledAt:    baseTime.Add(48 * time.Hour),
	}
	r.pg.items["pay_rzp_8102"] = domain.PGSettlement{
		PGTxnID:      "pay_rzp_8102",
		OrderID:      "ORD-IN-1021",
		GrossAmount:  money.FromStringOrZero("88700.0000"),
		PGFee:        money.FromStringOrZero("1774.0000"),
		Tax:          money.FromStringOrZero("316.0000"),
		NetAmount:    money.FromStringOrZero("86610.0000"),
		PayoutID:     payout81ID,
		PayoutStatus: domain.PayoutStatusSettled,
		SettledAt:    baseTime.Add(48 * time.Hour),
	}

	disc5ID := uuid.MustParse("00000000-0000-0000-0002-000000000001")
	r.discrepancies.items[disc5ID] = domain.Discrepancy{
		ID:               disc5ID,
		PayoutID:         &payout81ID,
		DiscrepancyLevel: domain.DiscrepancyLevel2PGBank,
		DiscrepancyType:  domain.TypeBankMissing,
		ExpectedAmount:   p81Net,
		ActualAmount:     money.Zero(),
		MismatchAmount:   p81Net,
		Severity:         domain.SeverityCritical,
		RawEvidence: domain.RawEvidence{
			BatchSettlements: []*domain.PGSettlement{
				func(s domain.PGSettlement) *domain.PGSettlement { return &s }(r.pg.items["pay_rzp_8101"]),
				func(s domain.PGSettlement) *domain.PGSettlement { return &s }(r.pg.items["pay_rzp_8102"]),
			},
			ExpectedFormula: fmt.Sprintf("Bank Credit UTR(%s) == Sum(PG Net) (%s)", utr81, p81Net.StringFixed(4)),
			ActualFormula:   "Bank Statement credit NOT FOUND for UTR " + utr81,
			CalculatedValues: map[string]interface{}{
				"utr_number":        utr81,
				"expected_credit":   p81Net.StringFixed(4),
				"actual_bank_found": "0.0000",
			},
		},
		AIDiagnosis: &domain.AIDiagnosis{
			RootCauseSummary:    fmt.Sprintf("Payment Gateway reports payout batch %s settled via UTR %s, but banking partner HDFC has not credited merchant nodal account after 72 hours.", payout81ID, utr81),
			DiscrepancyCategory: "Uncredited Bank Settlement / NEFT-RTGS Clearing Failure",
			ConfidenceScore:     97,
			RecommendedActions: []string{
				"Request Bank Nodal Acknowledgement Slip from Razorpay Treasury Desk.",
				"Escalate to HDFC Corporate Banking Helpdesk with UTR " + utr81 + " for inward remittance trace.",
			},
			DraftDisputeTicket: fmt.Sprintf("ESCALATION: Missing Inward Remittance\nUTR: %s\nPayout ID: %s\nExpected Amount: ₹%s\nNodal Account: HDFC 50200049182741\nStatus: PG reports settled, Bank reports ZERO credit. Immediate investigation requested.", utr81, payout81ID, p81Net.StringFixed(4)),
		},
		Status:     domain.StatusOpen,
		DetectedAt: baseTime.Add(52 * time.Hour),
	}

	// =========================================================================
	// 7. DISCREPANCY SCENARIO F: PAYOUT SHORTFALL (Level 2)
	// Payout ID: PO-IN-2026-084 | UTR: ICICR52026110411234
	// Expected Net: ₹248,600.0000 | Bank Received: ₹243,100.0000
	// Shortfall: ₹5,500.0000 (Undocumented PG fee clawback or reserve withholding)
	// =========================================================================
	payout84ID := "PO-IN-2026-084"
	utr84 := "ICICR52026110411234"
	p84ExpectedNet := money.FromStringOrZero("248600.0000")
	p84ActualBank := money.FromStringOrZero("243100.0000")
	p84Shortfall := p84ExpectedNet.Sub(p84ActualBank)

	r.pg.items["pay_rzp_8401"] = domain.PGSettlement{
		PGTxnID:      "pay_rzp_8401",
		OrderID:      "ORD-IN-1022",
		GrossAmount:  money.FromStringOrZero("254600.0000"),
		PGFee:        money.FromStringOrZero("5084.7500"),
		Tax:          money.FromStringOrZero("915.2500"),
		NetAmount:    p84ExpectedNet,
		PayoutID:     payout84ID,
		PayoutStatus: domain.PayoutStatusSettled,
		SettledAt:    baseTime.Add(72 * time.Hour),
	}

	r.bank.items["BNK-ICICI-084"] = domain.BankStatement{
		BankTxnID:       "BNK-ICICI-084",
		UTRNumber:       utr84,
		CreditAmount:    p84ActualBank,
		Description:     fmt.Sprintf("NEFT-ICICI-RAZORPAY-%s-%s", payout84ID, utr84),
		TransactionDate: baseTime.Add(74 * time.Hour),
	}

	disc6ID := uuid.MustParse("00000000-0000-0000-0002-000000000002")
	r.discrepancies.items[disc6ID] = domain.Discrepancy{
		ID:               disc6ID,
		PayoutID:         &payout84ID,
		DiscrepancyLevel: domain.DiscrepancyLevel2PGBank,
		DiscrepancyType:  domain.TypePayoutShortfall,
		ExpectedAmount:   p84ExpectedNet,
		ActualAmount:     p84ActualBank,
		MismatchAmount:   p84Shortfall,
		Severity:         domain.SeverityCritical,
		RawEvidence: domain.RawEvidence{
			BatchSettlements: []*domain.PGSettlement{
				func(s domain.PGSettlement) *domain.PGSettlement { return &s }(r.pg.items["pay_rzp_8401"]),
			},
			BankStatement:   func(b domain.BankStatement) *domain.BankStatement { return &b }(r.bank.items["BNK-ICICI-084"]),
			ExpectedFormula: fmt.Sprintf("Bank Credit (%s) == Expected Net (%s)", p84ExpectedNet.StringFixed(4), p84ExpectedNet.StringFixed(4)),
			ActualFormula:   fmt.Sprintf("Bank Credit (%s) < Expected Net (%s) -> Shortfall (%s)", p84ActualBank.StringFixed(4), p84ExpectedNet.StringFixed(4), p84Shortfall.StringFixed(4)),
			CalculatedValues: map[string]interface{}{
				"expected_net": p84ExpectedNet.StringFixed(4),
				"actual_bank":  p84ActualBank.StringFixed(4),
				"shortfall":    p84Shortfall.StringFixed(4),
			},
		},
		AIDiagnosis: &domain.AIDiagnosis{
			RootCauseSummary:    fmt.Sprintf("Settlement payout %s was deducted by ₹%s at bank disbursement; likely unnotified rolling security reserve retention or prior negative balance clawback.", payout84ID, p84Shortfall.StringFixed(4)),
			DiscrepancyCategory: "Unnotified Gateway Clawback / Rolling Reserve Retention",
			ConfidenceScore:     94,
			RecommendedActions: []string{
				"Download monthly settlement breakdown report from Razorpay Settlement API.",
				"Check for rolling reserve withholdings or chargeback debit deductions.",
			},
			DraftDisputeTicket: fmt.Sprintf("Settlement Shortfall Inquiry\nPayout Batch: %s\nUTR: %s\nExpected Net: ₹%s\nReceived Credit: ₹%s\nUnaccounted Shortfall: ₹%s\nPlease provide line-item tax and deduction ledger for this payout.",
				payout84ID, utr84, p84ExpectedNet.StringFixed(4), p84ActualBank.StringFixed(4), p84Shortfall.StringFixed(4)),
		},
		Status:     domain.StatusOpen,
		DetectedAt: baseTime.Add(75 * time.Hour),
	}

	// =========================================================================
	// 8. GLOBAL USD / STRIPE / JPMORGAN BATCH
	// Batch: PO-US-2026-001 | UTR: JPMC-ACH-99128471 | Orders ORD-US-5001..5003
	// =========================================================================
	usPayoutID := "PO-US-2026-001"
	usUTR := "JPMC-ACH-99128471"
	var usNetTotal money.Decimal

	for i := 1; i <= 3; i++ {
		ordID := fmt.Sprintf("ORD-US-500%d", i)
		gross := money.FromInt(int64(250 * i))
		// Stripe standard rate: 2.9% + $0.30, 0 tax
		fee := gross.Mul(money.FromStringOrZero("0.029")).Add(money.FromStringOrZero("0.3000")).RoundBank(4)
		net := gross.Sub(fee)
		usNetTotal = usNetTotal.Add(net)

		r.orders.items[ordID] = domain.Order{
			OrderID:     ordID,
			CustomerID:  fmt.Sprintf("CUST-US-10%d", i),
			GrossAmount: gross,
			Currency:    "USD",
			OrderStatus: domain.OrderStatusCaptured,
			CreatedAt:   baseTime.Add(time.Duration(i*2) * time.Hour),
		}

		txnID := fmt.Sprintf("ch_3M4v8%d", i)
		r.pg.items[txnID] = domain.PGSettlement{
			PGTxnID:      txnID,
			OrderID:      ordID,
			GrossAmount:  gross,
			PGFee:        fee,
			Tax:          money.Zero(),
			NetAmount:    net,
			PayoutID:     usPayoutID,
			PayoutStatus: domain.PayoutStatusSettled,
			SettledAt:    baseTime.Add(time.Duration(i*2+24) * time.Hour),
		}
	}

	r.bank.items["BNK-JPMC-501"] = domain.BankStatement{
		BankTxnID:       "BNK-JPMC-501",
		UTRNumber:       usUTR,
		CreditAmount:    usNetTotal,
		Description:     fmt.Sprintf("STRIPE PAYMENTS JPMC ACH %s %s", usPayoutID, usUTR),
		TransactionDate: baseTime.Add(30 * time.Hour),
	}
}

// MockOrderRepo implements OrderRepository in-memory.
type MockOrderRepo struct {
	mu    sync.RWMutex
	items map[string]domain.Order
}

func (r *MockOrderRepo) InsertBatch(ctx context.Context, orders []domain.Order) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, o := range orders {
		r.items[o.OrderID] = o
	}
	return nil
}

func (r *MockOrderRepo) FindAll(ctx context.Context) ([]domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]domain.Order, 0, len(r.items))
	for _, o := range r.items {
		res = append(res, o)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].CreatedAt.After(res[j].CreatedAt)
	})
	return res, nil
}

func (r *MockOrderRepo) FindByID(ctx context.Context, orderID string) (*domain.Order, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	o, exists := r.items[orderID]
	if !exists {
		return nil, nil
	}
	return &o, nil
}

func (r *MockOrderRepo) Count(ctx context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.items)), nil
}

func (r *MockOrderRepo) Clear(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = make(map[string]domain.Order)
	return nil
}

// MockPGRepo implements PGRepository in-memory.
type MockPGRepo struct {
	mu    sync.RWMutex
	items map[string]domain.PGSettlement
}

func (r *MockPGRepo) InsertBatch(ctx context.Context, settlements []domain.PGSettlement) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, s := range settlements {
		r.items[s.PGTxnID] = s
	}
	return nil
}

func (r *MockPGRepo) FindAll(ctx context.Context) ([]domain.PGSettlement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]domain.PGSettlement, 0, len(r.items))
	for _, s := range r.items {
		res = append(res, s)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].SettledAt.After(res[j].SettledAt)
	})
	return res, nil
}

func (r *MockPGRepo) FindByOrderID(ctx context.Context, orderID string) (*domain.PGSettlement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.items {
		if s.OrderID == orderID {
			return &s, nil
		}
	}
	return nil, nil
}

func (r *MockPGRepo) FindByPayoutID(ctx context.Context, payoutID string) ([]domain.PGSettlement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var res []domain.PGSettlement
	for _, s := range r.items {
		if s.PayoutID == payoutID {
			res = append(res, s)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].SettledAt.After(res[j].SettledAt)
	})
	return res, nil
}

func (r *MockPGRepo) FindDistinctPayoutIDs(ctx context.Context) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := make(map[string]bool)
	var res []string
	for _, s := range r.items {
		if s.PayoutID != "" && !seen[s.PayoutID] {
			seen[s.PayoutID] = true
			res = append(res, s.PayoutID)
		}
	}
	sort.Strings(res)
	return res, nil
}

func (r *MockPGRepo) Count(ctx context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.items)), nil
}

func (r *MockPGRepo) Clear(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = make(map[string]domain.PGSettlement)
	return nil
}

// MockBankRepo implements BankRepository in-memory.
type MockBankRepo struct {
	mu    sync.RWMutex
	items map[string]domain.BankStatement
}

func (r *MockBankRepo) InsertBatch(ctx context.Context, statements []domain.BankStatement) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, b := range statements {
		r.items[b.BankTxnID] = b
	}
	return nil
}

func (r *MockBankRepo) FindAll(ctx context.Context) ([]domain.BankStatement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]domain.BankStatement, 0, len(r.items))
	for _, b := range r.items {
		res = append(res, b)
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].TransactionDate.After(res[j].TransactionDate)
	})
	return res, nil
}

func (r *MockBankRepo) FindByUTR(ctx context.Context, utr string) (*domain.BankStatement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, b := range r.items {
		if strings.EqualFold(b.UTRNumber, utr) {
			return &b, nil
		}
	}
	return nil, nil
}

func (r *MockBankRepo) Count(ctx context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.items)), nil
}

func (r *MockBankRepo) Clear(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = make(map[string]domain.BankStatement)
	return nil
}

// MockDiscrepancyRepo implements DiscrepancyRepository in-memory.
type MockDiscrepancyRepo struct {
	mu    sync.RWMutex
	items map[uuid.UUID]domain.Discrepancy
}

func (r *MockDiscrepancyRepo) InsertBatch(ctx context.Context, discrepancies []domain.Discrepancy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, d := range discrepancies {
		r.items[d.ID] = d
	}
	return nil
}

func (r *MockDiscrepancyRepo) FindAll(ctx context.Context, filter DiscrepancyFilter) ([]domain.Discrepancy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []domain.Discrepancy
	for _, d := range r.items {
		if filter.Level != nil && d.DiscrepancyLevel != *filter.Level {
			continue
		}
		if filter.Type != nil && d.DiscrepancyType != *filter.Type {
			continue
		}
		if filter.Severity != nil && d.Severity != *filter.Severity {
			continue
		}
		if filter.Status != nil && d.Status != *filter.Status {
			continue
		}
		if filter.OrderID != nil && (d.OrderID == nil || *d.OrderID != *filter.OrderID) {
			continue
		}
		if filter.PayoutID != nil && (d.PayoutID == nil || *d.PayoutID != *filter.PayoutID) {
			continue
		}
		matched = append(matched, d)
	}

	sort.Slice(matched, func(i, j int) bool {
		return matched[i].DetectedAt.After(matched[j].DetectedAt)
	})

	if filter.Offset > 0 {
		if filter.Offset >= len(matched) {
			return []domain.Discrepancy{}, nil
		}
		matched = matched[filter.Offset:]
	}

	if filter.Limit > 0 && len(matched) > filter.Limit {
		matched = matched[:filter.Limit]
	}

	return matched, nil
}

func (r *MockDiscrepancyRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Discrepancy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, exists := r.items[id]
	if !exists {
		return nil, nil
	}
	return &d, nil
}

func (r *MockDiscrepancyRepo) FindByOrderID(ctx context.Context, orderID string) (*domain.Discrepancy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, d := range r.items {
		if d.OrderID != nil && *d.OrderID == orderID {
			return &d, nil
		}
	}
	return nil, nil
}

func (r *MockDiscrepancyRepo) FindByPayoutID(ctx context.Context, payoutID string) (*domain.Discrepancy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, d := range r.items {
		if d.PayoutID != nil && *d.PayoutID == payoutID {
			return &d, nil
		}
	}
	return nil, nil
}

func (r *MockDiscrepancyRepo) UpdateDiagnosis(ctx context.Context, id uuid.UUID, diagnosis domain.AIDiagnosis) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, exists := r.items[id]
	if !exists {
		return errors.New("discrepancy not found")
	}
	d.AIDiagnosis = &diagnosis
	r.items[id] = d
	return nil
}

func (r *MockDiscrepancyRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.DiscrepancyStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, exists := r.items[id]
	if !exists {
		return errors.New("discrepancy not found")
	}
	d.Status = status
	r.items[id] = d
	return nil
}

func (r *MockDiscrepancyRepo) Count(ctx context.Context) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return int64(len(r.items)), nil
}

func (r *MockDiscrepancyRepo) Clear(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.items = make(map[uuid.UUID]domain.Discrepancy)
	return nil
}
