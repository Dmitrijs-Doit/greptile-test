package application

import (
	"context"
	"fmt"

	"os"
	"sync"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"github.com/go-resty/resty/v2"

	googleCloudConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/service"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/dataStructures"
	billingService "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/service"
	billingUtils "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils"
	billingConsts "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/mailer"
	"github.com/doitintl/hello/scheduled-tasks/notification"
)

func NewAutomationManager(log logger.Provider, conn *connection.Connection, oldLog *logger.Logging) *AutomationManager {
	return &AutomationManager{
		Logger:            log,
		Connection:        conn,
		metadata:          service.NewMetadata(log, conn),
		customerBQClient:  billingService.NewExternalBigQueryClient(log, conn),
		table:             service.NewTable(log, conn),
		onboarding:        service.NewOnboarding(log, conn),
		serviceAccount:    service.NewServiceAccount(log, conn),
		billingTableQuery: billingService.NewTableQuery(log, conn),
		notification:      notification.NewNotification(),
		billingMetadata:   billingService.NewMetadata(log, conn),
		orchestration:     NewAutomationOrchestrator(log, conn),
	}
}

type AutomationManager struct {
	Logger logger.Provider
	*connection.Connection
	metadata         *service.Metadata
	customerBQClient billingService.ExternalBigQueryClient
	table            *service.Table
	//ba               *service.BillingAccount
	onboarding        *service.Onboarding
	serviceAccount    *service.ServiceAccount
	billingTableQuery billingService.TableQuery
	notification      *notification.Notification
	billingMetadata   billingService.Metadata
	orchestration     *AutomationOrchestrator
}

//func (a *AutomationManager) RunAutomation(ctx context.Context) error {
//	logger := a.Logger(ctx)
//	startTime := time.Now()
//	jobTtl := startTime.Add(consts.AutomationJobMaxDuration)
//	taskTtl := startTime.Add(consts.AutomationTaskMaxDuration)
//
//	amm, err := a.initAutomation(ctx, &taskTtl)
//	if err != nil {
//		//switch err.(type) {
//		//case *errors.FirstIterationError:
//		//	logger.Error(err)
//		//	return nil
//		//}
//		err = fmt.Errorf("unable to initAutomation. Caused by %s", err)
//		logger.Error(err)
//		return err
//	}
//
//	amm, err = a.updateTasks(ctx, amm, &jobTtl, &taskTtl)
//	if err != nil {
//		err = fmt.Errorf("unable to updateTasks. Caused by %s", err)
//		logger.Error(err)
//		return err
//	}
//
//	amm, err = a.runTasks(ctx, amm)
//	if err != nil {
//		err = fmt.Errorf("unable to updateTasks. Caused by %s", err)
//		logger.Error(err)
//		return err
//	}
//
//	amm, err = a.tearDown(ctx, amm)
//	if err != nil {
//		err = fmt.Errorf("unable to updateTasks. Caused by %s", err)
//		logger.Error(err)
//		return err
//	}
//
//	logger.Infof("End of iteration %d after %v", amm.Iteration, time.Since(startTime))
//	return nil
//}

func (a *AutomationManager) RunAutomation(ctx context.Context) error {
	logger := a.Logger(ctx)
	startTime := time.Now()
	jobTtl := startTime.Add(consts.AutomationJobMaxDuration)
	taskTtl := startTime.Add(consts.AutomationTaskMaxDuration)

	amm, aom, err := a.initAutomation(ctx, &taskTtl)
	if err != nil {
		err = fmt.Errorf("unable to initAutomation. Caused by %s", err)
		logger.Error(err)

		return err
	}

	switch amm.Stage {
	case dataStructures.AutomationManagerStagePending:
		err = a.billingMetadata.DeleteInternalTaskMetadata(ctx, googleCloudConsts.MasterBillingAccount)
		if err != nil {
			err = fmt.Errorf("unable to DeleteInternalTaskMetadata for BA %s. Caused by %s", googleCloudConsts.MasterBillingAccount, err)
			logger.Error(err)

			return err
		}

		err = a.metadata.DeleteDeprecatedAutomationTasks(ctx, aom.Version)
		if err != nil {
			err = fmt.Errorf("unable to DeleteDeprecatedAutomationTasks. Caused by %s", err)
			logger.Error(err)

			return err
		}

		err = a.metadata.CreateAutomationTasks(ctx, aom)
		if err != nil {
			err = fmt.Errorf("unable to CreateAutomationTasks. Caused by %s", err)
			logger.Error(err)

			return err
		}

		atms, err := a.metadata.GetTasksByVersion(ctx, aom.Version)
		if err != nil {
			err = fmt.Errorf("unable to GetTasksByVersion for version %d. Caused by %s", aom.Version, err)
			logger.Error(err)

			return err
		}

		err = a.onboarding.SendOnboardingRequests(ctx, atms)
		if err != nil {
			err = fmt.Errorf("unable to SendOnboardingRequests. Caused by %s", err)
			logger.Error(err)

			return err
		}

		amm, err = a.metadata.SetAutomationManager(ctx, func(ctx context.Context, amm *dataStructures.AutomationManagerMetadata) error {
			amm.Stage = dataStructures.AutomationManagerStageWriting
			return nil
		})
		if err != nil {
			err = fmt.Errorf("unable to SetAutomationManager. Caused by %s", err)
			logger.Error(err)

			return err
		}
		//return amm, errors.NewFirstIterationError()

	case dataStructures.AutomationManagerStageWriting:
		amm, err = a.updateTasks(ctx, amm, &jobTtl, &taskTtl)
		if err != nil {
			err = fmt.Errorf("unable to updateTasks. Caused by %s", err)
			logger.Error(err)

			return err
		}

		_, err = a.runTasks(ctx, amm)
		if err != nil {
			err = fmt.Errorf("unable to updateTasks. Caused by %s", err)
			logger.Error(err)

			return err
		}

		if aom.WriteTime == nil || time.Until(*aom.WriteTime) < 0 {
			_, err = a.metadata.SetAutomationManager(ctx, func(ctx context.Context, amm *dataStructures.AutomationManagerMetadata) error {
				amm.Stage = dataStructures.AutomationManagerStageWaitToVerifyRowCount
				return nil
			})
			if err != nil {
				err = fmt.Errorf("unable to SetAutomationManager. Caused by %s", err)
				logger.Error(err)

				return err
			}
		}

		return nil
	case dataStructures.AutomationManagerStageWaitToVerifyRowCount:
		if time.Until(*aom.WaitUntilVerificationTime) > 0 {
			logger.Infof("waiting until %s to start the verification", aom.WaitUntilVerificationTime)
			return nil
		} else {
			_, err = a.verifyRowCount(ctx, amm)
			if err != nil {
				err = fmt.Errorf("unable to verifyRowCount. Caused by %s", err)
				logger.Error(err)

				return err
			}

			_, err = a.metadata.SetAutomationManager(ctx, func(ctx context.Context, amm *dataStructures.AutomationManagerMetadata) error {
				amm.Stage = dataStructures.AutomationManagerStageVerifyingRowCount
				return nil
			})
			if err != nil {
				err = fmt.Errorf("unable to SetAutomationManager. Caused by %s", err)
				logger.Error(err)

				return err
			}
		}

	case dataStructures.AutomationManagerStageVerifyingRowCount:
		atms, err := a.metadata.GetAutomationTasksMetadataByVersion(ctx, amm.Version)
		if err != nil {
			err = fmt.Errorf("unable to GetAutomationTasksMetadataByVersion. Caused by %s", err)
			logger.Error(err)

			return err
		}

		allVerified := true

		for _, atm := range atms {
			if !atm.Verified {
				allVerified = false

				logger.Warning("no verification for BA %s. retrying...", atm.BillingAccountID)
			}
		}

		if allVerified {
			amm, err = a.metadata.SetAutomationManager(ctx, func(ctx context.Context, amm *dataStructures.AutomationManagerMetadata) error {
				amm.Stage = dataStructures.AutomationManagerStageNotifying
				return nil
			})
			if err != nil {
				err = fmt.Errorf("unable to SetAutomationManager. Caused by %s", err)
				logger.Error(err)

				return err
			}
		} else {
			amm, err = a.verifyRowCount(ctx, amm)
			if err != nil {
				err = fmt.Errorf("unable to verifyRowCount. Caused by %s", err)
				logger.Error(err)

				return err
			}
		}

	case dataStructures.AutomationManagerStageNotifying:
		generalResult, err := a.sendNotification(ctx, amm)
		if err != nil {
			err = fmt.Errorf("unable to sendNotification. Caused by %s", err)
			logger.Error(err)

			return err
		}

		amm, err = a.metadata.SetAutomationManager(ctx, func(ctx context.Context, amm *dataStructures.AutomationManagerMetadata) error {
			if generalResult {
				amm.Stage = dataStructures.AutomationManagerStageCleanup
			} else {
				amm.Stage = dataStructures.AutomationManagerStageFailed
			}

			return nil
		})
		if err != nil {
			err = fmt.Errorf("unable to SetAutomationManager. Caused by %s", err)
			logger.Error(err)

			return err
		}

	case dataStructures.AutomationManagerStageCleanup:
		atms, err := a.metadata.GetAutomationTasksMetadataByVersion(ctx, amm.Version)
		if err != nil {
			err = fmt.Errorf("unable to GetAutomationTasksMetadataByVersion. Caused by %s", err)
			logger.Error(err)

			return err
		}

		err = a.onboarding.SendOffboardingRequests(ctx, atms)
		if err != nil {
			err = fmt.Errorf("unable to SendOffboardingRequests. Caused by %s", err)
			logger.Error(err)

			return err
		}

		err = a.orchestration.DeleteAutomation(ctx)
		if err != nil {
			err = fmt.Errorf("unable to DeleteAutomation. Caused by %s", err)
			logger.Error(err)

			return err
		}

		amm, err = a.metadata.SetAutomationManager(ctx, func(ctx context.Context, amm *dataStructures.AutomationManagerMetadata) error {
			amm.Stage = dataStructures.AutomationManagerStageDone
			return nil
		})

		if err != nil {
			err = fmt.Errorf("unable to SetAutomationManager. Caused by %s", err)
			logger.Error(err)

			return err
		}

	case dataStructures.AutomationManagerStageDone:
		logger.Infof("Automation Suite done. Pending for next iteration")

	case dataStructures.AutomationManagerStageFailed:
		logger.Infof("Automation Suite FAILED. check for previous errors")
	}

	amm, err = a.tearDown(ctx, amm)
	if err != nil {
		err = fmt.Errorf("unable to updateTasks. Caused by %s", err)
		logger.Error(err)

		return err
	}

	logger.Infof("End of iteration %d after %v", amm.Iteration, time.Since(startTime))

	return nil
}

//billingTableQuery

func getResult(writtenRows *dataStructures.WrittenRows) string {
	if writtenRows != nil && writtenRows.ExpectedWrittenRows == writtenRows.CustomerWrittenRows &&
		writtenRows.ExpectedWrittenRows == writtenRows.LocalWrittenRows &&
		writtenRows.ExpectedWrittenRows == writtenRows.UnifiedWrittenRows {
		return "PASSED"
	}

	return "FAILED"
}

func (a *AutomationManager) sendNotification(ctx context.Context, amm *dataStructures.AutomationManagerMetadata) (bool, error) {
	logger := a.Logger(ctx)

	atms, err := a.metadata.GetAutomationTasksMetadataByVersion(ctx, amm.Version)
	if err != nil {
		err = fmt.Errorf("unable to GetAutomationTasksMetadataByVersion. Caused by %s", err)
		logger.Error(err)

		return false, err
	}

	body := "automation results:"
	//body := "TEST TEST TEST TEST"

	generalResult := true

	for _, atm := range atms {
		body = fmt.Sprintf("%s\nBA %s:%s Expected: %d Customer: %d Local: %d Unified: %d",
			body, atm.BillingAccountID, getResult(atm.WrittenRows), atm.WrittenRows.ExpectedWrittenRows,
			atm.WrittenRows.CustomerWrittenRows, atm.WrittenRows.LocalWrittenRows, atm.WrittenRows.UnifiedWrittenRows)
		generalResult = generalResult && getResult(atm.WrittenRows) == "PASSED"
	}

	sn := &mailer.SimpleNotification{
		Subject: fmt.Sprintf("Automation Suite Result for env %s", billingUtils.GetProjectName()),
		//Subject:   fmt.Sprintf("URGENT - %s - Flexsave Billing Data Is Not Updated", billingUtils.GetProjectName()),
		Preheader: fmt.Sprintf("Automation Suite Result for env %s", billingUtils.GetProjectName()),
		CCs:       []string{"lionel@doit-intl.com"},
	}

	mnt := &notification.MailNotificationTarget{
		To:                 "lionel@doit-intl.com",
		SimpleNotification: sn,
	}

	snt := &notification.SlackNotificationTarget{
		Channel: "#billing-data-pipe-automation-results",
		//Channel: "C04B5F4K5E3",
	}

	severity := notification.SeverityUrgent

	if generalResult {
		severity = notification.SeverityInfo
	}

	a.notification.SendNotification(ctx, severity, []string{body}, mnt, snt)

	return generalResult, nil
}

func (a *AutomationManager) verifyRowCount(ctx context.Context, amm *dataStructures.AutomationManagerMetadata) (*dataStructures.AutomationManagerMetadata, error) {
	logger := a.Logger(ctx)
	verifyRowCountTasksWg := sync.WaitGroup{}

	atms, err := a.metadata.GetAutomationTasksMetadataByVersion(ctx, amm.Version)
	if err != nil {
		err = fmt.Errorf("unable to GetAutomationTasksMetadataByVersion. Caused by %s", err)
		logger.Error(err)

		return nil, err
	}

	verifyRowCountTasksWg.Add(len(atms))

	for _, atm := range atms {
		go func(atm *dataStructures.AutomationTaskMetadata) {
			defer func(amt *dataStructures.AutomationTaskMetadata) {
				verifyRowCountTasksWg.Done()
				logger.Info("done verifying for BA %s", atm.BillingAccountID)
			}(atm)

			if atm.Verified {
				return
			}

			localRowCount, err := a.billingTableQuery.GetLocalRowsCountPerTimeRange(ctx, atm.BillingAccountID, nil, nil)
			if err != nil {
				err = fmt.Errorf("unable to GetLocalRowsCountPerTimeRange. Caused by %s", err)
				logger.Error(err)

				return
			}

			unifiedRowCount, err := a.billingTableQuery.GetFromUnifiedTableRowsCountPerTimeRange(ctx, atm.BillingAccountID, nil, nil)
			if err != nil {
				err = fmt.Errorf("unable to GetLocalRowsCountPerTimeRange. Caused by %s", err)
				logger.Error(err)

				return
			}

			customerRowCount, err := a.billingTableQuery.GetCustomerRowsCountPerTimeRange(ctx, atm.BillingAccountID, nil, nil)
			if err != nil {
				err = fmt.Errorf("unable to GetLocalRowsCountPerTimeRange. Caused by %s", err)
				logger.Error(err)

				return
			}

			_, err = a.metadata.SetAutomationTask(ctx, atm.BillingAccountID, func(ctx context.Context, uamt *dataStructures.AutomationTaskMetadata) error {
				uamt.WrittenRows.LocalWrittenRows = localRowCount
				uamt.WrittenRows.UnifiedWrittenRows = unifiedRowCount
				uamt.WrittenRows.CustomerWrittenRows = customerRowCount
				uamt.Verified = true

				return nil
			})
			if err != nil {
				err = fmt.Errorf("unable to SetAutomationTask for BA %s. Caused by %s", atm.BillingAccountID, err)
				logger.Error(err)
			}
		}(atm)
	}

	verifyRowCountTasksWg.Wait()

	return amm, nil
}

func (a *AutomationManager) updateTasks(ctx context.Context, amm *dataStructures.AutomationManagerMetadata, jobTtl *time.Time, taskTtl *time.Time) (*dataStructures.AutomationManagerMetadata, error) {
	logger := a.Logger(ctx)
	updateTasksWg := sync.WaitGroup{}

	atms, err := a.metadata.GetAutomationTasksMetadataByVersion(ctx, amm.Version)
	if err != nil {
		err = fmt.Errorf("unable to GetAutomationTasksMetadataByVersion. Caused by %s", err)
		logger.Error(err)

		return nil, err
	}

	updateTasksWg.Add(len(atms))

	for _, atm := range atms {
		go func(atm *dataStructures.AutomationTaskMetadata) {
			defer updateTasksWg.Done()

			_, err = a.metadata.SetAutomationTask(ctx, atm.BillingAccountID, func(ctx context.Context, uamt *dataStructures.AutomationTaskMetadata) error {
				uamt.Iteration = amm.Iteration
				uamt.TTL = taskTtl
				uamt.JobTimeout = jobTtl
				//TODO handle hanging tasks
				uamt.Running = false

				return nil
			})
			if err != nil {
				err = fmt.Errorf("unable to SetAutomationTask for BA %s. Caused by %s", atm.BillingAccountID, err)
				logger.Error(err)
			}
		}(atm)
	}

	updateTasksWg.Wait()

	return amm, nil
}

func (a *AutomationManager) runTasks(ctx context.Context, amm *dataStructures.AutomationManagerMetadata) (*dataStructures.AutomationManagerMetadata, error) {
	logger := a.Logger(ctx)
	runTasksWg := sync.WaitGroup{}

	atms, err := a.metadata.GetAutomationTasksMetadataByIteration(ctx, amm.Version)
	if err != nil {
		err = fmt.Errorf("unable to GetAutomationTasksMetadataByVersion. Caused by %s", err)
		logger.Error(err)

		return nil, err
	}

	runTasksWg.Add(len(atms))

	for _, atm := range atms {
		go func(atm *dataStructures.AutomationTaskMetadata) {
			defer runTasksWg.Done()

			err = a.createCloudTask(ctx, atm)
			if err != nil {
				err = fmt.Errorf("unable to SetAutomationTask for BA %s. Caused by %s", atm.BillingAccountID, err)
				logger.Error(err)
			}
		}(atm)
	}

	runTasksWg.Wait()

	return amm, nil
}

func (a *AutomationManager) tearDown(ctx context.Context, amm *dataStructures.AutomationManagerMetadata) (*dataStructures.AutomationManagerMetadata, error) {
	logger := a.Logger(ctx)

	uamm, err := a.metadata.SetAutomationManager(ctx, func(ctx context.Context, amm *dataStructures.AutomationManagerMetadata) error {
		amm.Running = false
		return nil
	})
	if err != nil {
		err = fmt.Errorf("unable to SetAutomationManager. Caused by %s", err)
		logger.Error(err)

		return amm, err
	}

	amm = uamm

	return amm, nil
}

func (m *AutomationManager) createCloudTask(ctx context.Context, atm *dataStructures.AutomationTaskMetadata) error {
	logger := m.Logger(ctx)
	body := &dataStructures.AutomationTaskRequest{
		BillingAccountID: atm.BillingAccountID,
		Iteration:        atm.Iteration,
		Version:          atm.Version,
	}

	if common.IsLocalhost {
		restClient := resty.New()

		logger.Infof("sending task %+v", body)

		response, err := restClient.R().SetBody(body).Post(fmt.Sprintf("http://localhost:%s/tasks/flexsave-standalone/google-cloud/billing/automation/task", os.Getenv("PORT")))
		if err != nil {
			logger.Errorf("unable to run task %+v. Caused by %s", body, err.Error())
		} else {
			logger.Infof("task %+v triggered. Details: %+v", body, response.RawResponse)
		}

		logger.Info(body)
	} else {
		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   "/tasks/flexsave-standalone/google-cloud/billing/automation/task",
			Queue:  common.TaskQueueFlexSaveStandaloneAutomationTasks,
		}

		task, err := m.CloudTaskClient.CreateTask(ctx, config.Config(body))
		if err != nil {
			logger.Errorf("unable to schedule task. Caused by %s", err)
			return err
		}

		logger.Infof("scheduled task %s", task.String())

		return nil
	}

	return nil
}

func (a *AutomationManager) initAutomation(ctx context.Context, ttl *time.Time) (*dataStructures.AutomationManagerMetadata, *dataStructures.AutomationOrchestratorMetadata, error) {
	logger := a.Logger(ctx)

	aom, err := a.metadata.GetOrchestration(ctx)
	if err != nil {
		err = fmt.Errorf("unable to GetOrchestration. Caused by %s", err)
		logger.Error(err)

		return nil, nil, err
	}

	amm, err := a.metadata.GetManager(ctx)
	if err != nil {
		err = fmt.Errorf("unable to GetOrchestration. Caused by %s", err)
		logger.Error(err)

		return nil, nil, err
	}

	if aom.Version != amm.Version {
		err = fmt.Errorf("unable to run automation. Caused by invalid MD version: expected %d but found %d", aom.Version, amm.Version)
		logger.Error(err)

		return nil, nil, err
	}

	if billingUtils.GetProjectName() == billingConsts.BillingProjectProd {
		err = fmt.Errorf("unable to run automation. Caused by unable to run in env %s. Caused by unable tu run in production", billingUtils.GetProjectName())
		logger.Error(err)

		return nil, nil, err
	}

	amm, err = a.metadata.SetAutomationManager(ctx, func(ctx context.Context, amm *dataStructures.AutomationManagerMetadata) error {
		//TODO handle timeout case
		amm.Running = true
		amm.Iteration = amm.Iteration + 1
		amm.TTL = ttl

		return nil
	})

	if err != nil {
		err = fmt.Errorf("unable to SetAutomationManager. Caused by %s", err)
		logger.Error(err)

		return nil, nil, err
	}

	return amm, aom, nil
}
