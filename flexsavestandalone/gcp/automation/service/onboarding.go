package service

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"github.com/go-resty/resty/v2"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/utils"
	billingDataStructures "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Onboarding struct {
	loggerProvider logger.Provider
	*connection.Connection
}

func NewOnboarding(log logger.Provider, conn *connection.Connection) *Onboarding {
	return &Onboarding{
		loggerProvider: log,
		Connection:     conn,
	}
}

func (o *Onboarding) SendOnboardingRequests(ctx context.Context, atms []*dataStructures.AutomationTaskMetadata) error {
	logger := o.loggerProvider(ctx)
	onboardingWG := sync.WaitGroup{}
	onboardingWG.Add(len(atms))

	for i, atm := range atms {
		go func(atm *dataStructures.AutomationTaskMetadata) {
			defer onboardingWG.Done()

			onboardBody := billingDataStructures.OnboardingRequestBody{
				BillingAccountID:    atm.BillingAccountID,
				ProjectID:           consts.DummyBQProjectName,
				Dataset:             consts.DummyBQDatasetName,
				TableID:             atm.BQTable.TableID,
				CustomerID:          utils.GetDummyCustomerID(atm.BillingAccountID),
				ServiceAccountEmail: atm.ServiceAccount,
			}

			if common.IsLocalhost {
				restClient := resty.New()

				response, err := restClient.SetTimeout(time.Minute * 10).R().SetBody(onboardBody).Post(fmt.Sprintf("http://localhost:%s/tasks/flexsave-standalone/google-cloud/billing/onboarding", os.Getenv("PORT")))
				if err != nil {
					logger.Errorf("unable to run task %+v. Caused by %s", onboardBody, err.Error())
				} else {
					logger.Infof("task %+v triggered. Details: %+v", onboardBody, response.RawResponse)
				}
			} else {
				config := common.CloudTaskConfig{
					Method: cloudtaskspb.HttpMethod_POST,
					Path:   "/tasks/flexsave-standalone/google-cloud/billing/onboarding",
					Queue:  common.TaskQueueFlexSaveStandaloneOnboarding,
				}

				task, err := o.CloudTaskClient.CreateTask(ctx, config.Config(onboardBody))
				if err != nil {
					logger.Errorf("unable to schedule onboarding task. Caused by %s", err)
				} else {
					logger.Infof("scheduled task %s", task.String())
				}
			}
		}(atm)

		if i%10 == 0 {
			logger.Infof("waiting to finish onboarding balk ...")
			time.Sleep(time.Second * 20)
		}
	}

	onboardingWG.Wait()

	return nil
}

func (o *Onboarding) SendOffboardingRequests(ctx context.Context, atms []*dataStructures.AutomationTaskMetadata) error {
	logger := o.loggerProvider(ctx)
	offboardingWG := sync.WaitGroup{}
	offboardingWG.Add(len(atms))

	for i, atm := range atms {
		go func(atm *dataStructures.AutomationTaskMetadata) {
			defer offboardingWG.Done()

			onboardBody := billingDataStructures.DeleteBillingRequestBody{
				BillingAccountID: atm.BillingAccountID,
			}

			if common.IsLocalhost {
				restClient := resty.New()

				response, err := restClient.SetTimeout(time.Minute * 10).R().SetBody(onboardBody).Post(fmt.Sprintf("http://localhost:%s/tasks/flexsave-standalone/google-cloud/billing/offboarding", os.Getenv("PORT")))
				if err != nil {
					logger.Errorf("unable to run task %+v. Caused by %s", onboardBody, err.Error())
				} else {
					logger.Infof("task %+v triggered. Details: %+v", onboardBody, response.RawResponse)
				}
			} else {
				config := common.CloudTaskConfig{
					Method: cloudtaskspb.HttpMethod_POST,
					Path:   "/tasks/flexsave-standalone/google-cloud/billing/offboarding",
					Queue:  common.TaskQueueFlexSaveStandaloneOffboarding,
				}

				task, err := o.CloudTaskClient.CreateTask(ctx, config.Config(onboardBody))
				if err != nil {
					logger.Errorf("unable to schedule offboarding task. Caused by %s", err)
				} else {
					logger.Infof("scheduled task %s", task.String())
				}
			}
		}(atm)

		if i%10 == 0 {
			logger.Infof("waiting to finish offboarding balk ...")
			time.Sleep(time.Second * 20)
		}
	}

	offboardingWG.Wait()

	return nil
}
