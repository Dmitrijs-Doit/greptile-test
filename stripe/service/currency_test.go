package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/stripe/stripe-go/v74"
)

// TestCurrencyToUpper tests the CurrencyToUpper function
func TestCurrencyToUpper(t *testing.T) {
	payload := []struct {
		currency stripe.Currency
		expected string
	}{
		{stripe.CurrencyUSD, "USD"},
		{stripe.CurrencyEUR, "EUR"},
		{stripe.CurrencyGBP, "GBP"},
	}

	for _, p := range payload {
		assert.Equal(t, p.expected, CurrencyToUpperString(p.currency))
	}
}
