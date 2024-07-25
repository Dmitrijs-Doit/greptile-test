package dal

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/billing-explainer/domain"
)

type BigQueryDAL struct {
	client *bigquery.Client
}

func NewBigQueryDAL(ctx context.Context, projectID string) *BigQueryDAL {
	client, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	return &BigQueryDAL{client: client}
}

func (d *BigQueryDAL) GetInvoiceSummary(ctx context.Context, explainerParams domain.BillingExplainerParams, payerTable, accountIDString, PayerID, flexsaveCondition string) ([]domain.SummaryBQ, error) {
	var queryString = GetSummaryPageQuery(explainerParams.CustomerTable, payerTable, accountIDString, PayerID, flexsaveCondition)

	query := d.client.Query(queryString)
	query.Parameters = []bigquery.QueryParameter{
		{Name: "startDateTime", Value: explainerParams.StartOfMonth},
		{Name: "endDateTime", Value: explainerParams.EndOfMonth},
	}
	query.JobIDConfig = bigquery.JobIDConfig{
		JobID:          "be-summary",
		AddJobIDSuffix: true,
	}
	it, err := query.Read(ctx)

	if err != nil {
		return nil, err
	}

	var results []domain.SummaryBQ

	for {
		var row domain.SummaryBQ
		err := it.Next(&row)

		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		results = append(results, row)
	}

	return results, nil
}

func (d *BigQueryDAL) GetPayerIDFromAccountsHistory(ctx context.Context, startOfMonth string, customerID string) ([]domain.PayerAccountHistoryResult, error) {
	const baseQuery = `
        SELECT
            DISTINCT payer_id
            FROM
            %s
        WHERE
            DATE(timestamp) >= @startDateTime
            AND customer_id = @customerID
            AND account_id NOT IN (SELECT DISTINCT(aws_account_id) FROM %s)
    `

	accountsHistoryTableID := fmt.Sprintf("%s.%s.%s", "doitintl-cmp-aws-data", "accounts", "accounts_history")
	flexsaveAccountsTableID := fmt.Sprintf("%s.measurement.flexsave_accounts", "me-doit-intl-com")
	queryString := fmt.Sprintf(baseQuery, accountsHistoryTableID, flexsaveAccountsTableID)

	query := d.client.Query(queryString)
	query.Parameters = []bigquery.QueryParameter{
		{Name: "startDateTime", Value: startOfMonth},
		{Name: "customerID", Value: customerID},
	}
	query.JobIDConfig = bigquery.JobIDConfig{
		JobID:          "be-get-payerid-from-accounthistory",
		AddJobIDSuffix: true,
	}

	it, err := query.Read(ctx)

	if err != nil {
		return nil, err
	}

	var results []domain.PayerAccountHistoryResult

	for {
		var row domain.PayerAccountHistoryResult
		err := it.Next(&row)

		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		results = append(results, row)
	}

	return results, nil
}

func (d *BigQueryDAL) GetServiceBreakdownData(ctx context.Context, explainerParams domain.BillingExplainerParams, payerTable, accountIDString, PayerID, flexsaveCondition string) ([]domain.ServiceRecord, error) {
	var queryString = GetServiceBreakdownQuery(explainerParams.CustomerTable, payerTable, accountIDString, PayerID, flexsaveCondition)

	query := d.client.Query(queryString)
	query.Parameters = []bigquery.QueryParameter{
		{Name: "startDateTime", Value: explainerParams.StartOfMonth},
		{Name: "endDateTime", Value: explainerParams.EndOfMonth},
	}
	query.JobIDConfig = bigquery.JobIDConfig{
		JobID:          "be-service-breakdown",
		AddJobIDSuffix: true,
	}

	it, err := query.Read(ctx)
	if err != nil {
		return nil, err
	}

	var results []domain.ServiceRecord

	for {
		var row domain.ServiceRecord
		err := it.Next(&row)

		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		results = append(results, row)
	}

	return results, nil
}

func (d *BigQueryDAL) GetAccountBreakdownData(ctx context.Context, explainerParams domain.BillingExplainerParams, payerTable, accountIDString, PayerID, flexsaveCondition string) ([]domain.AccountRecord, error) {
	var queryString = GetAccountBreakdownQuery(explainerParams.CustomerTable, payerTable, accountIDString, PayerID, flexsaveCondition)

	query := d.client.Query(queryString)
	query.Parameters = []bigquery.QueryParameter{
		{Name: "startDateTime", Value: explainerParams.StartOfMonth},
		{Name: "endDateTime", Value: explainerParams.EndOfMonth},
	}
	query.JobIDConfig = bigquery.JobIDConfig{
		JobID:          "be-account-breakdown",
		AddJobIDSuffix: true,
	}

	it, err := query.Read(ctx)
	if err != nil {
		return nil, err
	}

	var results []domain.AccountRecord

	for {
		var row domain.AccountRecord
		err := it.Next(&row)

		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		results = append(results, row)
	}

	return results, nil
}
