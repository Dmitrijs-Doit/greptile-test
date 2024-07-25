package domain

import (
	"cloud.google.com/go/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type BigQueryTableUpdateRequest struct {
	DefaultProjectID      string
	DefaultDatasetID      string
	DestinationProjectID  string
	DestinationDatasetID  string
	DestinationTableName  string
	AllPartitions         bool
	WriteDisposition      bigquery.TableWriteDisposition
	DatasetMetadata       bigquery.DatasetMetadata
	QueryParameters       []bigquery.QueryParameter
	ConfigJobID           string
	Clustering            *bigquery.Clustering
	WaitTillDone          bool
	DML                   bool
	CSP                   bool
	FromDate              string
	FromDateNumPartitions int
	IsStandalone          bool

	// Labeling Strategy
	// https://doitintl.atlassian.net/wiki/spaces/ENG/pages/437846102/Platform+Finops+Strategy+crawl#Resource-labeling-strategy
	House   common.House
	Feature common.Feature
	Module  common.Module
}
