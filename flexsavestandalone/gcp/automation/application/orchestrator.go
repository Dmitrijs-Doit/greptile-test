package application

import (
	"context"
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/service"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func NewAutomationOrchestrator(log logger.Provider, conn *connection.Connection) *AutomationOrchestrator {
	return &AutomationOrchestrator{
		Logger:     log,
		Connection: conn,
		metadata:   service.NewMetadata(log, conn),
		table:      service.NewTable(log, conn),
		//ba:         service.NewBillingAccount(log, conn),
		sa: service.NewServiceAccount(log, conn),
	}
}

type AutomationOrchestrator struct {
	Logger logger.Provider
	*connection.Connection
	metadata *service.Metadata
	table    *service.Table
	//ba       *service.BillingAccount
	sa *service.ServiceAccount
}

func (a *AutomationOrchestrator) CreateOrchestration(ctx context.Context, or *dataStructures.OrchestratorRequest) error {
	logger := a.Logger(ctx)

	aom, err := a.metadata.CreateOrchestration(ctx, or)
	if err != nil {
		err = fmt.Errorf("unable to CreateOrchestration for request %v. Caused by %s", or, err)
		logger.Error(err)

		return err
	}

	_, err = a.metadata.SetAutomationManager(ctx, func(ctx context.Context, amm *dataStructures.AutomationManagerMetadata) error {
		amm.Running = true
		amm.Iteration = 0
		amm.Version = aom.Version
		amm.Stage = dataStructures.AutomationManagerStagePending

		return nil
	})

	if err != nil {
		err = fmt.Errorf("unable to SetAutomationManager. Caused by %s", err)
		logger.Error(err)

		return err
	}

	err = a.table.CreateDummyTables(ctx, aom)
	if err != nil {
		err = fmt.Errorf("unable to CreateDummyTables. Caused by %s", err)
		logger.Error(err)

		return err
	}

	return nil
}

func (a *AutomationOrchestrator) StopOrchestration(ctx context.Context) error {
	logger := a.Logger(ctx)

	err := a.metadata.StopOrchestration(ctx)
	if err != nil {
		err = fmt.Errorf("unable to StopOrchestration. Caused by %s", err)
		logger.Error(err)
	}

	return err
}

func (a *AutomationOrchestrator) DeleteAutomation(ctx context.Context) error {
	logger := a.Logger(ctx)

	err := a.metadata.DeleteOrchestration(ctx)
	if err != nil {
		err = fmt.Errorf("unable to StopOrchestration. Caused by %s", err)
		logger.Error(err)
	}

	err = a.metadata.DeleteAutomationManager(ctx)
	if err != nil {
		err = fmt.Errorf("unable to DeleteAutomationManager. Caused by %s", err)
		logger.Error(err)
	}

	//atms, err := a.metadata.GetAllAutomationTasksMetadata(ctx)
	//if err != nil {
	//	err = fmt.Errorf("unable to GetAllAutomationTasksMetadata. Caused by %s", err)
	//	logger.Error(err)
	//}

	//err = a.ba.DeleteBillingAccounts(ctx, atms)
	//if err != nil {
	//	err = fmt.Errorf("unable to DeleteAutomationTasks. Caused by %s", err)
	//	logger.Error(err)
	//}

	err = a.metadata.DeleteAutomationTasks(ctx)
	if err != nil {
		err = fmt.Errorf("unable to DeleteAutomationTasks. Caused by %s", err)
		logger.Error(err)
	}

	err = a.table.DeleteDummyTables(ctx)
	if err != nil {
		err = fmt.Errorf("unable to DeleteDummyTables. Caused by %s", err)
		logger.Error(err)
	}

	err = a.sa.DeleteServiceAccountsMetadata(ctx)
	if err != nil {
		err = fmt.Errorf("unable to DeleteServiceAccountsMetadata. Caused by %s", err)
		logger.Error(err)
	}

	//DeleteAllServiceAccountsMetadata

	err = a.metadata.CreateAutomationManager(ctx)
	if err != nil {
		err = fmt.Errorf("unable to CreateAutomationManager. Caused by %s", err)
		logger.Error(err)

		return err
	}

	err = a.metadata.CreateAutomationOrchestrator(ctx)
	if err != nil {
		err = fmt.Errorf("unable to CreateAutomationOrchestrator. Caused by %s", err)
		logger.Error(err)

		return err
	}

	err = a.sa.CreateServiceAccountsMetadata(ctx)
	if err != nil {
		err = fmt.Errorf("unable to CreateServiceAccountsMetadata. Caused by %s", err)
		logger.Error(err)

		return err
	}

	err = a.sa.GrantPermissionsToSAs(ctx)
	if err != nil {
		err = fmt.Errorf("unable to CreateServiceAccountsMetadata. Caused by %s", err)
		logger.Error(err)

		return err
	}

	return err
}
