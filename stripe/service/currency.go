package service

import (
	"strings"

	"github.com/stripe/stripe-go/v74"
)

func CurrencyToUpperString(currency stripe.Currency) string {
	return strings.ToUpper(string(currency))
}
