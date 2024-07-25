package application

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"

	googleCloudConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/service"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/bq_utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	sharedDal "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/shared/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func NewAlternativeManager(log logger.Provider, conn *connection.Connection) *AlternativeManager {
	return &AlternativeManager{
		loggerProvider:   log,
		Connection:       conn,
		metadata:         service.NewMetadata(log, conn),
		table:            service.NewTable(log, conn),
		assets:           service.NewAssets(log, conn),
		tQuery:           service.NewTableQuery(log, conn),
		bqUtils:          bq_utils.NewBQ_UTils(log, conn),
		job:              service.NewJob(log, conn),
		dataCopier:       service.NewBillingDataCopierService(log, conn),
		importStatus:     sharedDal.NewBillingImportStatusWithClient(conn.Firestore(context.Background())),
		billingEvent:     sharedDal.NewBillingUpdateFirestoreWithClient(func(ctx context.Context) *firestore.Client { return conn.Firestore(context.Background()) }),
		customerBQClient: service.NewExternalBigQueryClient(log, conn),
		bqTable:          service.NewTable(log, conn),
		config:           service.NewPipelineConfig(log, conn),
		bucket:           service.NewBucket(log, conn),
		dataset:          service.NewDataset(log, conn),
	}
}

type AlternativeManager struct {
	loggerProvider logger.Provider
	*connection.Connection
	metadata         service.Metadata
	table            service.Table
	assets           *service.Assets
	tQuery           service.TableQuery
	bqUtils          *bq_utils.BQ_Utils
	job              *service.Job
	dataCopier       *service.BillingDataCopierService
	importStatus     sharedDal.BillingImportStatus
	billingEvent     *sharedDal.BillingUpdateFirestore
	customerBQClient service.ExternalBigQueryClient
	bqTable          service.Table
	config           service.PipelineConfig
	bucket           service.Bucket
	dataset          *service.Dataset
}

func (a *AlternativeManager) RunAlternativeManager(ctx context.Context) (err error) {
	logger := a.loggerProvider(ctx)
	startDate := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -35)
	endDate := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, 1)
	segment := &dataStructures.Segment{
		StartTime: &startDate,
		EndTime:   &endDate,
	}

	bq, err := a.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		err = fmt.Errorf("unable to GetBQClientByProjectID. Cause by %s", err)
		logger.Error(err)

		return err
	}

	err = a.dataset.CreateAlternativeUnifiedDataset(ctx)
	if err != nil {
		err = fmt.Errorf("unable to CreateAlternativeUnifiedDataset. Cause by %s", err)
		logger.Error(err)

		return err
	}

	err = a.dataset.CreateAlternativeLocalDataset(ctx)
	if err != nil {
		err = fmt.Errorf("unable to CreateAlternativeLocalDataset. Cause by %s", err)
		logger.Error(err)

		return err
	}

	err = a.table.DeleteAlternativeTmpTable(ctx, bq)
	if err != nil {
		err = fmt.Errorf("unable to DeleteAlternativeTmpTable. Cause by %s", err)
		logger.Error(err)

		return err
	}

	err = a.table.CreateAlternativeTmpTable(ctx)
	if err != nil {
		err = fmt.Errorf("unable to CreateAlternativeTmpTable. Cause by %s", err)
		logger.Error(err)

		return err
	}

	err = a.table.CreateAlternativeUnifiedTable(ctx)
	if err != nil {
		err = fmt.Errorf("unable to CreateAlternativeUnifiedTable. Cause by %s", err)
		logger.Error(err)

		return err
	}

	etms, err := a.metadata.GetExternalTasksMetadata(ctx)
	if err != nil {
		err = fmt.Errorf("unable to GetExternalTasksMetadata. Cause by %s", err)
		logger.Error(err)

		return err
	}

	for _, etm := range etms {
		//update alternative local tables
		err = a.table.CreateAlternativeLocalTable(ctx, etm.BillingAccount)
		if err != nil {
			err = fmt.Errorf("unable to CreateAlternativeLocalTable. Cause by %s", err)
			logger.Error(err)

			return err
		}

		path, err := a.RunExternalToBucketTask(ctx, etm.BillingAccount, segment)
		if err != nil {
			err = fmt.Errorf("unable to RunExternalToBucketTask. Cause by %s", err)
			logger.Error(err)

			return err
		}

		err = a.RunFromBucketExternalTask(ctx, etm.BillingAccount, path)
		if err != nil {
			err = fmt.Errorf("unable to RunExternalToBucketTask. Cause by %s", err)
			logger.Error(err)

			return err
		}
	}

	itms, err := a.metadata.GetInternalTasksMetadata(ctx)
	if err != nil {
		err = fmt.Errorf("unable to GetInternalTasksMetadata. Cause by %s", err)
		logger.Error(err)

		return err
	}

	for _, itm := range itms {
		if itm.BillingAccount == googleCloudConsts.MasterBillingAccount {
			continue
		}

		err = a.RunInternalTask(ctx, itm.BillingAccount)
		if err != nil {
			err = fmt.Errorf("unable to RunInternalTask. Cause by %s", err)
			logger.Error(err)

			return err
		}
	}

	currDate := startDate

	for !currDate.Equal(endDate) {
		logger.Infof("copying partition %s", currDate)

		err = a.copyPartitionFromTmpTable(ctx, bq, &currDate)
		if err != nil {
			err = fmt.Errorf("unable to copyPartitionFromTmpTable. Cause by %s", err)
			logger.Error(err)

			return err
		}

		currDate = currDate.AddDate(0, 0, 1)
	}

	return nil
}

func (s *AlternativeManager) RunExternalToBucketTask(ctx context.Context, billingAccount string, segment *dataStructures.Segment) (path string, err error) {
	logger := s.loggerProvider(ctx)

	etm, err := s.metadata.GetExternalTaskMetadata(ctx, billingAccount)
	if err != nil {
		err = fmt.Errorf("unable to GetExternalTasksMetadata. Cause by %s", err)
		logger.Error(err)

		return "", err
	}

	customerBQ, err := s.customerBQClient.GetCustomerBQClient(ctx, etm.BillingAccount)
	if err != nil {
		err = fmt.Errorf("unable to GetCustomerBQClient. Cause by %s", err)
		logger.Error(err)

		return "", err
	}
	defer customerBQ.Close()

	dataToBucket := service.CopyToBucketData{
		Segment:             segment,
		ServiceAccountEmail: etm.ServiceAccountEmail,
		RunQueryData: &common.BQExecuteQueryData{
			BillingAccountID: etm.BillingAccount,
			DefaultTable:     etm.BQTable,
			DestinationTable: etm.BQTable,
			WriteDisposition: bigquery.WriteTruncate,
			WaitTillDone:     true,
			Internal:         false,
		},
	}

	path, err = s.copyToBucket(ctx, customerBQ, &dataToBucket, etm.BillingAccount)
	if err != nil {
		err = fmt.Errorf("unable to copyToBucket. Cause by %s", err)
		logger.Error(err)

		return "", err
	}

	return path, nil
}

func (s *AlternativeManager) copyToBucket(ctx context.Context, bq *bigquery.Client, data *service.CopyToBucketData, billingID string) (path string, err error) {
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

	data.FileURI = fmt.Sprintf("gs://%s/alternative/%s/%s/*.gzip", data.BucketName, data.RunQueryData.BillingAccountID, fmt.Sprint(lastBucketWriteTimestamp))

	data.LastBucketWriteTimestamp = lastBucketWriteTimestamp

	if err != nil {
		logger.Errorf("Error saving bucket info: %v", err)
		return "", err
	}

	jobID, err := s.dataCopier.CopyFromCustomerTableToBucket(ctx, bq, data)
	if err != nil {
		return "", err
	}

	logger.Infof("job %s created for BA %s.", jobID, billingID)

	return data.FileURI, nil
}

func (s *AlternativeManager) RunFromBucketExternalTask(ctx context.Context, billingAccount string, path string) error {
	logger := s.loggerProvider(ctx)

	dataFromBucket := service.CopyFromBucketData{
		FileURI: path,
		RunQueryData: &common.BQExecuteQueryData{
			BillingAccountID: billingAccount,
			WriteDisposition: bigquery.WriteTruncate,
			WaitTillDone:     true,
			Internal:         false,
		},
	}

	jobID, err := s.copyFromBucket(ctx, &dataFromBucket)
	if err != nil {
		return err
	}

	logger.Infof("job %s created for BA %s.", jobID, billingAccount)

	return nil
}

func (s *AlternativeManager) copyFromBucket(ctx context.Context, data *service.CopyFromBucketData) (string, error) {
	logger := s.loggerProvider(ctx)
	jobID, err := s.dataCopier.CopyFromBucketToAlternativeTable(ctx, data)

	if err != nil {
		logger.Errorf("error while copying from bucket to table: %s", err)
		return "", err
	}

	logger.Infof("job %s created for BA %s.", jobID, data.RunQueryData.BillingAccountID)

	return jobID, err
}

func (s *AlternativeManager) RunInternalTask(ctx context.Context, billingAccount string) (err error) {
	logger := s.loggerProvider(ctx)

	itm, err := s.metadata.GetInternalTaskMetadata(ctx, billingAccount)
	if err != nil {
		err = fmt.Errorf("unable to GetInternalTaskMetadata. Cause by %s", err)
		logger.Error(err)

		return err
	}

	job, err := s.tQuery.CopyFromAlternativeLocalToTmpTable(ctx, itm)
	if err != nil {
		logger.Errorf("unable to CopyFromLocalToTmpTable for BA %s. Caused by %s", itm.BillingAccount, err)
		return err
	}

	logger.Infof("job %s created for BA %s.", job.ID(), itm.BillingAccount)

	jobStatus, err := job.Wait(ctx)
	if err != nil {
		err = fmt.Errorf("unable to wait for job %s. Caused by %s", job.ID(), err)
		logger.Error(err)

		return err
	}

	err = jobStatus.Err()
	if err != nil {
		err = fmt.Errorf("job %s was unsuccessful. Caused by %s", job.ID(), err)
		logger.Error(err)

		return err
	}

	return err
}

func (s *AlternativeManager) copyPartitionFromTmpTable(ctx context.Context, bq *bigquery.Client, currDate *time.Time) error {
	logger := s.loggerProvider(ctx)
	currPartition := currDate.Format(consts.Partition2FieldFormat)

	srcTableTargetPartion := fmt.Sprintf("%s$%s", consts.UnifiedAlternativeRawBillingTable, currPartition)
	dstTableTargetPartion := fmt.Sprintf("%s$%s", consts.UnifiedAlternativeGCPRawTable, currPartition)
	copier := bq.Dataset(consts.AlternativeUnifiedGCPBillingDataset).Table(dstTableTargetPartion).CopierFrom(bq.Dataset(consts.AlternativeLocalBillingDataset).Table(srcTableTargetPartion))
	copier.WriteDisposition = bigquery.WriteTruncate

	job, err := copier.Run(ctx)
	if err != nil {
		logger.Error(err)
		return err
	}

	status, err := job.Wait(ctx)
	if err != nil {
		logger.Error(err)
		return err
	}

	if err := status.Err(); err != nil {
		logger.Error(err)
		return err
	}

	return nil
}
