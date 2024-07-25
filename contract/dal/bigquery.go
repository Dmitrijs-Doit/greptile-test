package dal

import (
	"context"
	"fmt"

	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/contract/domain"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

const projectID = "doitintl-cmp-gcp-data"
const gcpDataset = "gcp_billing"
const gcpTable = "gcp_raw_billing"
const supportServiceID = "2062-016F-44A2"

type BigQuery struct {
	BigQueryClientFun connection.BigQueryFromContextFun
}

func NewContractBigqueryWithClient(fun connection.BigQueryFromContextFun) *BigQuery {
	return &BigQuery{
		BigQueryClientFun: fun,
	}
}

func (s *BigQuery) GetBillingAccountsSKU(ctx context.Context, startDate string, endDate string) ([]domain.SKUBillingRecord, error) {
	billingAccountsSKUQuery := s.BuildBillingAccountsSKUQuery(startDate, endDate)
	query := s.BigQueryClientFun(ctx).Query(billingAccountsSKUQuery)
	query.Labels = map[string]string{
		common.LabelKeyHouse.String():   common.HouseData.String(),
		common.LabelKeyEnv.String():     common.GetEnvironmentLabel(),
		common.LabelKeyModule.String():  "contracts",
		common.LabelKeyFeature.String(): "update-gcp-contracts-support",
	}

	iter, err := query.Read(ctx)
	if err != nil {
		return nil, err
	}

	rows := make([]domain.SKUBillingRecord, 0)

	var row domain.SKUBillingRecord

	for {
		err = iter.Next(&row)
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		rows = append(rows, row)
	}

	return rows, nil
}

func (s *BigQuery) BuildBillingAccountsSKUQuery(startDate string, endDate string) string {
	return fmt.Sprintf(
		`SELECT billing_account_id, sku.id AS sku_id, TIMESTAMP(MAX(usage_end_time)) AS latest_usage_date
		FROM %s.%s.%s
		WHERE usage_end_time BETWEEN '%s' AND '%s'
		AND service.id = '%s'
		GROUP BY billing_account_id, sku_id
		ORDER BY latest_usage_date, billing_account_id DESC`,
		projectID, gcpDataset, gcpTable, startDate, endDate, supportServiceID)
}
