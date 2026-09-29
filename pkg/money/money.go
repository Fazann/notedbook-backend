package money

import (
	"errors"
	"regexp"

	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/shopspring/decimal"
)

// MaxMinor is the largest amount in minor units; below 2^53 so JavaScript clients keep it exact.
const MaxMinor int64 = 999_999_999_999_999

// ErrInvalidAmount is returned by ToMinor for text that is not a plain decimal, has more decimal places than
// the currency allows, or is out of range.
var ErrInvalidAmount = errors.New("invalid amount")

// minorDigits is the number of decimal places of each currency's minor unit: USD cents, KHR riel, MYR sen.
var minorDigits = map[enums.CurrencyEnum]int32{
	enums.CurrencyEnumUSD:    2,
	enums.CurrencyEnumKHRiel: 0,
	enums.CurrencyEnumMYR:    2,
}

// minAmounts is the smallest amount in minor units a currency accepts, when above 1: riel has no coin below 100.
var minAmounts = map[enums.CurrencyEnum]int64{
	enums.CurrencyEnumKHRiel: 100,
}

// Plain decimal only: no exponent, sign or thousands separators.
var decimalPattern = regexp.MustCompile(`^\d+(\.\d+)?$`)

// Digits returns the number of decimal places of the currency (2 for USD, 0 for KHR).
func Digits(currency enums.CurrencyEnum) int32 {
	return minorDigits[currency]
}

// MinAmount returns the smallest amount in minor units the currency accepts: 100 for KHR, 1 otherwise.
func MinAmount(currency enums.CurrencyEnum) int64 {
	if minimum, ok := minAmounts[currency]; ok {
		return minimum
	}
	return 1
}

// ToDecimalString formats an amount in minor units as a decimal string with the currency's decimal places:
// (1000, USD) → "10.00", (40000, KHR) → "40000".
func ToDecimalString(amount int64, currency enums.CurrencyEnum) string {
	digits := Digits(currency)
	return decimal.New(amount, -digits).StringFixed(digits)
}

// ToMinor parses a decimal string into minor units: ("10.00", USD) → 1000, ("40000", KHR) → 40000.
// Extra decimal places are accepted only when they are zeros ("10.50" is 10.5 USD but invalid for KHR).
// The result is between 0 and MaxMinor.
func ToMinor(value string, currency enums.CurrencyEnum) (int64, error) {
	if !decimalPattern.MatchString(value) {
		return 0, ErrInvalidAmount
	}
	d, err := decimal.NewFromString(value)
	if err != nil {
		return 0, ErrInvalidAmount
	}

	minor := d.Shift(Digits(currency))
	if !minor.IsInteger() || minor.GreaterThan(decimal.NewFromInt(MaxMinor)) {
		return 0, ErrInvalidAmount
	}
	return minor.IntPart(), nil
}
