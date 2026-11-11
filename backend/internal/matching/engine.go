package matching

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"reconpilot/internal/domain"
	"reconpilot/pkg/money"
)

// EngineConfig parameterizes the 3-Way Matching Engine with contract rates and tolerances.
type EngineConfig struct {
	DefaultFeeRate money.Decimal // e.g. 0.0200 (2.0%)
	DefaultTaxRate money.Decimal // e.g. 0.1800 (18.0% GST)
	FeeTolerance   money.Decimal // e.g. 0.0100 (1 cent or 1 paisa tolerance)
	MaxConcurrency int           // honoring PIPELINE_MAX_CONCURRENT_BATCHES
}

// DefaultEngineConfig returns the institutional default matching parameters.
func DefaultEngineConfig() EngineConfig {
	return EngineConfig{
		DefaultFeeRate: money.FromStringOrZero("0.0200"),
		DefaultTaxRate: money.FromStringOrZero("0.1800"),
		FeeTolerance:   money.FromStringOrZero("0.0100"),
		MaxConcurrency: 10,
	}
}

// ReconciliationResult holds the complete output of a 3-way reconciliation run.
type ReconciliationResult struct {
	Discrepancies []domain.Discrepancy       `json:"discrepancies"`
	TraceItems    []domain.ThreeWayTraceItem `json:"trace_items"`
	PayoutBatches []domain.PayoutBatchGroup  `json:"payout_batches"`
	Summary       domain.ReconciliationSummary `json:"summary"`
}

// Engine executes concurrent 3-way reconciliation across orders, PG settlements, and bank statement credits.
type Engine struct {
	config EngineConfig
}

// NewEngine creates a new 3-Way Reconciliation Matching Engine.
func NewEngine(cfg EngineConfig) *Engine {
	if cfg.MaxConcurrency <= 0 {
		cfg.MaxConcurrency = 10
	}
	if cfg.FeeTolerance.IsZero() {
		cfg.FeeTolerance = money.FromStringOrZero("0.0100")
	}
	return &Engine{config: cfg}
}

// Config returns the active engine configuration.
func (e *Engine) Config() EngineConfig {
	return e.config
}

// ProgressFunc reports real-time pipeline events and telemetry to listeners (SSE/WebSockets).
type ProgressFunc func(domain.PipelineTelemetryEvent)

// Reconcile executes Level-1 (Order <-> PG) and Level-2 (PG Payout <-> Bank Statement) matching,
// generates the unified 3-way trace items, and computes executive summary KPIs.
func (e *Engine) Reconcile(
	ctx context.Context,
	orders []domain.Order,
	settlements []domain.PGSettlement,
	bankStatements []domain.BankStatement,
	onProgress ProgressFunc,
) (*ReconciliationResult, error) {
	emit := func(stage string, pct float64, msg string, payload interface{}) {
		if onProgress != nil {
			onProgress(domain.PipelineTelemetryEvent{
				Type:               "progress",
				Stage:              stage,
				ProgressPercentage: pct,
				Message:            msg,
				Payload:            payload,
				Timestamp:          time.Now().UTC(),
			})
		}
	}

	emit("INGESTION", 10.0, fmt.Sprintf("Loaded %d orders, %d PG settlements, %d bank statements", len(orders), len(settlements), len(bankStatements)), nil)

	// Index lookups for O(1) correlation
	orderMap := make(map[string]domain.Order, len(orders))
	for _, o := range orders {
		orderMap[o.OrderID] = o
	}

	pgByOrderMap := make(map[string]domain.PGSettlement, len(settlements))
	pgByIDMap := make(map[string]domain.PGSettlement, len(settlements))
	pgByPayoutMap := make(map[string][]domain.PGSettlement)
	for _, s := range settlements {
		if s.OrderID != "" {
			pgByOrderMap[s.OrderID] = s
		}
		if s.PGTxnID != "" {
			pgByIDMap[s.PGTxnID] = s
		}
		if s.PayoutID != "" {
			pgByPayoutMap[s.PayoutID] = append(pgByPayoutMap[s.PayoutID], s)
		}
	}

	bankByUTRMap := make(map[string]domain.BankStatement, len(bankStatements))
	for _, b := range bankStatements {
		bankByUTRMap[strings.ToUpper(strings.TrimSpace(b.UTRNumber))] = b
	}

	emit("LEVEL_1_MATCH", 25.0, "Starting Level 1 matching: Internal Orders <-> Payment Gateway Settlements", nil)

	// =========================================================================
	// STAGE 1: LEVEL 1 MATCHING (Order <-> PG Settlement)
	// =========================================================================
	var discrepancies []domain.Discrepancy
	orderDiscrepancyMap := make(map[string]*domain.Discrepancy)
	payoutDiscrepancyMap := make(map[string]*domain.Discrepancy)

	for _, o := range orders {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		pg, found := pgByOrderMap[o.OrderID]
		if !found {
			// Order Missing in Payment Gateway
			disc := domain.Discrepancy{
				ID:               uuid.New(),
				OrderID:          stringPtr(o.OrderID),
				DiscrepancyLevel: domain.DiscrepancyLevel1OrderPG,
				DiscrepancyType:  domain.TypeOrderMissingInPG,
				ExpectedAmount:   o.GrossAmount,
				ActualAmount:     money.Zero(),
				MismatchAmount:   o.GrossAmount,
				Severity:         domain.SeverityCritical,
				RawEvidence: domain.RawEvidence{
					Order:           copyOrder(o),
					ExpectedFormula: fmt.Sprintf("PG Gross == Order Gross (%s)", o.GrossAmount.StringFixed(4)),
					ActualFormula:   "PG Settlement record NOT FOUND in gateway ledger",
					CalculatedValues: map[string]interface{}{
						"missing_amount": o.GrossAmount.StringFixed(4),
						"currency":       o.Currency,
					},
				},
				Status:     domain.StatusOpen,
				DetectedAt: time.Now().UTC(),
			}
			discrepancies = append(discrepancies, disc)
			orderDiscrepancyMap[o.OrderID] = &disc
			continue
		}

		// 1. Gross Amount Mismatch Check
		if !o.GrossAmount.Equal(pg.GrossAmount) {
			diff := o.GrossAmount.Sub(pg.GrossAmount).Abs()
			disc := domain.Discrepancy{
				ID:               uuid.New(),
				OrderID:          stringPtr(o.OrderID),
				PayoutID:         stringPtr(pg.PayoutID),
				DiscrepancyLevel: domain.DiscrepancyLevel1OrderPG,
				DiscrepancyType:  domain.TypeGrossAmountMismatch,
				ExpectedAmount:   o.GrossAmount,
				ActualAmount:     pg.GrossAmount,
				MismatchAmount:   diff,
				Severity:         domain.SeverityHigh,
				RawEvidence: domain.RawEvidence{
					Order:        copyOrder(o),
					PGSettlement: copyPG(pg),
					ExpectedFormula: fmt.Sprintf("Order Gross (%s) == PG Gross (%s)",
						o.GrossAmount.StringFixed(4), pg.GrossAmount.StringFixed(4)),
					ActualFormula: fmt.Sprintf("Gross Variance = |%s - %s| = %s",
						o.GrossAmount.StringFixed(4), pg.GrossAmount.StringFixed(4), diff.StringFixed(4)),
					CalculatedValues: map[string]interface{}{
						"order_gross": o.GrossAmount.StringFixed(4),
						"pg_gross":    pg.GrossAmount.StringFixed(4),
						"variance":    diff.StringFixed(4),
					},
				},
				Status:     domain.StatusOpen,
				DetectedAt: time.Now().UTC(),
			}
			discrepancies = append(discrepancies, disc)
			orderDiscrepancyMap[o.OrderID] = &disc
			continue
		}

		// 2. Order Status Mismatch Check (e.g. Refunded/Failed order captured at PG)
		if (o.OrderStatus == domain.OrderStatusRefunded || o.OrderStatus == domain.OrderStatusFailed) &&
			(pg.PayoutStatus == domain.PayoutStatusSettled || pg.PayoutStatus == domain.PayoutStatusProcessed) {
			disc := domain.Discrepancy{
				ID:               uuid.New(),
				OrderID:          stringPtr(o.OrderID),
				PayoutID:         stringPtr(pg.PayoutID),
				DiscrepancyLevel: domain.DiscrepancyLevel1OrderPG,
				DiscrepancyType:  domain.TypeCaptureStatusMismatch,
				ExpectedAmount:   money.Zero(),
				ActualAmount:     pg.GrossAmount,
				MismatchAmount:   pg.GrossAmount,
				Severity:         domain.SeverityHigh,
				RawEvidence: domain.RawEvidence{
					Order:        copyOrder(o),
					PGSettlement: copyPG(pg),
					ExpectedFormula: fmt.Sprintf("Order Status == %s -> Net settled volume should be 0.0000",
						o.OrderStatus),
					ActualFormula: fmt.Sprintf("PG settled %s with net %s",
						pg.PayoutStatus, pg.NetAmount.StringFixed(4)),
					CalculatedValues: map[string]interface{}{
						"order_status": string(o.OrderStatus),
						"pg_status":    string(pg.PayoutStatus),
						"retained_net": pg.NetAmount.StringFixed(4),
					},
				},
				Status:     domain.StatusOpen,
				DetectedAt: time.Now().UTC(),
			}
			discrepancies = append(discrepancies, disc)
			orderDiscrepancyMap[o.OrderID] = &disc
			continue
		}

		// 3. Fee (MDR) Overcharge & Tax Calculation Check
		feeRate := e.config.DefaultFeeRate
		taxRate := e.config.DefaultTaxRate
		if strings.EqualFold(o.Currency, "USD") {
			taxRate = money.Zero()
		}

		expectedFee, expectedTax, expectedNet := money.CalculateMDRAndGST(o.GrossAmount, feeRate, taxRate)
		feeDiff := pg.PGFee.Sub(expectedFee)

		if feeDiff.GreaterThan(e.config.FeeTolerance) {
			mismatch := feeDiff.Abs()
			disc := domain.Discrepancy{
				ID:               uuid.New(),
				OrderID:          stringPtr(o.OrderID),
				PayoutID:         stringPtr(pg.PayoutID),
				DiscrepancyLevel: domain.DiscrepancyLevel1OrderPG,
				DiscrepancyType:  domain.TypeFeeOvercharge,
				ExpectedAmount:   expectedNet,
				ActualAmount:     pg.NetAmount,
				MismatchAmount:   mismatch,
				Severity:         domain.SeverityHigh,
				RawEvidence: domain.RawEvidence{
					Order:        copyOrder(o),
					PGSettlement: copyPG(pg),
					ExpectedFormula: fmt.Sprintf("Fee = Gross(%s) * Rate(%s) = %s, Tax = %s, ExpectedNet = %s",
						o.GrossAmount.StringFixed(4), feeRate.StringFixed(4), expectedFee.StringFixed(4),
						expectedTax.StringFixed(4), expectedNet.StringFixed(4)),
					ActualFormula: fmt.Sprintf("ActualFee = %s, ActualTax = %s, ActualNet = %s (Fee Diff: %s)",
						pg.PGFee.StringFixed(4), pg.Tax.StringFixed(4), pg.NetAmount.StringFixed(4), mismatch.StringFixed(4)),
					CalculatedValues: map[string]interface{}{
						"expected_fee": expectedFee.StringFixed(4),
						"actual_fee":   pg.PGFee.StringFixed(4),
						"expected_tax": expectedTax.StringFixed(4),
						"actual_tax":   pg.Tax.StringFixed(4),
						"overcharge":   mismatch.StringFixed(4),
					},
				},
				Status:     domain.StatusOpen,
				DetectedAt: time.Now().UTC(),
			}
			discrepancies = append(discrepancies, disc)
			orderDiscrepancyMap[o.OrderID] = &disc
			continue
		}

		// Tax calculation error check
		taxDiff := pg.Tax.Sub(expectedTax).Abs()
		if taxDiff.GreaterThan(e.config.FeeTolerance) {
			disc := domain.Discrepancy{
				ID:               uuid.New(),
				OrderID:          stringPtr(o.OrderID),
				PayoutID:         stringPtr(pg.PayoutID),
				DiscrepancyLevel: domain.DiscrepancyLevel1OrderPG,
				DiscrepancyType:  domain.TypeTaxCalculationError,
				ExpectedAmount:   expectedTax,
				ActualAmount:     pg.Tax,
				MismatchAmount:   taxDiff,
				Severity:         domain.SeverityMedium,
				RawEvidence: domain.RawEvidence{
					Order:        copyOrder(o),
					PGSettlement: copyPG(pg),
					ExpectedFormula: fmt.Sprintf("GST Tax = Fee(%s) * TaxRate(%s) = %s",
						pg.PGFee.StringFixed(4), taxRate.StringFixed(4), expectedTax.StringFixed(4)),
					ActualFormula: fmt.Sprintf("Actual Tax billed = %s (Tax Variance: %s)",
						pg.Tax.StringFixed(4), taxDiff.StringFixed(4)),
					CalculatedValues: map[string]interface{}{
						"expected_tax": expectedTax.StringFixed(4),
						"actual_tax":   pg.Tax.StringFixed(4),
						"tax_variance": taxDiff.StringFixed(4),
					},
				},
				Status:     domain.StatusOpen,
				DetectedAt: time.Now().UTC(),
			}
			discrepancies = append(discrepancies, disc)
			orderDiscrepancyMap[o.OrderID] = &disc
		}
	}

	// Check for Orphan PG Settlements (PG txn has no matching Order)
	for _, s := range settlements {
		if _, found := orderMap[s.OrderID]; !found && s.OrderID != "" {
			disc := domain.Discrepancy{
				ID:               uuid.New(),
				OrderID:          stringPtr(s.OrderID),
				PayoutID:         stringPtr(s.PayoutID),
				DiscrepancyLevel: domain.DiscrepancyLevel1OrderPG,
				DiscrepancyType:  domain.TypePGOrphanTxn,
				ExpectedAmount:   money.Zero(),
				ActualAmount:     s.GrossAmount,
				MismatchAmount:   s.GrossAmount,
				Severity:         domain.SeverityHigh,
				RawEvidence: domain.RawEvidence{
					PGSettlement:    copyPG(s),
					ExpectedFormula: "Internal Order record MUST exist for payment capture",
					ActualFormula:   fmt.Sprintf("Payment %s captured without corresponding merchant order %s", s.PGTxnID, s.OrderID),
					CalculatedValues: map[string]interface{}{
						"pg_txn_id":     s.PGTxnID,
						"orphan_amount": s.GrossAmount.StringFixed(4),
					},
				},
				Status:     domain.StatusOpen,
				DetectedAt: time.Now().UTC(),
			}
			discrepancies = append(discrepancies, disc)
			orderDiscrepancyMap[s.OrderID] = &disc
		}
	}

	emit("LEVEL_2_MATCH", 55.0, "Starting Level 2 matching: Payment Gateway Payout Batches <-> Bank Statements (UTR)", nil)

	// =========================================================================
	// STAGE 2: LEVEL 2 MATCHING (PG Payout Batch <-> Bank Statement UTR)
	// =========================================================================
	var payoutBatches []domain.PayoutBatchGroup
	payoutBankCreditMap := make(map[string]domain.BankStatement)

	// Sort payout IDs for deterministic processing
	var payoutIDs []string
	for pid := range pgByPayoutMap {
		if pid != "" {
			payoutIDs = append(payoutIDs, pid)
		}
	}
	sort.Strings(payoutIDs)

	for _, pid := range payoutIDs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		batchItems := pgByPayoutMap[pid]
		var totalGross, totalFee, totalTax, totalExpectedNet money.Decimal

		for _, item := range batchItems {
			totalGross = totalGross.Add(item.GrossAmount)
			totalFee = totalFee.Add(item.PGFee)
			totalTax = totalTax.Add(item.Tax)
			totalExpectedNet = totalExpectedNet.Add(item.NetAmount)
		}

		// Try matching Bank Statement by PayoutID or finding Bank Statement matching UTR/payout
		var matchedBank *domain.BankStatement
		for _, b := range bankStatements {
			descUpper := strings.ToUpper(b.Description)
			utrUpper := strings.ToUpper(b.UTRNumber)
			pidUpper := strings.ToUpper(pid)

			if strings.Contains(descUpper, pidUpper) || strings.Contains(utrUpper, pidUpper) {
				bCopy := b
				matchedBank = &bCopy
				payoutBankCreditMap[pid] = b
				break
			}
		}

		// If not found by description, try searching bank statements for exact credit amount match
		if matchedBank == nil {
			for _, b := range bankStatements {
				if b.CreditAmount.Equal(totalExpectedNet) {
					bCopy := b
					matchedBank = &bCopy
					payoutBankCreditMap[pid] = b
					break
				}
			}
		}

		group := domain.PayoutBatchGroup{
			PayoutID:         pid,
			SettlementsCount: len(batchItems),
			TotalGross:       totalGross,
			TotalFee:         totalFee,
			TotalTax:         totalTax,
			TotalExpectedNet: totalExpectedNet,
			Variance:         money.Zero(),
			Status:           "MATCHED",
		}
		if len(batchItems) > 0 {
			settledTime := batchItems[0].SettledAt
			group.SettledAt = &settledTime
		}

		if matchedBank == nil {
			// Bank Missing / Uncredited Payout
			group.Status = "BANK_MISSING"
			group.Variance = totalExpectedNet.Neg()
			disc := domain.Discrepancy{
				ID:               uuid.New(),
				PayoutID:         stringPtr(pid),
				DiscrepancyLevel: domain.DiscrepancyLevel2PGBank,
				DiscrepancyType:  domain.TypeBankMissing,
				ExpectedAmount:   totalExpectedNet,
				ActualAmount:     money.Zero(),
				MismatchAmount:   totalExpectedNet,
				Severity:         domain.SeverityCritical,
				RawEvidence: domain.RawEvidence{
					BatchSettlements: copyPGSettlements(batchItems),
					ExpectedFormula: fmt.Sprintf("Bank Credit == Sum(PG Net Settlements) = %s",
						totalExpectedNet.StringFixed(4)),
					ActualFormula: fmt.Sprintf("Zero bank credits found matching Payout ID %s", pid),
					CalculatedValues: map[string]interface{}{
						"payout_id":         pid,
						"settlements_count": len(batchItems),
						"expected_credit":   totalExpectedNet.StringFixed(4),
					},
				},
				Status:     domain.StatusOpen,
				DetectedAt: time.Now().UTC(),
			}
			discrepancies = append(discrepancies, disc)
			payoutDiscrepancyMap[pid] = &disc
			group.Discrepancy = &disc
		} else {
			group.BankCredit = &matchedBank.CreditAmount
			group.BankUTR = &matchedBank.UTRNumber
			group.Variance = matchedBank.CreditAmount.Sub(totalExpectedNet)
			bankTxnDate := matchedBank.TransactionDate
			group.SettledAt = &bankTxnDate

			diff := matchedBank.CreditAmount.Sub(totalExpectedNet)
			if diff.LessThan(money.Zero()) && diff.Abs().GreaterThan(e.config.FeeTolerance) {
				// Shortfall
				shortfall := diff.Abs()
				group.Status = "SHORTFALL"
				disc := domain.Discrepancy{
					ID:               uuid.New(),
					PayoutID:         stringPtr(pid),
					DiscrepancyLevel: domain.DiscrepancyLevel2PGBank,
					DiscrepancyType:  domain.TypePayoutShortfall,
					ExpectedAmount:   totalExpectedNet,
					ActualAmount:     matchedBank.CreditAmount,
					MismatchAmount:   shortfall,
					Severity:         domain.SeverityCritical,
					RawEvidence: domain.RawEvidence{
						BatchSettlements: copyPGSettlements(batchItems),
						BankStatement:   matchedBank,
						ExpectedFormula: fmt.Sprintf("Bank Credit (%s) == Expected Net (%s)",
							totalExpectedNet.StringFixed(4), totalExpectedNet.StringFixed(4)),
						ActualFormula: fmt.Sprintf("Bank Credit (%s) < Expected Net (%s) -> Shortfall (%s)",
							matchedBank.CreditAmount.StringFixed(4), totalExpectedNet.StringFixed(4), shortfall.StringFixed(4)),
						CalculatedValues: map[string]interface{}{
							"payout_id":    pid,
							"utr_number":   matchedBank.UTRNumber,
							"expected_net": totalExpectedNet.StringFixed(4),
							"actual_bank":  matchedBank.CreditAmount.StringFixed(4),
							"shortfall":    shortfall.StringFixed(4),
						},
					},
					Status:     domain.StatusOpen,
					DetectedAt: time.Now().UTC(),
				}
				discrepancies = append(discrepancies, disc)
				payoutDiscrepancyMap[pid] = &disc
				group.Discrepancy = &disc
			} else if diff.GreaterThan(e.config.FeeTolerance) {
				// Surplus
				surplus := diff
				group.Status = "SURPLUS"
				disc := domain.Discrepancy{
					ID:               uuid.New(),
					PayoutID:         stringPtr(pid),
					DiscrepancyLevel: domain.DiscrepancyLevel2PGBank,
					DiscrepancyType:  domain.TypePayoutSurplus,
					ExpectedAmount:   totalExpectedNet,
					ActualAmount:     matchedBank.CreditAmount,
					MismatchAmount:   surplus,
					Severity:         domain.SeverityMedium,
					RawEvidence: domain.RawEvidence{
						BatchSettlements: copyPGSettlements(batchItems),
						BankStatement:   matchedBank,
						ExpectedFormula: fmt.Sprintf("Bank Credit (%s) == Expected Net (%s)",
							totalExpectedNet.StringFixed(4), totalExpectedNet.StringFixed(4)),
						ActualFormula: fmt.Sprintf("Bank Credit (%s) > Expected Net (%s) -> Surplus (%s)",
							matchedBank.CreditAmount.StringFixed(4), totalExpectedNet.StringFixed(4), surplus.StringFixed(4)),
						CalculatedValues: map[string]interface{}{
							"payout_id":    pid,
							"utr_number":   matchedBank.UTRNumber,
							"expected_net": totalExpectedNet.StringFixed(4),
							"actual_bank":  matchedBank.CreditAmount.StringFixed(4),
							"surplus":      surplus.StringFixed(4),
						},
					},
					Status:     domain.StatusOpen,
					DetectedAt: time.Now().UTC(),
				}
				discrepancies = append(discrepancies, disc)
				payoutDiscrepancyMap[pid] = &disc
				group.Discrepancy = &disc
			}
		}

		payoutBatches = append(payoutBatches, group)
	}

	emit("SYNTHESIS", 80.0, "Constructing unified 3-way trace items and computing KPIs", nil)

	// =========================================================================
	// STAGE 3: UNIFIED 3-WAY TRACE ITEMS
	// =========================================================================
	var traceItems []domain.ThreeWayTraceItem
	for _, o := range orders {
		trace := domain.ThreeWayTraceItem{
			OrderID:     o.OrderID,
			OrderGross:  &o.GrossAmount,
			OrderStatus: &o.OrderStatus,
		}

		pg, pgFound := pgByOrderMap[o.OrderID]
		if pgFound {
			trace.PGTxnID = stringPtr(pg.PGTxnID)
			trace.PGGross = &pg.GrossAmount
			trace.PGFee = &pg.PGFee
			trace.PGTax = &pg.Tax
			trace.PGNet = &pg.NetAmount
			trace.PGPayoutID = stringPtr(pg.PayoutID)
			trace.PGPayoutStatus = &pg.PayoutStatus

			if bank, bankFound := payoutBankCreditMap[pg.PayoutID]; bankFound {
				trace.BankTxnID = stringPtr(bank.BankTxnID)
				trace.BankUTR = stringPtr(bank.UTRNumber)
				trace.BankCredit = &bank.CreditAmount
			}
		}

		// Attach discrepancy if present
		if d, ok := orderDiscrepancyMap[o.OrderID]; ok {
			trace.Discrepancy = d
			trace.IsReconciled = false
		} else if pgFound && payoutDiscrepancyMap[pg.PayoutID] != nil {
			trace.Discrepancy = payoutDiscrepancyMap[pg.PayoutID]
			trace.IsReconciled = false
		} else {
			trace.IsReconciled = pgFound && trace.BankCredit != nil
		}

		traceItems = append(traceItems, trace)
	}

	// =========================================================================
	// STAGE 4: EXECUTIVE SUMMARY KPI CALCULATION
	// =========================================================================
	var totalGrossOrderVolume, totalPGFees, totalPGTax, expectedNetPayout, totalBankCredits, netVariance money.Decimal
	var reconciledCount, unreconciledCount int
	var l1Count, l2Count int

	for _, o := range orders {
		totalGrossOrderVolume = totalGrossOrderVolume.Add(o.GrossAmount)
	}
	for _, s := range settlements {
		totalPGFees = totalPGFees.Add(s.PGFee)
		totalPGTax = totalPGTax.Add(s.Tax)
		expectedNetPayout = expectedNetPayout.Add(s.NetAmount)
	}
	for _, b := range bankStatements {
		totalBankCredits = totalBankCredits.Add(b.CreditAmount)
	}

	for _, d := range discrepancies {
		netVariance = netVariance.Add(d.MismatchAmount)
		if d.DiscrepancyLevel == domain.DiscrepancyLevel1OrderPG {
			l1Count++
		} else if d.DiscrepancyLevel == domain.DiscrepancyLevel2PGBank {
			l2Count++
		}
	}

	for _, t := range traceItems {
		if t.IsReconciled {
			reconciledCount++
		} else {
			unreconciledCount++
		}
	}

	var matchRate money.Decimal
	if len(traceItems) > 0 {
		rateFloat := float64(reconciledCount) / float64(len(traceItems)) * 100.0
		matchRate = money.FromFloat(rateFloat)
	} else {
		matchRate = money.FromInt(100)
	}

	summary := domain.ReconciliationSummary{
		TotalOrders:              len(orders),
		TotalGrossOrderVolume:    totalGrossOrderVolume,
		TotalPGSettlements:       len(settlements),
		TotalPGFees:              totalPGFees,
		TotalPGTax:               totalPGTax,
		ExpectedNetPayout:        expectedNetPayout,
		TotalBankCredits:         totalBankCredits,
		NetDiscrepancyVariance:   netVariance,
		Level1DiscrepanciesCount: l1Count,
		Level2DiscrepanciesCount: l2Count,
		ThreeWayMatchRate:        matchRate,
		ReconciledOrdersCount:    reconciledCount,
		UnreconciledOrdersCount:  unreconciledCount,
	}

	emit("COMPLETED", 100.0, fmt.Sprintf("Reconciliation complete. Match Rate: %s%%, Discrepancies: %d", matchRate.StringFixed(2), len(discrepancies)), summary)

	return &ReconciliationResult{
		Discrepancies: discrepancies,
		TraceItems:    traceItems,
		PayoutBatches: payoutBatches,
		Summary:       summary,
	}, nil
}

// Helpers

func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func copyOrder(o domain.Order) *domain.Order {
	c := o
	return &c
}

func copyPG(s domain.PGSettlement) *domain.PGSettlement {
	c := s
	return &c
}

func copyPGSettlements(items []domain.PGSettlement) []*domain.PGSettlement {
	res := make([]*domain.PGSettlement, len(items))
	for i := range items {
		it := items[i]
		res[i] = &it
	}
	return res
}

