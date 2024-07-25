package fixer

import (
	"context"

	"github.com/doitintl/http"
)

type LatestRatesInput struct {
	Base    Currency   `json:"base"`
	Symbols []Currency `json:"symbols"`
}

type LatestRatesOutput struct {
	Success   bool                 `json:"success"`
	Date      string               `json:"date"`
	Timestamp int64                `json:"timestamp"`
	Base      Currency             `json:"base"`
	Rates     map[Currency]float64 `json:"rates"`
}

func (s *FixerService) LatestRates(ctx context.Context, input *LatestRatesInput) (*LatestRatesOutput, error) {
	symbols, err := s.parseSymbols(input.Symbols)
	if err != nil {
		return nil, err
	}

	queryParams := map[string][]string{
		"base":    {string(input.Base)},
		"symbols": {symbols},
	}

	var res LatestRatesOutput

	req := &http.Request{
		URL:          "/latest",
		ResponseType: &res,
		QueryParams:  queryParams,
	}

	if _, err := s.fixerAPI.Get(ctx, req); err != nil {
		return nil, err
	}

	return &res, nil
}
