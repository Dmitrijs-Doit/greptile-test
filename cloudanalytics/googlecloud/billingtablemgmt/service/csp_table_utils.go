package googlecloud

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"google.golang.org/api/googleapi"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/cspreport"
	cspReportDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/cspreport/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	tableMgmtDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/service"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

func (s *BillingTableManagementService) getCurrentCSPBillingTempTableName(
	ctx context.Context,
	data *domain.CSPBillingAccountUpdateData,
) (string, int, error) {
	l := s.loggerProvider(ctx)
	billingAccountID := data.BillingAccountID
	id := data.TaskID
	domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: ** getCurrentCSPBillingTempTableName **\n", id, billingAccountID), nil)

	domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: ** getOrIncTableIndex **\n", data.TaskID, data.BillingAccountID), nil)

	runID, idx, err := s.cspDAL.GetOrIncTableIndex(ctx, -1, data)
	if err != nil {
		return "", -1, err
	}

	var suffix string
	if data.Mode == domain.CSPUpdateAllMode {
		suffix = runID
	} else {
		suffix = strings.Replace(billingAccountID, "-", "_", -1)
	}

	return s.getCSPBillingTempTableName(suffix, idx), idx, err
}

func (s *BillingTableManagementService) getCSPBillingTempTableName(suffix string, idx int) string {
	if common.Production {
		return fmt.Sprintf("TMP_%s_doitintl_billing_export_v1_%d", suffix, idx)
	}

	return fmt.Sprintf("TMP_%s_doitintl_billing_export_v1beta_%d", suffix, idx)
}

func (s *BillingTableManagementService) cspAddRangeQueryParams(
	basePath string,
	data *domain.CSPBillingAccountsTableUpdateData,
) string {
	if data.AllPartitions {
		basePath += "&allPartitions=true"
	} else if data.FromDate != "" {
		basePath += "&from=" + data.FromDate

		if data.FromDateNumPartitions > 0 {
			basePath += "&numPartitions=" + strconv.Itoa(data.FromDateNumPartitions)
		}
	}

	return basePath
}

func (s *BillingTableManagementService) createGoogleCloudCSPBillingAccountTableUpdateTask(
	ctx context.Context,
	data *domain.CSPBillingAccountsTableUpdateData,
) error {
	var (
		recreateTasks []string
		failedTasks   []string
		uri           string
	)

	l := s.loggerProvider(ctx)
	baseURI := s.cspAddRangeQueryParams("/tasks/analytics/google-cloud/csp-accounts/%s?updateAll=true", data)

	for i, billingAccountID := range data.Accounts {
		uri := fmt.Sprintf(baseURI, billingAccountID)

		if abort, err := s.createTask(ctx, false, uri,
			&domain.CSPBillingAccountUpdateData{
				TaskID:           i,
				BillingAccountID: billingAccountID,
				TableUpdateData:  data,
			}); abort {
			return err
		} else if err != nil {
			domain.DebugPrintToLogs(l, fmt.Sprintf("Adding to recreate %s\n", billingAccountID), nil)
			recreateTasks = append(recreateTasks, billingAccountID)
		}
	}

	domain.DebugPrintToLogs(l, fmt.Sprintf("Reecreate %d tasks %v\n", len(recreateTasks), recreateTasks), nil)

	if len(recreateTasks) > 0 {
		domain.DebugPrintToLogs(l, fmt.Sprintf("Recreating %d tasks: %v", len(recreateTasks), recreateTasks), nil)

		for i, billingAccountID := range recreateTasks {
			uri = fmt.Sprintf(baseURI, billingAccountID)

			if abort, err := s.createTask(ctx, true, uri,
				&domain.CSPBillingAccountUpdateData{
					TaskID:           i,
					BillingAccountID: billingAccountID,
					TableUpdateData:  data,
				}); abort {
				return err
			} else if err != nil {
				failedTasks = append(failedTasks, billingAccountID)
			}
		}

		if len(failedTasks) > 0 {
			l.Errorf("Failed to create %d tasks: %v", len(failedTasks), failedTasks)
			return nil
		}
	}

	domain.DebugPrintToLogs(l, fmt.Sprintf("All tasks created.\n"), nil)

	return nil
}

func (s *BillingTableManagementService) createTask(
	ctx context.Context,
	lastTry bool,
	uri string,
	data *domain.CSPBillingAccountUpdateData,
) (bool, error) {
	l := s.loggerProvider(ctx)
	billingAccountID := data.BillingAccountID

	var fsData domain.CSPFirestoreData
	if err := s.cspDAL.GetFirestoreData(ctx, data, &fsData); err != nil {
		return false, err
	}

	if _, ok := fsData.Tasks[billingAccountID]; ok {
		domain.DebugPrintToLogs(l, fmt.Sprintf("%s: Tried launching task twice.", billingAccountID), nil)
		return false, nil
	}

	if _, err := s.cspDAL.SetTaskState(ctx, domain.TaskStateCreated, data); err != nil {
		switch err.(type) {
		case *domain.DuplicateTaskError:
			return false, nil
		default:
			domain.DebugPrintToLogs(l, fmt.Sprintf("%s: Error adding to launched\n", billingAccountID), err)
			return true, err
		}
	}

	config := common.CloudTaskConfig{
		Method:           cloudtaskspb.HttpMethod_POST,
		Path:             uri,
		Queue:            common.TaskQueueCloudAnalyticsCSP,
		DispatchDeadline: durationpb.New(6 * time.Hour),
	}

	_, err := s.conn.CloudTaskClient.CreateAppEngineTask(ctx, config.AppEngineConfig(nil))
	if err != nil {
		if _, err := s.cspDAL.SetTaskState(ctx, domain.TaskStateFailedToCreate, data); err != nil {
			domain.DebugPrintToLogs(l, fmt.Sprintf("%s: Error removing from launched\n", billingAccountID), err)
			return true, err
		}

		domain.DebugPrintToLogs(l, fmt.Sprintf("%s: Error creating task\n", billingAccountID), err)

		if lastTry {
			domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: ** decStillRunning **\n", data.TaskID, data.BillingAccountID), nil)

			if _, err = s.cspDAL.DecStillRunning(ctx, data); err != nil {
				domain.DebugPrintToLogs(l, fmt.Sprintf("%s: Error decreasing counter\n", billingAccountID), nil)
				return true, err
			}
		}

		return false, err
	}

	return false, nil
}

func (s *BillingTableManagementService) appendToTempCSPBillingAccountTable(
	ctx context.Context,
	data *domain.CSPBillingAccountUpdateData,
) error {
	l := s.loggerProvider(ctx)
	billingAccountID := data.BillingAccountID
	id := data.TaskID
	domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: ** appendToTempCSPBillingAccountTable **\n", id, billingAccountID), nil)

	shouldAppendData, err := s.shouldAppendToTempTable(ctx, data)
	if err != nil {
		return err
	}

	domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Processing.\n", id, billingAccountID), nil)

	if shouldAppendData {
		err = s.appendToTempTable(ctx, data)
	}

	if data.Mode == domain.CSPUpdateAllMode {
		if err != nil {
			if err := s.failedCopyTempTableIfFinished(ctx, data); err != nil {
				return err
			}
		} else {
			if err = s.succeededCopyTempTableIfFinished(ctx, shouldAppendData, data); err != nil {
				return err
			}
		}
	} else {
		if err != nil {
			return err
		}

		if err := s.createCopyTempTablesTask(ctx, data); err != nil {
			domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Failed wrapping up.\n", id, billingAccountID), err)
			return err
		}

		_, err = s.cspDAL.SetTaskState(ctx, domain.TaskStateProcessed, data)
		if err != nil {
			domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Error while adding to processed\n", id, billingAccountID), err)
		}

		domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: ** decStillRunning **\n", data.TaskID, data.BillingAccountID), nil)

		_, err = s.cspDAL.DecStillRunning(ctx, data)
		if err != nil {
			domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Error decreasing counter\n", id, billingAccountID), err)
		}

		domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Finished.\n", id, billingAccountID), nil)
	}

	return err
}

func (s *BillingTableManagementService) shouldAppendToTempTable(
	ctx context.Context,
	data *domain.CSPBillingAccountUpdateData,
) (bool, error) {
	l := s.loggerProvider(ctx)
	shouldAppendData := true

	if data.Mode == domain.CSPUpdateAllMode {
		shouldAppendData = false

		var fsData domain.CSPFirestoreData
		if err := s.cspDAL.GetFirestoreData(ctx, data, &fsData); err != nil {
			return false, err
		}

		if accounttData, ok := fsData.Tasks[data.BillingAccountID]; ok {
			if accounttData.State == domain.TaskStateProcessed {
				domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: ALREADY PROCESSED\n", data.TaskID, data.BillingAccountID), nil)
				return false, nil
			}

			if accounttData.BillingDataCopied == false {
				shouldAppendData = true
			}
		} else {
			shouldAppendData = true
		}
	}

	return shouldAppendData, nil
}

func (s *BillingTableManagementService) appendToTempTable(
	ctx context.Context,
	data *domain.CSPBillingAccountUpdateData,
) error {
	l := s.loggerProvider(ctx)
	billingAccountID := data.BillingAccountID
	id := data.TaskID

	tableName, idx, err := s.getCurrentCSPBillingTempTableName(ctx, data)
	if err != nil {
		domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Error fetching temp target table name.\n", id, billingAccountID), err)

		if err := s.failedCopyTempTableIfFinished(ctx, data); err != nil {
			domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Failed wrapping up.\n", id, billingAccountID), err)
			return err
		}

		return err
	}

	requestData := tableMgmtDomain.BigQueryTableUpdateRequest{
		ConfigJobID:           "cloud_analytics_csp_gcp_billing_account_" + billingAccountID,
		DefaultProjectID:      domain.BillingProjectProd,
		DefaultDatasetID:      domain.BillingDataset,
		DestinationProjectID:  data.TableUpdateData.DestinationProjectID,
		DestinationDatasetID:  data.TableUpdateData.DestinationDatasetID,
		DestinationTableName:  tableName,
		WriteDisposition:      bigquery.WriteAppend,
		AllPartitions:         data.TableUpdateData.AllPartitions,
		FromDate:              data.TableUpdateData.FromDate,
		FromDateNumPartitions: data.TableUpdateData.FromDateNumPartitions,
		WaitTillDone:          true,
		CSP:                   true,
		Clustering:            &bigquery.Clustering{Fields: []string{"billing_account_id"}},

		House:   common.HouseAdoption,
		Feature: common.FeatureCloudAnalytics,
		Module:  common.ModuleTableManagementCsp,
	}

	if err := s.updateBillingAccountTable(ctx, billingAccountID, &requestData); err != nil {
		domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: ERROR returned from updateBillingAccountTable temp table\n", id, billingAccountID), err)

		if gapiErr, ok := err.(*googleapi.Error); ok {
			if gapiErr.Code == http.StatusForbidden && !strings.Contains(gapiErr.Message, "project_and_region exceeded quota") {
				domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: ** getOrIncTableIndex **\n", data.TaskID, data.BillingAccountID), nil)

				_, newIdx, err := s.cspDAL.GetOrIncTableIndex(ctx, idx, data)
				if err != nil {
					domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Error obtaining new table index.\n", id, billingAccountID), err)
				} else {
					domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Will be writing to new table index: %d\n", id, billingAccountID, newIdx), nil)
				}
			}
		}

		return err
	}

	domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: SUCCEEDED ** RunBillingTableUpdateQuery ** \n", id, billingAccountID), nil)

	if data.Mode == domain.CSPUpdateAllMode {
		if alreadyCopied, err := s.cspDAL.SetDataCopied(ctx, data); alreadyCopied {
			return nil
		} else if err != nil {
			domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Error while updating copy state.\n", id, billingAccountID), err)
		}
	}

	return err
}

func (s *BillingTableManagementService) failedCopyTempTableIfFinished(
	ctx context.Context,
	data *domain.CSPBillingAccountUpdateData,
) error {
	l := s.loggerProvider(ctx)
	billingAccountID := data.BillingAccountID
	id := data.TaskID
	domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: ** failedCopyTempTableIfFinished **\n", id, billingAccountID), nil)

	domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Adding to failed.\n", id, billingAccountID), nil)

	stillRunning, err := s.cspDAL.SetTaskState(ctx, domain.TaskStateFailed, data)
	if err != nil {
		domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Error while adding to failed\n", id, billingAccountID), err)
		return err
	}

	domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Still running %d\n", id, billingAccountID, stillRunning), err)

	if stillRunning <= 0 {
		if err := s.createCopyTempTablesTask(ctx, data); err != nil {
			domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Failed wrapping up.\n", id, billingAccountID), err)
			return err
		}

		domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Finished.\n", id, billingAccountID), nil)
	}

	return nil
}

func (s *BillingTableManagementService) succeededCopyTempTableIfFinished(
	ctx context.Context,
	shouldDecStillRunning bool,
	data *domain.CSPBillingAccountUpdateData,
) error {
	l := s.loggerProvider(ctx)
	billingAccountID := data.BillingAccountID
	id := data.TaskID
	domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: ** succeededCopyTempTableIfFinished **\n", id, billingAccountID), nil)

	var err error

	var stillRunning int

	if shouldDecStillRunning {
		domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: ** decStillRunning **\n", data.TaskID, data.BillingAccountID), nil)

		stillRunning, err = s.cspDAL.DecStillRunning(ctx, data)
		if err != nil {
			domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Error decreasing counter\n", id, billingAccountID), err)
			return err
		}
	} else {
		var fsData domain.CSPFirestoreData
		if err := s.cspDAL.GetFirestoreData(ctx, data, &fsData); err != nil {
			return err
		}

		stillRunning = fsData.StillRunning
	}

	domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Still running %d\n", id, billingAccountID, stillRunning), err)

	if _, err := s.cspDAL.SetTaskState(ctx, domain.TaskStateProcessed, data); err != nil {
		domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Error while adding to processed\n", id, billingAccountID), err)
	}

	if stillRunning <= 0 {
		if err := s.createCopyTempTablesTask(ctx, data); err != nil {
			domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Failed wrapping up.\n", id, billingAccountID), err)
			return err
		}
	}

	domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Finished.\n", id, billingAccountID), nil)

	return nil
}

func (s *BillingTableManagementService) createCopyTempTablesTask(
	ctx context.Context,
	data *domain.CSPBillingAccountUpdateData,
) error {
	l := s.loggerProvider(ctx)
	billingAccountID := data.BillingAccountID
	id := data.TaskID
	domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: ** updateCSPTableAndDeleteTemp **\n", id, billingAccountID), nil)

	if data.Mode == domain.CSPUpdateAllMode {
		var fsData domain.CSPFirestoreData
		if err := s.cspDAL.GetFirestoreData(ctx, data, &fsData); err == nil {
			for account, accountData := range fsData.Tasks {
				if accountData.BillingDataCopied == false {
					domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Billing data of account %s was not copied!!\n", id, billingAccountID, account), err)
				}
			}
		}
	}

	baseURI := "/tasks/analytics/google-cloud/csp-accounts-finalize?"
	if data.Mode != domain.CSPUpdateAllMode {
		baseURI += "account=" + billingAccountID
	}

	baseURI = s.cspAddRangeQueryParams(baseURI, data.TableUpdateData)

	config := common.CloudTaskConfig{
		Method:           cloudtaskspb.HttpMethod_POST,
		Path:             baseURI,
		Queue:            common.TaskQueueCloudAnalyticsCSP,
		DispatchDeadline: durationpb.New(6 * time.Hour),
	}

	_, err := s.conn.CloudTaskClient.CreateAppEngineTask(ctx, config.AppEngineConfig(nil))
	if err != nil {
		domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Error creating final step task.\n", id, billingAccountID), err)
		return err
	}

	domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Final step task created.\n", id, billingAccountID), nil)

	return nil
}

func (s *BillingTableManagementService) joinAllCSPTempTables(
	ctx context.Context,
	data *domain.CSPBillingAccountUpdateData,
) []error {
	l := s.loggerProvider(ctx)
	billingAccountID := data.BillingAccountID
	domain.DebugPrintToLogs(l, fmt.Sprintf("%s: ** joinAllCSPTempTables **\n", billingAccountID), nil)

	var fsData domain.CSPFirestoreData
	if err := s.cspDAL.GetFirestoreData(ctx, data, &fsData); err != nil {
		return []error{err}
	}

	scheduleTime := time.Now().UTC()

	errors := []error{}

	for i := 1; i <= fsData.TableIndex; i++ {
		if _, ok := fsData.TempCopied[strconv.Itoa(i)]; !ok {
			if err := s.createCopyOneTempTableTask(ctx, i, scheduleTime, data); err != nil {
				errors = append(errors, err)
				domain.DebugPrintToLogs(l, fmt.Sprintf("%s: Join temp table task index: %d was not created.\n", billingAccountID, i), err)
			}

			scheduleTime = scheduleTime.Add(time.Minute * 8)
		}
	}

	return errors
}

func (s *BillingTableManagementService) createCopyOneTempTableTask(
	ctx context.Context,
	idx int,
	scheduleTime time.Time,
	data *domain.CSPBillingAccountUpdateData,
) error {
	l := s.loggerProvider(ctx)
	billingAccountID := data.BillingAccountID
	domain.DebugPrintToLogs(l, fmt.Sprintf("%s: ** createCopyOneTempTableTask ** index: %d\n", billingAccountID, idx), nil)

	baseURI := "/tasks/analytics/google-cloud/csp-accounts-join?idx=" + strconv.Itoa(idx)
	if data.Mode != domain.CSPUpdateAllMode {
		baseURI += "&account=" + billingAccountID
	}

	baseURI = s.cspAddRangeQueryParams(baseURI, data.TableUpdateData)

	config := &common.CloudTaskConfig{
		Method:       cloudtaskspb.HttpMethod_POST,
		Path:         baseURI,
		Queue:        common.TaskQueueCloudAnalyticsCSP,
		ScheduleTime: common.TimeToTimestamp(scheduleTime),
	}

	_, err := s.conn.CloudTaskClient.CreateAppEngineTask(ctx, config.AppEngineConfig(nil))
	if err != nil {
		domain.DebugPrintToLogs(l, fmt.Sprintf("%s: Error creating join temp table task index: %d.\n", billingAccountID, idx), err)
		return err
	}

	domain.DebugPrintToLogs(l, fmt.Sprintf("%s: Join temp table task created index: %d.\n", billingAccountID, idx), nil)

	return nil
}

func (s *BillingTableManagementService) joinCSPTempTable(
	ctx context.Context,
	bq *bigquery.Client,
	tableIdx int,
	data *domain.CSPBillingAccountUpdateData,
) error {
	l := s.loggerProvider(ctx)
	billingAccountID := data.BillingAccountID
	domain.DebugPrintToLogs(l, fmt.Sprintf("%s: ** joinCSPTempTable **\n", billingAccountID), nil)

	onePartition := !(data.TableUpdateData.AllPartitions ||
		(data.TableUpdateData.FromDate != "" && data.TableUpdateData.FromDateNumPartitions > 1))

	var suffix string

	if data.Mode == domain.CSPUpdateAllMode {
		var fsData domain.CSPFirestoreData
		if err := s.cspDAL.GetFirestoreData(ctx, data, &fsData); err != nil {
			domain.DebugPrintToLogs(l, fmt.Sprintf("%s: Error fetching temp table suffix\n", billingAccountID), err)
			return err
		}

		suffix = fsData.RunID
	} else {
		suffix = strings.Replace(billingAccountID, "-", "_", -1)
	}

	query := cspreport.GetCSPTableMetadataQuery(onePartition,
		&cspReportDomain.CSPMetadataQueryData{
			Cloud:                    common.Assets.GoogleCloud,
			BillingDataTableFullName: fmt.Sprintf("%s.%s.%s", domain.GetBillingProject(), domain.GetCustomerBillingDataset(consts.MasterBillingAccount), s.getCSPBillingTempTableName(suffix, tableIdx)),
			MetadataTableFullName:    fmt.Sprintf("%s.%s.%s", cspreport.GetCSPMetadataProject(), cspreport.GetCSPMetadataDataset(), cspreport.GetCSPMetadataTable()),
			BindIDField:              "billing_account_id",
			MetadataBindIDField:      "id",
		})

	domain.DebugPrintToLogs(l, fmt.Sprintf("%s: %s\n", billingAccountID, query), nil)

	writeDisposition := bigquery.WriteAppend
	if onePartition && data.Mode == domain.CSPUpdateAllMode {
		writeDisposition = bigquery.WriteTruncate
	}

	err := service.RunBillingTableUpdateQuery(ctx, bq, query,
		&tableMgmtDomain.BigQueryTableUpdateRequest{
			DefaultProjectID:      domain.GetBillingProject(),
			DefaultDatasetID:      domain.GetCustomerBillingDataset(consts.MasterBillingAccount),
			DestinationProjectID:  domain.GetBillingProject(),
			DestinationDatasetID:  domain.GetCustomerBillingDataset(consts.MasterBillingAccount),
			DestinationTableName:  domain.GetCSPFullBillingTableName(),
			WriteDisposition:      writeDisposition,
			AllPartitions:         data.TableUpdateData.AllPartitions,
			FromDate:              data.TableUpdateData.FromDate,
			FromDateNumPartitions: data.TableUpdateData.FromDateNumPartitions,
			WaitTillDone:          true,
			CSP:                   true,
			ConfigJobID:           "cloud_analytics_csp_join_temp_table_-" + strconv.Itoa(tableIdx),
			Clustering:            service.GetTableClustering(true),

			House:   common.HouseAdoption,
			Feature: common.FeatureCloudAnalytics,
			Module:  common.ModuleTableManagementCsp,
		})

	return err
}

func (s *BillingTableManagementService) createCSPAggregatedTableTask(
	ctx context.Context,
	data *domain.CSPBillingAccountUpdateData,
) error {
	l := s.loggerProvider(ctx)
	billingAccountID := data.BillingAccountID
	id := data.TaskID
	domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: ** createCSPAggregatedTableTask **\n", id, billingAccountID), nil)

	baseURI := "/tasks/analytics/google-cloud/csp-accounts-aggregate?"
	if data.Mode != domain.CSPUpdateAllMode {
		baseURI += "account=" + billingAccountID
	}

	if data.TableUpdateData.AllPartitions {
		baseURI += "&allPartitions=true"
	}

	// Aggregate from date is not supported at this time, we can manually run a query
	// to aggregate from a specific date until [CMP-14035] is done.
	if data.TableUpdateData.FromDate != "" {
		return nil
	}

	config := common.CloudTaskConfig{
		Method: cloudtaskspb.HttpMethod_POST,
		Path:   baseURI,
		Queue:  common.TaskQueueCloudAnalyticsCSP,
	}

	_, err := s.conn.CloudTaskClient.CreateAppEngineTask(ctx, config.AppEngineConfig(nil))
	if err != nil {
		domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Error creating aggregated table task.\n", id, billingAccountID), err)
		return err
	}

	domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Aggregated table task created.\n", id, billingAccountID), nil)

	return nil
}

func (s *BillingTableManagementService) deleteTempTables(
	ctx context.Context,
	bq *bigquery.Client,
	data *domain.CSPBillingAccountUpdateData,
) []error {
	// Don't delete temp tables automatically on manual allPartitions/fromDate updates
	if data.TableUpdateData.AllPartitions || data.TableUpdateData.FromDate != "" {
		return nil
	}

	l := s.loggerProvider(ctx)
	billingAccountID := data.BillingAccountID
	domain.DebugPrintToLogs(l, fmt.Sprintf("%s: ** DeleteTempTables **\n", billingAccountID), nil)
	domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: ** getOrIncTableIndex **\n", data.TaskID, data.BillingAccountID), nil)

	runID, idx, err := s.cspDAL.GetOrIncTableIndex(ctx, -1, data)
	if err != nil {
		domain.DebugPrintToLogs(l, fmt.Sprintf("%s: Error fetching table max index.\n", billingAccountID), err)
		return []error{err}
	}

	var fsData domain.CSPFirestoreData
	if err := s.cspDAL.GetFirestoreData(ctx, data, &fsData); err != nil {
		return []error{err}
	}

	var suffix string
	if data.Mode == domain.CSPUpdateAllMode {
		suffix = runID
	} else {
		suffix = strings.Replace(billingAccountID, "-", "_", -1)
	}

	errors := []error{}

	for i := 1; i <= idx; i++ {
		if _, ok := fsData.TempCopied[strconv.Itoa(idx)]; ok {
			tableName := s.getCSPBillingTempTableName(suffix, i)
			tableRef := bq.DatasetInProject(domain.GetBillingProject(), domain.GetCustomerBillingDataset(consts.MasterBillingAccount)).Table(tableName)

			if err := tableRef.Delete(ctx); err != nil {
				domain.DebugPrintToLogs(l, fmt.Sprintf("%s: Error deleting table index %d.\n", billingAccountID, i), err)
				errors = append(errors, err)
			} else {
				domain.DebugPrintToLogs(l, fmt.Sprintf("%s: Deleted table index %d.\n", billingAccountID, i), nil)
			}
		}
	}

	return errors
}

func (s *BillingTableManagementService) deleteCSPBillingAccount(
	ctx context.Context,
	bq *bigquery.Client,
	data *domain.CSPBillingAccountUpdateData,
) error {
	l := s.loggerProvider(ctx)
	domain.DebugPrintToLogs(l, fmt.Sprintf("%s: ** deleteCSPBillingAccount **\n", data.BillingAccountID), nil)

	if err := s.deleteCSPBillingAccountFromTable(ctx, bq, domain.GetFullCSPFullBillingTable(), data); err != nil {
		return err
	}

	if err := s.deleteCSPBillingAccountFromTable(ctx, bq, domain.GetFullCSPBillingTable(), data); err != nil {
		return err
	}

	return nil
}

func (s *BillingTableManagementService) deleteCSPBillingAccountFromTable(
	ctx context.Context,
	bq *bigquery.Client,
	fullTableName string,
	data *domain.CSPBillingAccountUpdateData,
) error {
	l := s.loggerProvider(ctx)
	billingAccountID := data.BillingAccountID
	domain.DebugPrintToLogs(l, fmt.Sprintf("%s: ** deleteCSPBillingAccountFromTable ** %s\n", billingAccountID, fullTableName), nil)

	queryData := tableMgmtDomain.BigQueryTableUpdateRequest{
		DefaultProjectID:     domain.GetBillingProject(),
		DefaultDatasetID:     domain.GetCustomerBillingDataset(consts.MasterBillingAccount),
		DestinationProjectID: domain.GetBillingProject(),
		DestinationDatasetID: "",
		DestinationTableName: "",
		WriteDisposition:     bigquery.WriteTruncate,
		AllPartitions:        true,
		WaitTillDone:         true,
		ConfigJobID:          "cloud_analytics_csp_gcp_delete_billing_account_" + billingAccountID,
		DML:                  true,
		CSP:                  true,

		House:   common.HouseAdoption,
		Feature: common.FeatureCloudAnalytics,
		Module:  common.ModuleTableManagementCsp,
	}

	partitionFilter := "DATE(export_time) >= '2018-01-01'"

	if data.TableUpdateData.FromDate != "" {
		switch data.TableUpdateData.FromDateNumPartitions {
		case 0:
			partitionFilter = fmt.Sprintf("DATE(export_time) >= '%s'", data.TableUpdateData.FromDate)
		case 1:
			partitionFilter = fmt.Sprintf("DATE(export_time) = '%s'", data.TableUpdateData.FromDate)
		default:
			// Example, fromDate = 2019-01-01, numPartitions = 3
			// partitionFilter = DATE(export_time) >= '2019-01-01' AND DATE(export_time) < DATE_ADD('2019-01-01', INTERVAL 3 DAY)
			// meaning export_time between 2019-01-01 and 2019-01-03 (inclusive)
			partitionFilter = fmt.Sprintf("DATE(export_time) >= '%s' AND DATE(export_time) < DATE_ADD('%s', INTERVAL %d DAY)",
				data.TableUpdateData.FromDate, data.TableUpdateData.FromDate, data.TableUpdateData.FromDateNumPartitions)
		}
	}

	query := fmt.Sprintf("DELETE FROM `%s` WHERE %s", fullTableName, partitionFilter)

	if billingAccountID != "" {
		query += fmt.Sprintf(` AND billing_account_id = "%s"`, billingAccountID)
	}

	if err := service.RunBillingTableUpdateQuery(ctx, bq, query, &queryData); err != nil {
		if e, ok := err.(*googleapi.Error); ok {
			if e.Code != http.StatusNotFound {
				domain.DebugPrintToLogs(l, fmt.Sprintf("%s: Error deleting account data from %s table.\n", billingAccountID, fullTableName), err)
				return err
			}
		}
	}

	return nil
}
