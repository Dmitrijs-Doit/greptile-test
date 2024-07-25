package domain

import (
	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
)

type AttrFiltersParams struct {
	CompositeFilters     []string
	AttrConditions       []string
	AttrGroupsConditions map[string]string
	AttrRows             []*domain.QueryRequestX
	QueryParams          []bigquery.QueryParameter
	MetricFilters        []string
}
