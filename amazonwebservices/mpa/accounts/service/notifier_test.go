package service

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	awsmocks "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal/mocks"
	domain2 "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/service/iface"
	mpamocks "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/service/iface/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func TestNotifierService_NotifyIfNecessary(t *testing.T) {
	var (
		ctx           = context.Background()
		fromPayerID   = "000746511415"
		toPayerID     = "017920819041"
		fromPayer     = iface.Payer{ID: fromPayerID}
		toPayer       = iface.Payer{ID: toPayerID}
		accountName   = "accountName"
		fsAccountName = "fsAccountName"
		someErr       = errors.New("some error")
		move          = iface.AccountMove{
			AccountName: accountName,
			FromPayer:   fromPayer,
			ToPayer:     toPayer}
		dedicatedPayerAcc = domain2.MasterPayerAccount{
			TenancyType: dal.DedicatedTenancy,
		}
		sharedPayerAcc = domain2.MasterPayerAccount{
			TenancyType: dal.SharedTenancy,
		}
	)

	type fields struct {
		masterPayerAccounts awsmocks.MasterPayerAccounts
		billing             mpamocks.Billing
		publisher           mpamocks.NotificationPublisher
	}

	type args struct {
		ctx  context.Context
		move iface.AccountMove
	}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr error
		assert  func(*testing.T, *fields)
	}{
		{
			name: "happy path dedicated payer",
			args: args{
				ctx:  ctx,
				move: move,
			},
			on: func(f *fields) {
				f.masterPayerAccounts.On("GetMasterPayerAccount", ctx, fromPayerID).Return(&dedicatedPayerAcc, nil).Once()
				f.billing.On("GetCoveredUsage", ctx, move.AccountID, move.FromPayer).Return(iface.CoveredUsage{SPCost: dedicatedPayerThreshold + 1}, nil).Once()
				f.publisher.On("PublishSlackNotification", ctx, mock.Anything).Return(nil).Once()
			},
			assert: func(t *testing.T, fields *fields) {
				fields.publisher.AssertNumberOfCalls(t, "PublishSlackNotification", 1)
			},
		},
		{
			name: "happy path shared payer",
			args: args{ctx: ctx, move: move},
			on: func(f *fields) {
				f.masterPayerAccounts.On("GetMasterPayerAccount", ctx, fromPayerID).Return(&sharedPayerAcc, nil).Once()
				f.billing.On("GetCoveredUsage", ctx, move.AccountID, move.FromPayer).Return(iface.CoveredUsage{SPCost: sharedPayerThreshold + 1}, nil).Once()
				f.publisher.On("PublishSlackNotification", ctx, mock.Anything).Return(nil).Once()
			},
			assert: func(t *testing.T, fields *fields) {
				fields.publisher.AssertNumberOfCalls(t, "PublishSlackNotification", 1)
			},
		},
		{
			name: "lower than dedicated payer threshold",
			args: args{ctx: ctx, move: move},
			on: func(f *fields) {
				f.masterPayerAccounts.On("GetMasterPayerAccount", ctx, fromPayerID).Return(&dedicatedPayerAcc, nil).Once()
				f.billing.On("GetCoveredUsage", ctx, move.AccountID, move.FromPayer).Return(iface.CoveredUsage{SPCost: dedicatedPayerThreshold - 1}, nil).Once()
			},
			assert: func(t *testing.T, fields *fields) {
				fields.publisher.AssertNumberOfCalls(t, "PublishSlackNotification", 0)
			},
		},
		{
			name: "lower than shared payer threshold",
			args: args{ctx: ctx, move: move},
			on: func(f *fields) {
				f.masterPayerAccounts.On("GetMasterPayerAccount", ctx, fromPayerID).Return(&sharedPayerAcc, nil).Once()
				f.billing.On("GetCoveredUsage", ctx, move.AccountID, move.FromPayer).Return(iface.CoveredUsage{SPCost: sharedPayerThreshold - 1}, nil).Once()
			},
			assert: func(t *testing.T, fields *fields) {
				fields.publisher.AssertNumberOfCalls(t, "PublishSlackNotification", 0)
			},
		},
		{
			name: "same account",
			args: args{
				ctx: ctx,
				move: iface.AccountMove{
					AccountName: accountName,
					FromPayer:   iface.Payer{ID: fromPayerID},
					ToPayer:     iface.Payer{ID: fromPayerID}}},
			assert: func(t *testing.T, fields *fields) {
				fields.publisher.AssertNumberOfCalls(t, "PublishSlackNotification", 0)
			},
		},
		{
			name: "doit owned account",
			args: args{
				ctx: ctx,
				move: iface.AccountMove{
					AccountName: fsAccountName,
					FromPayer:   iface.Payer{ID: fromPayerID},
					ToPayer:     iface.Payer{ID: toPayerID}}},
			assert: func(t *testing.T, fields *fields) {
				fields.publisher.AssertNumberOfCalls(t, "PublishSlackNotification", 0)
			},
		},
		{
			name: "not payer account",
			args: args{
				ctx: ctx,
				move: iface.AccountMove{
					FromPayer: iface.Payer{ID: "notPayerID"},
					ToPayer:   iface.Payer{ID: toPayerID}}},
			on: func(f *fields) {
				f.masterPayerAccounts.On("GetMasterPayerAccount", ctx, "notPayerID").Return(nil, dal.ErrorNotFound).Once()
			},
			assert: func(t *testing.T, fields *fields) {
				fields.publisher.AssertNumberOfCalls(t, "PublishSlackNotification", 0)
			},
		},
		{
			name: "failed check account existence",
			args: args{ctx: ctx, move: move},
			on: func(f *fields) {
				f.masterPayerAccounts.On("GetMasterPayerAccount", ctx, fromPayerID).Return(nil, someErr).Once()
			},
			assert: func(t *testing.T, fields *fields) {
				fields.publisher.AssertNumberOfCalls(t, "PublishSlackNotification", 0)
			},
			wantErr: someErr,
		},
		{
			name: "failed get billing data",
			args: args{ctx: ctx, move: move},
			on: func(f *fields) {
				f.masterPayerAccounts.On("GetMasterPayerAccount", ctx, fromPayerID).Return(&dedicatedPayerAcc, nil).Once()
				f.billing.On("GetCoveredUsage", ctx, move.AccountID, move.FromPayer).Return(iface.CoveredUsage{}, someErr).Once()
			},
			assert: func(t *testing.T, fields *fields) {
				fields.publisher.AssertNumberOfCalls(t, "PublishSlackNotification", 0)
			},
			wantErr: someErr,
		},
		{
			name: "failed publish notification",
			on: func(f *fields) {
				f.masterPayerAccounts.On("GetMasterPayerAccount", ctx, fromPayerID).Return(&dedicatedPayerAcc, nil).Once()
				f.billing.On("GetCoveredUsage", ctx, move.AccountID, move.FromPayer).Return(iface.CoveredUsage{SPCost: 50.0}, nil).Once()
				f.publisher.On("PublishSlackNotification", ctx, mock.Anything).Return(someErr).Once()
			},
			args:    args{ctx: ctx, move: move},
			wantErr: someErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			n := &NotifierService{
				masterPayerAccounts: &fields.masterPayerAccounts,
				billing:             &fields.billing,
				publisher:           &fields.publisher,
				loggerProvider:      logger.FromContext,
			}

			err := n.NotifyIfNecessary(ctx, tt.args.move, iface.MovedAccount)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			if tt.assert != nil {
				tt.assert(t, &fields)
			}
		})
	}
}
