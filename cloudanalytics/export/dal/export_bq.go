package dal

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/googleapi"
)

const (
	DatasetAWS = "aws_billing"
)

type BigQueryDAL struct {
	viewsClient       *bigquery.Client
	billingDataClient *bigquery.Client
}

func NewBigQueryDAL(ctx context.Context, viewsProject, billingDataProject string) *BigQueryDAL {
	viewsClient, err := bigquery.NewClient(ctx, viewsProject)
	if err != nil {
		log.Fatalf("Failed to create viewsClient client: %v", err)
	}

	billingDataClient, err := bigquery.NewClient(ctx, billingDataProject)
	if err != nil {
		log.Fatalf("Failed to create billingDataClient client: %v", err)
	}

	return &BigQueryDAL{viewsClient: viewsClient, billingDataClient: billingDataClient}
}

func (d *BigQueryDAL) CheckViewExists(ctx context.Context, customerID string) (bool, error) {
	datasetRef := d.viewsClient.Dataset(getAWSBillingDatasetID(customerID))
	_, err := datasetRef.Metadata(ctx)
	if err != nil {
		// create the dataset
		metaData := &bigquery.DatasetMetadata{}
		err = datasetRef.Create(ctx, metaData)
		if err != nil {
			return false, err
		}
		return false, nil
	}

	tableRef := datasetRef.Table(getAWSBillingTableID(customerID))
	_, err = tableRef.Metadata(ctx)
	if err != nil {
		if e, ok := err.(*googleapi.Error); ok {
			if e.Code == http.StatusNotFound {
				return false, nil
			}
		}
		return false, err
	}

	return true, nil
}

func (d *BigQueryDAL) CreateViewAWS(ctx context.Context, customerID string) error {
	datasetRef := d.viewsClient.Dataset(getAWSBillingDatasetID(customerID))
	viewRef := datasetRef.Table(getAWSBillingTableID(customerID))

	sourceTable := fmt.Sprintf("%s.%s.%s", d.billingDataClient.Project(), getAWSBillingDatasetID(customerID), getAWSBillingTableID(customerID))

	query := `
	SELECT
		billing_account_id,
		project_id,
		service_description,
		service_id,
		sku_description,
		sku_id,
		usage_date_time,
		usage_start_time,
		usage_end_time,
		labels,
		system_labels,
		location,
		export_time,
		cost,
		currency,
		currency_conversion_rate,
		usage,
		invoice,
		cost_type,
		STRUCT(report[OFFSET(0)].cost AS cost, report[OFFSET(0)].usage AS usage, report[OFFSET(0)].savings AS savings, report[OFFSET(0)].savings_description AS savings_description, report[OFFSET(0)].credit AS credit) AS report,
		resource_id,
		operation,
		aws_metric,
		row_id,
		description
	FROM
		` + sourceTable

	metaData := &bigquery.TableMetadata{
		ViewQuery: query,
	}

	return viewRef.Create(ctx, metaData)
}

func (d *BigQueryDAL) AuthorizeView(ctx context.Context, customerID, customerEmail string) error {
	srcDataset := d.billingDataClient.Dataset(getAWSBillingDatasetID(customerID))
	viewDataset := d.viewsClient.Dataset(getAWSBillingDatasetID(customerID))
	view := viewDataset.Table(getAWSBillingTableID(customerID))

	// Add the customer email to the ACL for the dataset containing the view
	vMeta, err := viewDataset.Metadata(ctx)
	if err != nil {
		return err
	}

	vUpdateMeta := bigquery.DatasetMetadataToUpdate{
		Access: append(vMeta.Access, &bigquery.AccessEntry{
			Role:       bigquery.ReaderRole,
			EntityType: bigquery.UserEmailEntity,
			Entity:     customerEmail,
		}),
	}
	if _, err := viewDataset.Update(ctx, vUpdateMeta, vMeta.ETag); err != nil {
		return err
	}

	// Authorize the view against a source dataset
	srcMeta, err := srcDataset.Metadata(ctx)
	if err != nil {
		return err
	}
	srcUpdateMeta := bigquery.DatasetMetadataToUpdate{

		Access: append(srcMeta.Access, &bigquery.AccessEntry{
			EntityType: bigquery.ViewEntity,
			View:       view,
		}),
	}
	if _, err := srcDataset.Update(ctx, srcUpdateMeta, srcMeta.ETag); err != nil {
		return err
	}
	return nil
}

func getAWSBillingDatasetID(customerID string) string {
	return fmt.Sprintf("%s_%s", DatasetAWS, customerID)
}

func getAWSBillingTableID(customerID string) string {
	return fmt.Sprintf("doitintl_billing_export_v1_%s", customerID)
}
