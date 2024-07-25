package service

import (
	"context"
	"fmt"
	"net/http"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"

	domainBackfill "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/backfill/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

// QueryResultRow contains info about query result rows count
type QueryResultRow struct {
	Count int64
}

// Define constants used in functions below
const (
	tableSizeQueryFormat       = "SELECT COUNT(*) as count FROM `%s.%s`"
	partitionSizeQueryFormat   = "SELECT COUNT(*) as count FROM `%s.%s` WHERE DATE(export_time) = DATE('%s')"
	deletePartitionQueryFormat = "DELETE FROM `%s.%s` WHERE DATE(export_time) = DATE('%s')"
)

func (clients *clients) exportTableToRegionBucket(
	ctx context.Context,
	sourceTableDetails *domainBackfill.BillingTable,
	gcsURI,
	srcTable,
	location string,
) error {
	l := logger.FromContext(ctx)
	gcsRef := bigquery.NewGCSReference(gcsURI)
	gcsRef.DestinationFormat = bigquery.JSON
	gcsRef.Compression = bigquery.Gzip

	extractor := clients.customerBQClient.DatasetInProject(sourceTableDetails.Project, sourceTableDetails.Dataset).Table(srcTable).ExtractorTo(gcsRef)
	extractor.DisableHeader = true
	extractor.Location = location

	job, err := extractor.Run(ctx)
	if err != nil {
		return err
	}

	l.Infof("export job id %s project %s", job.ID(), job.ProjectID())

	status, err := job.Wait(ctx)
	if err != nil {
		return err
	}

	if err := status.Err(); err != nil {
		return err
	}

	return nil
}

func (clients *clients) createCustomerBillingDataTable(
	ctx context.Context,
	templateDatasetID, templateTableID, dstDatasetID, dstTableID string,
) error {
	// If it doesn't exist, run a copy job to copy the template table as customer billing data table.
	// The location of this table can be set in FS under app/direct-gcp-accounts-pipeline as templateTable
	templateDataset := clients.bigqueryClient.Dataset(templateDatasetID)
	dstDataset := clients.bigqueryClient.Dataset(dstDatasetID)
	copier := dstDataset.Table(dstTableID).CopierFrom(templateDataset.Table(templateTableID))
	copier.WriteDisposition = bigquery.WriteEmpty

	job, err := copier.Run(ctx)
	if err != nil {
		return err
	}

	status, err := job.Wait(ctx)
	if err != nil {
		return err
	}

	if err := status.Err(); err != nil {
		return err
	}

	return nil
}

func (clients *clients) createDatasetAndGrantPermissions(ctx context.Context, datasetID, clientEmail string) error {
	l := logger.FromContext(ctx)

	// Check if destination table with customer billing data exists
	exists, err := clients.datasetExists(ctx, datasetID)
	if err != nil {
		return err
	}

	l.Infof("Destintaion dataset %s exists: %t", datasetID, exists)

	// If the dataset doesn't exist- create it.
	if !exists {
		if err := clients.bigqueryClient.Dataset(datasetID).Create(ctx, &bigquery.DatasetMetadata{
			Location: "US",
		}); err != nil {
			return err
		}

		l.Infof("Destintaion dataset created!")

		// Grant customer SA Writer role to the destination dataset which is needed to copy data from their project into our project
		if err := clients.updateDatasetAccessControl(ctx, datasetID, clientEmail); err != nil {
			return err
		}

		l.Infof("Permissions granted")
	}

	return nil
}

func (clients *clients) updateDatasetAccessControl(ctx context.Context, datasetID, customerEmail string) error {
	dataset := clients.bigqueryClient.Dataset(datasetID)

	meta, err := dataset.Metadata(ctx)
	if err != nil {
		return err
	}
	// Append a new access control entry to the existing access list.
	update := bigquery.DatasetMetadataToUpdate{
		Access: append(meta.Access, &bigquery.AccessEntry{
			Role:       bigquery.WriterRole,
			EntityType: bigquery.UserEmailEntity,
			Entity:     customerEmail},
		),
	}

	// Leverage the ETag for the update to assert there's been no modifications to the
	// dataset since the metadata was originally read.
	if _, err := dataset.Update(ctx, update, meta.ETag); err != nil {
		return err
	}

	return nil
}

func (clients *clients) loadFilesToBQ(
	ctx context.Context,
	projectID,
	datasetID,
	tableID,
	gcsURI string,
) (int64, error) {
	l := logger.FromContext(ctx)
	gcsRef := bigquery.NewGCSReference(gcsURI)
	gcsRef.SourceFormat = bigquery.JSON
	gcsRef.AutoDetect = false
	loader := clients.bigqueryClient.Dataset(datasetID).Table(tableID).LoaderFrom(gcsRef)
	loader.WriteDisposition = bigquery.WriteAppend

	job, err := loader.Run(ctx)
	if err != nil {
		return 0, err
	}

	l.Infof("load job id %s, project %s", job.ID(), job.ProjectID())

	status, err := job.Wait(ctx)
	if err != nil {
		return 0, err
	}

	if status.Err() != nil {
		return 0, fmt.Errorf("job completed with error: %v", status.Err())
	}

	return status.Statistics.Details.(*bigquery.LoadStatistics).OutputRows, nil
}

func (clients *clients) tableExists(ctx context.Context, datasetID, tableID string) (bool, error) {
	tableRef := clients.bigqueryClient.Dataset(datasetID).Table(tableID)

	_, err := tableRef.Metadata(ctx)
	if err != nil {
		if gapiErr, ok := err.(*googleapi.Error); ok {
			if gapiErr.Code == http.StatusNotFound {
				return false, nil
			}
		}

		return false, err
	}

	return true, nil
}

func (clients *clients) datasetExists(ctx context.Context, datasetID string) (bool, error) {
	tableRef := clients.bigqueryClient.Dataset(datasetID)

	_, err := tableRef.Metadata(ctx)
	if err != nil {
		if gapiErr, ok := err.(*googleapi.Error); ok {
			if gapiErr.Code == http.StatusNotFound {
				return false, nil
			}
		}

		return false, err
	}

	return true, nil
}

func getDatasetLocation(ctx context.Context, bq *bigquery.Client, projectID, datasetID string) (string, error) {
	meta, err := bq.DatasetInProject(projectID, datasetID).Metadata(ctx)
	if err != nil {
		return "", err
	}

	return meta.Location, nil
}

func executeQuery(ctx context.Context, bq *bigquery.Client, queryString string) ([]*QueryResultRow, error) {
	query := bq.Query(queryString)

	rows, err := query.Read(ctx)
	if err != nil {
		return nil, err
	}

	results, err := convertResults(rows)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func convertResults(iter *bigquery.RowIterator) ([]*QueryResultRow, error) {
	results := make([]*QueryResultRow, 0)

	for {
		var row QueryResultRow

		err := iter.Next(&row)
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("error iterating through results: %v", err)
		}

		results = append(results, &row)
	}

	return results, nil
}
