package converter

import (
	"fmt"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/fixer/converter/iface"

	"github.com/doitintl/hello/scheduled-tasks/fixer"
)

const format string = "2006-01-02"

type CurrencyConverterService struct {
}

func NewCurrencyConverterService() iface.Converter {
	return &CurrencyConverterService{}
}

// Convert - converts currency to another currency at a given date rate
func (s *CurrencyConverterService) Convert(from string, to string, amount float64, date time.Time) (float64, error) {
	if from == to {
		return amount, nil
	}

	if fixer.CurrencyHistoricalTimeseries[date.Year()] == nil {
		return amount, fmt.Errorf("year %d not found in the fixer", date.Year())
	}

	if fixer.CurrencyHistoricalTimeseries[date.Year()][date.Format(format)] == nil {
		return amount, fmt.Errorf("date %s not found in the fixer", date.Format(format))
	}

	currencyRates := fixer.CurrencyHistoricalTimeseries[date.Year()][date.Format(format)]

	fromRate, ok := currencyRates[fixer.CodeToLabel(from)]
	if !ok {
		return amount, fmt.Errorf("invalid 'from' currency %s", from)
	}

	if fromRate == 0 {
		return amount, fmt.Errorf("could not get exchange rate for %s", from)
	}

	toRate, ok := currencyRates[fixer.CodeToLabel(to)]
	if !ok {
		return amount, fmt.Errorf("invalid 'to' currency %s", to)
	}

	if toRate == 0 {
		return amount, fmt.Errorf("could not get exchange rate for %s", to)
	}

	return amount * toRate / fromRate, nil
}
