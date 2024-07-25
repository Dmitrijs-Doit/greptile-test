package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/application"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type AutomationFS_SA_Billing struct {
	loggerProvider         logger.Provider
	conn                   *connection.Connection
	automationBilling      *application.AutomationManager
	automationOrchestrator *application.AutomationOrchestrator
	automationTask         *application.AutomationTask
}

func NewAutomationFS_SA_Billing(log logger.Provider, conn *connection.Connection, oldLog *logger.Logging) *AutomationFS_SA_Billing {
	return &AutomationFS_SA_Billing{
		loggerProvider:         log,
		conn:                   conn,
		automationBilling:      application.NewAutomationManager(log, conn, oldLog),
		automationOrchestrator: application.NewAutomationOrchestrator(log, conn),
		automationTask:         application.NewAutomationTask(log, conn),
	}
}

func (a *AutomationFS_SA_Billing) RunAutomation(ctx *gin.Context) error {
	defer ctx.Done()

	err := a.automationBilling.RunAutomation(ctx)
	if err != nil {
		//TODO handle error
		return err
	}

	return nil
}

func (a *AutomationFS_SA_Billing) ResetAllAutomation(ctx *gin.Context) error {
	defer ctx.Done()

	err := a.automationOrchestrator.DeleteAutomation(ctx)
	if err != nil {
		//TODO handle error
		return err
	}

	return nil
}

func (a *AutomationFS_SA_Billing) CreateOrchestration(ctx *gin.Context) error {
	defer ctx.Done()

	var or dataStructures.OrchestratorRequest
	if err := ctx.ShouldBindJSON(&or); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	err := a.automationOrchestrator.CreateOrchestration(ctx, &or)
	if err != nil {
		//TODO handle error
		return err
	}

	return nil
}

func (a *AutomationFS_SA_Billing) StopOrchestration(ctx *gin.Context) error {
	defer ctx.Done()

	err := a.automationOrchestrator.StopOrchestration(ctx)
	if err != nil {
		//TODO handle error
		return err
	}

	return nil
}

func (a *AutomationFS_SA_Billing) RunTask(ctx *gin.Context) error {
	defer ctx.Done()

	var atr dataStructures.AutomationTaskRequest
	if err := ctx.ShouldBindJSON(&atr); err != nil {
		return web.NewRequestError(err, http.StatusBadRequest)
	}

	err := a.automationTask.RunTask(ctx, &atr)
	if err != nil {
		//TODO handle error
		return err
	}

	return nil
}
