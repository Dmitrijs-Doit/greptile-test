package converter

import (
	"fmt"
	"testing"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/stretchr/testify/assert"
)

func setupService() *CurrencyConverterService {
	return &CurrencyConverterService{}
}

func initCurrencyHistoricalTimeseries() {
	fixer.CurrencyHistoricalTimeseries = make(map[int]map[string]map[string]float64)
	fixer.CurrencyHistoricalTimeseries[2022] = make(map[string]map[string]float64)
	fixer.CurrencyHistoricalTimeseries[2022]["2022-01-01"] = map[string]float64{
		"USD": 1.0,
		"EUR": 0.91,
		"ILS": 3.28,
		"CAD": 0,
	}
}

func TestCalculateCurrencyRate(t *testing.T) {
	s := setupService()

	initCurrencyHistoricalTimeseries()

	testedDate := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	invalidDate := time.Date(2002, 1, 1, 0, 0, 0, 0, time.UTC)
	invalidFullDate := time.Date(2022, 1, 2, 0, 0, 0, 0, time.UTC)
	invalidDateError := fmt.Errorf("year %d not found in the fixer", invalidDate.Year())
	invalidFullDateError := fmt.Errorf("date %s not found in the fixer", invalidFullDate.Format(format))
	invalidFrom := fmt.Errorf("invalid 'from' currency %s", "HKD")
	invalidTo := fmt.Errorf("invalid 'to' currency %s", "HKD")
	emptyExchangeRate := fmt.Errorf("could not get exchange rate for %s", "CAD")

	testCases := []struct {
		name           string
		from           string
		to             string
		amount         float64
		date           time.Time
		expectedAmount float64
		expectedError  error
	}{
		{name: "USD/ILS", from: "USD", to: "ILS", amount: 5.2, date: testedDate, expectedAmount: 17.056},
		{name: "ILS/USD", from: "ILS", to: "USD", amount: 5.2, date: testedDate, expectedAmount: 1.5853658536585367},
		{name: "EUR/USD", from: "EUR", to: "USD", amount: 20, date: testedDate, expectedAmount: 21.978021978021978},
		{name: "EUR/ILS", from: "EUR", to: "ILS", amount: 10, date: testedDate, expectedAmount: 36.04395604395604},
		{name: "USD/ILS - invalid year", from: "USD", to: "ILS", amount: 5.2, date: invalidDate, expectedAmount: 5.2, expectedError: invalidDateError},
		{name: "USD/ILS - invalid date", from: "USD", to: "ILS", amount: 5.2, date: invalidFullDate, expectedAmount: 5.2, expectedError: invalidFullDateError},
		{name: "invalid from", from: "HKD", to: "ILS", amount: 10, date: testedDate, expectedAmount: 10, expectedError: invalidFrom},
		{name: "invalid to", from: "ILS", to: "HKD", amount: 10, date: testedDate, expectedAmount: 10, expectedError: invalidTo},
		{name: "empty exchange rate", from: "CAD", to: "ILS", amount: 10, date: testedDate, expectedAmount: 10, expectedError: emptyExchangeRate},
		{name: "USD/USD", from: "USD", to: "USD", amount: 5.2, date: testedDate, expectedAmount: 5.2},
		{name: "empty exchange rate to", from: "USD", to: "CAD", amount: 10, date: testedDate, expectedAmount: 10, expectedError: emptyExchangeRate},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := s.Convert(tc.from, tc.to, tc.amount, tc.date)
			if err != nil {
				assert.Equal(t, tc.expectedError, err)
			}

			assert.Equal(t, tc.expectedAmount, result)
		})
	}
}
