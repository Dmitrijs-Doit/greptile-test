package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/auth"
	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	assetsDalMocks "github.com/doitintl/hello/scheduled-tasks/assets/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	bucketsDalMocks "github.com/doitintl/hello/scheduled-tasks/buckets/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/mocks"
	attributionServiceMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service/attributiontier/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	metadataIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/iface"
	metadataServiceMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/mocks"
	attributionQueryMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/mocks"
	reportsDalMock "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDalMock "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	entityDalMocks "github.com/doitintl/hello/scheduled-tasks/entity/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

const (
	email                = "requester@example.com"
	userID               = "test_user_id"
	customerID           = "test_customer_id"
	successAttributionID = "create_attribution_success"
	attributionID        = "attributionID"
	attributionID2       = "attributionID2"
	newDescription       = "new description"
)

func TestAttributionsService_validateNotInReports(t *testing.T) {
	type fields struct {
		reportsDal *reportsDalMock.Reports
	}

	type args struct {
		customerID     string
		requesterEmail string
		attributionID  string
	}

	ctx := context.Background()

	values := []string{attributionID}

	requesterEmail := "user@doit.com"

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "success - attribution not in reports",
			args: args{
				customerID:     "customerID",
				requesterEmail: requesterEmail,
				attributionID:  attributionID,
			},
			wantErr: false,
			on: func(f *fields) {
				f.reportsDal.On("GetCustomerReports", ctx, "customerID").Return([]*report.Report{
					{
						Config: &report.Config{
							Filters: []*report.ConfigFilter{},
						},
					},
				}, nil)
			},
		},
		{
			name: "success - attribution found in reports",
			args: args{
				customerID:     customerID,
				requesterEmail: requesterEmail,
				attributionID:  attributionID,
			},
			wantErr:     true,
			expectedErr: ErrAttrsExistReports,
			on: func(f *fields) {
				f.reportsDal.On("GetCustomerReports", ctx, customerID).Return([]*report.Report{
					{
						Config: &report.Config{
							Filters: []*report.ConfigFilter{
								{
									BaseConfigFilter: report.BaseConfigFilter{
										ID:     "attribution:attribution",
										Values: &values,
									},
								},
							},
						},
					},
				}, nil)
			},
		},
		{
			name: "err on empty customerID",
			args: args{
				customerID:     "",
				requesterEmail: requesterEmail,
				attributionID:  attributionID,
			},
			wantErr:     true,
			expectedErr: errors.New("invalid customer id"),
		},
		{
			name: "err on empty attributionID",
			args: args{
				customerID:     customerID,
				requesterEmail: requesterEmail,
				attributionID:  "",
			},
			wantErr:     true,
			expectedErr: errors.New("invalid attribution id"),
		},
		{
			name: "err on get customer reports",
			args: args{
				customerID:     customerID,
				requesterEmail: requesterEmail,
				attributionID:  attributionID,
			},
			wantErr:     true,
			expectedErr: errors.New("error getting customer reports"),
			on: func(f *fields) {
				f.reportsDal.On("GetCustomerReports", ctx, customerID).Return(nil, errors.New("error getting customer reports"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				reportsDal: &reportsDalMock.Reports{},
			}

			s := &AttributionsService{
				reportsDal: tt.fields.reportsDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			if _, err := s.validateNotInReports(
				ctx,
				tt.args.customerID,
				tt.args.requesterEmail,
				tt.args.attributionID,
			); err != nil {
				if !tt.wantErr || err.Error() != tt.expectedErr.Error() {
					t.Errorf("AttributionsService.validateNotInReports() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestAttributionService_CreateAttribution(t *testing.T) {
	type fields struct {
		loggerProviderMock *loggerMocks.ILogger
		dal                *mocks.Attributions
		customersDal       *customerDalMock.Customers
		reportsDal         *reportsDalMock.Reports
		metadataService    *metadataServiceMock.MetadataIface
		attributionsQuery  *attributionQueryMock.IAttributionQuery
	}

	type args struct {
		ctx                  context.Context
		createAttributionReq *CreateAttributionRequest
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, common.CtxKeys.DoitEmployee, true)
	ctx = context.WithValue(ctx, common.CtxKeys.Email, email)

	log, err := logger.NewLogging(ctx)
	if err != nil {
		t.Fatal(err)
	}

	conn, err := connection.NewConnection(ctx, log)
	if err != nil {
		t.Fatal(err)
	}

	attributionReq := &CreateAttributionRequest{
		CustomerID: customerID,
		UserID:     userID,
		Email:      email,
		Attribution: attribution.Attribution{
			Filters: []report.BaseConfigFilter{
				{
					Key:  "key",
					Type: "type",
				},
			},
		},
	}

	attributionReqWithAttributionFilter := &CreateAttributionRequest{
		CustomerID: customerID,
		UserID:     userID,
		Email:      email,
		Attribution: attribution.Attribution{
			Filters: []report.BaseConfigFilter{
				{
					Key:  "attribution",
					Type: "attribution",
				},
			},
		},
	}

	customer := &common.Customer{
		Snapshot: &firestore.DocumentSnapshot{
			Ref: &firestore.DocumentRef{
				ID: customerID,
			},
		},
		EarlyAccessFeatures: []string{},
	}

	timeNow := time.Now()

	attributionSuccessResponse := &attribution.Attribution{
		ID:           successAttributionID,
		TimeCreated:  timeNow,
		TimeModified: timeNow,
	}
	expectedAttribution := &attribution.AttributionAPI{
		ID:         successAttributionID,
		CreateTime: timeNow.UnixMilli(),
		UpdateTime: timeNow.UnixMilli(),
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error

		on func(*fields)
	}{
		{
			name: "successfully create attribution",
			args: args{
				ctx:                  ctx,
				createAttributionReq: attributionReq,
			},
			wantErr: false,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.metadataService.On("ExternalAPIList", mock.AnythingOfType("iface.ExternalAPIListArgs")).
					Return(metadataIface.ExternalAPIListRes{
						metadataIface.ExternalAPIListItem{
							ID:   "key",
							Type: "type",
						},
					},
						nil,
					)
				f.attributionsQuery.On("ValidateFormula", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("int"), mock.AnythingOfType("string")).Return(nil)
				f.customersDal.On("GetCustomer", ctx, customerID).Return(customer, nil)
				f.customersDal.On("GetCustomerOrPresentationModeCustomer", ctx, customerID).Return(customer, nil)
				f.dal.On("CreateAttribution", ctx, mock.AnythingOfType("*attribution.Attribution")).Return(attributionSuccessResponse, nil)
			},
			expectedErr: nil,
		},
		{
			name: "create attribution with invalid filter key",
			args: args{
				ctx:                  ctx,
				createAttributionReq: attributionReq,
			},
			wantErr: true,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.customersDal.On("GetCustomerOrPresentationModeCustomer", ctx, customerID).Return(customer, nil)
				f.metadataService.On("ExternalAPIList", mock.AnythingOfType("iface.ExternalAPIListArgs")).
					Return(metadataIface.ExternalAPIListRes{
						metadataIface.ExternalAPIListItem{
							ID:   "other_key",
							Type: "type",
						},
					},
						nil,
					)
			},
			expectedErr: errors.New("filter 1 is not valid"),
		},
		{
			name: "create attribution with invalid filter type",
			args: args{
				ctx:                  ctx,
				createAttributionReq: attributionReq,
			},
			wantErr: true,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.customersDal.On("GetCustomerOrPresentationModeCustomer", ctx, customerID).Return(customer, nil)
				f.metadataService.On("ExternalAPIList", mock.AnythingOfType("iface.ExternalAPIListArgs")).
					Return(metadataIface.ExternalAPIListRes{
						metadataIface.ExternalAPIListItem{
							ID:   "id",
							Type: "other_type",
						},
					},
						nil,
					)
			},
			expectedErr: errors.New("filter 1 is not valid"),
		},
		{
			name: "create attribution with invalid filter type attribution",
			args: args{
				ctx:                  ctx,
				createAttributionReq: attributionReqWithAttributionFilter,
			},
			wantErr: true,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.customersDal.On("GetCustomerOrPresentationModeCustomer", ctx, customerID).Return(customer, nil)
				f.metadataService.On("ExternalAPIList", mock.AnythingOfType("iface.ExternalAPIListArgs")).
					Return(metadataIface.ExternalAPIListRes{
						metadataIface.ExternalAPIListItem{
							ID:   "attribution",
							Type: "attribution",
						},
					},
						nil,
					)
			},
			expectedErr: errors.New("filter 1 is not valid"),
		},
		{
			name: "create attribution with invalid filter type attributiongroup",
			args: args{
				ctx:                  ctx,
				createAttributionReq: attributionReqWithAttributionFilter,
			},
			wantErr: true,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.customersDal.On("GetCustomerOrPresentationModeCustomer", ctx, customerID).Return(customer, nil)
				f.metadataService.On("ExternalAPIList", mock.AnythingOfType("iface.ExternalAPIListArgs")).
					Return(metadataIface.ExternalAPIListRes{
						metadataIface.ExternalAPIListItem{
							ID:   "attribution",
							Type: "attribution_group",
						},
					},
						nil,
					)
			},
			expectedErr: errors.New("filter 1 is not valid"),
		},
		{
			name: "create attribution with invalid formula",
			args: args{
				ctx:                  ctx,
				createAttributionReq: attributionReq,
			},
			wantErr: true,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.customersDal.On("GetCustomerOrPresentationModeCustomer", ctx, customerID).Return(customer, nil)
				f.metadataService.On("ExternalAPIList", mock.AnythingOfType("iface.ExternalAPIListArgs")).
					Return(metadataIface.ExternalAPIListRes{
						metadataIface.ExternalAPIListItem{
							ID:   "key",
							Type: "type",
						},
					},
						nil,
					)
				f.attributionsQuery.On("ValidateFormula", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("int"), mock.AnythingOfType("string")).Return(errors.New("invalid formula"))
			},
			expectedErr: errors.New("invalid formula"),
		},
		{
			name: "create attribution getCustomer error",
			args: args{
				ctx:                  ctx,
				createAttributionReq: attributionReq,
			},
			wantErr: true,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.customersDal.On("GetCustomerOrPresentationModeCustomer", ctx, customerID).Return(customer, nil)
				f.metadataService.On("ExternalAPIList", mock.AnythingOfType("iface.ExternalAPIListArgs")).
					Return(metadataIface.ExternalAPIListRes{
						metadataIface.ExternalAPIListItem{
							ID:   "key",
							Type: "type",
						},
					},
						nil,
					)
				f.attributionsQuery.On("ValidateFormula", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("int"), mock.AnythingOfType("string")).Return(nil)
				f.customersDal.On("GetCustomer", ctx, customerID).Return(nil, errors.New("get customer error"))
			},
			expectedErr: errors.New("get customer error"),
		},
		{
			name: "create attribution attributionsDal error",
			args: args{
				ctx:                  ctx,
				createAttributionReq: attributionReq,
			},
			wantErr: true,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.customersDal.On("GetCustomerOrPresentationModeCustomer", ctx, customerID).Return(customer, nil)
				f.metadataService.On("ExternalAPIList", mock.AnythingOfType("iface.ExternalAPIListArgs")).
					Return(metadataIface.ExternalAPIListRes{
						metadataIface.ExternalAPIListItem{
							ID:   "key",
							Type: "type",
						},
					},
						nil,
					)
				f.attributionsQuery.On("ValidateFormula", ctx, mock.AnythingOfType("*bigquery.Client"), mock.AnythingOfType("int"), mock.AnythingOfType("string")).Return(nil)
				f.customersDal.On("GetCustomer", ctx, customerID).Return(customer, nil)
				f.dal.On("CreateAttribution", ctx, mock.AnythingOfType("*attribution.Attribution")).Return(nil, errors.New("attributionsDal error"))
			},
			expectedErr: errors.New("attributionsDal error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProviderMock: &loggerMocks.ILogger{},
				dal:                &mocks.Attributions{},
				reportsDal:         &reportsDalMock.Reports{},
				customersDal:       &customerDalMock.Customers{},
				attributionsQuery:  &attributionQueryMock.IAttributionQuery{},
				metadataService:    &metadataServiceMock.MetadataIface{},
			}

			s := &AttributionsService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return tt.fields.loggerProviderMock
				},
				conn:             conn,
				dal:              tt.fields.dal,
				reportsDal:       tt.fields.reportsDal,
				customerDal:      tt.fields.customersDal,
				attributionQuery: tt.fields.attributionsQuery,
				metadataService:  tt.fields.metadataService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			got, err := s.CreateAttribution(tt.args.ctx, tt.args.createAttributionReq)
			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionsService.CreateAttribution() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			} else if !tt.wantErr {
				assert.Equal(t, expectedAttribution, got)
			}
		})
	}
}

func TestAttributionService_GetAttribution(t *testing.T) {
	type fields struct {
		dal *mocks.Attributions
	}

	type args struct {
		ctx            context.Context
		attributionsID string
		isDoitEmployee bool
	}

	ctx := context.Background()

	var invalidAccessRole collab.PublicAccess = "invalid"

	attr := &attribution.Attribution{
		ID: attributionID,
	}

	customerAttr := &attribution.Attribution{
		ID:       attributionID,
		Type:     "custom",
		Customer: &firestore.DocumentRef{ID: "not_valid_ID"},
	}

	accessAttribution := &attribution.Attribution{
		ID:       attributionID,
		Type:     "custom",
		Customer: &firestore.DocumentRef{ID: customerID},
		Access: collab.Access{
			Public: &invalidAccessRole,
			Collaborators: []collab.Collaborator{
				{
					Email: "otherEmail",
					Role:  collab.CollaboratorRoleOwner,
				},
			},
		},
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		expectedErr error
		expectedRes *attribution.AttributionAPI
		on          func(*fields)
	}{
		{
			name: "Success - Got attribution",
			args: args{
				ctx:            ctx,
				attributionsID: attributionID,
				isDoitEmployee: true,
			},
			expectedErr: nil,
			expectedRes: toAttributionAPIItem(attr),
			on: func(f *fields) {
				f.dal.On("GetAttribution", ctx, attr.ID).Return(attr, nil)
			},
		},
		{
			name: "Fail - Attribution not found",
			args: args{
				ctx:            ctx,
				attributionsID: attributionID,
				isDoitEmployee: true,
			},
			expectedErr: attribution.ErrNotFound,
			expectedRes: nil,
			on: func(f *fields) {
				f.dal.On("GetAttribution", ctx, attr.ID).Return(nil, attribution.ErrNotFound)
			},
		},
		{
			name: "Success - Wrong customer ID",
			args: args{
				ctx:            ctx,
				attributionsID: attributionID,
				isDoitEmployee: true,
			},
			expectedErr: ErrWrongCustomer,
			expectedRes: nil,
			on: func(f *fields) {
				f.dal.On("GetAttribution", ctx, attr.ID).Return(customerAttr, nil)
			},
		},
		{
			name: "Success - Missing permission to view", // invalid access role and not in collaborator list
			args: args{
				ctx:            ctx,
				attributionsID: attributionID,
				isDoitEmployee: false,
			},
			expectedErr: ErrMissingPermissions,
			expectedRes: nil,
			on: func(f *fields) {
				f.dal.On("GetAttribution", ctx, attr.ID).Return(accessAttribution, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				dal: &mocks.Attributions{},
			}

			s := &AttributionsService{
				dal: tt.fields.dal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			got, err := s.GetAttribution(tt.args.ctx, tt.args.attributionsID, tt.args.isDoitEmployee, customerID, email)

			assert.Equal(t, tt.expectedErr, err)
			assert.Equal(t, tt.expectedRes, got)
		})
	}
}

func TestAttributionService_GetAttributions(t *testing.T) {
	type fields struct {
		dal *mocks.Attributions
	}

	type args struct {
		ctx             context.Context
		attributionsIDs []string
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, common.CtxKeys.DoitEmployee, true)
	ctx = context.WithValue(ctx, common.CtxKeys.Email, email)

	attrRef1 := firestore.DocumentRef{}
	attrRef2 := firestore.DocumentRef{}

	var attributions []*attribution.Attribution

	someDalError := errors.New("some dal error")

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		expectedRes []*attribution.Attribution

		on func(*fields)
	}{
		{
			name: "return list of attributions",
			args: args{
				ctx:             ctx,
				attributionsIDs: []string{"111", "222"},
			},
			wantErr: false,
			on: func(f *fields) {
				f.dal.On("GetRef", ctx, "111").Return(&attrRef1)
				f.dal.On("GetRef", ctx, "222").Return(&attrRef2)
				f.dal.On("GetAttributions", ctx, []*firestore.DocumentRef{
					&attrRef1,
					&attrRef2,
				}).Return(attributions, nil)
			},
			expectedErr: nil,
			expectedRes: attributions,
		},
		{
			name: "return error on dal error",
			args: args{
				ctx:             ctx,
				attributionsIDs: []string{"111", "222"},
			},
			wantErr: true,
			on: func(f *fields) {
				f.dal.On("GetRef", ctx, "111").Return(&attrRef1)
				f.dal.On("GetRef", ctx, "222").Return(&attrRef2)
				f.dal.On("GetAttributions", ctx, []*firestore.DocumentRef{
					&attrRef1,
					&attrRef2,
				}).Return(nil, someDalError)
			},
			expectedErr: someDalError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				dal: &mocks.Attributions{},
			}

			s := &AttributionsService{
				dal: tt.fields.dal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			got, err := s.GetAttributions(tt.args.ctx, tt.args.attributionsIDs)
			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionsService.GetAttributions() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			} else if !tt.wantErr {
				assert.Equal(t, tt.expectedRes, got)
			}
		})
	}
}

func TestAttributionService_UpdateAttribution(t *testing.T) {
	type fields struct {
		loggerProviderMock *loggerMocks.ILogger
		dal                *mocks.Attributions
		customersDal       *customerDalMock.Customers
		reportsDal         *reportsDalMock.Reports
		metadataService    *metadataServiceMock.MetadataIface
		attributionsQuery  *attributionQueryMock.IAttributionQuery
	}

	type args struct {
		ctx                  context.Context
		updateAttributionReq *UpdateAttributionRequest
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, common.CtxKeys.DoitEmployee, true)
	ctx = context.WithValue(ctx, auth.CtxKeyVerifiedCustomerID, customerID)
	ctx = context.WithValue(ctx, common.CtxKeys.Email, email)

	log, err := logger.NewLogging(ctx)
	if err != nil {
		t.Fatal(err)
	}

	conn, err := connection.NewConnection(ctx, log)
	if err != nil {
		t.Fatal(err)
	}

	completeAttributionRequest, onlyNameRequest, onlyDescriptionRequest, onlyFormulaRequest, onlyFiltersRequest := setupUpdateAttributionData()

	fmt.Print(onlyFiltersRequest)

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error

		on func(*fields)
	}{
		{
			name: "successfully update complete attribution",
			args: args{
				ctx:                  ctx,
				updateAttributionReq: completeAttributionRequest.request,
			},
			wantErr: false,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.dal.On("GetAttribution", ctx, attributionID).Return(&attribution.Attribution{
					Customer: &firestore.DocumentRef{},
				}, nil)
				f.metadataService.On("ExternalAPIList", mock.AnythingOfType("iface.ExternalAPIListArgs")).
					Return(metadataIface.ExternalAPIListRes{
						metadataIface.ExternalAPIListItem{
							ID:   "key",
							Type: "type",
						},
					},
						nil,
					)
				f.attributionsQuery.On("ValidateFormula", ctx, mock.AnythingOfType("*bigquery.Client"), 1, completeAttributionRequest.request.Attribution.Formula).Return(nil)
				f.dal.On("UpdateAttribution", ctx, attributionID, completeAttributionRequest.firestoreUpdates).Return(nil)
			},
			expectedErr: nil,
		},
		{
			name: "successfully update only name",
			args: args{
				ctx:                  ctx,
				updateAttributionReq: onlyNameRequest.request,
			},
			wantErr: false,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.dal.On("GetAttribution", ctx, attributionID).Return(&attribution.Attribution{
					Customer: &firestore.DocumentRef{},
				}, nil)
				f.metadataService.On("ExternalAPIList", mock.Anything).Return(nil, errors.New("ExternalAPIList shouldn't be called"))
				f.attributionsQuery.On("ValidateFormula", ctx, mock.AnythingOfType("*bigquery.Client"), mock.Anything, mock.Anything).Return(errors.New("Validate formula shouldn't be called"))
				f.dal.On("UpdateAttribution", ctx, attributionID, onlyNameRequest.firestoreUpdates).Return(nil)
			},
			expectedErr: nil,
		},
		{
			name: "successfully update only description",
			args: args{
				ctx:                  ctx,
				updateAttributionReq: onlyDescriptionRequest.request,
			},
			wantErr: false,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.dal.On("GetAttribution", ctx, attributionID).Return(&attribution.Attribution{
					Customer: &firestore.DocumentRef{},
				}, nil)
				f.metadataService.On("ExternalAPIList", mock.Anything).Return(nil, errors.New("ExternalAPIList shouldn't be called"))
				f.attributionsQuery.On("ValidateFormula", ctx, mock.AnythingOfType("*bigquery.Client"), mock.Anything, mock.Anything).Return(errors.New("Validate formula shouldn't be called"))
				f.dal.On("UpdateAttribution", ctx, attributionID, onlyDescriptionRequest.firestoreUpdates).Return(nil)
			},
			expectedErr: nil,
		},
		{
			name: "successfully update only formula",
			args: args{
				ctx:                  ctx,
				updateAttributionReq: onlyFormulaRequest.request,
			},
			wantErr: false,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.dal.On("GetAttribution", ctx, attributionID).Return(&attribution.Attribution{
					Filters:  []report.BaseConfigFilter{{}, {}},
					Formula:  "other_formula",
					Customer: &firestore.DocumentRef{},
				}, nil)
				f.metadataService.On("ExternalAPIList", mock.Anything).Return(nil, errors.New("ExternalAPIList shouldn't be called"))
				f.attributionsQuery.On("ValidateFormula", ctx, mock.AnythingOfType("*bigquery.Client"), 2, onlyFormulaRequest.request.Attribution.Formula).Return(nil)
				f.dal.On("UpdateAttribution", ctx, attributionID, onlyFormulaRequest.firestoreUpdates).Return(nil)
			},
			expectedErr: nil,
		},
		{
			name: "successfully update only filters",
			args: args{
				ctx:                  ctx,
				updateAttributionReq: onlyFiltersRequest.request,
			},
			wantErr: false,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.dal.On("GetAttribution", ctx, attributionID).Return(&attribution.Attribution{
					Filters:  []report.BaseConfigFilter{{}, {}},
					Formula:  "other_formula",
					Customer: &firestore.DocumentRef{},
				}, nil)
				f.metadataService.On("ExternalAPIList", mock.AnythingOfType("iface.ExternalAPIListArgs")).
					Return(metadataIface.ExternalAPIListRes{
						metadataIface.ExternalAPIListItem{
							ID:   "key",
							Type: "type",
						},
					},
						nil,
					)
				f.attributionsQuery.On("ValidateFormula", ctx, mock.AnythingOfType("*bigquery.Client"), 1, "other_formula").Return(nil)
				f.dal.On("UpdateAttribution", ctx, attributionID, onlyFiltersRequest.firestoreUpdates).Return(nil)
			},
			expectedErr: nil,
		},
		{
			name: "error getting attribution",
			args: args{
				ctx:                  ctx,
				updateAttributionReq: completeAttributionRequest.request,
			},
			wantErr: true,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.dal.On("GetAttribution", ctx, attributionID).Return(nil, errors.New("error getting attributions"))
			},
			expectedErr: errors.New("error getting attributions"),
		},
		{
			name: "get attribution returns null",
			args: args{
				ctx:                  ctx,
				updateAttributionReq: completeAttributionRequest.request,
			},
			wantErr: true,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.dal.On("GetAttribution", ctx, attributionID).Return(nil, nil)
			},
			expectedErr: ErrNotFound,
		},
		{
			name: "error validating formula",
			args: args{
				ctx:                  ctx,
				updateAttributionReq: completeAttributionRequest.request,
			},
			wantErr: true,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.dal.On("GetAttribution", ctx, attributionID).Return(&attribution.Attribution{
					Customer: &firestore.DocumentRef{},
				}, nil)
				f.attributionsQuery.On("ValidateFormula", ctx, mock.AnythingOfType("*bigquery.Client"), 1, completeAttributionRequest.request.Attribution.Formula).Return(errors.New("invalid formula"))
			},
			expectedErr: errors.New("invalid formula"),
		},
		{
			name: "error validating filter key",
			args: args{
				ctx:                  ctx,
				updateAttributionReq: completeAttributionRequest.request,
			},
			wantErr: true,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.dal.On("GetAttribution", ctx, attributionID).Return(&attribution.Attribution{
					Customer: &firestore.DocumentRef{},
				}, nil)
				f.metadataService.On("ExternalAPIList", mock.AnythingOfType("iface.ExternalAPIListArgs")).
					Return(metadataIface.ExternalAPIListRes{
						metadataIface.ExternalAPIListItem{
							ID:   "other_key",
							Type: "type",
						},
					},
						nil,
					)
				f.attributionsQuery.On("ValidateFormula", ctx, mock.AnythingOfType("*bigquery.Client"), 1, completeAttributionRequest.request.Attribution.Formula).Return(nil)
			},
			expectedErr: errors.New("filter 1 is not valid"),
		},
		{
			name: "error validating filter type",
			args: args{
				ctx:                  ctx,
				updateAttributionReq: completeAttributionRequest.request,
			},
			wantErr: true,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.dal.On("GetAttribution", ctx, attributionID).Return(&attribution.Attribution{
					Customer: &firestore.DocumentRef{},
				}, nil)
				f.metadataService.On("ExternalAPIList", mock.AnythingOfType("iface.ExternalAPIListArgs")).
					Return(metadataIface.ExternalAPIListRes{
						metadataIface.ExternalAPIListItem{
							ID:   "id",
							Type: "other_type",
						},
					},
						nil,
					)
				f.attributionsQuery.On("ValidateFormula", ctx, mock.AnythingOfType("*bigquery.Client"), 1, completeAttributionRequest.request.Attribution.Formula).Return(nil)
			},
			expectedErr: errors.New("filter 1 is not valid"),
		},
		{
			name: "update attribution dal error",
			args: args{
				ctx:                  ctx,
				updateAttributionReq: completeAttributionRequest.request,
			},
			wantErr: true,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.dal.On("GetAttribution", ctx, attributionID).Return(&attribution.Attribution{
					Customer: &firestore.DocumentRef{},
				}, nil)
				f.metadataService.On("ExternalAPIList", mock.AnythingOfType("iface.ExternalAPIListArgs")).
					Return(metadataIface.ExternalAPIListRes{
						metadataIface.ExternalAPIListItem{
							ID:   "key",
							Type: "type",
						},
					},
						nil,
					)
				f.attributionsQuery.On("ValidateFormula", ctx, mock.AnythingOfType("*bigquery.Client"), 1, completeAttributionRequest.request.Attribution.Formula).Return(nil)
				f.dal.On("UpdateAttribution", ctx, attributionID, completeAttributionRequest.firestoreUpdates).Return(errors.New("update attribution dal error"))
			},
			expectedErr: errors.New("update attribution dal error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProviderMock: &loggerMocks.ILogger{},
				dal:                &mocks.Attributions{},
				reportsDal:         &reportsDalMock.Reports{},
				customersDal:       &customerDalMock.Customers{},
				attributionsQuery:  &attributionQueryMock.IAttributionQuery{},
				metadataService:    &metadataServiceMock.MetadataIface{},
			}

			s := &AttributionsService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return tt.fields.loggerProviderMock
				},
				conn:             conn,
				dal:              tt.fields.dal,
				reportsDal:       tt.fields.reportsDal,
				customerDal:      tt.fields.customersDal,
				attributionQuery: tt.fields.attributionsQuery,
				metadataService:  tt.fields.metadataService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			_, err := s.UpdateAttribution(tt.args.ctx, tt.args.updateAttributionReq)
			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionsService.UpdateAttribution() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			}
		})
	}
}

func TestAttributionService_UpdateAttributions(t *testing.T) {
	type fields struct {
		loggerProviderMock *loggerMocks.ILogger
		dal                *mocks.Attributions
		customersDal       *customerDalMock.Customers
		reportsDal         *reportsDalMock.Reports
		metadataService    *metadataServiceMock.MetadataIface
		attributionsQuery  *attributionQueryMock.IAttributionQuery
	}

	type args struct {
		ctx          context.Context
		customerID   string
		attributions []*attribution.Attribution
		userID       string
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, common.CtxKeys.DoitEmployee, true)
	ctx = context.WithValue(ctx, auth.CtxKeyVerifiedCustomerID, customerID)
	ctx = context.WithValue(ctx, common.CtxKeys.Email, email)

	log, err := logger.NewLogging(ctx)
	if err != nil {
		t.Fatal(err)
	}

	conn, err := connection.NewConnection(ctx, log)
	if err != nil {
		t.Fatal(err)
	}

	attributionsToUpdate := []*attribution.Attribution{
		{
			ID:          attributionID,
			Description: newDescription,
		},
		{
			ID:          attributionID2,
			Description: newDescription,
		},
	}

	req1 := UdateAttributionTestData{
		&UpdateAttributionRequest{
			CustomerID: customerID,
			UserID:     userID,
			Attribution: attribution.Attribution{
				ID:          attributionID,
				Description: newDescription,
			},
		},
		[]firestore.Update{
			{Path: "description", Value: newDescription},
		},
	}

	req2 := UdateAttributionTestData{
		&UpdateAttributionRequest{
			CustomerID: customerID,
			UserID:     userID,
			Attribution: attribution.Attribution{
				ID:          attributionID2,
				Description: newDescription,
			},
		},
		[]firestore.Update{
			{Path: "description", Value: newDescription},
		},
	}

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error
		on          func(*fields)
	}{
		{
			name: "successfully update attributions",
			args: args{
				ctx:          ctx,
				customerID:   customerID,
				userID:       userID,
				attributions: attributionsToUpdate,
			},
			wantErr: false,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.dal.On("GetAttribution", ctx, attributionID).Return(&attribution.Attribution{
					Customer: &firestore.DocumentRef{},
				}, nil)
				f.dal.On("GetAttribution", ctx, attributionID2).Return(&attribution.Attribution{
					Customer: &firestore.DocumentRef{},
				}, nil)
				f.dal.On("UpdateAttribution", ctx, attributionID, req1.firestoreUpdates).Return(nil)
				f.dal.On("UpdateAttribution", ctx, attributionID2, req2.firestoreUpdates).Return(nil)
			},
			expectedErr: nil,
		},
		{
			name: "error getting attributions",
			args: args{
				ctx:          ctx,
				customerID:   customerID,
				userID:       userID,
				attributions: attributionsToUpdate,
			},
			wantErr: true,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.dal.On("GetAttribution", ctx, attributionID).Return(nil, errors.New("error getting attributions"))
			},
			expectedErr: errors.New("error getting attributions"),
		},
		{
			name: "get attribution returns null",
			args: args{
				ctx:          ctx,
				customerID:   customerID,
				userID:       userID,
				attributions: attributionsToUpdate,
			},
			wantErr: true,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.dal.On("GetAttribution", ctx, attributionID).Return(nil, nil)
			},
			expectedErr: ErrNotFound,
		},
		{
			name: "update attribution dal error",
			args: args{
				ctx:          ctx,
				customerID:   customerID,
				userID:       userID,
				attributions: attributionsToUpdate,
			},
			wantErr: true,
			on: func(f *fields) {
				f.loggerProviderMock.On("Warningf", mock.AnythingOfType("string")).Once()
				f.dal.On("GetAttribution", ctx, attributionID).Return(&attribution.Attribution{
					Customer: &firestore.DocumentRef{},
				}, nil)
				f.dal.On("UpdateAttribution", ctx, attributionID, req1.firestoreUpdates).Return(errors.New("update attribution dal error"))
			},
			expectedErr: errors.New("update attribution dal error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProviderMock: &loggerMocks.ILogger{},
				dal:                &mocks.Attributions{},
				reportsDal:         &reportsDalMock.Reports{},
				customersDal:       &customerDalMock.Customers{},
				attributionsQuery:  &attributionQueryMock.IAttributionQuery{},
				metadataService:    &metadataServiceMock.MetadataIface{},
			}

			s := &AttributionsService{
				loggerProvider: func(ctx context.Context) logger.ILogger {
					return tt.fields.loggerProviderMock
				},
				conn:             conn,
				dal:              tt.fields.dal,
				reportsDal:       tt.fields.reportsDal,
				customerDal:      tt.fields.customersDal,
				attributionQuery: tt.fields.attributionsQuery,
				metadataService:  tt.fields.metadataService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			_, err := s.UpdateAttributions(tt.args.ctx, tt.args.customerID, tt.args.attributions, tt.args.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionsService.UpdateAttributions() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			}
		})
	}
}

func TestAttributionService_ListAttributions(t *testing.T) {
	type fields struct {
		dal                    *mocks.Attributions
		customersDal           *customerDalMock.Customers
		reportsDal             *reportsDalMock.Reports
		metadataService        *metadataServiceMock.MetadataIface
		attributionsQuery      *attributionQueryMock.IAttributionQuery
		attributionTierService *attributionServiceMock.AttributionTierService
	}

	type args struct {
		ctx                     context.Context
		listAttributionsRequest *customerapi.Request
	}

	t0, _ := time.Parse("2006-01-02", "2021-01-01")
	t1, _ := time.Parse("2006-01-02", "2022-01-01")
	t2, _ := time.Parse("2006-01-02", "2023-01-01")

	attr := attribution.AttributionListItem{
		ID:         "1",
		CreateTime: t0.UnixMilli(),
		UpdateTime: t0.UnixMilli(),
	}

	attr1 := attribution.AttributionListItem{
		ID:         "2",
		CreateTime: t1.UnixMilli(),
		UpdateTime: t1.UnixMilli(),
	}

	attr2 := attribution.AttributionListItem{
		ID:         "3",
		CreateTime: t2.UnixMilli(),
		UpdateTime: t2.UnixMilli(),
	}

	req := &customerapi.Request{
		SortBy:     "createTime",
		SortOrder:  firestore.Desc,
		CustomerID: customerID,
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, common.CtxKeys.DoitEmployee, true)
	ctx = context.WithValue(ctx, auth.CtxKeyVerifiedCustomerID, customerID)
	ctx = context.WithValue(ctx, common.CtxKeys.Email, email)

	tests := []struct {
		name        string
		fields      fields
		args        args
		expectedErr error
		want        *attribution.AttributionsList

		on func(*fields)
	}{
		{
			name:        "Default list is ordered in reverse chronological order",
			fields:      fields{},
			args:        args{ctx: ctx, listAttributionsRequest: req},
			expectedErr: nil,
			want: &attribution.AttributionsList{
				RowCount:     3,
				Attributions: []customerapi.SortableItem{attr2, attr1, attr}, // Reverse chronological order
			},
			on: func(f *fields) {
				f.customersDal.On("GetRef", ctx, customerID).Return(&firestore.DocumentRef{})
				f.attributionTierService.
					On(
						"CheckAccessToCustomAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
				f.attributionTierService.
					On(
						"CheckAccessToPresetAttribution",
						ctx,
						customerID,
					).
					Return(nil, nil).
					Once()
				f.dal.On("ListAttributions", ctx, req, &firestore.DocumentRef{}).Return([]attribution.Attribution{{
					ID:           "1",
					TimeCreated:  t0,
					TimeModified: t0,
				}, {
					ID:           "2",
					TimeCreated:  t1,
					TimeModified: t1,
				}, {
					ID:           "3",
					TimeCreated:  t2,
					TimeModified: t2,
				}}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				dal:                    &mocks.Attributions{},
				reportsDal:             &reportsDalMock.Reports{},
				customersDal:           &customerDalMock.Customers{},
				attributionsQuery:      &attributionQueryMock.IAttributionQuery{},
				metadataService:        &metadataServiceMock.MetadataIface{},
				attributionTierService: &attributionServiceMock.AttributionTierService{},
			}

			s := &AttributionsService{
				dal:                    tt.fields.dal,
				reportsDal:             tt.fields.reportsDal,
				customerDal:            tt.fields.customersDal,
				attributionQuery:       tt.fields.attributionsQuery,
				metadataService:        tt.fields.metadataService,
				attributionTierService: tt.fields.attributionTierService,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			attributions, err := s.ListAttributions(tt.args.ctx, tt.args.listAttributionsRequest)
			if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			}

			assert.Equal(t, tt.want, attributions, "should be equal")
		})
	}
}

type UdateAttributionTestData struct {
	request          *UpdateAttributionRequest
	firestoreUpdates []firestore.Update
}

func setupUpdateAttributionData() (UdateAttributionTestData, UdateAttributionTestData, UdateAttributionTestData, UdateAttributionTestData, UdateAttributionTestData) {
	completeAttributionRequest :=
		UdateAttributionTestData{
			&UpdateAttributionRequest{
				CustomerID: customerID,
				UserID:     userID,
				Attribution: attribution.Attribution{
					ID:          attributionID,
					Name:        "name",
					Description: "description",
					Formula:     "test_formula",
					Filters: []report.BaseConfigFilter{
						{Key: "key", Type: "type"},
					},
				},
			},
			[]firestore.Update{
				{Path: "name", Value: "name"},
				{Path: "description", Value: "description"},
				{Path: "filters", Value: []report.BaseConfigFilter{
					{Key: "key", Type: "type"},
				}},
				{Path: "formula", Value: "test_formula"},
			},
		}

	onlyNameRequest :=
		UdateAttributionTestData{
			&UpdateAttributionRequest{
				CustomerID: customerID,
				UserID:     userID,
				Attribution: attribution.Attribution{
					ID:   attributionID,
					Name: "name",
				},
			},
			[]firestore.Update{
				{Path: "name", Value: "name"},
			},
		}

	onlyDescriptionRequest :=
		UdateAttributionTestData{
			&UpdateAttributionRequest{
				CustomerID: customerID,
				UserID:     userID,
				Attribution: attribution.Attribution{
					ID:          attributionID,
					Description: "description",
				},
			},
			[]firestore.Update{
				{Path: "description", Value: "description"},
			},
		}

	onlyFormulaRequest :=
		UdateAttributionTestData{
			&UpdateAttributionRequest{
				CustomerID: customerID,
				UserID:     userID,
				Attribution: attribution.Attribution{
					ID:      attributionID,
					Formula: "test_formula",
				},
			},
			[]firestore.Update{
				{Path: "formula", Value: "test_formula"},
			},
		}

	onlyFiltersRequest := UdateAttributionTestData{
		&UpdateAttributionRequest{
			CustomerID: customerID,
			UserID:     userID,
			Attribution: attribution.Attribution{
				ID: attributionID,
				Filters: []report.BaseConfigFilter{
					{Key: "key", Type: "type"},
				},
			},
		},
		[]firestore.Update{
			{Path: "filters", Value: []report.BaseConfigFilter{
				{Key: "key", Type: "type"},
			}},
		},
	}

	return completeAttributionRequest, onlyNameRequest, onlyDescriptionRequest, onlyFormulaRequest, onlyFiltersRequest
}

func TestAttributionService_toAttributionComponentAPI(t *testing.T) {
	a := &attribution.Attribution{
		Filters: []report.BaseConfigFilter{
			{
				Regexp: nil,
				Values: &[]string{
					"test",
				},
				Inverse:   true,
				AllowNull: true,
				Key:       "cloud_provider",
				Type:      metadata.MetadataFieldTypeFixed,
			},
			{
				Regexp: nil,
				Values: &[]string{
					"test",
				},
				Inverse:   false,
				AllowNull: true,
				Key:       "cloud_provider",
				Type:      metadata.MetadataFieldTypeFixed,
			},
		},
	}
	aa := toAttributionComponentAPI(a)

	if len(aa) != len(a.Filters) {
		t.Errorf("toAttributionComponentAPI expected length: %d, got: %d", len(a.Filters), len(aa))
	}

	if aa[0].AllowNull != a.Filters[0].AllowNull {
		t.Errorf("toAttributionComponentAPI expected allowNull: %v, got: %v", a.Filters[0].AllowNull, aa[0].AllowNull)
	}

	if aa[1].AllowNull != a.Filters[1].AllowNull {
		t.Errorf("toAttributionComponentAPI expected allowNull: %v, got: %v", a.Filters[0].AllowNull, aa[1].AllowNull)
	}
}

func TestAttributionService_validateName(t *testing.T) {
	s := &AttributionsService{}

	validLength := strings.Repeat("t", attrNameMaxLength)
	err := s.validateName(validLength)
	assert.Nil(t, err)

	tooLong := strings.Repeat("t", attrNameMaxLength+1)
	err = s.validateName(tooLong)
	assert.Equal(t, err, ErrNameTooLong)
}

func TestAttributionService_validateDescription(t *testing.T) {
	s := &AttributionsService{}

	validLength := strings.Repeat("t", attrDescMaxLength)
	err := s.validateDescription(validLength)
	assert.Nil(t, err)

	noDesc := ""
	err = s.validateDescription(noDesc)
	assert.Nil(t, err)

	tooLong := strings.Repeat("t", attrDescMaxLength+1)
	err = s.validateDescription(tooLong)
	assert.Equal(t, err, ErrDescriptionTooLong)
}

func TestAttributionService_CreateBucketAttribution(t *testing.T) {
	type fields struct {
		dal          *mocks.Attributions
		bucketsDal   *bucketsDalMocks.Buckets
		entityDal    *entityDalMocks.Entites
		assetsDal    *assetsDalMocks.Assets
		customersDal *customerDalMock.Customers
	}

	type args struct {
		ctx                  context.Context
		createAttributionReq *SyncBucketAttributionRequest
	}

	customer := &common.Customer{
		Snapshot: &firestore.DocumentSnapshot{
			Ref: &firestore.DocumentRef{
				ID: customerID,
			},
		},
	}

	createAttributionReq := &SyncBucketAttributionRequest{
		Customer: customer,
		Bucket: &common.Bucket{
			Name: "bucketName",
			Ref: &firestore.DocumentRef{
				ID: "bucketID",
			},
		},
		Entity: &common.Entity{
			Name:       "entityName",
			PriorityID: "123",
			Snapshot: &firestore.DocumentSnapshot{
				Ref: &firestore.DocumentRef{
					ID: "entityID",
				},
			},
		},
		Assets: []*pkg.BaseAsset{
			{
				ID:        fmt.Sprintf("%s-first-ID", common.Assets.GoogleCloud),
				AssetType: common.Assets.GoogleCloud,
			},
			{
				ID:        fmt.Sprintf("%s-second-ID", common.Assets.GoogleCloudProject),
				AssetType: common.Assets.GoogleCloudProject,
			},
			{
				ID:        fmt.Sprintf("%s-doitintl-fs-third-ID", common.Assets.GoogleCloudProject),
				AssetType: common.Assets.GoogleCloudProject,
			},
		},
	}

	expectedFilters := []report.BaseConfigFilter{
		{
			Key:       metadata.MetadataFieldKeyProjectID,
			Type:      metadata.MetadataFieldTypeFixed,
			Values:    &[]string{"second-ID"},
			ID:        "fixed:project_id",
			Field:     "T.project_id",
			AllowNull: false,
			Regexp:    nil,
			Inverse:   false,
		},
		{
			Key:       metadata.MetadataFieldKeyCmpFlexsaveProject,
			Type:      metadata.MetadataFieldTypeSystemLabel,
			Values:    &[]string{"doitintl-fs-third-ID"},
			ID:        "system_label:Y21wL2ZsZXhzYXZlX3Byb2plY3Q=",
			Field:     "T.system_labels",
			AllowNull: false,
			Regexp:    nil,
			Inverse:   false,
		},
		{
			Key:       metadata.MetadataFieldKeyBillingAccountID,
			Type:      metadata.MetadataFieldTypeFixed,
			Values:    &[]string{"first-ID"},
			ID:        "fixed:billing_account_id",
			Field:     "T.billing_account_id",
			AllowNull: false,
			Regexp:    nil,
			Inverse:   false,
		},
		{
			Key:       metadata.MetadataFieldKeyProjectID,
			Type:      metadata.MetadataFieldTypeFixed,
			ID:        "fixed:project_id",
			Field:     "T.project_id",
			AllowNull: true,
		},
		{
			Key:       metadata.MetadataFieldKeyServiceDescription,
			Type:      metadata.MetadataFieldTypeFixed,
			Values:    &[]string{"Looker"},
			ID:        "fixed:service_description",
			Field:     "T.service_description",
			AllowNull: false,
			Regexp:    nil,
			Inverse:   true,
		},
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, common.CtxKeys.DoitEmployee, true)
	ctx = context.WithValue(ctx, common.CtxKeys.Email, email)

	attributionSuccessResponse := &attribution.Attribution{
		ID: successAttributionID,
		Ref: &firestore.DocumentRef{
			ID: successAttributionID,
		},
	}

	publicRoleViewer := collab.PublicAccessView

	tests := []struct {
		name        string
		fields      fields
		args        args
		wantErr     bool
		expectedErr error

		on func(*fields)
	}{
		{
			name: "successfully create attribution",
			args: args{
				ctx,
				createAttributionReq,
			},
			wantErr: false,
			on: func(f *fields) {
				f.dal.On("CreateAttribution", ctx, &attribution.Attribution{
					Access: collab.Access{
						Collaborators: []collab.Collaborator{{Email: "doit.com", Role: collab.CollaboratorRoleOwner}},
						Public:        &publicRoleViewer},
					Customer:       customer.Snapshot.Ref,
					Type:           "managed",
					Classification: "invoice",
					Hidden:         true,
				}).Return(attributionSuccessResponse, nil)
				f.bucketsDal.On("UpdateBucket", ctx, createAttributionReq.Entity.Snapshot.Ref.ID, createAttributionReq.Bucket.Ref.ID, []firestore.Update{
					{Path: "attribution", Value: attributionSuccessResponse.Ref},
				}).Return(nil)
				f.dal.On("UpdateAttribution", ctx, attributionSuccessResponse.ID, []firestore.Update{
					{Path: "name", Value: "[123] entityName - bucketName"},
					{Path: "filters", Value: expectedFilters},
					{Path: "formula", Value: getAttributionFormula(expectedFilters)},
					{Path: "cloud", Value: []string{"google-cloud"}},
					{Path: "type", Value: "managed"},
					{Path: "classification", Value: "invoice"},
					{Path: "hidden", Value: true},
					{Path: "collaborators", Value: []collab.Collaborator{{Email: "doit.com", Role: collab.CollaboratorRoleOwner}}},
					{Path: "public", Value: &publicRoleViewer},
				}).Return(nil)
			},
			expectedErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				dal:          &mocks.Attributions{},
				customersDal: &customerDalMock.Customers{},
				bucketsDal:   &bucketsDalMocks.Buckets{},
				entityDal:    &entityDalMocks.Entites{},
				assetsDal:    &assetsDalMocks.Assets{},
			}

			s := &AttributionsService{
				dal:         tt.fields.dal,
				customerDal: tt.fields.customersDal,
				bucketsDal:  tt.fields.bucketsDal,
				entityDal:   tt.fields.entityDal,
				assetsDal:   tt.fields.assetsDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			got, err := s.CreateBucketAttribution(tt.args.ctx, tt.args.createAttributionReq)
			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionsService.CreateBucketAttribution() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			} else if !tt.wantErr {
				assert.Equal(t, attributionSuccessResponse.Ref, got)
			}
		})
	}
}

func TestAttributionService_getDefaultBucketAssets(t *testing.T) {
	type fields struct {
		assetsDal *assetsDalMocks.Assets
	}

	type args struct {
		ctx context.Context
		req *SyncBucketAttributionRequest
	}

	ctx := context.Background()

	testEntity := &common.Entity{
		Name: "entityName",
		Snapshot: &firestore.DocumentSnapshot{
			Ref: &firestore.DocumentRef{
				ID: "entityID",
			},
		},
	}

	getNonEmptyDefaultBucketAssets := &SyncBucketAttributionRequest{
		Entity: testEntity,
		Bucket: &common.Bucket{
			Name: "bucketName",
			Ref: &firestore.DocumentRef{
				ID: "bucketID",
			},
		},
		Assets: []*pkg.BaseAsset{
			{AssetType: common.Assets.AmazonWebServices, ID: "aws"},
		},
	}

	getEmptyDefaultBucketAssets := &SyncBucketAttributionRequest{
		Entity: testEntity,
		Bucket: &common.Bucket{
			Name: "bucketName",
			Ref: &firestore.DocumentRef{
				ID: "bucketID",
			},
		},
	}

	getAssetsInEntityMixedTypes := []*pkg.BaseAsset{
		{AssetType: common.Assets.AmazonWebServices, ID: "aws1"},
		{AssetType: common.Assets.GoogleCloud},
		{AssetType: common.Assets.GoogleCloudProject},
		{AssetType: common.Assets.GSuite},
	}

	getAssetsInEntityOneType := []*pkg.BaseAsset{
		{AssetType: common.Assets.AmazonWebServices, ID: "aws"},
		{AssetType: common.Assets.AmazonWebServices, ID: "aws1"},
	}

	getAssetsInEntityMixedTypesButJustOneUnassigned := []*pkg.BaseAsset{
		{AssetType: common.Assets.AmazonWebServices, ID: "aws"},
		{AssetType: common.Assets.AmazonWebServices, ID: "aws1"},
		{AssetType: common.Assets.GoogleCloud, Bucket: &firestore.DocumentRef{}},
		{AssetType: common.Assets.GoogleCloudProject, Bucket: &firestore.DocumentRef{}},
		{AssetType: common.Assets.GSuite},
	}

	entityUnassignedAssetsOfDifferentType := []*pkg.BaseAsset{
		{AssetType: common.Assets.AmazonWebServices, ID: "aws1"},
		{AssetType: common.Assets.GoogleCloudProject},
	}

	expectedAssetsForBucketRequest := []*pkg.BaseAsset{
		{AssetType: common.Assets.AmazonWebServices, ID: "aws"},
		{AssetType: common.Assets.AmazonWebServices, ID: "aws1"},
	}

	tests := []struct {
		name           string
		fields         fields
		args           args
		expectedAssets []*pkg.BaseAsset
		wantErr        bool
		expectedErr    error
		on             func(*fields)
	}{
		{
			name: "Successfully get assets for non empty default bucket",
			args: args{
				ctx,
				getNonEmptyDefaultBucketAssets,
			},
			expectedAssets: expectedAssetsForBucketRequest,
			on: func(f *fields) {
				f.assetsDal.On(
					"GetAssetsInEntity", ctx, getNonEmptyDefaultBucketAssets.Entity.Snapshot.Ref).
					Return(getAssetsInEntityMixedTypes, nil)
			},
		},
		{
			name: "Successfully get assets for empty default bucket",
			args: args{
				ctx,
				getEmptyDefaultBucketAssets,
			},
			expectedAssets: expectedAssetsForBucketRequest,
			on: func(f *fields) {
				f.assetsDal.On(
					"GetAssetsInEntity", ctx, getEmptyDefaultBucketAssets.Entity.Snapshot.Ref).
					Return(getAssetsInEntityOneType, nil)
			},
		},
		{
			name: "Successfully get assets for default bucket (all unassigned assets of same type)",
			args: args{
				ctx,
				getEmptyDefaultBucketAssets,
			},
			expectedAssets: expectedAssetsForBucketRequest,
			on: func(f *fields) {
				f.assetsDal.On(
					"GetAssetsInEntity", ctx, getEmptyDefaultBucketAssets.Entity.Snapshot.Ref).
					Return(getAssetsInEntityMixedTypesButJustOneUnassigned, nil)
			},
		},
		{
			name: "Return request assets and all unassigned assets of same type",
			args: args{
				ctx,
				getNonEmptyDefaultBucketAssets,
			},
			expectedAssets: expectedAssetsForBucketRequest,
			on: func(f *fields) {
				f.assetsDal.On(
					"GetAssetsInEntity", ctx, getEmptyDefaultBucketAssets.Entity.Snapshot.Ref).
					Return(entityUnassignedAssetsOfDifferentType, nil)
			},
		},
		{
			name: "If request assets is empty and unassigned assets have different type return nil",
			args: args{
				ctx,
				getEmptyDefaultBucketAssets,
			},
			expectedAssets: nil,
			on: func(f *fields) {
				f.assetsDal.On(
					"GetAssetsInEntity", ctx, getEmptyDefaultBucketAssets.Entity.Snapshot.Ref).
					Return(entityUnassignedAssetsOfDifferentType, nil)
			},
		},
		{
			name: "Error getting entity assets",
			args: args{
				ctx,
				getNonEmptyDefaultBucketAssets,
			},
			on: func(f *fields) {
				f.assetsDal.On("GetAssetsInEntity", ctx, mock.AnythingOfType("*firestore.DocumentRef")).Return(nil, errors.New("error getting assets in entity"))
			},
			wantErr:     true,
			expectedErr: errors.New("error getting assets in entity"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				assetsDal: &assetsDalMocks.Assets{},
			}

			s := &AttributionsService{
				assetsDal: tt.fields.assetsDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			assets, err := s.getDefaultBucketAssets(tt.args.ctx, tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionsService.getAttributionNameAndAssets() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			} else if !tt.wantErr {
				assert.Equal(t, tt.expectedAssets, assets)
			}
		})
	}
}

func TestAttributionService_getAttributionFormula(t *testing.T) {
	type args struct {
		filters []report.BaseConfigFilter
	}

	tests := []struct {
		name           string
		args           args
		expectedResult string
	}{
		{
			name: "formula for 0 filters",
			args: args{
				filters: []report.BaseConfigFilter{},
			},
			expectedResult: "",
		},
		{
			name: "formula for 1 filters",
			args: args{
				filters: []report.BaseConfigFilter{
					{},
				},
			},
			expectedResult: "A",
		},
		{
			name: "formula for 2 filters",
			args: args{
				filters: []report.BaseConfigFilter{
					{},
					{},
				},
			},
			expectedResult: "A OR B",
		},
		{
			name: "formula for billing account filters",
			args: args{
				filters: []report.BaseConfigFilter{
					{Key: metadata.MetadataFieldKeyBillingAccountID},
					{},
					{},
					{},
				},
			},
			expectedResult: "(A AND B AND C) OR D",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getAttributionFormula(tt.args.filters)
			assert.Equal(t, tt.expectedResult, got)
		})
	}
}

func TestAttributionService_generateAttributionFilters(t *testing.T) {
	type fields struct {
		assetsDal *assetsDalMocks.Assets
	}

	type args struct {
		assets []*pkg.BaseAsset
	}

	ctx := context.Background()
	gcpAssets := []*pkg.BaseAsset{
		{
			ID:        fmt.Sprintf("%s-first-ID", common.Assets.GoogleCloud),
			AssetType: common.Assets.GoogleCloud,
		},
		{
			ID:        fmt.Sprintf("%s-second-ID", common.Assets.GoogleCloudProject),
			AssetType: common.Assets.GoogleCloudProject,
		},
		{
			ID:        fmt.Sprintf("%s-doitintl-fs-third-ID", common.Assets.GoogleCloudProject),
			AssetType: common.Assets.GoogleCloudProject,
		},
	}

	awsAssetsNoPayerAccount := []*pkg.BaseAsset{
		{
			ID:        fmt.Sprintf("%s-fourth-ID", common.Assets.AmazonWebServices),
			AssetType: common.Assets.AmazonWebServices,
		},
	}

	awsAssetsPayerAccount := []*pkg.BaseAsset{
		{
			ID:        fmt.Sprintf("%s-fifth-ID", common.Assets.AmazonWebServices),
			AssetType: common.Assets.AmazonWebServices,
		},
	}

	expectedGCPFilters := []report.BaseConfigFilter{
		{
			Key:       metadata.MetadataFieldKeyProjectID,
			Type:      metadata.MetadataFieldTypeFixed,
			Values:    &[]string{"second-ID"},
			ID:        "fixed:project_id",
			Field:     "T.project_id",
			AllowNull: false,
			Regexp:    nil,
			Inverse:   false,
		},
		{
			Key:       metadata.MetadataFieldKeyCmpFlexsaveProject,
			Type:      metadata.MetadataFieldTypeSystemLabel,
			Values:    &[]string{"doitintl-fs-third-ID"},
			ID:        "system_label:Y21wL2ZsZXhzYXZlX3Byb2plY3Q=",
			Field:     "T.system_labels",
			AllowNull: false,
			Regexp:    nil,
			Inverse:   false,
		},
		{
			Key:       metadata.MetadataFieldKeyBillingAccountID,
			Type:      metadata.MetadataFieldTypeFixed,
			Values:    &[]string{"first-ID"},
			ID:        "fixed:billing_account_id",
			Field:     "T.billing_account_id",
			AllowNull: false,
			Regexp:    nil,
			Inverse:   false,
		},
		{
			Key:       metadata.MetadataFieldKeyProjectID,
			Type:      metadata.MetadataFieldTypeFixed,
			ID:        "fixed:project_id",
			Field:     "T.project_id",
			AllowNull: true,
		},
		{
			Key:       metadata.MetadataFieldKeyServiceDescription,
			Type:      metadata.MetadataFieldTypeFixed,
			Values:    &[]string{"Looker"},
			ID:        "fixed:service_description",
			Field:     "T.service_description",
			AllowNull: false,
			Regexp:    nil,
			Inverse:   true,
		},
	}

	expectedAWSFiltersNoPayerAccount := []report.BaseConfigFilter{
		{
			Key:       metadata.MetadataFieldKeyProjectID,
			Type:      metadata.MetadataFieldTypeFixed,
			Values:    &[]string{"fourth-ID"},
			ID:        "fixed:project_id",
			Field:     "T.project_id",
			AllowNull: false,
			Regexp:    nil,
			Inverse:   false,
		},
	}

	expectedAWSFiltersPayerAccount := []report.BaseConfigFilter{
		{
			Key:       metadata.MetadataFieldKeyProjectID,
			Type:      metadata.MetadataFieldTypeFixed,
			Values:    &[]string{"fifth-ID"},
			ID:        "fixed:project_id",
			Field:     "T.project_id",
			AllowNull: false,
			Regexp:    nil,
			Inverse:   false,
		},
		{
			Key:    metadata.MetadataFieldKeyAwsPayerAccountID,
			Type:   metadata.MetadataFieldTypeSystemLabel,
			Values: &[]string{"fifth-ID"},
			ID:     "system_label:YXdzL3BheWVyX2FjY291bnRfaWQ=",
			Field:  "T.system_labels",
		},
		{
			Key:    metadata.MetadataFieldKeyProjectName,
			Type:   metadata.MetadataFieldTypeFixed,
			Values: &[]string{"Flexsave"},
			ID:     "fixed:project_name",
			Field:  "T.project_name",
		},
	}

	tests := []struct {
		name            string
		args            args
		expectedFilters []report.BaseConfigFilter
		wantErr         bool
		expectedErr     error
		fields          fields
		on              func(*fields)
	}{
		{
			name: "Successfully generate gcp filters",
			args: args{
				gcpAssets,
			},
			expectedFilters: expectedGCPFilters,
		},
		{
			name: "Successfully generate aws filters no payer account",
			args: args{
				awsAssetsNoPayerAccount,
			},
			expectedFilters: expectedAWSFiltersNoPayerAccount,
			on: func(f *fields) {
				f.assetsDal.On("GetAWSAsset", ctx, awsAssetsNoPayerAccount[0].ID).Return(&pkg.AWSAsset{
					Properties: &pkg.AWSProperties{
						AccountID: "4",
					},
				}, nil)
			},
		},
		{
			name: "Successfully generate aws filters with payer account",
			args: args{
				awsAssetsPayerAccount,
			},
			expectedFilters: expectedAWSFiltersPayerAccount,
			on: func(f *fields) {
				f.assetsDal.On("GetAWSAsset", ctx, awsAssetsPayerAccount[0].ID).Return(&pkg.AWSAsset{
					Properties: &pkg.AWSProperties{
						AccountID: "5",
						OrganizationInfo: &pkg.OrganizationInfo{
							PayerAccount: &domain.PayerAccount{
								AccountID: "5",
							},
						},
					},
				}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				assetsDal: &assetsDalMocks.Assets{},
			}

			s := &AttributionsService{
				assetsDal: tt.fields.assetsDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			got, err := s.generateInvoiceAttributionFilters(ctx, tt.args.assets)
			if (err != nil) != tt.wantErr {
				t.Errorf("AttributionsService.generateAttributionFilters() error = %v, wantErr %v", err, tt.wantErr)
			} else if tt.expectedErr != nil {
				assert.Equal(t, tt.expectedErr, err)
			} else if !tt.wantErr {
				assert.Equal(t, tt.expectedFilters, got)
			}
		})
	}
}
