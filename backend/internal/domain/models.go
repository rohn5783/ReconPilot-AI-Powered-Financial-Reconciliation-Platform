package domain

import (
	"time"

	"github.com/google/uuid"

	"reconpilot/pkg/money"
)

// OrderStatus defines the lifecycle state of an internal merchant order.
type OrderStatus string

const (
	OrderStatusCaptured OrderStatus = "CAPTURED"
	OrderStatusPending  OrderStatus = "PENDING"
	OrderStatusFailed   OrderStatus = "FAILED"
	OrderStatusRefunded OrderStatus = "REFUNDED"
)

// Order represents an internal merchant e-commerce order record.
type Order struct {
	OrderID     string        `json:"order_id" db:"order_id"`
	CustomerID  string        `json:"customer_id" db:"customer_id"`
	GrossAmount money.Decimal `json:"gross_amount" db:"gross_amount"`
	Currency    string        `json:"currency" db:"currency"`
	OrderStatus OrderStatus   `json:"order_status" db:"order_status"`
	CreatedAt   time.Time     `json:"created_at" db:"created_at"`
}

// PayoutStatus defines payment gateway payout settlement state.
type PayoutStatus string

const (
	PayoutStatusSettled   PayoutStatus = "SETTLED"
	PayoutStatusProcessed PayoutStatus = "PROCESSED"
	PayoutStatusPending   PayoutStatus = "PENDING"
	PayoutStatusFailed    PayoutStatus = "FAILED"
)

// PGSettlement represents a payment gateway settlement transaction (Razorpay, Stripe, etc.).
type PGSettlement struct {
	PGTxnID      string        `json:"pg_txn_id" db:"pg_txn_id"`
	OrderID      string        `json:"order_id" db:"order_id"`
	GrossAmount  money.Decimal `json:"gross_amount" db:"gross_amount"`
	PGFee        money.Decimal `json:"pg_fee" db:"pg_fee"`
	Tax          money.Decimal `json:"tax" db:"tax"`
	NetAmount    money.Decimal `json:"net_amount" db:"net_amount"`
	PayoutID     string        `json:"payout_id" db:"payout_id"`
	PayoutStatus PayoutStatus  `json:"payout_status" db:"payout_status"`
	SettledAt    time.Time     `json:"settled_at" db:"settled_at"`
}

// BankStatement represents a credited line item from the merchant's bank account (HDFC, ICICI, etc.).
type BankStatement struct {
	BankTxnID       string        `json:"bank_txn_id" db:"bank_txn_id"`
	UTRNumber       string        `json:"utr_number" db:"utr_number"`
	CreditAmount    money.Decimal `json:"credit_amount" db:"credit_amount"`
	Description     string        `json:"description" db:"description"`
	TransactionDate time.Time     `json:"transaction_date" db:"transaction_date"`
}

// DiscrepancyLevel categorizes which stage of the 3-way reconciliation detected the mismatch.
type DiscrepancyLevel string

const (
	DiscrepancyLevel1OrderPG DiscrepancyLevel = "LEVEL_1_ORDER_PG"
	DiscrepancyLevel2PGBank  DiscrepancyLevel = "LEVEL_2_PG_BANK"
)

// DiscrepancyType enumerates exact financial divergence classifications.
type DiscrepancyType string

const (
	// Level 1: Order <-> Payment Gateway
	TypeFeeOvercharge         DiscrepancyType = "FEE_OVERCHARGE"
	TypeTaxCalculationError   DiscrepancyType = "TAX_CALCULATION_ERROR"
	TypeOrderMissingInPG      DiscrepancyType = "ORDER_MISSING_IN_PG"
	TypePGOrphanTxn           DiscrepancyType = "PG_ORPHAN_TXN"
	TypeCaptureStatusMismatch DiscrepancyType = "CAPTURE_STATUS_MISMATCH"
	TypeGrossAmountMismatch   DiscrepancyType = "GROSS_AMOUNT_MISMATCH"

	// Level 2: Payment Gateway Payout <-> Bank Statement (UTR)
	TypeUnsettledPayout DiscrepancyType = "UNSETTLED_PAYOUT"
	TypeBankMissing     DiscrepancyType = "BANK_MISSING"
	TypePayoutShortfall DiscrepancyType = "PAYOUT_SHORTFALL"
	TypePayoutSurplus   DiscrepancyType = "PAYOUT_SURPLUS"
)

// DiscrepancySeverity categorizes financial urgency.
type DiscrepancySeverity string

const (
	SeverityCritical DiscrepancySeverity = "CRITICAL"
	SeverityHigh     DiscrepancySeverity = "HIGH"
	SeverityMedium   DiscrepancySeverity = "MEDIUM"
	SeverityLow      DiscrepancySeverity = "LOW"
)

// DiscrepancyStatus represents operational resolution state.
type DiscrepancyStatus string

const (
	StatusOpen        DiscrepancyStatus = "OPEN"
	StatusInDispute   DiscrepancyStatus = "IN_DISPUTE"
	StatusResolved    DiscrepancyStatus = "RESOLVED"
	StatusWrittenOff  DiscrepancyStatus = "WRITTEN_OFF"
)

// RawEvidence captures the exact state snapshots for financial audibility and formula verification.
type RawEvidence struct {
	Order            *Order                 `json:"order,omitempty"`
	PGSettlement     *PGSettlement          `json:"pg_settlement,omitempty"`
	BankStatement    *BankStatement         `json:"bank_statement,omitempty"`
	BatchSettlements []*PGSettlement        `json:"batch_settlements,omitempty"`
	ExpectedFormula  string                 `json:"expected_formula"`
	ActualFormula    string                 `json:"actual_formula"`
	CalculatedValues map[string]interface{} `json:"calculated_values"`
}

// AIDiagnosis captures the structured output synthesized by Google Gemini 2.5 Flash.
type AIDiagnosis struct {
	RootCauseSummary    string   `json:"root_cause_summary"`
	DiscrepancyCategory string   `json:"discrepancy_category"`
	ConfidenceScore     int      `json:"confidence_score"` // 0 - 100
	RecommendedActions  []string `json:"recommended_actions"`
	DraftDisputeTicket  string   `json:"draft_dispute_ticket"`
}

// Discrepancy represents a detected financial variance across the 3-way reconciliation pipeline.
type Discrepancy struct {
	ID               uuid.UUID           `json:"id" db:"id"`
	OrderID          *string             `json:"order_id,omitempty" db:"order_id"`
	PayoutID         *string             `json:"payout_id,omitempty" db:"payout_id"`
	DiscrepancyLevel DiscrepancyLevel    `json:"discrepancy_level" db:"discrepancy_level"`
	DiscrepancyType  DiscrepancyType     `json:"discrepancy_type" db:"discrepancy_type"`
	ExpectedAmount   money.Decimal       `json:"expected_amount" db:"expected_amount"`
	ActualAmount     money.Decimal       `json:"actual_amount" db:"actual_amount"`
	MismatchAmount   money.Decimal       `json:"mismatch_amount" db:"mismatch_amount"`
	Severity         DiscrepancySeverity `json:"severity" db:"severity"`
	RawEvidence      RawEvidence         `json:"raw_evidence" db:"raw_evidence"`
	AIDiagnosis      *AIDiagnosis        `json:"ai_diagnosis,omitempty" db:"ai_diagnosis"`
	Status           DiscrepancyStatus   `json:"status" db:"status"`
	DetectedAt       time.Time           `json:"detected_at" db:"detected_at"`
}

// ThreeWayTraceItem represents an unified 3-column financial ledger trace entry for UI exploration:
// [Internal Order] -> [PG Settlement (Gross - Fee - Tax = Net)] -> [Bank Statement UTR]
type ThreeWayTraceItem struct {
	OrderID        string         `json:"order_id"`
	OrderGross     *money.Decimal `json:"order_gross,omitempty"`
	OrderStatus    *OrderStatus   `json:"order_status,omitempty"`
	PGTxnID        *string        `json:"pg_txn_id,omitempty"`
	PGGross        *money.Decimal `json:"pg_gross,omitempty"`
	PGFee          *money.Decimal `json:"pg_fee,omitempty"`
	PGTax          *money.Decimal `json:"pg_tax,omitempty"`
	PGNet          *money.Decimal `json:"pg_net,omitempty"`
	PGPayoutID     *string        `json:"pg_payout_id,omitempty"`
	PGPayoutStatus *PayoutStatus  `json:"pg_payout_status,omitempty"`
	BankTxnID      *string        `json:"bank_txn_id,omitempty"`
	BankUTR        *string        `json:"bank_utr,omitempty"`
	BankCredit     *money.Decimal `json:"bank_credit,omitempty"`
	IsReconciled   bool           `json:"is_reconciled"`
	Discrepancy    *Discrepancy   `json:"discrepancy,omitempty"`
}

// PayoutBatchGroup aggregates PG settlements grouped by PayoutID/UTR for Level-2 bank matching.
type PayoutBatchGroup struct {
	PayoutID         string         `json:"payout_id"`
	SettlementsCount int            `json:"settlements_count"`
	TotalGross       money.Decimal  `json:"total_gross"`
	TotalFee         money.Decimal  `json:"total_fee"`
	TotalTax         money.Decimal  `json:"total_tax"`
	TotalExpectedNet money.Decimal  `json:"total_expected_net"`
	BankCredit       *money.Decimal `json:"bank_credit,omitempty"`
	BankUTR          *string        `json:"bank_utr,omitempty"`
	Status           string         `json:"status"` // MATCHED, SHORTFALL, SURPLUS, BANK_MISSING
	Discrepancy      *Discrepancy   `json:"discrepancy,omitempty"`
}

// ReconciliationSummary provides high-density financial KPIs for the workstation header.
type ReconciliationSummary struct {
	TotalOrders              int           `json:"total_orders"`
	TotalGrossOrderVolume    money.Decimal `json:"total_gross_order_volume"`
	TotalPGSettlements       int           `json:"total_pg_settlements"`
	TotalPGFees              money.Decimal `json:"total_pg_fees"`
	TotalPGTax               money.Decimal `json:"total_pg_tax"`
	ExpectedNetPayout        money.Decimal `json:"expected_net_payout"`
	TotalBankCredits         money.Decimal `json:"total_bank_credits"`
	NetDiscrepancyVariance   money.Decimal `json:"net_discrepancy_variance"`
	Level1DiscrepanciesCount int           `json:"level_1_discrepancies_count"`
	Level2DiscrepanciesCount int           `json:"level_2_discrepancies_count"`
	ThreeWayMatchRate        money.Decimal `json:"three_way_match_rate"` // e.g. 98.45%
	ReconciledOrdersCount    int           `json:"reconciled_orders_count"`
	UnreconciledOrdersCount  int           `json:"unreconciled_orders_count"`
}

// PipelineTelemetryEvent carries real-time streaming SSE/WebSocket progress and terminal updates.
type PipelineTelemetryEvent struct {
	Type               string      `json:"type"` // "progress", "log", "kpi_update", "discrepancy", "completed"
	Stage              string      `json:"stage"` // "INGESTION", "LEVEL_1_MATCH", "LEVEL_2_MATCH", "AI_DIAGNOSIS", "COMPLETED"
	ProgressPercentage float64     `json:"progress_percentage"`
	Message            string      `json:"message"`
	Payload            interface{} `json:"payload,omitempty"`
	Timestamp          time.Time   `json:"timestamp"`
}

