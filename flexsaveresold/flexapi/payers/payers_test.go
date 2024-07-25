package payers

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
	"github.com/doitintl/http"
	mockClient "github.com/doitintl/http/mocks"
	testUtils "github.com/doitintl/tests"
)

func TestFlexAPIService_CreatePayerConfigForCustomer(t *testing.T) {
	var (
		contextMock   = mock.MatchedBy(func(_ context.Context) bool { return true })
		customerID    = "customerID"
		primaryDomain = "primaryDomain"
		payerAccount1 = "payerAccount1"
		payerAccount2 = "payerAccount2"
		name1         = "name1 "
		name2         = "name2"
		friendlyName1 = "friendlyName1"
		friendlyName2 = "friendlyName2"

		active  = "active"
		pending = "pending"

		resoldConfigType = "aws-flexsave-resold"

		errDefault = errors.New("let's work the problem, people")
	)

	flexapiPayerConfig := []types.PayerConfig{
		{
			AccountID:       payerAccount2,
			CustomerID:      customerID,
			Status:          active,
			Type:            resoldConfigType,
			PrimaryDomain:   primaryDomain,
			FriendlyName:    friendlyName2,
			Name:            name2,
			SageMakerStatus: pending,
			RDSStatus:       pending,
		},
		{
			AccountID:       payerAccount1,
			CustomerID:      customerID,
			Status:          active,
			Type:            resoldConfigType,
			PrimaryDomain:   primaryDomain,
			FriendlyName:    friendlyName1,
			Name:            name1,
			SageMakerStatus: pending,
			RDSStatus:       pending,
		},
	}

	payload := types.PayerConfigCreatePayload{PayerConfigs: flexapiPayerConfig}

	type fields struct {
		client mockClient.IClient
	}

	tests := []struct {
		name    string
		on      func(*fields)
		payload types.PayerConfigCreatePayload
		wantErr error
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.client.On("Post", contextMock, &http.Request{
					URL:     "/payers",
					Payload: &payload,
				}).Return(nil, nil)
			},
			payload: payload,
		},
		{
			name: "failed to create config in flexapi",
			on: func(f *fields) {
				f.client.On("Post", contextMock, &http.Request{
					URL:     "/payers",
					Payload: &payload,
				}).Return(nil, errDefault)
			},
			payload: payload,
			wantErr: errDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{}
			if tt.on != nil {
				tt.on(&fields)
			}

			s := &service{
				flexAPIClient: &fields.client,
			}

			err := s.CreatePayerConfigForCustomer(context.Background(), tt.payload)
			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_GetFlexsaveConfigurationCustomer(t *testing.T) {
	type fields struct {
		client *mockClient.IClient
	}

	tests := []struct {
		on      func(f *fields)
		name    string
		fields  fields
		want    []*types.PayerConfig
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "gets data without an error",
			on: func(f *fields) {
				f.client.On("Get", mock.Anything, mock.Anything).Return(nil, nil).Run(func(args mock.Arguments) {
					request := args.Get(1).(*http.Request)
					*request.ResponseType.(*[]*types.PayerConfig) = []*types.PayerConfig{
						{
							CustomerID: "czekolada",
							AccountID:  "batonik",
						},
					}
				}).Once()
			},
			fields: fields{
				client: &mockClient.IClient{},
			},
			want: []*types.PayerConfig{
				{
					CustomerID: "czekolada",
					AccountID:  "batonik",
				},
			},
			wantErr: assert.ErrorAssertionFunc(assert.NoError),
		},
		{
			name: "returns error from client",
			on: func(f *fields) {
				f.client.
					On("Get", mock.Anything, mock.Anything).
					Return(nil, errors.New("failure")).
					Once()
			},
			fields: fields{
				client: &mockClient.IClient{},
			},
			want:    nil,
			wantErr: testUtils.AssertError("failure"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.on(&tt.fields)

			s := &service{
				flexAPIClient: tt.fields.client,
			}

			got, err := s.GetPayerConfigsForCustomer(context.Background(), "czekolada")
			if !tt.wantErr(t, err) {
				return
			}

			assert.Equalf(t, tt.want, got, "GetPayerConfigsForCustomer")
		})
	}
}

func TestService_GetAWSStandaloneCustomerIDs(t *testing.T) {
	type fields struct {
		client *mockClient.IClient
	}

	tests := []struct {
		on      func(f *fields)
		name    string
		fields  fields
		want    []string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "gets data without an error",
			on: func(f *fields) {
				f.client.On("Get", mock.Anything, mock.Anything).Return(nil, nil).Run(func(args mock.Arguments) {
					request := args.Get(1).(*http.Request)
					ids := []string{"accountID"}
					*request.ResponseType.(*[]string) = ids
				}).Once()
			},
			fields: fields{
				client: &mockClient.IClient{},
			},
			want:    []string{"accountID"},
			wantErr: assert.ErrorAssertionFunc(assert.NoError),
		},
		{
			name: "returns error from client",
			on: func(f *fields) {
				f.client.
					On("Get", mock.Anything, mock.Anything).
					Return(nil, errors.New("failure")).
					Once()
			},
			fields: fields{
				client: &mockClient.IClient{},
			},
			want:    nil,
			wantErr: testUtils.AssertError("failure"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.on(&tt.fields)

			s := &service{
				flexAPIClient: tt.fields.client,
			}

			got, err := s.GetAWSStandaloneCustomerIDs(context.Background())
			if !tt.wantErr(t, err) {
				return
			}

			assert.Equalf(t, tt.want, got, "GetAWSStandaloneCustomerIDs")
		})
	}
}

func Test_service_UpdatePayerConfigsForCustomer(t *testing.T) {
	type fields struct {
		flexAPIClient *mockClient.IClient
	}

	type args struct {
		ctx     context.Context
		configs []types.PayerConfig
	}

	var (
		email  = "doit_employee@doit.com"
		reason = "payer has own savings plan"
	)

	ctxEmpty := context.Background()

	ctxWithData := context.Background()
	ctxWithData = context.WithValue(ctxWithData, common.CtxKeys.Email, email)                // nolint:staticcheck
	ctxWithData = context.WithValue(ctxWithData, utils.StatusChangeReasonContextKey, reason) // nolint:staticcheck

	payerConfigs := []types.PayerConfig{
		{
			AccountID:       "accountID",
			CustomerID:      "customerID",
			Status:          "active",
			Type:            "aws-flexsave-resold",
			PrimaryDomain:   "primaryDomain",
			FriendlyName:    "friendlyName",
			Name:            "name",
			SageMakerStatus: "pending",
			RDSStatus:       "pending",
		},
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		on      func(*fields)
		assert  func(*testing.T, *fields)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "email and reason set",
			fields: fields{
				flexAPIClient: &mockClient.IClient{},
			},
			args: args{
				ctx:     ctxWithData,
				configs: payerConfigs,
			},
			on: func(f *fields) {
				f.flexAPIClient.On("Put", mock.Anything, mock.Anything).Return(nil, nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.flexAPIClient.AssertCalled(t, "Put", mock.Anything, mock.MatchedBy(func(r *http.Request) bool {
					payload := r.Payload.(*types.PayerConfigUpdatePayload)
					return payload.ChangedBy == email && payload.Reason == reason
				}))
			},
			wantErr: assert.ErrorAssertionFunc(assert.NoError),
		},
		{
			name: "email and reason not set",
			fields: fields{
				flexAPIClient: &mockClient.IClient{},
			},
			args: args{
				ctx:     ctxEmpty,
				configs: payerConfigs,
			},
			on: func(f *fields) {
				f.flexAPIClient.On("Put", mock.Anything, mock.Anything).Return(nil, nil)
			},
			assert: func(t *testing.T, f *fields) {
				f.flexAPIClient.AssertCalled(t, "Put", mock.Anything, mock.MatchedBy(func(r *http.Request) bool {
					payload := r.Payload.(*types.PayerConfigUpdatePayload)
					return payload.ChangedBy == "" && payload.Reason == ""
				}))
			},
			wantErr: assert.ErrorAssertionFunc(assert.NoError),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &service{
				flexAPIClient: tt.fields.flexAPIClient,
			}

			tt.on(&tt.fields)

			_, err := s.UpdatePayerConfigsForCustomer(tt.args.ctx, tt.args.configs)

			if !tt.wantErr(t, err, fmt.Sprintf("UpdatePayerConfigsForCustomer(%v, %v)", tt.args.ctx, tt.args.configs)) {
				return
			}

			tt.assert(t, &tt.fields)
		})
	}
}

func Test_service_UpdateStatusWithRequired(t *testing.T) {
	type fields struct {
		flexAPIClient *mockClient.IClient
	}

	type args struct {
		serviceType   utils.FlexsaveType
		serviceStatus string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		on      func(*fields)
	}{
		{
			name:    "updates correct fields for RDS",
			wantErr: false,
			fields:  fields{flexAPIClient: &mockClient.IClient{}},
			args: args{
				serviceType:   utils.RDSFlexsaveType,
				serviceStatus: utils.Active,
			},
			on: func(f *fields) {
				f.flexAPIClient.On("Get", mock.Anything, mock.Anything).Return(nil, nil).Run(func(args mock.Arguments) {
					request := args.Get(1).(*http.Request)
					*request.ResponseType.(*types.PayerConfig) = types.PayerConfig{
						CustomerID:    "customer-id",
						AccountID:     "account-id",
						PrimaryDomain: "primary-domain",
						Name:          "shaggy",
						Status:        utils.Pending,
						Type:          utils.Resold,
					}
				}).Once()

				f.flexAPIClient.On("Put", mock.Anything, mock.Anything).Return(nil, nil).Run(func(args mock.Arguments) {
					request := args.Get(1).(*http.Request)

					payload := request.Payload.(*types.PayerConfigUpdatePayload).PayerConfigs[0]

					assert.Equal(t, "customer-id", payload.CustomerID)
					assert.Equal(t, "account-id", payload.AccountID)
					assert.Equal(t, "primary-domain", payload.PrimaryDomain)
					assert.Equal(t, "shaggy", payload.Name)
					assert.Equal(t, utils.Pending, payload.Status)
					assert.Equal(t, utils.Resold, payload.Type)
					assert.Equal(t, utils.Active, payload.RDSStatus)
				}).Once()
			},
		},
		{
			name:    "updates correct fields for SageMaker",
			wantErr: false,
			fields:  fields{flexAPIClient: &mockClient.IClient{}},
			args: args{
				serviceType:   utils.SageMakerFlexsaveType,
				serviceStatus: utils.Active,
			},
			on: func(f *fields) {
				f.flexAPIClient.On("Get", mock.Anything, mock.Anything).Return(nil, nil).Run(func(args mock.Arguments) {
					request := args.Get(1).(*http.Request)
					*request.ResponseType.(*types.PayerConfig) = types.PayerConfig{
						CustomerID:    "customer-id",
						AccountID:     "account-id",
						PrimaryDomain: "primary-domain",
						Name:          "shaggy",
						Status:        utils.Pending,
						Type:          utils.Resold,
					}
				}).Once()

				f.flexAPIClient.On("Put", mock.Anything, mock.Anything).Return(nil, nil).Run(func(args mock.Arguments) {
					request := args.Get(1).(*http.Request)

					payload := request.Payload.(*types.PayerConfigUpdatePayload).PayerConfigs[0]

					assert.Equal(t, "customer-id", payload.CustomerID)
					assert.Equal(t, "account-id", payload.AccountID)
					assert.Equal(t, "primary-domain", payload.PrimaryDomain)
					assert.Equal(t, "shaggy", payload.Name)
					assert.Equal(t, utils.Pending, payload.Status)
					assert.Equal(t, utils.Resold, payload.Type)
					assert.Equal(t, utils.Active, payload.SageMakerStatus)
				}).Once()
			},
		},
		{
			name:    "updates correct fields for Compute",
			wantErr: false,
			fields:  fields{flexAPIClient: &mockClient.IClient{}},
			args: args{
				serviceType:   utils.ComputeFlexsaveType,
				serviceStatus: utils.Active,
			},
			on: func(f *fields) {
				f.flexAPIClient.On("Get", mock.Anything, mock.Anything).Return(nil, nil).Run(func(args mock.Arguments) {
					request := args.Get(1).(*http.Request)
					*request.ResponseType.(*types.PayerConfig) = types.PayerConfig{
						CustomerID:    "customer-id",
						AccountID:     "account-id",
						PrimaryDomain: "primary-domain",
						Name:          "shaggy",
						Status:        utils.Pending,
						Type:          utils.Resold,
					}
				}).Once()

				f.flexAPIClient.On("Put", mock.Anything, mock.Anything).Return(nil, nil).Run(func(args mock.Arguments) {
					request := args.Get(1).(*http.Request)

					payload := request.Payload.(*types.PayerConfigUpdatePayload).PayerConfigs[0]

					assert.Equal(t, "customer-id", payload.CustomerID)
					assert.Equal(t, "account-id", payload.AccountID)
					assert.Equal(t, "primary-domain", payload.PrimaryDomain)
					assert.Equal(t, "shaggy", payload.Name)
					assert.Equal(t, utils.Active, payload.Status)
					assert.Equal(t, utils.Resold, payload.Type)
				}).Once()
			},
		},
		{
			name:    "getting payer returns error",
			wantErr: true,
			fields:  fields{flexAPIClient: &mockClient.IClient{}},
			on: func(f *fields) {
				f.flexAPIClient.On("Get", mock.Anything, mock.Anything).Return(nil, errors.New("err")).Once()
			},
		},

		{
			name:    "updating payer returns error",
			wantErr: true,
			fields:  fields{flexAPIClient: &mockClient.IClient{}},
			on: func(f *fields) {
				f.flexAPIClient.On("Get", mock.Anything, mock.Anything).Return(nil, nil).Run(func(args mock.Arguments) {
					request := args.Get(1).(*http.Request)
					*request.ResponseType.(*types.PayerConfig) = types.PayerConfig{
						CustomerID:    "customer-id",
						AccountID:     "account-id",
						PrimaryDomain: "primary-domain",
						Name:          "shaggy",
						Status:        "status",
						Type:          "type o negative",
					}
				}).Once()

				f.flexAPIClient.On("Put", mock.Anything, mock.Anything).Return(nil, errors.New("err")).Once()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.on(&tt.fields)
			s := &service{
				flexAPIClient: tt.fields.flexAPIClient,
			}

			err := s.UpdateStatusWithRequired(context.Background(), "account1", tt.args.serviceType, tt.args.serviceStatus)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
