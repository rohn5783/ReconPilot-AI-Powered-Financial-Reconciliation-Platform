package service

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"time"

	"reconpilot/internal/domain"
)

// ExportService generates institutional audit files (CSV/JSON) for reconciliation results.
type ExportService struct{}

// NewExportService creates an ExportService.
func NewExportService() *ExportService {
	return &ExportService{}
}

// ExportTracesCSV exports 3-way trace items to CSV.
func (s *ExportService) ExportTracesCSV(traces []domain.ThreeWayTraceItem) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	headers := []string{
		"Order ID",
		"Order Gross",
		"Order Status",
		"PG Txn ID",
		"PG Gross",
		"PG Fee",
		"PG Tax",
		"PG Net",
		"PG Payout ID",
		"PG Status",
		"Bank Txn ID",
		"Bank UTR",
		"Bank Credit",
		"Reconciliation Status",
		"Discrepancy Type",
		"Discrepancy Variance",
	}
	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("write csv headers: %w", err)
	}

	for _, t := range traces {
		orderGross := ""
		if t.OrderGross != nil {
			orderGross = t.OrderGross.StringFixed(4)
		}
		orderStatus := ""
		if t.OrderStatus != nil {
			orderStatus = string(*t.OrderStatus)
		}

		pgTxnID := ""
		if t.PGTxnID != nil {
			pgTxnID = *t.PGTxnID
		}
		pgGross := ""
		if t.PGGross != nil {
			pgGross = t.PGGross.StringFixed(4)
		}
		pgFee := ""
		if t.PGFee != nil {
			pgFee = t.PGFee.StringFixed(4)
		}
		pgTax := ""
		if t.PGTax != nil {
			pgTax = t.PGTax.StringFixed(4)
		}
		pgNet := ""
		if t.PGNet != nil {
			pgNet = t.PGNet.StringFixed(4)
		}
		pgPayoutID := ""
		if t.PGPayoutID != nil {
			pgPayoutID = *t.PGPayoutID
		}
		pgStatus := ""
		if t.PGPayoutStatus != nil {
			pgStatus = string(*t.PGPayoutStatus)
		}

		bankTxnID := ""
		if t.BankTxnID != nil {
			bankTxnID = *t.BankTxnID
		}
		bankUTR := ""
		if t.BankUTR != nil {
			bankUTR = *t.BankUTR
		}
		bankCredit := ""
		if t.BankCredit != nil {
			bankCredit = t.BankCredit.StringFixed(4)
		}

		reconStatus := "RECONCILED"
		discType := ""
		discVariance := ""
		if !t.IsReconciled {
			reconStatus = "DISCREPANCY"
			if t.Discrepancy != nil {
				discType = string(t.Discrepancy.DiscrepancyType)
				discVariance = t.Discrepancy.MismatchAmount.StringFixed(4)
			}
		}

		row := []string{
			t.OrderID,
			orderGross,
			orderStatus,
			pgTxnID,
			pgGross,
			pgFee,
			pgTax,
			pgNet,
			pgPayoutID,
			pgStatus,
			bankTxnID,
			bankUTR,
			bankCredit,
			reconStatus,
			discType,
			discVariance,
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("write csv row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("flush csv writer: %w", err)
	}

	return buf.Bytes(), nil
}

// ExportDiscrepanciesCSV exports detailed discrepancy audit records.
func (s *ExportService) ExportDiscrepanciesCSV(discrepancies []domain.Discrepancy) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	headers := []string{
		"Discrepancy ID",
		"Order ID",
		"Payout ID",
		"Level",
		"Classification",
		"Severity",
		"Status",
		"Expected Amount",
		"Actual Amount",
		"Mismatch Variance",
		"Detected At",
		"AI Category",
		"AI Confidence",
		"AI Root Cause Summary",
	}
	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("write csv headers: %w", err)
	}

	for _, d := range discrepancies {
		orderID := ""
		if d.OrderID != nil {
			orderID = *d.OrderID
		}
		payoutID := ""
		if d.PayoutID != nil {
			payoutID = *d.PayoutID
		}

		aiCat := ""
		aiConf := ""
		aiSummary := ""
		if d.AIDiagnosis != nil {
			aiCat = d.AIDiagnosis.DiscrepancyCategory
			aiConf = fmt.Sprintf("%d%%", d.AIDiagnosis.ConfidenceScore)
			aiSummary = d.AIDiagnosis.RootCauseSummary
		}

		row := []string{
			d.ID.String(),
			orderID,
			payoutID,
			string(d.DiscrepancyLevel),
			string(d.DiscrepancyType),
			string(d.Severity),
			string(d.Status),
			d.ExpectedAmount.StringFixed(4),
			d.ActualAmount.StringFixed(4),
			d.MismatchAmount.StringFixed(4),
			d.DetectedAt.Format(time.RFC3339),
			aiCat,
			aiConf,
			aiSummary,
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("write csv row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("flush csv writer: %w", err)
	}

	return buf.Bytes(), nil
}

// ExportJSON exports complete structured audit payload.
func (s *ExportService) ExportJSON(payload interface{}) ([]byte, error) {
	return json.MarshalIndent(payload, "", "  ")
}

