package service

import (
	"context"
	"errors"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"testing"
	"time"

	azureMock "github.com/doitintl/azure/mocks"
	"github.com/doitintl/hello/scheduled-tasks/azure/dal"
	firestoreMock "github.com/doitintl/hello/scheduled-tasks/azure/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/azure/iface"
	customerMock "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	saasConsoleMock "github.com/doitintl/hello/scheduled-tasks/saasconsole/mocks"
	testUtils "github.com/doitintl/tests"
	"github.com/stretchr/testify/mock"
	"github.com/zeebo/assert"
)

func billingDataConfigMatcher(expected dal.BillingDataConfig) interface{} {
	return mock.MatchedBy(func(actual dal.BillingDataConfig) bool {
		return actual.CustomerID == expected.CustomerID &&
			actual.Container == expected.Container &&
			actual.Account == expected.Account &&
			actual.ResourceGroup == expected.ResourceGroup &&
			actual.SubscriptionID == expected.SubscriptionID &&
			actual.Directory == expected.Directory
	})
}

func Test_service_StoreBillingDataConnection(t *testing.T) {
	type fields struct {
		azureDAL     azureMock.Service
		firestoreDAL firestoreMock.FirestoreDAL
		customerDAL  customerMock.Customers
		saasConsole  saasConsoleMock.SlackUtils
	}

	type args struct {
		data iface.Payload
	}

	tests := []struct {
		name        string
		expectedErr string
		on          func(f *fields)
		args        args
	}{
		{
			name:        "return error when GetCustomerBillingDataConfigs fails",
			expectedErr: "billing-config-err",
			on: func(f *fields) {
				f.firestoreDAL.On("GetCustomerBillingDataConfigs", mock.Anything, "customer-1").Return(nil, errors.New("billing-config-err"))
			},
			args: args{},
		},

		{
			name:        "returns error if config exists",
			expectedErr: "Connection with these details already exist",
			on: func(f *fields) {
				f.firestoreDAL.On("GetCustomerBillingDataConfigs", mock.Anything, "customer-1").Return([]dal.BillingDataConfig{
					{
						CustomerID:     "customer-1",
						Container:      "container-1",
						Account:        "account-1",
						ResourceGroup:  "resource-group-1",
						SubscriptionID: "subscription-id-1",
						CreatedAt:      time.Time{},
					},
				}, nil)
			},
			args: args{
				data: iface.Payload{
					Account:        "account-1",
					Container:      "container-1",
					ResourceGroup:  "resource-group-1",
					SubscriptionID: "subscription-id-1",
				},
			},
		},

		{
			name:        "returns error unable to connect",
			expectedErr: "unable to connect",
			on: func(f *fields) {
				f.firestoreDAL.On("GetCustomerBillingDataConfigs", mock.Anything, "customer-1").Return([]dal.BillingDataConfig{}, nil)
				f.azureDAL.On("VerifyBillingDataConnection", mock.Anything, "subscription-id-1", "resource-group-1", "account-1", "container-1", "directory-1").
					Return("", errors.New("unable to connect"))
			},
			args: args{
				data: iface.Payload{
					Account:        "account-1",
					Container:      "container-1",
					ResourceGroup:  "resource-group-1",
					SubscriptionID: "subscription-id-1",
					Directory:      "directory-1",
				},
			},
		},

		{
			name:        "creating connection returns error",
			expectedErr: "unable to create",
			on: func(f *fields) {
				f.firestoreDAL.On("GetCustomerBillingDataConfigs", mock.Anything, "customer-1").Return([]dal.BillingDataConfig{}, nil)
				f.azureDAL.On("VerifyBillingDataConnection", mock.Anything, "subscription-id-1", "resource-group-1", "account-1", "container-1", "directory-1").
					Return("directory-1/subdirectory-1", nil)
				f.firestoreDAL.On("CreateCustomerBillingDataConfig", mock.Anything, billingDataConfigMatcher(dal.BillingDataConfig{
					CustomerID:     "customer-1",
					Container:      "container-1",
					Account:        "account-1",
					ResourceGroup:  "resource-group-1",
					SubscriptionID: "subscription-id-1",
					Directory:      "directory-1/subdirectory-1",
				})).Return(errors.New("unable to create"))
			},
			args: args{
				data: iface.Payload{
					Account:        "account-1",
					Container:      "container-1",
					ResourceGroup:  "resource-group-1",
					SubscriptionID: "subscription-id-1",
					Directory:      "directory-1",
				},
			},
		},

		{
			name:        "creating connection success",
			expectedErr: "",
			on: func(f *fields) {
				f.firestoreDAL.On("GetCustomerBillingDataConfigs", mock.Anything, "customer-1").Return([]dal.BillingDataConfig{}, nil)
				f.azureDAL.On("VerifyBillingDataConnection", mock.Anything, "subscription-id-1", "resource-group-1", "account-1", "container-1", "directory-1").
					Return("directory-1/subdirectory-1", nil)
				f.firestoreDAL.On("CreateCustomerBillingDataConfig", mock.Anything, billingDataConfigMatcher(dal.BillingDataConfig{
					CustomerID:     "customer-1",
					Container:      "container-1",
					Account:        "account-1",
					ResourceGroup:  "resource-group-1",
					SubscriptionID: "subscription-id-1",
					Directory:      "directory-1/subdirectory-1",
				})).Return(nil)
				f.customerDAL.On("UpdateCustomerFieldValueDeep", mock.Anything, "customer-1", []string{"enabledSaaSConsole", "AZURE"}, true).
					Return(nil)
				f.customerDAL.On("GetCustomer", mock.Anything, "customer-1").
					Return(&common.Customer{Name: "customer-1"}, nil)
				f.saasConsole.On("PublishOnboardSuccessSlackNotification", mock.Anything, pkg.AZURE, &f.customerDAL, "customer-1", "account-1").
					Return(nil)
			},
			args: args{
				data: iface.Payload{
					Account:        "account-1",
					Container:      "container-1",
					ResourceGroup:  "resource-group-1",
					SubscriptionID: "subscription-id-1",
					Directory:      "directory-1",
				},
			},
		},

		{
			name:        "updating customer field value returns error",
			expectedErr: "unable to update customer",
			on: func(f *fields) {
				f.firestoreDAL.On("GetCustomerBillingDataConfigs", mock.Anything, "customer-1").Return([]dal.BillingDataConfig{}, nil)
				f.azureDAL.On("VerifyBillingDataConnection", mock.Anything, "subscription-id-1", "resource-group-1", "account-1", "container-1", "directory-1").
					Return("directory-1/subdirectory-1", nil)
				f.firestoreDAL.On("CreateCustomerBillingDataConfig", mock.Anything, billingDataConfigMatcher(dal.BillingDataConfig{
					CustomerID:     "customer-1",
					Container:      "container-1",
					Account:        "account-1",
					ResourceGroup:  "resource-group-1",
					SubscriptionID: "subscription-id-1",
					Directory:      "directory-1/subdirectory-1",
				})).Return(nil)
				f.customerDAL.On("UpdateCustomerFieldValueDeep", mock.Anything, "customer-1", []string{"enabledSaaSConsole", "AZURE"}, true).
					Return(errors.New("unable to update customer"))
				f.saasConsole.AssertNotCalled(t, "PublishOnboardSuccessSlackNotification", mock.Anything, pkg.AZURE, &f.customerDAL, "customer-1", "account-1")
			},
			args: args{
				data: iface.Payload{
					Account:        "account-1",
					Container:      "container-1",
					ResourceGroup:  "resource-group-1",
					SubscriptionID: "subscription-id-1",
					Directory:      "directory-1",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{
				azureDAL:     azureMock.Service{},
				firestoreDAL: firestoreMock.FirestoreDAL{},
				customerDAL:  customerMock.Customers{},
				saasConsole:  saasConsoleMock.SlackUtils{},
			}

			tt.on(&f)

			s := &service{
				azureDAL:     &f.azureDAL,
				firestoreDAL: &f.firestoreDAL,
				customerDAL:  &f.customerDAL,
				slackUtils:   &f.saasConsole,
			}

			gotErr := s.StoreBillingDataConnection(context.Background(), "customer-1", tt.args.data)
			testUtils.WantErr(t, gotErr, tt.expectedErr)

			f.firestoreDAL.AssertExpectations(t)
			f.saasConsole.AssertExpectations(t)
			f.azureDAL.AssertExpectations(t)
		})
	}
}

func Test_service_GetStorageAccountNameForOnboarding(t *testing.T) {
	type fields struct {
		firestoreDAL firestoreMock.FirestoreDAL
	}

	tests := []struct {
		name        string
		expectedErr string
		on          func(f *fields)
		want        string
	}{
		{
			name:        "return error when GetCustomerBillingDataConfigs fails",
			want:        "",
			expectedErr: "billing-config-err",
			on: func(f *fields) {
				f.firestoreDAL.On("GetCustomerBillingDataConfigs", mock.Anything, "abcde").Return(nil, errors.New("billing-config-err"))
			},
		},

		{
			name:        "if no previous integrations exists, return customer id",
			want:        "abcde",
			expectedErr: "",
			on: func(f *fields) {
				f.firestoreDAL.On("GetCustomerBillingDataConfigs", mock.Anything, "abcde").Return([]dal.BillingDataConfig{}, nil)
			},
		},

		{
			name:        "if multiple previous integrations exists, return customer id with suffix",
			want:        "abcde2",
			expectedErr: "",
			on: func(f *fields) {
				f.firestoreDAL.On("GetCustomerBillingDataConfigs", mock.Anything, "abcde").Return([]dal.BillingDataConfig{{}}, nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{
				firestoreDAL: firestoreMock.FirestoreDAL{},
			}

			tt.on(&f)

			s := &service{
				firestoreDAL: &f.firestoreDAL,
			}

			got, gotErr := s.GetStorageAccountNameForOnboarding(context.Background(), "abcde")
			testUtils.WantErr(t, gotErr, tt.expectedErr)
			assert.Equal(t, got, tt.want)

			f.firestoreDAL.AssertExpectations(t)

		})
	}
}
