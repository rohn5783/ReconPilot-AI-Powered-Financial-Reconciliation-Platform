package money

import (
	"testing"
)

func TestMoneyBasicArithmetic(t *testing.T) {
	a := MustFromString("100.5000")
	b := MustFromString("50.2500")

	// Add
	sum := Add(a, b)
	if !Equal(sum, MustFromString("150.7500")) {
		t.Fatalf("expected 150.7500, got %s", Format4(sum))
	}

	// Sub
	diff := Sub(a, b)
	if !Equal(diff, MustFromString("50.2500")) {
		t.Fatalf("expected 50.2500, got %s", Format4(diff))
	}

	// Mul
	product := Mul(MustFromString("10.0000"), MustFromString("2.5000"))
	if !Equal(product, MustFromString("25.0000")) {
		t.Fatalf("expected 25.0000, got %s", Format4(product))
	}

	// Div
	quotient := Div(MustFromString("100.0000"), MustFromString("4.0000"))
	if !Equal(quotient, MustFromString("25.0000")) {
		t.Fatalf("expected 25.0000, got %s", Format4(quotient))
	}

	// Div by Zero safe guard
	zeroDiv := Div(MustFromString("100.0000"), Zero())
	if !IsZero(zeroDiv) {
		t.Fatalf("expected zero on division by zero, got %s", Format4(zeroDiv))
	}
}

func TestMDRAndGSTCalculation(t *testing.T) {
	gross := MustFromString("10000.0000")
	mdrPct := MustFromString("2.0000") // 2% MDR
	gstPct := MustFromString("18.0000") // 18% GST on fee

	fee, tax, net := CalculateMDRAndGST(gross, mdrPct, gstPct)

	// fee = 10000 * 2% = 200.0000
	if !Equal(fee, MustFromString("200.0000")) {
		t.Fatalf("expected fee 200.0000, got %s", Format4(fee))
	}

	// tax = 200 * 18% = 36.0000
	if !Equal(tax, MustFromString("36.0000")) {
		t.Fatalf("expected tax 36.0000, got %s", Format4(tax))
	}

	// net = 10000 - 200 - 36 = 9764.0000
	if !Equal(net, MustFromString("9764.0000")) {
		t.Fatalf("expected net 9764.0000, got %s", Format4(net))
	}

	// Check formula exactness: gross - fee - tax == net
	formulaCheck := Sub(gross, Add(fee, tax))
	if !Equal(formulaCheck, net) {
		t.Fatalf("formula gross - fee - tax != net: %s != %s", Format4(formulaCheck), Format4(net))
	}
}

func TestMoneyParsingAndFormatting(t *testing.T) {
	// Clean string with currency symbol and commas
	parsed, err := FromString("₹ 1,49,999.50")
	if err != nil {
		t.Fatalf("unexpected error parsing currency: %v", err)
	}
	if !Equal(parsed, MustFromString("149999.5000")) {
		t.Fatalf("expected 149999.5000, got %s", Format4(parsed))
	}

	// Format currency
	formatted := FormatCurrency(MustFromString("1234567.8900"), "INR")
	expected := "INR 1,234,567.89"
	if formatted != expected {
		t.Fatalf("expected '%s', got '%s'", expected, formatted)
	}

	// Negative formatting
	negFormatted := FormatCurrency(MustFromString("-5432.1000"), "USD")
	expectedNeg := "USD -5,432.10"
	if negFormatted != expectedNeg {
		t.Fatalf("expected '%s', got '%s'", expectedNeg, negFormatted)
	}
}

func TestComparisonsAndAbs(t *testing.T) {
	neg := MustFromString("-45.6700")
	pos := Abs(neg)
	if !Equal(pos, MustFromString("45.6700")) {
		t.Fatalf("expected 45.6700, got %s", Format4(pos))
	}

	if !IsNegative(neg) {
		t.Fatalf("expected neg to be negative")
	}
	if !IsPositive(pos) {
		t.Fatalf("expected pos to be positive")
	}

	if !GreaterThan(pos, neg) {
		t.Fatalf("expected pos > neg")
	}
	if !LessThan(neg, pos) {
		t.Fatalf("expected neg < pos")
	}
}

