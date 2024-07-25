package aws

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-sdk-go/service/costandusagereportservice"
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
	standaloneCustomerAccountID = "023946476650"
	standaloneCustomerID        = "s5x8qLHg0EGIJVBFayAb"
	documentID                  = "amazon-web-services-s5x8qLHg0EGIJVBFayAb"
	standaloneID                = "amazon-web-services-s5x8qLHg0EGIJVBFayAb.accounts.023946476650"
)

func Test_validateSingleCUR(t *testing.T) {
	type fields struct {
		Logger *loggerMocks.ILogger
	}

	type args struct {
		Ctx      context.Context
		Report   *costandusagereportservice.ReportDefinition
		S3Bucket string
		CurPath  string
	}

	ctx := context.Background()
	prefix := "CUR"
	name := "doitintl-awsops102"
	prefixNameName := fmt.Sprintf("%s/%s/%s", prefix, name, name)
	prefixName := fmt.Sprintf("%s/%s", prefix, name)

	report1 := costandusagereportservice.ReportDefinition{
		S3Bucket:                 &name,
		ReportName:               &name,
		S3Prefix:                 &prefix,
		TimeUnit:                 &hourly,
		Format:                   &reportOrCsv,
		AdditionalSchemaElements: []*string{&resources},
	}

	report2 := costandusagereportservice.ReportDefinition{
		S3Bucket:                 &name,
		ReportName:               &name,
		S3Prefix:                 &prefixName,
		TimeUnit:                 &hourly,
		Format:                   &reportOrCsv,
		AdditionalSchemaElements: []*string{&resources},
	}

	tests := []struct {
		name string
		args *args
		out  bool
	}{
		{
			name: "report prefix = 'CUR', name = 'doitintl-awsops102'. given curPath is prefix 'CUR'",
			args: &args{
				ctx,
				&report1,
				name,
				prefix,
			},
			out: true,
		},
		{
			name: "report prefix = 'CUR', name = 'doitintl-awsops102'. given curPath is name 'doitintl-awsops102'",
			args: &args{
				ctx,
				&report1,
				name,
				name,
			},
			out: true,
		},
		{
			name: "report prefix = 'CUR', name = 'doitintl-awsops102'. given curPath is full path 'CUR/doitintl-awsops102'",
			args: &args{
				ctx,
				&report1,
				name,
				prefixNameName,
			},
			out: true,
		},
		{
			name: "report prefix = 'CUR/doitintl-awsops102', name = 'doitintl-awsops102'. given curPath is prefix 'CUR/doitintl-awsops102'",
			args: &args{
				ctx,
				&report2,
				name,
				prefixName,
			},
			out: true,
		},
		{
			name: "report prefix = 'CUR/doitintl-awsops102', name = 'doitintl-awsops102'. given curPath is name 'doitintl-awsops102'",
			args: &args{
				ctx,
				&report2,
				name,
				name,
			},
			out: true,
		},
		{
			name: "report prefix = 'CUR/doitintl-awsops102', name = 'doitintl-awsops102'. given curPath is full path 'CUR/doitintl-awsops102/doitintl-awsops102'",
			args: &args{
				ctx,
				&report2,
				name,
				prefixNameName,
			},
			out: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				Logger: &loggerMocks.ILogger{},
			}

			s := &AwsStandaloneService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return f.Logger
				},
				now: func() time.Time {
					return time.Now()
				},
			}
			valid, _ := s.validateSingleCUR(ctx, tt.args.Report, tt.args.S3Bucket, tt.args.CurPath)
			assert.Equal(t, tt.out, valid)
		})
	}
}

func Test_StackDeletion(t *testing.T) {
	type fields struct {
		Logger *loggerMocks.ILogger
	}

	type args struct {
		Ctx        context.Context
		CustomerID string
	}

	ctx := context.Background()

	tests := []struct {
		name   string
		args   *args
		on     func(*fields)
		assert func(*testing.T, *fields)
	}{
		{
			name: "valid",
			args: &args{
				ctx,
				standaloneCustomerID,
			},
			on: func(f *fields) {
				f.Logger.
					On("Infof", "%s for customer %s", stackDeletion, standaloneCustomerID).
					On("SetLabels", mock.Anything)

			},
			assert: func(t *testing.T, f *fields) {
				f.Logger.AssertNumberOfCalls(t, "Infof", 1)
			},
		},
		{
			name: "no customer",
			args: &args{
				ctx,
				"",
			},
			on: func(f *fields) {
				f.Logger.
					On("Infof", "%s for customer %s", stackDeletion, "").
					On("SetLabels", mock.Anything)
			},
			assert: func(t *testing.T, f *fields) {
				f.Logger.AssertNumberOfCalls(t, "Infof", 1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				Logger: &loggerMocks.ILogger{},
			}

			s := &AwsStandaloneService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return f.Logger
				},
			}

			if tt.on != nil {
				tt.on(f)
			}

			s.StackDeletion(tt.args.Ctx, tt.args.CustomerID)

			if tt.assert != nil {
				tt.assert(t, f)
			}
		})
	}
}

func Test_utils(t *testing.T) { //	test: s.getDocument, s.composeStandaloneID, s.getRoleArn, s.getPayerAccountName, s.validateFields
	type args struct {
		AccountID  string
		CustomerID string
		req        *AWSStandaloneRequest
	}

	type results struct {
		DocumentID       string
		StandaloneID     string
		RoleArn          string
		PayerAccountName string
		Err              error
	}

	tests := []struct {
		name string
		args *args
		out  *results
	}{
		{
			name: "valid",
			args: &args{
				standaloneCustomerAccountID,
				standaloneCustomerID,
				&AWSStandaloneRequest{
					AccountID:  standaloneCustomerAccountID,
					CustomerID: standaloneCustomerID,
				},
			},
			out: &results{
				"amazon-web-services-s5x8qLHg0EGIJVBFayAb",
				"amazon-web-services-s5x8qLHg0EGIJVBFayAb.accounts.023946476650",
				"arn:aws:iam::023946476650:role/doitintl_cmp",
				"standalone-payer-023946476650",
				nil,
			},
		},
		{
			name: "empty",
			args: &args{
				"",
				"",
				&AWSStandaloneRequest{},
			},
			out: &results{
				"",
				"",
				"arn:aws:iam:::role/doitintl_cmp",
				"standalone-payer-",
				flexsavestandalone.ErrorCustomerID,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AwsStandaloneService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return &loggerMocks.ILogger{}
				},
			}

			documentID := s.getDocumentID(tt.args.CustomerID)
			standaloneID := s.composeStandaloneID(tt.args.CustomerID, tt.args.AccountID)
			roleArn := getRoleArn(standaloneRole, tt.args.AccountID)
			payerAccountName := s.getPayerAccountName(tt.args.AccountID)
			err := s.validateFields(tt.args.req, false)

			assert.Equal(t, tt.out.DocumentID, documentID)
			assert.Equal(t, tt.out.StandaloneID, standaloneID)
			assert.Equal(t, tt.out.RoleArn, roleArn)
			assert.Equal(t, tt.out.PayerAccountName, payerAccountName)
			assert.Equal(t, tt.out.Err, err)

			if tt.out.Err != nil {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.NoError(t, err)
			}
		})
	}
}

func Test_updateError(t *testing.T) {
	type fields struct {
		Logger                *loggerMocks.ILogger
		FlexsaveStandaloneDAL *mocks.FlexsaveStandalone
	}

	type args struct {
		Ctx        context.Context
		CustomerID string
		AccountID  string
		Step       pkg.StandaloneOnboardingStep
		Err        error
	}

	testError := errors.New("test error")
	step := pkg.OnboardingStepSavings
	onboardingTestError := pkg.StandaloneOnboardingError{
		Type:    pkg.OnboardingErrorTypeGeneral,
		Message: testError.Error(),
	}

	onboardingTestErrorCUR := pkg.StandaloneOnboardingError{
		Type:    pkg.OnboardingErrorTypeCUR,
		Message: testError.Error(),
	}
	ctx := context.Background()

	tests := []struct {
		name   string
		args   *args
		on     func(*fields)
		assert func(*testing.T, *fields)
	}{
		{
			name: "valid - simple error",
			args: &args{
				ctx,
				standaloneCustomerID,
				standaloneCustomerAccountID,
				step,
				testError,
			},
			on: func(f *fields) {
				f.FlexsaveStandaloneDAL.
					On("UpdateStandaloneOnboardingError", ctx, standaloneID, &onboardingTestError, step).
					Return(nil).
					Once()
				f.Logger.
					On("Errorf", mock.Anything, mock.Anything, mock.Anything).
					On("SetLabels", mock.Anything)
			},
			assert: func(t *testing.T, f *fields) {
				f.FlexsaveStandaloneDAL.AssertNumberOfCalls(t, "UpdateStandaloneOnboardingError", 1)
				f.Logger.AssertNumberOfCalls(t, "Errorf", 1)
			},
		},
		{
			name: "valid - standalone error",
			args: &args{
				ctx,
				standaloneCustomerID,
				standaloneCustomerAccountID,
				step,
				&onboardingTestErrorCUR,
			},
			on: func(f *fields) {
				f.FlexsaveStandaloneDAL.
					On("UpdateStandaloneOnboardingError", ctx, standaloneID, &onboardingTestErrorCUR, step).
					Return(nil).
					Once()
				f.Logger.
					On("Errorf", mock.Anything, mock.Anything, mock.Anything).
					On("SetLabels", mock.Anything)
			},
			assert: func(t *testing.T, f *fields) {
				f.FlexsaveStandaloneDAL.AssertNumberOfCalls(t, "UpdateStandaloneOnboardingError", 1)
				f.Logger.AssertNumberOfCalls(t, "Errorf", 1)
			},
		},
		{
			name: "missing customerID",
			args: &args{
				ctx,
				"",
				"",
				step,
				testError,
			},
			on: func(f *fields) {
				f.Logger.
					On("Errorf", mock.Anything, mock.Anything, mock.Anything).
					On("SetLabels", mock.Anything)
			},
			assert: func(t *testing.T, f *fields) {
				f.FlexsaveStandaloneDAL.AssertNotCalled(t, "UpdateStandaloneOnboardingError")
				f.Logger.AssertNumberOfCalls(t, "Errorf", 1)
			},
		},
		{
			name: "DAL error",
			args: &args{
				ctx,
				standaloneCustomerID,
				standaloneCustomerAccountID,
				step,
				testError,
			},
			on: func(f *fields) {
				f.FlexsaveStandaloneDAL.
					On("UpdateStandaloneOnboardingError", ctx, standaloneID, &onboardingTestError, step).
					Return(flexsavestandalone.ErrorCustomerID).
					Once()
				f.Logger.
					On("Errorf", mock.Anything, mock.Anything, mock.Anything).
					On("Error", mock.Anything).
					On("SetLabels", mock.Anything)
			},
			assert: func(t *testing.T, f *fields) {
				f.FlexsaveStandaloneDAL.AssertNumberOfCalls(t, "UpdateStandaloneOnboardingError", 1)
				f.Logger.AssertNumberOfCalls(t, "Errorf", 1)
				f.Logger.AssertNumberOfCalls(t, "Error", 1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				Logger:                &loggerMocks.ILogger{},
				FlexsaveStandaloneDAL: &mocks.FlexsaveStandalone{},
			}

			s := &AwsStandaloneService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return f.Logger
				},
				flexsaveStandaloneDAL: f.FlexsaveStandaloneDAL,
			}

			if tt.on != nil {
				tt.on(f)
			}

			s.updateError(tt.args.Ctx, tt.args.CustomerID, tt.args.AccountID, tt.args.Step, tt.args.Err)

			if tt.assert != nil {
				tt.assert(t, f)
			}
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
		Ctx          context.Context
		CustomerID   string
		StandaloneID string
	}

	type results struct {
		EntityRef *firestore.DocumentRef
		Err       error
	}

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
				standaloneID,
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
				standaloneID,
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
				standaloneID,
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
				standaloneID,
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
				standaloneID,
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
				standaloneID,
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

			s := &AwsStandaloneService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return f.Logger
				},
				flexsaveStandaloneDAL: f.FlexsaveStandaloneDAL,
				entitiesDAL:           f.EntitiesDAL,
				customersDAL:          f.CustomersDAL,
			}

			if tt.on != nil {
				tt.on(f)
			}

			res, err := s.getEntity(tt.args.Ctx, tt.args.CustomerID, tt.args.StandaloneID)

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

func Test_validateCustomerAccountID(t *testing.T) {
	type fields struct {
		Logger                *loggerMocks.ILogger
		FlexsaveStandaloneDAL *mocks.FlexsaveStandalone
	}

	type args struct {
		Ctx        context.Context
		CustomerID string
		AccountID  string
	}

	ctx := context.Background()
	validDoc := pkg.AWSStandaloneOnboarding{
		AccountID: standaloneCustomerAccountID,
	}
	invalidDoc := pkg.AWSStandaloneOnboarding{
		AccountID: "bad AccountID",
	}
	errorNotFound := errors.New("not found")

	tests := []struct {
		name   string
		args   *args
		on     func(*fields)
		out    error
		assert func(*testing.T, *fields)
	}{
		{
			name: "no customer",
			args: &args{
				ctx,
				"",
				standaloneCustomerAccountID,
			},
			on: func(f *fields) {
				f.FlexsaveStandaloneDAL.
					On("GetAWSStandaloneOnboarding", ctx, "").
					Return(nil, flexsavestandalone.ErrorCustomerID).
					Once()
			},
			out: flexsavestandalone.ErrorCustomerID,
			assert: func(t *testing.T, f *fields) {
				f.FlexsaveStandaloneDAL.AssertNumberOfCalls(t, "GetAWSStandaloneOnboarding", 1)
			},
		},
		{
			name: "no account",
			args: &args{
				ctx,
				standaloneCustomerID,
				"",
			},
			on: func(f *fields) {
				f.FlexsaveStandaloneDAL.
					On("GetAWSStandaloneOnboarding", ctx, documentID).
					Return(&validDoc, nil).
					Once()
			},
			out: errorBadAccountID,
			assert: func(t *testing.T, f *fields) {
				f.FlexsaveStandaloneDAL.AssertNumberOfCalls(t, "GetAWSStandaloneOnboarding", 1)
			},
		},
		{
			name: "no doc",
			args: &args{
				ctx,
				standaloneCustomerID,
				standaloneCustomerAccountID,
			},
			on: func(f *fields) {
				f.FlexsaveStandaloneDAL.
					On("GetAWSStandaloneOnboarding", ctx, standaloneID).
					Return(nil, errorNotFound).
					Once()
			},
			out: errorNotFound,
			assert: func(t *testing.T, f *fields) {
				f.FlexsaveStandaloneDAL.AssertNumberOfCalls(t, "GetAWSStandaloneOnboarding", 1)
			},
		},
		{
			name: "invalid",
			args: &args{
				ctx,
				standaloneCustomerID,
				standaloneCustomerAccountID,
			},
			on: func(f *fields) {
				f.FlexsaveStandaloneDAL.
					On("GetAWSStandaloneOnboarding", ctx, standaloneID).
					Return(&invalidDoc, nil).
					Once()
			},
			out: errorBadAccountID,
			assert: func(t *testing.T, f *fields) {
				f.FlexsaveStandaloneDAL.AssertNumberOfCalls(t, "GetAWSStandaloneOnboarding", 1)
			},
		},
		{
			name: "valid",
			args: &args{
				ctx,
				standaloneCustomerID,
				standaloneCustomerAccountID,
			},
			on: func(f *fields) {
				f.FlexsaveStandaloneDAL.
					On("GetAWSStandaloneOnboarding", ctx, standaloneID).
					Return(&validDoc, nil).
					Once()
			},
			out: nil,
			assert: func(t *testing.T, f *fields) {
				f.FlexsaveStandaloneDAL.AssertNumberOfCalls(t, "GetAWSStandaloneOnboarding", 1)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				Logger:                &loggerMocks.ILogger{},
				FlexsaveStandaloneDAL: &mocks.FlexsaveStandalone{},
			}

			s := &AwsStandaloneService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return f.Logger
				},
				flexsaveStandaloneDAL: f.FlexsaveStandaloneDAL,
			}

			if tt.on != nil {
				tt.on(f)
			}

			err := s.validateCustomerAccountID(tt.args.Ctx, tt.args.CustomerID, tt.args.AccountID)

			if tt.assert != nil {
				tt.assert(t, f)
			}

			assert.Equal(t, tt.out, err)

			if err != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
