package googlecloud

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	tableMgmtDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/service"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

func (s *BillingTableManagementService) UpdateAggregatedTable(
	ctx context.Context,
	billingAccountID string,
	interval string,
	fromDate string,
	numPartitions int,
	allPartitions bool,
) error {
	l := s.loggerProvider(ctx)

	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		l.Warningf("no bq client found for origin, using default")
	}

	if exists, _, err := common.BigQueryDatasetExists(ctx, bq, domain.GetBillingProject(), domain.GetCustomerBillingDataset(billingAccountID)); err != nil {
		return fmt.Errorf("error checking if dataset exists; %v", err)
	} else if !exists {
		l.Warningf("destination dataset %s does not exist", domain.GetCustomerBillingDataset(billingAccountID))
		return nil
	}

	query := domain.GetAggregatedQuery(allPartitions, billingAccountID, false, interval)
	l.Debug(query)

	configJobID := fmt.Sprintf("%s-%s-%s", "cloud_analytics_gcp_update_aggregated_table", interval, billingAccountID)

	return service.RunBillingTableUpdateQuery(ctx, bq, query,
		&tableMgmtDomain.BigQueryTableUpdateRequest{
			DefaultProjectID:      domain.GetBillingProject(),
			DefaultDatasetID:      domain.GetCustomerBillingDataset(billingAccountID),
			DestinationProjectID:  domain.GetBillingProject(),
			DestinationDatasetID:  domain.GetCustomerBillingDataset(billingAccountID),
			DestinationTableName:  domain.GetCustomerBillingTable(billingAccountID, interval),
			AllPartitions:         allPartitions,
			WriteDisposition:      bigquery.WriteTruncate,
			ConfigJobID:           configJobID,
			WaitTillDone:          true,
			CSP:                   false,
			FromDate:              fromDate,
			FromDateNumPartitions: numPartitions,
			Clustering:            service.GetTableClustering(false),

			House:   common.HouseAdoption,
			Feature: common.FeatureCloudAnalytics,
			Module:  common.ModuleTableManagement,
		})
}

func (s *BillingTableManagementService) UpdateAllAggregatedTables(
	ctx context.Context,
	billingAccountID string,
	fromDate string,
	numPartitions int,
	allPartitions bool,
) []error {
	l := s.loggerProvider(ctx)

	var errs []error

	for _, interval := range query.GetAggregatedTableIntervals() {
		err := s.UpdateAggregatedTable(ctx, billingAccountID, interval, fromDate, numPartitions, allPartitions)
		if err != nil {
			l.Errorf("failed to update aggregated table; %v", err)
			errs = append(errs, err)
		}
	}

	return errs
}
