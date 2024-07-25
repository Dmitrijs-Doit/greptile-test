package service

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	dal_mocks "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/service/iface"
	flexapi_mocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/mocks"
)

func TestBillingService_GetCoveredUsage(t *testing.T) {
	var (
		accID           = "accountID"
		payerNumber     = 3
		fromPayer       = iface.Payer{ID: "payerID", DisplayName: fmt.Sprintf("payer #%d", payerNumber)}
		spARNs          = []string{"arn1", "arn2"}
		coveredUsage    = iface.CoveredUsage{SPCost: 30, RICost: 25}
		dalCoveredUsage = dal.CoveredUsage{SPCost: 30, RICost: 25}
		someErr         = errors.New("some err")
		ctx             = context.Background()
	)

	type fields struct {
		billing dal_mocks.Billing
		flexAPI flexapi_mocks.FlexAPI
	}

	type args struct {
		ctx       context.Context
		accountID string
		from      iface.Payer
	}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr error
		want    iface.CoveredUsage
	}{
		{
			name: "happy path",
			on: func(fields *fields) {
				fields.flexAPI.On("ListARNs", ctx).Return(spARNs, nil)
				fields.billing.On("GetCoveredUsage", ctx, accID, fromPayer.ID, 3, spARNs, riAccountIDs).Return(dalCoveredUsage, nil)
			},
			args: args{ctx: ctx, accountID: accID, from: fromPayer},
			want: coveredUsage,
		},
		{
			name: "failed list ARNs",
			on: func(fields *fields) {
				fields.flexAPI.On("ListARNs", ctx).Return(nil, someErr)
			},
			args:    args{ctx: ctx, accountID: accID},
			wantErr: someErr,
		},
		{
			name: "failed get covered usage",
			on: func(fields *fields) {
				fields.flexAPI.On("ListARNs", ctx).Return(spARNs, nil)
				fields.billing.On("GetCoveredUsage", ctx, accID, fromPayer.ID, 3, spARNs, riAccountIDs).Return(dalCoveredUsage, someErr)
			},
			args:    args{ctx: ctx, accountID: accID, from: fromPayer},
			wantErr: someErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &BillingService{
				billing: &fields.billing,
				flexAPI: &fields.flexAPI,
			}

			got, err := s.GetCoveredUsage(tt.args.ctx, tt.args.accountID, tt.args.from)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BillingService.GetCoveredUsage() = %v, want %v", got, tt.want)
			}
		})
	}
}
