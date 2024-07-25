package payermanager

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/doitintl/firestore/mocks"
	payerMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/payers/mocks"
	computestatemocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/compute/mocks"
	rdsstatemocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/rds/mocks"
	sagemakerstatemocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/payermanager/sagemaker/mocks"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	active  = "active"
	pending = "pending"
)

func Test_service_UpdateNonStatusPayerConfigFields(t *testing.T) {
	var (
		ctx        = context.Background()
		accountID  = "1234454"
		customerID = "AHBDHB"

		payer = mockPayerConfig(customerID, accountID, pending, nil, nil)
		spend = 2.0

		someErr = errors.New("something went wrong")
	)

	type fields struct {
		payers       payerMocks.Service
		integrations mocks.Integrations
	}

	tests := []struct {
		name    string
		on      func(*fields)
		entry   FormEntry
		wantErr error
	}{
		{
			name: "happy path, only update non status fields",
			entry: FormEntry{
				Status:   active,
				Managed:  "manual",
				MinSpend: nil,
				MaxSpend: &spend,
				Discount: nil,
			},
			on: func(f *fields) {
				f.payers.On("UpdatePayerConfigsForCustomer", ctx, mock.MatchedBy(func(arg []types.PayerConfig) bool {
					assert.NotNil(t, arg[0].MaxSpend)
					assert.Equal(t, arg[0].Managed, "manual")
					assert.NotEqual(t, arg[0].Status, active)
					return true
				})).Return([]types.PayerConfig{}, nil)
			},
		},
		{
			name: "failed to update payer config",
			entry: FormEntry{
				Status:   active,
				Managed:  "manual",
				MinSpend: nil,
				MaxSpend: &spend,
				Discount: nil,
			},
			on: func(f *fields) {
				f.payers.On("UpdatePayerConfigsForCustomer", ctx, mock.MatchedBy(func(arg []types.PayerConfig) bool {
					assert.NotNil(t, arg[0].MaxSpend)
					assert.Equal(t, arg[0].Managed, "manual")
					assert.NotEqual(t, arg[0].Status, active)
					return true
				})).Return([]types.PayerConfig{}, someErr)
			},
			wantErr: fmt.Errorf("UpdatePayerConfigsForCustomer() failed for payer '%s': %w", accountID, someErr),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &service{
				payers:       &fields.payers,
				integrations: &fields.integrations,
			}

			err := s.UpdateNonStatusPayerConfigFields(ctx, payer, tt.entry)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
			}
		})
	}
}

func mockPayerConfig(customerID, accountID, status string, timeEnabled, timeDisabled *time.Time) types.PayerConfig {
	return types.PayerConfig{
		CustomerID:       customerID,
		AccountID:        accountID,
		PrimaryDomain:    "primary-domain",
		FriendlyName:     "friendly-name",
		Name:             "name",
		Status:           status,
		Type:             "",
		Managed:          "",
		TimeEnabled:      timeEnabled,
		TimeDisabled:     timeDisabled,
		LastUpdated:      nil,
		TargetPercentage: nil,
		MinSpend:         nil,
		MaxSpend:         nil,
		DiscountDetails:  nil,
	}
}

func Test_service_ProcessPayerStateTransitionForFlexsaveType(t *testing.T) {
	var (
		ctx        = context.Background()
		accountID  = "1234454"
		customerID = "AHBDHB"
	)

	type fields struct {
		rds       rdsstatemocks.Service
		sagemaker sagemakerstatemocks.Service
		compute   computestatemocks.Service
	}

	tests := []struct {
		name          string
		on            func(*fields)
		wantErr       error
		flexsaveType  utils.FlexsaveType
		initialStatus string
		targetStatus  string
	}{
		{
			name:          "compute - active to pending ",
			flexsaveType:  utils.ComputeFlexsaveType,
			initialStatus: utils.Active,
			targetStatus:  utils.Pending,
			on: func(f *fields) {
				f.compute.On("ProcessPayerStatusTransition", ctx, customerID, accountID, utils.Active, utils.Pending).Return(nil)
			},
		},
		{
			name:          "sagemaker - active to disabled",
			flexsaveType:  utils.SageMakerFlexsaveType,
			initialStatus: utils.Active,
			targetStatus:  utils.Disabled,
			on: func(f *fields) {
				f.sagemaker.On("ProcessPayerStatusTransition", ctx, customerID, accountID, utils.Active, utils.Disabled).Return(nil)
			},
		},
		{
			name:          "rds - disabled to pending",
			flexsaveType:  utils.RDSFlexsaveType,
			initialStatus: utils.Disabled,
			targetStatus:  utils.Pending,
			on: func(f *fields) {
				f.rds.On("ProcessPayerStatusTransition", ctx, customerID, accountID, utils.Disabled, utils.Pending).Return(nil)
			},
		},
		{
			name:          "unrecognised flexsave type",
			flexsaveType:  "FlexRI",
			initialStatus: utils.Disabled,
			targetStatus:  utils.Pending,
			wantErr:       errors.New("unsupported FlexsaveType"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &service{
				rds:       &fields.rds,
				sagemaker: &fields.sagemaker,
				compute:   &fields.compute,
			}

			err := s.ProcessPayerStatusTransition(ctx, customerID, accountID, tt.initialStatus, tt.targetStatus, tt.flexsaveType)
			if err != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, tt.wantErr)
			}
		})
	}
}
