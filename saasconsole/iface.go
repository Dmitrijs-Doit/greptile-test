package saasconsole

import (
	"context"
	"time"

	"github.com/doitintl/firestore/pkg"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
)

type SlackUtils interface {
	PublishOnboardSuccessSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL customerDal.Customers, customerID, accountID string) error
	PublishBillingReadySuccessSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL customerDal.Customers, customerID, accountID string, trialEndDate time.Time) error
	PublishOnboardErrorSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL customerDal.Customers, customerID, accountID string, originalError error) error
	PublishBillingReadyErrorSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL customerDal.Customers, customerID, accountID string, originalError error) error
	PublishTrialSetErrorSlackNotification(ctx context.Context, customersDAL customerDal.Customers, customerID string, originalError error) error
	PublishBillingDataAlertSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL customerDal.Customers, customerID, accountID, issueMsd string, hours int) error
	PublishPermissionsAlertSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL customerDal.Customers, customerID, accountID string, status *pkg.AWSCloudConnectStatus, criticalPermissions []string) error
	PublishDeactivatedBillingSlackNotification(ctx context.Context, customersDAL customerDal.Customers, customersAccounts map[string]map[pkg.StandalonePlatform][]string) error
}
