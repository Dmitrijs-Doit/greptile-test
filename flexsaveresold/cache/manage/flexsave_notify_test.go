package manage

import (
	"context"
	"errors"
	"fmt"
	monitoringDomain "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/monitoring/domain"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func Test_flexsaveManageNotify_SendActivatedNotification(t *testing.T) {
	type fields struct {
		customersDAL customerMocks.Customers
		Connection   *connection.Connection
	}

	defaultErr := errors.New("some error")
	ctx := context.Background()
	customerID := "0123abcdef"

	hourlyCommitment := 1.0

	tests := []struct {
		name             string
		on               func(*fields)
		config           *pkg.FlexsaveConfiguration
		wantErr          error
		hourlyCommitment *float64
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.customersDAL.On("GetCustomer", ctx, customerID).Return(&common.Customer{}, nil)
			},
			config: &pkg.FlexsaveConfiguration{
				AWS: pkg.FlexsaveSavings{},
			},
			hourlyCommitment: &hourlyCommitment,
		},
		{
			name: "failed to get customer",
			on: func(f *fields) {
				f.customersDAL.On("GetCustomer", ctx, customerID).Return(&common.Customer{}, defaultErr)
			},
			wantErr: defaultErr,
			config: &pkg.FlexsaveConfiguration{
				AWS: pkg.FlexsaveSavings{},
			},
		},
		{
			name: "no hourly commitment",
			on: func(f *fields) {
				f.customersDAL.On("GetCustomer", ctx, customerID).Return(&common.Customer{}, defaultErr)
			},
			wantErr: defaultErr,
			config: &pkg.FlexsaveConfiguration{
				AWS: pkg.FlexsaveSavings{},
			},
			hourlyCommitment: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			logging, err := logger.NewLogging(ctx)
			if err != nil {
				assert.NoError(t, err)
			}

			conn, err := connection.NewConnection(ctx, logging)
			if err != nil {
				assert.NoError(t, err)
			}

			s := &flexsaveManageNotify{
				loggerProvider: logger.FromContext,
				customersDAL:   &fields.customersDAL,
				Connection:     conn,
			}
			err = s.SendActivatedNotification(ctx, customerID, tt.hourlyCommitment, []string{"111"})

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_flexsaveManageNotify_NotifyAboutPayerConfigSet(t *testing.T) {
	type fields struct {
	}

	type args struct {
		ctx           context.Context
		primaryDomain string
		accountID     string
	}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr error
	}{
		{
			name: "happy path - notification created",
			args: args{
				ctx:           context.Background(),
				primaryDomain: "primary-domain-test",
				accountID:     "000112",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &flexsaveManageNotify{}
			err := s.NotifyAboutPayerConfigSet(tt.args.ctx, tt.args.primaryDomain, tt.args.accountID)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_flexsaveManageNotify_SendWelcomeEmail(t *testing.T) {
	type fields struct {
		customersDAL customerMocks.Customers
		emailService mocks.EmailInterface
	}

	var am []*common.AccountManager

	defaultErr := errors.New("some error")
	ctx := context.Background()
	customerID := "0123abcdef"

	logging, err := logger.NewLogging(ctx)
	if err != nil {
		assert.NoError(t, err)
	}

	conn, err := connection.NewConnection(ctx, logging)
	if err != nil {
		assert.NoError(t, err)
	}

	ref := conn.Firestore(ctx).Collection("customers").Doc(customerID)

	params := types.WelcomeEmailParams{
		CustomerID:  customerID,
		Cloud:       common.AWS,
		Marketplace: false,
	}

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr error
	}{
		{
			name: "happy path - email sent",
			on: func(f *fields) {
				f.customersDAL.On("GetCustomer", context.Background(), customerID).Return(&common.Customer{}, nil)
				f.customersDAL.On("GetRef", context.Background(), customerID).Return(ref)
				f.emailService.On("SendWelcomeEmail", context.Background(), &params, []*common.User{}, am).Return(nil)
			},
		},
		{
			name: "failed to get customer",
			on: func(f *fields) {
				f.customersDAL.On("GetCustomer", context.Background(), customerID).Return(&common.Customer{}, defaultErr)
				f.customersDAL.On("GetRef", context.Background(), customerID).Return(ref, nil)
				f.emailService.On("SendWelcomeEmail", context.Background(), &params, []*common.User{}, am).Return(nil)
			},
			wantErr: defaultErr,
		},
		{
			name: "failed to send welcome email",
			on: func(f *fields) {
				f.customersDAL.On("GetCustomer", context.Background(), customerID).Return(&common.Customer{}, nil)
				f.customersDAL.On("GetRef", context.Background(), customerID).Return(ref, nil)
				f.emailService.On("SendWelcomeEmail", context.Background(), &params, []*common.User{}, am).Return(defaultErr)
			},
			wantErr: defaultErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &flexsaveManageNotify{
				loggerProvider: logger.FromContext,
				customersDAL:   &fields.customersDAL,
				emailService:   &fields.emailService,
				Connection:     conn,
			}

			err = s.SendWelcomeEmail(ctx, customerID)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_flexsaveManageNotify_SendSageMakerActivatedNotification(t *testing.T) {
	type fields struct {
		customersDAL customerMocks.Customers
		Connection   *connection.Connection
	}

	defaultErr := errors.New("some error")
	ctx := context.Background()
	customerID := "0123abcdef"

	tests := []struct {
		name    string
		on      func(*fields)
		config  *pkg.FlexsaveConfiguration
		wantErr error
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.customersDAL.On("GetCustomer", ctx, customerID).Return(&common.Customer{}, nil)
			},
			config: &pkg.FlexsaveConfiguration{
				AWS: pkg.FlexsaveSavings{},
			},
		},
		{
			name: "failed to get customer",
			on: func(f *fields) {
				f.customersDAL.On("GetCustomer", ctx, customerID).Return(&common.Customer{}, defaultErr)
			},
			wantErr: defaultErr,
			config: &pkg.FlexsaveConfiguration{
				AWS: pkg.FlexsaveSavings{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			logging, err := logger.NewLogging(ctx)
			if err != nil {
				assert.NoError(t, err)
			}

			conn, err := connection.NewConnection(ctx, logging)
			if err != nil {
				assert.NoError(t, err)
			}

			s := &flexsaveManageNotify{
				loggerProvider: logger.FromContext,
				customersDAL:   &fields.customersDAL,
				Connection:     conn,
			}
			err = s.SendSageMakerActivatedNotification(ctx, customerID, []string{"111"})

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_publishFlexsaveTypeActivatedSlackNotification(t *testing.T) {
	type args struct {
		ctx          context.Context
		customerID   string
		customerName string
		accounts     []string
		flexsaveType utils.FlexsaveType
	}

	tests := []struct {
		name    string
		args    args
		mockErr error
		wantErr error
	}{
		{
			name: "happy path",
			args: args{
				ctx:          context.Background(),
				customerID:   "0123abcdef",
				customerName: "Test Customer",
				accounts:     []string{"111"},
				flexsaveType: utils.SageMakerFlexsaveType,
			},
			mockErr: nil,
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := publishFlexsaveTypeActivatedSlackNotification(tt.args.ctx, tt.args.customerID, tt.args.customerName, tt.args.accounts, tt.args.flexsaveType)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_createSageMakerActivatedNotification(t *testing.T) {
	tests := []struct {
		name         string
		customerID   string
		customerName string
		accounts     []string
		flexsaveType utils.FlexsaveType
	}{
		{
			name:         "create SageMaker notification",
			customerID:   "testID",
			customerName: "testName",
			accounts:     []string{"123", "456"},
			flexsaveType: utils.SageMakerFlexsaveType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createFlexsaveTypeActivatedNotification(tt.customerID, tt.customerName, tt.accounts, tt.flexsaveType)
			assertFlexsaveTypeActivatedNotification(t, tt.customerID, tt.customerName, tt.accounts, tt.flexsaveType, result)
		})
	}
}

func assertFlexsaveTypeActivatedNotification(t *testing.T, customerID, customerName string, accounts []string, flexsaveType utils.FlexsaveType, result map[string]interface{}) {
	attachment := result["attachments"].([]map[string]interface{})[0]

	assert.Equal(t, greenColour, attachment["color"])

	ts := attachment["ts"].(int64)
	currentTimestamp := time.Now().Unix()
	assert.InDelta(t, currentTimestamp, ts, 2, "Timestamp difference is greater than 2 seconds")

	fields := attachment["fields"].([]map[string]interface{})

	expectedValueMessage := fmt.Sprintf(":tada: *Flexsave %s* has just been enabled for <https://console.doit.com/customers/%s|%s> \n", flexsaveType.ToTitle(), customerID, customerName)
	assert.Equal(t, expectedValueMessage, fields[0]["value"])

	expectedAccounts := "Payer Accounts: " + fmt.Sprint(accounts)
	assert.Equal(t, expectedAccounts, fields[1]["value"])
}

func Test_flexsaveManageNotify_SendRDSActivatedNotification(t *testing.T) {
	type fields struct {
		customersDAL customerMocks.Customers
		Connection   *connection.Connection
	}

	defaultErr := errors.New("some error")
	ctx := context.Background()
	customerID := "0123abcdef"

	tests := []struct {
		name    string
		on      func(*fields)
		config  *pkg.FlexsaveConfiguration
		wantErr error
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.customersDAL.On("GetCustomer", ctx, customerID).Return(&common.Customer{}, nil)
			},
		},
		{
			name: "failed to get customer",
			on: func(f *fields) {
				f.customersDAL.On("GetCustomer", ctx, customerID).Return(nil, defaultErr)
			},
			wantErr: defaultErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			logging, err := logger.NewLogging(ctx)
			if err != nil {
				assert.NoError(t, err)
			}

			conn, err := connection.NewConnection(ctx, logging)
			if err != nil {
				assert.NoError(t, err)
			}

			s := &flexsaveManageNotify{
				loggerProvider: logger.FromContext,
				customersDAL:   &fields.customersDAL,
				Connection:     conn,
			}
			err = s.SendRDSActivatedNotification(ctx, customerID, []string{"111"})

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_flexsaveManageNotify_NotifySharedPayerSavingsDiscrepancies(t *testing.T) {
	type args struct {
		ctx           context.Context
		discrepancies monitoringDomain.SharedPayerSavingsDiscrepancies
	}

	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "successful notification",
			args: args{
				ctx: context.TODO(),
				discrepancies: monitoringDomain.SharedPayerSavingsDiscrepancies{
					{CustomerID: "customer123", LastMonthSavings: 500.00},
					{CustomerID: "customer456", LastMonthSavings: 300.00},
				},
			},
			wantErr: assert.NoError,
		},
		{
			name: "no error with empty discrepancies",
			args: args{
				ctx:           context.TODO(),
				discrepancies: monitoringDomain.SharedPayerSavingsDiscrepancies{},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &flexsaveManageNotify{}

			tt.wantErr(t, s.NotifySharedPayerSavingsDiscrepancies(tt.args.ctx, tt.args.discrepancies), fmt.Sprintf("NotifySharedPayerSavingsDiscrepancies(%v, %v)", tt.args.ctx, tt.args.discrepancies))
		})
	}
}
