package fixer

import (
	"context"
	"strconv"
	"time"

	"github.com/doitintl/http"
)

type ConvertInput struct {
	To     Currency   `json:"to"`
	From   Currency   `json:"from"`
	Amount float64    `json:"amount"`
	Date   *time.Time `json:"date"`
}

type ConvertOutput struct {
	Success bool               `json:"success"`
	Query   ConvertOutputQuery `json:"query"`
	Date    string             `json:"date"`
	Result  float64            `json:"result"`
	Info    ConvertOutputInfo  `json:"info"`
}

type ConvertOutputQuery struct {
	To     Currency `json:"to"`
	From   Currency `json:"from"`
	Amount float64  `json:"amount"`
}

type ConvertOutputInfo struct {
	Timestamp int64   `json:"timestamp"`
	Rate      float64 `json:"rate"`
}

func (s *FixerService) Convert(ctx context.Context, input *ConvertInput) (*ConvertOutput, error) {
	amount := strconv.FormatFloat(input.Amount, 'f', -1, 64)
	queryParams := map[string][]string{
		"from":   {string(input.From)},
		"to":     {string(input.To)},
		"amount": {amount},
	}

	if input.Date != nil && !input.Date.IsZero() {
		queryParams["date"] = []string{input.Date.UTC().Format("2006-01-02")}
	}

	var res ConvertOutput

	req := &http.Request{
		URL:          "/convert",
		ResponseType: &res,
		QueryParams:  queryParams,
	}

	if _, err := s.fixerAPI.Get(ctx, req); err != nil {
		return nil, err
	}

	return &res, nil
}
