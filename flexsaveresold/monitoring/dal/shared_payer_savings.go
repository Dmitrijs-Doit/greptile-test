package dal

import (
	"context"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"

	sharedBQ "github.com/doitintl/bigquery"
	"github.com/doitintl/bigquery/iface"
	"github.com/doitintl/errors"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/monitoring/domain"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	sharedPayerFlexsaveBillingTableView = "aws_custom_billing.aws_custom_billing_export_recent"
	savingsDiscrepancyThreshold         = -5.0
)

type sharedPayerSavings struct {
	client         *bigquery.Client
	projectID      string
	queryHandler   iface.QueryHandler
	loggerProvider logger.Provider
}

type SharedPayerSavings interface {
	DetectSharedPayerSavingsDiscrepancies(ctx context.Context, date string) (domain.SharedPayerSavingsDiscrepancies, error)
}

func NewSharedPayerSavings(ctx context.Context, projectID string, loggerProvider logger.Provider) (SharedPayerSavings, error) {
	bq, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	if projectID == "" {
		return nil, errors.New("no project id provided")
	}

	return &sharedPayerSavings{
		client:         bq,
		projectID:      projectID,
		queryHandler:   sharedBQ.QueryHandler{},
		loggerProvider: loggerProvider,
	}, nil
}

// A shared payer saving discrepancy is classified as an instance where a customer had Flexsave savings
// [indicated by the presence of billing line items with CostType "Flexsave"] of over
// 'savingsDiscrepancyThreshold' last month but has zero this month.
func (s sharedPayerSavings) DetectSharedPayerSavingsDiscrepancies(ctx context.Context, currentDate string) (domain.SharedPayerSavingsDiscrepancies, error) {
	query := s.client.Query(`
	WITH Discrepancies AS (
    	SELECT
			customer AS customer_id,
        SUM(CASE
                WHEN DATE_TRUNC(usage_date_time, MONTH) = DATE_TRUNC(@currentDate, MONTH)
                THEN cost ELSE 0 END) AS current_month_cost,
        SUM(CASE
                WHEN DATE_TRUNC(usage_date_time, MONTH) = DATE_TRUNC(DATE_SUB(@currentDate, INTERVAL 1 MONTH), MONTH)
                THEN cost ELSE 0 END) AS previous_month_cost
    	FROM
        	` + sharedPayerFlexsaveBillingTableView + `
    	WHERE
			usage_date_time >= DATE_TRUNC(DATE_SUB(@currentDate, INTERVAL 1 MONTH), MONTH)
		GROUP BY
        	customer
		)
	SELECT
		customer_id,
		-1 * previous_month_cost AS last_month_savings,
	FROM
		Discrepancies
	WHERE
		current_month_cost >= 0 AND
		previous_month_cost < @savingsDiscrepancyThreshold
	`)

	query.Parameters = []bigquery.QueryParameter{
		{
			Name:  "currentDate",
			Value: currentDate,
		},
		{
			Name:  "savingsDiscrepancyThreshold",
			Value: savingsDiscrepancyThreshold,
		},
	}

	it, err := s.queryHandler.Read(ctx, query)
	if err != nil {
		return nil, err
	}

	var discrepancies domain.SharedPayerSavingsDiscrepancies

	for {
		var row domain.SharedPayerSavingsDiscrepancy

		err = it.Next(&row)
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			return nil, err
		}

		discrepancies = append(discrepancies, row)
	}

	return discrepancies, nil
}
