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
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

const (
	standaloneCustomerID = "s5x8qLHg0EGIJVBFayAb"
	documentID           = "google-cloud-s5x8qLHg0EGIJVBFayAb"
	standaloneID         = "google-cloud-s5x8qLHg0EGIJVBFayAb" // "google-cloud-s5x8qLHg0EGIJVBFayAb.accounts.ORG_ID"
	// TODO fssa - standaloneID should have org id to support multiple accounts
)

func Test_updateError(t *testing.T) {
	type fields struct {
		Logger                *loggerMocks.ILogger
		FlexsaveStandaloneDAL *mocks.FlexsaveStandalone
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
	ctx = context.WithValue(ctx, flexsavestandalone.CustomerIDKey, standaloneCustomerID)

	tests := []struct {
		name   string
		args   *args
		on     func(*fields)
		out    *GCPStandaloneResponse
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
				f.FlexsaveStandaloneDAL.
					On("UpdateStandaloneOnboardingError", ctx, standaloneID, &onboardingTestError, step).
					Return(nil).
					Once()
				f.Logger.
					On("Errorf", mock.AnythingOfType("string"), mock.AnythingOfType("pkg.StandaloneOnboardingStep"), mock.AnythingOfType("*errors.errorString")).
					On("SetLabels", mock.Anything)
			},
			out: Failure,
			assert: func(t *testing.T, f *fields) {
				f.Logger.AssertNumberOfCalls(t, "Errorf", 1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				Logger:                &loggerMocks.ILogger{},
				FlexsaveStandaloneDAL: &mocks.FlexsaveStandalone{},
			}

			s := &GcpStandaloneOnboardingService{
				func(ctx context.Context) logger.ILogger {
					return f.Logger
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				f.FlexsaveStandaloneDAL,
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

			out := s.updateError(tt.args.Ctx, tt.args.Step, tt.args.Err)

			if tt.assert != nil {
				tt.assert(t, f)
			}

			assert.Equal(t, Failure, out)
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
		StandaloneID string
		MissingError error
	}

	testError := "test"
	missingSomethingError := errors.New("missing " + testError)
	// gcpID := "google-cloud-" + standaloneCustomerID
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
				standaloneID,
				missingSomethingError,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &GcpStandaloneOnboardingService{
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
				nil,
				nil,
				nil,
			}

			resDocumentID := s.getDocumentID(tt.args.CustomerID)
			resAssetID := s.getAssetID(tt.args.CustomerID)
			resStandaloneID := s.composeStandaloneID(tt.args.CustomerID, "")
			missingError := flexsavestandalone.GetMissingError(tt.args.Err)
			assert.Equal(t, resDocumentID, tt.out.DocumentID)
			assert.Equal(t, resStandaloneID, tt.out.StandaloneID)
			assert.Equal(t, resAssetID, tt.out.AssetID)
			assert.Equal(t, missingError, tt.out.MissingError)
		})
	}
}

func Test_getEntity(t *testing.T) {
	type fields struct {
		Logger                *loggerMocks.ILogger
		FlexsaveStandaloneDAL *mocks.FlexsaveStandalone
		CustomersDAL          *customerMocks.Customers
		EntitiesDAL           *mocks.Entities
	}

	type args struct {
		Ctx        context.Context
		CustomerID string
	}

	type results struct {
		EntityRef *firestore.DocumentRef
		Err       error
	}

	// ID := fmt.Sprintf("%s-%s", common.Assets.GoogleCloud, standaloneCustomerID)
	entityID := "entityID"
	priorityID := "priorityID"
	entityRef := firestore.DocumentRef{ID: entityID}
	entityValid := pkg.Entity{Active: true, PriorityID: priorityID}
	entityInvalid := pkg.Entity{Active: false, PriorityID: "nottheonewearelookingfor"}
	customer := common.Customer{
		Entities: []*firestore.DocumentRef{&entityRef, &entityRef, &entityRef},
	}
	customerError := errors.New("no customer")
	entityError := errors.New("no entity")
	priorityError := errors.New("no priority")
	ctx := context.Background()

	tests := []struct {
		name   string
		args   *args
		on     func(*fields)
		out    *results
		assert func(*testing.T, *fields)
	}{
		{
			name: "no customer",
			args: &args{
				ctx,
				standaloneCustomerID,
			},
			on: func(f *fields) {
				f.FlexsaveStandaloneDAL.
					On("GetStandalonePriorityID", ctx, standaloneID).
					Return(priorityID, nil).
					Once()
				f.CustomersDAL.
					On("GetCustomer", ctx, standaloneCustomerID).
					Return(nil, customerError).
					Once()
			},
			out: &results{
				nil,
				customerError,
			},
			assert: func(t *testing.T, f *fields) {
				f.FlexsaveStandaloneDAL.AssertNumberOfCalls(t, "GetStandalonePriorityID", 1)
				f.CustomersDAL.AssertNumberOfCalls(t, "GetCustomer", 1)
				f.EntitiesDAL.AssertNotCalled(t, "GetEntity")
			},
		},
		{
			name: "no entity",
			args: &args{
				ctx,
				standaloneCustomerID,
			},
			on: func(f *fields) {
				f.FlexsaveStandaloneDAL.
					On("GetStandalonePriorityID", ctx, standaloneID).
					Return(priorityID, nil).
					Once()
				f.CustomersDAL.
					On("GetCustomer", ctx, standaloneCustomerID).
					Return(&customer, nil).
					Once()
				f.EntitiesDAL.
					On("GetEntity", ctx, entityID).
					Return(nil, entityError).
					Once()
			},
			out: &results{
				nil,
				entityError,
			},
			assert: func(t *testing.T, f *fields) {
				f.FlexsaveStandaloneDAL.AssertNumberOfCalls(t, "GetStandalonePriorityID", 1)
				f.CustomersDAL.AssertNumberOfCalls(t, "GetCustomer", 1)
				f.EntitiesDAL.AssertNumberOfCalls(t, "GetEntity", 1)
			},
		},
		{
			name: "no valid entity no priority",
			args: &args{
				ctx,
				standaloneCustomerID,
			},
			on: func(f *fields) {
				f.FlexsaveStandaloneDAL.
					On("GetStandalonePriorityID", ctx, standaloneID).
					Return("", priorityError).
					Once()
				f.CustomersDAL.
					On("GetCustomer", ctx, standaloneCustomerID).
					Return(&customer, nil).
					Once()
				f.EntitiesDAL.
					On("GetEntity", ctx, entityID).
					Return(&entityInvalid, nil).
					Times(3)
			},
			out: &results{
				nil,
				flexsavestandalone.ErrorBillingProfile,
			},
			assert: func(t *testing.T, f *fields) {
				f.FlexsaveStandaloneDAL.AssertNumberOfCalls(t, "GetStandalonePriorityID", 1)
				f.CustomersDAL.AssertNumberOfCalls(t, "GetCustomer", 1)
				f.EntitiesDAL.AssertNumberOfCalls(t, "GetEntity", 3)
			},
		},
		{
			name: "no valid entity",
			args: &args{
				ctx,
				standaloneCustomerID,
			},
			on: func(f *fields) {
				f.FlexsaveStandaloneDAL.
					On("GetStandalonePriorityID", ctx, standaloneID).
					Return(priorityID, nil).
					Once()
				f.CustomersDAL.
					On("GetCustomer", ctx, standaloneCustomerID).
					Return(&customer, nil).
					Once()
				f.EntitiesDAL.
					On("GetEntity", ctx, entityID).
					Return(&entityInvalid, nil).
					Times(3)
			},
			out: &results{
				nil,
				flexsavestandalone.ErrorBillingProfile,
			},
			assert: func(t *testing.T, f *fields) {
				f.FlexsaveStandaloneDAL.AssertNumberOfCalls(t, "GetStandalonePriorityID", 1)
				f.CustomersDAL.AssertNumberOfCalls(t, "GetCustomer", 1)
				f.EntitiesDAL.AssertNumberOfCalls(t, "GetEntity", 3)
			},
		},
		{
			name: "valid selectedPriorityId",
			args: &args{
				ctx,
				standaloneCustomerID,
			},
			on: func(f *fields) {
				f.FlexsaveStandaloneDAL.
					On("GetStandalonePriorityID", ctx, standaloneID).
					Return(priorityID, nil).
					Once()
				f.CustomersDAL.
					On("GetCustomer", ctx, standaloneCustomerID).
					Return(&customer, nil).
					Once()
				f.EntitiesDAL.
					On("GetEntity", ctx, entityID).
					Return(&entityInvalid, nil).
					Times(2)
				f.EntitiesDAL.
					On("GetEntity", ctx, entityID).
					Return(&entityValid, nil).
					Once()
			},
			out: &results{
				&entityRef,
				nil,
			},
			assert: func(t *testing.T, f *fields) {
				f.FlexsaveStandaloneDAL.AssertNumberOfCalls(t, "GetStandalonePriorityID", 1)
				f.CustomersDAL.AssertNotCalled(t, "GetCustomer")
				f.EntitiesDAL.AssertNumberOfCalls(t, "GetEntity", 3)
			},
		},
		{
			name: "valid entity no priority",
			args: &args{
				ctx,
				standaloneCustomerID,
			},
			on: func(f *fields) {
				f.FlexsaveStandaloneDAL.
					On("GetStandalonePriorityID", ctx, standaloneID).
					Return("", priorityError).
					Once()
				f.CustomersDAL.
					On("GetCustomer", ctx, standaloneCustomerID).
					Return(&customer, nil).
					Once()
				f.EntitiesDAL.
					On("GetEntity", ctx, entityID).
					Return(&entityValid, nil).
					Times(3)
			},
			out: &results{
				&entityRef,
				nil,
			},
			assert: func(t *testing.T, f *fields) {
				f.FlexsaveStandaloneDAL.AssertNumberOfCalls(t, "GetStandalonePriorityID", 1)
				f.CustomersDAL.AssertNumberOfCalls(t, "GetCustomer", 1)
				f.EntitiesDAL.AssertNumberOfCalls(t, "GetEntity", 1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				Logger:                &loggerMocks.ILogger{},
				FlexsaveStandaloneDAL: &mocks.FlexsaveStandalone{},
				CustomersDAL:          &customerMocks.Customers{},
				EntitiesDAL:           &mocks.Entities{},
			}

			s := &GcpStandaloneOnboardingService{
				func(ctx context.Context) logger.ILogger {
					return f.Logger
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				f.FlexsaveStandaloneDAL,
				nil,
				f.EntitiesDAL,
				nil,
				nil,
				nil,
				f.CustomersDAL,
				nil,
			}

			if tt.on != nil {
				tt.on(f)
			}

			res, err := s.getEntity(tt.args.Ctx, tt.args.CustomerID, standaloneID)

			if tt.assert != nil {
				tt.assert(t, f)
			}

			if err != nil {
				assert.Nil(t, res)
				assert.Error(t, err)
				assert.Equal(t, tt.out.Err, err)
			} else {
				assert.NotNil(t, res)
				assert.Equal(t, res, tt.out.EntityRef)
				assert.NoError(t, err)
			}
		})
	}
}

func Test_AddContract(t *testing.T) {
	type fields struct {
		Logger                *loggerMocks.ILogger
		FlexsaveStandaloneDAL *mocks.FlexsaveStandalone
		CustomersDAL          *customerMocks.Customers
		EntitiesDAL           *mocks.Entities
		ContractsDAL          *mocks.Contracts
		AccountManagersDAL    *mocks.AccountManagers
	}

	type args struct {
		Ctx context.Context
		Req *flexsavestandalone.StandaloneContractRequest
	}

	req := &flexsavestandalone.StandaloneContractRequest{
		"ofir.cohen@doit-intl.com",
		standaloneCustomerID,
		"",
		2.0,
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
	ctxWithCustomerID := context.WithValue(ctx, flexsavestandalone.CustomerIDKey, standaloneCustomerID)
	testError := errors.New("test error")
	contractID := fmt.Sprintf("%s-%s", common.Assets.GoogleCloud, standaloneCustomerID)
	amID := "EwTS8g54q8TXi13eqkaC"
	amRef := firestore.DocumentRef{ID: amID}

	tests := []struct {
		name   string
		args   *args
		on     func(*fields)
		out    *GCPStandaloneResponse
		assert func(*testing.T, *fields)
	}{
		{
			name: "valid",
			args: &args{
				ctx,
				req,
			},
			on: func(f *fields) {
				f.FlexsaveStandaloneDAL.
					On("GetStandalonePriorityID", ctxWithCustomerID, standaloneID).
					Return(priorityID, nil).
					Once().
					On("AgreedStandaloneContract", ctxWithCustomerID, contractID, 2.0).
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
			out: Success,
			assert: func(t *testing.T, f *fields) {
				f.FlexsaveStandaloneDAL.AssertNumberOfCalls(t, "GetStandalonePriorityID", 0)
				f.CustomersDAL.AssertNumberOfCalls(t, "GetRef", 1)
				f.AccountManagersDAL.AssertNumberOfCalls(t, "GetRef", 1)
				f.FlexsaveStandaloneDAL.AssertNumberOfCalls(t, "AgreedStandaloneContract", 1)
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
				f.FlexsaveStandaloneDAL.
					On("GetStandalonePriorityID", ctxWithCustomerID, standaloneID).
					Return("", nil).
					Once().
					On("UpdateStandaloneOnboardingError", ctxWithCustomerID, standaloneID, mock.AnythingOfType("*pkg.StandaloneOnboardingError"), mock.AnythingOfType("pkg.StandaloneOnboardingStep")).
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
			out: Failure,
			assert: func(t *testing.T, f *fields) {
				f.AccountManagersDAL.AssertNumberOfCalls(t, "GetRef", 1)
				f.ContractsDAL.AssertNumberOfCalls(t, "Add", 1)
				f.FlexsaveStandaloneDAL.AssertNotCalled(t, "AgreedStandaloneContract")
				f.Logger.AssertNumberOfCalls(t, "Errorf", 1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				Logger:                &loggerMocks.ILogger{},
				FlexsaveStandaloneDAL: &mocks.FlexsaveStandalone{},
				CustomersDAL:          &customerMocks.Customers{},
				EntitiesDAL:           &mocks.Entities{},
				ContractsDAL:          &mocks.Contracts{},
				AccountManagersDAL:    &mocks.AccountManagers{},
			}

			s := &GcpStandaloneOnboardingService{
				func(ctx context.Context) logger.ILogger {
					return f.Logger
				},
				nil,
				nil,
				nil,
				nil,
				nil,
				f.FlexsaveStandaloneDAL,
				f.ContractsDAL,
				f.EntitiesDAL,
				nil,
				f.AccountManagersDAL,
				nil,
				f.CustomersDAL,
				nil,
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

// also tests flexsavestandalone.EnrichLogger
func Test_getLogger(t *testing.T) {
	type fields struct {
		Logger *loggerMocks.ILogger
	}

	type args struct {
		Ctx context.Context
	}

	loggerMock := &loggerMocks.ILogger{}
	ctx := context.Background()
	ctxWithCustomerID := context.WithValue(ctx, flexsavestandalone.CustomerIDKey, standaloneCustomerID)

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

			s := &GcpStandaloneOnboardingService{
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
