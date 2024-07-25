package validator

import (
	"context"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"github.com/doitintl/firestore/pkg"
	reqIface "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

const (
	permissionsLogPrefix        string = "SaaS Console - Permissions Validator - "
	permissionsSlackErrorFormat string = "%scouldn't send permissions alert slack notification for customer %s, %s"
)

func (s *SaaSConsoleValidatorService) ValidatePermissions(ctx context.Context) error {
	logger := s.loggerProvider(ctx)

	logger.Debugf("%sStarted", permissionsLogPrefix)

	// if err := s.validateGCPPermissions(ctx); err != nil {
	// 	return err
	// }

	if err := s.validateAWSPermissions(ctx); err != nil {
		return err
	}

	logger.Debugf("%sFinished", permissionsLogPrefix)

	return nil
}

func (s *SaaSConsoleValidatorService) validateAWSPermissions(ctx context.Context) error {
	logger := s.loggerProvider(ctx)

	cloudConnects, err := s.cloudConnectDAL.GetAllAWSCloudConnect(ctx)
	if err != nil {
		logger.Errorf("%scouldn't fetch all standalone aws cloud connect docs, %s", billingLogPrefix, err)
	}

	for _, cloudConnect := range cloudConnects {
		customer, err := s.customersDAL.GetCustomer(ctx, cloudConnect.Customer.ID)
		if err != nil {
			logger.Errorf("%scouldn't fetch customer %s, %s", billingLogPrefix, cloudConnect.Customer.ID, err)
			continue
		}

		if customer.EnabledSaaSConsole == nil || !customer.EnabledSaaSConsole.AWS {
			continue
		}

		if active, _, err := s.tiersService.IsCustomerOnActiveTier(ctx, cloudConnect.Customer, []pkg.PackageTierType{pkg.NavigatorPackageTierType}); err != nil {
			logger.Errorf("%scouldn't check if customer %s is on active tier, %s", permissionsLogPrefix, cloudConnect.Customer.ID, err)
		} else if !active {
			continue
		}

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   "/tasks/billing-standalone/validate-aws-account",
			Queue:  common.TaskQueueAWSSaaSValidateConnection,
		}

		body := reqIface.ValidateSaaSRequest{
			CustomerID: cloudConnect.Customer.ID,
			AccountID:  cloudConnect.AccountID,
			RoleArn:    cloudConnect.Arn,
			CURBucket:  cloudConnect.BillingEtl.Settings.Bucket,
			CURPath:    cloudConnect.BillingEtl.Settings.CurBasePath,
		}

		if _, err := s.conn.CloudTaskClient.CreateTask(ctx, config.Config(body)); err != nil {
			logger.Errorf("%scouldn't create task for customer %s, account %s, %s", permissionsLogPrefix, cloudConnect.Customer.ID, cloudConnect.AccountID, err)
		}
	}

	return nil
}
