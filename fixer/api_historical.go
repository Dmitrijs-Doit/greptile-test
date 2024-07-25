package fixer

import (
	"context"
	"fmt"
	"time"

	"github.com/doitintl/http"
)

type HistoricalRatesInput struct {
	Base    Currency   `json:"base"`
	Symbols []Currency `json:"symbols"`
	Date    *time.Time `json:"date"`
}

type HistoricalRatesOutput struct {
	Success    bool                 `json:"success"`
	Historical bool                 `json:"historical"`
	Date       string               `json:"date"`
	Timestamp  int64                `json:"timestamp"`
	Base       Currency             `json:"base"`
	Rates      map[Currency]float64 `json:"rates"`
}

func (s *FixerService) HistoricalRates(ctx context.Context, input *HistoricalRatesInput) (*HistoricalRatesOutput, error) {
	if input.Date == nil || input.Date.IsZero() {
		return nil, ErrInvalidDate
	}

	symbols, err := s.parseSymbols(input.Symbols)
	if err != nil {
		return nil, err
	}

	queryParams := map[string][]string{
		"base":    {string(input.Base)},
		"symbols": {symbols},
	}

	var res HistoricalRatesOutput

	req := &http.Request{
		URL:          fmt.Sprintf("/%s", input.Date.Format("2006-01-02")),
		ResponseType: &res,
		QueryParams:  queryParams,
	}

	if _, err := s.fixerAPI.Get(ctx, req); err != nil {
		return nil, err
	}

	return &res, nil
}
