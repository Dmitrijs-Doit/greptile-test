package flexapi

import (
	"context"
	"errors"
	stdhttp "net/http"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/testutils"
	"github.com/doitintl/http"
	mockClient "github.com/doitintl/http/mocks"
)

func TestFlexAPIService_ListFlexSaveAccounts(t *testing.T) {
	type args struct {
		ctx context.Context
	}

	type fields struct {
		client *mockClient.IClient
	}

	var accounts = []*Account{{AccountID: "accountID1"}}

	var result = []string{"accountID1"}

	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
		on      func(f *fields)
		want    []string
	}{
		{
			name: "Success",
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
			on: func(f *fields) {
				f.client.On("Get", testutils.ContextBackgroundMock, mock.AnythingOfType("*http.Request")).Return(nil, nil).Run(func(args mock.Arguments) {
					request := args.Get(1).(*http.Request)
					*request.ResponseType.(*[]*Account) = accounts
				}).Once()
			},
			want: result,
		},
		{
			name: "Get error",
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
			on: func(f *fields) {
				f.client.On("Get", testutils.ContextBackgroundMock, mock.AnythingOfType("*http.Request")).Return(nil, errors.New("fail")).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				client: &mockClient.IClient{},
			}
			s := &Service{
				flexAPIClient: tt.fields.client,
				mu:            &sync.Mutex{},
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			got, err := s.ListFlexsaveAccounts(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("service.ListFlexsaveAccounts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("service.ListFlexsaveAccounts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFlexAPIService_ListFlexsaveAccountsWithCache(t *testing.T) {
	var accounts = []*Account{{AccountID: "accountID"}}

	type fields struct {
		timeUpdatedAccounts time.Time
		flexsaveAccounts    []*Account
		mu                  *sync.Mutex
		client              *mockClient.IClient
	}

	ctx := context.Background()

	type args struct {
		ctx         context.Context
		refreshTime time.Duration
	}

	var result = []string{"accountID"}

	var wantAccounts = []*Account{{AccountID: "accountID"}}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		on      func(f *fields)
		want    []string
	}{
		{
			name: "without refresh accounts",
			args: args{
				ctx:         ctx,
				refreshTime: time.Minute,
			},
			wantErr: false,
			fields: fields{
				flexsaveAccounts: []*Account{
					{
						AccountID: "accountID",
					},
				},
				timeUpdatedAccounts: time.Now().Add(time.Hour),
			},
			want: result,
		},
		{
			name: "with refresh accounts",
			args: args{
				ctx:         ctx,
				refreshTime: time.Minute,
			},
			wantErr: false,
			fields: fields{
				flexsaveAccounts: []*Account{
					{
						AccountID: "accountID",
					},
				},
				timeUpdatedAccounts: time.Now().Add(-time.Hour),
				mu:                  &sync.Mutex{},
			},
			on: func(f *fields) {
				f.client.On("Get", testutils.ContextBackgroundMock, mock.AnythingOfType("*http.Request")).Return(nil, nil).Run(func(args mock.Arguments) {
					request := args.Get(1).(*http.Request)
					*request.ResponseType.(*[]*Account) = accounts
				}).Once()
			},
			want: result,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				client:              &mockClient.IClient{},
				timeUpdatedAccounts: tt.fields.timeUpdatedAccounts,
				flexsaveAccounts:    tt.fields.flexsaveAccounts,
				mu:                  tt.fields.mu,
			}

			s := &Service{
				flexAPIClient:       fields.client,
				timeUpdatedAccounts: fields.timeUpdatedAccounts,
				flexsaveAccounts:    fields.flexsaveAccounts,
				mu:                  fields.mu,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			got, err := s.ListFlexsaveAccountsWithCache(tt.args.ctx, tt.args.refreshTime)
			if (err != nil) != tt.wantErr {
				t.Errorf("service.ListFlexsaveAccountsWithCache() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("service.ListFlexsaveAccountsWithCache() = %v, want %v", got, tt.want)
			}

			if !reflect.DeepEqual(wantAccounts, s.flexsaveAccounts) {
				t.Errorf("service.ListFlexsaveAccountsWithCache() = %v, want %v", s.flexsaveAccounts, wantAccounts)
			}
		})
	}
}

func TestService_GetRDSPayerRecommendations(t *testing.T) {
	type fields struct {
		flexAPIClient *mockClient.IClient
	}

	type args struct {
		payerID string
	}

	tests := []struct {
		on      func(f *fields)
		name    string
		fields  fields
		args    args
		want    []RDSBottomUpRecommendation
		wantErr bool
	}{
		{
			name: "Gets data correctly",
			on: func(f *fields) {
				f.flexAPIClient.
					On("Get", testutils.ContextBackgroundMock, mock.AnythingOfType("*http.Request")).
					Return(nil, nil).Run(func(args mock.Arguments) {
					request := args.Get(1).(*http.Request)
					*request.ResponseType.(*[]RDSBottomUpRecommendation) = []RDSBottomUpRecommendation{
						{
							Database: "la databasa",
						},
					}
				}).
					Once()
			},
			fields: fields{
				flexAPIClient: &mockClient.IClient{},
			},
			args: args{
				payerID: "payerID",
			},
			want: []RDSBottomUpRecommendation{
				{
					Database: "la databasa",
				},
			},
			wantErr: false,
		},
		{
			name: "simple error",
			on: func(f *fields) {
				f.flexAPIClient.
					On("Get", testutils.ContextBackgroundMock, mock.AnythingOfType("*http.Request")).
					Return(nil, http.WebError{
						Err:  errors.New("error"),
						Code: stdhttp.StatusInternalServerError,
					}).
					Once()
			},
			fields: fields{
				flexAPIClient: &mockClient.IClient{},
			},
			args: args{
				payerID: "payerID",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "not found error",
			on: func(f *fields) {
				f.flexAPIClient.
					On("Get", testutils.ContextBackgroundMock, mock.AnythingOfType("*http.Request")).
					Return(nil, http.WebError{
						Err:  errors.New("not found"),
						Code: stdhttp.StatusNotFound,
					}).
					Once()
			},
			fields: fields{
				flexAPIClient: &mockClient.IClient{},
			},
			args: args{
				payerID: "payerID",
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.on(&tt.fields)
			s := &Service{
				flexAPIClient: tt.fields.flexAPIClient,
			}

			got, err := s.GetRDSPayerRecommendations(context.Background(), tt.args.payerID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRDSPayerRecommendations() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRDSPayerRecommendations() got = %v, want %v", got, tt.want)
			}
		})
	}
}
