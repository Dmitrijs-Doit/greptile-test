package tablemanagement

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/microsoftazure"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/service"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

const (
	configJobIDPrefix = "cloud_analytics_azure_update_aggregated_table"
	taskPathTemplate  = "/tasks/analytics/microsoft-azure/customers/%s/aggregate-all"
)

func (s *BillingTableManagementService) UpdateAggregatedTable(ctx context.Context, suffix, interval string, allPartitions bool) error {
	l := s.loggerProvider(ctx)

	if suffix == "" {
		return ErrSuffixIsEmpty
	}

	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		l.Warningf("no bq client found for origin, using default")
	}

	if exists, _, err := common.BigQueryDatasetExists(ctx, bq, microsoftazure.GetBillingProject(), microsoftazure.GetCustomerBillingDataset(suffix)); err != nil {
		return err
	} else if !exists {
		l.Warningf("destination dataset %s does not exist", microsoftazure.GetCustomerBillingDataset(suffix))
		return nil
	}

	aggregatedQuery := getAggregatedQuery(allPartitions, suffix, false, interval)
	l.Debug(aggregatedQuery)

	configJobID := fmt.Sprintf("%s-%s-%s", configJobIDPrefix, interval, suffix)

	l.Infof("REPORT STATUS: Updating MS AZURE aggregated tables for %s", suffix)

	if err := service.RunBillingTableUpdateQuery(ctx, bq, aggregatedQuery,
		&domain.BigQueryTableUpdateRequest{
			DefaultProjectID:     microsoftazure.GetBillingProject(),
			DefaultDatasetID:     microsoftazure.GetCustomerBillingDataset(suffix),
			DestinationProjectID: microsoftazure.GetBillingProject(),
			DestinationDatasetID: microsoftazure.GetCustomerBillingDataset(suffix),
			DestinationTableName: microsoftazure.GetCustomerBillingTable(suffix, interval),
			AllPartitions:        allPartitions,
			WriteDisposition:     bigquery.WriteTruncate,
			ConfigJobID:          configJobID,
			WaitTillDone:         false,
			CSP:                  false,
			Clustering:           service.GetTableClustering(false),

			House:   common.HouseAdoption,
			Feature: common.FeatureCloudAnalytics,
			Module:  common.ModuleTableManagement,
		}); err != nil {
		l.Errorf("error updating aggregated ms azure table: %v table %v: ", err, microsoftazure.GetFullCustomerBillingTable(suffix, interval))
		return err
	}

	l.Infof("REPORT STATUS: Updating MS Azure report status for %s", suffix)

	if err := s.reportStatusService.UpdateReportStatus(ctx, suffix, common.ReportStatus{
		Status: map[string]common.StatusInfo{
			common.Assets.MicrosoftAzure: {
				LastUpdate: time.Now(),
			},
		},
	}); err != nil {
		l.Error(err)
	}

	return nil
}

func (s *BillingTableManagementService) UpdateAllAggregatedTablesAllCustomers(ctx context.Context, allPartitions bool) []error {
	l := s.loggerProvider(ctx)

	var errs []error

	docSnaps, err := s.customerDAL.GetMSAzureCustomers(ctx)
	if err != nil {
		l.Error("could not retrieve customers: %s", err)
		return []error{err}
	}

	for _, docSnap := range docSnaps {
		if err := s.createAggregatedTableTask(ctx, docSnap.Ref.ID, allPartitions); err != nil {
			l.Errorf("could not create aggregate task: %s", err)
			errs = append(errs, err)
		}
	}

	return errs
}

// createAggregatedTableTask creates task to aggregate Microsoft Azure customer
func (s *BillingTableManagementService) createAggregatedTableTask(ctx context.Context, suffix string, allPartitions bool) error {
	if suffix == "" {
		return ErrSuffixIsEmpty
	}

	path := fmt.Sprintf(taskPathTemplate, suffix)
	if allPartitions {
		path += "?allPartitions=true"
	}

	config := common.CloudTaskConfig{
		Method: cloudtaskspb.HttpMethod_GET,
		Path:   path,
		Queue:  common.TaskQueueCloudAnalyticsTablesAzure,
	}

	_, err := s.conn.CloudTaskClient.CreateTask(ctx, config.Config(nil))

	return err
}

func (s *BillingTableManagementService) UpdateAllAggregatedTables(ctx context.Context, suffix string, allPartitions bool) []error {
	l := s.loggerProvider(ctx)

	var errs []error

	for _, interval := range query.GetAggregatedTableIntervals() {
		if err := s.UpdateAggregatedTable(ctx, suffix, interval, allPartitions); err != nil {
			l.Errorf("could not update aggregated table: %s", err)
			errs = append(errs, err)
		}
	}

	return errs
}
