package common

import (
	"context"
	"net/http"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/googleapi"
)

// BigQueryTableExists returns true if a table in the specified path (project, dataset, table id) exists
// otherwise returns false
func BigQueryTableExists(ctx context.Context, bq *bigquery.Client, projectID, datasetID, tableID string) (bool, *bigquery.TableMetadata, error) {
	md, err := bq.DatasetInProject(projectID, datasetID).Table(tableID).Metadata(ctx)
	if err != nil {
		if gapiErr, ok := err.(*googleapi.Error); ok && gapiErr.Code == http.StatusNotFound {
			return false, nil, nil
		}

		return false, nil, err
	}

	return true, md, nil
}

// BigQueryDatasetExists returns true if a dataset in the specified path (project, dataset id) exists
// otherwise returns false
func BigQueryDatasetExists(ctx context.Context, bq *bigquery.Client, projectID, datasetID string) (bool, *bigquery.DatasetMetadata, error) {
	md, err := bq.DatasetInProject(projectID, datasetID).Metadata(ctx)
	if err != nil {
		if gapiErr, ok := err.(*googleapi.Error); ok && gapiErr.Code == http.StatusNotFound {
			return false, nil, nil
		}

		return false, nil, err
	}

	return true, md, nil
}
