package iface

import (
	"context"

	"cloud.google.com/go/bigquery"

	dal "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/dal/firestore"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
)

//go:generate mockery --name Executor --output ../mocks --case=underscore
type Executor interface {
	Execute(
		ctx context.Context,
		customerBQ *bigquery.Client,
		replacements domain.Replacements,
		transformerContext domain.TransformerContext,
		queriesPerMode map[bqmodels.Mode]map[bqmodels.QueryName]string,
		hasTableDiscovery bool,
	) (dal.RecommendationSummary, []error)
}
