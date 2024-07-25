package dal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"

	doitBigquery "github.com/doitintl/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	DataHubDataset     = "datahub_api"
	DataHubEventsTable = "events"
	DataHubEventsView  = "events_no_dup"
)

const deleteJobIDPrefix = "cloud_analytics_datahub_delete_events"
const getSummaryJobIDPrefix = "cloud_analytics_datahub_get_customer_data_summary"
const getCustomersWithSoftDeleteDataJobIDPrefix = "cloud_analytics_datahub_get_customers_with_soft_delete_data"

type DataHubBigQuery struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
}

func NewDataHubBigQuery(
	loggerProvider logger.Provider,
	conn *connection.Connection,
) iface.DataHubBigQuery {
	return &DataHubBigQuery{
		loggerProvider: loggerProvider,
		conn:           conn,
	}
}

// GetEventsTable returns the full path of the DataHub events table.
func (d *DataHubBigQuery) GetEventsTable(projectID string) string {
	return fmt.Sprintf("%s.%s.%s", projectID, DataHubDataset, DataHubEventsTable)
}

// DeleteBigQueryData deletes all of the customer's DataHub API BQ data.
func (d *DataHubBigQuery) DeleteBigQueryData(ctx context.Context, customerID string) error {
	l := d.loggerProvider(ctx)
	bq := d.conn.Bigquery(ctx)

	query := d.getBaseDeleteQuery(bq.Project())

	queryJob := bq.Query(query)

	queryJob.Parameters = []bigquery.QueryParameter{
		{Name: "customer_id", Value: customerID},
	}

	queryJob.JobIDConfig = getJobIDConfig(deleteJobIDPrefix, customerID)
	queryJob.Labels = getQueryJobLabels(ctx, customerID)

	job, err := queryJob.Run(ctx)
	if err != nil {
		return err
	}

	l.Info(job.ID())

	jobStatus, err := job.Status(ctx)
	if err != nil {
		return err
	}

	if err := jobStatus.Err(); err != nil {
		return err
	}

	return nil
}

func (d *DataHubBigQuery) DeleteBigQueryDataByEventIDs(
	ctx context.Context,
	customerID string,
	deleteReq domain.DeleteEventsReq,
	deletedBy string,
) error {
	l := d.loggerProvider(ctx)
	bq := d.conn.Bigquery(ctx)

	query := d.getBaseDeleteQuery(bq.Project())
	query += "AND event_id IN UNNEST(@event_ids)"

	queryJob := bq.Query(query)

	queryJob.Parameters = []bigquery.QueryParameter{
		{Name: "customer_id", Value: customerID},
		{Name: "event_ids", Value: deleteReq.EventIDs},
		{Name: "deleted_by", Value: deletedBy},
	}

	queryJob.JobIDConfig = getJobIDConfig(deleteJobIDPrefix, customerID)
	queryJob.Labels = getQueryJobLabels(ctx, customerID)

	job, err := queryJob.Run(ctx)
	if err != nil {
		return err
	}

	l.Info(job.ID())

	jobStatus, err := job.Status(ctx)
	if err != nil {
		return err
	}

	if err := jobStatus.Err(); err != nil {
		return err
	}

	return nil
}

func (d *DataHubBigQuery) DeleteBigQueryDataByClouds(
	ctx context.Context,
	customerID string,
	deleteDatasetsReq domain.DeleteDatasetsReq,
	deletedBy string,
) error {
	l := d.loggerProvider(ctx)
	bq := d.conn.Bigquery(ctx)

	query := d.getBaseDeleteQuery(bq.Project())
	query += "AND cloud IN UNNEST(@clouds)"

	queryJob := bq.Query(query)

	queryJob.Parameters = []bigquery.QueryParameter{
		{Name: "customer_id", Value: customerID},
		{Name: "clouds", Value: deleteDatasetsReq.Datasets},
		{Name: "deleted_by", Value: deletedBy},
	}

	queryJob.JobIDConfig = getJobIDConfig(deleteJobIDPrefix, customerID)
	queryJob.Labels = getQueryJobLabels(ctx, customerID)

	job, err := queryJob.Run(ctx)
	if err != nil {
		return err
	}

	l.Info(job.ID())

	ctxWait, cancelWait := context.WithTimeout(ctx, time.Minute)
	defer cancelWait()

	status, err := job.Wait(ctxWait)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			l.Debug("query running for too long; abort.")

			return ErrDeleteTookTooLongRunningAsync

		}

		return err
	}

	if err := status.Err(); err != nil {
		return err
	}

	return nil
}

func (d *DataHubBigQuery) DeleteBigQueryDataByBatches(
	ctx context.Context,
	customerID string,
	datasetName string,
	deleteReq domain.DeleteBatchesReq,
	deletedBy string,
) error {
	l := d.loggerProvider(ctx)
	bq := d.conn.Bigquery(ctx)

	query := d.getBaseDeleteQuery(bq.Project())
	query += "AND cloud = @dataset AND batch IN UNNEST(@batches)"

	queryJob := bq.Query(query)

	queryJob.Parameters = []bigquery.QueryParameter{
		{Name: "customer_id", Value: customerID},
		{Name: "dataset", Value: datasetName},
		{Name: "batches", Value: deleteReq.Batches},
		{Name: "deleted_by", Value: deletedBy},
	}

	queryJob.JobIDConfig = getJobIDConfig(deleteJobIDPrefix, customerID)
	queryJob.Labels = getQueryJobLabels(ctx, customerID)

	job, err := queryJob.Run(ctx)
	if err != nil {
		return err
	}

	l.Info(job.ID())

	jobStatus, err := job.Status(ctx)
	if err != nil {
		return err
	}

	if err := jobStatus.Err(); err != nil {
		return err
	}

	return nil
}

func (d *DataHubBigQuery) DeleteBigQueryDataHard(
	ctx context.Context,
	customerID string,
	softDeleteIntervalDays int,
) error {
	l := d.loggerProvider(ctx)
	bq := d.conn.Bigquery(ctx)

	query := d.getBaseDeleteHardQuery(bq.Project(), softDeleteIntervalDays)

	queryJob := bq.Query(query)

	queryJob.Parameters = []bigquery.QueryParameter{
		{Name: "customer_id", Value: customerID},
	}

	queryJob.JobIDConfig = getJobIDConfig(deleteJobIDPrefix, customerID)
	queryJob.Labels = getQueryJobLabels(ctx, customerID)

	job, err := queryJob.Run(ctx)
	if err != nil {
		return err
	}

	l.Info(job.ID())

	jobStatus, err := job.Status(ctx)
	if err != nil {
		return err
	}

	if err := jobStatus.Err(); err != nil {
		return err
	}

	return nil
}

func (d *DataHubBigQuery) GetCustomerDatasets(
	ctx context.Context,
	customerID string,
) ([]domain.CachedDataset, error) {
	l := d.loggerProvider(ctx)
	bq := d.conn.Bigquery(ctx)

	query := d.getCustomerDatasetsQuery(bq.Project())

	queryJob := bq.Query(query)

	queryJob.Parameters = []bigquery.QueryParameter{
		{Name: "customer_id", Value: customerID},
	}

	queryJob.JobIDConfig = getJobIDConfig(getSummaryJobIDPrefix, customerID)
	queryJob.Labels = getQueryJobLabels(ctx, customerID)

	job, err := queryJob.Run(ctx)
	if err != nil {
		return nil, err
	}

	l.Info(job.ID())

	jobStatus, err := job.Status(ctx)
	if err != nil {
		return nil, err
	}

	if err := jobStatus.Err(); err != nil {
		return nil, err
	}

	rowIterator, err := job.Read(ctx)
	if err != nil {
		return nil, err
	}

	return doitBigquery.LoadRows[domain.CachedDataset](rowIterator)
}

func (d *DataHubBigQuery) GetCustomerDatasetBatches(
	ctx context.Context,
	customerID string,
	datasetName string,
) ([]domain.DatasetBatch, error) {
	l := d.loggerProvider(ctx)
	bq := d.conn.Bigquery(ctx)

	query := d.getCustomerDatasetBatchesQuery(bq.Project())

	queryJob := bq.Query(query)

	queryJob.Parameters = []bigquery.QueryParameter{
		{Name: "customer_id", Value: customerID},
		{Name: "dataset", Value: datasetName},
	}

	queryJob.JobIDConfig = getJobIDConfig(getSummaryJobIDPrefix, customerID)
	queryJob.Labels = getQueryJobLabels(ctx, customerID)

	job, err := queryJob.Run(ctx)
	if err != nil {
		return nil, err
	}

	l.Info(job.ID())

	jobStatus, err := job.Status(ctx)
	if err != nil {
		return nil, err
	}

	if err := jobStatus.Err(); err != nil {
		return nil, err
	}

	rowIterator, err := job.Read(ctx)
	if err != nil {
		return nil, err
	}

	return doitBigquery.LoadRows[domain.DatasetBatch](rowIterator)
}

func (d *DataHubBigQuery) GetCustomersWithSoftDeleteData(
	ctx context.Context,
	softDeleteIntervalDays int,
) ([]string, error) {
	l := d.loggerProvider(ctx)
	bq := d.conn.Bigquery(ctx)

	query := d.getCustomersWithSoftDeleteQuery(bq.Project(), softDeleteIntervalDays)

	queryJob := bq.Query(query)

	queryJob.JobIDConfig = getJobIDConfig(getCustomersWithSoftDeleteDataJobIDPrefix, "")
	queryJob.Labels = getQueryJobLabels(ctx, "")

	job, err := queryJob.Run(ctx)
	if err != nil {
		return nil, err
	}

	l.Info(job.ID())

	jobStatus, err := job.Status(ctx)
	if err != nil {
		return nil, err
	}

	if err := jobStatus.Err(); err != nil {
		return nil, err
	}

	rowIterator, err := job.Read(ctx)
	if err != nil {
		return nil, err
	}

	customersData, err := doitBigquery.LoadRows[domain.CustomerWithSoftDeleteData](rowIterator)
	if err != nil {
		return nil, err
	}

	var customerIDs []string
	for _, c := range customersData {
		customerIDs = append(customerIDs, c.CustomerID)
	}

	return customerIDs, nil
}

func (d *DataHubBigQuery) getBaseDeleteQuery(projectID string) string {
	return fmt.Sprintf(`UPDATE
		%s
	SET
		delete.time = DATETIME(CURRENT_TIMESTAMP()),
		delete.deleted_by = @deleted_by
	WHERE
		customer_id = @customer_id
		AND TIMESTAMP(export_time) < TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 90 MINUTE)
		`, d.GetEventsTable(projectID))
}

func getQueryJobLabels(ctx context.Context, customerID string) map[string]string {
	house, feature, module := domainOrigin.MapOriginToHouseFeatureModule(domainOrigin.QueryOriginFromContext(ctx))

	jobLabels := map[string]string{
		common.LabelKeyEnv.String():     common.GetEnvironmentLabel(),
		common.LabelKeyHouse.String():   house.String(),
		common.LabelKeyFeature.String(): feature.String(),
		common.LabelKeyModule.String():  module.String(),
	}

	if customerID != "" {
		jobLabels[common.LabelKeyCustomer.String()] = strings.ToLower(customerID)
	}

	return jobLabels
}

// getJobIDConfig sets the job ID for the query job.
func getJobIDConfig(prefix, customerID string) bigquery.JobIDConfig {
	if customerID == "" {
		return bigquery.JobIDConfig{
			JobID:          prefix,
			AddJobIDSuffix: true,
		}
	}

	return bigquery.JobIDConfig{
		JobID:          fmt.Sprintf("%s-%s", prefix, customerID),
		AddJobIDSuffix: true,
	}
}

func (d *DataHubBigQuery) getCustomerDatasetsQuery(projectID string) string {
	return fmt.Sprintf(`
		SELECT
			cloud,
			COUNT(*) as records,
			MAX(export_time) as lastUpdated,
			ANY_VALUE(updated_by HAVING MAX export_time) AS updatedBy
		FROM
			%s
		WHERE
			customer_id = @customer_id
			AND delete IS NULL
		GROUP BY
			cloud`, d.GetEventsTable(projectID))
}

func (d *DataHubBigQuery) getCustomerDatasetBatchesQuery(projectID string) string {
	return fmt.Sprintf(`
		SELECT
			source,
			batch,
			COUNT(*) AS records,
			MAX(export_time) as lastUpdated,
			ANY_VALUE(updated_by HAVING MAX export_time) AS updatedBy,
		FROM
			%s
		WHERE
			customer_id = @customer_id
		    AND cloud = @dataset
		    AND source = "csv"
		    AND delete IS NULL
		GROUP BY
			source, batch
	`, d.GetEventsTable(projectID))
}

func (d *DataHubBigQuery) getBaseDeleteHardQuery(projectID string, softDeleteIntervalDays int) string {
	return fmt.Sprintf(`
		DELETE
		FROM
			%s
		WHERE
			customer_id = @customer_id
		    AND
		       delete IS NOT NULL
		    AND
			   delete.time < DATE_SUB(CURRENT_DATE(), INTERVAL %d DAY)
	`, d.GetEventsTable(projectID), softDeleteIntervalDays)
}

func (d *DataHubBigQuery) getCustomersWithSoftDeleteQuery(
	projectID string,
	softDeleteIntervalDays int,
) string {
	return fmt.Sprintf(`
		SELECT
			customer_id
		FROM
			%s
		WHERE
		     delete.time IS NOT NULL
           AND
			 delete.time < DATE_SUB(CURRENT_DATE(), INTERVAL %d DAY)
		GROUP BY
			customer_id
	`, d.GetEventsTable(projectID), softDeleteIntervalDays)
}
