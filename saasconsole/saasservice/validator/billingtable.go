package validator

import (
	"context"
	"fmt"

	"github.com/doitintl/firestore/pkg"
	awsUtils "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/amazonwebservices/utils"
	gcpUtils "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	"google.golang.org/api/iterator"
)

func (s *SaaSConsoleValidatorService) AccountHasLatestBillingData(ctx context.Context, platform pkg.StandalonePlatform, customerID, accountID string) bool {
	logger := s.loggerProvider(ctx)

	query := s.getLatestBillingDataQuery(ctx, platform, customerID, accountID)

	logger.Debugf("%squery: %s", billingLogPrefix, query)

	queryJob := s.conn.Bigquery(ctx).Query(query)

	it, err := queryJob.Read(ctx)
	if err != nil {
		logger.Errorf("%scouldn't query latest billing data for %s, account id %s, %s", billingLogPrefix, customerID, accountID, err)
		return false
	}

	var value struct {
		Count int `bigquery:"numRows"`
	}

	for {
		err := it.Next(&value)
		if err == iterator.Done {
			break
		}

		if err != nil {
			logger.Errorf("%scouldn't query latest billing data for %s, account id %s, %s", billingLogPrefix, customerID, accountID, err)
			return false
		}

		if value.Count > 0 {
			return true
		}
	}

	return false
}

func (s *SaaSConsoleValidatorService) getLatestBillingDataQuery(ctx context.Context, platform pkg.StandalonePlatform, customerID, accountID string) string {
	query := "SELECT count(*) as numRows FROM"

	switch platform {
	case pkg.GCP:
		query = fmt.Sprintf("%s `%s.%s.%s` WHERE", query, gcpUtils.GetBillingProject(), gcpUtils.GetCustomerBillingDataset(accountID), gcpUtils.GetCustomerBillingTable(accountID, ""))
	case pkg.AWS:
		query = fmt.Sprintf("%s `%s.%s.%s` WHERE project_id = \"%s\" AND", query, awsUtils.GetBillingProject(), awsUtils.GetCustomerBillingDataset(customerID), awsUtils.GetCustomerBillingTable(customerID, ""), accountID)
	default:
	}

	query = fmt.Sprintf("%s DATE(export_time) >= DATE_ADD(CURRENT_DATE(), INTERVAL -3 DAY)", query)

	return query
}
