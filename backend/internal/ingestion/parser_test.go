package ingestion

import (
	"bytes"
	"strings"
	"testing"

	"reconpilot/internal/domain"
	"reconpilot/pkg/money"
)

func TestParseOrdersCSV(t *testing.T) {
	csvData := `order_id,customer_id,gross_amount,currency,order_status,created_at
ORD-101,CUST-A,15000.5000,INR,CAPTURED,2026-11-01 10:00:00
ORD-102,CUST-B,₹2450.00,INR,PENDING,01/11/2026
ORD-103,CUST-C,$120.00,USD,REFUNDED,2026-11-02T14:30:00Z
`
	orders, err := ParseOrdersCSV(strings.NewReader(csvData), true)
	if err != nil {
		t.Fatalf("ParseOrdersCSV failed: %v", err)
	}

	if len(orders) != 3 {
		t.Fatalf("Expected 3 orders, got %d", len(orders))
	}

	if orders[0].OrderID != "ORD-101" || !orders[0].GrossAmount.Equal(money.FromStringOrZero("15000.5000")) {
		t.Errorf("Order 0 mismatch: %+v", orders[0])
	}
	if orders[1].OrderStatus != domain.OrderStatusPending || !orders[1].GrossAmount.Equal(money.FromStringOrZero("2450.0000")) {
		t.Errorf("Order 1 mismatch: %+v", orders[1])
	}
	if orders[2].Currency != "USD" || orders[2].OrderStatus != domain.OrderStatusRefunded {
		t.Errorf("Order 2 mismatch: %+v", orders[2])
	}
}

func TestParseOrdersCSVStrictValidationFailure(t *testing.T) {
	// Missing required gross_amount
	csvData := `order_id,customer_id,created_at
ORD-101,CUST-A,2026-11-01
`
	_, err := ParseOrdersCSV(strings.NewReader(csvData), true)
	if err == nil {
		t.Errorf("Expected error for missing required header gross_amount, got nil")
	}

	// Invalid status in strict mode
	badStatusCSV := `order_id,gross_amount,order_status
ORD-101,1000,UNKNOWN_STATUS
`
	_, err = ParseOrdersCSV(strings.NewReader(badStatusCSV), true)
	if err == nil {
		t.Errorf("Expected error for invalid order_status in strict mode, got nil")
	}
}

func TestParsePGSettlementsCSV(t *testing.T) {
	csvData := `pg_txn_id,order_id,gross_amount,pg_fee,tax,net_amount,payout_id,payout_status,settled_at
pay_rzp_1,ORD-101,10000.00,200.00,36.00,9764.00,PO-001,SETTLED,2026-11-02 12:00:00
pay_rzp_2,ORD-102,5000.00,100.00,18.00,,PO-001,PROCESSED,2026-11-02 14:00:00
`
	settlements, err := ParsePGSettlementsCSV(strings.NewReader(csvData), true)
	if err != nil {
		t.Fatalf("ParsePGSettlementsCSV failed: %v", err)
	}

	if len(settlements) != 2 {
		t.Fatalf("Expected 2 settlements, got %d", len(settlements))
	}

	s0 := settlements[0]
	if s0.PGTxnID != "pay_rzp_1" || !s0.NetAmount.Equal(money.FromStringOrZero("9764.0000")) {
		t.Errorf("Settlement 0 net amount mismatch: got %s", s0.NetAmount.StringFixed(4))
	}

	// In settlement 1, net_amount was empty in CSV, so parser computes gross - fee - tax = 5000 - 100 - 18 = 4882
	s1 := settlements[1]
	expectedNet := money.FromStringOrZero("4882.0000")
	if !s1.NetAmount.Equal(expectedNet) {
		t.Errorf("Settlement 1 calculated net mismatch: expected %s, got %s", expectedNet.StringFixed(4), s1.NetAmount.StringFixed(4))
	}
}

func TestParseBankStatementsCSV(t *testing.T) {
	csvData := `bank_txn_id,utr_number,credit_amount,description,transaction_date
BNK-001,HDFCR52026110290101,9764.0000,CMS/RAZORPAY SOFTWARE/PO-001,2026-11-02 16:30:00
BNK-002,ICICR52026110290102,4882.0000,NEFT-ICICI-PAYOUT-PO-001,02-11-2026
`
	statements, err := ParseBankStatementsCSV(strings.NewReader(csvData), true)
	if err != nil {
		t.Fatalf("ParseBankStatementsCSV failed: %v", err)
	}

	if len(statements) != 2 {
		t.Fatalf("Expected 2 statements, got %d", len(statements))
	}

	if statements[0].UTRNumber != "HDFCR52026110290101" || !statements[0].CreditAmount.Equal(money.FromStringOrZero("9764.0000")) {
		t.Errorf("Statement 0 mismatch: %+v", statements[0])
	}
}

func TestStreamOrdersCSVWithRingBuffer(t *testing.T) {
	// Construct CSV with 50 rows
	var buf bytes.Buffer
	buf.WriteString("order_id,customer_id,gross_amount,currency,order_status,created_at\n")
	for i := 1; i <= 50; i++ {
		buf.WriteString(strings.Join([]string{
			"ORD-" + string(rune(1000+i)),
			"CUST-X",
			"500.00",
			"INR",
			"CAPTURED",
			"2026-11-01\n",
		}, ","))
	}

	var batchCount int
	var totalOrders int

	err := StreamOrdersCSV(&buf, false, 64, 10, func(batch []domain.Order) error {
		batchCount++
		totalOrders += len(batch)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamOrdersCSV failed: %v", err)
	}

	if totalOrders != 50 {
		t.Errorf("Expected 50 streamed orders, got %d", totalOrders)
	}
	if batchCount != 5 {
		t.Errorf("Expected 5 batches of size 10, got %d", batchCount)
	}
}

