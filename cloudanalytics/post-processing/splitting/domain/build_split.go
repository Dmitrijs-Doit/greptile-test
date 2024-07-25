package domain

import (
	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/domain/split"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
)

type BuildSplit struct {
	MetricsLength int
	RowsCols      []*domainQuery.QueryRequestX
	NumRows       int
	NumCols       int
	ResRows       *[][]bigquery.Value
	SplitsReq     *[]split.Split
	Attributions  []*domainQuery.QueryRequestX
}
