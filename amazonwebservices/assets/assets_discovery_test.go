package assets

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/stretchr/testify/mock"

	cloudtasksMock "github.com/doitintl/cloudtasks/mocks"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/firestore/mocks"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	mpaMocks "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal/mocks"
	amazonwebservicesDomain "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	entityMocks "github.com/doitintl/hello/scheduled-tasks/entity/dal/mocks"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/testutils"
)

func TestAWSAssetsService_getEntityRef(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		entitiesDAL    *entityMocks.Entites
	}

	type args struct {
		ctx      context.Context
		customer *common.Customer
	}

	entity := &common.Entity{
		Active:   true,
		Snapshot: &firestore.DocumentSnapshot{},
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *firestore.DocumentRef
		wantErr bool
		on      func(f *fields)
	}{
		{
			name: "happy path",
			args: args{
				ctx: context.Background(),
				customer: &common.Customer{
					Entities: []*firestore.DocumentRef{
						{ID: "1"},
						{ID: "2"},
						{ID: "3"},
					},
				},
			},
			on: func(f *fields) {
				f.entitiesDAL.On("GetEntity", testutils.ContextBackgroundMock, "1").Return(entity, nil).Once()
				f.entitiesDAL.On("GetEntity", testutils.ContextBackgroundMock, "2").Return(entity, errors.New("error")).Once()
				f.entitiesDAL.On("GetEntity", testutils.ContextBackgroundMock, "3").Return(entity, nil).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				loggerProvider: logger.FromContext,
				entitiesDAL:    &entityMocks.Entites{},
			}
			s := &AWSAssetsService{
				loggerProvider: fields.loggerProvider,
				entitiesDAL:    fields.entitiesDAL,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			if got := s.getEntityRef(tt.args.ctx, tt.args.customer); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AWSAssetsService.getEntityRef() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAWSAssetsService_getAccount(t *testing.T) {
	type args struct {
		account            *organizations.Account
		masterPayerAccount *amazonwebservicesDomain.MasterPayerAccount
	}

	stringText := "text"

	timeStamp := &time.Time{}

	account := &organizations.Account{
		Id:              &stringText,
		Name:            &stringText,
		Status:          &stringText,
		Arn:             &stringText,
		Email:           &stringText,
		JoinedMethod:    &stringText,
		JoinedTimestamp: timeStamp,
	}

	masterPayerAccount := &amazonwebservicesDomain.MasterPayerAccount{}

	want := &amazonwebservices.Account{
		ID:              stringText,
		Name:            stringText,
		Status:          stringText,
		Arn:             stringText,
		Email:           stringText,
		JoinedMethod:    stringText,
		JoinedTimestamp: *timeStamp,
		PayerAccount: &amazonwebservicesDomain.PayerAccount{
			AccountID:   "",
			DisplayName: "",
		},
	}
	tests := []struct {
		name string
		args args
		want *amazonwebservices.Account
	}{
		{
			name: "Happy path",
			args: args{
				account:            account,
				masterPayerAccount: masterPayerAccount,
			},
			want: want,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AWSAssetsService{}
			if got := s.getAccount(tt.args.account, tt.args.masterPayerAccount); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AWSAssetsService.getAccount() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAWSAssetsService_getAssetProperties(t *testing.T) {
	type args struct {
		account       *organizations.Account
		accountOrg    *amazonwebservices.Account
		hasSauronRole bool
		Support       pkg.AWSSettingsSupport
	}

	stringText := "text"

	timeStamp := &time.Time{}

	account := &organizations.Account{
		Id:              &stringText,
		Name:            &stringText,
		Status:          &stringText,
		Arn:             &stringText,
		Email:           &stringText,
		JoinedMethod:    &stringText,
		JoinedTimestamp: timeStamp,
	}

	accountOrg := &amazonwebservices.Account{}

	support := pkg.AWSSettingsSupport{}

	want := &pkg.AWSProperties{
		AccountID:    stringText,
		Name:         stringText,
		FriendlyName: stringText,
		SauronRole:   true,
		OrganizationInfo: &pkg.OrganizationInfo{
			PayerAccount: accountOrg.PayerAccount,
			Status:       accountOrg.Status,
			Email:        accountOrg.Email,
		},
		Support: &support,
	}

	tests := []struct {
		name string
		args args
		want *pkg.AWSProperties
	}{
		{
			name: "Happy path",
			args: args{
				account:       account,
				accountOrg:    accountOrg,
				hasSauronRole: true,
				Support:       support,
			},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AWSAssetsService{}
			if got := s.getAssetProperties(tt.args.account, tt.args.accountOrg, tt.args.hasSauronRole, &tt.args.Support); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AWSAssetsService.getAssetProperties() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAWSAssetsService_buildPathsNoCustomer(t *testing.T) {
	type args struct {
		ctx       context.Context
		batch     *mocks.Batch
		paths     *[]firestore.FieldPath
		mpaData   *mpaData
		assetData *assetData
	}

	mpaData := &mpaData{
		customerRef: &firestore.DocumentRef{},
		assetType:   "",
	}

	assetData := &assetData{
		assetSettingsRef: &firestore.DocumentRef{},
		asset:            &amazonwebservices.Asset{},
	}

	paths := []firestore.FieldPath{}

	batch := mocks.Batch{}
	batch.On("Set", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.DocumentRef"), mock.AnythingOfType("map[string]interface {}"), mock.AnythingOfType("firestore.merge")).Return(nil).Once()
	batch.On("Set", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.DocumentRef"), mock.AnythingOfType("map[string]interface {}"), mock.AnythingOfType("firestore.merge")).Return(errors.New("error")).Once()

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Happy path",
			args: args{
				ctx:       context.Background(),
				mpaData:   mpaData,
				assetData: assetData,
				paths:     &paths,
				batch:     &batch,
			},
		},
		{
			name: "Batch set error",
			args: args{
				ctx:       context.Background(),
				paths:     &paths,
				batch:     &batch,
				mpaData:   mpaData,
				assetData: assetData,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AWSAssetsService{}
			if err := s.buildPathsNoCustomer(tt.args.ctx, tt.args.batch, tt.args.mpaData, tt.args.assetData, tt.args.paths); (err != nil) != tt.wantErr {
				t.Errorf("AWSAssetsService.buildPathsNoCustomer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAWSAssetsService_UpdateAssetsAllMPA(t *testing.T) {
	type fields struct {
		loggerProvider  logger.Provider
		Connection      *connection.Connection
		cloudTaskClient *cloudtasksMock.CloudTaskClient
		mpaDAL          *mpaMocks.MasterPayerAccounts
	}

	type args struct {
		ctx context.Context
	}

	accounts := make(map[string]*amazonwebservicesDomain.MasterPayerAccount)
	accounts["123"] = &amazonwebservicesDomain.MasterPayerAccount{AccountNumber: "123"}

	mpa := &amazonwebservicesDomain.MasterPayerAccounts{
		Accounts: accounts,
	}

	ctx := context.Background()
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		on      func(f *fields)
	}{
		{
			name: "Happy path",
			args: args{
				ctx: ctx,
			},
			on: func(f *fields) {
				f.mpaDAL.On("GetMasterPayerAccounts", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.Client")).Return(mpa, nil)
				f.cloudTaskClient.On("CreateTask", testutils.ContextBackgroundMock, mock.AnythingOfType("*iface.Config")).Return(nil, nil)
			},
		},
		{
			name: "error on GetMasterPayerAccounts",
			args: args{
				ctx: ctx,
			},
			on: func(f *fields) {
				f.mpaDAL.On("GetMasterPayerAccounts", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.Client")).Return(mpa, errors.New("error"))
				f.cloudTaskClient.On("CreateTask", testutils.ContextBackgroundMock, mock.AnythingOfType("*iface.Config")).Return(nil, nil)
			},
			wantErr: true,
		},
		{
			name: "error on CreateTask",
			args: args{
				ctx: ctx,
			},
			on: func(f *fields) {
				f.mpaDAL.On("GetMasterPayerAccounts", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.Client")).Return(mpa, nil)
				f.cloudTaskClient.On("CreateTask", testutils.ContextBackgroundMock, mock.AnythingOfType("*iface.Config")).Return(nil, errors.New("error"))
			},
			wantErr: false,
		},
	}

	logging, err := logger.NewLogging(ctx)
	if err != nil {
		t.Errorf("main: could not initialize logging. error %s", err)
	}

	conn, err := connection.NewConnection(ctx, logging)
	if err != nil {
		t.Errorf("main: could not initialize db connections. error %s", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				loggerProvider:  logger.FromContext,
				Connection:      conn,
				cloudTaskClient: &cloudtasksMock.CloudTaskClient{},
				mpaDAL:          &mpaMocks.MasterPayerAccounts{},
			}
			s := &AWSAssetsService{
				loggerProvider:  fields.loggerProvider,
				conn:            fields.Connection,
				cloudTaskClient: fields.cloudTaskClient,
				mpaDAL:          fields.mpaDAL,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			if err := s.UpdateAssetsAllMPA(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("AWSAssetsService.UpdateAssetsAllMPA() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAWSAssetsService_buildPathsCustomer(t *testing.T) {
	type args struct {
		ctx       context.Context
		batch     iface.Batch
		mpaData   *mpaData
		assetData *assetData
		paths     *[]firestore.FieldPath
	}

	mpaDataWithID := &mpaData{
		customerRef: &firestore.DocumentRef{
			ID: "123",
		},
		assetType: "",
	}

	mpaDataEmpty := &mpaData{
		customerRef: &firestore.DocumentRef{},
		assetType:   "",
	}

	assetData := &assetData{
		assetSettingsRef: &firestore.DocumentRef{},
		asset: &amazonwebservices.Asset{
			Properties: &pkg.AWSProperties{},
		},
		assetSettings: &pkg.AWSAssetSettings{
			BaseAsset: pkg.BaseAsset{
				Customer: &firestore.DocumentRef{
					ID: "123",
				},
				Entity: &firestore.DocumentRef{},
			},
		},
	}

	paths := []firestore.FieldPath{}

	ctx := context.Background()

	logging, err := logger.NewLogging(ctx)
	if err != nil {
		t.Errorf("main: could not initialize logging. error %s", err)
	}

	conn, err := connection.NewConnection(ctx, logging)
	if err != nil {
		t.Errorf("main: could not initialize db connections. error %s", err)
	}

	batch := mocks.Batch{}
	batch.On("Set", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.DocumentRef"), mock.AnythingOfType("map[string]interface {}"), mock.AnythingOfType("firestore.merge")).Return(errors.New("error")).Once()
	batch.On("Set", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.DocumentRef"), mock.AnythingOfType("map[string]interface {}"), mock.AnythingOfType("firestore.merge")).Return(nil).Once()

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Happy path",
			args: args{
				ctx:       ctx,
				batch:     &batch,
				mpaData:   mpaDataWithID,
				assetData: assetData,
				paths:     &paths,
			},
		},
		{
			name: "Error on batch.Set",
			args: args{
				ctx:       ctx,
				batch:     &batch,
				mpaData:   mpaDataEmpty,
				assetData: assetData,
				paths:     &paths,
			},
			wantErr: true,
		},
		{
			name: "Happy path - else",
			args: args{
				ctx:       ctx,
				batch:     &batch,
				mpaData:   mpaDataEmpty,
				assetData: assetData,
				paths:     &paths,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AWSAssetsService{conn: conn}

			if err := s.buildPathsCustomer(tt.args.ctx, tt.args.batch, tt.args.mpaData, tt.args.assetData, tt.args.paths); (err != nil) != tt.wantErr {
				t.Errorf("AWSAssetsService.buildPathsCustomer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAWSAssetsService_buildPaths(t *testing.T) {
	type args struct {
		ctx       context.Context
		batch     iface.Batch
		mpaData   *mpaData
		assetData *assetData
		paths     *[]firestore.FieldPath
	}

	mpaDataWithID := &mpaData{
		customerRef: &firestore.DocumentRef{
			ID: "123",
		},
		assetType: "",
	}

	mpaDataEmpty := &mpaData{
		customerRef: &firestore.DocumentRef{
			ID: fb.Orphan.ID,
		},
		assetType: "",
	}

	assetData := &assetData{
		assetSettingsRef: &firestore.DocumentRef{},
		asset: &amazonwebservices.Asset{
			Properties: &pkg.AWSProperties{},
		},
		assetSettings: &pkg.AWSAssetSettings{
			BaseAsset: pkg.BaseAsset{
				Customer: &firestore.DocumentRef{
					ID: "123",
				},
			},
		},
	}

	paths := []firestore.FieldPath{}

	ctx := context.Background()

	logging, err := logger.NewLogging(ctx)
	if err != nil {
		t.Errorf("main: could not initialize logging. error %s", err)
	}

	conn, err := connection.NewConnection(ctx, logging)
	if err != nil {
		t.Errorf("main: could not initialize db connections. error %s", err)
	}

	batch := mocks.Batch{}
	batch.On("Set", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.DocumentRef"), mock.AnythingOfType("map[string]interface {}"), mock.AnythingOfType("firestore.merge")).Return(nil).Once()
	batch.On("Set", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.DocumentRef"), mock.AnythingOfType("map[string]interface {}"), mock.AnythingOfType("firestore.merge")).Return(errors.New("error")).Once()
	batch.On("Set", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.DocumentRef"), mock.AnythingOfType("map[string]interface {}"), mock.AnythingOfType("firestore.merge")).Return(nil).Once()
	batch.On("Set", testutils.ContextBackgroundMock, mock.AnythingOfType("*firestore.DocumentRef"), mock.AnythingOfType("map[string]interface {}"), mock.AnythingOfType("firestore.merge")).Return(errors.New("error")).Once()

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Happy path",
			args: args{
				ctx:       ctx,
				batch:     &batch,
				mpaData:   mpaDataWithID,
				assetData: assetData,
				paths:     &paths,
			},
		},
		{
			name: "error on buildPathsCustomer",
			args: args{
				ctx:       ctx,
				batch:     &batch,
				mpaData:   mpaDataWithID,
				assetData: assetData,
				paths:     &paths,
			},
			wantErr: true,
		},
		{
			name: "Happy path - else",
			args: args{
				ctx:       ctx,
				batch:     &batch,
				mpaData:   mpaDataEmpty,
				assetData: assetData,
				paths:     &paths,
			},
		},
		{
			name: "error on buildPathsNoCustomer",
			args: args{
				ctx:       ctx,
				batch:     &batch,
				mpaData:   mpaDataEmpty,
				assetData: assetData,
				paths:     &paths,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &AWSAssetsService{
				loggerProvider: logger.FromContext,
				conn:           conn,
			}
			if err := s.buildPaths(tt.args.ctx, tt.args.batch, tt.args.mpaData, tt.args.assetData, tt.args.paths); (err != nil) != tt.wantErr {
				t.Errorf("AWSAssetsService.buildPaths() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
