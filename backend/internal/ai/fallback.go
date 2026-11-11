package ai

import (
	"fmt"

	"reconpilot/internal/domain"
)

// RuleBasedDiagnose provides an institutional deterministic diagnosis engine that activates
// when Gemini API key is unconfigured, rate-limited, or in offline MOCK_MODE.
func RuleBasedDiagnose(d *domain.Discrepancy) *domain.AIDiagnosis {
	if d == nil {
		return nil
	}

	switch d.DiscrepancyType {
	case domain.TypeFeeOvercharge:
		orderID := "N/A"
		if d.OrderID != nil {
			orderID = *d.OrderID
		}
		payoutID := "N/A"
		if d.PayoutID != nil {
			payoutID = *d.PayoutID
		}
		mismatchStr := d.MismatchAmount.StringFixed(4)

		return &domain.AIDiagnosis{
			RootCauseSummary: fmt.Sprintf("Payment Gateway applied an unauthorized fee schedule exceeding the agreed contract rate, resulting in an unapproved deduction of ₹%s on Order %s.",
				mismatchStr, orderID),
			DiscrepancyCategory: "Gateway Fee Schedule Drift / Contract Overbilling",
			ConfidenceScore:     98,
			RecommendedActions: []string{
				"File fee reversal claim with Payment Gateway Merchant Operations citing negotiated Merchant Service Agreement.",
				fmt.Sprintf("Request credit adjustment note of ₹%s against current payout batch %s.", mismatchStr, payoutID),
				"Review automated pricing webhooks and card interchange tier mapping.",
			},
			DraftDisputeTicket: fmt.Sprintf("MERCHANT DISPUTE NOTICE: MDR Fee Overcharge\n"+
				"--------------------------------------------------\n"+
				"Order Reference: %s\n"+
				"Payout Batch: %s\n"+
				"Expected Net: ₹%s | Actual Settled: ₹%s\n"+
				"Disputed Overcharge Amount: ₹%s\n"+
				"Evidence: Gateway MDR fee deducted exceeds negotiated contract rate.\n"+
				"Resolution Requested: Credit note of ₹%s within 48 hours to merchant nodal account.",
				orderID, payoutID, d.ExpectedAmount.StringFixed(4), d.ActualAmount.StringFixed(4), mismatchStr, mismatchStr),
		}

	case domain.TypeTaxCalculationError:
		orderID := "N/A"
		if d.OrderID != nil {
			orderID = *d.OrderID
		}
		return &domain.AIDiagnosis{
			RootCauseSummary: fmt.Sprintf("Payment Gateway GST tax invoice calculation deviates from statutory 18.00%% rate on MDR charges for Order %s by ₹%s.",
				orderID, d.MismatchAmount.StringFixed(4)),
			DiscrepancyCategory: "Statutory Tax Invoice Inconsistency",
			ConfidenceScore:     94,
			RecommendedActions: []string{
				"Request revised GSTR-1 Annexure debit/credit note from payment gateway.",
				"Reconcile inward Input Tax Credit (ITC) ledger before monthly GST return filing.",
			},
			DraftDisputeTicket: fmt.Sprintf("TAX RECONCILIATION NOTICE: Inconsistent GST Calculation\n"+
				"Order ID: %s | Variance: ₹%s\n"+
				"Expected Statutory GST: ₹%s | Billed: ₹%s\n"+
				"Please re-issue monthly tax invoice with corrected GST breakdown.",
				orderID, d.MismatchAmount.StringFixed(4), d.ExpectedAmount.StringFixed(4), d.ActualAmount.StringFixed(4)),
		}

	case domain.TypeOrderMissingInPG:
		orderID := "N/A"
		if d.OrderID != nil {
			orderID = *d.OrderID
		}
		return &domain.AIDiagnosis{
			RootCauseSummary: fmt.Sprintf("Merchant order %s was marked CAPTURED internally, but no corresponding settlement or authorization transaction exists in gateway records. Likely dropped webhook or uncaptured customer pre-auth.",
				orderID),
			DiscrepancyCategory: "Unsettled Merchant Volume / Dropped Gateway Webhook",
			ConfidenceScore:     95,
			RecommendedActions: []string{
				fmt.Sprintf("Query Payment Gateway Payments API by merchant receipt ID %s.", orderID),
				"If payment failed at PG, mark internal order FAILED and cancel shipment dispatch.",
				"If payment was authorized, trigger manual backfill settlement capture.",
			},
			DraftDisputeTicket: fmt.Sprintf("TRANSACTION INVESTIGATION REQUEST\n"+
				"Order ID: %s\n"+
				"Captured Amount: ₹%s\n"+
				"Status: Internal ledger shows CAPTURED, Gateway shows NO RECORD.\n"+
				"Please verify gateway authorization status or initiate settlement ingest.",
				orderID, d.ExpectedAmount.StringFixed(4)),
		}

	case domain.TypePGOrphanTxn:
		orderID := "N/A"
		if d.OrderID != nil {
			orderID = *d.OrderID
		}
		return &domain.AIDiagnosis{
			RootCauseSummary: fmt.Sprintf("Payment gateway captured ₹%s for reference %s without a corresponding record in the merchant order database. Likely orphaned checkout session or payment link payment.",
				d.ActualAmount.StringFixed(4), orderID),
			DiscrepancyCategory: "Orphan Payment Gateway Capture / Unlinked Receipt",
			ConfidenceScore:     91,
			RecommendedActions: []string{
				"Cross-reference customer email and phone in gateway transaction metadata against customer directory.",
				"Create manual back-office order record or issue customer refund if fulfillment cannot be completed.",
			},
			DraftDisputeTicket: fmt.Sprintf("INTERNAL AUDIT NOTICE: Unlinked Gateway Capture\n"+
				"Gateway Reference: %s\n"+
				"Amount Captured: ₹%s\n"+
				"Action: Identify customer order context or reverse funds.",
				orderID, d.ActualAmount.StringFixed(4)),
		}

	case domain.TypeCaptureStatusMismatch:
		orderID := "N/A"
		if d.OrderID != nil {
			orderID = *d.OrderID
		}
		return &domain.AIDiagnosis{
			RootCauseSummary: fmt.Sprintf("Internal order %s was marked REFUNDED or FAILED, but payment gateway retained and settled funds of ₹%s. Refund API call may have failed or timed out.",
				orderID, d.ActualAmount.StringFixed(4)),
			DiscrepancyCategory: "Unexecuted Refund / Double-Retention Hazard",
			ConfidenceScore:     96,
			RecommendedActions: []string{
				fmt.Sprintf("Initiate immediate refund via Gateway API for Order %s.", orderID),
				"Verify customer bank account has not initiated a chargeback dispute.",
			},
			DraftDisputeTicket: fmt.Sprintf("ACTION REQUIRED: Execute Pending Customer Refund\n"+
				"Order ID: %s | Amount: ₹%s\n"+
				"Internal Status: REFUNDED | Gateway Status: SETTLED\n"+
				"Execute immediate outward refund to prevent customer chargeback escalation.",
				orderID, d.ActualAmount.StringFixed(4)),
		}

	case domain.TypeGrossAmountMismatch:
		orderID := "N/A"
		if d.OrderID != nil {
			orderID = *d.OrderID
		}
		return &domain.AIDiagnosis{
			RootCauseSummary: fmt.Sprintf("Gross order volume of ₹%s differs from gateway settled gross of ₹%s (variance: ₹%s). Likely promotional discount misallocation or partial capture.",
				d.ExpectedAmount.StringFixed(4), d.ActualAmount.StringFixed(4), d.MismatchAmount.StringFixed(4)),
			DiscrepancyCategory: "Promotional Code Accounting Mismatch / Partial Capture",
			ConfidenceScore:     90,
			RecommendedActions: []string{
				"Verify whether coupon code or merchant discount voucher was applied at checkout.",
				"Post promotional expense journal entry to balance difference against marketing budget.",
			},
			DraftDisputeTicket: fmt.Sprintf("ORDER RECONCILIATION NOTICE: Gross Amount Variance\n"+
				"Order ID: %s\n"+
				"Order Gross: ₹%s | Gateway Gross: ₹%s | Variance: ₹%s\n"+
				"Reconcile promotional subsidy or partial capture ledger.",
				orderID, d.ExpectedAmount.StringFixed(4), d.ActualAmount.StringFixed(4), d.MismatchAmount.StringFixed(4)),
		}

	case domain.TypeBankMissing:
		payoutID := "N/A"
		if d.PayoutID != nil {
			payoutID = *d.PayoutID
		}
		return &domain.AIDiagnosis{
			RootCauseSummary: fmt.Sprintf("Payment Gateway reports payout batch %s of ₹%s was settled, but bank statements show NO credited remittance. Potential nodal clearing failure or incorrect beneficiary IFSC/account.",
				payoutID, d.ExpectedAmount.StringFixed(4)),
			DiscrepancyCategory: "Uncredited Bank Remittance / Nodal Clearing Failure",
			ConfidenceScore:     97,
			RecommendedActions: []string{
				fmt.Sprintf("Request NEFT/RTGS Bank Acknowledgement Slip from Gateway Treasury Desk for Payout %s.", payoutID),
				"Lodge outward remittance investigation with corporate banking partner.",
			},
			DraftDisputeTicket: fmt.Sprintf("CRITICAL REMITTANCE INQUIRY: Missing Payout Credit\n"+
				"Payout Batch ID: %s\n"+
				"Settled Net Expected: ₹%s\n"+
				"Bank Statement Status: ZERO credit found.\n"+
				"Please furnish Bank Reference Number (UTR) and clearing confirmation slip immediately.",
				payoutID, d.ExpectedAmount.StringFixed(4)),
		}

	case domain.TypePayoutShortfall:
		payoutID := "N/A"
		if d.PayoutID != nil {
			payoutID = *d.PayoutID
		}
		return &domain.AIDiagnosis{
			RootCauseSummary: fmt.Sprintf("Payout batch %s received ₹%s in bank account against expected net of ₹%s (shortfall of ₹%s). Unnotified rolling security reserve retention or chargeback recovery deduction.",
				payoutID, d.ActualAmount.StringFixed(4), d.ExpectedAmount.StringFixed(4), d.MismatchAmount.StringFixed(4)),
			DiscrepancyCategory: "Gateway Settlement Clawback / Rolling Reserve Deduction",
			ConfidenceScore:     93,
			RecommendedActions: []string{
				fmt.Sprintf("Download line-item settlement deduction annexure for Payout %s.", payoutID),
				"Verify whether rolling reserve threshold was revised by gateway risk committee.",
			},
			DraftDisputeTicket: fmt.Sprintf("SETTLEMENT SHORTFALL DISPUTE\n"+
				"Payout Batch: %s\n"+
				"Expected Net: ₹%s | Received: ₹%s | Shortfall: ₹%s\n"+
				"Please furnish detailed deduction ledger explaining the ₹%s variance.",
				payoutID, d.ExpectedAmount.StringFixed(4), d.ActualAmount.StringFixed(4), d.MismatchAmount.StringFixed(4), d.MismatchAmount.StringFixed(4)),
		}

	case domain.TypePayoutSurplus:
		payoutID := "N/A"
		if d.PayoutID != nil {
			payoutID = *d.PayoutID
		}
		return &domain.AIDiagnosis{
			RootCauseSummary: fmt.Sprintf("Bank credited ₹%s for payout batch %s exceeding expected settlement of ₹%s by ₹%s. Likely delayed reversal credit or prior period settlement adjustment.",
				d.ActualAmount.StringFixed(4), payoutID, d.ExpectedAmount.StringFixed(4), d.MismatchAmount.StringFixed(4)),
			DiscrepancyCategory: "Unidentified Surplus Remittance / Retroactive Credit",
			ConfidenceScore:     88,
			RecommendedActions: []string{
				"Credit suspense clearing account pending gateway monthly reconciliation.",
				"Verify whether prior chargeback dispute was won and re-credited.",
			},
			DraftDisputeTicket: fmt.Sprintf("REMITTANCE SURPLUS ADVICE\n"+
				"Payout Batch: %s | Surplus Credit: ₹%s\n"+
				"Please confirm purpose of excess credit for financial auditing.",
				payoutID, d.MismatchAmount.StringFixed(4)),
		}

	default:
		return &domain.AIDiagnosis{
			RootCauseSummary:    fmt.Sprintf("Financial discrepancy detected with variance of ₹%s. Manual forensic inspection advised.", d.MismatchAmount.StringFixed(4)),
			DiscrepancyCategory: "General Ledger Variance",
			ConfidenceScore:     85,
			RecommendedActions: []string{
				"Inspect raw order, settlement, and bank transaction payloads.",
				"Review manual journal entries.",
			},
			DraftDisputeTicket: fmt.Sprintf("Audit Notice: Discrepancy %s requires manual ledger reconciliation.", d.ID),
		}
	}
}

