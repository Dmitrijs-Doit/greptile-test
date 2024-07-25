package fixer

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const unsupportedCurrency = "XYZ"

func TestSupportedCurrency(t *testing.T) {
	currencies := []string{"USD", "GBP", "AUD", "EUR", "ILS", "CAD", "DKK", "NOK", "SEK", "BRL", "SGD", "MXN", "CHF", "MYR", "TWD", "ZAR", "JPY", "IDR"}

	for _, c := range currencies {
		result := SupportedCurrency(c)
		assert.True(t, result)
	}

	result := SupportedCurrency(unsupportedCurrency)
	assert.False(t, result)
}

func TestGetCurrencySymbol(t *testing.T) {
	expressions := map[string]string{
		"USD": "$",
		"EUR": "€",
		"GBP": "£",
		"ILS": "₪",
		"CAD": "C$",
		"AUD": "A$",
		"DKK": "kr",
		"NOK": "kr",
		"SEK": "kr",
		"BRL": "R$",
		"SGD": "S$",
		"MXN": "MX$",
		"CHF": "Fr.",
		"MYR": "RM",
		"TWD": "NT$",
		"ZAR": "R",
		"JPY": "¥",
		"IDR": "Rp",
	}

	for input, expectedOutput := range expressions {
		result, supported := GetCurrencySymbol(input)
		assert.True(t, supported)
		assert.Equal(t, result, expectedOutput)
	}

	result, supported := GetCurrencySymbol(unsupportedCurrency)
	assert.False(t, supported)
	assert.Equal(t, result, unsupportedCurrency)
}

func TestCodeToLabel(t *testing.T) {
	expressions := map[string]string{
		"USD":               "USD",
		"EUR":               "EUR",
		"GBP":               "GBP",
		"ILS":               "ILS",
		"CAD":               "CAD",
		"AUD":               "AUD",
		"DKK":               "DKK",
		"NOK":               "NOK",
		"SEK":               "SEK",
		"BRL":               "BRL",
		"SGD":               "SGD",
		"MXN":               "MXN",
		"CHF":               "CHF",
		"MYR":               "MYR",
		"TWD":               "TWD",
		"ZAR":               "ZAR",
		"JPY":               "JPY",
		"IDR":               "IDR",
		unsupportedCurrency: unsupportedCurrency,
	}

	for input, expectedOutput := range expressions {
		result := CodeToLabel(input)
		assert.Equal(t, result, expectedOutput)
	}
}

type currecnyFormatInput struct {
	amountInt64   int64
	amountFloat64 float64
	currency      string
	fracDigits    int
}

func TestFormatCurrencyAmountInt64(t *testing.T) {
	expressions := map[currecnyFormatInput]string{
		{amountInt64: 12345, currency: "USD"}:               "$123.45",
		{amountInt64: 12345, currency: unsupportedCurrency}: fmt.Sprintf("123.45 %s", unsupportedCurrency),
	}

	msgPrinter := message.NewPrinter(language.English)

	for input, expectedOutput := range expressions {
		result := FormatCurrencyAmountInt64(msgPrinter, input.amountInt64, input.currency)
		assert.Equal(t, result, expectedOutput)
	}
}

func TestFormatCurrencyAmountFloat(t *testing.T) {
	expressions := map[currecnyFormatInput]string{
		{amountFloat64: 123.4567, currency: "USD", fracDigits: 2}:            "$123.46",
		{amountFloat64: 123.4, currency: unsupportedCurrency, fracDigits: 2}: fmt.Sprintf("123.40 %s", unsupportedCurrency),
		{amountFloat64: 124.34, currency: "USD", fracDigits: 0}:              "$124",
		{amountFloat64: 124.000, currency: "USD", fracDigits: 1}:             "$124.0",
		{amountFloat64: 124, currency: "USD", fracDigits: 2}:                 "$124.00",
		{amountFloat64: 1, currency: "USD", fracDigits: 3}:                   "$1.000",
	}

	msgPrinter := message.NewPrinter(language.English)

	for input, expectedOutput := range expressions {
		result := FormatCurrencyAmountFloat(msgPrinter, input.amountFloat64, input.fracDigits, input.currency)
		assert.Equal(t, expectedOutput, result)
	}
}
