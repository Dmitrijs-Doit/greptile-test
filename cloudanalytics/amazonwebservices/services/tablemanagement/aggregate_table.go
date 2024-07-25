package tablemanagement

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/amazonwebservices/utils"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/service"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

func (s *BillingTableManagementService) UpdateAggregatedTable(
	ctx context.Context,
	suffix string,
	interval string,
	allPartitions bool,
) error {
	l := s.loggerProvider(ctx)

	if suffix == "" {
		return ErrSuffixIsEmpty
	}

	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		l.Warningf("no bq client found for origin, using default")
	}

	if exists, _, err := common.BigQueryDatasetExists(ctx, bq, utils.GetBillingProject(), utils.GetCustomerBillingDataset(suffix)); err != nil {
		return err
	} else if !exists {
		l.Warningf("destination dataset %s does not exist", utils.GetCustomerBillingDataset(suffix))
		return nil
	}

	query := getAggregatedQuery(allPartitions, suffix, false, interval)
	l.Debug(query)

	configJobID := fmt.Sprintf("%s-%s-%s", "cloud_analytics_aws_update_aggregated_table", interval, suffix)

	l.Infof("REPORT STATUS: Updating AWS aggregated tables for %s", suffix)

	if err := service.RunBillingTableUpdateQuery(ctx, bq, query,
		&domain.BigQueryTableUpdateRequest{
			DefaultProjectID:     utils.GetBillingProject(),
			DefaultDatasetID:     utils.GetCustomerBillingDataset(suffix),
			DestinationProjectID: utils.GetBillingProject(),
			DestinationDatasetID: utils.GetCustomerBillingDataset(suffix),
			DestinationTableName: utils.GetCustomerBillingTable(suffix, interval),
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
		l.Errorf("Error updating aggregated AWS table: %s", err)
		return err
	}

	l.Infof("REPORT STATUS: Updating AWS report status for %s", suffix)

	if err := s.reportStatusService.UpdateReportStatus(ctx, suffix, common.ReportStatus{
		Status: map[string]common.StatusInfo{
			string(common.AWSReportStatus): {
				LastUpdate: time.Now(),
			},
		},
	}); err != nil {
		l.Error(err)
	}

	return nil
}

func (s *BillingTableManagementService) UpdateAllAggregatedTablesAllCustomers(
	ctx context.Context,
	allPartitions bool,
) []error {
	l := s.loggerProvider(ctx)

	var errs []error

	docSnaps, err := s.customerDAL.GetAWSCustomers(ctx)
	if err != nil {
		l.Error("could not retrieve customers: %s", err)
		errs = append(errs, err)

		return errs
	}

	for _, docSnap := range docSnaps {
		isRecalculated, err := common.GetCustomerIsRecalculatedFlag(ctx, docSnap.Ref)
		if err != nil {
			l.Errorf("could not check isRecalculated: %s", err)
			errs = append(errs, err)

			continue
		}

		var suffixes []string
		if isRecalculated {
			suffixes = append(suffixes, docSnap.Ref.ID)
		} else {
			chtDocSnaps, err := s.customerDAL.GetCloudhealthCustomers(ctx, docSnap.Ref)
			if err != nil {
				l.Errorf("could not retrieve cloudhealth documents: %s", err)
				errs = append(errs, err)

				continue
			}

			if len(chtDocSnaps) == 0 {
				l.Warningf("could not find customer %s in cht customers collection", docSnap.Ref.ID)
				continue
			}

			for _, chtDocSnap := range chtDocSnaps {
				suffixes = append(suffixes, chtDocSnap.Ref.ID)
			}
		}

		for _, suffix := range suffixes {
			if err := CreateAggregatedTableTask(ctx, suffix, allPartitions); err != nil {
				l.Errorf("could not create aggregate task: %s", err)
				errs = append(errs, err)
			}
		}
	}

	return errs
}

// CreateAggregatedTableTask creates task to aggregate AWS customer
func CreateAggregatedTableTask(ctx context.Context, suffix string, allPartitions bool) error {
	if suffix == "" {
		return errors.New("invalud empty suffix")
	}

	path := "/tasks/analytics/amazon-web-services/accounts/" + suffix + "/aggregate-all"
	if allPartitions {
		path += "?allPartitions=true"
	}

	config := common.CloudTaskConfig{
		Method: cloudtaskspb.HttpMethod_POST,
		Path:   path,
		Queue:  common.TaskQueueCloudAnalyticsTablesAWS,
	}

	_, err := common.CreateCloudTask(ctx, &config)

	return err
}

func (s *BillingTableManagementService) UpdateAllAggregatedTables(
	ctx context.Context,
	suffix string,
	allPartitions bool,
) []error {
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
