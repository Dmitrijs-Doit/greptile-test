package aws

import (
	"context"
	"errors"
	"testing"

	"cloud.google.com/go/firestore"
	cloudTaskClientMocks "github.com/doitintl/cloudtasks/mocks"
	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/mocks"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSaaSConsoleAWSService_UpdateAllSaaSAssets(t *testing.T) {
	ctx := context.Background()
	contextMock := mock.MatchedBy(func(_ context.Context) bool { return true })

	testCustomer := "aws-test-customer"
	testAccount := "aws-test-account"

	testCustomerRef := firestore.DocumentRef{ID: testCustomer}

	type fields struct {
		customerDal        *customerMocks.Customers
		saasConsoleDAL     *mocks.SaaSConsoleOnboard
		cloudConnectDAL    *mocks.CloudConnect
		cloudTaskClient    *cloudTaskClientMocks.CloudTaskClient
		loggerProviderMock loggerMocks.ILogger
	}

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr bool
	}{
		{
			name: "DAL error",
			on: func(f *fields) {
				f.saasConsoleDAL.On("GetAWSOnboardedAccountIDsByCustomer",
					contextMock).
					Return(nil, errors.New("dal error")).Once()
			},
			wantErr: true,
		},
		{
			name: "Create task fails",
			on: func(f *fields) {
				f.saasConsoleDAL.On("GetAWSOnboardedAccountIDsByCustomer",
					contextMock).
					Return(map[string][]string{testCustomer: {testAccount}}, nil).Once()
				f.customerDal.On("GetRef", contextMock, testCustomer).
					Return(&testCustomerRef).Once()
				f.cloudConnectDAL.On("GetAWSCloudConnect", contextMock, &testCustomerRef, common.Assets.AmazonWebServices, testAccount).
					Return(&pkg.AWSCloudConnect{BillingEtl: &pkg.BillingEtl{Settings: &pkg.BillingEtlSettings{Active: true}}}, nil).Once()
				f.cloudTaskClient.On("CreateTask", contextMock, mock.AnythingOfType("*iface.Config")).
					Return(nil, errors.New("create task error")).Once()
				f.loggerProviderMock.On("Errorf", createTaskErrTpl, testCustomer, mock.AnythingOfType("*errors.errorString")).Once()
			},
		},
		{
			name: "No active saas accounts - 1",
			on: func(f *fields) {
				f.saasConsoleDAL.On("GetAWSOnboardedAccountIDsByCustomer",
					contextMock).
					Return(map[string][]string{testCustomer: {testAccount}}, nil).Once()
				f.customerDal.On("GetRef", contextMock, testCustomer).
					Return(&testCustomerRef).Once()
				f.cloudConnectDAL.On("GetAWSCloudConnect", contextMock, &testCustomerRef, common.Assets.AmazonWebServices, testAccount).
					Return(nil, doitFirestore.ErrNotFound).Once()
			},
		},
		{
			name: "No active saas accounts - 2",
			on: func(f *fields) {
				f.saasConsoleDAL.On("GetAWSOnboardedAccountIDsByCustomer",
					contextMock).
					Return(map[string][]string{testCustomer: {testAccount}}, nil).Once()
				f.customerDal.On("GetRef", contextMock, testCustomer).
					Return(&testCustomerRef).Once()
				f.cloudConnectDAL.On("GetAWSCloudConnect", contextMock, &testCustomerRef, common.Assets.AmazonWebServices, testAccount).
					Return(&pkg.AWSCloudConnect{BillingEtl: &pkg.BillingEtl{Settings: &pkg.BillingEtlSettings{Active: false}}}, nil).Once()
			},
		},
		{
			name: "Happy path",
			on: func(f *fields) {
				f.saasConsoleDAL.On("GetAWSOnboardedAccountIDsByCustomer",
					contextMock).
					Return(map[string][]string{testCustomer: {testAccount}}, nil).Once()
				f.customerDal.On("GetRef", contextMock, testCustomer).
					Return(&testCustomerRef).Once()
				f.cloudConnectDAL.On("GetAWSCloudConnect", contextMock, &testCustomerRef, common.Assets.AmazonWebServices, testAccount).
					Return(&pkg.AWSCloudConnect{BillingEtl: &pkg.BillingEtl{Settings: &pkg.BillingEtlSettings{Active: true}}}, nil).Once()
				f.cloudTaskClient.On("CreateTask", contextMock, mock.AnythingOfType("*iface.Config")).
					Return(nil, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				customerDal:        customerMocks.NewCustomers(t),
				saasConsoleDAL:     mocks.NewSaaSConsoleOnboard(t),
				cloudConnectDAL:    mocks.NewCloudConnect(t),
				cloudTaskClient:    cloudTaskClientMocks.NewCloudTaskClient(t),
				loggerProviderMock: loggerMocks.ILogger{},
			}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &AWSSaaSConsoleOnboardService{
				customersDAL:    fields.customerDal,
				saasConsoleDAL:  fields.saasConsoleDAL,
				cloudConnectDAL: fields.cloudConnectDAL,
				cloudTaskClient: fields.cloudTaskClient,
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProviderMock
				},
			}

			err := s.UpdateAllSaaSAssets(ctx)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
