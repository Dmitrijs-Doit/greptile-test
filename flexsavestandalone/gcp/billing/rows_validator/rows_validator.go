package rows_validator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	doitFirestore "github.com/doitintl/firestore"
	googleCloudConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	"github.com/doitintl/hello/scheduled-tasks/common"
	standaloneCommon "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
)

// How much behind the customer's latest record are we allowed to be.
const allowedDelay = 9 * time.Hour

func (s *RowsValidator) ValidateRows(ctx *gin.Context) error {
	logger := s.loggerProvider(ctx)

	etms, err := s.metadata.GetActiveInternalTasksMetadata(ctx)
	if err != nil {
		logger.Errorf("%cerror fetching internal metadata: %s", logPrefix, err)
		return err
	}

	for i, etm := range etms {
		go func(billingAccount string) {
			config := common.CloudTaskConfig{
				Method: cloudtaskspb.HttpMethod_GET,
				Path:   fmt.Sprintf("/tasks/flexsave-standalone/google-cloud/billing/rows_validate/%s", billingAccount),
				Queue:  common.TaskQueueFlexSaveStandaloneRowsValidatorTasks,
			}

			task, err := s.CloudTaskClient.CreateTask(ctx, config.Config(nil))
			if err != nil {
				logger.Errorf("%sunable to schedule rows validator task: %s", logPrefix, err)
			}

			logger.Infof("%srows validator task %s for BA %s scheduled", task.String(), logPrefix, billingAccount)
		}(etm.BillingAccount)

		if i%10 == 0 {
			logger.Infof("%swaiting to finish rows validator balk", logPrefix)
			time.Sleep(time.Minute)
		}
	}

	return nil
}

func errorf(funcName string, billingAccount string, originalError error) error {
	msg := fmt.Sprintf("unable to %s", funcName)
	if billingAccount != "" {
		msg = fmt.Sprintf(" %s for BA %s.", msg, billingAccount)
	}

	if originalError != nil {
		msg = fmt.Sprintf("%s Caused by %s", msg, originalError)
	}

	return fmt.Errorf(msg)
}

func (s *RowsValidator) RunMonitor(ctx *gin.Context) error {
	logger := s.loggerProvider(ctx)

	etms, err := s.metadata.GetActiveInternalTasksMetadata(ctx)
	if err != nil {
		err = errorf("GetActiveInternalTasksMetadata", "", err)
		logger.Error(err)

		return err
	}

	for i, etm := range etms {
		go func(billingAccount string) {
			config := common.CloudTaskConfig{
				Method: cloudtaskspb.HttpMethod_GET,
				Path:   fmt.Sprintf("/tasks/flexsave-standalone/google-cloud/billing/monitor/%s", billingAccount),
				Queue:  common.TaskQueueFlexSaveStandaloneMonitorTasks,
			}

			task, err := s.CloudTaskClient.CreateTask(ctx, config.Config(nil))
			if err != nil {
				logger.Error(errorf("CreateTask", etm.BillingAccount, err))
			}

			logger.Infof("monitor task %s for BA %s scheduled", task.String(), billingAccount)
		}(etm.BillingAccount)

		if i%10 == 0 {
			logger.Infof("waiting to finish rows validator balk")
			time.Sleep(time.Minute)
		}
	}

	return nil
}

func (v *RowsValidator) RunMonitorForBilling(ctx context.Context, billingAccount string) error {
	logger := v.loggerProvider(ctx)

	itm, err := v.metadata.GetInternalTaskMetadata(ctx, billingAccount)
	if err != nil {
		err = errorf("GetInternalTaskMetadata", billingAccount, err)
		v.sendMonitorIssueNotification(ctx, itm.BillingAccount, nil, AllTests, err)
		logger.Error(err)

		return err
	}

	err = v.isAllowedToRun(ctx, itm)
	if err != nil {
		err = errorf("RunMonitorForBilling", billingAccount, err)
		v.sendMonitorIssueNotification(ctx, itm.BillingAccount, v.getSegment(ctx, itm), AllTests, err)
		logger.Error(err)

		return err
	}

	err = v.VerifyRows(ctx, itm)
	if err != nil {
		err = errorf("VerifyRows", billingAccount, err)
		v.sendMonitorIssueNotification(ctx, itm.BillingAccount, v.getSegment(ctx, itm), VerifyRows, err)
		logger.Error(err)
	}

	err = v.dataWasUpdatedLately2(ctx, itm)
	if err != nil {
		err = errorf("dataWasUpdatedLately2", billingAccount, err)
		v.sendMonitorIssueNotification(ctx, itm.BillingAccount, v.getSegment(ctx, itm), DataFlowing, err)
		logger.Error(err)
	}

	err = v.verifyThereIsDataFromPastWeek(ctx, itm)
	if err != nil {
		err = errorf("verifyThereIsDataFromPastWeek", billingAccount, err)
		v.sendMonitorIssueNotification(ctx, itm.BillingAccount, v.getSegment(ctx, itm), DataExists, err)
		logger.Error(err)
	}

	return nil
}

func (r *RowsValidator) verifyThereIsDataFromPastWeek(ctx context.Context, itm *dataStructures.InternalTaskMetadata) error {
	logger := r.loggerProvider(ctx)
	endTime := time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour)
	startTime := endTime.AddDate(0, 0, -7)

	count, err := r.tableQuery.GetLUnifiedRowsCount(ctx, itm.BillingAccount, &dataStructures.Segment{
		StartTime: &startTime,
		EndTime:   &endTime,
	})
	if err != nil {
		err = errorf("verifyThereIsDataFromPastWeek", itm.BillingAccount, err)
		logger.Error(err)

		return err
	}

	currStart := startTime
	currEnd := startTime.AddDate(0, 0, 1)
	currKey := dataStructures.HashableSegment{
		StartTime: currStart,
		EndTime:   currEnd,
	}

	for currKey.StartTime.Before(endTime) {
		if _, ok := count[currKey]; !ok {
			err = fmt.Errorf("invalid row count for BA %s on date %v", itm.BillingAccount, currKey.StartTime)
			logger.Error(err)
			r.sendPastWeekNotification(ctx, itm.BillingAccount, currKey.StartTime.Format(consts.PartitionFieldFormat))
		}

		currKey.StartTime = currKey.EndTime
		currKey.EndTime = currKey.EndTime.AddDate(0, 0, 1)
	}

	return nil
}

func (v *RowsValidator) isAllowedToRun(ctx context.Context, itm *dataStructures.InternalTaskMetadata) error {
	if itm.OnBoarding {
		return fmt.Errorf("unable to monitor BA %s. Billing is still onboarding", itm.BillingAccount)
	}

	if itm.Segment == nil || itm.Segment.StartTime == nil {
		return fmt.Errorf("unable to monitor BA %s. Invalid segement found %+v", itm.BillingAccount, itm.Segment)
	}

	if itm.LifeCycleStage == dataStructures.LifeCycleStageDeprecated {
		return fmt.Errorf("unable to monitor BA %s. BA is in deprecated state and about to be removed", itm.BillingAccount)
	}

	return nil
}

func (v *RowsValidator) VerifyRows(ctx context.Context, itm *dataStructures.InternalTaskMetadata) error {
	logger := v.loggerProvider(ctx)
	logger.Infof("verifing rows for BA %s", itm.BillingAccount)

	invalidCounts, err := v.compareRowsByExportTime(ctx, itm)
	if err != nil {
		err = errorf("compareRowsByExportTime", itm.BillingAccount, err)
		logger.Error(err)

		return err
	}

	if len(invalidCounts) == 0 {
		//v.sendValidUpdateNotification2(ctx, itm.BillingAccount, v.getSegment(ctx, itm))
	} else {
		v.sendRowsMismatchNotification2(ctx, itm.BillingAccount, invalidCounts)
	}

	return nil
}

type InvalidCounts struct {
	timestamp string
	expected  int64
	found     int64
}

func (s *RowsValidator) compareRowsByExportTime(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (invalidCounts []*InvalidCounts, err error) {
	logger := s.loggerProvider(ctx)

	tablesCountByExportTime, err := s.createTableCountsByExportTime(ctx, itm, s.getSegment(ctx, itm))
	if err != nil {
		err = errorf("createTableCountsByExportTime", itm.BillingAccount, err)
		logger.Error(err)

		return nil, err
	}

	for exportTime, count := range tablesCountByExportTime {
		if itm.BillingAccount != googleCloudConsts.MasterBillingAccount {
			if count.customer != count.unified {
				invalidCounts = append(invalidCounts, &InvalidCounts{
					timestamp: exportTime,
					found:     count.unified,
					expected:  count.customer,
				})
			}
		} else {
			if count.local != count.unified {
				invalidCounts = append(invalidCounts, &InvalidCounts{
					timestamp: exportTime,
					found:     count.unified,
					expected:  count.local,
				})
			}
		}
	}

	return invalidCounts, nil
}

type tableCount struct {
	local    int64
	unified  int64
	customer int64
}

func (s *RowsValidator) createTableCountsByExportTime(ctx context.Context, itm *dataStructures.InternalTaskMetadata, segment *dataStructures.Segment) (tablesCountByExportTime map[string]*tableCount, err error) {
	//GetCustomerRowsCountByExportTime
	logger := s.loggerProvider(ctx)
	tablesCountByExportTime = make(map[string]*tableCount)

	if itm.BillingAccount != googleCloudConsts.MasterBillingAccount {
		customerRowsByExportTime, err := s.tableQuery.GetCustomerRowsCountByExportTime(ctx, itm.BillingAccount, segment)
		if err != nil {
			err = errorf("GetCustomerRowsCountByExportTime", itm.BillingAccount, err)
			logger.Error(err)

			return nil, err
		}

		for k, v := range customerRowsByExportTime {
			if _, ok := tablesCountByExportTime[k]; !ok {
				tablesCountByExportTime[k] = &tableCount{}
			}

			tablesCountByExportTime[k].customer = v
		}
	}

	localRowsByExportTime, err := s.tableQuery.GetLocalRowsCountByExportTime(ctx, itm.BillingAccount, segment)
	if err != nil {
		err = errorf("GetLocalRowsCountByExportTime", itm.BillingAccount, err)
		logger.Error(err)

		return nil, err
	}

	for k, v := range localRowsByExportTime {
		if _, ok := tablesCountByExportTime[k]; !ok {
			tablesCountByExportTime[k] = &tableCount{}
		}

		tablesCountByExportTime[k].local = v
	}

	unifiedRowsByExportTime, err := s.tableQuery.GetLUnifiedRowsCountByExportTime(ctx, itm.BillingAccount, segment)
	if err != nil {
		err = errorf("GetLUnifiedRowsCountByExportTime", itm.BillingAccount, err)
		logger.Error(err)

		return nil, err
	}

	for k, v := range unifiedRowsByExportTime {
		if _, ok := tablesCountByExportTime[k]; !ok {
			tablesCountByExportTime[k] = &tableCount{}
		}

		tablesCountByExportTime[k].unified = v
	}

	return tablesCountByExportTime, nil
}

func (v *RowsValidator) getSegment(ctx context.Context, itm *dataStructures.InternalTaskMetadata) *dataStructures.Segment {
	end := *itm.Segment.StartTime
	if itm.State == dataStructures.InternalTaskStateDone || itm.State == dataStructures.InternalTaskStateSkipped {
		end = *itm.Segment.EndTime
	}

	start := end.AddDate(0, -3, 0)

	return &dataStructures.Segment{
		StartTime: &start,
		EndTime:   &end,
	}
}

// TODO lionel remove this
func (s *RowsValidator) ValidateCustomerRows(ctx *gin.Context, billingAccountID string) error {
	logger := s.loggerProvider(ctx)

	var lastUpdatedDate *time.Time

	var err error

	var bq *bigquery.Client
	if billingAccountID != googleCloudConsts.MasterBillingAccount {
		bq, err = s.customerBQClient.GetCustomerBQClient(ctx, billingAccountID)
		if err != nil {
			logger.Errorf("%sError getting customer BQ client for %s: %v", logPrefix, billingAccountID, err)
			return err
		}
		defer bq.Close()
	}

	itm, err := s.metadata.GetInternalTaskMetadata(ctx, billingAccountID)
	if err != nil {
		logger.Errorf("%sError getting internal task metadata for %s: %v", logPrefix, billingAccountID, err)
		return err
	}

	rvm, err := s.getRowsValidatorMetadata(ctx, bq, billingAccountID, itm)
	if err != nil {
		logger.Errorf("%sUnable to get rows validator last metadata for %s: %v", logPrefix, billingAccountID, err)
		return err
	}

	if itm.OnBoarding || itm.Segment == nil || itm.Segment.StartTime == nil {
		logger.Infof("skipping BA %s since segment it not been set or it's still onboarding", itm.BillingAccount)
		return nil
	}

	if itm.State == dataStructures.InternalTaskStateDone || itm.State == dataStructures.InternalTaskStateSkipped {
		lastUpdatedDate = itm.Segment.EndTime
	} else {
		lastUpdatedDate = itm.Segment.StartTime
	}

	segments := s.getAllSegments(ctx, rvm, lastUpdatedDate, itm)

	if len(segments) > 0 {
		invalidSegments, _, queryFailedOn := s.getInvalidSegments(ctx, billingAccountID, segments)
		aggregated := s.aggregate(s.deduplicate(invalidSegments))
		invalidSegmentsData := &invalidSegmentData{
			billingAccountID: billingAccountID,
			segments:         aggregated,
		}

		queryFailed := didAnyQueryFailed(queryFailedOn)

		if len(invalidSegmentsData.segments) > 0 {
			s.sendRowsMismatchNotification(ctx, invalidSegmentsData)
		} else if !queryFailed {
			s.sendValidUpdateNotification(ctx, billingAccountID, segments)
		}

		if queryFailed {
			s.sendQueryFailedNotification(ctx, billingAccountID, queryFailedOn)
		}
	}

	if updated, err := s.dataWasUpdatedLately(ctx, bq, billingAccountID, itm); err != nil {
		logger.Errorf("%sError checking whether data is updated in time %s: %v", logPrefix, billingAccountID, err)
	} else if !updated {
		s.sendDataNotUpdatedNotification(ctx, billingAccountID, time.Hour*3)
	}

	return nil
}

// TODO lionel remove this
func didAnyQueryFailed(failedOn *tableQueryErrors) bool {
	for _, failed := range *failedOn {
		if len(failed) > 0 {
			return true
		}
	}

	return false
}

func (s *RowsValidator) dataWasUpdatedLately(ctx context.Context, bq *bigquery.Client, billingAccountID string, itm *dataStructures.InternalTaskMetadata) (bool, error) {
	var err error

	var lastOriginUpdated time.Time

	logger := s.loggerProvider(ctx)

	lastLocalUpdated, err := s.tableQuery.GetUnifiedTableNewestRecordByBA(ctx, itm)
	if err != nil {
		return true, err
	}

	logger.Infof("latest row found on unified of BA %s is on %s", billingAccountID, lastLocalUpdated.UTC())

	if billingAccountID != googleCloudConsts.MasterBillingAccount {
		var etm *dataStructures.ExternalTaskMetadata

		etm, err = s.metadata.GetExternalTaskMetadata(ctx, billingAccountID)
		if err != nil {
			return true, err
		}

		lastOriginUpdated, err = s.tableQuery.GetCustomersTableOldestRecordTimeNewerThan(ctx, bq, etm.BQTable, lastLocalUpdated)

		if err != nil {
			return true, err
		}
	} else {
		lastOriginUpdated, err = s.tableQuery.GetRawBillingNewestRecordTime(ctx)
		if err != nil {
			return true, err
		}
	}

	logger.Infof("latest row found on customer's side of BA %s is on %s", billingAccountID, lastLocalUpdated.UTC())

	if lastOriginUpdated.UTC().Sub(time.Now().UTC()) > allowedDelay {
		return false, nil
	}

	return true, nil
}

func (s *RowsValidator) dataWasUpdatedLately2(ctx context.Context, itm *dataStructures.InternalTaskMetadata) error {
	var err error

	var lastOriginUpdated time.Time

	logger := s.loggerProvider(ctx)

	lastLocalUpdated, err := s.tableQuery.GetUnifiedTableNewestRecordByBA(ctx, itm)
	if err != nil {
		return err
	}

	logger.Infof("latest row found on unified of BA %s is on %s", itm.BillingAccount, lastLocalUpdated.UTC())

	if itm.BillingAccount != googleCloudConsts.MasterBillingAccount {
		var etm *dataStructures.ExternalTaskMetadata

		etm, err = s.metadata.GetExternalTaskMetadata(ctx, itm.BillingAccount)
		if err != nil {
			return err
		}

		bq, err := s.customerBQClient.GetCustomerBQClient(ctx, itm.BillingAccount)
		if err != nil {
			logger.Errorf("%sError getting customer BQ client for %s: %v", logPrefix, itm.BillingAccount, err)
			return err
		}

		lastOriginUpdated, err = s.tableQuery.GetCustomersTableOldestRecordTimeNewerThan(ctx, bq, etm.BQTable, lastLocalUpdated)

		if err != nil {
			switch err.(type) {
			case *standaloneCommon.EmptyBillingTableError:
				logger.Infof("there are no newer lines on the customer's table then %v", lastLocalUpdated)
				return nil
			default:
				return err
			}
		}
	} else {
		lastOriginUpdated, err = s.tableQuery.GetRawBillingNewestRecordTime(ctx)
		if err != nil {
			return err
		}
	}

	logger.Infof("latest row found on customer's side of BA %s is on %s", itm.BillingAccount, lastLocalUpdated.UTC())
	//if lastOriginUpdated.UTC().Sub(time.Now().UTC()) > allowedDelay {
	if time.Now().UTC().Sub(lastOriginUpdated.UTC()) > allowedDelay {
		logger.Errorf("data was not updated for BA %s for %s", itm.BillingAccount, time.Now().UTC().Sub(lastOriginUpdated.UTC()))
		s.sendDataNotUpdatedNotification(ctx, itm.BillingAccount, time.Now().UTC().Sub(lastOriginUpdated.UTC()))
	}

	return nil
}

func (s *RowsValidator) getInvalidSegments(ctx context.Context, billingAccountID string, segments []*dataStructures.Segment) (sortableInvalidSegments, []*dataStructures.Segment, *tableQueryErrors) {
	logger := s.loggerProvider(ctx)

	invalidSegmentsData := &invalidSegmentData{
		billingAccountID: billingAccountID,
		segments:         make(sortableInvalidSegments, 0),
	}

	queryFailedOn := make(tableQueryErrors)

	var wg sync.WaitGroup

	limiter := rate.NewLimiter(rate.Every(1*time.Minute/50), 1)

	errChan := make(chan ([]tableRowsCountErrors), len(segments))

	for _, segment := range segments {
		wg.Add(1)

		go func(segment *dataStructures.Segment, errChan chan []tableRowsCountErrors) {
			defer wg.Done()

			logger.Infof("%sChecking segment %s - %s", logPrefix, segment.StartTime.Format(consts.ExportTimeLayoutWithMillis), segment.EndTime.Format(consts.ExportTimeLayoutWithMillis))

			rowsCountMap, rowCountErrors := s.getCounts(ctx, billingAccountID, segment)
			if len(rowCountErrors) > 0 {
				errChan <- rowCountErrors
				return
			}
			// This segment is empty and therefore has no subsegments, so we add it explicitly.
			if len(s.longestMap(rowsCountMap)) == 0 {
				segmentStart := *segment.StartTime
				segmentEnd := *segment.EndTime

				invalidSegmentsData.emptySegments = append(invalidSegmentsData.emptySegments, &dataStructures.Segment{
					StartTime: &segmentStart,
					EndTime:   &segmentEnd,
				})
			}

			for subSegment := range s.longestMap(rowsCountMap) {
				mismatch := false

				if billingAccountID != googleCloudConsts.MasterBillingAccount {
					if rowsCountMap[customerTableType][subSegment] != rowsCountMap[localTableType][subSegment] {
						mismatch = true

						logger.Errorf("%sRows count mismatch. BA %s, segment %s - %s, customer count: %d, local count: %d", logPrefix, billingAccountID,
							subSegment.StartTime.Format(consts.ExportTimeLayoutWithMillis),
							subSegment.EndTime.Format(consts.ExportTimeLayoutWithMillis),
							rowsCountMap[customerTableType][subSegment],
							rowsCountMap[localTableType][subSegment])
					}

					if rowsCountMap[customerTableType][subSegment] != rowsCountMap[unifiedTableType][subSegment] {
						mismatch = true

						logger.Errorf("%sRows count mismatch. BA %s, segment %s - %s, customer count: %d, unified count: %d", logPrefix, billingAccountID,
							subSegment.StartTime.Format(consts.ExportTimeLayoutWithMillis),
							subSegment.EndTime.Format(consts.ExportTimeLayoutWithMillis),
							rowsCountMap[customerTableType][subSegment],
							rowsCountMap[unifiedTableType][subSegment])
					}
				} else {
					if rowsCountMap[localTableType][subSegment] != rowsCountMap[unifiedTableType][subSegment] {
						mismatch = true

						logger.Errorf("%sRows count mismatch. Raw Billing %s, segment %s - %s, local count: %d, unified count: %d", logPrefix, billingAccountID,
							subSegment.StartTime.Format(consts.ExportTimeLayoutWithMillis),
							subSegment.EndTime.Format(consts.ExportTimeLayoutWithMillis),
							rowsCountMap[localTableType][subSegment],
							rowsCountMap[unifiedTableType][subSegment])
					}
				}

				if !mismatch {
					logger.Infof("%sValid segment %s - %s for %s", logPrefix, subSegment.StartTime.Format(consts.ExportTimeLayoutWithMillis), subSegment.EndTime.Format(consts.ExportTimeLayoutWithMillis), billingAccountID)

					if rowsCountMap[localTableType][subSegment] == 0 {
						segmentStart := subSegment.StartTime
						segmentEnd := subSegment.EndTime

						invalidSegmentsData.emptySegments = append(invalidSegmentsData.emptySegments, &dataStructures.Segment{
							StartTime: &segmentStart,
							EndTime:   &segmentEnd,
						})
					}
				} else {
					rowsCount := make(map[tableType]int)
					for t := range rowsCountMap {
						rowsCount[t] = rowsCountMap[t][subSegment]
					}

					segmentStart := subSegment.StartTime
					segmentEnd := subSegment.EndTime

					invalidSegmentsData.segments = append(invalidSegmentsData.segments, &invalidSegments{
						segment: &dataStructures.Segment{
							StartTime: &segmentStart,
							EndTime:   &segmentEnd,
						},
						rowsCount: rowsCount,
					})
				}
			}
		}(segment, errChan)

		err := limiter.Wait(ctx)
		if err != nil {
			logger.Error(err)
		}
	}

	wg.Wait()
	close(errChan)

	for rowsCountErrors := range errChan {
		for _, e := range rowsCountErrors {
			queryFailedOn[e.tt] = append(queryFailedOn[e.tt], &segmentError{
				segment: e.segment,
				err:     e.err,
			})
		}
	}

	leastInvalidSegmentsData := &invalidSegmentData{
		billingAccountID: billingAccountID,
		segments:         make(sortableInvalidSegments, 0),
	}

	for _, segment := range invalidSegmentsData.segments {
		smallerSegments := s.getSmallerSegments(segment.segment)
		if len(smallerSegments) > 0 {
			smallerInvalidSegments, _, smallerQueryFailedOn := s.getInvalidSegments(ctx, billingAccountID, smallerSegments)
			if len(smallerInvalidSegments) > 0 {
				leastInvalidSegmentsData.segments = append(leastInvalidSegmentsData.segments, smallerInvalidSegments...)
			} else {
				leastInvalidSegmentsData.segments = append(leastInvalidSegmentsData.segments, segment)
			}

			for t, failed := range *smallerQueryFailedOn {
				queryFailedOn[t] = append(queryFailedOn[t], failed...)
			}
		} else {
			leastInvalidSegmentsData.segments = append(leastInvalidSegmentsData.segments, segment)
		}
	}

	return leastInvalidSegmentsData.segments, invalidSegmentsData.emptySegments, &queryFailedOn
}

// TODO lionel remove this
func (s *RowsValidator) getRowsValidatorMetadata(ctx *gin.Context, bq *bigquery.Client, billingAccountID string, itm *dataStructures.InternalTaskMetadata) (*dataStructures.RowsValidatorMetadata, error) {
	logger := s.loggerProvider(ctx)

	var err error

	md, err := s.metadata.GetRowsValidatorMetadata(ctx, billingAccountID)
	if err != nil && err != doitFirestore.ErrNotFound {
		return nil, err
	}

	if md == nil || md.LastValidated == nil || err == doitFirestore.ErrNotFound {
		var startingTime time.Time

		if billingAccountID == googleCloudConsts.MasterBillingAccount {
			startingTime, err = s.tableQuery.GetLocalTableOldestRecordTime(ctx, itm)
			if err != nil {
				return nil, err
			}
		} else {
			bq, err := s.customerBQClient.GetCustomerBQClient(ctx, billingAccountID)
			if err != nil {
				logger.Errorf("%sError getting customer BQ client for %s: %v", logPrefix, billingAccountID, err)
				return nil, err
			}
			defer bq.Close()

			etm, err := s.metadata.GetExternalTaskMetadata(ctx, billingAccountID)
			if err != nil {
				return nil, err
			}

			startingTime, err = s.tableQuery.GetCustomersTableOldestRecordTime(ctx, bq, etm.BQTable)
			if err != nil {
				return nil, err
			}
		}

		startingTime = startingTime.Add(-1 * time.Second)
		md = &dataStructures.RowsValidatorMetadata{
			LastValidated: &startingTime,
		}
	}

	return md, nil
}

func (s *RowsValidator) getAllSegments(ctx context.Context, md *dataStructures.RowsValidatorMetadata, lastUpdatedDate *time.Time, itm *dataStructures.InternalTaskMetadata) []*dataStructures.Segment {
	var segments []*dataStructures.Segment

	end := *lastUpdatedDate
	start := end.AddDate(0, -3, 0)

	segments = append(segments, &dataStructures.Segment{
		StartTime: &start,
		EndTime:   &end,
	})

	return segments
}

func (s *RowsValidator) getCounts(ctx context.Context, billingAccountID string, segment *dataStructures.Segment) (map[tableType]map[dataStructures.HashableSegment]int, []tableRowsCountErrors) {
	logger := s.loggerProvider(ctx)

	var err error

	countErrors := []tableRowsCountErrors{}

	rowsCount := make(map[tableType]map[dataStructures.HashableSegment]int)

	for _, t := range tableTypes {
		for s := range rowsCount[t] {
			rowsCount[t][s] = -1
		}
	}

	if billingAccountID != googleCloudConsts.MasterBillingAccount {
		rowsCount[customerTableType], err = s.tableQuery.GetCustomerRowsCount(ctx, billingAccountID, segment)
		if err != nil {
			countErrors = append(countErrors, tableRowsCountErrors{
				segment: segment,
				tt:      customerTableType,
				err:     err,
			})

			logger.Errorf("%sError getting customer rows count for : %v", logPrefix, billingAccountID, err)
		}
	}

	rowsCount[localTableType], err = s.tableQuery.GetLocalRowsCount(ctx, billingAccountID, segment)
	if err != nil {
		countErrors = append(countErrors, tableRowsCountErrors{
			segment: segment,
			tt:      localTableType,
			err:     err,
		})

		logger.Errorf("%sError getting local rows count for %s: %v", logPrefix, billingAccountID, err)
	}

	rowsCount[unifiedTableType], err = s.tableQuery.GetLUnifiedRowsCount(ctx, billingAccountID, segment)
	if err != nil {
		countErrors = append(countErrors, tableRowsCountErrors{
			segment: segment,
			tt:      unifiedTableType,
			err:     err,
		})

		logger.Errorf("%sError getting unified rows count for %s: %v", logPrefix, billingAccountID, err)
	}

	return rowsCount, countErrors
}
