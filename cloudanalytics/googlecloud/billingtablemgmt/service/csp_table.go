package googlecloud

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"cloud.google.com/go/bigquery"

	cspTaskReporterDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/csptaskreporter/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	queryDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	tableMgmtDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/service"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/googlecloud"
)

func (s *BillingTableManagementService) UpdateCSPBillingAccounts(
	ctx context.Context,
	params domain.UpdateCspTaskParams,
	numPartitions int,
	allPartitions bool,
	fromDate string,
) error {
	l := s.loggerProvider(ctx)

	domain.DebugPrintToLogs(l, "%s: ** UpdateCSPBillingAccounts **\n", nil)

	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		l.Warningf("no bq client found for origin, using default")
	}

	// Check if the FULL table exists and if not this should be an "allPartitions" update
	tableExists, _, err := common.BigQueryTableExists(ctx, bq, domain.GetBillingProject(), domain.GetCSPBillingDataset(), domain.GetCSPFullBillingTableName())
	if err != nil {
		return err
	}

	if !tableExists {
		allPartitions = true
	}

	docSnaps, err := s.cspDAL.GetAssetsForTask(ctx, &params)
	if err != nil {
		return err
	}

	activeStandaloneAccounts, err := s.cspDAL.GetActiveStandaloneAccounts(ctx)
	if err != nil {
		return err
	}

	if len(docSnaps) == 0 {
		return nil
	}

	var accounts []string

	for _, docSnap := range docSnaps {
		var asset googlecloud.Asset
		if err := docSnap.DataTo(&asset); err != nil {
			return err
		}

		if asset.Properties.BillingAccountID == common.E2ETestBillingAccountID {
			continue
		}

		if asset.AssetType == common.Assets.GoogleCloudStandalone {
			if _, ok := activeStandaloneAccounts[asset.Properties.BillingAccountID]; !ok {
				continue
			}
		}

		accounts = append(accounts, asset.Properties.BillingAccountID)
	}

	if _, err := s.cspDAL.GetFirestoreCountersDocRef(ctx, domain.CSPUpdateAllMode, "").Set(ctx, domain.CSPFirestoreData{
		StillRunning: len(accounts),
		Processed:    0,
		Tasks:        make(map[string]domain.TaskStateData),
		TableIndex:   1,
		TempCopied:   make(map[string]string),
		RunID:        common.RandomSequenceN(10),
	}); err != nil {
		return err
	}

	var data = domain.CSPBillingAccountsTableUpdateData{
		Accounts:              accounts,
		AllPartitions:         allPartitions,
		FromDate:              fromDate,
		FromDateNumPartitions: numPartitions,
		DestinationProjectID:  domain.GetBillingProject(),
		DestinationDatasetID:  domain.GetCSPBillingDataset(),
	}

	if err := s.createGoogleCloudCSPBillingAccountTableUpdateTask(ctx, &data); err != nil {
		return err
	}

	return nil
}

func (s *BillingTableManagementService) AppendToTempCSPBillingAccountTable(
	ctx context.Context,
	billingAccountID string,
	updateAll bool,
	allPartitions bool,
	numPartitions int,
	fromDate string,
) error {
	l := s.loggerProvider(ctx)

	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		l.Warningf("no bq client found for origin, using default")
	}

	domain.DebugPrintToLogs(l, fmt.Sprintf("%s: ** AppendToTempCSPBillingAccountTable **\n", billingAccountID), nil)
	domain.DebugPrintToLogs(l, fmt.Sprintf("%s: allPartitions: %v", billingAccountID, allPartitions), nil)

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	taskID := r1.Intn(100000)

	var mode domain.CSPMode
	if updateAll {
		mode = domain.CSPUpdateAllMode
	} else if allPartitions {
		// updating all partitions of single account
		mode = domain.CSPReloadSingleMode
	} else {
		mode = domain.CSPUpdateSingleMode
	}

	singleAccountData := domain.CSPBillingAccountUpdateData{
		TaskID:           taskID,
		BillingAccountID: billingAccountID,
		Mode:             mode,
		TableUpdateData: &domain.CSPBillingAccountsTableUpdateData{
			DestinationProjectID:  domain.GetBillingProject(),
			DestinationDatasetID:  domain.GetCSPBillingDataset(),
			AllPartitions:         allPartitions,
			FromDate:              fromDate,
			FromDateNumPartitions: numPartitions,
		},
	}

	taskSummary := &cspTaskReporterDomain.TaskSummary{
		TaskID:    fmt.Sprintf("%v", taskID),
		AccountID: billingAccountID,
		Parameters: cspTaskReporterDomain.TaskParameters{
			AccountID:     billingAccountID,
			UpdateAll:     updateAll,
			AllPartitions: allPartitions,
			NumPartitions: numPartitions,
			FromDate:      fromDate,
		},
		TaskType: cspTaskReporterDomain.TaskTypeGCP,
		Status:   cspTaskReporterDomain.TaskStatusSuccess,
	}

	defer s.cspTaskReporter.LogTaskSummary(ctx, taskSummary)

	taskSummary.Stage = domain.StageAppendToTempCSPBillingAccountTableSetTaskState

	if mode == domain.CSPUpdateAllMode {
		if _, err := s.cspDAL.SetTaskState(ctx, domain.TaskStateRunning, &singleAccountData); err != nil {
			switch err.(type) {
			case *domain.DuplicateTaskError:
				domain.DebugPrintToLogs(l, "", err)

				taskSummary.Status = cspTaskReporterDomain.TaskStatusNonAlertingTermination
				taskSummary.Error = err

				return err
			default:
				taskSummary.Status = cspTaskReporterDomain.TaskStatusFailed
				taskSummary.Error = err
				domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: Error setting task state to running.\n", taskID, billingAccountID), err)

				return err
			}
		}
	} else {
		_, err := s.cspDAL.GetFirestoreCountersDocRef(ctx, mode, billingAccountID).
			Set(ctx, domain.CSPFirestoreData{
				StillRunning: 1,
				Processed:    0,
				Tasks:        make(map[string]domain.TaskStateData),
				TableIndex:   1,
				TempCopied:   make(map[string]string),
			})
		if err != nil {
			taskSummary.Status = cspTaskReporterDomain.TaskStatusFailed
			taskSummary.Error = err

			return err
		}

		taskSummary.Stage = domain.StageAppendToTempCSPBillingAccountTableDeleteCSPBillingAccount
		// If updating specific account, delete the existing data for that account (if any)
		// otherwise we will end up with duplicate records
		if err := s.deleteCSPBillingAccount(ctx, bq, &singleAccountData); err != nil {
			taskSummary.Status = cspTaskReporterDomain.TaskStatusFailed
			taskSummary.Error = err

			return err
		}
	}

	taskSummary.Stage = domain.StageAppendToTempCSPBillingAccountTableAppendToTempCSPBillingAccountTable

	err := s.appendToTempCSPBillingAccountTable(ctx, &singleAccountData)
	if err != nil {
		domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: ERROR returned ** appendToTempCSPBillingAccountTable ** \n", taskID, billingAccountID), err)

		taskSummary.Status = cspTaskReporterDomain.TaskStatusFailed
		taskSummary.Error = err

		return err
	}

	domain.DebugPrintToLogs(l, fmt.Sprintf("%d - %s: SUCCEEDED ** appendToTempCSPBillingAccountTable ** \n", taskID, billingAccountID), nil)

	return nil
}

func (s *BillingTableManagementService) UpdateCSPTableAndDeleteTemp(
	ctx context.Context,
	billingAccountID string,
	allPartitions bool,
	fromDate string,
	numPartitions int,
) error {
	l := s.loggerProvider(ctx)

	domain.DebugPrintToLogs(l, " ** UpdateCSPTableAndDeleteTemp **\n", nil)

	var mode domain.CSPMode = domain.CSPUpdateAllMode
	if billingAccountID != "" {
		mode = domain.CSPUpdateSingleMode
	}

	domain.DebugPrintToLogs(l, fmt.Sprintf("%s: allPartitions: %v", billingAccountID, allPartitions), nil)

	taskSummary := &cspTaskReporterDomain.TaskSummary{
		AccountID: billingAccountID,
		Parameters: cspTaskReporterDomain.TaskParameters{
			AccountID:     billingAccountID,
			AllPartitions: allPartitions,
			NumPartitions: numPartitions,
			FromDate:      fromDate,
		},
		TaskType: cspTaskReporterDomain.TaskTypeGCP,
		Status:   cspTaskReporterDomain.TaskStatusSuccess,
	}

	defer s.cspTaskReporter.LogTaskSummary(ctx, taskSummary)

	taskSummary.Stage = domain.StageUpdateCSPTableAndDeleteTempBigQueryTableExists

	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		l.Warningf("no bq client found for origin, using default")
	}

	tableExists, _, err := common.BigQueryTableExists(ctx, bq, domain.GetBillingProject(), domain.GetCSPBillingDataset(), domain.GetCSPFullBillingTableName())
	if err != nil {
		taskSummary.Status = cspTaskReporterDomain.TaskStatusFailed
		taskSummary.Error = err

		return err
	}

	data := domain.CSPBillingAccountUpdateData{
		BillingAccountID: billingAccountID,
		Mode:             mode,
		TableUpdateData: &domain.CSPBillingAccountsTableUpdateData{
			DestinationProjectID:  domain.GetBillingProject(),
			DestinationDatasetID:  domain.GetCSPBillingDataset(),
			AllPartitions:         allPartitions,
			FromDate:              fromDate,
			FromDateNumPartitions: numPartitions,
		},
	}

	if tableExists && mode != domain.CSPUpdateSingleMode {
		if fromDate != "" {
			taskSummary.Stage = domain.StageUpdateCSPTableAndDeleteTempDeleteCSPBillingAccountFromTable
			if err := s.deleteCSPBillingAccountFromTable(ctx, bq, domain.GetFullCSPFullBillingTable(), &data); err != nil {
				taskSummary.Status = cspTaskReporterDomain.TaskStatusFailed
				taskSummary.Error = err

				return err
			}
		} else if allPartitions {
			// If "allPartitions" operation then delete the "FULL" table before temp tables are appended to it.
			taskSummary.Stage = domain.StageUpdateCSPTableAndDeleteTempDstTableDelete
			dstTable := bq.DatasetInProject(domain.GetBillingProject(), domain.GetCSPBillingDataset()).Table(domain.GetCSPFullBillingTableName())

			if err := dstTable.Delete(ctx); err != nil {
				taskSummary.Status = cspTaskReporterDomain.TaskStatusFailed
				taskSummary.Error = err

				return err
			}
		}
	}

	taskSummary.Stage = domain.StageUpdateCSPTableAndDeleteTempJoinAllCSPTempTables
	errors := s.joinAllCSPTempTables(ctx, &data)

	if len(errors) > 0 {
		taskSummary.Status = cspTaskReporterDomain.TaskStatusFailed
		taskSummary.Error = fmt.Errorf("%v", errors)
		domain.DebugPrintToLogs(l, fmt.Sprintf("%s: Error creating join temp table tasks.\n", billingAccountID), errors[0])
	}

	return nil
}

func (s *BillingTableManagementService) JoinCSPTempTable(
	ctx context.Context,
	billingAccountID string,
	idx int,
	allPartitions bool,
	fromDate string,
	numPartitions int,
) error {
	l := s.loggerProvider(ctx)

	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		l.Warningf("no bq client found for origin, using default")
	}

	domain.DebugPrintToLogs(l, fmt.Sprintf("%s: ** joinCSPTempTable **\n", billingAccountID), nil)
	domain.DebugPrintToLogs(l, fmt.Sprintf("%s: Copying table index: %d\n", billingAccountID, idx), nil)
	domain.DebugPrintToLogs(l, fmt.Sprintf("%s: allPartitions: %v", billingAccountID, allPartitions), nil)

	var mode domain.CSPMode = domain.CSPUpdateAllMode
	if billingAccountID != "" {
		mode = domain.CSPUpdateSingleMode
	}

	data := domain.CSPBillingAccountUpdateData{
		BillingAccountID: billingAccountID,
		Mode:             mode,
		TableUpdateData: &domain.CSPBillingAccountsTableUpdateData{
			DestinationProjectID:  domain.GetBillingProject(),
			DestinationDatasetID:  domain.GetCSPBillingDataset(),
			AllPartitions:         allPartitions,
			FromDate:              fromDate,
			FromDateNumPartitions: numPartitions,
		},
	}

	taskSummary := &cspTaskReporterDomain.TaskSummary{
		AccountID: billingAccountID,
		Parameters: cspTaskReporterDomain.TaskParameters{
			AccountID:     billingAccountID,
			AllPartitions: allPartitions,
			NumPartitions: numPartitions,
			FromDate:      fromDate,
		},
		TaskType: cspTaskReporterDomain.TaskTypeGCP,
		Status:   cspTaskReporterDomain.TaskStatusSuccess,
	}

	defer s.cspTaskReporter.LogTaskSummary(ctx, taskSummary)

	taskSummary.Stage = domain.StageJoinCSPTempTableAddRemoveToCopiedTables

	_, err := s.cspDAL.AddRemoveToCopiedTables(ctx, true, idx, false, &data)
	if err != nil {
		switch err.(type) {
		case *domain.DuplicateTaskError:
			taskSummary.Status = cspTaskReporterDomain.TaskStatusNonAlertingTermination
			taskSummary.Error = err
			domain.DebugPrintToLogs(l, "Duplicate task. Aborting", err)

			return err
		default:
			taskSummary.Status = cspTaskReporterDomain.TaskStatusFailed
			taskSummary.Error = err
			domain.DebugPrintToLogs(l, fmt.Sprintf("%s: Error adding to copied temp tables index %d\n", billingAccountID, idx), err)

			return err
		}
	}

	taskSummary.Stage = domain.StageJoinCSPTempTableJoinCSPTempTable

	if err := s.joinCSPTempTable(ctx, bq, idx, &data); err != nil {
		domain.DebugPrintToLogs(l, fmt.Sprintf("%s: Error copying temp table index %d\n", billingAccountID, idx), err)

		if _, err := s.cspDAL.AddRemoveToCopiedTables(ctx, false, idx, true, &data); err != nil {
			domain.DebugPrintToLogs(l, fmt.Sprintf("%s: Error adding to copied temp tables index %d\n", billingAccountID, idx), err)
		}

		taskSummary.Status = cspTaskReporterDomain.TaskStatusFailed
		taskSummary.Error = err

		return err
	}

	taskSummary.Stage = domain.StageJoinCSPTempTableAddRemoveToCopiedTables

	stillToCopy, err := s.cspDAL.AddRemoveToCopiedTables(ctx, true, idx, true, &data)
	if err != nil {
		domain.DebugPrintToLogs(l, fmt.Sprintf("%s: Error adding to done copied temp tables index %d\n", billingAccountID, idx), err)

		taskSummary.Status = cspTaskReporterDomain.TaskStatusFailed
		taskSummary.Error = err

		return err
	}

	domain.DebugPrintToLogs(l, fmt.Sprintf("Still to copy %d tables.\n", stillToCopy), nil)

	if stillToCopy == 0 {
		if err := s.reportStatusService.UpdateReportStatus(ctx, queryDomain.CSPCustomerID, common.ReportStatus{
			Status: map[string]common.StatusInfo{
				string(common.GoogleCloudReportStatus): {
					LastUpdate: time.Now(),
				},
			},
		}); err != nil {
			l.Error(err)
		}

		domain.DebugPrintToLogs(l, "Full CSP table is ready.\n", nil)

		taskSummary.Stage = domain.StageJoinCSPTempTableCreateCSPAggregatedTableTask
		if err := s.createCSPAggregatedTableTask(ctx, &data); err != nil {
			taskSummary.Status = cspTaskReporterDomain.TaskStatusFailed
			taskSummary.Error = err

			return err
		}

		if errors := s.deleteTempTables(ctx, bq, &data); len(errors) > 0 {
			domain.DebugPrintToLogs(l, "Error deleting temp tables.\n", errors[0])
		}
	}

	return nil
}

func (s *BillingTableManagementService) UpdateCSPAggregatedTable(
	ctx context.Context,
	billingAccountID string,
	allPartitions bool,
) error {
	l := s.loggerProvider(ctx)

	l.Debug(" ** UpdateCSPAggregatedTable **\n")

	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		l.Warningf("no bq client found for origin, using default")
	}

	query := domain.GetAggregatedQuery(allPartitions, billingAccountID, true, queryDomain.BillingTableSuffixDay)

	writeDisposition := bigquery.WriteTruncate
	if billingAccountID != "" {
		writeDisposition = bigquery.WriteAppend
	}

	taskSummary := &cspTaskReporterDomain.TaskSummary{
		AccountID: billingAccountID,
		Stage:     domain.StageUpdateCSPAggregatedTable,
		Parameters: cspTaskReporterDomain.TaskParameters{
			AccountID:     billingAccountID,
			AllPartitions: allPartitions,
		},
		TaskType: cspTaskReporterDomain.TaskTypeGCP,
		Status:   cspTaskReporterDomain.TaskStatusSuccess,
	}

	defer s.cspTaskReporter.LogTaskSummary(ctx, taskSummary)

	err := service.RunBillingTableUpdateQuery(ctx, bq, query,
		&tableMgmtDomain.BigQueryTableUpdateRequest{
			DefaultProjectID:     domain.GetBillingProject(),
			DefaultDatasetID:     domain.GetCSPBillingDataset(),
			DestinationProjectID: domain.GetBillingProject(),
			DestinationDatasetID: domain.GetCSPBillingDataset(),
			DestinationTableName: domain.GetCSPBillingTableName(),
			AllPartitions:        allPartitions,
			WriteDisposition:     writeDisposition,
			ConfigJobID:          "cloud_analytics_csp_gcp_aggregated",
			WaitTillDone:         false,
			CSP:                  true,
			Clustering:           service.GetTableClustering(true),

			House:   common.HouseAdoption,
			Feature: common.FeatureCloudAnalytics,
			Module:  common.ModuleTableManagementCsp,
		})

	if err != nil {
		taskSummary.Status = cspTaskReporterDomain.TaskStatusFailed
		taskSummary.Error = err
	}

	return err
}
