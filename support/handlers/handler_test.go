package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/support/service"
	serviceMock "github.com/doitintl/hello/scheduled-tasks/support/service/mocks"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func GetContext() *gin.Context {
	request := httptest.NewRequest(http.MethodPost, "http://example.com/foo", nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request

	return ctx
}

type fields struct {
	loggerProvider logger.Provider
	service        *serviceMock.SupportServiceInterface
}

func TestSupport_ListPlatformsHandler(t *testing.T) {
	ctx := GetContext()

	tests := []struct {
		name    string
		fields  fields
		on      func(*fields)
		wantErr error
	}{
		{
			name: "happy path",
			on: func(f *fields) {
				f.service.
					On(
						"ListPlatforms",
						ctx,
					).
					Return(&service.PlatformsAPI{}, nil).
					Once()
			},
		},
		{
			name:    "service returned error",
			wantErr: errors.New("some error"),
			on: func(f *fields) {
				f.service.
					On(
						"ListPlatforms",
						ctx,
					).
					Return(nil, errors.New("some error")).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider: logger.FromContext,
				service:        &serviceMock.SupportServiceInterface{},
			}

			h := &Support{
				loggerProvider: tt.fields.loggerProvider,
				service:        tt.fields.service,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			response := h.ListPlatforms(ctx)

			if tt.wantErr != nil {
				assert.Error(t, response, tt.wantErr.Error())
			} else {
				assert.NoError(t, response)
			}
		})
	}
}

func TestSupport_ListProductsHandler(t *testing.T) {
	ctx := GetContext()

	type args struct {
		platform string
	}

	testPlatform := "test-platform"
	url := "https://test.com?platform=" + testPlatform

	tests := []struct {
		name    string
		args    args
		fields  fields
		on      func(*fields)
		wantErr error
	}{
		{
			name: "happy path",
			args: args{
				platform: testPlatform,
			},
			on: func(f *fields) {
				f.service.
					On(
						"ListProducts",
						ctx,
						testPlatform,
					).
					Return(&service.ProductsAPI{}, nil).
					Once()
			},
		},
		{
			name: "error invalid platform",
			args: args{
				platform: testPlatform,
			},
			wantErr: service.ErrInvalidPlatform,
			on: func(f *fields) {
				f.service.
					On(
						"ListProducts",
						ctx,
						testPlatform,
					).
					Return(nil, service.ErrInvalidPlatform).
					Once()
			},
		},
		{
			name: "service returned error",
			args: args{
				platform: testPlatform,
			},
			wantErr: errors.New("some error"),
			on: func(f *fields) {
				f.service.
					On(
						"ListProducts",
						ctx,
						testPlatform,
					).
					Return(nil, errors.New("some error")).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider: logger.FromContext,
				service:        &serviceMock.SupportServiceInterface{},
			}

			h := &Support{
				loggerProvider: tt.fields.loggerProvider,
				service:        tt.fields.service,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			ctx.Request = httptest.NewRequest(http.MethodGet, url, nil)
			response := h.ListProducts(ctx)

			if tt.wantErr != nil {
				assert.Error(t, response, tt.wantErr.Error())
			} else {
				assert.NoError(t, response)
			}
		})
	}
}
func TestSupport_ChangeCustomerTier(t *testing.T) {
	ctx := GetContext()

	type args struct {
		customerID string
		req        RequestBody
	}

	tests := []struct {
		name    string
		args    args
		fields  fields
		on      func(*fields)
		wantErr error
	}{
		{
			name: "happy path - one-time support",
			args: args{
				customerID: "test-customer",
				req: RequestBody{
					ServiceType: "one-time",
					PackageType: "oneTimeTicket",
					Email:       "test@test.com",
				},
			},
			on: func(f *fields) {
				f.service.
					On(
						"ApplyOneTimeSupport",
						ctx,
						"test-customer",
						pkg.OneTimeProductType("oneTimeTicket"),
						"test@test.com",
					).
					Return(nil).
					Once()
			},
		},
		{
			name: "happy path - new support tier",
			args: args{
				customerID: "test-customer",
				req: RequestBody{
					ServiceType: "subscription",
					PackageType: "premium",
					Email:       "test@test.com",
				},
			},
			on: func(f *fields) {
				f.service.
					On(
						"ApplyNewSupportTier",
						ctx,
						"test-customer",
						pkg.TierNameType("premium"),
					).
					Return(nil).
					Once()
			},
		},
		{
			name: "error from service",
			args: args{
				customerID: "test-customer",
				req: RequestBody{
					ServiceType: "one-time",
					PackageType: "test-package",
					Email:       "test@test.com",
				},
			},
			wantErr: errors.New("some error"),
			on: func(f *fields) {
				f.service.
					On(
						"ApplyOneTimeSupport",
						ctx,
						"test-customer",
						pkg.OneTimeProductType("test-package"),
						"test@test.com",
					).
					Return(errors.New("some error")).
					Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				loggerProvider: logger.FromContext,
				service:        &serviceMock.SupportServiceInterface{},
			}

			h := &Support{
				loggerProvider: tt.fields.loggerProvider,
				service:        tt.fields.service,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			ctx.Params = gin.Params{{Key: "customerID", Value: tt.args.customerID}}
			body, _ := json.Marshal(tt.args.req)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/customer", bytes.NewBuffer(body))
			response := h.ChangeCustomerTier(ctx)

			if tt.wantErr != nil {
				assert.Error(t, response, tt.wantErr.Error())
			} else {
				assert.NoError(t, response)
			}
		})
	}
}
