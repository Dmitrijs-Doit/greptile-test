package service

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/iam/apiv1/iampb"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/bq_utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type CopyToBucketData struct {
	BucketName               string
	FileURI                  string
	Segment                  *dataStructures.Segment
	ServiceAccountEmail      string
	RunQueryData             *common.BQExecuteQueryData
	LastBucketWriteTimestamp int64
}

type CopyFromBucketData struct {
	FileURI          string
	DestinationTable *dataStructures.BillingTableInfo
	RunQueryData     *common.BQExecuteQueryData
}

type BillingDataCopierService struct {
	loggerProvider logger.Provider
	*connection.Connection
	config  *dal.PipelineConfigFirestore
	query   *dal.Query
	bqUtils *bq_utils.BQ_Utils
}

func NewBillingDataCopierService(log logger.Provider, conn *connection.Connection) *BillingDataCopierService {
	return &BillingDataCopierService{
		log,
		conn,
		dal.NewPipelineConfigWithClient(conn.Firestore),
		dal.NewQuery(log, conn),
		bq_utils.NewBQ_UTils(log, conn),
	}
}

func (s *BillingDataCopierService) CopyFromCustomerTableToBucket(ctx context.Context, bq *bigquery.Client, data *CopyToBucketData) (string, error) {
	logger := s.loggerProvider(ctx)

	if err := s.grantServiceAccountPermissionsOnBucket(ctx, data); err != nil {
		logger.Error(err)
		return "", err
	}

	var err error

	data.RunQueryData.Query, err = utils.GetExportDataToBucketQuery(data.RunQueryData.DefaultTable, data.FileURI, data.Segment)
	if err != nil {
		return "", err
	}

	data.RunQueryData.Export = true

	logger.Infof("Export query:\n%s", data.RunQueryData.Query)

	// SET ANY OTHER DATA TO data.RunQueryData IF NEEDED

	data.RunQueryData.Internal = false

	job, err := s.query.ExecQueryAsync(ctx, bq, data.RunQueryData)
	if err != nil {
		logger.Errorf("Error executing query: %v", err)
		return "", err
	}

	if data.RunQueryData.WaitTillDone {
		jobStatus, err := job.Wait(ctx)
		if err != nil {
			err = fmt.Errorf("unable to wait for job %s. Caused by %s", job.ID(), err)
			logger.Error(err)

			return "", err
		}

		err = jobStatus.Err()
		if err != nil {
			err = fmt.Errorf("job %s was unsuccessful. Caused by %s", job.ID(), err)
			logger.Error(err)

			return "", err
		}
	}

	return job.ID(), nil
}

func (s *BillingDataCopierService) grantServiceAccountPermissionsOnBucket(ctx context.Context, data *CopyToBucketData) error {
	logger := s.loggerProvider(ctx)
	bucket := s.CloudStorage(ctx).Bucket(data.BucketName)

	policy, err := bucket.IAM().V3().Policy(ctx)
	if err != nil {
		return err
	}

	roleGranted := false

	role := fmt.Sprintf("projects/%s/roles/%s", utils.GetProjectName(), consts.DedicatedRole)
	for _, p := range policy.Bindings {
		if p.Role == role {
			for _, member := range p.Members {
				if member == "serviceAccount:"+data.ServiceAccountEmail {
					logger.Infof("member %s found on role %s for BA %s", member, role, data.RunQueryData.BillingAccountID)

					roleGranted = true

					break
				}
			}

			break
		}
	}

	if !roleGranted {
		binding := &iampb.Binding{
			Role:    role,
			Members: []string{"serviceAccount:" + data.ServiceAccountEmail},
		}
		logger.Infof("binding %+v created for role %s for BA %s", binding, role, data.RunQueryData.BillingAccountID)

		policy.Bindings = append(policy.Bindings, binding)
		if err := bucket.IAM().V3().SetPolicy(ctx, policy); err != nil {
			err = fmt.Errorf("unable to set policy %+v for BA %s. Caused by %s", policy, data.RunQueryData.BillingAccountID, err)
			return err
		}
	}

	return nil
}

func (s *BillingDataCopierService) CopyFromBucketToTable(ctx context.Context, data *CopyFromBucketData) (string, error) {
	bq, err := s.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		return "", err
	}

	logger := s.loggerProvider(ctx)

	gcsRef := bigquery.NewGCSReference(data.FileURI)
	gcsRef.SourceFormat = bigquery.JSON
	gcsRef.Compression = bigquery.Gzip

	loader := bq.Dataset(consts.LocalBillingDataset).Table(utils.GetLocalCopyAccountTableName(data.RunQueryData.BillingAccountID)).LoaderFrom(gcsRef)
	loader.WriteDisposition = bigquery.WriteAppend
	loader.CreateDisposition = bigquery.CreateIfNeeded
	loader.JobIDConfig = bigquery.JobIDConfig{JobID: data.RunQueryData.ConfigJobID, AddJobIDSuffix: true}

	job, err := loader.Run(ctx)
	if err != nil {
		logger.Errorf("Error while running loader job: %v", err)
		return "", err
	}

	if data.RunQueryData.WaitTillDone {
		jobStatus, err := job.Wait(ctx)
		if err != nil {
			err = fmt.Errorf("unable to wait for job %s. Caused by %s", job.ID(), err)
			logger.Error(err)

			return "", err
		}

		err = jobStatus.Err()
		if err != nil {
			err = fmt.Errorf("job %s was unsuccessful. Caused by %s", job.ID(), err)
			logger.Error(err)

			return "", err
		}
	}

	return job.ID(), nil
}

func (s *BillingDataCopierService) CopyFromBucketToAlternativeTable(ctx context.Context, data *CopyFromBucketData) (string, error) {
	bq, err := s.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		return "", err
	}

	logger := s.loggerProvider(ctx)

	gcsRef := bigquery.NewGCSReference(data.FileURI)
	gcsRef.SourceFormat = bigquery.JSON
	gcsRef.Compression = bigquery.Gzip

	loader := bq.Dataset(consts.AlternativeLocalBillingDataset).Table(utils.GetAlternativeLocalCopyAccountTableName(data.RunQueryData.BillingAccountID)).LoaderFrom(gcsRef)
	loader.WriteDisposition = bigquery.WriteTruncate
	loader.CreateDisposition = bigquery.CreateIfNeeded

	//loader.JobIDConfig = bigquery.JobIDConfig{JobID: data.RunQueryData.ConfigJobID, AddJobIDSuffix: true}

	job, err := loader.Run(ctx)
	if err != nil {
		logger.Errorf("Error while running loader job: %v", err)
		return "", err
	}

	if data.RunQueryData.WaitTillDone {
		jobStatus, err := job.Wait(ctx)
		if err != nil {
			err = fmt.Errorf("unable to wait for job %s. Caused by %s", job.ID(), err)
			logger.Error(err)

			return "", err
		}

		err = jobStatus.Err()
		if err != nil {
			err = fmt.Errorf("job %s was unsuccessful. Caused by %s", job.ID(), err)
			logger.Error(err)

			return "", err
		}
	}

	return job.ID(), nil
}
