package ingestion

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"reconpilot/internal/domain"
	"reconpilot/pkg/money"
	"reconpilot/pkg/ringbuffer"
)

var (
	ErrEmptyFile        = errors.New("input file is empty")
	ErrMissingHeaders   = errors.New("missing required headers")
	ErrInvalidRecord    = errors.New("invalid record format")
	ErrStrictValidation = errors.New("strict schema validation failure")
)

// Supported date formats for automated banking and gateway timestamp parsing.
var supportedDateLayouts = []string{
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02T15:04:05Z",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
	"2006-01-02",
	"02/01/2006 15:04:05",
	"02/01/2006 15:04",
	"02/01/2006",
	"02-01-2006 15:04:05",
	"02-01-2006",
	"01/02/2006 15:04:05",
	"01/02/2006",
}

// ParseFlexibleDate tries all supported date layouts.
func ParseFlexibleDate(s string) (time.Time, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return time.Now().UTC(), nil
	}

	for _, layout := range supportedDateLayouts {
		if t, err := time.Parse(layout, trimmed); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse date string %q with supported layouts", s)
}

func normalizeHeader(h string) string {
	cleaned := strings.TrimSpace(strings.ToLower(h))
	cleaned = strings.ReplaceAll(cleaned, " ", "_")
	cleaned = strings.ReplaceAll(cleaned, "-", "_")
	cleaned = strings.ReplaceAll(cleaned, "\"", "")
	return cleaned
}

// ============================================================================
// 1. ORDERS CSV PARSER
// ============================================================================

// ParseOrdersCSV reads and parses an Orders CSV into a slice of domain.Order.
func ParseOrdersCSV(r io.Reader, strictSchema bool) ([]domain.Order, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, ErrEmptyFile
		}
		return nil, fmt.Errorf("read orders csv header: %w", err)
	}

	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[normalizeHeader(h)] = i
	}

	reqHeaders := []string{"order_id", "gross_amount"}
	for _, req := range reqHeaders {
		if _, ok := headerMap[req]; !ok {
			return nil, fmt.Errorf("%w: missing required column '%s'", ErrMissingHeaders, req)
		}
	}

	var orders []domain.Order
	lineNum := 1

	for {
		lineNum++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("orders csv row %d: %w", lineNum, err)
		}

		getCol := func(key string) string {
			if idx, ok := headerMap[key]; ok && idx < len(record) {
				return strings.TrimSpace(record[idx])
			}
			return ""
		}

		orderID := getCol("order_id")
		if orderID == "" {
			if strictSchema {
				return nil, fmt.Errorf("%w: row %d has empty order_id", ErrStrictValidation, lineNum)
			}
			continue
		}

		customerID := getCol("customer_id")
		if customerID == "" && strictSchema {
			return nil, fmt.Errorf("%w: row %d has empty customer_id", ErrStrictValidation, lineNum)
		}

		grossStr := getCol("gross_amount")
		gross, err := money.FromString(grossStr)
		if err != nil {
			if strictSchema {
				return nil, fmt.Errorf("%w: row %d invalid gross_amount %q: %v", ErrStrictValidation, lineNum, grossStr, err)
			}
			gross = money.Zero()
		}

		currency := strings.ToUpper(getCol("currency"))
		if currency == "" {
			currency = "INR"
		}

		statusStr := strings.ToUpper(getCol("order_status"))
		if statusStr == "" {
			statusStr = strings.ToUpper(getCol("status"))
		}
		orderStatus := domain.OrderStatusCaptured
		switch domain.OrderStatus(statusStr) {
		case domain.OrderStatusPending:
			orderStatus = domain.OrderStatusPending
		case domain.OrderStatusFailed:
			orderStatus = domain.OrderStatusFailed
		case domain.OrderStatusRefunded:
			orderStatus = domain.OrderStatusRefunded
		case domain.OrderStatusCaptured:
			orderStatus = domain.OrderStatusCaptured
		default:
			if strictSchema && statusStr != "" {
				return nil, fmt.Errorf("%w: row %d invalid order_status %q", ErrStrictValidation, lineNum, statusStr)
			}
		}

		createdAtStr := getCol("created_at")
		if createdAtStr == "" {
			createdAtStr = getCol("date")
		}
		createdAt, err := ParseFlexibleDate(createdAtStr)
		if err != nil && strictSchema {
			return nil, fmt.Errorf("%w: row %d invalid created_at %q: %v", ErrStrictValidation, lineNum, createdAtStr, err)
		}

		orders = append(orders, domain.Order{
			OrderID:     orderID,
			CustomerID:  customerID,
			GrossAmount: gross,
			Currency:    currency,
			OrderStatus: orderStatus,
			CreatedAt:   createdAt,
		})
	}

	return orders, nil
}

// StreamOrdersCSV streams large orders CSV through a bounded ring buffer in chunks to prevent OOM.
func StreamOrdersCSV(r io.Reader, strictSchema bool, bufferKB int, batchSize int, onBatch func([]domain.Order) error) error {
	rb := ringbuffer.NewFromKB(bufferKB)
	errChan := make(chan error, 1)

	go func() {
		defer rb.Close()
		buf := make([]byte, 32*1024)
		for {
			n, readErr := r.Read(buf)
			if n > 0 {
				if _, writeErr := rb.Write(buf[:n]); writeErr != nil {
					errChan <- writeErr
					return
				}
			}
			if readErr != nil {
				if readErr != io.EOF {
					errChan <- readErr
				}
				return
			}
		}
	}()

	orders, err := ParseOrdersCSV(rb, strictSchema)
	if err != nil {
		return err
	}

	select {
	case writeErr := <-errChan:
		if writeErr != nil {
			return writeErr
		}
	default:
	}

	if batchSize <= 0 {
		batchSize = 500
	}

	for i := 0; i < len(orders); i += batchSize {
		end := i + batchSize
		if end > len(orders) {
			end = len(orders)
		}
		if err := onBatch(orders[i:end]); err != nil {
			return fmt.Errorf("onBatch callback failed: %w", err)
		}
	}
	return nil
}

// ============================================================================
// 2. PAYMENT GATEWAY SETTLEMENTS CSV PARSER (Razorpay, Stripe, etc.)
// ============================================================================

// ParsePGSettlementsCSV reads and parses a PG Settlement CSV.
func ParsePGSettlementsCSV(r io.Reader, strictSchema bool) ([]domain.PGSettlement, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, ErrEmptyFile
		}
		return nil, fmt.Errorf("read pg settlement csv header: %w", err)
	}

	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[normalizeHeader(h)] = i
	}

	reqHeaders := []string{"pg_txn_id", "order_id", "gross_amount"}
	for _, req := range reqHeaders {
		if _, ok := headerMap[req]; !ok {
			return nil, fmt.Errorf("%w: missing required column '%s'", ErrMissingHeaders, req)
		}
	}

	var settlements []domain.PGSettlement
	lineNum := 1

	for {
		lineNum++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("pg settlements csv row %d: %w", lineNum, err)
		}

		getCol := func(key string) string {
			if idx, ok := headerMap[key]; ok && idx < len(record) {
				return strings.TrimSpace(record[idx])
			}
			return ""
		}

		pgTxnID := getCol("pg_txn_id")
		if pgTxnID == "" {
			pgTxnID = getCol("payment_id")
		}
		if pgTxnID == "" {
			if strictSchema {
				return nil, fmt.Errorf("%w: row %d has empty pg_txn_id", ErrStrictValidation, lineNum)
			}
			continue
		}

		orderID := getCol("order_id")
		if orderID == "" && strictSchema {
			return nil, fmt.Errorf("%w: row %d has empty order_id", ErrStrictValidation, lineNum)
		}

		grossStr := getCol("gross_amount")
		if grossStr == "" {
			grossStr = getCol("amount")
		}
		gross, err := money.FromString(grossStr)
		if err != nil {
			if strictSchema {
				return nil, fmt.Errorf("%w: row %d invalid gross_amount %q: %v", ErrStrictValidation, lineNum, grossStr, err)
			}
			gross = money.Zero()
		}

		feeStr := getCol("pg_fee")
		if feeStr == "" {
			feeStr = getCol("fee")
		}
		fee, _ := money.FromString(feeStr)

		taxStr := getCol("tax")
		if taxStr == "" {
			taxStr = getCol("gst")
		}
		tax, _ := money.FromString(taxStr)

		netStr := getCol("net_amount")
		if netStr == "" {
			netStr = getCol("settled_amount")
		}
		net, err := money.FromString(netStr)
		if err != nil || net.IsZero() {
			// If net is missing or zero, compute net = gross - fee - tax
			net = gross.Sub(fee).Sub(tax)
		}

		payoutID := getCol("payout_id")
		if payoutID == "" {
			payoutID = getCol("settlement_id")
		}

		statusStr := strings.ToUpper(getCol("payout_status"))
		if statusStr == "" {
			statusStr = strings.ToUpper(getCol("status"))
		}
		payoutStatus := domain.PayoutStatusSettled
		switch domain.PayoutStatus(statusStr) {
		case domain.PayoutStatusProcessed:
			payoutStatus = domain.PayoutStatusProcessed
		case domain.PayoutStatusPending:
			payoutStatus = domain.PayoutStatusPending
		case domain.PayoutStatusFailed:
			payoutStatus = domain.PayoutStatusFailed
		case domain.PayoutStatusSettled:
			payoutStatus = domain.PayoutStatusSettled
		default:
			if statusStr != "" && strictSchema {
				return nil, fmt.Errorf("%w: row %d invalid payout_status %q", ErrStrictValidation, lineNum, statusStr)
			}
		}

		settledAtStr := getCol("settled_at")
		if settledAtStr == "" {
			settledAtStr = getCol("created_at")
		}
		settledAt, err := ParseFlexibleDate(settledAtStr)
		if err != nil && strictSchema {
			return nil, fmt.Errorf("%w: row %d invalid settled_at %q: %v", ErrStrictValidation, lineNum, settledAtStr, err)
		}

		settlements = append(settlements, domain.PGSettlement{
			PGTxnID:      pgTxnID,
			OrderID:      orderID,
			GrossAmount:  gross,
			PGFee:        fee,
			Tax:          tax,
			NetAmount:    net,
			PayoutID:     payoutID,
			PayoutStatus: payoutStatus,
			SettledAt:    settledAt,
		})
	}

	return settlements, nil
}

// StreamPGSettlementsCSV streams PG settlements CSV in bounded memory chunks.
func StreamPGSettlementsCSV(r io.Reader, strictSchema bool, bufferKB int, batchSize int, onBatch func([]domain.PGSettlement) error) error {
	rb := ringbuffer.NewFromKB(bufferKB)
	errChan := make(chan error, 1)

	go func() {
		defer rb.Close()
		buf := make([]byte, 32*1024)
		for {
			n, readErr := r.Read(buf)
			if n > 0 {
				if _, writeErr := rb.Write(buf[:n]); writeErr != nil {
					errChan <- writeErr
					return
				}
			}
			if readErr != nil {
				if readErr != io.EOF {
					errChan <- readErr
				}
				return
			}
		}
	}()

	settlements, err := ParsePGSettlementsCSV(rb, strictSchema)
	if err != nil {
		return err
	}

	select {
	case writeErr := <-errChan:
		if writeErr != nil {
			return writeErr
		}
	default:
	}

	if batchSize <= 0 {
		batchSize = 500
	}

	for i := 0; i < len(settlements); i += batchSize {
		end := i + batchSize
		if end > len(settlements) {
			end = len(settlements)
		}
		if err := onBatch(settlements[i:end]); err != nil {
			return fmt.Errorf("onBatch callback failed: %w", err)
		}
	}
	return nil
}

// ============================================================================
// 3. BANK STATEMENT CSV PARSER (HDFC, ICICI, etc.)
// ============================================================================

// ParseBankStatementsCSV reads and parses Bank Statement line items.
func ParseBankStatementsCSV(r io.Reader, strictSchema bool) ([]domain.BankStatement, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true
	reader.FieldsPerRecord = -1

	headers, err := reader.Read()
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil, ErrEmptyFile
		}
		return nil, fmt.Errorf("read bank statement csv header: %w", err)
	}

	headerMap := make(map[string]int)
	for i, h := range headers {
		headerMap[normalizeHeader(h)] = i
	}

	reqHeaders := []string{"utr_number", "credit_amount"}
	for _, req := range reqHeaders {
		if _, ok := headerMap[req]; !ok {
			// Check fallback: utr, credit
			if req == "utr_number" && (headerMap["utr"] != 0 || headerMap["reference_no"] != 0) {
				continue
			}
			if req == "credit_amount" && (headerMap["credit"] != 0 || headerMap["amount"] != 0) {
				continue
			}
			return nil, fmt.Errorf("%w: missing required column '%s'", ErrMissingHeaders, req)
		}
	}

	var statements []domain.BankStatement
	lineNum := 1

	for {
		lineNum++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("bank statements csv row %d: %w", lineNum, err)
		}

		getCol := func(keys ...string) string {
			for _, k := range keys {
				if idx, ok := headerMap[k]; ok && idx < len(record) {
					val := strings.TrimSpace(record[idx])
					if val != "" {
						return val
					}
				}
			}
			return ""
		}

		bankTxnID := getCol("bank_txn_id", "transaction_id", "txn_id")
		utrNumber := getCol("utr_number", "utr", "reference_no", "ref_no")
		if utrNumber == "" {
			if strictSchema {
				return nil, fmt.Errorf("%w: row %d has empty utr_number", ErrStrictValidation, lineNum)
			}
			continue
		}

		if bankTxnID == "" {
			bankTxnID = "TXN-" + utrNumber
		}

		creditStr := getCol("credit_amount", "credit", "amount")
		credit, err := money.FromString(creditStr)
		if err != nil {
			if strictSchema {
				return nil, fmt.Errorf("%w: row %d invalid credit_amount %q: %v", ErrStrictValidation, lineNum, creditStr, err)
			}
			credit = money.Zero()
		}

		// Skip rows where credit is 0 if debit-only row in bank statement
		if credit.IsZero() && !strictSchema {
			continue
		}

		description := getCol("description", "narration", "particulars", "remarks")
		dateStr := getCol("transaction_date", "date", "txn_date", "value_date")
		txnDate, err := ParseFlexibleDate(dateStr)
		if err != nil && strictSchema {
			return nil, fmt.Errorf("%w: row %d invalid transaction_date %q: %v", ErrStrictValidation, lineNum, dateStr, err)
		}

		statements = append(statements, domain.BankStatement{
			BankTxnID:       bankTxnID,
			UTRNumber:       utrNumber,
			CreditAmount:    credit,
			Description:     description,
			TransactionDate: txnDate,
		})
	}

	return statements, nil
}

// StreamBankStatementsCSV streams bank statements CSV in bounded memory chunks.
func StreamBankStatementsCSV(r io.Reader, strictSchema bool, bufferKB int, batchSize int, onBatch func([]domain.BankStatement) error) error {
	rb := ringbuffer.NewFromKB(bufferKB)
	errChan := make(chan error, 1)

	go func() {
		defer rb.Close()
		buf := make([]byte, 32*1024)
		for {
			n, readErr := r.Read(buf)
			if n > 0 {
				if _, writeErr := rb.Write(buf[:n]); writeErr != nil {
					errChan <- writeErr
					return
				}
			}
			if readErr != nil {
				if readErr != io.EOF {
					errChan <- readErr
				}
				return
			}
		}
	}()

	statements, err := ParseBankStatementsCSV(rb, strictSchema)
	if err != nil {
		return err
	}

	select {
	case writeErr := <-errChan:
		if writeErr != nil {
			return writeErr
		}
	default:
	}

	if batchSize <= 0 {
		batchSize = 500
	}

	for i := 0; i < len(statements); i += batchSize {
		end := i + batchSize
		if end > len(statements) {
			end = len(statements)
		}
		if err := onBatch(statements[i:end]); err != nil {
			return fmt.Errorf("onBatch callback failed: %w", err)
		}
	}
	return nil
}
