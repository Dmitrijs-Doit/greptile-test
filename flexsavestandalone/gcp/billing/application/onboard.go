package application

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	googleCloudConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/service"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/bq_utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/shared"
	sharedDal "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/shared/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type onboardingStep string

const (
	onboardingStepInternalTaskMetadata              onboardingStep = "internal task metadata"
	onboardingStepExternalTaskMetadata              onboardingStep = "external task metadata"
	onboardingStepCustomerBQClient                  onboardingStep = "customer BQ client"
	onboardingStepGetTableLocation                  onboardingStep = "get table location"
	onboardingStepGetCustomersTableLatestRecordTime onboardingStep = "get customer's table latest record time"
	onboardingStepGetCustomersTableOldestRecordTime onboardingStep = "get customer's table oldest record time"
	onboardingStepCreateMetadataForNewBillingID     onboardingStep = "create metadata for new billing id"
	onboardingStepFindOrCreateBucket                onboardingStep = "find or create bucket"
	onboardingStepCreateLocalTable                  onboardingStep = "create local table"
	onboardingStepNotifyStarted                     onboardingStep = "notify started"
	onboardingStepCompleted                         onboardingStep = "completed"
)

type Onboarding struct {
	loggerProvider logger.Provider
	*connection.Connection
	metadata         service.Metadata
	table            service.Table
	assets           *service.Assets
	bqTable          service.Table
	config           service.PipelineConfig
	bucket           service.Bucket
	customerBQClient service.ExternalBigQueryClient
	bqUtils          *bq_utils.BQ_Utils
	dataset          *service.Dataset
	tQuery           service.TableQuery
	importStatus     sharedDal.BillingImportStatus
}

func NewOnboarding(log logger.Provider, conn *connection.Connection) *Onboarding {
	return &Onboarding{
		loggerProvider:   log,
		Connection:       conn,
		metadata:         service.NewMetadata(log, conn),
		table:            service.NewTable(log, conn),
		assets:           service.NewAssets(log, conn),
		bqTable:          service.NewTable(log, conn),
		config:           service.NewPipelineConfig(log, conn),
		bucket:           service.NewBucket(log, conn),
		customerBQClient: service.NewExternalBigQueryClient(log, conn),
		bqUtils:          bq_utils.NewBQ_UTils(log, conn),
		dataset:          service.NewDataset(log, conn),
		tQuery:           service.NewTableQuery(log, conn),
		importStatus:     sharedDal.NewBillingImportStatusWithClient(conn.Firestore(context.Background())),
	}
}

func (o *Onboarding) Onboard(ctx context.Context, requestParams *dataStructures.OnboardingRequestBody) error {
	var err error

	var step onboardingStep

	logger := o.loggerProvider(ctx)

	defer func() {
		e := err
		s := step
		o.rollback(ctx, requestParams, s, e)
	}()

	step = onboardingStepInternalTaskMetadata

	logger.Infof("requestParams %+v", requestParams)

	itm, err := o.metadata.GetInternalTaskMetadata(ctx, requestParams.BillingAccountID)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			err = fmt.Errorf("unable to GetInternalTaskMetadata. Caused by %s", err)
			logger.Error(err)

			return err
		}
	}

	if itm != nil {
		err = fmt.Errorf("unable to Onboard BA %s. Caused by external task %s already exists", requestParams.BillingAccountID, requestParams.BillingAccountID)
		logger.Error(err)

		return err
	}

	step = onboardingStepExternalTaskMetadata

	etm, err := o.metadata.GetExternalTaskMetadata(ctx, requestParams.BillingAccountID)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			err = fmt.Errorf("unable to GetExternalTaskMetadata. Caused by %s", err)
			logger.Error(err)

			return err
		}
	}

	if etm != nil {
		err = fmt.Errorf("unable to Onboard BA %s. Caused by external task %s already exists", requestParams.BillingAccountID, requestParams.BillingAccountID)
		logger.Error(err)

		return err
	}

	step = onboardingStepCustomerBQClient

	customerBQ, err := o.customerBQClient.GetCustomerBQClientWithParams(ctx, requestParams.ServiceAccountEmail, requestParams.ProjectID)
	if err != nil {
		err = fmt.Errorf("unable to GetCustomerBQClient. Caused by %s", err)
		logger.Error(err)

		return err
	}

	t := &dataStructures.BillingTableInfo{
		ProjectID: requestParams.ProjectID,
		DatasetID: requestParams.Dataset,
		TableID:   requestParams.TableID,
	}

	step = onboardingStepGetTableLocation
	location, err := o.bqTable.GetTableLocation(ctx, customerBQ, t)

	if err != nil {
		err := fmt.Errorf("unable to get location for BA %s. Caused by %s", requestParams.BillingAccountID, err)
		logger.Error(err)

		return err
	}

	step = onboardingStepGetCustomersTableLatestRecordTime

	latestTime, err := o.tQuery.GetCustomersTableNewestRecordTime(ctx, customerBQ, t)
	if err != nil {
		switch err.(type) {
		case *common.EmptyBillingTableError:
			logger.Infof("table seems to be empty. Setting the latestTime to now")

			latestTime = time.Now()
		default:
			err := fmt.Errorf("unable to GetCustomersTableNewestRecordTime for BA %s. Caused by %s", requestParams.BillingAccountID, err)
			logger.Error(err)

			return err
		}
	}

	step = onboardingStepGetCustomersTableOldestRecordTime

	oldestTime, err := o.tQuery.GetCustomersTableOldestRecordTime(ctx, customerBQ, t)
	if err != nil {
		switch err.(type) {
		case *common.EmptyBillingTableError:
			logger.Infof("table seems to be empty. Setting the oldestTime to now")

			oldestTime = time.Now()
		default:
			err := fmt.Errorf("unable to GetCustomersTableOldestRecordTime for BA %s. Caused by %s", requestParams.BillingAccountID, err)
			logger.Error(err)

			return err
		}
	}

	step = onboardingStepCreateMetadataForNewBillingID

	err = o.metadata.CreateMetadataForNewBillingID(ctx, requestParams, location, &latestTime, &oldestTime)
	if err != nil {
		err := fmt.Errorf("unable to CreateMetadataForNewBillingID for BA %s. Caused by %s", requestParams.BillingAccountID, err)
		logger.Error(err)

		return err
	}

	step = onboardingStepFindOrCreateBucket

	err = o.FindOrCreateBucket(ctx, requestParams)
	if err != nil {
		err := fmt.Errorf("unable to FindOrCreateBucket for BA %s with location %s. Caused by %s", requestParams.BillingAccountID, location, err)
		logger.Error(err)

		return err
	}

	step = onboardingStepCreateLocalTable

	err = o.table.CreateLocalTable(ctx, requestParams.BillingAccountID)
	if err != nil {
		err := fmt.Errorf("unable to CreateLocalTableTable for BA %s with location %s. Caused by %s", requestParams.BillingAccountID, location, err)
		logger.Error(err)

		return err
	}

	step = onboardingStepNotifyStarted

	err = o.notifyStarted(ctx, requestParams)
	if err != nil {
		err := fmt.Errorf("unable to notifyStarted for BA %s. Caused by %s", requestParams.BillingAccountID, err)
		logger.Error(err)

		return err
	}

	step = onboardingStepCompleted

	return nil
}

func (o *Onboarding) notifyStarted(ctx context.Context, requestParams *dataStructures.OnboardingRequestBody) error {
	logger := o.loggerProvider(ctx)
	//TODO skip in case of dummy user
	if utils.IsDummy(requestParams.Dataset) {
		logger.Infof("skipping notification for dummy account.")
		return nil
	}

	is, err := o.importStatus.GetBillingImportStatus(ctx, requestParams.CustomerID, requestParams.BillingAccountID)
	if err != nil {
		err = fmt.Errorf("unable to GetBillingImportStatus. Caused by %s", err)
		logger.Error(err)

		return err
	}

	if is.Status != shared.BillingImportStatusPending {
		err = fmt.Errorf("invalid billingImportStatus found on BA %s. Expected %s but found %s", requestParams.BillingAccountID, shared.BillingImportStatusPending, is.Status)
		logger.Error(err)

		return err
	}

	err = o.importStatus.SetStatusStarted(ctx, requestParams.CustomerID, requestParams.BillingAccountID)
	if err != nil {
		err = fmt.Errorf("unable to SetStatusStarted. Caused by %s", err)
		logger.Error(err)

		return err
	}

	logger.Infof("NOTIFIED SetStatusStarted for BA %s", requestParams.BillingAccountID)

	return nil
}

func (o *Onboarding) RemoveBilling(ctx context.Context, requestParams *dataStructures.DeleteBillingRequestBody) (err error) {
	logger := o.loggerProvider(ctx)
	billingAccount := requestParams.BillingAccountID

	errorHandler := func(err error) error {
		//TODO handle error
		return err
	}

	ctx, cancelF, err := utils.SetupContext(ctx, logger, fmt.Sprintf(consts.CtxDeleteBATemplate, billingAccount))
	defer cancelF()

	if err != nil {
		return errorHandler(err)
	}

	if billingAccount == googleCloudConsts.MasterBillingAccount {
		_, err = o.metadata.SetInternalTaskMetadata(ctx, billingAccount, func(ctx context.Context,
			oitm *dataStructures.InternalTaskMetadata) error {
			if !oitm.OnBoarding {
				oitm.LifeCycleStage = dataStructures.LifeCycleStageDeprecated
			}

			return nil
		})
	} else {
		_, _, err = o.metadata.SetInternalAndExternalTasksMetadata(ctx, billingAccount, func(ctx context.Context, oetm *dataStructures.ExternalTaskMetadata,
			oitm *dataStructures.InternalTaskMetadata) error {
			if !oetm.OnBoarding && !oitm.OnBoarding {
				oetm.LifeCycleStage = dataStructures.LifeCycleStageDeprecated
				oitm.LifeCycleStage = dataStructures.LifeCycleStageDeprecated
			}

			return nil
		})
	}

	return err
}

// Restore the whole system to initial state
func (o *Onboarding) RemoveAll(ctx context.Context) (err error) {
	logger := o.loggerProvider(ctx)

	errorHandler := func(err error) error {
		//TODO handle error
		return err
	}

	ctx, cancelF, err := utils.SetupContext(ctx, logger, consts.CtxDeleteAllTemplate)
	defer cancelF()

	if err != nil {
		return errorHandler(err)
	}

	err = o.deleteInternalMetadata(ctx)
	if err != nil {
		//TODO handle error
		logger.Errorf("unable to deleteInternalMetadata. Caused by %s", err)
		return err
	}

	err = o.deleteExternalMetadata(ctx)
	if err != nil {
		//TODO handle error
		logger.Errorf("unable to deleteExternalMetadata. Caused by %s", err)
		return err
	}

	err = o.deleteRowValidatorMetadata(ctx)
	if err != nil {
		//TODO handle error
		logger.Errorf("unable to deleteRowValidatorMetadata. Caused by %s", err)
		return err
	}

	rawBillingTargetTime, err := o.tQuery.GetRawBillingNewestRecordTime(ctx)
	if err != nil {
		//TODO handle error
		logger.Errorf("unable to GetRawBillingNewestRecordTime. Caused by %s", err)
		return err
	}

	rawBillingOldestTime, err := o.tQuery.GetRawBillingOldestRecordTime(ctx)
	if err != nil {
		logger.Errorf("unable to GetRawBillingOldestRecordTime. Caused by %s", err)
		return err
	}

	err = o.metadata.CreateAllMetadata(ctx, &rawBillingTargetTime, &rawBillingOldestTime)
	if err != nil {
		//TODO handle error
		logger.Errorf("unable to CreateAllMetadata. Caused by %s", err)
		return err
	}

	err = o.config.CreatePipelineConfigDoc(ctx, &dataStructures.PipelineConfig{
		TemplateBillingDataProjectID: consts.BillingProjectProd,
		TemplateBillingDataDatasetID: consts.ResellRawBillingDataset,
		TemplateBillingDataTableID:   consts.ResellRawBillingTable,
	})

	if err != nil {
		//TODO handle error
		logger.Errorf("unable to CreatePipelineConfigDoc. Caused by %s", err)
		return err
	}

	err = o.dataset.CreateUnifiedDataset(ctx)
	if err != nil {
		//TODO handle error
		logger.Errorf("unable to CreateUnifiedDataset. Caused by %s", err)
		return err
	}

	err = o.dataset.CreateLocalDataset(ctx)
	if err != nil {
		//TODO handle error
		logger.Errorf("unable to CreateLocalDataset. Caused by %s", err)
		return err
	}

	err = o.tQuery.TruncateUnifiedTableContent(ctx)
	if err != nil {
		//TODO handle error
		logger.Errorf("unable to TruncateUnifiedTableContent. Caused by %s", err)
		return err
	}

	return nil
}

func (m *Onboarding) deleteInternalMetadata(ctx context.Context) (err error) {
	//TODO make a way to know which project we're using
	logger := m.loggerProvider(ctx)
	bq, err := m.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())

	if err != nil {
		logger.Errorf("unable to GetBQClientByProjectID for project %s. Caused by %s", utils.GetProjectName(), err)
		return err
	}

	err = m.table.DeleteAnyTmpTable(ctx, bq)
	if err != nil {
		//TODO handle error
		logger.Errorf("unable to DeleteTmpTable. Caused by %s", err)
		return err
	}

	err = m.table.DeleteUnifiedTable(ctx)
	if err != nil {
		//TODO handle error
		logger.Errorf("unable to DeleteUnifiedTable. Caused by %s", err)
		return err
	}

	err = m.metadata.DeleteInternalManagerMetadata(ctx)
	if err != nil {
		//TODO handle error
		logger.Errorf("unable to DeleteInternalManagerMetadata. Caused by %s", err)
		return err
	}

	err = m.metadata.DeleteInternalTasksMetadata(ctx)
	if err != nil {
		//TODO handle error
		logger.Errorf("unable to DeleteInternalTasksMetadata. Caused by %s", err)
		return err
	}

	logger.Info("DELETED InternalMetadata successfully")

	return nil
}

func (m *Onboarding) deleteExternalMetadata(ctx context.Context) (err error) {
	logger := m.loggerProvider(ctx)
	errorHandler := func(err error) error {
		//TODO handle error
		return err
	}

	bq, err := m.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		logger.Errorf("unable to GetBQClientByProjectID. Caused by %s", err)
		return errorHandler(err)
	}

	err = m.table.DeleteAllLocalTables(ctx, bq)
	if err != nil {
		logger.Errorf("unable to DeleteAllLocalTables. Caused by %s", err)
		return errorHandler(err)
	}

	err = m.metadata.DeleteExternalTasksMetadata(ctx)
	if err != nil {
		//TODO handle error
		logger.Errorf("unable to DeleteExternalTasksMetadata. Caused by %s", err)
		return err
	}

	err = m.bucket.DeleteAllBuckets(ctx)
	if err != nil {
		logger.Errorf("unable to DeleteAllBuckets. Caused by %s", err)
		return errorHandler(err)
	}

	err = m.metadata.DeleteExternalManagerMetadata(ctx)
	if err != nil {
		//TODO handle error
		return err
	}

	err = m.config.DeletePipelineConfigDoc(ctx)
	if err != nil {
		return errorHandler(err)
	}

	return nil
}

func (o *Onboarding) deleteRowValidatorMetadata(ctx context.Context) error {
	logger := o.loggerProvider(ctx)

	err := o.metadata.DeleteValidatorMetadata(ctx)
	if err != nil {
		err = fmt.Errorf("unable to DeleteValidatorMetadata. Caused by %s", err)
		logger.Error(err)
	}

	return err
}

func (o *Onboarding) FindOrCreateBucket(ctx context.Context, requestParams *dataStructures.OnboardingRequestBody) (err error) {
	logger := o.loggerProvider(ctx)
	BQTable := dataStructures.BillingTableInfo{
		TableID:   requestParams.TableID,
		DatasetID: requestParams.Dataset,
		ProjectID: requestParams.ProjectID,
	}
	billingAccountID := requestParams.BillingAccountID
	serviceAccountEmail := requestParams.ServiceAccountEmail

	customerBQ, err := o.customerBQClient.GetCustomerBQClient(ctx, billingAccountID)
	if err != nil {
		return err
	}

	defer customerBQ.Close()

	location, err := o.bqTable.GetTableLocation(ctx, customerBQ, &BQTable)
	if err != nil {
		return err
	}

	_, err = o.config.GetRegionBucket(ctx, location)
	if err != nil {
		if err != common.ErrBucketNotFound {
			err = fmt.Errorf("unable to GetRegionBucket to BA %s. Caused by %s", billingAccountID, err)
			logger.Error(err)

			return err
		}

		bucketName, err := o.bucket.Create(ctx, location, false)
		if err != nil {
			err = fmt.Errorf("unable to bucket.Create to BA %s. Caused by %s", billingAccountID, err)
			logger.Error(err)

			return err
		}

		if err := o.config.SetRegionBucket(ctx, location, bucketName); err != nil {
			err = fmt.Errorf("unable to SetRegionBucket to BA %s. Caused by %s", billingAccountID, err)
			logger.Error(err)

			return err
		}

		err = o.bucket.GrantServiceAccountPermissionsOnBucket(ctx, bucketName, serviceAccountEmail, billingAccountID)
		if err != nil {
			err = fmt.Errorf("unable to GrantServiceAccountPermissionsOnBucket to BA %s. Caused by %s", billingAccountID, err)
			logger.Error(err)

			return err
		}
	}

	return nil
}
