package domain

import (
	doitErrors "github.com/doitintl/errors"
	bqmodels "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/domain/bigquery"
)

func HandleExecutorError(operation string, err error, timeRange bqmodels.TimeRange, queryID bqmodels.QueryName) error {
	return doitErrors.Wrapf(err, "%s() failed for timeRange '%s' and query '%v'", operation, timeRange, queryID)
}
