package iface

import (
	"context"

	insightsSDK "github.com/doitintl/insights/sdk"
)

//go:generate mockery --name Insights --output ../mocks --case=underscore
type Insights interface {
	PostInsightResults(ctx context.Context, results []insightsSDK.InsightResponse) error
}
