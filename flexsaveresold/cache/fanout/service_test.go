package fanout

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/cloudtasks/iface"
	ctmocks "github.com/doitintl/cloudtasks/mocks"
	firestoreerr "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/mocks"
	fspkg "github.com/doitintl/firestore/pkg"
	assetDalMocks "github.com/doitintl/hello/scheduled-tasks/assets/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	customerDalMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func Test_service_runComputeCache(t *testing.T) {
	sampleConfig := iface.Config{
		Project:             "doitintl-cmp-dev",
		Location:            "us-central1",
		QueueID:             "flexsave-cache",
		URL:                 "https://scheduled-tasks-dot-doitintl-cmp-dev.uc.r.appspot.com/tasks/flex-ri/cache/unicorn",
		Audience:            "scheduled-tasks",
		ServiceAccountEmail: "gcp-jobs@doitintl-cmp-dev.iam.gserviceaccount.com",
		Payload:             nil,
		HTTPMethod:          2,
		DispatchDeadline:    nil,
		ScheduleTime:        nil,
	}

	ref := firestore.DocumentRef{ID: "unicorn"}

	type fields struct {
		loggerProviderMock   loggerMocks.ILogger
		customersDalMock     customerDalMocks.Customers
		cloudTaskServiceMock ctmocks.CloudTaskClient
		assetsDalMock        assetDalMocks.Assets
		integrationsDal      mocks.Integrations
	}

	tests := []struct {
		name string
		on   func(*fields)
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)

				f.assetsDalMock.
					On("GetAWSStandaloneAssets", mock.Anything, &ref).
					Once().
					Return(nil, nil)

				f.integrationsDal.On("GetFlexsaveConfigurationCustomer", mock.Anything, "unicorn").Once().Return(&fspkg.FlexsaveConfiguration{
					AWS: fspkg.FlexsaveSavings{
						TimeDisabled: nil,
					},
				}, nil)

				f.cloudTaskServiceMock.
					On("CreateTask", mock.Anything, &sampleConfig).
					Once().
					Return(nil, nil).Once()
			},
		},
		{
			name: "task is not created with standalone assets failure",
			on: func(f *fields) {
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)

				f.assetsDalMock.
					On("GetAWSStandaloneAssets", mock.Anything, &ref).
					Once().
					Return(nil, errors.New("boom"))

				f.integrationsDal.On("GetFlexsaveConfigurationCustomer", mock.Anything, "unicorn").Once().Return(&fspkg.FlexsaveConfiguration{
					AWS: fspkg.FlexsaveSavings{
						TimeDisabled: nil,
					},
				}, nil)

				f.loggerProviderMock.On("Error", errors.New("boom"))
			},
		},
		{
			name: "unable to create task for customer",
			on: func(f *fields) {
				f.assetsDalMock.
					On("GetAWSStandaloneAssets", mock.Anything, &ref).
					Once().
					Return([]*pkg.AWSAsset{}, nil)

				f.cloudTaskServiceMock.
					On("CreateTask", mock.Anything, &sampleConfig).
					Once().
					Return(nil, errors.New("failed generating jobs for [unicorn]")).Once()

				f.integrationsDal.On("GetFlexsaveConfigurationCustomer", mock.Anything, "unicorn").Once().Return(&fspkg.FlexsaveConfiguration{
					AWS: fspkg.FlexsaveSavings{
						TimeDisabled: nil,
					},
				}, nil)

				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
				f.loggerProviderMock.On("Errorf", mock.Anything, mock.Anything)
			},
		},
		{
			name: "skips customer due to standalone asset",
			on: func(f *fields) {
				f.assetsDalMock.
					On("GetAWSStandaloneAssets", mock.Anything, &ref).Return([]*pkg.AWSAsset{
					&pkg.AWSAsset{
						BaseAsset: pkg.BaseAsset{
							AssetType: "amazon-web-services-standalone",
						},
					},
				}, nil)

				f.cloudTaskServiceMock.
					On("CreateTask", mock.Anything, &sampleConfig).
					Once().
					Return(nil, errors.New("failed generating jobs for [unicorn]"))

				f.integrationsDal.On("GetFlexsaveConfigurationCustomer", mock.Anything, "unicorn").Once().Return(&fspkg.FlexsaveConfiguration{
					AWS: fspkg.FlexsaveSavings{
						TimeDisabled: nil,
					},
				}, nil)

				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
			},
		},
		{
			name: "for disabled customer there is no need to create a task, so I guess sad path, lol",
			on: func(f *fields) {
				someTime := time.Date(2022, 5, 4, 0, 0, 0, 0, time.UTC)
				f.integrationsDal.On("GetFlexsaveConfigurationCustomer", mock.Anything, "unicorn").Once().Return(&fspkg.FlexsaveConfiguration{
					AWS: fspkg.FlexsaveSavings{
						TimeDisabled: &someTime,
					},
				}, nil)

				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)
			},
		},
		{
			name: "for customer with no cache created (err 'not found' returned from service), we still want to create a task",
			on: func(f *fields) {
				f.assetsDalMock.
					On("GetAWSStandaloneAssets", mock.Anything, &ref).
					Once().
					Return(nil, errors.New("boom"))

				f.cloudTaskServiceMock.
					On("CreateTask", mock.Anything, &sampleConfig).
					Once().
					Return(nil, nil)

				f.integrationsDal.On("GetFlexsaveConfigurationCustomer", mock.Anything, "unicorn").Once().Return(nil, firestoreerr.ErrNotFound)

				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)

				f.loggerProviderMock.On("Error", mock.Anything)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{
				customersDalMock:     customerDalMocks.Customers{},
				cloudTaskServiceMock: ctmocks.CloudTaskClient{},
				assetsDalMock:        assetDalMocks.Assets{},
				loggerProviderMock:   loggerMocks.ILogger{},
			}

			tt.on(&f)

			s := service{
				func(ctx context.Context) logger.ILogger {
					return &f.loggerProviderMock
				},
				&f.customersDalMock,
				&f.cloudTaskServiceMock,
				&f.assetsDalMock,
				&f.integrationsDal,
			}

			ctx := context.Background()

			s.runComputeCache(ctx, s.loggerProvider(ctx), []*firestore.DocumentRef{&ref})
		})
	}
}

func Test_service_CreateSavingsPlanCacheForAllCustomers(t *testing.T) {
	sampleConfig := iface.Config{
		Project:             "doitintl-cmp-dev",
		Location:            "us-central1",
		QueueID:             "flexsave-savings-plans-cache",
		URL:                 "https://scheduled-tasks-dot-doitintl-cmp-dev.uc.r.appspot.com/tasks/flex-ri/savings-plans-cache/mr_customer",
		Audience:            "scheduled-tasks",
		ServiceAccountEmail: "gcp-jobs@doitintl-cmp-dev.iam.gserviceaccount.com",
		Payload:             nil,
		HTTPMethod:          2,
		DispatchDeadline:    nil,
		ScheduleTime:        nil,
	}

	type fields struct {
		loggerProviderMock   loggerMocks.ILogger
		integrationsDalMock  mocks.Integrations
		cloudTaskServiceMock ctmocks.CloudTaskClient
	}

	ctx := context.Background()

	tests := []struct {
		name         string
		assertResult func(assert.TestingT, error)
		on           func(*fields)
	}{
		{
			name: "task is created",
			on: func(f *fields) {
				ids := []string{"mr_customer"}

				f.integrationsDalMock.
					On("GetAWSEligibleCustomerIDs", ctx).
					Once().
					Return(ids, nil)

				f.cloudTaskServiceMock.
					On("CreateTask", mock.Anything, &sampleConfig).
					Once().
					Return(nil, nil)

				f.loggerProviderMock.On("Infof", "running savings plan cache job for %d customers", 1)

			},
			assertResult: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unable to get customers",
			on: func(f *fields) {
				f.integrationsDalMock.
					On("GetAWSEligibleCustomerIDs", ctx).
					Once().
					Return(nil, fmt.Errorf("oh dear"))
			},
			assertResult: func(t assert.TestingT, err error) {
				assert.ErrorContains(t, err, "oh dear")
			},
		},

		{
			name: "unable to create task for customer",
			on: func(f *fields) {
				ids := []string{"mr_customer"}
				f.integrationsDalMock.
					On("GetAWSEligibleCustomerIDs", ctx).
					Once().
					Return(ids, nil)
				f.loggerProviderMock.On("Infof", "running savings plan cache job for %d customers", 1)
				f.cloudTaskServiceMock.
					On("CreateTask", mock.Anything, &sampleConfig).
					Once().
					Return(nil, errors.New("failed generating jobs for [unicorn]")).Once()

				f.loggerProviderMock.On("Errorf", "unable to create savings plan cache generation task for customer: %s", "mr_customer")

			},
			assertResult: func(t assert.TestingT, err error) {
				assert.ErrorContains(t, err, "failed generating jobs for mr_customer")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{
				integrationsDalMock:  mocks.Integrations{},
				cloudTaskServiceMock: ctmocks.CloudTaskClient{},
				loggerProviderMock:   loggerMocks.ILogger{},
			}

			tt.on(&f)

			s := service{
				func(ctx context.Context) logger.ILogger {
					return &f.loggerProviderMock
				},
				nil,
				&f.cloudTaskServiceMock,
				nil,
				&f.integrationsDalMock,
			}

			result := s.CreateSavingsPlansCacheForAllCustomers(context.Background())

			tt.assertResult(t, result)
		})
	}
}

func Test_service_runRDSCache(t *testing.T) {
	var (
		ctx     = context.Background()
		someErr = errors.New("something went wrong")

		sampleConfig = iface.Config{
			Project:             "doitintl-cmp-dev",
			Location:            "us-central1",
			QueueID:             "flexsave-cache",
			URL:                 "https://scheduled-tasks-dot-doitintl-cmp-dev.uc.r.appspot.com/tasks/flexsave-rds/run-cache/unicorn",
			Audience:            "scheduled-tasks",
			ServiceAccountEmail: "gcp-jobs@doitintl-cmp-dev.iam.gserviceaccount.com",
			Payload:             nil,
			HTTPMethod:          2,
			DispatchDeadline:    nil,
			ScheduleTime:        nil,
		}

		ref = firestore.DocumentRef{ID: "unicorn"}
	)

	type fields struct {
		loggerProviderMock   loggerMocks.ILogger
		customersDalMock     customerDalMocks.Customers
		cloudTaskServiceMock ctmocks.CloudTaskClient
		assetsDalMock        assetDalMocks.Assets
		integrationsDal      mocks.Integrations
	}

	tests := []struct {
		name string
		on   func(*fields)
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)

				f.cloudTaskServiceMock.On("CreateTask", ctx, &sampleConfig).Return(nil, nil)
			},
		},
		{
			name: "failed to create task",
			on: func(f *fields) {
				f.loggerProviderMock.On("Infof", mock.Anything, mock.Anything)

				f.cloudTaskServiceMock.On("CreateTask", ctx, &sampleConfig).Return(nil, someErr)

				f.loggerProviderMock.On("Errorf", mock.Anything, ref.ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{
				customersDalMock:     customerDalMocks.Customers{},
				cloudTaskServiceMock: ctmocks.CloudTaskClient{},
				assetsDalMock:        assetDalMocks.Assets{},
				loggerProviderMock:   loggerMocks.ILogger{},
			}

			tt.on(&f)

			s := service{
				func(ctx context.Context) logger.ILogger {
					return &f.loggerProviderMock
				},
				&f.customersDalMock,
				&f.cloudTaskServiceMock,
				&f.assetsDalMock,
				&f.integrationsDal,
			}

			s.runRDSCache(ctx, s.loggerProvider(ctx), []*firestore.DocumentRef{&ref})
		})
	}
}

func Test_service_CreateCacheForAllCustomers(t *testing.T) {
	var (
		ctx       = context.Background()
		customer1 = "customer-1"
		someErr   = errors.New("something wrong")
	)

	snapsMock := []*firestore.DocumentSnapshot{
		{Ref: &firestore.DocumentRef{ID: customer1}},
	}

	type fields struct {
		loggerProvider   loggerMocks.ILogger
		customersDAL     customerDalMocks.Customers
		cloudTaskService ctmocks.CloudTaskClient
		assetsDal        assetDalMocks.Assets
		integrationsDAL  mocks.Integrations
	}

	tests := []struct {
		name    string
		fields  fields
		on      func(*fields)
		wantErr error
	}{
		{
			name: "happy path - aws customers snaps obtained successfully",
			on: func(f *fields) {
				f.customersDAL.On("GetAWSCustomers", ctx).Return(snapsMock, nil).Once()

				f.loggerProvider.On("Infof", mock.AnythingOfType("string"), mock.AnythingOfType("int")).Times(3)

				f.assetsDal.
					On("GetAWSStandaloneAssets", mock.Anything, snapsMock[0].Ref).
					Once().
					Return(nil, nil).Once()

				f.integrationsDAL.On("GetFlexsaveConfigurationCustomer", mock.Anything, snapsMock[0].Ref.ID).Once().Return(&fspkg.FlexsaveConfiguration{
					AWS: fspkg.FlexsaveSavings{
						TimeDisabled: nil,
					},
				}, nil).Once()

				f.cloudTaskService.
					On("CreateTask", mock.Anything, mock.MatchedBy(func(arg *iface.Config) bool {
						assert.Contains(t, arg.URL, snapsMock[0].Ref.ID)
						return true
					})).
					Return(nil, nil).Times(3)
			},
		},
		{
			name: "failed to obtain customer snaps",
			on: func(f *fields) {
				f.customersDAL.On("GetAWSCustomers", ctx).Return(snapsMock, someErr).Once()
			},
			wantErr: someErr,
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := service{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &fields.loggerProvider
				},
				customersDAL:     &fields.customersDAL,
				cloudTaskService: &fields.cloudTaskService,
				assetsDal:        &fields.assetsDal,
				integrationsDAL:  &fields.integrationsDAL,
			}

			err := s.CreateCacheForAllCustomers(ctx)
			if err != nil {
				assert.ErrorContains(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
			}
		})
	}
}
