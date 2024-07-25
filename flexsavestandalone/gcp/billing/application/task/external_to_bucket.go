package task

import (
	"context"
	"fmt"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils"

	"cloud.google.com/go/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/service"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type ExternalToBucketTask struct {
	loggerProvider logger.Provider
	*connection.Connection
	metadata         service.Metadata
	dataCopier       *service.BillingDataCopierService
	customerBQClient service.ExternalBigQueryClient
	bucket           service.Bucket
	config           service.PipelineConfig
	bqTable          service.Table
}

func NewExternalToBucketTask(log logger.Provider, conn *connection.Connection) *ExternalToBucketTask {
	return &ExternalToBucketTask{
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

func (s *ExternalToBucketTask) RunExternalToBucketTask(ctx context.Context, rp *dataStructures.UpdateRequestBody) error {
	logger := s.loggerProvider(ctx)

	logger.Infof("+++++++ running toBucket task for BillingID # %s for iteration  %d +++++++", rp.BillingAccountID, rp.Iteration)

	//Get the task metadata, and check that the iteration is correct, set status to scehduleToBucket
	etm, err := s.initTask(ctx, rp)
	if err != nil {
		logger.Errorf("Error setting task metadata: %v", err)
		return err
	}

	//context will time out after TTL expires:
	ctx, cancel := context.WithTimeout(ctx, time.Until(*etm.ExternalTaskJobs.ToBucketJob.WaitToStartTimeout))
	defer cancel()

	//logger.Infof("-------- running to bucket task for BillingID # %s for iteration  %d ---------", etm.BillingAccount, etm.Iteration)

	customerBQ, err := s.customerBQClient.GetCustomerBQClient(ctx, etm.BillingAccount)
	if err != nil {
		logger.Errorf("Error getting customer BQ client: %v", err)
		return err
	}
	defer customerBQ.Close()

	dataToBucket := service.CopyToBucketData{
		Segment:             etm.Segment,
		ServiceAccountEmail: etm.ServiceAccountEmail,
		RunQueryData: &common.BQExecuteQueryData{
			BillingAccountID: etm.BillingAccount,
			DefaultTable:     etm.BQTable,
			DestinationTable: etm.BQTable,
			WriteDisposition: bigquery.WriteAppend,
			WaitTillDone:     false,
			Internal:         false,
			ConfigJobID:      utils.GetToBucketJobPrefix(etm.BillingAccount, etm.Iteration),
		},
	}

	//copy to bucket and save job id in an atomic operation, nonblocking (job needs to be created before TTL expires):
	jobID, err := s.copyToBucket(ctx, customerBQ, &dataToBucket, etm.BillingAccount)
	if err != nil {
		logger.Error(err)
		return err
	}
	//TODO Add job status created to metadata

	_, err = s.metadata.SetExternalTaskMetadata(ctx, etm.BillingAccount, func(ctx context.Context, oetm *dataStructures.ExternalTaskMetadata) error {
		oetm.ExternalTaskJobs.ToBucketJob.JobID = jobID
		oetm.ExternalTaskJobs.ToBucketJob.JobStatus = dataStructures.JobCreated
		oetm.State = dataStructures.ExternalTaskStateToTaskScheduled

		return nil
	})

	if err != nil {
		logger.Error(err)
		return err
	}

	logger.Infof("-------- writing to bucket for BillingID # %s for iteration  %d ---------", etm.BillingAccount, etm.Iteration)

	return nil
}

func (s *ExternalToBucketTask) initTask(ctx context.Context, rp *dataStructures.UpdateRequestBody) (etm *dataStructures.ExternalTaskMetadata, err error) {
	logger := s.loggerProvider(ctx)

	etm, err = s.metadata.SetExternalTaskMetadata(ctx, rp.BillingAccountID, func(ctx context.Context, oetm *dataStructures.ExternalTaskMetadata) error {
		if oetm.Iteration != rp.Iteration {
			err = fmt.Errorf("invalid iteration. Expected %d but found %d", rp.Iteration, oetm.Iteration)
			logger.Error(err)

			return err
		}

		if oetm.State != dataStructures.ExternalTaskStateWaitingForToBucket {
			err = fmt.Errorf("invalid state for BA %s. Expected %s but found %s", oetm.BillingAccount, dataStructures.ExternalTaskStateWaitingForToBucket, oetm.State)
			logger.Error(err)

			return err
		}

		if oetm.ExternalTaskJobs == nil || oetm.ExternalTaskJobs.ToBucketJob == nil || oetm.ExternalTaskJobs.ToBucketJob.WaitToStartTimeout == nil || oetm.ExternalTaskJobs.ToBucketJob.WaitToFinishTimeout == nil {
			err = fmt.Errorf("invalid MD state. MD found ExternalJobs %v. Terminating task", oetm.ExternalTaskJobs)
			logger.Error(err)

			return err
		}

		oetm.State = dataStructures.ExternalTaskStateToTaskScheduled

		return nil
	})
	if err != nil {
		err = fmt.Errorf("unable to SetExternalTaskMetadata for BA %s. Caused by %s", rp.BillingAccountID, err)
		logger.Error(err)

		return nil, err
	}

	logger.Infof("external task ToBucket for BA %s iteration %d started. Remaining time to run %v", etm.BillingAccount, etm.Iteration, etm.ExternalTaskJobs.ToBucketJob.WaitToStartTimeout.Sub(time.Now()))

	return etm, nil
}

func (s *ExternalToBucketTask) copyToBucket(ctx context.Context, bq *bigquery.Client, data *service.CopyToBucketData, billingID string) (string, error) {
	logger := s.loggerProvider(ctx)

	location, err := s.bqTable.GetTableLocation(ctx, bq, data.RunQueryData.DefaultTable)
	if err != nil {
		logger.Errorf("Error getting table location: %v", err)
		return "", err
	}

	bucketName, err := s.config.GetRegionBucket(ctx, location)
	if err != nil {
		if err != common.ErrBucketNotFound {
			return "", err
		}

		bucketName, err = s.bucket.Create(ctx, location, false)
		if err != nil {
			return "", err
		}

		if err := s.config.SetRegionBucket(ctx, location, bucketName); err != nil {
			return "", err
		}
	}

	data.BucketName = bucketName

	lastBucketWriteTimestamp := time.Now().Unix()

	data.FileURI = fmt.Sprintf("gs://%s/%s/%s/*.gzip", data.BucketName, data.RunQueryData.BillingAccountID, fmt.Sprint(lastBucketWriteTimestamp))

	data.LastBucketWriteTimestamp = lastBucketWriteTimestamp

	_, err = s.metadata.SetExternalTaskMetadata(ctx, billingID, func(ctx context.Context, oetm *dataStructures.ExternalTaskMetadata) error {
		oetm.Bucket = &dataStructures.BucketData{
			BucketName:               data.BucketName,
			LastBucketWriteTimestamp: data.LastBucketWriteTimestamp,
			FileURI:                  data.FileURI,
		}

		return nil
	})

	if err != nil {
		logger.Errorf("Error saving bucket info: %v", err)
		return "", err
	}

	jobID, err := s.dataCopier.CopyFromCustomerTableToBucket(ctx, bq, data)
	if err != nil {
		return "", err
	}

	logger.Infof("job %s created for BA %s.", jobID, billingID)

	return jobID, nil
}
