package application

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"
	resty "github.com/go-resty/resty/v2"

	googleCloudConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	"github.com/doitintl/hello/scheduled-tasks/common"
	billingCommon "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/service"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/bq_utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/shared"
	sharedDal "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/shared/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/shared/domain"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func NewInternalManager(log logger.Provider, conn *connection.Connection) *InternalManager {
	return &InternalManager{
		loggerProvider: log,
		Connection:     conn,
		metadata:       service.NewMetadata(log, conn),
		table:          service.NewTable(log, conn),
		assets:         service.NewAssets(log, conn),
		tQuery:         service.NewTableQuery(log, conn),
		bqUtils:        bq_utils.NewBQ_UTils(log, conn),
		job:            service.NewJob(log, conn),
		dataCopier:     service.NewBillingDataCopierService(log, conn),
		importStatus:   sharedDal.NewBillingImportStatusWithClient(conn.Firestore(context.Background())),
		billingEvent:   sharedDal.NewBillingUpdateFirestoreWithClient(func(ctx context.Context) *firestore.Client { return conn.Firestore(context.Background()) }),
	}
}

type InternalManager struct {
	loggerProvider logger.Provider
	*connection.Connection
	metadata     service.Metadata
	table        service.Table
	assets       *service.Assets
	tQuery       service.TableQuery
	bqUtils      *bq_utils.BQ_Utils
	job          *service.Job
	dataCopier   *service.BillingDataCopierService
	importStatus sharedDal.BillingImportStatus
	billingEvent *sharedDal.BillingUpdateFirestore
}

func (m *InternalManager) RunInternalManager(ctx context.Context) (err error) {
	startTime := time.Now()
	logger := m.loggerProvider(ctx)
	errorHandler := func(err error) error {
		//TODO handle error
		return err
	}

	imm, err := m.initInternalManagerTask(ctx)
	if err != nil {
		return errorHandler(err)
	}

	ctx, cancelF, err := utils.SetupContext(ctx, logger, fmt.Sprintf(consts.CtxInternalManagerTemplate, imm.Iteration))
	defer cancelF()

	if err != nil {
		return errorHandler(err)
	}

	imm, err = m.updateInternalTasks(ctx, imm)
	if err != nil {
		return errorHandler(err)
	}

	imm, err = m.createInternalTasks(ctx, imm)
	if err != nil {
		return errorHandler(err)
	}

	imm, err = m.waitUntilTasksAreDone(ctx, imm)
	if err != nil {
		return errorHandler(err)
	}

	imm, err = m.copyRowsToUnifiedTable(ctx, imm)
	if err != nil {
		return errorHandler(err)
	}

	imm, err = m.notifyAll(ctx, imm)
	if err != nil {
		return errorHandler(err)
	}

	imm, err = m.handleLifeCycleChanges(ctx, imm)
	if err != nil {
		return errorHandler(err)
	}

	err = m.tearDown(ctx, imm)
	if err != nil {
		return errorHandler(err)
	}

	logger.Infof("@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@ END OF INTERNAL ITERATION #%d AFTER %v @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@", imm.Iteration, time.Since(startTime))

	return nil
}

func (m *InternalManager) SingleRecovery(ctx context.Context) error {
	logger := m.loggerProvider(ctx)
	logger.Infof("****** SINGLE RECOVERY STARTED ******")

	itms, err := m.metadata.GetInternalTasksMetadata(ctx)
	if err != nil {
		err = fmt.Errorf("recovery failed. Caused by %s", err)
		logger.Error(err)
		logger.Infof("****** SINGLE RECOVERY FAILED ******")

		return err
	}

	for _, itm := range itms {
		if itm.State == dataStructures.InternalTaskStateDone || itm.State == dataStructures.InternalTaskStateFailed ||
			itm.State == dataStructures.InternalTaskStateOnboarding || itm.State == dataStructures.InternalTaskStateInitializing ||
			itm.State == dataStructures.InternalTaskStateSkipped {
			continue
		}

		if itm.State == dataStructures.InternalTaskStateVerified || itm.State == dataStructures.InternalTaskStateNotified {
			_, err = m.metadata.SetInternalTaskMetadata(ctx, itm.BillingAccount, func(ctx context.Context, itmo *dataStructures.InternalTaskMetadata) error {
				itmo.State = dataStructures.InternalTaskStateDone
				return nil
			})
			if err != nil {
				err = fmt.Errorf("recovery failed. Caused by %s", err)
				logger.Error(err)
				logger.Infof("****** SINGLE RECOVERY FAILED ******")

				return err
			}

			continue
		}

		if itm.State == dataStructures.InternalTaskStatePending || itm.State == dataStructures.InternalTaskStateRunning {
			_, err = m.metadata.SetInternalTaskMetadata(ctx, itm.BillingAccount, func(ctx context.Context, itmo *dataStructures.InternalTaskMetadata) error {
				itmo.State = dataStructures.InternalTaskStateFailed
				return nil
			})
			if err != nil {
				err = fmt.Errorf("recovery failed. Caused by %s", err)
				logger.Error(err)
				logger.Infof("****** SINGLE RECOVERY FAILED ******")

				return err
			}

			continue
		}

		err = fmt.Errorf("error: SINGLE RECOVERY FAILED. Invalid State=%s detected", itm.State)

		return err
	}

	logger.Infof("****** SINGLE RECOVERY SUCCEEDED ******")

	return nil
}

func (m *InternalManager) GeneralRecovery(ctx context.Context) error {
	logger := m.loggerProvider(ctx)

	bq, err := m.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		//handle error
		return err
	}

	mm, err := m.metadata.GetInternalManagerMetadata(ctx)
	if err != nil {
		//handle error
		logger.Errorf("unable to get managerMetadata. Caused by %s", err)
		return err
	}

	if mm.State == dataStructures.InternalManagerStateDone {
		logger.Infof("****** SKIPPING RECOVERY ******")
		return nil
	}

	if mm.State != dataStructures.InternalManagerStateFailed && time.Now().Before(*mm.TTL) {
		err := fmt.Errorf("terminating iteration since an older iteration still has %v time to run", time.Until(*mm.TTL))
		logger.Error(err)

		return err
	}

	if mm.Recovery.Recovering && time.Now().Before(*mm.Recovery.RecoveringTTL) {
		logger.Infof("****** SKIPPING RECOVERY ******")
		return nil
	}

	logger.Infof("****** STARTING RECOVERY ******")

	mm, err = m.metadata.SetInternalManagerMetadata(ctx, func(ctx context.Context, omm *dataStructures.InternalManagerMetadata) error {
		if omm.Recovery.Recovering && time.Now().Before(*omm.Recovery.RecoveringTTL) {
			err = fmt.Errorf("unable to set recovery MD because seems another recovery is in process.")
			logger.Error(err)

			return err
		}

		omm.Recovery.Recovering = true
		ttl := time.Now().Add(consts.InternalManagerRecoverMaxDuration)
		omm.Recovery.RecoveringTTL = &ttl
		omm.Recovery.Iteration = omm.Recovery.Iteration + 1

		return nil
	})

	if err != nil {
		err = fmt.Errorf("failed to SetInternalManagerMetadata. Caused by %s", err)
		logger.Error(err)
		logger.Infof("****** RECOVERY FAILED ******")

		return err
	}

	err = m.recovery(ctx, mm, bq)
	if err != nil {
		err = fmt.Errorf("recovery failed. Caused by %s", err)
		logger.Error(err)
		logger.Infof("****** RECOVERY FAILED ******")

		return err
	} else {
		logger.Infof("****** RECOVERY DONE ******")
	}

	return nil
}

func (m *InternalManager) recovery(ctx context.Context, mm *dataStructures.InternalManagerMetadata, bq *bigquery.Client) (err error) {
	logger := m.loggerProvider(ctx)

	if time.Now().After(*mm.Recovery.RecoveringTTL) {
		err = errors.New("terminating recovery since time is up")
		logger.Error(err)

		return err
	}

	logger.Infof("recovering from state %s", mm.State)

	switch mm.State {
	case dataStructures.InternalManagerStateFailed:
		err = m.table.DeleteTmpTable(ctx, bq, mm.Iteration)
		if err != nil {
			//TODO handle error
			return m.recovery(ctx, mm, bq)
		}

		_, err = m.metadata.SetInternalManagerMetadata(ctx, func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error {
			mm.State = dataStructures.InternalManagerStateFailed
			return nil
		})

		if err != nil {
			logger.Infof("unable to set metadata. Caused by %s", err)
			return m.recovery(ctx, mm, bq)
		}

		err = m.metadata.MarkAllCurrentInternalTasksAsFailed(ctx, mm.Iteration)
		if err != nil {
			logger.Errorf("unable to mark all internal tasks as failed. Caused by %s", err)
			return err
		}

		return nil
	case dataStructures.InternalManagerStateDone:
		_, err = m.metadata.SetInternalManagerMetadata(ctx, func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error {
			mm.State = dataStructures.InternalManagerStateDone
			return nil
		})

		if err != nil {
			logger.Infof("unable to set metadata. Caused by %s", err)
			return m.recovery(ctx, mm, bq)
		}

		return nil
	case dataStructures.InternalManagerStateCopiedToUnified:
		umm, err := m.notifyAll(ctx, mm)
		if err != nil {
			err = fmt.Errorf("unable to notifyAll. Caused by %s", err)
			logger.Error(err)

			return m.recovery(ctx, mm, bq)
		}

		mm = umm
		_, err = m.metadata.SetInternalManagerMetadata(ctx, func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error {
			mm.State = dataStructures.InternalManagerStateNotified
			return nil
		})

		if err != nil {
			logger.Infof("unable to set metadata. Caused by %s", err)
			return m.recovery(ctx, mm, bq)
		}

	case dataStructures.InternalManagerStateNotified:
		err = m.metadata.MarkAllInternalVerifiedTasksAsDone(ctx)
		if err != nil {
			logger.Errorf("unable to mark all internal tasks as done. Caused by %s", err)
			return err
		}

		if err != nil {
			logger.Infof("unable to set metadata. Caused by %s", err)
			return m.recovery(ctx, mm, bq)
		}

		_, err = m.metadata.SetInternalManagerMetadata(ctx, func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error {
			mm.State = dataStructures.InternalManagerStateDone
			return nil
		})

		if err != nil {
			logger.Infof("unable to set metadata. Caused by %s", err)
			return m.recovery(ctx, mm, bq)
		}

		return m.recovery(ctx, mm, bq)
	case dataStructures.InternalManagerStateMarked:
		if mm.CopyToUnifiedTableJob == nil || mm.CopyToUnifiedTableJob.JobID == "" {
			logger.Infof("jobID for task CopyToUnifiedTableJob not found. Attempting to recover")

			jobID, err := m.job.GetJobByPrefix(ctx, bq, utils.GetCopyToUnifiedTableJobPrefix(mm.Iteration))
			if err != nil {
				//handle
				logger.Errorf("unable to get jobID from jobs. Caused by %s", err)
				return m.recovery(ctx, mm, bq)
			}

			if jobID == "" {
				mm.State = dataStructures.InternalManagerStateFailed
				return m.recovery(ctx, mm, bq)
			}

			_, err = m.metadata.SetInternalManagerMetadata(ctx, func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error {
				mm.CopyToUnifiedTableJob = &dataStructures.Job{
					JobID:     jobID,
					JobStatus: dataStructures.JobCreated,
				}

				return nil
			})

			return m.recovery(ctx, mm, bq)
		}

		switch mm.CopyToUnifiedTableJob.JobStatus {
		case dataStructures.JobCreated:
			job, err := bq.JobFromID(ctx, mm.CopyToUnifiedTableJob.JobID)
			if err != nil {
				logger.Errorf("unable to get JobFromID. Caused by %s", err)
				return m.recovery(ctx, mm, bq)
			}

			jobStatus, err := job.Status(ctx)
			if err != nil {
				logger.Errorf("unable to get jobStatus. Caused by %s", err)
				return m.recovery(ctx, mm, bq)
			}

			if jobStatus.Done() {
				if jobStatus.Err() == nil {
					mm.CopyToUnifiedTableJob.JobStatus = dataStructures.JobDone // save status
					_, err = m.metadata.SetInternalManagerMetadata(ctx, func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error {
						mm.CopyToUnifiedTableJob.JobStatus = dataStructures.JobCreated
						return nil
					})

					return m.recovery(ctx, mm, bq)
				} else {
					mm.CopyToUnifiedTableJob.JobStatus = dataStructures.JobFailed
					_, err = m.metadata.SetInternalManagerMetadata(ctx, func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error {
						mm.CopyToUnifiedTableJob.JobStatus = dataStructures.JobFailed
						return nil
					})

					return m.recovery(ctx, mm, bq)
				}
			} else {
				err = m.job.CancelRunningJob(ctx, bq, mm.CopyToUnifiedTableJob.JobID)
				if err != nil {
					mm.CopyToUnifiedTableJob.JobStatus = dataStructures.JobStuck
					_, err = m.metadata.SetInternalManagerMetadata(ctx, func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error {
						mm.CopyToUnifiedTableJob.JobStatus = dataStructures.JobStuck
						return nil
					})

					return m.recovery(ctx, mm, bq)
				} else {
					mm.CopyToUnifiedTableJob.JobStatus = dataStructures.JobCanceled
					_, err = m.metadata.SetInternalManagerMetadata(ctx, func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error {
						mm.CopyToUnifiedTableJob.JobStatus = dataStructures.JobCanceled
						return nil
					})

					return m.recovery(ctx, mm, bq)
				}
			}
		case dataStructures.JobDone:
			mm.State = dataStructures.InternalManagerStateCopiedToUnified
			return m.recovery(ctx, mm, bq)
		case dataStructures.JobStuck:
			err = m.job.CancelRunningJob(ctx, bq, mm.CopyToUnifiedTableJob.JobID)
			if err != nil {
				logger.Errorf("unable to get job unstuck. Caused by %s", err)
				return m.recovery(ctx, mm, bq)
			}
			//todo handle!
		case dataStructures.JobCanceled:
			mm.State = dataStructures.InternalManagerStateFailed
			return m.recovery(ctx, mm, bq)
		}
	default:
		mm.State = dataStructures.InternalManagerStateFailed
		return m.recovery(ctx, mm, bq)
	}

	return nil
}

func (m *InternalManager) tearDown(ctx context.Context, managerMetadata *dataStructures.InternalManagerMetadata) (err error) {
	logger := m.loggerProvider(ctx)

	bq, err := m.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		err = fmt.Errorf("unable to GetBQClientByProjectID. Caused by %s", err)
		logger.Error(err)

		return err
	}

	err = m.table.DeleteTmpTable(ctx, bq, managerMetadata.Iteration)
	if err != nil {
		err = fmt.Errorf("unable to DeleteTmpTable. Caused by %s", err)
		logger.Error(err)

		return err
	}

	notifiedItms, err := m.metadata.GetAllInternalTasksMetadataByParams(ctx, managerMetadata.Iteration, []dataStructures.InternalTaskState{dataStructures.InternalTaskStateNotified})
	if err != nil {
		err = fmt.Errorf("unable to GetAllInternalTasksMetadataByParams. Caused by %s", err)
		logger.Error(err)

		return err
	}

	doneUpdate := sync.WaitGroup{}
	doneUpdate.Add(len(notifiedItms))

	for _, itm := range notifiedItms {
		go func(itm *dataStructures.InternalTaskMetadata) {
			defer doneUpdate.Done()

			_, err = m.metadata.SetInternalTaskMetadata(ctx, itm.BillingAccount, func(ctx context.Context, itm *dataStructures.InternalTaskMetadata) error {
				itm.State = dataStructures.InternalTaskStateDone
				logger.Infof("MARKING BA %s as DONE", itm.BillingAccount)

				if itm.CopyHistory.Status == dataStructures.CopyHistoryStatusNotified {
					itm.CopyHistory.Status = dataStructures.CopyHistoryStatusDone
				}

				return nil
			})
			if err != nil {
				err = fmt.Errorf("unable to SetInternalTaskMetadata for BA %s. Caused by %s", itm.BillingAccount, err)
				logger.Error(err)
				logger.Infof("MARKED 1 for task update as DONE but FAILED to BA %s", itm.BillingAccount)
			} else {
				logger.Infof("MARKED 1 for task update as DONE and SUCCESSFUL to BA %s", itm.BillingAccount)
			}
		}(itm)
	}

	doneUpdate.Wait()
	logger.Info("DONE marking BAs as DONE")

	_, err = m.metadata.SetInternalManagerMetadata(ctx, func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error {
		mm.State = dataStructures.InternalManagerStateDone
		return nil
	})
	if err != nil {
		err = fmt.Errorf("unable to SetInternalManagerMetadata. Caused by %s", err)
		logger.Error(err)

		return err
	}

	return nil
}

func (m *InternalManager) copyRowsToUnifiedTable(ctx context.Context, managerMetadata *dataStructures.InternalManagerMetadata) (updatedMm *dataStructures.InternalManagerMetadata, err error) {
	logger := m.loggerProvider(ctx)
	logger.Infof("COPYING all rows to %s in iteration %d", consts.UnifiedGCPRawTable, managerMetadata.Iteration)

	verifiedItms, err := m.metadata.GetAllInternalTasksMetadataByParams(ctx, managerMetadata.Iteration, []dataStructures.InternalTaskState{dataStructures.InternalTaskStateVerified})
	if err != nil {
		err = fmt.Errorf("unable to GetAllInternalTasksMetadataByParams. Caused by %s", err)
		logger.Error(err)

		return managerMetadata, err
	}

	if len(verifiedItms) > 0 {
		job, err := m.tQuery.CopyFromTmpTableAllRows(ctx, managerMetadata, verifiedItms)
		if err != nil {
			err = fmt.Errorf("unable to CopyFromTmpTableAllVerifiedRows. Caused by %s", err)

			_, updateErr := m.metadata.SetInternalManagerMetadata(ctx, func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error {
				mm.State = dataStructures.InternalManagerStateFailed
				return nil
			})
			if updateErr != nil {
				updateErr = fmt.Errorf("unable to set metadata. Caused by %s", updateErr)
				logger.Error(updateErr)
			}

			logger.Error(err)

			return nil, err
		}

		updatedMm, err = m.metadata.SetInternalManagerMetadata(ctx, func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error {
			mm.CopyToUnifiedTableJob.JobID = job.ID()
			mm.CopyToUnifiedTableJob.JobStatus = dataStructures.JobCreated

			return nil
		})

		if err != nil {
			err = fmt.Errorf("unable to set metadata. Caused by %s", err)
			logger.Error(err)
		}

		err = m.job.HandleRunningJob(ctx, job, updatedMm.CopyToUnifiedTableJob.WaitToFinishTimeout, consts.InternalTaskMaxExtentionDuration)
		if err != nil {
			logger.Errorf("unable to HandleRunningJob. Caused by %s", err)

			var jobStatus dataStructures.JobStatus

			var immStatus dataStructures.InternalManagerState

			switch err.(type) {
			case *billingCommon.JobExecutionFailure:
				jobStatus = dataStructures.JobFailed
				immStatus = dataStructures.InternalManagerStateFailed

			case *billingCommon.JobExecutionStuck:
				jobStatus = dataStructures.JobStuck
				immStatus = dataStructures.InternalManagerStateFailed

			case *billingCommon.JobCanceled:
				jobStatus = dataStructures.JobCanceled
				immStatus = dataStructures.InternalManagerStateFailed

			case *billingCommon.JobCancelButFinished:
				logger.Infof("marking job as successful")

				jobStatus = dataStructures.JobDone
				immStatus = dataStructures.InternalManagerStateCopiedToUnified
				err = nil
			}

			updatedMm, updateErr := m.metadata.SetInternalManagerMetadata(ctx, func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error {
				if mm.CopyToUnifiedTableJob == nil {
					return fmt.Errorf("invalid metadata state %v", mm)
				}

				mm.CopyToUnifiedTableJob.JobStatus = jobStatus
				mm.State = immStatus

				return nil
			})
			if updateErr != nil {
				updateErr = fmt.Errorf("unable to set metadata. Caused by %s", updateErr)
				logger.Error(updateErr)
			}

			return updatedMm, err
		}
	}

	updatedMm, err = m.metadata.SetInternalManagerMetadata(ctx, func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error {
		if mm.CopyToUnifiedTableJob == nil {
			return fmt.Errorf("invalid metadata state %v", mm)
		}

		mm.CopyToUnifiedTableJob.JobStatus = dataStructures.JobDone
		mm.State = dataStructures.InternalManagerStateCopiedToUnified

		return nil
	})
	if err != nil {
		err = fmt.Errorf("unable to set metadata. Caused by %s", err)
		logger.Error(err)

		return updatedMm, err
	}

	logger.Infof("COPIED all verified rows to %s in iteration %d", consts.UnifiedGCPRawTable, managerMetadata.Iteration)

	return updatedMm, nil
}

func (m *InternalManager) handleLifeCycleChanges(ctx context.Context, managerMetadata *dataStructures.InternalManagerMetadata) (updatedMm *dataStructures.InternalManagerMetadata, err error) {
	logger := m.loggerProvider(ctx)

	var tasksHandledWg sync.WaitGroup

	itms, err := m.metadata.GetInternalTasksMetadata(ctx)
	if err != nil {
		err = fmt.Errorf("unable to GetInternalTasksMetadata. Caused by %s", err)
		logger.Error(err)

		return updatedMm, err
	}

	for _, itm := range itms {
		func(itm *dataStructures.InternalTaskMetadata) {
			tasksHandledWg.Add(1)
			defer tasksHandledWg.Done()

			if itm.LifeCycleStage == "" || itm.LifeCycleStage == dataStructures.LifeCycleStageCreated {
				if !itm.OnBoarding {
					_, err := m.metadata.SetInternalTaskMetadata(ctx, itm.BillingAccount, func(ctx context.Context, oitm *dataStructures.InternalTaskMetadata) error {
						if itm.LifeCycleStage == "" || itm.LifeCycleStage == dataStructures.LifeCycleStageCreated {
							oitm.LifeCycleStage = dataStructures.LifeCycleStageActive
						}

						return nil
					})
					if err != nil {
						err = fmt.Errorf("unable to update metadata for BA %s. Caused by %s", itm.BillingAccount, err)
						logger.Error(err)

						return
					}
				}
			} else if itm.LifeCycleStage == dataStructures.LifeCycleStageDeprecated {
				if itm.InternalTaskJobs.DeleteFromUnifiedTable != nil && itm.InternalTaskJobs.DeleteFromUnifiedTable.JobID != "" {
					bq, err := m.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
					if err != nil {
						err = fmt.Errorf("unable to GetBQClientByProjectID. Caused by %s", err)
						logger.Error(err)

						return
					}

					jobStatus, err := m.job.GetJobStatus(ctx, bq, itm.InternalTaskJobs.DeleteFromUnifiedTable.JobID, consts.DoitLocation)
					if err != nil {
						err = fmt.Errorf("unable to get jobStatus for job %s. Caused by %s", itm.InternalTaskJobs.DeleteFromUnifiedTable.JobID, err)
						logger.Error(err)

						return
					}

					if jobStatus.Done() {
						if jobStatus.Err() != nil {
							err = fmt.Errorf("unable to execute job %s. Caused by %s. Details: %s", itm.InternalTaskJobs.DeleteFromUnifiedTable.JobID, jobStatus.Err(), jobStatus.Errors)
							logger.Error(err)
							logger.Infof("retrying execution of job for BA %s", itm.BillingAccount)

							_, err = m.metadata.SetInternalTaskMetadata(ctx, itm.BillingAccount, func(ctx context.Context, oitm *dataStructures.InternalTaskMetadata) error {
								if oitm.LifeCycleStage == dataStructures.LifeCycleStageDeprecated {
									oitm.InternalTaskJobs.DeleteFromUnifiedTable.JobStatus = dataStructures.JobFailed
									itm.InternalTaskJobs.DeleteFromUnifiedTable.JobID = ""
								}

								return nil
							})
							if err != nil {
								err = fmt.Errorf("unable to update metadata for BA %s. Caused by %s", itm.BillingAccount, err)
								logger.Error(err)

								return
							}

							return
						}

						if m.canCreateBillingUpdateEvent(itm.BillingAccount) && jobStatus.Err() == nil {
							logger.Infof("finished deleting rows of unified table for SA %s", itm.BillingAccount)
							itm.InternalTaskJobs.DeleteFromUnifiedTable.JobStatus = dataStructures.JobDone
							//event for offboarding
							now := time.Now().UTC()

							var startTime time.Time
							if itm.BQTable.OldestPartition != nil {
								startTime = itm.BQTable.OldestPartition.UTC().Truncate(time.Hour * 24)
							} else {
								err = fmt.Errorf("unable to CreateBillingUpdateEvent. Caused by %s", fmt.Errorf("unable to find oldestRecord in metadata"))
								logger.Error(err)

								return
							}

							var endTime time.Time
							if itm.Segment != nil && itm.Segment.EndTime != nil {
								endTime = itm.Segment.EndTime.UTC().Truncate(time.Hour * 24)
							} else {
								endTime = time.Now().UTC().Truncate(time.Hour * 24)
							}

							err = m.billingEvent.CreateBillingUpdateEvent(ctx, &domain.BillingEvent{
								BillingAccountID: itm.BillingAccount,
								TimeCreated:      &now,
								TimeCompleted:    nil,
								EventType:        domain.BillingUpdateEventOffboarding,
								EventRange: domain.Range{
									StartTime: &startTime,
									EndTime:   &endTime,
								},
							})
							if err != nil {
								err = fmt.Errorf("unable to CreateBillingUpdateEvent. Caused by %s", err)
								logger.Error(err)

								return
							}
						}

						err = m.metadata.DeleteInternalTaskMetadata(ctx, itm.BillingAccount)
						if err != nil {
							err = fmt.Errorf("unable to DeleteInternalTaskMetadata. Caused by %s", err)
							logger.Error(err)
						} else {
							logger.Infof("Internal MD for BA %s deleted successfully", itm.BillingAccount)
						}

						return
					}
				} else {
					toStart := time.Now().Add(consts.WaitForJobOnTaskMaxDuration)
					toEnd := time.Now().Add(consts.WaitForInternalFlowToFinishDeletingBillingData)

					job, err := m.tQuery.DeleteRowsFromUnifiedByBA(ctx, itm.BillingAccount)
					if err != nil {
						err = fmt.Errorf("unable to DeleteRowsFromUnifiedByBA. Caused by %s", err)
						logger.Error(err)

						return
					}

					_, err = m.metadata.SetInternalTaskMetadata(ctx, itm.BillingAccount, func(ctx context.Context, oitm *dataStructures.InternalTaskMetadata) error {
						if oitm.LifeCycleStage == dataStructures.LifeCycleStageDeprecated {
							oitm.InternalTaskJobs.DeleteFromUnifiedTable = &dataStructures.Job{
								WaitToStartTimeout:  &toStart,
								WaitToFinishTimeout: &toEnd,
								JobID:               job.ID(),
								JobStatus:           dataStructures.JobCreated,
							}
						}

						return nil
					})
					if err != nil {
						err = fmt.Errorf("unable to update metadata for BA %s. Caused by %s", itm.BillingAccount, err)
						logger.Error(err)
					}
				}
			} else if itm.LifeCycleStage == dataStructures.LifeCycleStagePaused {
				logger.Warning("Internal flow is paused for BA %s", itm.BillingAccount)
			}
		}(itm)
	}

	tasksHandledWg.Wait()

	updatedMm, err = m.metadata.SetInternalManagerMetadata(ctx, func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error {
		mm.State = dataStructures.InternalManagerStateNotified
		return nil
	})
	if err != nil {
		err = fmt.Errorf("unable to set metadata. Caused by %s", err)
		logger.Error(err)

		return nil, err
	}

	return updatedMm, nil
}
func (m *InternalManager) canCreateBillingUpdateEvent(BillingAccount string) bool {
	return BillingAccount != googleCloudConsts.MasterBillingAccount
}
func (m *InternalManager) notifyAll(ctx context.Context, managerMetadata *dataStructures.InternalManagerMetadata) (updatedMm *dataStructures.InternalManagerMetadata, err error) {
	logger := m.loggerProvider(ctx)

	var notifyingTasks sync.WaitGroup

	verifiedTasks, err := m.metadata.GetAllInternalTasksMetadataByParams(ctx, managerMetadata.Iteration, []dataStructures.InternalTaskState{dataStructures.InternalTaskStateVerified})
	if err != nil {
		err = fmt.Errorf("unable to GetInternalTasksMetadata. Caused by %s", err)
		logger.Error(err)

		return updatedMm, err
	}

	notifyingTasks.Add(len(verifiedTasks))

	for _, itm := range verifiedTasks {
		func(itm *dataStructures.InternalTaskMetadata) {
			defer notifyingTasks.Done()

			if itm.State != dataStructures.InternalTaskStateVerified {
				logger.Infof("SKIPPING BA %s since state is NOT %s", itm.BillingAccount, itm.State)
				return
			}

			err = m.notifyCopyHistoryFinishedIfNecessary(ctx, itm)
			if err != nil {
				err = fmt.Errorf("unable to notifyCopyHistoryFinishedIfNecessary for BA %s. Caused by %s", itm.BillingAccount, err)
				logger.Error(err)
			}

			err = m.notifyOldPartitionUpdateIfNecessary(ctx, itm)
			if err != nil {
				err = fmt.Errorf("unable to notifyOldPartitionUpdateIfNecessary for BA %s. Caused by %s", itm.BillingAccount, err)
				logger.Error(err)

				return
			}

			_, err := m.metadata.SetInternalTaskMetadata(ctx, itm.BillingAccount, func(ctx context.Context, itm *dataStructures.InternalTaskMetadata) error {
				itm.State = dataStructures.InternalTaskStateNotified
				return nil
			})
			if err != nil {
				err = fmt.Errorf("unable to update metadata for BA %s. Caused by %s", itm.BillingAccount, err)
				logger.Error(err)
			}
		}(itm)
	}

	notifyingTasks.Wait()

	updatedMm, err = m.metadata.SetInternalManagerMetadata(ctx, func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error {
		mm.State = dataStructures.InternalManagerStateNotified
		return nil
	})
	if err != nil {
		err = fmt.Errorf("unable to set metadata. Caused by %s", err)
		logger.Error(err)

		return nil, err
	}

	return updatedMm, nil
}

func (m *InternalManager) notifyCopyHistoryFinishedIfNecessary(ctx context.Context, itm *dataStructures.InternalTaskMetadata) error {
	logger := m.loggerProvider(ctx)
	if itm.CopyHistory.Status != dataStructures.CopyHistoryStatusCopying || itm.CopyHistory.TargetTime.After(*itm.Segment.EndTime) {
		logger.Infof("SKIPPING notifyCopyHistoryFinishedIfNecessary of BA %s", itm.BillingAccount)
		return nil
	}

	//TODO add dummy
	if m.canCreateBillingUpdateEvent(itm.BillingAccount) && !itm.Dummy {
		logger.Infof("NOTIFYING notifyCopyHistoryFinishedIfNecessary of BA %s", itm.BillingAccount)

		is, err := m.importStatus.GetBillingImportStatus(ctx, itm.CustomerID, itm.BillingAccount)
		if err != nil {
			err = fmt.Errorf("unable to GetBillingImportStatus. Caused by %s", err)
			logger.Error(err)

			return err
		}

		if is.Status == shared.BillingImportStatusStarted || is.Status == shared.BillingImportStatusPending {
			err = m.importStatus.SetStatusCompleted(ctx, itm.CustomerID, itm.BillingAccount)
			if err != nil {
				err = fmt.Errorf("unable to SetStatusCompleted. Caused by %s", err)
				logger.Error(err)

				return err
			}
		}

		now := time.Now()

		startTime, err := m.tQuery.GetLocalTableOldestRecordTime(ctx, itm)
		if err != nil {
			err = fmt.Errorf("unable to GetLocalTableOldestRecordTime. Caused by %s", err)
			logger.Error(err)

			return err
		}

		startTime = startTime.UTC().Truncate(time.Hour * 24)
		endTime := itm.Segment.EndTime.UTC().Truncate(time.Hour * 24)

		err = m.billingEvent.CreateBillingUpdateEvent(ctx, &domain.BillingEvent{
			BillingAccountID: itm.BillingAccount,
			TimeCreated:      &now,
			TimeCompleted:    nil,
			EventType:        domain.BillingUpdateEventOnboarding,
			EventRange: domain.Range{
				StartTime: &startTime,
				EndTime:   &endTime,
			},
		})
		if err != nil {
			err = fmt.Errorf("unable to CreateBillingUpdateEvent. Caused by %s", err)
			logger.Error(err)

			return err
		}
	} else {
		logger.Infof("SKIPPING notifyCopyHistoryFinishedIfNecessary of BA %s", itm.BillingAccount)
	}

	_, err := m.metadata.SetInternalTaskMetadata(ctx, itm.BillingAccount, func(ctx context.Context, itm *dataStructures.InternalTaskMetadata) error {
		itm.CopyHistory.Status = dataStructures.CopyHistoryStatusNotified
		return nil
	})

	if err != nil {
		err = fmt.Errorf("unable to update metadata for BA %s. Caused by %s", itm.BillingAccount, err)
		logger.Error(err)

		return err
	}

	return nil
}

func (m *InternalManager) notifyOldPartitionUpdateIfNecessary(ctx context.Context, itm *dataStructures.InternalTaskMetadata) error {
	logger := m.loggerProvider(ctx)
	if itm.CopyHistory.Status != dataStructures.CopyHistoryStatusDone || !itm.Segment.StartTime.Before(time.Now().Truncate(time.Hour*24).AddDate(0, 0, -1)) {
		logger.Infof("SKIPPING notifyOldPartitionUpdateIfNecessary of BA %s", itm.BillingAccount)
		return nil
	}

	if m.canCreateBillingUpdateEvent(itm.BillingAccount) {
		logger.Infof("NOTIFYING notifyOldPartitionUpdateIfNecessary of BA %s", itm.BillingAccount)
		startTime := itm.Segment.StartTime.UTC().Truncate(time.Hour * 24)
		endTime := itm.Segment.EndTime.UTC().Truncate(time.Hour * 24)
		now := time.Now()

		err := m.billingEvent.CreateBillingUpdateEvent(ctx, &domain.BillingEvent{
			BillingAccountID: itm.BillingAccount,
			TimeCreated:      &now,
			TimeCompleted:    nil,
			EventType:        domain.BillingUpdateEventBackfill,
			EventRange: domain.Range{
				StartTime: &startTime,
				EndTime:   &endTime,
			},
		})
		if err != nil {
			err = fmt.Errorf("unable to CreateBillingUpdateEvent. Caused by %s", err)
			logger.Error(err)

			return err
		}

		logger.Infof("NOTIFICATION of BA %s [%v, %v]", itm.BillingAccount, itm.Segment.StartTime.Truncate(24*time.Hour), itm.Segment.EndTime.Truncate(24*time.Hour))
	}

	return nil
}

func (m *InternalManager) waitUntilTasksAreDone(ctx context.Context, managerMetadata *dataStructures.InternalManagerMetadata) (updatedMm *dataStructures.InternalManagerMetadata, err error) {
	logger := m.loggerProvider(ctx)

	var tasksDone sync.WaitGroup

	logger.Info("START waitUntilTasksAreDone")

	itms, err := m.metadata.GetAllInternalTasksMetadataByParams(ctx, managerMetadata.Iteration, []dataStructures.InternalTaskState{dataStructures.InternalTaskStatePending, dataStructures.InternalTaskStateRunning})
	if err != nil {
		//handle error
		return nil, err
	}

	for _, itm := range itms {
		logger.Infof("BA %s started", itm.BillingAccount)
		tasksDone.Add(1)

		go func(ctx context.Context, itm *dataStructures.InternalTaskMetadata, tasksDone *sync.WaitGroup) {
			err = m.waitUntilTaskIsDone(ctx, itm, tasksDone)
			if err != nil {
				logger.Errorf("TASK for BA %s failed.", itm.BillingAccount)
			}
		}(ctx, itm, &tasksDone)
	}

	logger.Info("WAITING for waitUntilTaskIsDone to be done")
	tasksDone.Wait()
	logger.Info("DONE waiting waitUntilTaskIsDone")

	toStart := time.Now().Add(consts.WaitForJobOnTaskMaxDuration)
	toEnd := time.Now().Add(consts.InternalTaskMaxDuration)
	updatedMm, err = m.metadata.SetInternalManagerMetadata(ctx, func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error {
		mm.State = dataStructures.InternalManagerStateTasksDone
		mm.CopyToUnifiedTableJob.WaitToStartTimeout = &toStart
		mm.CopyToUnifiedTableJob.WaitToFinishTimeout = &toEnd

		return nil
	})

	return updatedMm, nil
}

func (m *InternalManager) waitUntilTaskIsDone(ctx context.Context, itm *dataStructures.InternalTaskMetadata, tasksDone *sync.WaitGroup) (err error) {
	logger := m.loggerProvider(ctx)

	var jobID string

	defer func() {
		tasksDone.Done()

		if err != nil {
			logger.Errorf("MARKED job %s of BA %s as done but FAILED. Caused by ", jobID, itm.BillingAccount, err)
		} else {
			logger.Infof("MARKED job %s of BA %s as done and SUCCEEDED", jobID, itm.BillingAccount)
		}
	}()

	if itm.State == dataStructures.InternalTaskStateOnboarding || itm.State == dataStructures.InternalTaskStateSkipped {
		logger.Infof("SKIPPING process since BA %s is %s", itm.BillingAccount, string(itm.State))
		return nil
	}

	job, err := m.waitUntilJobStarts(ctx, itm)
	if err != nil {
		logger.Errorf("unable to waitUntilJobStarts for BA %s. Caused by %s", itm.BillingAccount, err)
		_, updateErr := m.metadata.SetInternalTaskMetadata(ctx, itm.BillingAccount, func(ctx context.Context, itm *dataStructures.InternalTaskMetadata) error {
			itm.State = dataStructures.InternalTaskStateFailed
			return nil
		})
		updateErr = fmt.Errorf("unable to SetInternalTaskMetadata. Caused by %s", updateErr)
		logger.Error(updateErr)

		return err
	}

	jobID = job.ID()

	var jobStatus dataStructures.JobStatus

	var itmStatus dataStructures.InternalTaskState

	jobStatus = dataStructures.JobDone
	itmStatus = dataStructures.InternalTaskStateVerified

	err = m.job.HandleRunningJob(ctx, job, itm.InternalTaskJobs.FromLocalTableToTmpTable.WaitToFinishTimeout, consts.InternalTaskMaxExtentionDuration)
	if err != nil {
		logger.Errorf("unable to HandleRunningJob. Caused by %s", err)

		switch err.(type) {
		case *billingCommon.JobExecutionFailure:
			jobStatus = dataStructures.JobFailed
			itmStatus = dataStructures.InternalTaskStateFailed

		case *billingCommon.JobExecutionStuck:
			jobStatus = dataStructures.JobStuck
			itmStatus = dataStructures.InternalTaskStateFailed

		case *billingCommon.JobCanceled:
			jobStatus = dataStructures.JobCanceled
			itmStatus = dataStructures.InternalTaskStateFailed

		case *billingCommon.JobCancelButFinished:
			logger.Infof("marking job as successful")

			jobStatus = dataStructures.JobDone
			err = nil
		}
	}

	_, updateErr := m.metadata.SetInternalTaskMetadata(ctx, itm.BillingAccount, func(ctx context.Context, itm *dataStructures.InternalTaskMetadata) error {
		itm.State = itmStatus
		itm.InternalTaskJobs.FromLocalTableToTmpTable.JobStatus = jobStatus

		return nil
	})
	if updateErr != nil {
		updateErr = fmt.Errorf("unable to set metadata for BA %s. Caused by %s", itm.BillingAccount, updateErr)
		logger.Error(updateErr)
	}

	return err
}

func (m *InternalManager) waitUntilJobStarts(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (job *bigquery.Job, err error) {
	logger := m.loggerProvider(ctx)

	bq, err := m.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		err = fmt.Errorf("unable to GetBQClientByProjectID. Caused by %s", err)
		logger.Error(err)

		return nil, err
	}

	jobStartedCh := make(chan struct{})
	errorCh := make(chan struct{})

	go func() {
		for {
			time.Sleep(10 * time.Second)

			itm, err = m.metadata.GetInternalTaskMetadata(ctx, itm.BillingAccount)
			if err != nil {
				//handle error
				errorCh <- struct{}{}

				logger.Errorf("unable to get internalMetadata %s", itm.BillingAccount)

				break
			}

			if itm.State == dataStructures.InternalTaskStateFailed || itm.State == dataStructures.InternalTaskStateOnboarding {
				//handle error
				logger.Errorf("BA %s internalTask invalid status %s", itm.BillingAccount, itm.State)
				errorCh <- struct{}{}

				break
			}

			if itm.InternalTaskJobs == nil || itm.InternalTaskJobs.FromLocalTableToTmpTable == nil {
				continue
			}

			if itm.InternalTaskJobs.FromLocalTableToTmpTable.JobID != "" {
				logger.Infof("BA %s job %s found", itm.BillingAccount, itm.InternalTaskJobs.FromLocalTableToTmpTable.JobID)
				jobStartedCh <- struct{}{}

				break
			}
		}
	}()

	select {
	case <-time.After(time.Until(*itm.InternalTaskJobs.FromLocalTableToTmpTable.WaitToStartTimeout)):
		logger.Errorf("timeout for starting job for billingID %s", itm.BillingAccount)
		return nil, fmt.Errorf("timeout for starting job for billingID %s", itm.BillingAccount)
	case <-errorCh:
		logger.Errorf("unable to wait for job for billingID %s. Caused by:%s", itm.BillingAccount, err)
		return nil, fmt.Errorf("unable to wait for job for billingID %s. Caused by:%s", itm.BillingAccount, err)
	case <-jobStartedCh:
		time.Sleep(10 * time.Second)

		job, err = bq.JobFromID(ctx, itm.InternalTaskJobs.FromLocalTableToTmpTable.JobID)
		if err != nil {
			err = fmt.Errorf("unable to JobFromID for job %s. Caused by %s", itm.InternalTaskJobs.FromLocalTableToTmpTable.JobID, err)
			logger.Error(err)

			return nil, err
		}

		return job, nil
	}
}

func (m *InternalManager) initInternalManagerTask(ctx context.Context) (internalManagerMetadata *dataStructures.InternalManagerMetadata, err error) {
	logger := m.loggerProvider(ctx)

	err = m.GeneralRecovery(ctx)
	if err != nil {
		return nil, err
	}

	err = m.SingleRecovery(ctx)
	if err != nil {
		return nil, err
	}

	internalManagerMetadata, err = m.metadata.CatchInternalManagerMetadata(ctx)
	if err != nil {
		//TODO handle error
		logger.Errorf("unable to catch Internal manager metadata. Caused by %s", err)
		return nil, err
	}

	logger.Infof("############################## STARTING INTERNAL ITERATION #%d ##############################", internalManagerMetadata.Iteration)

	err = m.table.CreateTmpTable(ctx, internalManagerMetadata.Iteration)
	if err != nil {
		//TODO handle error
		return nil, err
	}

	internalManagerMetadata, err = m.metadata.SetInternalManagerMetadata(ctx, func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error {
		mm.State = dataStructures.InternalManagerStateTmpTableCreated
		return nil
	})
	if err != nil {
		//handle error
		return nil, err
	}

	return internalManagerMetadata, nil
}

func (m *InternalManager) createInternalTasks(ctx context.Context, managerMetadata *dataStructures.InternalManagerMetadata) (updatedManagerMetadata *dataStructures.InternalManagerMetadata, err error) {
	logger := m.loggerProvider(ctx)

	pendingTasks, err := m.metadata.GetAllInternalTasksMetadataByParams(ctx, managerMetadata.Iteration, []dataStructures.InternalTaskState{dataStructures.InternalTaskStatePending})
	if err != nil {
		err = fmt.Errorf("unable to GetAllInternalTasksMetadataByParams. Caused by %s", err)
		logger.Error(err)

		return nil, err
	}

	var tasks sync.WaitGroup

	tasks.Add(len(pendingTasks))

	for _, internalTask := range pendingTasks {
		logger.Infof("CREATING cloud-task for BA %s", internalTask.BillingAccount)

		go func(internalTask *dataStructures.InternalTaskMetadata) {
			BA := internalTask.BillingAccount
			defer func() {
				tasks.Done()
				logger.Infof("DONE creating cloud-task for BA %s", BA)
			}()

			err = m.createInternalCloudTask(ctx, internalTask)
			if err != nil {
				//TODO handle error
				logger.Errorf("unable to createInternalCloudTask for BA %s. Caused by %s", err)

				_, err = m.metadata.SetInternalTaskMetadata(ctx, internalTask.BillingAccount, func(ctx context.Context, itm *dataStructures.InternalTaskMetadata) error {
					itm.State = dataStructures.InternalTaskStateFailed
					return nil
				})
				if err != nil {
					logger.Errorf("unable to SetInternalTaskMetadata for BA %s. Caused by %s", err)
				}
			}
		}(internalTask)
	}

	logger.Info("WAITING for cloud-tasks to be dispatched")
	tasks.Wait()
	logger.Info("DONE all cloud-tasks dispatched")

	updatedManagerMetadata, err = m.metadata.SetInternalManagerMetadata(ctx, func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error {
		mm.State = dataStructures.InternalManagerStateTasksCreated
		return nil
	})

	return updatedManagerMetadata, nil
}

func (m *InternalManager) updateInternalTasks(ctx context.Context, managerMetadata *dataStructures.InternalManagerMetadata) (updatedManagerMetadata *dataStructures.InternalManagerMetadata, err error) {
	logger := m.loggerProvider(ctx)

	internalTasks, err := m.metadata.GetActiveInternalTasksMetadata(ctx)
	if err != nil {
		//TODO handle error
		return nil, err
	}

	taskTTL := managerMetadata.TTL.Add(-time.Minute)

	var tasks sync.WaitGroup

	logger.Info("START updateInternalTasks step")

	for _, internalTask := range internalTasks {
		tasks.Add(1)
		logger.Infof("start internalTask metadata for BA %s", internalTask.BillingAccount)

		go func(internalTask *dataStructures.InternalTaskMetadata) {
			BA := internalTask.BillingAccount
			defer func() {
				tasks.Done()
				logger.Infof("done internalTask metadata for BA %s", BA)
			}()

			internalTask, err = m.updateTaskMetadata(ctx, internalTask, managerMetadata.Iteration, taskTTL)
			if err != nil {
				err = fmt.Errorf("unable to SetInternalTaskMetadata for BA %s. Caused by %s", BA, err)
				logger.Error(err)
			} else {
				logger.Infof("MD for BA %s updated. expected segment %+v", BA, internalTask.Segment)
			}
		}(internalTask)
	}

	logger.Info("WAITING for BAs to finish updating internalTask metadata")
	tasks.Wait()
	logger.Info("DONE waiting for BAs to finish updating internalTask metadata")
	logger.Info("DONE updateInternalTasks step")

	updatedManagerMetadata, err = m.metadata.SetInternalManagerMetadata(ctx, func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error {
		mm.State = dataStructures.InternalManagerStateTasksUpdated
		return nil
	})

	return updatedManagerMetadata, nil
}

func (m *InternalManager) createInternalCloudTask(ctx context.Context, internalTaskMetadata *dataStructures.InternalTaskMetadata) error {
	logger := m.loggerProvider(ctx)
	body := &dataStructures.UpdateRequestBody{
		BillingAccountID: internalTaskMetadata.BillingAccount,
		Iteration:        internalTaskMetadata.Iteration,
	}

	if internalTaskMetadata.State == dataStructures.InternalTaskStateOnboarding || internalTaskMetadata.State == dataStructures.InternalTaskStateSkipped {
		logger.Infof("SKIPPING createInternalCloudTask since the state of the BA %s is %s", internalTaskMetadata.BillingAccount, string(internalTaskMetadata.State))
		return nil
	}

	if common.IsLocalhost {
		body := &dataStructures.UpdateRequestBody{
			BillingAccountID: internalTaskMetadata.BillingAccount,
			Iteration:        internalTaskMetadata.Iteration,
		}
		restClient := resty.New()

		logger.Infof("sending task %+v", body)

		response, err := restClient.R().SetBody(body).Post(fmt.Sprintf("http://localhost:%s/tasks/flexsave-standalone/google-cloud/billing/internal/tasks", os.Getenv("PORT")))
		if err != nil {
			logger.Errorf("unable to run task %+v. Caused by %s", body, err.Error())
		} else {
			logger.Infof("task %+v triggered. Details: %+v", body, response.RawResponse)
		}

		logger.Info(body)
	} else {
		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   "/tasks/flexsave-standalone/google-cloud/billing/internal/tasks",
			Queue:  common.TaskQueueFlexSaveStandaloneInternalTasks,
		}

		task, err := m.CloudTaskClient.CreateTask(ctx, config.Config(body))
		if err != nil {
			logger.Errorf("unable to schedule task. Caused by %s", err)
			return err
		}

		logger.Infof("scheduled task %s", task.String())

		return nil
	}

	return nil
}

func (m *InternalManager) updateTaskMetadata(ctx context.Context, internalTaskMetadata *dataStructures.InternalTaskMetadata, iteration int64, taskTTL time.Time) (updatedInternalMetadata *dataStructures.InternalTaskMetadata, err error) {
	logger := m.loggerProvider(ctx)
	toStart := time.Now().Add(consts.WaitForJobOnTaskMaxDuration)
	toEnd := time.Now().Add(consts.InternalTaskMaxDuration)

	return m.metadata.SetInternalTaskMetadata(ctx, internalTaskMetadata.BillingAccount, func(ctx context.Context, itm *dataStructures.InternalTaskMetadata) error {
		switch itm.State {
		case dataStructures.InternalTaskStateSkipped:
			itm.TTL = &taskTTL
			itm.Iteration = iteration

			latestTime, err := m.tQuery.GetLocalTableNewestRecordTime(ctx, itm)
			if err != nil {
				return err
			}

			if !itm.Segment.EndTime.Equal(latestTime) {
				itm.State = dataStructures.InternalTaskStateDone
				return nil
			}

			logger.Infof("SKIPPING process since no new lines were found for BA %s", itm.BillingAccount)
		case dataStructures.InternalTaskStateDone:
			itm.TTL = &taskTTL
			itm.Iteration = iteration
			itm.InternalTaskJobs = &dataStructures.InternalTaskJobs{
				FromLocalTableToTmpTable: &dataStructures.Job{
					JobID:               "",
					JobStatus:           dataStructures.JobPending,
					WaitToStartTimeout:  &toStart,
					WaitToFinishTimeout: &toEnd,
				},
			}
			err = m.calculateNextSegment(ctx, itm)

			if err != nil {
				logger.Errorf("unable to calculate next Segment for BA %s. Caused by %s", itm.BillingAccount, err)
				return err
			}

			if itm.Segment.StartTime.Equal(*itm.Segment.EndTime) {
				logger.Infof("SKIPPING process since no new lines were found for BA %s", itm.BillingAccount)
				itm.State = dataStructures.InternalTaskStateSkipped

				return nil
			}

			itm.State = dataStructures.InternalTaskStatePending
		case dataStructures.InternalTaskStateOnboarding:
			itm.Iteration = iteration
		case dataStructures.InternalTaskStateInitializing:
			itm.TTL = &taskTTL
			itm.State = dataStructures.InternalTaskStatePending
			itm.Iteration = iteration
			err = m.calculateNextSegment(ctx, itm)

			if err != nil {
				logger.Errorf("unable to calculate next Segment for BA %s. Caused by %s", itm.BillingAccount, err)
				return err
			}

			itm.InternalTaskJobs = &dataStructures.InternalTaskJobs{
				FromLocalTableToTmpTable: &dataStructures.Job{
					JobID:               "",
					JobStatus:           dataStructures.JobPending,
					WaitToStartTimeout:  &toStart,
					WaitToFinishTimeout: &toEnd,
				},
			}
			itm.CopyHistory.Status = dataStructures.CopyHistoryStatusCopying
		case dataStructures.InternalTaskStateFailed:
			if itm.Segment == nil || itm.Segment.StartTime == nil || itm.Segment.EndTime == nil {
				err = m.calculateNextSegment(ctx, itm)
				if err != nil {
					logger.Errorf("unable to calculate next Segment for BA %s. Caused by %s", itm.BillingAccount, err)
					return err
				}
			}

			itm.TTL = &taskTTL
			itm.State = dataStructures.InternalTaskStatePending
			itm.Iteration = iteration
			itm.InternalTaskJobs = &dataStructures.InternalTaskJobs{
				FromLocalTableToTmpTable: &dataStructures.Job{
					JobID:               "",
					JobStatus:           dataStructures.JobPending,
					WaitToStartTimeout:  &toStart,
					WaitToFinishTimeout: &toEnd,
				},
			}
		}

		return nil
	})
}

func (m *InternalManager) calculateNextSegment(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (err error) {
	logger := m.loggerProvider(ctx)

	if itm.Segment == nil || itm.Segment.EndTime == nil {
		startingTime, err := m.tQuery.GetLocalTableOldestRecordTime(ctx, itm)
		if err != nil {
			err = fmt.Errorf("unable to GetLocalTableOldestRecordTime for BA %s. Caused by %s", itm.BillingAccount, err)
			logger.Error(err)

			return err
		}

		startingTime = startingTime.Add(-time.Second)

		itm.Segment = &dataStructures.Segment{
			EndTime: &startingTime,
		}
	}

	itm.Segment.StartTime = itm.Segment.EndTime
	latestTime, err := m.tQuery.GetLocalTableNewestRecordTime(ctx, itm)

	if err != nil {
		err = fmt.Errorf("unable to GetLocalTableNewestRecordTime for BA %s. Caused by %s", itm.BillingAccount, err)
		logger.Error(err)

		return err
	}

	itm.Segment.EndTime = calculateNextEndTime(*itm.Segment.EndTime, latestTime)

	return nil
}

func calculateNextEndTime(startingTime time.Time, tableLatestDate time.Time) *time.Time {
	startingTimePlusThreeMonths := startingTime.AddDate(0, consts.OnboardingSegmentIntervalInMonths, 0)
	//nowMinusBuffer := time.Now().Add(-consts.FetchBufferDuration)
	if startingTimePlusThreeMonths.Before(tableLatestDate) {
		return &startingTimePlusThreeMonths
	}

	return &tableLatestDate
}
