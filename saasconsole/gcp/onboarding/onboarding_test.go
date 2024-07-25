package onboarding

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/firestore/mocks"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole"
)

const (
	standaloneCustomerID = "s5x8qLHg0EGIJVBFayAb"
	documentID           = "google-cloud-s5x8qLHg0EGIJVBFayAb" // "google-cloud-<standaloneCustomerID>
	accountID            = "AAAAAA-BBBBBB-CCCCC"
	standaloneID         = "google-cloud-s5x8qLHg0EGIJVBFayAb.accounts.AAAAAA-BBBBBB-CCCCC" // "google-cloud-<standaloneCustomerID>.<accountID>"
)

func Test_updateError(t *testing.T) {
	type fields struct {
		Logger                *loggerMocks.ILogger
		SaaSConsoleOnboardDAL *mocks.SaaSConsoleOnboard
	}

	type args struct {
		Ctx  context.Context
		Step pkg.StandaloneOnboardingStep
		Err  error
	}

	testError := errors.New("test error")
	step := pkg.OnboardingStepSavings
	onboardingTestError := pkg.StandaloneOnboardingError{
		Type:    pkg.OnboardingErrorTypeGeneral,
		Message: testError.Error(),
	}
	ctx := context.Background()
	ctx = context.WithValue(ctx, saasconsole.CustomerIDKey, standaloneCustomerID)

	tests := []struct {
		name   string
		args   *args
		on     func(*fields)
		out    *saasconsole.OnboardingResponse
		assert func(*testing.T, *fields)
	}{
		{
			name: "test",
			args: &args{
				ctx,
				step,
				testError,
			},
			on: func(f *fields) {
				f.SaaSConsoleOnboardDAL.
					On("UpdateGCPOnboardingError", ctx, standaloneID, &onboardingTestError, step).
					Return(nil).
					Once()
				f.Logger.
					On("Errorf", mock.AnythingOfType("string"), mock.AnythingOfType("pkg.StandaloneOnboardingStep"), mock.AnythingOfType("*errors.errorString")).
					On("SetLabels", mock.Anything)
			},
			out: saasconsole.Failure,
			assert: func(t *testing.T, f *fields) {
				f.Logger.AssertNumberOfCalls(t, "Errorf", 1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				Logger:                &loggerMocks.ILogger{},
				SaaSConsoleOnboardDAL: &mocks.SaaSConsoleOnboard{},
			}

			s := &GCPSaaSConsoleOnboardService{
				func(ctx context.Context) logger.ILogger {
					return f.Logger
				},
				nil,
				nil,
				nil,
				f.SaaSConsoleOnboardDAL,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
			}

			if tt.on != nil {
				tt.on(f)
			}

			out := s.updateError(tt.args.Ctx, standaloneCustomerID, accountID, tt.args.Step, tt.args.Err)

			if tt.assert != nil {
				tt.assert(t, f)
			}

			assert.Equal(t, saasconsole.Failure, out)
		})
	}
}

func Test_utils(t *testing.T) {
	type args struct {
		Err        string
		CustomerID string
	}

	type results struct {
		DocumentID   string
		AssetID      string
		MissingError error
	}

	testError := "test"
	missingSomethingError := errors.New("missing " + testError)
	assetID := "google-cloud-standalone-" + standaloneCustomerID

	tests := []struct {
		name string
		args *args
		out  *results
	}{
		{
			name: "test",
			args: &args{
				testError,
				standaloneCustomerID,
			},
			out: &results{
				documentID,
				assetID,
				missingSomethingError,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &GCPSaaSConsoleOnboardService{
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
			}

			resDocumentID := s.getDocumentID(tt.args.CustomerID)
			resAssetID := s.getAssetID(tt.args.CustomerID)
			missingError := saasconsole.GetMissingError(tt.args.Err)
			assert.Equal(t, resDocumentID, tt.out.DocumentID)
			assert.Equal(t, resAssetID, tt.out.AssetID)
			assert.Equal(t, missingError, tt.out.MissingError)
		})
	}
}

func Test_AddContract(t *testing.T) {
	type fields struct {
		Logger                *loggerMocks.ILogger
		SaaSConsoleOnboardDAL *mocks.SaaSConsoleOnboard
		CustomersDAL          *customerMocks.Customers
		EntitiesDAL           *mocks.Entities
		ContractsDAL          *mocks.Contracts
		AccountManagersDAL    *mocks.AccountManagers
	}

	type args struct {
		Ctx context.Context
		Req *saasconsole.StandaloneContractRequest
	}

	req := &saasconsole.StandaloneContractRequest{
		Email:           "ofir.cohen@doit-intl.com",
		CustomerID:      standaloneCustomerID,
		AccountID:       "AAAAAA-BBBBBB-CCCCC",
		ContractVersion: 2.0,
	}

	// ID := fmt.Sprintf("%s-%s", common.Assets.GoogleCloud, standaloneCustomerID)
	entityID := "entityID"
	priorityID := "priorityID"
	entityRef := firestore.DocumentRef{ID: entityID}
	customer := common.Customer{
		Entities: []*firestore.DocumentRef{&entityRef, &entityRef, &entityRef},
	}
	customerRef := firestore.DocumentRef{}
	ctx := context.Background()
	ctxWithCustomerID := context.WithValue(ctx, saasconsole.CustomerIDKey, standaloneCustomerID)
	testError := errors.New("test error")
	contractID := fmt.Sprintf("%s-%s", common.Assets.GoogleCloud, standaloneCustomerID)
	amID := "bNfRTZiE2a4eFRiyWbQA"
	amRef := firestore.DocumentRef{ID: amID}

	tests := []struct {
		name   string
		args   *args
		on     func(*fields)
		out    *saasconsole.OnboardingResponse
		assert func(*testing.T, *fields)
	}{
		{
			name: "valid",
			args: &args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.SaaSConsoleOnboardDAL.
					On("GetStandalonePriorityID", ctxWithCustomerID, standaloneID).
					Return(priorityID, nil).
					Once().
					On("AgreedContract", ctxWithCustomerID, contractID, 2.0).
					Return(nil).
					Once()
				f.CustomersDAL.
					On("GetCustomer", ctxWithCustomerID, standaloneCustomerID).
					Return(&customer, nil).
					Once().
					On("GetRef", ctxWithCustomerID, standaloneCustomerID).
					Return(&customerRef).
					Once()
				f.AccountManagersDAL.
					On("GetRef", amID).
					Return(&amRef).
					Once()
				f.ContractsDAL.
					On("Add", ctxWithCustomerID, mock.AnythingOfType("*pkg.Contract")).
					Return(nil).
					Once()
				f.Logger.On("SetLabels", mock.Anything)
			},
			out: saasconsole.Success,
			assert: func(t *testing.T, f *fields) {
				f.SaaSConsoleOnboardDAL.AssertNumberOfCalls(t, "GetStandalonePriorityID", 0)
				f.CustomersDAL.AssertNumberOfCalls(t, "GetRef", 1)
				f.AccountManagersDAL.AssertNumberOfCalls(t, "GetRef", 1)
				f.SaaSConsoleOnboardDAL.AssertNumberOfCalls(t, "AgreedContract", 1)
				f.ContractsDAL.AssertNumberOfCalls(t, "Add", 1)
				f.Logger.AssertNumberOfCalls(t, "Errorf", 0)
			},
		},
		{
			name: "contracts DAL failure",
			args: &args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.SaaSConsoleOnboardDAL.
					On("GetStandalonePriorityID", ctxWithCustomerID, standaloneID).
					Return("", nil).
					Once().
					On("UpdateGCPOnboardingError", ctxWithCustomerID, standaloneID, mock.AnythingOfType("*pkg.StandaloneOnboardingError"), mock.AnythingOfType("pkg.StandaloneOnboardingStep")).
					Return(nil).
					Once()
				f.CustomersDAL.
					On("GetCustomer", ctxWithCustomerID, standaloneCustomerID).
					Return(&customer, nil).
					Twice().
					On("GetRef", ctxWithCustomerID, standaloneCustomerID).
					Return(&customerRef).
					Once()
				f.AccountManagersDAL.
					On("GetRef", amID).
					Return(&amRef).
					Once()
				f.ContractsDAL.
					On("Add", ctxWithCustomerID, mock.AnythingOfType("*pkg.Contract")).
					Return(testError).
					Once()
				f.Logger.
					On("Errorf", mock.AnythingOfType("string"), mock.AnythingOfType("pkg.StandaloneOnboardingStep"), mock.AnythingOfType("*errors.errorString")).
					On("SetLabels", mock.Anything)
			},
			out: saasconsole.Failure,
			assert: func(t *testing.T, f *fields) {
				f.AccountManagersDAL.AssertNumberOfCalls(t, "GetRef", 1)
				f.ContractsDAL.AssertNumberOfCalls(t, "Add", 1)
				f.SaaSConsoleOnboardDAL.AssertNotCalled(t, "AgreedContract")
				f.Logger.AssertNumberOfCalls(t, "Errorf", 1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				Logger:                &loggerMocks.ILogger{},
				SaaSConsoleOnboardDAL: &mocks.SaaSConsoleOnboard{},
				CustomersDAL:          &customerMocks.Customers{},
				EntitiesDAL:           &mocks.Entities{},
				ContractsDAL:          &mocks.Contracts{},
				AccountManagersDAL:    &mocks.AccountManagers{},
			}

			s := &GCPSaaSConsoleOnboardService{
				func(ctx context.Context) logger.ILogger {
					return f.Logger
				},
				nil,
				nil,
				nil,
				f.SaaSConsoleOnboardDAL,
				f.ContractsDAL,
				f.EntitiesDAL,
				nil,
				f.AccountManagersDAL,
				nil,
				f.CustomersDAL,
			}

			if tt.on != nil {
				tt.on(f)
			}

			out := s.AddContract(tt.args.Ctx, tt.args.Req)

			if tt.assert != nil {
				tt.assert(t, f)
			}

			assert.Equal(t, out, tt.out)
		})
	}
}

// also tests saasconsole.EnrichLogger
func Test_getLogger(t *testing.T) {
	type fields struct {
		Logger *loggerMocks.ILogger
	}

	type args struct {
		Ctx context.Context
	}

	loggerMock := &loggerMocks.ILogger{}
	ctx := context.Background()
	ctxWithCustomerID := context.WithValue(ctx, saasconsole.CustomerIDKey, standaloneCustomerID)

	tests := []struct {
		name   string
		args   *args
		on     func(*fields)
		out    logger.ILogger
		assert func(*testing.T, *fields)
	}{
		{
			name: "no customer id",
			args: &args{
				ctx,
			},
			on: func(f *fields) {
				f.Logger.
					On("SetLabels", mock.Anything).
					Once()
			},
			out: loggerMock,
			assert: func(t *testing.T, f *fields) {
			},
		},
		{
			name: "with customer ID",
			args: &args{
				ctxWithCustomerID,
			},
			on: func(f *fields) {
				f.Logger.
					On("SetLabels", mock.Anything).
					Once()
			},
			out: loggerMock,
			assert: func(t *testing.T, f *fields) {
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				Logger: loggerMock,
			}

			s := &GCPSaaSConsoleOnboardService{
				func(ctx context.Context) logger.ILogger {
					return f.Logger
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
			}

			if tt.on != nil {
				tt.on(f)
			}

			logger := s.getLogger(tt.args.Ctx)

			if tt.assert != nil {
				tt.assert(t, f)
			}

			assert.Equal(t, tt.out, logger)
			assert.NotNil(t, logger)
		})
	}
}
