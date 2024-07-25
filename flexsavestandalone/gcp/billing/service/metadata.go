package service

import (
	"context"
	"fmt"
	"time"

	googleCloudConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/retry"
)

type Metadata interface {
	CreateAllMetadata(ctx context.Context, rawBillingTargetTime *time.Time, rawBillingOldestTime *time.Time) (err error)
	CatchInternalManagerMetadata(ctx context.Context) (mm *dataStructures.InternalManagerMetadata, err error)
	CatchExternalManagerMetadata(ctx context.Context) (emm *dataStructures.ExternalManagerMetadata, err error)
	CreateMetadataForNewBillingID(ctx context.Context, requestParams *dataStructures.OnboardingRequestBody, location string, historyCopyTargetTime *time.Time, oldestRecordTime *time.Time) (err error)
	SetInternalManagerMetadata(ctx context.Context, updateF func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error) (updatedInternalMetadata *dataStructures.InternalManagerMetadata, err error)
	GetInternalManagerMetadata(ctx context.Context) (internalManagerMetadata *dataStructures.InternalManagerMetadata, err error)
	DeleteInternalManagerMetadata(ctx context.Context) (err error)
	SetInternalTasksMetadata(ctx context.Context, updateFunc func(ctx context.Context, internalTaskMetadata *dataStructures.InternalTaskMetadata) error) error
	MarkAllCurrentInternalTasksAsFailed(ctx context.Context, iteration int64) error
	MarkAllInternalVerifiedTasksAsDone(ctx context.Context) error
	SetInternalTaskMetadata(ctx context.Context, billingAccount string, updateFunc func(ctx context.Context, taskMetadata *dataStructures.InternalTaskMetadata) error) (updatedInternalMetadata *dataStructures.InternalTaskMetadata, err error)
	DeleteInternalTasksMetadata(ctx context.Context) error
	GetInternalTasksMetadata(ctx context.Context) (internalTasksMetadata []*dataStructures.InternalTaskMetadata, err error)
	GetCreatedInternalTasksMetadata(ctx context.Context) ([]*dataStructures.InternalTaskMetadata, error)
	GetActiveInternalTasksMetadata(ctx context.Context) ([]*dataStructures.InternalTaskMetadata, error)
	GetDeprecatedInternalTasksMetadata(ctx context.Context) ([]*dataStructures.InternalTaskMetadata, error)
	GetAllInternalTasksMetadataByParams(ctx context.Context, iteration int64, possibleStates []dataStructures.InternalTaskState) (internalTasksMetadata []*dataStructures.InternalTaskMetadata, err error)
	DeleteInternalTaskMetadata(ctx context.Context, billingID string) error
	GetInternalTaskMetadata(ctx context.Context, billingID string) (*dataStructures.InternalTaskMetadata, error)
	DeleteExternalManagerMetadata(ctx context.Context) (err error)
	SetExternalManagerMetadata(ctx context.Context, updateF func(ctx context.Context, emm *dataStructures.ExternalManagerMetadata) error) (uemm *dataStructures.ExternalManagerMetadata, err error)
	SetExternalTaskMetadata(ctx context.Context, billingAccount string, updateF func(ctx context.Context, etm *dataStructures.ExternalTaskMetadata) error) (uetm *dataStructures.ExternalTaskMetadata, err error)
	SetInternalAndExternalTasksMetadata(ctx context.Context, billingAccount string,
		updateFunc func(ctx context.Context, oetm *dataStructures.ExternalTaskMetadata,
			oitm *dataStructures.InternalTaskMetadata) error) (*dataStructures.InternalTaskMetadata, *dataStructures.ExternalTaskMetadata, error)
	DeleteExternalTasksMetadata(ctx context.Context) error
	DeleteExternalTaskMetadata(ctx context.Context, billingID string) error
	GetExternalTasksMetadata(ctx context.Context) ([]*dataStructures.ExternalTaskMetadata, error)
	GetCreatedExternalTasksMetadata(ctx context.Context) ([]*dataStructures.ExternalTaskMetadata, error)
	GetActiveExternalTasksMetadata(ctx context.Context) ([]*dataStructures.ExternalTaskMetadata, error)
	GetDeprecatedExternalTasksMetadata(ctx context.Context) ([]*dataStructures.ExternalTaskMetadata, error)
	GetExternalTaskMetadata(ctx context.Context, billingID string) (*dataStructures.ExternalTaskMetadata, error)
	GetRowsValidatorMetadata(ctx context.Context, billingAccoutnID string) (*dataStructures.RowsValidatorMetadata, error)
	SetRowsValidatorMetadata(ctx context.Context, billingAccoutnID string, md *dataStructures.RowsValidatorMetadata) error
	DeleteValidatorMetadata(ctx context.Context) error
}

type MetadataImpl struct {
	loggerProvider logger.Provider
	*connection.Connection
	metadataDal      *dal.Metadata
	rowsValidatorDal *dal.RowsValidatorFirestore
}

func NewMetadata(log logger.Provider, conn *connection.Connection) *MetadataImpl {
	return &MetadataImpl{
		loggerProvider:   log,
		Connection:       conn,
		metadataDal:      dal.NewMetadata(log, conn),
		rowsValidatorDal: dal.NewRowsValidatorWithClient(conn.Firestore),
	}
}

func (m *MetadataImpl) CreateAllMetadata(ctx context.Context, rawBillingTargetTime *time.Time, rawBillingOldestTime *time.Time) (err error) {
	err = retryFunctionWrapper(func() error {
		return m.metadataDal.CreateInternalManagerMetadata(ctx, createDefaultInternalManagerMetadata())
	})
	if err != nil {
		return err
	}

	//creates the internal metadata to import the master billing data
	//err = retryFunctionWrapper(func() error {
	//	return m.metadataDal.CreateInternalTasksMetadata(ctx, createDefaultInternalTasksMetadata(rawBillingTargetTime, rawBillingOldestTime))
	//})
	//if err != nil {
	//	return err
	//}

	err = m.metadataDal.CreateExternalManagerMetadata(ctx, createDefaultExternalManagerMetadata())
	if err != nil {
		return err
	}

	return nil
}

// Internal Manager

func (m *MetadataImpl) CatchInternalManagerMetadata(ctx context.Context) (mm *dataStructures.InternalManagerMetadata, err error) {
	ttl := time.Now().Add(consts.InternalManagerMaxDuration)

	referenceImm, err := m.GetInternalManagerMetadata(ctx)
	if err != nil {
		return nil, err
	}

	maxIteration := referenceImm.Iteration + 1
	stateCheckFunc := func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error {
		if mm.Iteration >= maxIteration {
			return fmt.Errorf("iterations are out of sync")
		}

		if mm.State == dataStructures.InternalManagerStateDone {
			mm.TTL = &ttl
			mm.State = dataStructures.InternalManagerStateStarted
			mm.Iteration = maxIteration
			mm.CopyToUnifiedTableJob = &dataStructures.Job{JobStatus: dataStructures.JobPending}
			mm.Recovery.Recovering = false

			return nil
		} else if mm.State == dataStructures.InternalManagerStateFailed {
			mm.TTL = &ttl
			mm.State = dataStructures.InternalManagerStateStarted
			mm.Iteration = maxIteration
			mm.CopyToUnifiedTableJob = &dataStructures.Job{JobStatus: dataStructures.JobPending}
			mm.Recovery.Recovering = false

			return nil
		} else {
			return fmt.Errorf("invalid state %s found for InternalManagerMetadata iteration %d", mm.State, mm.Iteration)
		}
	}

	mm, err = m.SetInternalManagerMetadata(ctx, stateCheckFunc)

	return mm, err
}

func (m *MetadataImpl) CatchExternalManagerMetadata(ctx context.Context) (emm *dataStructures.ExternalManagerMetadata, err error) {
	return m.metadataDal.SetExternalManagerMetadata(ctx, func(ctx context.Context, oemm *dataStructures.ExternalManagerMetadata) error {
		if !oemm.Running {
			oemm.Running = true
			oemm.TTL = time.Now().Add(consts.ExternalManagerMaxDuration)
			oemm.Iteration = oemm.Iteration + 1

			return nil
		} else if time.Now().After(oemm.TTL) {
			oemm.Running = false
			return nil
		}

		return fmt.Errorf("Seems there is an execution already running for another %s", time.Until(oemm.TTL))
	})
}

func (m *MetadataImpl) CreateMetadataForNewBillingID(ctx context.Context, requestParams *dataStructures.OnboardingRequestBody, location string, historyCopyTargetTime *time.Time, oldestRecordTime *time.Time) (err error) {
	truncatedOldestRecordTime := oldestRecordTime.Add(-time.Hour * 12)

	err = m.metadataDal.CreateExternalTaskMetadata(ctx, &dataStructures.ExternalTaskMetadata{
		CustomerID:          requestParams.CustomerID,
		BillingAccount:      requestParams.BillingAccountID,
		State:               dataStructures.ExternalTaskStatePending,
		OnBoarding:          true,
		TableLocation:       location,
		ServiceAccountEmail: requestParams.ServiceAccountEmail,
		LifeCycleStage:      dataStructures.LifeCycleStageCreated,
		BQTable: &dataStructures.BillingTableInfo{
			ProjectID:       requestParams.ProjectID,
			DatasetID:       requestParams.Dataset,
			TableID:         requestParams.TableID,
			OldestPartition: &truncatedOldestRecordTime,
		},
		Bucket: &dataStructures.BucketData{},
		ExternalTaskJobs: &dataStructures.ExternalTaskJobs{
			ToBucketJob: &dataStructures.Job{
				JobStatus: dataStructures.JobDone,
			},
			FromBucketJob: &dataStructures.Job{
				JobStatus: dataStructures.JobDone,
			},
		},
	})
	if err != nil {
		//TODO handle error
		return err
	}

	err = retryFunctionWrapper(func() error {
		return m.metadataDal.CreateInternalTaskMetadata(ctx, createDefaultInternalTaskMetadata(requestParams, historyCopyTargetTime, &truncatedOldestRecordTime))
	})

	return err
}

func (m *MetadataImpl) SetInternalManagerMetadata(ctx context.Context, updateF func(ctx context.Context, mm *dataStructures.InternalManagerMetadata) error) (updatedInternalMetadata *dataStructures.InternalManagerMetadata, err error) {
	refItm, err := m.GetInternalManagerMetadata(ctx)
	if err != nil {
		return nil, err
	}

	err = updateF(ctx, refItm)
	if err != nil {
		return nil, err
	}

	err = retryFunctionWrapper(func() error {
		updatedInternalMetadata, err = m.metadataDal.SetInternalManagerMetadata(ctx, updateF)
		return err
	})

	return updatedInternalMetadata, err
}

func (m *MetadataImpl) GetInternalManagerMetadata(ctx context.Context) (internalManagerMetadata *dataStructures.InternalManagerMetadata, err error) {
	err = retryFunctionWrapper(func() error {
		internalManagerMetadata, err = m.metadataDal.GetInternalManagerMetadata(ctx)
		return err
	})

	return internalManagerMetadata, err
}

func (m *MetadataImpl) DeleteInternalManagerMetadata(ctx context.Context) (err error) {
	err = retryFunctionWrapper(func() error {
		return m.metadataDal.DeleteInternalManagerMetadata(ctx)
	})

	return err
}

// Internal Tasks

func (m *MetadataImpl) SetInternalTasksMetadata(ctx context.Context, updateFunc func(ctx context.Context, internalTaskMetadata *dataStructures.InternalTaskMetadata) error) error {
	internalTasks, err := m.GetInternalTasksMetadata(ctx)
	if err != nil {
		return err
	}

	for _, task := range internalTasks {
		err = updateFunc(ctx, task)
		if err != nil {
			return err
		}
	}

	err = retryFunctionWrapper(func() error {
		return m.metadataDal.SetInternalTasksMetadata(ctx, updateFunc)
	})

	return err
}

func (m *MetadataImpl) MarkAllCurrentInternalTasksAsFailed(ctx context.Context, iteration int64) error {
	logger := m.loggerProvider(ctx)

	itms, err := m.GetAllInternalTasksMetadataByParams(ctx, iteration, []dataStructures.InternalTaskState{dataStructures.InternalTaskStateVerified})
	if err != nil {
		err = fmt.Errorf("unable to GetInternalTasksMetadata. Caused by %s", err)
		logger.Error(err)

		return err
	}

	for _, itm := range itms {
		_, err = m.SetInternalTaskMetadata(ctx, itm.BillingAccount, func(ctx context.Context, itm *dataStructures.InternalTaskMetadata) error {
			if itm.State != dataStructures.InternalTaskStateOnboarding && itm.State != dataStructures.InternalTaskStateInitializing && itm.State != dataStructures.InternalTaskStateSkipped {
				itm.State = dataStructures.InternalTaskStateFailed
			}

			return nil
		})
		if err != nil {
			err = fmt.Errorf("unable to SetInternalTaskMetadata. Caused by %s", err)
			logger.Error(err)

			return err
		}
	}

	return nil
}

func (m *MetadataImpl) MarkAllInternalVerifiedTasksAsDone(ctx context.Context) error {
	return m.SetInternalTasksMetadata(ctx, func(ctx context.Context, itm *dataStructures.InternalTaskMetadata) error {
		if itm.State == dataStructures.InternalTaskStateNotified {
			itm.State = dataStructures.InternalTaskStateDone
		}

		return nil
	})
}

func (m *MetadataImpl) SetInternalTaskMetadata(ctx context.Context, billingAccount string, updateFunc func(ctx context.Context, managerMetadata *dataStructures.InternalTaskMetadata) error) (updatedInternalMetadata *dataStructures.InternalTaskMetadata, err error) {
	testItm, err := m.GetInternalTaskMetadata(ctx, billingAccount)
	if err != nil {
		return nil, err
	}

	err = updateFunc(ctx, testItm)
	if err != nil {
		return nil, err
	}

	err = retryFunctionWrapper(func() error {
		updatedInternalMetadata, err = m.metadataDal.SetInternalTaskMetadata(ctx, billingAccount, updateFunc)
		return err
	})

	return updatedInternalMetadata, err
}

func (m *MetadataImpl) DeleteInternalTasksMetadata(ctx context.Context) error {
	err := retryFunctionWrapper(func() error {
		return m.metadataDal.DeleteAllInternalTasksMetadata(ctx)
	})

	return err
}

func (m *MetadataImpl) GetInternalTasksMetadata(ctx context.Context) (internalTasksMetadata []*dataStructures.InternalTaskMetadata, err error) {
	err = retryFunctionWrapper(func() error {
		internalTasksMetadata, err = m.metadataDal.GetAllInternalTasksMetadata(ctx)
		return err
	})

	return internalTasksMetadata, err
}

func (s *MetadataImpl) GetCreatedInternalTasksMetadata(ctx context.Context) ([]*dataStructures.InternalTaskMetadata, error) {
	var md []*dataStructures.InternalTaskMetadata

	var err error
	err = retryFunctionWrapper(func() error {
		md, err = s.metadataDal.GetInternalTasksMetadataByLifeCycleState(ctx, dataStructures.LifeCycleStageCreated)
		return err
	})

	return md, err
}

func (s *MetadataImpl) GetActiveInternalTasksMetadata(ctx context.Context) ([]*dataStructures.InternalTaskMetadata, error) {
	var md []*dataStructures.InternalTaskMetadata

	var err error
	err = retryFunctionWrapper(func() error {
		md, err = s.metadataDal.GetInternalTasksMetadataByLifeCycleState(ctx, dataStructures.LifeCycleStageActive)
		return err
	})

	return md, err
}

func (s *MetadataImpl) GetDeprecatedInternalTasksMetadata(ctx context.Context) ([]*dataStructures.InternalTaskMetadata, error) {
	var md []*dataStructures.InternalTaskMetadata

	var err error
	err = retryFunctionWrapper(func() error {
		md, err = s.metadataDal.GetInternalTasksMetadataByLifeCycleState(ctx, dataStructures.LifeCycleStageDeprecated)
		return err
	})

	return md, err
}

func (m *MetadataImpl) GetAllInternalTasksMetadataByParams(ctx context.Context, iteration int64, possibleStates []dataStructures.InternalTaskState) (internalTasksMetadata []*dataStructures.InternalTaskMetadata, err error) {
	err = retryFunctionWrapper(func() error {
		internalTasksMetadata, err = m.metadataDal.GetAllInternalTasksMetadataByParams(ctx, iteration, possibleStates)
		return err
	})

	return internalTasksMetadata, err
}

func (m *MetadataImpl) DeleteInternalTaskMetadata(ctx context.Context, billingID string) error {
	err := retryFunctionWrapper(func() error {
		return m.metadataDal.DeleteInternalTaskMetadata(ctx, billingID)
	})

	return err
}

func (m *MetadataImpl) GetInternalTaskMetadata(ctx context.Context, billingID string) (*dataStructures.InternalTaskMetadata, error) {
	var md *dataStructures.InternalTaskMetadata

	var err error
	err = retryFunctionWrapper(func() error {
		md, err = m.metadataDal.GetInternalTaskMetadata(ctx, billingID)
		return err
	})

	return md, err
}

// External Manager

func (m *MetadataImpl) DeleteExternalManagerMetadata(ctx context.Context) (err error) {
	return m.metadataDal.DeleteExternalManagerMetadata(ctx)
}

func (m *MetadataImpl) SetExternalManagerMetadata(ctx context.Context, updateF func(ctx context.Context, emm *dataStructures.ExternalManagerMetadata) error) (uemm *dataStructures.ExternalManagerMetadata, err error) {
	return m.metadataDal.SetExternalManagerMetadata(ctx, updateF)
}

func (m *MetadataImpl) SetExternalTaskMetadata(ctx context.Context, billingAccount string, updateF func(ctx context.Context, etm *dataStructures.ExternalTaskMetadata) error) (uetm *dataStructures.ExternalTaskMetadata, err error) {
	m.loggerProvider(ctx).Infof("SetExternalTaskMetadata for BA %s", billingAccount)
	return m.metadataDal.SetExternalTaskMetadata(ctx, billingAccount, updateF)
}

func (m *MetadataImpl) SetInternalAndExternalTasksMetadata(ctx context.Context, billingAccount string,
	updateFunc func(ctx context.Context, oetm *dataStructures.ExternalTaskMetadata,
		oitm *dataStructures.InternalTaskMetadata) error) (*dataStructures.InternalTaskMetadata, *dataStructures.ExternalTaskMetadata, error) {
	return m.metadataDal.SetInternalAndExternalTasksMetadata(ctx, billingAccount, updateFunc)
}

func (m *MetadataImpl) DeleteExternalTasksMetadata(ctx context.Context) error {
	return m.metadataDal.DeleteAllExternalTasksMetadata(ctx)
}

func (m *MetadataImpl) DeleteExternalTaskMetadata(ctx context.Context, billingID string) error {
	return m.metadataDal.DeleteExternalTaskMetadata(ctx, billingID)
}

// External Tasks

func (s *MetadataImpl) GetExternalTasksMetadata(ctx context.Context) ([]*dataStructures.ExternalTaskMetadata, error) {
	md, err := s.metadataDal.GetExternalTasksMetadata(ctx)
	return md, err
}

func (s *MetadataImpl) GetCreatedExternalTasksMetadata(ctx context.Context) ([]*dataStructures.ExternalTaskMetadata, error) {
	md, err := s.metadataDal.GetExternalTasksMetadataByLifeCycleState(ctx, dataStructures.LifeCycleStageCreated)
	return md, err
}

func (s *MetadataImpl) GetActiveExternalTasksMetadata(ctx context.Context) ([]*dataStructures.ExternalTaskMetadata, error) {
	md, err := s.metadataDal.GetExternalTasksMetadataByLifeCycleState(ctx, dataStructures.LifeCycleStageActive)
	return md, err
}

func (s *MetadataImpl) GetDeprecatedExternalTasksMetadata(ctx context.Context) ([]*dataStructures.ExternalTaskMetadata, error) {
	md, err := s.metadataDal.GetExternalTasksMetadataByLifeCycleState(ctx, dataStructures.LifeCycleStageDeprecated)
	return md, err
}

func (s *MetadataImpl) GetExternalTaskMetadata(ctx context.Context, billingID string) (*dataStructures.ExternalTaskMetadata, error) {
	md, err := s.metadataDal.GetExternalTaskMetadata(ctx, billingID)
	return md, err
}

// Validator

func (s *MetadataImpl) GetRowsValidatorMetadata(ctx context.Context, billingAccoutnID string) (*dataStructures.RowsValidatorMetadata, error) {
	md, err := s.rowsValidatorDal.GetRowsValidatorMetadata(ctx, billingAccoutnID)
	return md, err
}

func (s *MetadataImpl) SetRowsValidatorMetadata(ctx context.Context, billingAccoutnID string, md *dataStructures.RowsValidatorMetadata) error {
	return s.rowsValidatorDal.SetRowsValidatorMetadata(ctx, billingAccoutnID, md)
}

func (s *MetadataImpl) DeleteValidatorMetadata(ctx context.Context) error {
	return s.rowsValidatorDal.DeleteDocsRef(ctx)
}

func createDefaultExternalManagerMetadata() *dataStructures.ExternalManagerMetadata {
	return &dataStructures.ExternalManagerMetadata{
		Iteration: 0,
		TTL:       time.Time{},
	}
}

func createDefaultInternalManagerMetadata() *dataStructures.InternalManagerMetadata {
	return &dataStructures.InternalManagerMetadata{
		Iteration:             0,
		TTL:                   nil,
		State:                 dataStructures.InternalManagerStateDone,
		CopyToUnifiedTableJob: &dataStructures.Job{},
		Recovery:              &dataStructures.InternalRecovery{Recovering: false},
	}
}

func createDefaultInternalTasksMetadata(rawBillingTargetTime *time.Time, rawBillingOldestTime *time.Time) (internalTasksMetadata []*dataStructures.InternalTaskMetadata) {
	internalTasksMetadata = append(internalTasksMetadata, &dataStructures.InternalTaskMetadata{
		BillingAccount: googleCloudConsts.MasterBillingAccount,
		State:          dataStructures.InternalTaskStateInitializing,
		Iteration:      0,
		TTL:            &time.Time{},
		Segment:        nil,
		LifeCycleStage: dataStructures.LifeCycleStageActive,
		InternalTaskJobs: &dataStructures.InternalTaskJobs{
			FromLocalTableToTmpTable: &dataStructures.Job{},
			//MarkAsVerified:           &dataStructures.Job{},
		},
		BQTable: &dataStructures.BillingTableInfo{
			TableID:         consts.ResellRawBillingTable,
			DatasetID:       consts.ResellRawBillingDataset,
			ProjectID:       consts.BillingProjectProd,
			OldestPartition: rawBillingOldestTime,
		},
		CopyHistory: &dataStructures.CopyHistoryData{
			TargetTime: rawBillingTargetTime,
			Status:     dataStructures.CopyHistoryStatusPending,
		},
	})

	return internalTasksMetadata
}

func createDefaultInternalTaskMetadata(requestParams *dataStructures.OnboardingRequestBody, historyCopyTargetTime *time.Time, truncatedOldestRecordTime *time.Time) *dataStructures.InternalTaskMetadata {
	truncatedTargetTime := historyCopyTargetTime.Add(-time.Hour * 12)

	return &dataStructures.InternalTaskMetadata{
		BillingAccount:   requestParams.BillingAccountID,
		State:            dataStructures.InternalTaskStateOnboarding,
		Iteration:        0,
		TTL:              &time.Time{},
		InternalTaskJobs: &dataStructures.InternalTaskJobs{},
		Segment:          &dataStructures.Segment{},
		LifeCycleStage:   dataStructures.LifeCycleStageCreated,
		BQTable: &dataStructures.BillingTableInfo{
			TableID:         utils.GetLocalCopyAccountTableName(requestParams.BillingAccountID),
			DatasetID:       consts.LocalBillingDataset,
			ProjectID:       utils.GetProjectName(),
			OldestPartition: truncatedOldestRecordTime,
		},
		Dummy: utils.IsDummy(requestParams.Dataset),
		CopyHistory: &dataStructures.CopyHistoryData{
			TargetTime: &truncatedTargetTime,
			Status:     dataStructures.CopyHistoryStatusPending,
		},
		CustomerID: requestParams.CustomerID,
	}
}

func retryFunctionWrapper(retryFunction func() error) error {
	err := retry.BackOffDelayWithOriginalError(
		retryFunction,
		consts.MetadataOperationMaxRetries,
		consts.MetadataOperationFirstRetryDelay,
	)

	return err
}
