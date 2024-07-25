package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/utils"
	billingDatastructures "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Metadata struct {
	loggerProvider logger.Provider
	*connection.Connection
	dal *dal.Metadata
	sa  *ServiceAccount
}

func NewMetadata(log logger.Provider, conn *connection.Connection) *Metadata {
	return &Metadata{
		loggerProvider: log,
		Connection:     conn,
		dal:            dal.NewMetadata(log, conn),
		sa:             NewServiceAccount(log, conn),
	}
}

func (m *Metadata) GetOrchestration(ctx context.Context) (*dataStructures.AutomationOrchestratorMetadata, error) {
	return m.dal.GetAutomationOrchestratorMetadata(ctx)
}

func (m *Metadata) GetManager(ctx context.Context) (*dataStructures.AutomationManagerMetadata, error) {
	return m.dal.GetAutomationManagerMetadata(ctx)
}

func (m *Metadata) GetTasksByVersion(ctx context.Context, currVersion int64) (atms []*dataStructures.AutomationTaskMetadata, err error) {
	return m.dal.GetAutomationTasksMetadataByVersion(ctx, currVersion)
}

func (m *Metadata) CreateOrchestration(ctx context.Context, or *dataStructures.OrchestratorRequest) (*dataStructures.AutomationOrchestratorMetadata, error) {
	logger := m.loggerProvider(ctx)

	uaom, err := m.dal.CreateAutomationOrchestratorMetadata(ctx, m.createOrchestrationObject(or))
	if err != nil {
		err = fmt.Errorf("unable to CreateAutomationOrchestratorMetadata. Cause by %s", err)
		logger.Error(err)

		return nil, err
	}

	return uaom, nil
}

func (m *Metadata) createOrchestrationObject(or *dataStructures.OrchestratorRequest) *dataStructures.AutomationOrchestratorMetadata {
	creationTime := time.Now()
	durationInMinutes := time.Minute * time.Duration(or.DurationInMinutes)
	terminationTime := creationTime.Add(durationInMinutes)
	verificationTime := terminationTime.Add(2 * time.Hour)

	return &dataStructures.AutomationOrchestratorMetadata{
		Version:                   1,
		WriteTime:                 &terminationTime,
		WaitUntilVerificationTime: &verificationTime,
		CreationTime:              &creationTime,
		NumOfDummyUsers:           or.NumOfDummyUsers,
		MinNumOfKiloRowsPerHour:   or.MinNumOfKiloRowsPerHour,
		MaxNumOfKiloRowsPerHour:   or.MaxNumOfKiloRowsPerHour,
	}
}

func (m *Metadata) DeleteOrchestration(ctx context.Context) error {
	logger := m.loggerProvider(ctx)

	err := m.dal.DeleteAutomationOrchestratorMetadata(ctx)
	if err != nil {
		err = fmt.Errorf("unable to DeleteAutomationOrchestratorMetadata. Cause by %s", err)
		logger.Error(err)

		return err
	}

	return nil
}

func (m *Metadata) DeleteAutomationManager(ctx context.Context) error {
	logger := m.loggerProvider(ctx)

	err := m.dal.DeleteAutomationManagerMetadata(ctx)
	if err != nil {
		err = fmt.Errorf("unable to DeleteAutomationManagerMetadata. Cause by %s", err)
		logger.Error(err)

		return err
	}

	return nil
}

func (m *Metadata) DeleteAutomationTasks(ctx context.Context) error {
	return m.dal.DeleteAllAutomationTasksMetadata(ctx)
}

func (m *Metadata) CreateAutomationManager(ctx context.Context) error {
	return m.dal.CreateDefaultAutomationManagerMetadata(ctx)
}

func (m *Metadata) CreateAutomationOrchestrator(ctx context.Context) error {
	return m.dal.CreateDefaultAutomationOrchestratorMetadata(ctx)
}

func (m *Metadata) SetAutomationManager(ctx context.Context, updateFunc func(ctx context.Context, amm *dataStructures.AutomationManagerMetadata) error) (uamm *dataStructures.AutomationManagerMetadata, err error) {
	logger := m.loggerProvider(ctx)

	uamm, err = m.dal.SetAutomationManagerMetadata(ctx, updateFunc)
	if err != nil {
		err = fmt.Errorf("unable to SetAutomationManagerMetadata. Cause by %s", err)
		logger.Error(err)

		return nil, err
	}

	return uamm, nil
}

func (m *Metadata) SetAutomationTask(ctx context.Context, billingAccount string, updateFunc func(ctx context.Context, amt *dataStructures.AutomationTaskMetadata) error) (uamt *dataStructures.AutomationTaskMetadata, err error) {
	logger := m.loggerProvider(ctx)

	uamt, err = m.dal.SetAutomationTaskMetadata(ctx, billingAccount, updateFunc)
	if err != nil {
		err = fmt.Errorf("unable to SetAutomationTaskMetadata. Cause by %s", err)
		logger.Error(err)

		return nil, err
	}

	return uamt, nil
}

func (m *Metadata) DeleteDeprecatedAutomationTasks(ctx context.Context, currVersion int64) error {
	return m.dal.DeleteDeprecatedAutomationTasksMetadata(ctx, currVersion)
}

func (m *Metadata) GetAutomationTasksMetadataByVersion(ctx context.Context, currVersion int64) (atms []*dataStructures.AutomationTaskMetadata, err error) {
	return m.dal.GetAutomationTasksMetadataByVersion(ctx, currVersion)
}

func (m *Metadata) GetAllAutomationTasksMetadata(ctx context.Context) (atms []*dataStructures.AutomationTaskMetadata, err error) {
	return m.dal.GetAutomationTasksMetadata(ctx)
}

func (m *Metadata) GetAutomationTaskMetadata(ctx context.Context, billingAccount string) (atms *dataStructures.AutomationTaskMetadata, err error) {
	return m.dal.GetAutomationTaskMetadata(ctx, billingAccount)
}

func (m *Metadata) GetAutomationTasksMetadataByIteration(ctx context.Context, currVersion int64) (atms []*dataStructures.AutomationTaskMetadata, err error) {
	return m.dal.GetAutomationTasksMetadataByVersion(ctx, currVersion)
}

func (m *Metadata) CreateAutomationTasks(ctx context.Context, aom *dataStructures.AutomationOrchestratorMetadata) error {
	logger := m.loggerProvider(ctx)
	now := time.Now()
	waitForTasks := sync.WaitGroup{}
	waitForTasks.Add(aom.NumOfDummyUsers)

	for i := 0; i < aom.NumOfDummyUsers; i++ {
		go func(index int) {
			defer waitForTasks.Done()

			billingAccount := utils.GetDummyBillingAccount(aom.Version, index)
			tableName := utils.GetDummyTableName(aom.Version, index)

			sa, err := m.sa.GetServiceAccountForBillingAccount(ctx, billingAccount, tableName)
			if err != nil {
				err = fmt.Errorf("unable to GetServiceAccountForBillingAccount for BA %s. Caused by %s", billingAccount, err)
				logger.Error(err)

				return
			}

			err = m.dal.CreateAutomationTaskMetadata(ctx, &dataStructures.AutomationTaskMetadata{
				Iteration:        1,
				Version:          aom.Version,
				Verified:         false,
				BillingAccountID: billingAccount,
				ServiceAccount:   sa.ServiceAccountID,
				RowsPerHour:      aom.MinNumOfKiloRowsPerHour * 1000,
				WriteTime:        aom.WriteTime,
				StartTime:        &now,

				WrittenRows: &dataStructures.WrittenRows{},
				Active:      true,
				BQTable: &billingDatastructures.BillingTableInfo{
					ProjectID: consts.DummyBQProjectName,
					DatasetID: consts.DummyBQDatasetName,
					TableID:   tableName,
				},
			})
			if err != nil {
				err = fmt.Errorf("unable to CreateAutomationTaskMetadata for %s. Caused by %s", billingAccount, err)
				logger.Error(err)
			}
		}(i)
	}
	waitForTasks.Wait()

	return nil
}

func (m *Metadata) StopOrchestration(ctx context.Context) error {
	logger := m.loggerProvider(ctx)

	_, err := m.dal.SetAutomationOrchestratorMetadata(ctx, func(ctx context.Context, aom *dataStructures.AutomationOrchestratorMetadata) error {
		now := time.Now()
		aom.WriteTime = &now

		return nil
	})
	if err != nil {
		err = fmt.Errorf("unable to SetAutomationOrchestratorMetadata. Cause by %s", err)
		logger.Error(err)

		return err
	}

	return nil
}
