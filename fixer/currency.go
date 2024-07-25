package fixer

import (
	"strconv"

	"golang.org/x/text/message"
)

// The currency to use in this report.
// swagger:enum Currency
// default: "USD"
type Currency string

const (
	USD Currency = "USD"
	ILS Currency = "ILS"
	EUR Currency = "EUR"
	AUD Currency = "AUD"
	CAD Currency = "CAD"
	GBP Currency = "GBP"
	DKK Currency = "DKK"
	NOK Currency = "NOK"
	SEK Currency = "SEK"
	BRL Currency = "BRL"
	SGD Currency = "SGD"
	MXN Currency = "MXN"
	CHF Currency = "CHF"
	MYR Currency = "MYR"
	TWD Currency = "TWD"
	EGP Currency = "EGP"
	ZAR Currency = "ZAR"
	JPY Currency = "JPY"
	IDR Currency = "IDR"
)

var Currencies = []Currency{
	USD,
	ILS,
	EUR,
	AUD,
	CAD,
	GBP,
	DKK,
	NOK,
	SEK,
	BRL,
	SGD,
	MXN,
	CHF,
	MYR,
	TWD,
	EGP,
	ZAR,
	JPY,
	IDR,
}

func FromString(code string) Currency {
	switch code {
	case "USD":
		return USD
	case "ILS":
		return ILS
	case "EUR":
		return EUR
	case "AUD":
		return AUD
	case "CAD":
		return CAD
	case "GBP":
		return GBP
	case "DKK":
		return DKK
	case "NOK":
		return NOK
	case "SEK":
		return SEK
	case "BRL":
		return BRL
	case "SGD":
		return SGD
	case "MXN":
		return MXN
	case "CHF":
		return CHF
	case "MYR":
		return MYR
	case "TWD":
		return TWD
	case "EGP":
		return EGP
	case "ZAR":
		return ZAR
	case "JPY":
		return JPY
	case "IDR":
		return IDR
	default:
		return USD
	}
}

// Symbol returns the symbol for each fixer currency.
func (c Currency) Symbol() string {
	switch c {
	case USD:
		return "$"
	case EUR:
		return "€"
	case GBP:
		return "£"
	case ILS:
		return "₪"
	case CAD:
		return "C$"
	case AUD:
		return "A$"
	case DKK:
		return "kr"
	case NOK:
		return "kr"
	case SEK:
		return "kr"
	case BRL:
		return "R$"
	case SGD:
		return "S$"
	case MXN:
		return "MX$"
	case CHF:
		return "Fr."
	case MYR:
		return "RM"
	case TWD:
		return "NT$"
	case EGP:
		return "E£"
	case ZAR:
		return "R"
	case JPY:
		return "¥"
	case IDR:
		return "Rp"
	default:
		return ""
	}
}

func SupportedCurrency(code string) bool {
	for _, v := range Currencies {
		if code == string(v) {
			return true
		}
	}

	return false
}

func GetCurrencySymbol(currency string) (string, bool) {
	if SupportedCurrency(currency) {
		return FromString(currency).Symbol(), true
	}

	return currency, false
}

func CodeToLabel(code string) string {
	if SupportedCurrency(code) {
		return string(FromString(code))
	}

	return code
}

// FormatCurrencyAmountInt64 formats int64 amount with currency where (1550,USD) => $15.50
func FormatCurrencyAmountInt64(p *message.Printer, amount int64, currency string) string {
	symbol, ok := GetCurrencySymbol(currency)
	if !ok {
		return p.Sprintf("%d.%d %s", amount/100, amount%100, currency)
	}

	return p.Sprintf("%s%d.%d", symbol, amount/100, amount%100)
}

func FormatCurrencyAmountFloat(p *message.Printer, amount float64, fracDigits int, currency string) string {
	symbol, ok := GetCurrencySymbol(currency)

	precision := strconv.Itoa(fracDigits)
	if !ok {
		part := "%." + precision + "f %s"
		return p.Sprintf(part, amount, currency)
	}

	part := "%s%." + precision + "f"

	return p.Sprintf(part, symbol, amount)
}
