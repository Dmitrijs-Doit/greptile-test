package application

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	resty "github.com/go-resty/resty/v2"

	"github.com/doitintl/hello/scheduled-tasks/common"
	billingCommon "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/service"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/bq_utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type ExternalManager struct {
	loggerProvider logger.Provider
	*connection.Connection
	metadata         service.Metadata
	customerBQClient service.ExternalBigQueryClient
	dataCopier       *service.BillingDataCopierService
	query            service.TableQuery
	bqUtils          *bq_utils.BQ_Utils
	// TODO: remove after debug
	//task    *task.ExternalAccountBillingUpdateTask
	tQuery  service.TableQuery
	job     *service.Job
	bqTable service.Table
	bucket  service.Bucket
}

func NewExternalManager(log logger.Provider, conn *connection.Connection) *ExternalManager {
	return &ExternalManager{
		log,
		conn,
		service.NewMetadata(log, conn),
		service.NewExternalBigQueryClient(log, conn),
		service.NewBillingDataCopierService(log, conn),
		service.NewTableQuery(log, conn),
		// TODO: remove after debug
		bq_utils.NewBQ_UTils(log, conn),
		//task.NewExternalAccountBillingUpdateTask(log, conn),
		service.NewTableQuery(log, conn),
		service.NewJob(log, conn),
		service.NewTable(log, conn),
		service.NewBucket(log, conn),
	}
}

func (e *ExternalManager) RunExternalManager(ctx context.Context) error {
	startTime := time.Now()
	logger := e.loggerProvider(ctx)
	errorHandler := func(err error) error {
		//TODO handle error
		return err
	}

	emm, err := e.initExternalManager(ctx)
	if err != nil {
		return errorHandler(err)
	}

	ctx, cancelF, err := utils.SetupContext(ctx, logger, fmt.Sprintf(consts.CtxExternalManagerTemplate, emm.Iteration))
	defer cancelF()

	if err != nil {
		logger.Errorf("unable to SetupContext. Caused by %s", err)
		return errorHandler(err)
	}

	var filesDeleted sync.WaitGroup

	err = e.manageExternalTasks(ctx, emm, &filesDeleted)
	if err != nil {
		return errorHandler(err)
	}

	err = e.waitUntilFilesAreDeleted(ctx, &filesDeleted)
	if err != nil {
		return errorHandler(err)
	}

	err = e.markOnboardingDone(ctx)
	if err != nil {
		return errorHandler(err)
	}

	err = e.handleLifeCycleChanges(ctx)
	if err != nil {
		return errorHandler(err)
	}

	err = e.tearDown(ctx, emm)
	if err != nil {
		return errorHandler(err)
	}

	logger.Infof("@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@ END OF EXTERNAL ITERATION #%d AFTER %v @@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@", emm.Iteration, time.Now().Sub(startTime))

	return nil
}

func (m *ExternalManager) handleLifeCycleChanges(ctx context.Context) error {
	logger := m.loggerProvider(ctx)

	logger.Infof("WAITING handleLifeCycleChanges")

	etms, err := m.metadata.GetExternalTasksMetadata(ctx)
	if err != nil {
		err = fmt.Errorf("unable to GetExternalTasksMetadata. Caused by %s", err)
		logger.Error(err)

		return err
	}

	var allHandledWG sync.WaitGroup
	for _, etm := range etms {
		allHandledWG.Add(1)

		go func(etm *dataStructures.ExternalTaskMetadata) {
			defer allHandledWG.Done()

			if etm.LifeCycleStage == "" || etm.LifeCycleStage == dataStructures.LifeCycleStageCreated {
				_, err := m.metadata.SetExternalTaskMetadata(ctx, etm.BillingAccount, func(ctx context.Context, oetm *dataStructures.ExternalTaskMetadata) error {
					if etm.LifeCycleStage == "" || etm.LifeCycleStage == dataStructures.LifeCycleStageCreated {
						oetm.LifeCycleStage = dataStructures.LifeCycleStageActive
					}

					return nil
				})
				if err != nil {
					err = fmt.Errorf("unable to update LifeCycleStage of BA %s to %s", etm.BillingAccount, dataStructures.LifeCycleStageActive)
					logger.Error(err)
				}
			} else if etm.LifeCycleStage == dataStructures.LifeCycleStageDeprecated {
				err = m.bqTable.DeleteLocalTable(ctx, etm.BillingAccount)
				if err != nil {
					err = fmt.Errorf("unable to DeleteLocalTable. Caused by %s", err)
					logger.Error(err)

					return
				}

				err = m.metadata.DeleteExternalTaskMetadata(ctx, etm.BillingAccount)
				if err != nil {
					err = fmt.Errorf("unable to DeleteExternalTaskMetadata. Caused by %s", err)
					logger.Error(err)

					return
				}
			} else if etm.LifeCycleStage == dataStructures.LifeCycleStagePaused {
				logger.Warning("External flow is paused for BA %s", etm.BillingAccount)
			}
		}(etm)
	}

	allHandledWG.Wait()
	logger.Infof("DONE WAITING for handleLifeCycleChanges")

	return nil
}

func (m *ExternalManager) markOnboardingDone(ctx context.Context) error {
	logger := m.loggerProvider(ctx)

	etms, err := m.metadata.GetActiveExternalTasksMetadata(ctx)
	if err != nil {
		err = fmt.Errorf("unable to GetExternalTasksMetadata. Caused by %s", err)
		logger.Error(err)

		return err
	}

	var allMarkedWG sync.WaitGroup
	for _, etm := range etms {
		allMarkedWG.Add(1)

		go func(etm *dataStructures.ExternalTaskMetadata) {
			defer allMarkedWG.Done()

			if etm.State == dataStructures.ExternalTaskStateDoneOnboarding {
				_, _, err = m.metadata.SetInternalAndExternalTasksMetadata(ctx, etm.BillingAccount, func(ctx context.Context, oetm *dataStructures.ExternalTaskMetadata,
					oitm *dataStructures.InternalTaskMetadata) error {
					if oetm.State == dataStructures.ExternalTaskStateDoneOnboarding {
						logger.Infof("MARKING external and internal MD as NOT ONBOARDING")

						oetm.State = dataStructures.ExternalTaskStatePending
						oetm.OnBoarding = false
						oitm.OnBoarding = false
						oitm.State = dataStructures.InternalTaskStateInitializing
					}

					return nil
				})
			}
		}(etm)
	}

	allMarkedWG.Wait()
	logger.Infof("DONE WAITING for marking external and internal MD as NOT ONBOARDING")

	return nil
}

func (m *ExternalManager) waitUntilFilesAreDeleted(ctx context.Context, filesDeleted *sync.WaitGroup) error {
	logger := m.loggerProvider(ctx)
	logger.Infof("Waiting for all files to be deleted")

	doneCh := make(chan struct{}, 1)

	go func() {
		filesDeleted.Wait()
		doneCh <- struct{}{}
	}()

	select {
	case <-ctx.Done():
		err := fmt.Errorf("unable to finish deleting files. Context timeout")
		logger.Error(err)

		return err

	case <-doneCh:
		logger.Infof("FILES DELETED successfully")
		return nil
	}
}

func (m *ExternalManager) manageExternalTasks(ctx context.Context, emm *dataStructures.ExternalManagerMetadata, filesDeleted *sync.WaitGroup) (err error) {
	logger := m.loggerProvider(ctx)

	etms, err := m.metadata.GetActiveExternalTasksMetadata(ctx)
	if err != nil {
		//TODO handle error
		return err
	}

	var tasks sync.WaitGroup
	for _, etm := range etms {
		tasks.Add(1)

		go func(etm *dataStructures.ExternalTaskMetadata) error {
			BA := etm.BillingAccount

			customerBQ, err := m.customerBQClient.GetCustomerBQClient(ctx, etm.BillingAccount)
			if err != nil {
				logger.Errorf("unable to getCustomerBQClient. Caused by %s", err)
				return err
			}

			defer func() {
				tasks.Done()
				logger.Infof("DONE creating cloud-task for BA %s", BA)
				customerBQ.Close()
			}()

			etm, err = m.manageExternalTask(ctx, customerBQ, emm, etm, filesDeleted)
			if err != nil {
				//TODO handle error
				return err
			}

			return nil
		}(etm)
	}

	logger.Info("WAITING for cloud-tasks to be dispatched")
	tasks.Wait()
	logger.Info("DONE all cloud-tasks dispatched")

	return nil
}

func (m *ExternalManager) manageExternalTask(ctx context.Context, customerBQ *bigquery.Client, emm *dataStructures.ExternalManagerMetadata, etm *dataStructures.ExternalTaskMetadata, filesDeleted *sync.WaitGroup) (updatedetm *dataStructures.ExternalTaskMetadata, err error) {
	logger := m.loggerProvider(ctx)

	uetm, err := m.precalculateJobsIdsIfRequired(ctx, customerBQ, etm)
	if err != nil {
		err = fmt.Errorf("unable to updateExternalTask for BA %s. Caused by %s", etm.BillingAccount, err)
		logger.Error(err)

		return nil, err
	}

	uetm, err = m.updateExternalTask(ctx, customerBQ, emm, uetm, filesDeleted)
	if err != nil {
		err = fmt.Errorf("unable to updateExternalTask for BA %s. Caused by %s", etm.BillingAccount, err)
		logger.Error(err)

		return nil, err
	}

	etm = uetm

	err = m.createCloudTask(ctx, customerBQ, emm, uetm)
	if err != nil {
		err = fmt.Errorf("unable to createCloudTask for BA %s. Caused by %s", etm.BillingAccount, err)
		logger.Error(err)

		return nil, err
	}

	return uetm, nil
}

func (m *ExternalManager) createCloudTask(ctx context.Context, customerBQ *bigquery.Client, emm *dataStructures.ExternalManagerMetadata, etm *dataStructures.ExternalTaskMetadata) (err error) {
	logger := m.loggerProvider(ctx)
	logger.Infof("CREATING cloud-task for BA %s", etm.BillingAccount)

	if etm.State == dataStructures.ExternalTaskStateWaitingForToBucket {
		body := &dataStructures.UpdateRequestBody{
			BillingAccountID: etm.BillingAccount,
			Iteration:        etm.Iteration,
		}

		if common.IsLocalhost {
			restClient := resty.New()

			logger.Infof("sending task %+v", body)

			response, err := restClient.R().SetBody(body).Post(fmt.Sprintf("http://localhost:%s/tasks/flexsave-standalone/google-cloud/billing/external/tasks/to-bucket", os.Getenv("PORT")))
			if err != nil {
				logger.Errorf("unable to run task %+v. Caused by %s", body, err.Error())
			} else {
				logger.Infof("task %+v triggered. Details: %+v", body, response.RawResponse)
			}

			logger.Info(body)
		} else {
			config := common.CloudTaskConfig{
				Method: cloudtaskspb.HttpMethod_POST,
				Path:   "/tasks/flexsave-standalone/google-cloud/billing/external/tasks/to-bucket",
				Queue:  common.TaskQueueFlexSaveStandaloneExternalToBucketTasks,
			}

			task, err := m.CloudTaskClient.CreateTask(ctx, config.Config(body))
			if err != nil {
				logger.Errorf("unable to schedule task. Caused by %s", err)
				return err
			}

			logger.Infof("to-bucket task %s for BA %s scheduled", task.String(), etm.BillingAccount)

			return nil
		}

		return nil
	}

	if etm.State == dataStructures.ExternalTaskStateWaitingForFromBucket {
		body := &dataStructures.UpdateRequestBody{
			BillingAccountID: etm.BillingAccount,
			Iteration:        etm.Iteration,
		}

		if common.IsLocalhost {
			restClient := resty.New()

			logger.Infof("sending task %+v", body)

			response, err := restClient.R().SetBody(body).Post(fmt.Sprintf("http://localhost:%s/tasks/flexsave-standalone/google-cloud/billing/external/tasks/from-bucket", os.Getenv("PORT")))
			if err != nil {
				logger.Errorf("unable to run task %+v. Caused by %s", body, err.Error())
			} else {
				logger.Infof("task %+v triggered. Details: %+v", body, response.RawResponse)
			}

			logger.Info(body)
		} else {
			config := common.CloudTaskConfig{
				Method: cloudtaskspb.HttpMethod_POST,
				Path:   "/tasks/flexsave-standalone/google-cloud/billing/external/tasks/from-bucket",
				Queue:  common.TaskQueueFlexSaveStandaloneExternalFromBucketTasks,
			}

			task, err := m.CloudTaskClient.CreateTask(ctx, config.Config(body))
			if err != nil {
				logger.Errorf("unable to schedule task. Caused by %s", err)
				return err
			}

			logger.Infof("from-bucket task %s for BA %s scheduled", task.String(), etm.BillingAccount)

			return nil
		}

		return nil
	}

	return nil
}

func (m *ExternalManager) precalculateJobsIdsIfRequired(ctx context.Context, customerBQ *bigquery.Client, etm *dataStructures.ExternalTaskMetadata) (updatedetm *dataStructures.ExternalTaskMetadata, err error) {
	logger := m.loggerProvider(ctx)
	bq, err := m.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())

	if err != nil {
		logger.Errorf("unable to GetBQClientByProjectID. Caused by %s", err)
		return nil, err
	}

	if etm.ExternalTaskJobs.ToBucketJob.JobStatus == dataStructures.JobScheduleTimeout {
		err = m.precalculateJobsId(ctx, customerBQ, utils.GetToBucketJobPrefixCheck(etm.BillingAccount, etm.Iteration), etm.ExternalTaskJobs.ToBucketJob)
		if err != nil {
			err = fmt.Errorf("unable to precalculateJobsIdsIfRequired. Caused by %s", err)
			logger.Error(err)

			return nil, err
		}

		return m.metadata.SetExternalTaskMetadata(ctx, etm.BillingAccount, func(ctx context.Context, oetm *dataStructures.ExternalTaskMetadata) error {
			oetm.ExternalTaskJobs.ToBucketJob = etm.ExternalTaskJobs.ToBucketJob
			return nil
		})
	} else if etm.ExternalTaskJobs.FromBucketJob.JobStatus == dataStructures.JobScheduleTimeout {
		err = m.precalculateJobsId(ctx, bq, utils.GetFromBucketJobPrefixCheck(etm.BillingAccount, etm.Iteration), etm.ExternalTaskJobs.FromBucketJob)
		if err != nil {
			err = fmt.Errorf("unable to precalculateJobsIdsIfRequired. Caused by %s", err)
			logger.Error(err)

			return nil, err
		}

		return m.metadata.SetExternalTaskMetadata(ctx, etm.BillingAccount, func(ctx context.Context, oetm *dataStructures.ExternalTaskMetadata) error {
			oetm.ExternalTaskJobs.FromBucketJob = etm.ExternalTaskJobs.FromBucketJob
			return nil
		})
	}

	return etm, nil
}

func (m *ExternalManager) precalculateJobsId(ctx context.Context, bq *bigquery.Client, jobIdPrefix string, externalJob *dataStructures.Job) (err error) {
	logger := m.loggerProvider(ctx)
	timeB4 := time.Now()
	jobID, err := m.job.GetJobByPrefix(ctx, bq, jobIdPrefix)

	logger.Infof("GetJobByPrefix call took %s", time.Since(timeB4))

	if err != nil {
		err = fmt.Errorf("unable to GetJobByPrefix. Caused by %s", err)
		logger.Error(err)

		return err
	}

	if jobID != "" {
		externalJob.JobID = jobID
		externalJob.JobStatus = dataStructures.JobTimeout
	} else {
		externalJob.JobStatus = dataStructures.JobFailed
	}

	return nil
}

func (m *ExternalManager) updateExternalTask(ctx context.Context, customerBQ *bigquery.Client, emm *dataStructures.ExternalManagerMetadata, etm *dataStructures.ExternalTaskMetadata, filesDeleted *sync.WaitGroup) (updatedetm *dataStructures.ExternalTaskMetadata, err error) {
	logger := m.loggerProvider(ctx)
	logger.Infof("UPDATING external task for BA %s", etm.BillingAccount)

	bq, err := m.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		logger.Errorf("unable to GetBQClientByProjectID. Caused by %s", err)
		return nil, err
	}

	return m.metadata.SetExternalTaskMetadata(ctx, etm.BillingAccount, func(ctx context.Context, oetm *dataStructures.ExternalTaskMetadata) error {
		logger.Infof("UPDATING SetExternalTaskMetadata for BA %s", etm.BillingAccount)

		if etm.Iteration != oetm.Iteration {
			err := fmt.Errorf("invalid iteration # expected %d found %d", etm.Iteration, oetm.Iteration)
			logger.Error(err)

			return err
		}

		switch oetm.State {
		case dataStructures.ExternalTaskStatePending:
			oetm.Iteration = oetm.Iteration + 1
			if oetm.ExternalTaskJobs.FromBucketJob.JobStatus == dataStructures.JobDone && oetm.ExternalTaskJobs.ToBucketJob.JobStatus == dataStructures.JobDone {
				err = m.calculateNextSegment(ctx, customerBQ, oetm)
				if err != nil {
					switch err.(type) {
					case *billingCommon.EmptyBillingTableError:
						logger.Infof("SKIPPING process since table seems to be empty for BA %s", oetm.BillingAccount)
						return nil
					default:
						err := fmt.Errorf("unable to calculate next Segment for BA %s. Caused by %s", oetm.BillingAccount, err)
						logger.Error(err)

						return err
					}
				}

				if !oetm.Segment.StartTime.Before(*oetm.Segment.EndTime) {
					logger.Infof("SKIPPING process since no new lines were found for BA %s", oetm.BillingAccount)
					return nil
				}
			}

			go m.deleteBucketFiles(ctx, oetm.BillingAccount, oetm.Bucket, filesDeleted)
			oetm.Bucket = &dataStructures.BucketData{}
			oetm.ExternalTaskJobs.FromBucketJob = &dataStructures.Job{}
			toStart := time.Now().Add(consts.WaitForJobOnTaskMaxDuration)
			toEnd := time.Now().Add(utils.GetWaitForExternalJobToFinishMaxDuration(oetm.OnBoarding))

			oetm.ExternalTaskJobs.ToBucketJob = &dataStructures.Job{
				WaitToStartTimeout:  &toStart,
				WaitToFinishTimeout: &toEnd,
				JobStatus:           dataStructures.JobPending,
			}
			oetm.State = dataStructures.ExternalTaskStateWaitingForToBucket

			return nil
		case dataStructures.ExternalTaskStateWaitingForToBucket:
			//nil pointer check:
			err := utils.CheckToBucketJobForNilPointers(oetm.ExternalTaskJobs)
			if err != nil {
				return fmt.Errorf("MetaData error for %s, issue: %s for ToBucketTask", oetm.BillingAccount, err.Error())
			}

			if time.Now().After(*oetm.ExternalTaskJobs.ToBucketJob.WaitToStartTimeout) {
				logger.Errorf("timeout waiting for task to-bucket to start.")

				oetm.ExternalTaskJobs.ToBucketJob.JobStatus = dataStructures.JobScheduleTimeout
				oetm.State = dataStructures.ExternalTaskStateFailed

				return nil
			}
		case dataStructures.ExternalTaskStateToTaskScheduled:
			//nil pointer check:
			err := utils.CheckToBucketJobForNilPointers(oetm.ExternalTaskJobs)
			if err != nil {
				return fmt.Errorf("MetaData error for BA %s, issue: %s for ToBucketTask", oetm.BillingAccount, err.Error())
			}

			if oetm.ExternalTaskJobs.ToBucketJob.JobStatus == dataStructures.JobCreated {
				oetm.State = dataStructures.ExternalTaskStateToBucket
			} else if time.Now().After(*oetm.ExternalTaskJobs.ToBucketJob.WaitToStartTimeout) {
				logger.Errorf("timeout waiting for task of BA %s to notify jobID.", oetm.BillingAccount)
				oetm.State = dataStructures.ExternalTaskStateFailed
				oetm.ExternalTaskJobs.ToBucketJob.JobStatus = dataStructures.JobScheduleTimeout
			}
		case dataStructures.ExternalTaskStateToBucket:
			err = m.handleJob(ctx, customerBQ, oetm, oetm.ExternalTaskJobs.ToBucketJob, oetm.TableLocation)
			if err != nil {
				logger.Errorf("unable to handle ToBucket job %s for BA %s. Caused by %s", oetm.ExternalTaskJobs.ToBucketJob.JobID, oetm.BillingAccount, err)
				return err
			}

			if oetm.ExternalTaskJobs.ToBucketJob.JobStatus == dataStructures.JobDone {
				// send request
				toStart := time.Now().Add(consts.WaitForJobOnTaskMaxDuration)
				toEnd := time.Now().Add(utils.GetWaitForExternalJobToFinishMaxDuration(oetm.OnBoarding))
				oetm.ExternalTaskJobs.FromBucketJob = &dataStructures.Job{
					WaitToStartTimeout:  &toStart,
					WaitToFinishTimeout: &toEnd,
					JobStatus:           dataStructures.JobPending,
				}
				oetm.State = dataStructures.ExternalTaskStateWaitingForFromBucket
			}

			return nil
		case dataStructures.ExternalTaskStateWaitingForFromBucket:
			//nil pointer check:
			err := utils.CheckFromBucketJobForNilPointers(oetm.ExternalTaskJobs)
			if err != nil {
				return fmt.Errorf("MetaData error for %s, issue: %s for FromBucketTask", oetm.BillingAccount, err.Error())
			}

			if time.Now().After(*oetm.ExternalTaskJobs.FromBucketJob.WaitToStartTimeout) {
				logger.Errorf("timeout waiting for task from-bucket to start.")

				oetm.ExternalTaskJobs.FromBucketJob.JobStatus = dataStructures.JobScheduleTimeout
				oetm.State = dataStructures.ExternalTaskStateFailed

				return nil
			}
		case dataStructures.ExternalTaskStateFromTaskScheduled:
			//nil pointer check:
			err := utils.CheckFromBucketJobForNilPointers(oetm.ExternalTaskJobs)
			if err != nil {
				return fmt.Errorf("MetaData error for %s, issue: %s for FromBucketTask", oetm.BillingAccount, err.Error())
			}
			//check jobID or timeout or job created
			if oetm.ExternalTaskJobs.FromBucketJob.JobStatus == dataStructures.JobCreated {
				oetm.State = dataStructures.ExternalTaskStateFromBucket
				return nil
			}

			if time.Now().After(*oetm.ExternalTaskJobs.FromBucketJob.WaitToStartTimeout) {
				oetm.ExternalTaskJobs.FromBucketJob.JobStatus = dataStructures.JobScheduleTimeout
				oetm.State = dataStructures.ExternalTaskStateFailed
			}

			return nil

		case dataStructures.ExternalTaskStateFromBucket:
			//nil pointer check:
			err := utils.CheckFromBucketJobForNilPointers(oetm.ExternalTaskJobs)
			if err != nil {
				return fmt.Errorf("MetaData error for %s, issue: %s for FromBucketTask", oetm.BillingAccount, err.Error())
			}

			err = m.handleJob(ctx, bq, oetm, oetm.ExternalTaskJobs.FromBucketJob, consts.DoitLocation)
			if err != nil {
				logger.Errorf("unable to handle FromBucket job %s for BA %s. Caused by %s", oetm.ExternalTaskJobs.ToBucketJob.JobID, oetm.BillingAccount, err)
				return err
			}

			if oetm.ExternalTaskJobs.FromBucketJob.JobStatus == dataStructures.JobDone {
				if oetm.OnBoarding {
					oetm.State = dataStructures.ExternalTaskStateDoneOnboarding
				} else {
					oetm.State = dataStructures.ExternalTaskStatePending
				}
			}

		case dataStructures.ExternalTaskStateFailed:
			err = m.handleFailedFlow(ctx, emm, oetm, customerBQ)
			if err != nil {
				logger.Errorf("unable to handle failed status for BA %s. Caused by %s", oetm.BillingAccount, err)
				return err
			}
		}

		logger.Infof("DONE handling BA %s", etm.BillingAccount)

		return nil
	})
}

func (m *ExternalManager) handleFailedFlow(ctx context.Context, emm *dataStructures.ExternalManagerMetadata, etm *dataStructures.ExternalTaskMetadata, customerBq *bigquery.Client) (err error) {
	logger := m.loggerProvider(ctx)

	if etm.ExternalTaskJobs.ToBucketJob.JobStatus != dataStructures.JobDone {
		err = m.handleFailedJob(ctx, emm, customerBq, etm.TableLocation, etm.ExternalTaskJobs.ToBucketJob, utils.GetToBucketJobPrefixCheck(etm.BillingAccount, etm.Iteration), func() (bool, error) {
			return true, nil
		})
		if err != nil {
			err = fmt.Errorf("unable to handleFailedJob for job ToBucket for BA %s. Caused by %s", etm.BillingAccount, err)
			logger.Error(err)
			logger.Infof("recovering BA %s from error and retrying", etm.BillingAccount)
			etm.State = dataStructures.ExternalTaskStatePending

			return nil
		}

		if etm.ExternalTaskJobs.ToBucketJob.JobStatus == dataStructures.JobFailed {
			etm.State = dataStructures.ExternalTaskStatePending
		} else if etm.ExternalTaskJobs.ToBucketJob.JobStatus == dataStructures.JobDone {
			etm.State = dataStructures.ExternalTaskStateToBucket
		}
	} else if etm.ExternalTaskJobs.FromBucketJob.JobStatus != dataStructures.JobDone {
		bq, err := m.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
		if err != nil {
			err = fmt.Errorf("unable to GetBQClientByProjectID. Caused by %s", err)
			logger.Error(err)

			return err
		}

		//if etm.ExternalTaskJobs.FromBucketJob.JobID == "" {
		//	err = m.recoverMissingJob(ctx, etm.ExternalTaskJobs.FromBucketJob, bq, utils.GetFromBucketJobPrefix(etm.BillingAccount, etm.Iteration))
		//	if err != nil {
		//		err = fmt.Errorf("unable to recoverMissingJob for job ToBucket for BA %s. Caused by %s", etm.BillingAccount, err)
		//		logger.Error(err)
		//		return err
		//	}
		//}

		err = m.handleFailedJob(ctx, emm, bq, consts.DoitLocation, etm.ExternalTaskJobs.FromBucketJob, utils.GetFromBucketJobPrefixCheck(etm.BillingAccount, etm.Iteration), func() (bool, error) {
			//TODO add verification as follows:
			//CHeck if any row of segment is on local table
			//compare number of rows between customer and local table on segment.
			//if onboarding just delete everything
			return true, nil
		})
		if err != nil {
			err = fmt.Errorf("unable to recoverMissingJob for job FromBucket for BA %s. Caused by %s", etm.BillingAccount, err)
			logger.Error(err)

			return err
		}

		if etm.ExternalTaskJobs.FromBucketJob.JobStatus == dataStructures.JobFailed {
			etm.State = dataStructures.ExternalTaskStatePending
		} else if etm.ExternalTaskJobs.FromBucketJob.JobStatus == dataStructures.JobDone {
			etm.State = dataStructures.ExternalTaskStatePending
		}
	}

	return nil
}

func (m *ExternalManager) deleteBucketFiles(ctx context.Context, billingAccount string, bucket *dataStructures.BucketData, filesDeleted *sync.WaitGroup) {
	if bucket.LastBucketWriteTimestamp == 0 {
		m.loggerProvider(ctx).Info("Folder not found. SKIPPING files deletion...")
		return
	}

	filesDeleted.Add(1)

	defer filesDeleted.Done()

	err := m.bucket.DeleteFileFromBucket(ctx, bucket.BucketName, fmt.Sprint(bucket.LastBucketWriteTimestamp), billingAccount)
	if err != nil {
		m.loggerProvider(ctx).Errorf("unable to delete from folder %d. Caused by %s", bucket.LastBucketWriteTimestamp, err)
	} else {
		m.loggerProvider(ctx).Infof("successfully deleted folder %d", bucket.LastBucketWriteTimestamp)
	}
}

func (m *ExternalManager) handleFailedJob(ctx context.Context, emm *dataStructures.ExternalManagerMetadata, bq *bigquery.Client, jobLocation string, externalJob *dataStructures.Job, jobNamePrefix string, verificationFunc func() (bool, error)) (err error) {
	logger := m.loggerProvider(ctx)

	if time.Now().After(emm.TTL) {
		err = fmt.Errorf("unable to complete handleFailedJob on time. canceling")
		logger.Error(err)

		return err
	}

	switch externalJob.JobStatus {
	case dataStructures.JobScheduleTimeout:
		return nil
	case dataStructures.JobTimeout:
		err = m.job.CancelRunningJob(ctx, bq, externalJob.JobID)
		if err != nil {
			err = fmt.Errorf("unable to cancel job %s. Caused by %s", externalJob.JobID, err)
			logger.Error(err)

			return err
		}

		externalJob.JobStatus = dataStructures.JobCanceling

		return m.handleFailedJob(ctx, emm, bq, jobLocation, externalJob, jobNamePrefix, verificationFunc)
	case dataStructures.JobCanceling:
		jobStatus, err := m.job.GetJobStatus(ctx, bq, externalJob.JobID, jobLocation)
		if err != nil {
			err = fmt.Errorf("unable to get jobStatus for job %s. Caused by %s", externalJob.JobID, err)
			logger.Error(err)

			return err
		}

		if jobStatus.Done() {
			if jobStatus.Err() != nil {
				externalJob.JobStatus = dataStructures.JobCanceled
			} else {
				externalJob.JobStatus = dataStructures.JobDone
			}
		} else {
			logger.Infof("waiting for job %s to stop.", externalJob.JobID)
			time.Sleep(10 * time.Second)

			return m.handleFailedJob(ctx, emm, bq, jobLocation, externalJob, jobNamePrefix, verificationFunc)
		}
	case dataStructures.JobCanceled:
		//TODO implement with cancellation verification func
		verified, err := verificationFunc()
		if err != nil {
			err = fmt.Errorf("unable to get verify that for job %s didn't run. Caused by %s", externalJob.JobID, err)
			logger.Error(err)

			return err
		}

		if !verified {
			//TODO handle
		} else {
			externalJob.JobStatus = dataStructures.JobFailed
			return nil
		}

	case dataStructures.JobStuck:
		logger.Errorf("unable to cancel job %s. Retrying...", externalJob.JobID)
		externalJob.JobStatus = dataStructures.JobCanceling

		time.Sleep(time.Second * 10)

		return m.handleFailedJob(ctx, emm, bq, jobLocation, externalJob, jobNamePrefix, verificationFunc)
	}

	return nil
}

func (m *ExternalManager) handleJob(ctx context.Context, bq *bigquery.Client, etm *dataStructures.ExternalTaskMetadata, externalJob *dataStructures.Job, location string) (err error) {
	logger := m.loggerProvider(ctx)

	jobStatus, err := m.job.GetJobStatus(ctx, bq, externalJob.JobID, location)
	if err != nil {
		err = fmt.Errorf("unable to get job.Status for job %s. Caused by %s", externalJob.JobID, err)
		logger.Error(err)

		return err
	}

	if jobStatus.Done() {
		if jobStatus.Err() != nil {
			externalJob.JobStatus = dataStructures.JobFailed
			etm.State = dataStructures.ExternalTaskStateFailed
			logger.Errorf("unable to execute job %s for BA %s. Caused by %s. Caused by %s", externalJob.JobID, etm.BillingAccount, jobStatus.Err(), jobStatus.Errors)

			return nil
		} else {
			externalJob.JobStatus = dataStructures.JobDone
			return nil
		}
	}

	if time.Now().After(*externalJob.WaitToFinishTimeout) {
		externalJob.JobStatus = dataStructures.JobTimeout
		etm.State = dataStructures.ExternalTaskStateFailed

		return nil
	}

	return nil
}

func (m *ExternalManager) calculateNextSegment(ctx context.Context, customerBQ *bigquery.Client, etm *dataStructures.ExternalTaskMetadata) (err error) {
	logger := m.loggerProvider(ctx)

	if etm.Segment == nil {
		startingTime, err := m.tQuery.GetCustomersTableOldestRecordTime(ctx, customerBQ, etm.BQTable)
		if err != nil {
			logger.Errorf("unable to GetCustomersTableOldestRecordTime for BA %s. Caused by %s", etm.BillingAccount, err)
			return err
		}

		startingTime = startingTime.Add(-time.Second)
		etm.Segment = &dataStructures.Segment{
			EndTime: &startingTime,
		}
	}

	etm.Segment.StartTime = etm.Segment.EndTime
	latestTime, err := m.tQuery.GetCustomersTableNewestRecordTime(ctx, customerBQ, etm.BQTable)

	if err != nil {
		logger.Errorf("unable to GetCustomersTableNewestRecordTime for BA %s. Caused by %s", etm.BillingAccount, err)
		return err
	}

	etm.Segment.EndTime = &latestTime

	return nil
}

func (e *ExternalManager) initExternalManager(ctx context.Context) (emm *dataStructures.ExternalManagerMetadata, err error) {
	logger := e.loggerProvider(ctx)

	emm, err = e.metadata.CatchExternalManagerMetadata(ctx)
	if err != nil {
		logger.Errorf("unable to catch Enternal manager metadata. Caused by %s", err)
		return nil, err
	}

	if !emm.Running {
		err = fmt.Errorf("invalid state found. Skipping iteration")
		return nil, err
	}

	logger.Infof("############################## STARTING EXTERNAL ITERATION #%d ##############################", emm.Iteration)

	return emm, nil
}

func (m *ExternalManager) tearDown(ctx context.Context, emm *dataStructures.ExternalManagerMetadata) (err error) {
	logger := m.loggerProvider(ctx)

	_, err = m.metadata.SetExternalManagerMetadata(ctx, func(ctx context.Context, oemm *dataStructures.ExternalManagerMetadata) error {
		if emm.Iteration != oemm.Iteration {
			err = fmt.Errorf("invalid iteration number. Expected %d but found %d", emm.Iteration, oemm.Iteration)
			logger.Error(err)

			return err
		}

		oemm.Running = false

		return nil
	})
	if err != nil {
		err = fmt.Errorf("unable to SetExternalManagerMetadata. Caused by %s", err)
		logger.Error(err)

		return err
	}

	return nil
}
