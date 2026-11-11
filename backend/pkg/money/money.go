package money

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// Decimal aliases shopspring/decimal.Decimal to strictly guarantee exact 18,4 arithmetic.
type Decimal = decimal.Decimal

// Precision defines 4 decimal places matching PostgreSQL NUMERIC(18, 4).
const Precision = 4

var (
	// ZeroVal represents exact zero.
	ZeroVal = decimal.NewFromInt(0)
	// Hundred represents 100 for percentage calculations.
	Hundred = decimal.NewFromInt(100)
)

// Zero returns a Decimal representing 0.0000.
func Zero() Decimal {
	return ZeroVal
}

// FromInt creates a Decimal from int64.
func FromInt(n int64) Decimal {
	return decimal.NewFromInt(n)
}

// FromString parses a string into Decimal, trimming whitespace and currency symbols.
func FromString(s string) (Decimal, error) {
	cleaned := strings.TrimSpace(s)
	cleaned = strings.TrimPrefix(cleaned, "₹")
	cleaned = strings.TrimPrefix(cleaned, "$")
	cleaned = strings.TrimPrefix(cleaned, "€")
	cleaned = strings.TrimPrefix(cleaned, "£")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	cleaned = strings.TrimSpace(cleaned)

	if cleaned == "" {
		return ZeroVal, nil
	}

	d, err := decimal.NewFromString(cleaned)
	if err != nil {
		return ZeroVal, fmt.Errorf("invalid decimal money string '%s': %w", s, err)
	}
	return Round4(d), nil
}

// MustFromString parses a string or panics.
func MustFromString(s string) Decimal {
	d, err := FromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

// FromStringOrZero parses a string or returns ZeroVal on error.
func FromStringOrZero(s string) Decimal {
	d, err := FromString(s)
	if err != nil {
		return ZeroVal
	}
	return d
}

// FromFloat creates a Decimal from float64 and rounds to 4 places.
func FromFloat(f float64) Decimal {
	return Round4(decimal.NewFromFloat(f))
}

// Round4 rounds Decimal to exactly 4 decimal places using bankers rounding (HalfEven).
func Round4(d Decimal) Decimal {
	return d.RoundBank(Precision)
}

// Round2 rounds Decimal to 2 decimal places.
func Round2(d Decimal) Decimal {
	return d.RoundBank(2)
}

// Add computes a + b with exact 4-decimal precision.
func Add(a, b Decimal) Decimal {
	return Round4(a.Add(b))
}

// Sub computes a - b with exact 4-decimal precision.
func Sub(a, b Decimal) Decimal {
	return Round4(a.Sub(b))
}

// Mul computes a * b with exact 4-decimal precision.
func Mul(a, b Decimal) Decimal {
	return Round4(a.Mul(b))
}

// Div computes a / b. If b is zero, it returns ZeroVal to prevent runtime panics.
func Div(a, b Decimal) Decimal {
	if b.IsZero() {
		return ZeroVal
	}
	return Round4(a.DivRound(b, Precision))
}

// Abs returns absolute value.
func Abs(d Decimal) Decimal {
	return d.Abs()
}

// Equal checks equality between two decimals.
func Equal(a, b Decimal) bool {
	return a.Equal(b)
}

// GreaterThan checks if a > b.
func GreaterThan(a, b Decimal) bool {
	return a.GreaterThan(b)
}

// LessThan checks if a < b.
func LessThan(a, b Decimal) bool {
	return a.LessThan(b)
}

// GreaterThanOrEqual checks if a >= b.
func GreaterThanOrEqual(a, b Decimal) bool {
	return a.GreaterThanOrEqual(b)
}

// LessThanOrEqual checks if a <= b.
func LessThanOrEqual(a, b Decimal) bool {
	return a.LessThanOrEqual(b)
}

// IsZero checks if value is zero.
func IsZero(d Decimal) bool {
	return d.IsZero()
}

// IsNegative checks if value is strictly less than 0.
func IsNegative(d Decimal) bool {
	return d.IsNegative()
}

// IsPositive checks if value is strictly greater than 0.
func IsPositive(d Decimal) bool {
	return d.IsPositive()
}

// Percentage calculates (amount * pct) / 100 with exact 4-decimal precision.
func Percentage(amount, pct Decimal) Decimal {
	return Round4(amount.Mul(pct).DivRound(Hundred, Precision))
}

// CalculateMDRAndGST computes expected PG fee and tax based on MDR % and GST %:
// fee = gross * (mdrPct / 100)
// tax = fee * (gstPct / 100)
// net = gross - fee - tax
func CalculateMDRAndGST(gross, mdrPct, gstPct Decimal) (fee Decimal, tax Decimal, net Decimal) {
	fee = Percentage(gross, mdrPct)
	tax = Percentage(fee, gstPct)
	net = Sub(gross, Add(fee, tax))
	return fee, tax, net
}

// Format4 formats decimal to fixed 4 decimal places (e.g. "1234.5670").
func Format4(d Decimal) string {
	return d.StringFixed(Precision)
}

// Format2 formats decimal to fixed 2 decimal places for user-facing currency display.
func Format2(d Decimal) string {
	return d.StringFixed(2)
}

// FormatCurrency formats with ISO code (e.g. "INR 1,234.56").
func FormatCurrency(d Decimal, currency string) string {
	curr := strings.ToUpper(strings.TrimSpace(currency))
	if curr == "" {
		curr = "INR"
	}

	val := d.StringFixed(2)
	parts := strings.Split(val, ".")
	intPart := parts[0]
	decPart := parts[1]

	var formattedInt strings.Builder
	neg := false
	if strings.HasPrefix(intPart, "-") {
		neg = true
		intPart = intPart[1:]
	}

	length := len(intPart)
	for i, ch := range intPart {
		if i > 0 && (length-i)%3 == 0 {
			formattedInt.WriteRune(',')
		}
		formattedInt.WriteRune(ch)
	}

	res := formattedInt.String()
	if neg {
		res = "-" + res
	}

	return fmt.Sprintf("%s %s.%s", curr, res, decPart)
}

