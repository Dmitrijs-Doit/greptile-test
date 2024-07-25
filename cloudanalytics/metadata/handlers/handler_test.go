package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/auth"
	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func TestMetadataHandler_AttributionGroupsMetadata(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	email := "test@doit.com"
	customerID := "123"

	type fields struct {
		service *mocks.MetadataIface
	}

	tests := []struct {
		name         string
		on           func(*fields)
		wantedStatus int
		wantErr      bool
	}{
		{
			name: "successful return of attribution groups metadata",
			on: func(f *fields) {
				f.service.On("AttributionGroupsMetadata", mock.AnythingOfType("*gin.Context"), customerID, email).
					Return([]*domain.OrgMetadataModel{}, nil).
					Once()
			},
			wantedStatus: http.StatusOK,
		},
		{
			name: "error return of attribution groups metadata",
			on: func(f *fields) {
				f.service.On("AttributionGroupsMetadata", mock.AnythingOfType("*gin.Context"), customerID, email).
					Return(nil, errors.New("error returning attribution groups metadata")).
					Once()
			},
			wantedStatus: http.StatusInternalServerError,
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				service: &mocks.MetadataIface{},
			}

			h := &AnalyticsMetadata{
				service: fields.service,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			request := httptest.NewRequest(http.MethodGet, "/someRequest", nil)

			ctx.Set("email", email)
			ctx.Params = []gin.Param{
				{Key: "customerID", Value: customerID},
			}
			ctx.Request = request

			respond := h.AttributionGroupsMetadata(ctx)
			if (respond != nil) != tt.wantErr {
				t.Errorf("Metadata.AttributionGroupsMetadata() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestMetadataHandler_ExternalAPIGetDimensions(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		service        *mocks.MetadataIface
	}

	type args struct {
		typeFilter     string
		idFilter       string
		customerId     string
		userID         string
		isDoitEmployee bool
	}

	tests := []struct {
		name      string
		on        func(*fields)
		wantedErr error
		args      args
	}{
		{
			name: "successful return of external api metadata",
			args: args{
				typeFilter:     "type1",
				idFilter:       "key1",
				customerId:     "some-customer-id-1",
				userID:         "some-user-id-1",
				isDoitEmployee: true,
			},
			on: func(f *fields) {
				f.service.On("ExternalAPIGet",
					mock.MatchedBy(func(args iface.ExternalAPIGetArgs) bool {
						return args.CustomerID == "some-customer-id-1" && args.UserID == "some-user-id-1" && args.IsDoitEmployee == true && args.KeyFilter == "key1" && args.TypeFilter == "type1"
					})).
					Return(&iface.ExternalAPIGetRes{
						ID:    "key1",
						Type:  "type1",
						Label: "label1",
						Values: []iface.ExternalAPIGetValue{
							{Value: "test", Cloud: "cloud"},
						},
					}, nil)
			},
		},
		{
			name: "not found error return of external api metadata",
			args: args{
				typeFilter:     "type2",
				idFilter:       "key2",
				customerId:     "some-customer-id-2",
				userID:         "some-user-id-2",
				isDoitEmployee: false,
			},
			on: func(f *fields) {
				f.service.On("ExternalAPIGet", mock.MatchedBy(func(args iface.ExternalAPIGetArgs) bool {
					return args.CustomerID == "some-customer-id-2" && args.UserID == "some-user-id-2" && args.IsDoitEmployee == false && args.KeyFilter == "key2" && args.TypeFilter == "type2"
				})).Return(nil, metadata.ErrNotFound)
			},
			wantedErr: metadata.ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				loggerProvider: logger.FromContext,
				service:        &mocks.MetadataIface{},
			}

			h := &AnalyticsMetadata{
				loggerProvider: fields.loggerProvider,
				service:        fields.service,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/someRequest", nil)

			ctx.Set(auth.CtxKeyVerifiedCustomerID, tt.args.customerId)
			ctx.Set(common.CtxKeys.UserID, tt.args.userID)
			ctx.Set(common.CtxKeys.DoitEmployee, tt.args.isDoitEmployee)
			ctx.Set(common.CtxKeys.Email, "test@doit.com")

			reqQuery := "?type=" + tt.args.typeFilter + "&id=" + tt.args.idFilter
			ctx.Request.URL, _ = url.Parse(reqQuery)

			retVal := h.ExternalAPIGetDimensions(ctx)
			if retVal != nil && tt.wantedErr == nil {
				t.Errorf("Metadata.ExternalAPIGetDimensions() error = %v, wantErr %v", retVal, tt.wantedErr)
			}

			if tt.wantedErr != nil {
				assert.Equal(t, tt.wantedErr.Error(), retVal.Error())
			}
		})
	}
}

func TestMetadataHandler_ExternalAPIListDimensions(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		service        *mocks.MetadataIface
	}

	type args struct {
		customerId     string
		userID         string
		isDoitEmployee bool
	}

	tests := []struct {
		name      string
		on        func(*fields)
		wantedErr error
		args      args
	}{
		{
			name: "successful return of external api metadata",
			args: args{
				customerId:     "some-customer-id-1",
				userID:         "some-user-id-1",
				isDoitEmployee: true,
			},
			on: func(f *fields) {
				f.service.On("ExternalAPIListWithFilters", mock.MatchedBy(func(args iface.ExternalAPIListArgs) bool {
					return args.CustomerID == "some-customer-id-1" && args.UserID == "some-user-id-1" && args.IsDoitEmployee == true
				}), mock.Anything).Return(&domain.DimensionsExternalAPIList{
					PageToken: "token",
					RowCount:  1,
					Dimensions: []customerapi.SortableItem{
						metadata.DimensionListItem{
							ID:    "id1",
							Type:  "type1",
							Label: "label1",
						},
					},
				}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				loggerProvider: logger.FromContext,
				service:        &mocks.MetadataIface{},
			}

			h := &AnalyticsMetadata{
				loggerProvider: fields.loggerProvider,
				service:        fields.service,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/someRequest", nil)

			ctx.Set(auth.CtxKeyVerifiedCustomerID, tt.args.customerId)
			ctx.Set(common.CtxKeys.UserID, tt.args.userID)
			ctx.Set(common.CtxKeys.DoitEmployee, tt.args.isDoitEmployee)
			ctx.Set(common.CtxKeys.Email, "test@doit.com")

			retVal := h.ExternalAPIListDimensions(ctx)
			if retVal != nil && tt.wantedErr == nil {
				t.Errorf("Metadata.ExternalAPIListDimensions() error = %v, wantErr %v", retVal, tt.wantedErr)
			}

			if tt.wantedErr != nil {
				assert.Equal(t, tt.wantedErr.Error(), retVal.Error())
			}
		})
	}
}

func TestMetadataHandler_UpdateAWSCustomersMetadata(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		service        *mocks.MetadataIface
	}

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr bool
	}{
		{
			name: "successful return of update aws customers metadata",
			on: func(f *fields) {
				f.service.On("UpdateAWSAllCustomersMetadata", mock.AnythingOfType("*gin.Context")).
					Return([]error{}, nil).
					Once()
			},
		},
		{
			name: "error return of update aws customers metadata",
			on: func(f *fields) {
				f.service.On("UpdateAWSAllCustomersMetadata", mock.AnythingOfType("*gin.Context")).
					Return([]error{}, errors.New("some error")).
					Once()
			},
			wantErr: true,
		},
		{
			name: "partial error return success",
			on: func(f *fields) {
				f.service.On("UpdateAWSAllCustomersMetadata", mock.AnythingOfType("*gin.Context")).
					Return([]error{errors.New("some error")}, nil).
					Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{
				loggerProvider: logger.FromContext,
				service:        &mocks.MetadataIface{},
			}

			h := &AnalyticsMetadata{
				loggerProvider: f.loggerProvider,
				service:        f.service,
			}

			if tt.on != nil {
				tt.on(&f)
			}

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/someRequest", nil)

			respond := h.UpdateAWSAllCustomersMetadata(ctx)
			if (respond != nil) && !tt.wantErr {
				t.Errorf("AnalyticsMetadata.UpdateAWSAllCustomersMetadata() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestMetadataHandler_UpdateAWSCustomerMetadata(t *testing.T) {
	// email := "test@doit.com"
	customerID := "123"

	type fields struct {
		loggerProvider logger.Provider
		service        *mocks.MetadataIface
	}

	type args struct {
		customerID string
	}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr bool
	}{
		{
			name: "successful return of update aws customer metadata",
			on: func(f *fields) {
				f.service.On("UpdateAWSCustomerMetadata", mock.AnythingOfType("*gin.Context"), customerID, []*common.Organization(nil)).
					Return(nil).
					Once()
			},
			args: args{
				customerID: customerID,
			},
		},
		{
			name:    "error no customer id",
			wantErr: true,
		},
		{
			name: "error return of update aws customer metadata",
			on: func(f *fields) {
				f.service.On("UpdateAWSCustomerMetadata", mock.AnythingOfType("*gin.Context"), customerID, []*common.Organization(nil)).
					Return(errors.New("some error")).
					Once()
			},
			wantErr: true,
			args: args{
				customerID: customerID,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{
				loggerProvider: logger.FromContext,
				service:        &mocks.MetadataIface{},
			}

			h := &AnalyticsMetadata{
				loggerProvider: f.loggerProvider,
				service:        f.service,
			}

			if tt.on != nil {
				tt.on(&f)
			}

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/someRequest", nil)

			ctx.Params = []gin.Param{
				{Key: "customerID", Value: tt.args.customerID},
			}

			respond := h.UpdateAWSCustomerMetadata(ctx)
			if (respond != nil) && !tt.wantErr {
				t.Errorf("AnalyticsMetadata.UpdateAWSAllCustomersMetadata() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}

func TestMetadataHandler_UpdateDataHubMetadata(t *testing.T) {
	type fields struct {
		loggerProvider logger.Provider
		service        *mocks.MetadataIface
	}

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr bool
	}{
		{
			name: "successful return of update datahub api metadata",
			on: func(f *fields) {
				f.service.On("UpdateDataHubMetadata", mock.AnythingOfType("*gin.Context")).
					Return(nil).
					Once()
			},
		},
		{
			name: "error return of update datahub api metadata",
			on: func(f *fields) {
				f.service.On("UpdateDataHubMetadata", mock.AnythingOfType("*gin.Context")).
					Return(errors.New("some error")).
					Once()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := fields{
				loggerProvider: logger.FromContext,
				service:        mocks.NewMetadataIface(t),
			}

			h := &AnalyticsMetadata{
				loggerProvider: f.loggerProvider,
				service:        f.service,
			}

			if tt.on != nil {
				tt.on(&f)
			}

			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodGet, "/someRequest", nil)

			respond := h.UpdateDataHubMetadata(ctx)
			if (respond != nil) && !tt.wantErr {
				t.Errorf("AnalyticsMetadata.UpdateDataHubMetadata() error = %v, wantErr %v", respond, tt.wantErr)
			}
		})
	}
}
