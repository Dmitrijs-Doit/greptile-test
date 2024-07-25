package dal

import (
	"context"
)

//go:generate mockery --name Billing
type Billing interface {
	GetCoveredUsage(
		ctx context.Context,
		accountID, payerID string,
		payerNumber int,
		spARNs, riAccountIDs []string,
	) (CoveredUsage, error)
}

type CoveredUsage struct {
	SPCost float64 `bigquery:"sp_cost"`
	RICost float64 `bigquery:"ri_cost"`
}
