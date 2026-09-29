package money

import (
	"errors"
	"testing"

	"github.com/Fazann/notedbook-backend/pkg/enums"
)

func TestToDecimalString(t *testing.T) {
	tests := []struct {
		amount   int64
		currency enums.CurrencyEnum
		want     string
	}{
		{1000, enums.CurrencyEnumUSD, "10.00"},
		{1250, enums.CurrencyEnumUSD, "12.50"},
		{5, enums.CurrencyEnumUSD, "0.05"},
		{0, enums.CurrencyEnumMYR, "0.00"},
		{123456789, enums.CurrencyEnumMYR, "1234567.89"},
		{40000, enums.CurrencyEnumKHRiel, "40000"},
		{MaxMinor, enums.CurrencyEnumUSD, "9999999999999.99"},
	}

	for _, tt := range tests {
		if got := ToDecimalString(tt.amount, tt.currency); got != tt.want {
			t.Errorf("ToDecimalString(%d, %s) = %q, want %q", tt.amount, tt.currency, got, tt.want)
		}
	}
}

func TestMinAmount(t *testing.T) {
	want := map[enums.CurrencyEnum]int64{enums.CurrencyEnumUSD: 1, enums.CurrencyEnumMYR: 1, enums.CurrencyEnumKHRiel: 100}
	for currency, minimum := range want {
		if got := MinAmount(currency); got != minimum {
			t.Errorf("MinAmount(%s) = %d, want %d", currency, got, minimum)
		}
	}
}

func TestToMinor(t *testing.T) {
	tests := []struct {
		value    string
		currency enums.CurrencyEnum
		want     int64
		wantErr  bool
	}{
		{value: "10.00", currency: enums.CurrencyEnumUSD, want: 1000},
		{value: "10", currency: enums.CurrencyEnumUSD, want: 1000},
		{value: "12.5", currency: enums.CurrencyEnumUSD, want: 1250},
		{value: "0.05", currency: enums.CurrencyEnumMYR, want: 5},
		{value: "0", currency: enums.CurrencyEnumUSD, want: 0},
		{value: "40000", currency: enums.CurrencyEnumKHRiel, want: 40000},
		{value: "40000.00", currency: enums.CurrencyEnumKHRiel, want: 40000},
		{value: "10.500", currency: enums.CurrencyEnumUSD, want: 1050},
		{value: "9999999999999.99", currency: enums.CurrencyEnumUSD, want: MaxMinor},
		{value: "10.005", currency: enums.CurrencyEnumUSD, wantErr: true},
		{value: "10.5", currency: enums.CurrencyEnumKHRiel, wantErr: true},
		{value: "10000000000000.00", currency: enums.CurrencyEnumUSD, wantErr: true},
		{value: "-1.00", currency: enums.CurrencyEnumUSD, wantErr: true},
		{value: "1e3", currency: enums.CurrencyEnumUSD, wantErr: true},
		{value: "1,000.00", currency: enums.CurrencyEnumUSD, wantErr: true},
		{value: ".5", currency: enums.CurrencyEnumUSD, wantErr: true},
		{value: "", currency: enums.CurrencyEnumUSD, wantErr: true},
	}

	for _, tt := range tests {
		got, err := ToMinor(tt.value, tt.currency)
		if tt.wantErr {
			if !errors.Is(err, ErrInvalidAmount) {
				t.Errorf("ToMinor(%q, %s) err = %v, want ErrInvalidAmount", tt.value, tt.currency, err)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("ToMinor(%q, %s) = %d, %v, want %d", tt.value, tt.currency, got, err, tt.want)
		}
	}
}
