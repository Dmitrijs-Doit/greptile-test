package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	mpaMocks "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal/mocks"
	mpaDomain "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/ples/domain"
	assetsMock "github.com/doitintl/hello/scheduled-tasks/assets/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	flexsaveMocks "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/mocks"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func TestPLESService_validatePLESAccounts(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		assetsDal      assetsMock.AssetSettings
		mpaDal         mpaMocks.MasterPayerAccounts
		flexsaveAPI    flexsaveMocks.FlexAPI
	}

	type args struct {
		ctx      context.Context
		accounts []domain.PLESAccount
	}

	tests := []struct {
		name string
		on   func(f *fields)
		args args
		want []error
	}{
		{
			name: "Validate PLES Accounts - Success",
			args: args{
				ctx: context.TODO(),
				accounts: []domain.PLESAccount{
					{AccountID: "account1", AccountName: "fs-account1", PayerID: "payer1", SupportLevel: "basic"},
					{AccountID: "account2", AccountName: "Account 2", PayerID: "payer2", SupportLevel: "basic"},
				},
			},
			on: func(f *fields) {
				f.flexsaveAPI.On("ListFlexsaveAccounts", context.TODO()).Return([]string{"account1"}, nil)
				f.mpaDal.On("GetActiveAndRetiredPlesMpa", context.TODO()).Return(
					map[string]*mpaDomain.MasterPayerAccount{
						"payer1": {Status: "active"},
						"payer2": {Status: "active"},
					}, nil)
				f.assetsDal.On("GetAllAWSAssetSettings", context.TODO()).Return([]*pkg.AWSAssetSettings{
					{BaseAsset: pkg.BaseAsset{ID: "account1"}},
					{BaseAsset: pkg.BaseAsset{ID: "account2"}},
				}, nil)
			},
			want: []error{},
		},
		{
			name: "Validate PLES Accounts - Failure",
			args: args{
				ctx: context.TODO(),
				accounts: []domain.PLESAccount{
					{AccountID: "account1", AccountName: "Account 1", PayerID: "payer1", SupportLevel: "basic"},
					{AccountID: "account2", AccountName: "Account 2", PayerID: "payer2", SupportLevel: "basic"},
				},
			},
			on: func(f *fields) {
				f.flexsaveAPI.On("ListFlexsaveAccounts", context.TODO()).Return([]string{"account1"}, nil)
				f.mpaDal.On("GetActiveAndRetiredPlesMpa", context.TODO()).Return(
					map[string]*mpaDomain.MasterPayerAccount{
						"payer1": {Status: "active"},
						"payer3": {Status: "active"},
					}, nil)
				f.assetsDal.On("GetAllAWSAssetSettings", context.TODO()).Return([]*pkg.AWSAssetSettings{}, nil)
			},
			want: []error{
				ErrInvalidFlexsaveAccountName(2, "Account 1"),
				ErrAccountIDDoesNotExist(3, "account2"),
				ErrPayerIDDoesNotExist(3, "payer2"),
				ErrPayerIDNotInRequest("payer3"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				logger.FromContext,
				assetsMock.AssetSettings{},
				mpaMocks.MasterPayerAccounts{},
				flexsaveMocks.FlexAPI{},
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &PLESService{
				loggerProvider: fields.loggerProvider,
				assetsDal:      &fields.assetsDal,
				mpaDal:         &fields.mpaDal,
				flexsaveAPI:    &fields.flexsaveAPI,
			}
			if got := s.validatePLESAccounts(tt.args.ctx, tt.args.accounts); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PLESService.validatePLESAccounts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPLESService_getFlexsaveAccountDict(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		assetsDal      assetsMock.AssetSettings
		mpaDal         mpaMocks.MasterPayerAccounts
		flexsaveAPI    flexsaveMocks.FlexAPI
	}

	type args struct {
		ctx context.Context
	}

	tests := []struct {
		name    string
		args    args
		on      func(f *fields)
		want    map[string]bool
		wantErr bool
	}{
		{
			name: "Get Flexsave Account Dict - Success",
			args: args{
				ctx: context.TODO(),
			},
			on: func(f *fields) {
				f.flexsaveAPI.On("ListFlexsaveAccounts", context.TODO()).Return([]string{"account1", "account2"}, nil)
			},
			want: map[string]bool{
				"account1": false,
				"account2": false,
			},
			wantErr: false,
		},
		{
			name: "Get Flexsave Account Dict - Failure",
			args: args{
				ctx: context.TODO(),
			},
			on: func(f *fields) {
				f.flexsaveAPI.On("ListFlexsaveAccounts", context.TODO()).Return(nil, errors.New("error"))
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				logger.FromContext,
				assetsMock.AssetSettings{},
				mpaMocks.MasterPayerAccounts{},
				flexsaveMocks.FlexAPI{},
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &PLESService{
				loggerProvider: fields.loggerProvider,
				assetsDal:      &fields.assetsDal,
				mpaDal:         &fields.mpaDal,
				flexsaveAPI:    &fields.flexsaveAPI,
			}

			got, err := s.getFlexsaveAccountDict(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("PLESService.getFlexsaveAccountDict() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PLESService.getFlexsaveAccountDict() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPLESService_getMasterPayerAccountsWithPLESDict(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		assetsDal      assetsMock.AssetSettings
		mpaDal         mpaMocks.MasterPayerAccounts
		flexsaveAPI    flexsaveMocks.FlexAPI
	}

	type args struct {
		Accounts map[string]*mpaDomain.MasterPayerAccount
	}

	tests := []struct {
		name    string
		args    args
		on      func(f *fields)
		want    map[string]bool
		wantErr bool
	}{
		{
			name: "Get Master Payer Accounts with PLES Dict",
			args: args{
				Accounts: map[string]*mpaDomain.MasterPayerAccount{
					"payer1": {Status: "active"},
					"payer2": {Status: "active"},
					"payer3": {Status: "retired"},
				},
			},
			want: map[string]bool{
				"payer1": false,
				"payer2": false,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				logger.FromContext,
				assetsMock.AssetSettings{},
				mpaMocks.MasterPayerAccounts{},
				flexsaveMocks.FlexAPI{},
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &PLESService{
				loggerProvider: fields.loggerProvider,
				assetsDal:      &fields.assetsDal,
				mpaDal:         &fields.mpaDal,
				flexsaveAPI:    &fields.flexsaveAPI,
			}

			got, err := s.getActivePlesMpaTrackingDict(tt.args.Accounts)
			if (err != nil) != tt.wantErr {
				t.Errorf("PLESService.getMasterPayerAccountsWithPLESDict() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PLESService.getMasterPayerAccountsWithPLESDict() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPLESService_getAWSAssetsDict(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		assetsDal      assetsMock.AssetSettings
		mpaDal         mpaMocks.MasterPayerAccounts
		flexsaveAPI    flexsaveMocks.FlexAPI
	}

	type args struct {
		ctx context.Context
	}

	tests := []struct {
		name    string
		args    args
		on      func(f *fields)
		want    map[string]bool
		wantErr bool
	}{
		{
			name: "Get AWS assets successfully",
			args: args{
				ctx: context.TODO(),
			},
			on: func(f *fields) {
				f.assetsDal.On("GetAllAWSAssetSettings", context.TODO()).Return([]*pkg.AWSAssetSettings{
					{BaseAsset: pkg.BaseAsset{ID: "asset1"}},
					{BaseAsset: pkg.BaseAsset{ID: "asset2"}},
				}, nil)
			},
			want: map[string]bool{
				"asset1": false,
				"asset2": false,
			},
			wantErr: false,
		},
		{
			name: "Get AWS assets failed",
			args: args{
				ctx: context.TODO(),
			},
			on: func(f *fields) {
				f.assetsDal.On("GetAllAWSAssetSettings", context.TODO()).Return(nil, errors.New("error"))
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				logger.FromContext,
				assetsMock.AssetSettings{},
				mpaMocks.MasterPayerAccounts{},
				flexsaveMocks.FlexAPI{},
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			s := &PLESService{
				loggerProvider: fields.loggerProvider,
				assetsDal:      &fields.assetsDal,
				mpaDal:         &fields.mpaDal,
				flexsaveAPI:    &fields.flexsaveAPI,
			}

			got, err := s.getAWSAssetsDict(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("PLESService.getAWSAssetsDict() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PLESService.getAWSAssetsDict() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_validateAccountName(t *testing.T) {
	type args struct {
		accountName       string
		isFlexsaveAccount bool
		row               int
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Invalid account name",
			args: args{
				accountName:       "",
				isFlexsaveAccount: false,
				row:               0,
			},
			wantErr: true,
		},
		{
			name: "Valid account name",
			args: args{
				accountName:       "account-name",
				isFlexsaveAccount: false,
				row:               1,
			},
			wantErr: false,
		},
		{
			name: "Valid flexsave account name",
			args: args{
				accountName:       "fs-account-name",
				isFlexsaveAccount: true,
				row:               2,
			},
			wantErr: false,
		},
		{
			name: "Invalid flexsave account name",
			args: args{
				accountName:       "account-name",
				isFlexsaveAccount: true,
				row:               3,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateAccountName(tt.args.accountName, tt.args.isFlexsaveAccount, tt.args.row); (err != nil) != tt.wantErr {
				t.Errorf("validateAccountName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_validatePayerID(t *testing.T) {
	type args struct {
		payerID         string
		mpaPLESAccounts map[string]bool
		row             int
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Valid payer ID",
			args: args{
				payerID: "payer1",
				mpaPLESAccounts: map[string]bool{
					"payer1": true,
					"payer2": true,
				},
				row: 0,
			},
			wantErr: false,
		},
		{
			name: "Payer ID not in mpaPLESAccounts",
			args: args{
				payerID: "payer3",
				mpaPLESAccounts: map[string]bool{
					"payer1": true,
					"payer2": true,
				},
				row: 1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validatePayerID(tt.args.payerID, tt.args.mpaPLESAccounts, tt.args.row); (err != nil) != tt.wantErr {
				t.Errorf("validatePayerID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_validateAllPayerAccountsExistInRequest(t *testing.T) {
	type args struct {
		mpaPLESAccounts map[string]bool
	}

	tests := []struct {
		name string
		args args
		want []error
	}{
		{
			name: "All payer accounts exist in request",
			args: args{
				mpaPLESAccounts: map[string]bool{
					"payer1": true,
					"payer2": true,
				},
			},
			want: []error{},
		},
		{
			name: "Some payer accounts do not exist in request",
			args: args{
				mpaPLESAccounts: map[string]bool{
					"payer1": false,
					"payer2": true,
				},
			},
			want: []error{
				ErrPayerIDNotInRequest("payer1"),
			},
		},
		{
			name: "No payer accounts exist in request",
			args: args{
				mpaPLESAccounts: map[string]bool{
					"payer1": false,
					"payer2": false,
				},
			},
			want: []error{
				ErrPayerIDNotInRequest("payer1"),
				ErrPayerIDNotInRequest("payer2"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validateAllPayerAccountsExistInRequest(tt.args.mpaPLESAccounts); len(got) != len(tt.want) {
				t.Errorf("validateAllPayerAccountsExistInRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}
