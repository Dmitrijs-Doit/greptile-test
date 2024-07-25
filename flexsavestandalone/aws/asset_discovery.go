package aws

import (
	"context"
	"fmt"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

const (
	createTaskErrTpl = "failed to create aws standalone asset discovery task for customer %s with error: %s"
)

type UpdateStandAloneAssetsRequest struct {
	Accounts []string `json:"accounts"`
}

func (s *AwsStandaloneService) UpdateAllStandAloneAssets(ctx context.Context) error {
	l := s.loggerProvider(ctx)

	fssaAccounts, err := s.flexsaveStandaloneDAL.GetAWSStandaloneOnboardedAccountIDsByCustomer(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch standalone accounts; %s", err)
	}

	for customer, accounts := range fssaAccounts {
		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   fmt.Sprintf("/tasks/assets/%s/%s", common.Assets.AmazonWebServicesStandalone, customer),
			Queue:  common.TaskQueueAssetsAWSStandAlone,
		}

		payload := &UpdateStandAloneAssetsRequest{
			Accounts: accounts,
		}

		conf := config.Config(payload)

		if _, err = s.cloudTaskClient.CreateTask(ctx, conf); err != nil {
			l.Errorf(createTaskErrTpl, customer, err)
			continue
		}
	}

	return nil
}

func (s *AwsStandaloneService) UpdateStandAloneAssets(ctx context.Context, customerID string, accounts []string) error {
	for _, accountID := range accounts {
		err := s.CreateAsset(ctx, customerID, accountID)
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
