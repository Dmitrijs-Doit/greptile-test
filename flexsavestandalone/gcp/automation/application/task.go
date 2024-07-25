package application

import (
	"context"
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/service"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/dataStructures"
	billingCommon "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/common"
	billingService "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/service"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func NewAutomationTask(log logger.Provider, conn *connection.Connection) *AutomationTask {
	return &AutomationTask{
		Logger:           log,
		Connection:       conn,
		metadata:         service.NewMetadata(log, conn),
		tquery:           service.NewTableQuery(log, conn),
		job:              billingService.NewJob(log, conn),
		customerBQClient: billingService.NewExternalBigQueryClient(log, conn),
	}
}

type AutomationTask struct {
	Logger logger.Provider
	*connection.Connection
	metadata         *service.Metadata
	tquery           *service.TableQuery
	job              *billingService.Job
	customerBQClient billingService.ExternalBigQueryClient
}

func (t *AutomationTask) RunTask(ctx context.Context, atr *dataStructures.AutomationTaskRequest) (err error) {
	logger := t.Logger(ctx)
	logger.Infof("received request for %v", atr)

	atm, err := t.initTask(ctx, atr)
	if err != nil {
		err = fmt.Errorf("unable to initTask for BA %s. Caused by %s", atr.BillingAccountID, err)
		logger.Error(err)

		return err
	}

	atm, err = t.runTask(ctx, atm)
	if err != nil {
		err = fmt.Errorf("unable to runTask for BA %s. Caused by %s", atr.BillingAccountID, err)
		logger.Error(err)

		return err
	}

	return nil
}

func (t *AutomationTask) initTask(ctx context.Context, atr *dataStructures.AutomationTaskRequest) (atm *dataStructures.AutomationTaskMetadata, err error) {
	logger := t.Logger(ctx)

	atm, err = t.metadata.SetAutomationTask(ctx, atr.BillingAccountID, func(ctx context.Context, atm *dataStructures.AutomationTaskMetadata) error {
		if !atm.Active {
			err = fmt.Errorf("invalid task state. Task for ba %s is not active", atm.BillingAccountID)
			logger.Error(err)

			return err
		}

		if atm.Version != atr.Version {
			err = fmt.Errorf("invalid version for ba %s. Expected %d but found %d", atm.BillingAccountID, atm.Version, atr.Version)
			logger.Error(err)

			return err
		}

		if atm.Iteration != atr.Iteration {
			err = fmt.Errorf("invalid iteration for ba %s. Expected %d but found %d", atm.BillingAccountID, atm.Iteration, atr.Iteration)
			logger.Error(err)

			return err
		}

		if atm.Running {
			err = fmt.Errorf("invalid task state. Task for ba %s is already running", atm.BillingAccountID)
			logger.Error(err)

			return err
		}

		atm.Running = true

		return nil
	})
	if err != nil {
		err = fmt.Errorf("unable to SetAutomationTask for BA %s. Caused by %s", atr.BillingAccountID, err)
		logger.Error(err)

		return nil, err
	}

	return atm, nil
}

func (t *AutomationTask) runTask(ctx context.Context, atm *dataStructures.AutomationTaskMetadata) (uatm *dataStructures.AutomationTaskMetadata, err error) {
	logger := t.Logger(ctx)

	job, err := t.tquery.CopyFromLocalToTmpTable(ctx, atm)
	if err != nil {
		err = fmt.Errorf("unable to SetAutomationTask for BA %s. Caused by %s", atm.BillingAccountID, err)
		logger.Error(err)

		return nil, err
	}

	customerBQ, err := t.customerBQClient.GetCustomerBQClientWithParams(ctx, atm.ServiceAccount, consts.DummyBQProjectName)
	if err != nil {
		err = fmt.Errorf("unable to GetCustomerBQClient.Caused by %s", err)
		logger.Error(err)

		return nil, err
	}
	defer customerBQ.Close()

	uatm, err = t.metadata.SetAutomationTask(ctx, atm.BillingAccountID, func(ctx context.Context, atm *dataStructures.AutomationTaskMetadata) error {
		atm.JobID = job.ID()
		return nil
	})
	if err != nil {
		err = fmt.Errorf("unable to SetAutomationTask for BA %s. Caused by %s", atm.BillingAccountID, err)
		logger.Error(err)

		return nil, err
	}

	atm = uatm
	isActive := true
	rowsToAdd := atm.RowsPerHour

	err = t.job.HandleRunningJob(ctx, job, atm.JobTimeout, consts.DummyTaskMaxExtentionDuration)
	if err != nil {
		logger.Errorf("unable to HandleRunningJob. Caused by %s", err)

		switch err.(type) {
		case *billingCommon.JobExecutionFailure:
			rowsToAdd = 0

		case *billingCommon.JobExecutionStuck:
			isActive = false
			rowsToAdd = 0

		case *billingCommon.JobCanceled:
			rowsToAdd = 0
		}
	}

	uatm, err = t.metadata.SetAutomationTask(ctx, atm.BillingAccountID, func(ctx context.Context, oatm *dataStructures.AutomationTaskMetadata) error {
		oatm.Active = isActive
		oatm.WrittenRows.ExpectedWrittenRows = oatm.WrittenRows.ExpectedWrittenRows + rowsToAdd
		oatm.JobID = job.ID()
		oatm.Running = false

		return nil
	})

	if err != nil {
		err = fmt.Errorf("unable to SetAutomationTask for BA %s. Caused by %s", atm.BillingAccountID, err)
		logger.Error(err)

		return nil, err
	}

	return uatm, nil
}
