package mocks

import (
	"context"
	"time"

	"github.com/doitintl/firestore/pkg"

	"github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/stretchr/testify/mock"
)

type SlackUtils struct {
	mock.Mock
}

func (m *SlackUtils) PublishBillingReadySuccessSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL dal.Customers, customerID, accountID string, trialEndDate time.Time) error {
	args := m.Called(ctx, platform, customersDAL, customerID, accountID, trialEndDate)
	return args.Error(0)
}

func (m *SlackUtils) PublishOnboardErrorSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL dal.Customers, customerID, accountID string, originalError error) error {
	args := m.Called(ctx, platform, customersDAL, customerID, accountID, originalError)
	return args.Error(0)
}

func (m *SlackUtils) PublishBillingReadyErrorSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL dal.Customers, customerID, accountID string, originalError error) error {
	args := m.Called(ctx, platform, customersDAL, customerID, accountID, originalError)
	return args.Error(0)
}

func (m *SlackUtils) PublishTrialSetErrorSlackNotification(ctx context.Context, customersDAL dal.Customers, customerID string, originalError error) error {
	args := m.Called(ctx, customersDAL, customerID, originalError)
	return args.Error(0)
}

func (m *SlackUtils) PublishBillingDataAlertSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL dal.Customers, customerID, accountID, issueMsd string, hours int) error {
	args := m.Called(ctx, platform, customersDAL, customerID, accountID, issueMsd, hours)
	return args.Error(0)
}

func (m *SlackUtils) PublishPermissionsAlertSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL dal.Customers, customerID, accountID string, status *pkg.AWSCloudConnectStatus, criticalPermissions []string) error {
	args := m.Called(ctx, platform, customersDAL, customerID, accountID, status, criticalPermissions)
	return args.Error(0)
}

func (m *SlackUtils) PublishOnboardSuccessSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDal dal.Customers, customerID, account string) error {
	args := m.Called(ctx, platform, customersDal, customerID, account)
	return args.Error(0)
}

func (m *SlackUtils) PublishDeactivatedBillingSlackNotification(ctx context.Context, customersDal dal.Customers, customersAccounts map[string]map[pkg.StandalonePlatform][]string) error {
	args := m.Called(ctx, customersDal, customersAccounts)
	return args.Error(0)
}
