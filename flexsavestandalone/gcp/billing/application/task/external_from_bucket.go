package task

import (
	"context"
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils"

	//"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/service"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type ExternalFromBucketTask struct {
	loggerProvider logger.Provider
	*connection.Connection
	metadata         service.Metadata
	dataCopier       *service.BillingDataCopierService
	customerBQClient service.ExternalBigQueryClient
	bucket           service.Bucket
	config           service.PipelineConfig
	bqTable          service.Table
}

func NewExternalFromBucketBillingUpdateTask(log logger.Provider, conn *connection.Connection) *ExternalFromBucketTask {
	return &ExternalFromBucketTask{
		log,
		conn,
		service.NewMetadata(log, conn),
		service.NewBillingDataCopierService(log, conn),
		service.NewExternalBigQueryClient(log, conn),
		service.NewBucket(log, conn),
		service.NewPipelineConfig(log, conn),
		service.NewTable(log, conn),
	}
}

func (s *ExternalFromBucketTask) RunFromBucketExternalTask(ctx context.Context, rp *dataStructures.UpdateRequestBody) error {
	logger := s.loggerProvider(ctx)

	logger.Infof("+++++++ running fromBucket task for BillingID # %s for iteration  %d +++++++", rp.BillingAccountID, rp.Iteration)

	etm, err := s.initTask(ctx, rp)
	if err != nil {
		logger.Errorf("error while setting metadata: %s", err)
		return err
	}

	//context will time out after TTL expires:
	ctx, cancel := context.WithTimeout(ctx, time.Until(*etm.ExternalTaskJobs.FromBucketJob.WaitToStartTimeout))
	defer cancel()

	//logger.Infof("-------- running from bucket task for BillingID # %s for iteration  %d ---------", etm.BillingAccount, etm.Iteration)

	dataFromBucket := service.CopyFromBucketData{
		FileURI: etm.Bucket.FileURI,
		RunQueryData: &common.BQExecuteQueryData{
			BillingAccountID: etm.BillingAccount,
			WriteDisposition: bigquery.WriteAppend,
			WaitTillDone:     false,
			Internal:         false,
			ConfigJobID:      utils.GetFromBucketJobPrefix(etm.BillingAccount, etm.Iteration),
		},
	}

	jobID, err := s.copyFromBucket(ctx, &dataFromBucket)
	if err != nil {
		return err
	}

	etm, err = s.metadata.SetExternalTaskMetadata(ctx, etm.BillingAccount, func(ctx context.Context, oetm *dataStructures.ExternalTaskMetadata) error {
		oetm.ExternalTaskJobs.FromBucketJob.JobID = jobID
		oetm.ExternalTaskJobs.FromBucketJob.JobStatus = dataStructures.JobCreated
		oetm.State = dataStructures.ExternalTaskStateFromTaskScheduled

		return nil
	})
	if err != nil {
		logger.Errorf("error while setting JobID: %s", err)
		return err
	}

	logger.Infof("------- copying from bucket to table for BillingID # %s for iteration  %d -------", etm.BillingAccount, etm.Iteration)

	return nil
}

func (s *ExternalFromBucketTask) initTask(ctx context.Context, rp *dataStructures.UpdateRequestBody) (etm *dataStructures.ExternalTaskMetadata, err error) {
	logger := s.loggerProvider(ctx)

	etm, err = s.metadata.SetExternalTaskMetadata(ctx, rp.BillingAccountID, func(ctx context.Context, oetm *dataStructures.ExternalTaskMetadata) error {
		if oetm.Iteration != rp.Iteration {
			err = fmt.Errorf("invalid iteration. Expected %d but found %d", rp.Iteration, oetm.Iteration)
			logger.Error(err)

			return err
		}

		if oetm.State != dataStructures.ExternalTaskStateWaitingForFromBucket {
			err = fmt.Errorf("invalid state for BA %s. Expected %s but found %s", oetm.BillingAccount, dataStructures.ExternalTaskStateWaitingForFromBucket, oetm.State)
			logger.Error(err)

			return err
		}

		if oetm.ExternalTaskJobs == nil || oetm.ExternalTaskJobs.FromBucketJob == nil || oetm.ExternalTaskJobs.FromBucketJob.WaitToStartTimeout == nil || oetm.ExternalTaskJobs.FromBucketJob.WaitToFinishTimeout == nil {
			err = fmt.Errorf("invalid MD state. MD found ExternalJobs %v. Terminating task", oetm.ExternalTaskJobs)
			logger.Error(err)

			return err
		}

		oetm.State = dataStructures.ExternalTaskStateFromTaskScheduled

		return nil
	})
	if err != nil {
		err = fmt.Errorf("unable to SetExternalTaskMetadata for BA %s. Caused by %s", rp.BillingAccountID, err)
		logger.Error(err)

		return nil, err
	}

	logger.Infof("external task FromBucket for BA %s iteration %d started. Remaining time to run %v", etm.BillingAccount, etm.Iteration, etm.ExternalTaskJobs.FromBucketJob.WaitToStartTimeout.Sub(time.Now()))

	return etm, nil
}

func (s *ExternalFromBucketTask) copyFromBucket(ctx context.Context, data *service.CopyFromBucketData) (string, error) {
	logger := s.loggerProvider(ctx)
	jobID, err := s.dataCopier.CopyFromBucketToTable(ctx, data)

	if err != nil {
		logger.Errorf("error while copying from bucket to table: %s", err)
		return "", err
	}

	logger.Infof("job %s created for BA %s.", jobID, data.RunQueryData.BillingAccountID)

	return jobID, err
}
