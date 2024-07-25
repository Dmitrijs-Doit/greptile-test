package aws

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/firestore/mocks"
	"github.com/doitintl/firestore/pkg"
)

func TestAwsStandaloneService_createFlexsaveConfiguration(t *testing.T) {
	contextMock := mock.MatchedBy(func(_ context.Context) bool { return true })
	errDefault := errors.New("let's work the problem, people")

	nowFunc := func() time.Time {
		return time.Date(2022, 5, 5, 0, 0, 0, 0, time.UTC)
	}

	timeParams := timeParams{Now: nowFunc(), CurrentMonth: "5_2022", ApplicableMonths: []string{"5_2022", "4_2022"}, PreviousMonth: "6_2022"}

	spendSummary := &FlexsaveStandaloneData{
		Spend: map[string]*pkg.FlexsaveMonthSummary{
			"5_2022": {
				OnDemandSpend: 8,
				Savings:       4,
				SavingsRate:   5,
			},
			"4_2022": {
				OnDemandSpend: 8,
				Savings:       4,
				SavingsRate:   5,
			},
		},
		SavingsSummary: &pkg.FlexsaveSavingsSummary{
			CurrentMonth: &pkg.FlexsaveCurrentMonthSummary{
				Month:            "5_2022",
				ProjectedSavings: 9876.12,
			},
			NextMonth: &pkg.FlexsaveMonthSummary{
				Savings:       7638,
				OnDemandSpend: 18100,
				SavingsRate:   32,
			},
		},
	}

	configAttributes := configCreationAttributes{
		customerID:   "customerID",
		spendSummary: spendSummary,
	}

	flexsaveSavings := &pkg.FlexsaveSavings{
		Enabled:        true,
		SavingsHistory: spendSummary.Spend,
		SavingsSummary: spendSummary.SavingsSummary,
		TimeEnabled:    configAttributes.timeEnabled,
	}

	type fields struct {
		integrationsDAL mocks.Integrations
	}

	tests := []struct {
		name             string
		on               func(*fields)
		configAttributes configCreationAttributes
		wantErr          error
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.integrationsDAL.On("UpdateFlexsaveConfigurationCustomer",
					contextMock,
					configAttributes.customerID,
					mock.MatchedBy(func(argument map[string]*pkg.FlexsaveSavings) bool {
						a := assert.Equal(t, argument["AWS"].Enabled, flexsaveSavings.Enabled)
						b := assert.Equal(t, argument["AWS"].SavingsHistory, flexsaveSavings.SavingsHistory)
						c := assert.Equal(t, argument["AWS"].SavingsSummary, flexsaveSavings.SavingsSummary)
						d := assert.Equal(t, argument["AWS"].TimeEnabled, flexsaveSavings.TimeEnabled)
						if a && b && c && d {
							return true
						} else {
							return false
						}
					})).
					Return(nil)
			},
			configAttributes: configAttributes,
		},
		{
			name: "failed to update config",
			on: func(f *fields) {
				f.integrationsDAL.On("UpdateFlexsaveConfigurationCustomer",
					contextMock,
					configAttributes.customerID,
					mock.MatchedBy(func(argument map[string]*pkg.FlexsaveSavings) bool {
						a := assert.Equal(t, argument["AWS"].Enabled, flexsaveSavings.Enabled)
						b := assert.Equal(t, argument["AWS"].SavingsHistory, flexsaveSavings.SavingsHistory)
						c := assert.Equal(t, argument["AWS"].SavingsSummary, flexsaveSavings.SavingsSummary)
						d := assert.Equal(t, argument["AWS"].TimeEnabled, flexsaveSavings.TimeEnabled)
						if a && b && c && d {
							return true
						} else {
							return false
						}
					})).
					Return(errDefault)
			},
			configAttributes: configAttributes,
			wantErr:          errDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &AwsStandaloneService{
				integrationsDAL: &fields.integrationsDAL,
			}

			err := s.createFlexsaveConfiguration(context.Background(), tt.configAttributes, timeParams)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
