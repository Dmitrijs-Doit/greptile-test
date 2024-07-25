package saasconsole

import (
	"context"
	"time"

	"github.com/doitintl/firestore/pkg"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
)

type SlackUtilsFunctions struct{}

func (s *SlackUtilsFunctions) PublishOnboardSuccessSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL customerDal.Customers, customerID, accountID string) error {
	return PublishOnboardSuccessSlackNotification(ctx, platform, customersDAL, customerID, accountID)
}

func (s *SlackUtilsFunctions) PublishBillingReadySuccessSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL customerDal.Customers, customerID, accountID string, trialEndDate time.Time) error {
	return PublishBillingReadySuccessSlackNotification(ctx, platform, customersDAL, customerID, accountID, trialEndDate)
}

func (s *SlackUtilsFunctions) PublishOnboardErrorSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL customerDal.Customers, customerID, accountID string, originalError error) error {
	return PublishOnboardErrorSlackNotification(ctx, platform, customersDAL, customerID, accountID, originalError)
}

func (s *SlackUtilsFunctions) PublishBillingReadyErrorSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL customerDal.Customers, customerID, accountID string, originalError error) error {
	return PublishBillingReadyErrorSlackNotification(ctx, platform, customersDAL, customerID, accountID, originalError)
}

func (s *SlackUtilsFunctions) PublishTrialSetErrorSlackNotification(ctx context.Context, customersDAL customerDal.Customers, customerID string, originalError error) error {
	return PublishTrialSetErrorSlackNotification(ctx, customersDAL, customerID, originalError)
}

func (s *SlackUtilsFunctions) PublishBillingDataAlertSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL customerDal.Customers, customerID, accountID, issueMsd string, hours int) error {
	return PublishBillingDataAlertSlackNotification(ctx, platform, customersDAL, customerID, accountID, issueMsd, hours)
}

func (s *SlackUtilsFunctions) PublishPermissionsAlertSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL customerDal.Customers, customerID, accountID string, status *pkg.AWSCloudConnectStatus, criticalPermissions []string) error {
	return PublishPermissionsAlertSlackNotification(ctx, platform, customersDAL, customerID, accountID, status, criticalPermissions)
}

func (s *SlackUtilsFunctions) PublishDeactivatedBillingSlackNotification(ctx context.Context, customersDAL customerDal.Customers, customersAccounts map[string]map[pkg.StandalonePlatform][]string) error {
	return PublishDeactivatedBillingSlackNotification(ctx, customersDAL, customersAccounts)
}
