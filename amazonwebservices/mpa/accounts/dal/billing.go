package dal

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal/sql"
)

type BillingDAL struct {
	bigqueryClient *bigquery.Client
}

// NewBillingDAL returns a new BillingDAL instance with given project id.
func NewBillingDAL(ctx context.Context, projectID string) (*BillingDAL, error) {
	bq, err := bigquery.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewBillingDALWithClient(bq), nil
}

// NewBillingDALWithClient returns a new BillingDAL using given BigQuery client
func NewBillingDALWithClient(bq *bigquery.Client) *BillingDAL {
	return &BillingDAL{
		bigqueryClient: bq,
	}
}

func (d *BillingDAL) GetCoveredUsage(
	ctx context.Context,
	accountID, payerID string,
	payerNumber int,
	spARNs, riAccountIDs []string,
) (CoveredUsage, error) {
	q := fmt.Sprintf(sql.CoveredUsage, payerNumber, payerID, riARNsFilter(riAccountIDs))
	query := d.bigqueryClient.Query(q)

	today := time.Now().UTC()
	exportDate := today.AddDate(0, 0, -3).Format("2006-01-02")

	query.Parameters = []bigquery.QueryParameter{
		{Name: "export_date", Value: exportDate},
		{Name: "account_id", Value: accountID},
		{Name: "sp_arns", Value: spARNs},
	}

	iter, err := query.Read(ctx)
	if err != nil {
		return CoveredUsage{}, err
	}

	var result CoveredUsage

	err = iter.Next(&result)
	if err != nil {
		return CoveredUsage{}, err
	}

	return result, nil
}

func riARNsFilter(riAccountIDs []string) string {
	if len(riAccountIDs) == 0 {
		return ""
	}

	filter := "OR (" + riARNFilter(riAccountIDs[0])

	for i := 1; i < len(riAccountIDs); i++ {
		filter += " OR " + riARNFilter(riAccountIDs[i])
	}

	filter += ")"

	return filter
}

func riARNFilter(riAccountID string) string {
	return fmt.Sprintf("getKeyFromSystemLabels(system_labels, 'aws/ri_arn') LIKE '%%%s%%'", riAccountID)
}
