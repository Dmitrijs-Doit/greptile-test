package aws

import (
	"context"
	"fmt"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

const (
	awsSaaSAssets    = "amazon-web-services-saas"
	createTaskErrTpl = "failed to create aws saas asset discovery task for customer %s with error: %s"
)

type UpdateSaaSAssetsRequest struct {
	Accounts []string `json:"accounts"`
}

func (s *AWSSaaSConsoleOnboardService) UpdateAllSaaSAssets(ctx context.Context) error {
	logger := s.loggerProvider(ctx)

	saasAccounts, err := s.saasConsoleDAL.GetAWSOnboardedAccountIDsByCustomer(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch aws saas accounts; %s", err)
	}

	for customer, accounts := range saasAccounts {
		customerRef := s.customersDAL.GetRef(ctx, customer)

		activeAccounts := []string{}

		for _, account := range accounts {
			cloudConnect, err := s.cloudConnectDAL.GetAWSCloudConnect(ctx, customerRef, common.Assets.AmazonWebServices, account)
			if err == doitFirestore.ErrNotFound ||
				(cloudConnect != nil && cloudConnect.BillingEtl != nil &&
					cloudConnect.BillingEtl.Settings != nil && !cloudConnect.BillingEtl.Settings.Active) {
				continue
			}

			activeAccounts = append(activeAccounts, account)
		}

		if len(activeAccounts) == 0 {
			continue
		}

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   fmt.Sprintf("/tasks/assets/%s/%s", awsSaaSAssets, customer),
			Queue:  common.TaskQueueAssetsAWSSaaS,
		}

		payload := &UpdateSaaSAssetsRequest{
			Accounts: activeAccounts,
		}

		conf := config.Config(payload)

		if _, err = s.cloudTaskClient.CreateTask(ctx, conf); err != nil {
			logger.Errorf(createTaskErrTpl, customer, err)
			continue
		}
	}

	return nil
}

func (s *AWSSaaSConsoleOnboardService) UpdateSaaSAssets(ctx context.Context, customerID string, accounts []string) error {
	for _, accountID := range accounts {
		err := s.createAsset(ctx, customerID, accountID)
		if err != nil {
			return err
		}

		assetID := s.getAssetID(accountID)
		if err := s.assetsDAL.SetAssetMetadata(ctx, assetID, common.Assets.AmazonWebServicesStandalone); err != nil {
			return err
		}
	}

	accountID := accounts[0]

	return s.awsAssetsService.UpdateStandaloneAssets(ctx, customerID, accountID)
}
