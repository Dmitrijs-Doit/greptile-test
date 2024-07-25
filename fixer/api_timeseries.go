package fixer

import (
	"context"
	"time"

	"github.com/doitintl/http"
)

type TimeseriesInput struct {
	Base      Currency   `json:"base"`
	Symbols   []Currency `json:"symbols"`
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
}

type TimeseriesOutput struct {
	Success   bool                          `json:"success"`
	Timestamp int64                         `json:"timestamp"`
	StartDate string                        `json:"start_date"`
	EndDate   string                        `json:"end_date"`
	Base      Currency                      `json:"base"`
	Rates     map[string]map[string]float64 `json:"rates"`
}

func (s *FixerService) Timeseries(ctx context.Context, input *TimeseriesInput) (*TimeseriesOutput, error) {
	if input.StartDate == nil || input.StartDate.IsZero() {
		return nil, ErrInvalidStartDate
	}

	if input.EndDate == nil || input.EndDate.IsZero() {
		return nil, ErrInvalidEndDate
	}

	symbols, err := s.parseSymbols(input.Symbols)
	if err != nil {
		return nil, err
	}

	queryParams := map[string][]string{
		"base":       {string(input.Base)},
		"symbols":    {symbols},
		"start_date": {input.StartDate.Format("2006-01-02")},
		"end_date":   {input.EndDate.Format("2006-01-02")},
	}

	var res TimeseriesOutput
	req := &http.Request{
		URL:          "/timeseries",
		ResponseType: &res,
		QueryParams:  queryParams,
	}

	if _, err := s.fixerAPI.Get(ctx, req); err != nil {
		return nil, err
	}

	return &res, nil
}
