package dal

import (
	"bytes"
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type PlesBigQueryDal struct{}

const (
	ProdProjectID   = "doitintl-cmp-aws-data"
	DevProjectID    = "cmp-aws-etl-dev"
	AccountsDataset = "accounts"
	AccountsTable   = "ples_accounts_map"
)

func getBigQueryClient(ctx context.Context) (*bigquery.Client, error) {
	projectID := DevProjectID
	if common.Production {
		projectID = ProdProjectID
	}

	bq, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return bq, nil
}

func (d *PlesBigQueryDal) UpdatePlesAccounts(ctx context.Context, accounts *bytes.Buffer, monthPartition string) error {
	bqClient, err := getBigQueryClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to create bigquery client: %w", err)
	}

	defer bqClient.Close()

	source := bigquery.NewReaderSource(accounts)
	source.SkipLeadingRows = 1
	source.Schema = bigquery.Schema{
		{Name: "account_name", Type: bigquery.StringFieldType},
		{Name: "account_id", Type: bigquery.StringFieldType},
		{Name: "support_level", Type: bigquery.StringFieldType},
		{Name: "payer_id", Type: bigquery.StringFieldType},
		{Name: "invoice_month", Type: bigquery.TimestampFieldType},
		{Name: "update_time", Type: bigquery.TimestampFieldType},
	}

	loader := bqClient.Dataset(AccountsDataset).Table(AccountsTable + "$" + monthPartition).LoaderFrom(source)
	loader.WriteDisposition = bigquery.WriteTruncate

	job, err := loader.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to run loader job: %w", err)
	}

	status, err := job.Wait(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait loader job: %w", err)
	}

	if err := status.Err(); err != nil {
		return fmt.Errorf("error in job status: %w", err)
	}

	return nil
}
