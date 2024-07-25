package service

import (
	"errors"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/support/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/support/domain"
)

func TestSupportService_ListPlatforms(t *testing.T) {
	type fields struct {
		supportDal *mocks.Support
	}

	recorder := httptest.NewRecorder()
	platforms := []domain.Platform{
		{
			Asset:      "test asset",
			HelperText: "test helper text",
			Label:      "test label",
			Order:      1,
			Title:      "test title",
			Value:      "test value",
		},
	}

	tests := []struct {
		name      string
		fields    fields
		wantErr   error
		ctxValues map[string]interface{}
		on        func(*fields)
		response  *PlatformsAPI
	}{
		{
			name: "Happy path",
			response: &PlatformsAPI{
				Platforms: []PlatformAPI{{
					ID:          "test value",
					DisplayName: "test title",
				}},
			},
			on: func(f *fields) {
				f.supportDal.
					On("ListPlatforms", mock.Anything, false).
					Return(platforms, nil).
					Once()
			},
		}, {
			name: "Happy path SaaS customer",
			ctxValues: map[string]interface{}{
				"customerType": pkg.ProductOnlyCustomerType,
			},
			response: &PlatformsAPI{
				Platforms: []PlatformAPI{{
					ID:          "test value",
					DisplayName: "test title",
				}},
			},
			on: func(f *fields) {
				f.supportDal.
					On("ListPlatforms", mock.Anything, true).
					Return(platforms, nil).
					Once()
			},
		}, {
			name:    "ListPlatforms returns error",
			wantErr: errors.New("some error"),
			on: func(f *fields) {
				f.supportDal.
					On("ListPlatforms", mock.Anything, false).
					Return(nil, errors.New("some error")).
					Once()
			},
		},
	}

	for _, tt := range tests {
		ctx, _ := gin.CreateTestContext(recorder)

		if tt.ctxValues != nil {
			for k, v := range tt.ctxValues {
				ctx.Set(k, v)
			}
		}

		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				supportDal: &mocks.Support{},
			}
			s := &SupportService{
				supportDal: tt.fields.supportDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			r, err := s.ListPlatforms(ctx)

			if tt.wantErr != nil {
				assert.Error(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)

				if tt.response != nil {
					assert.Equal(t, tt.response, r)
				}
			}
		})
	}
}

func TestSupportService_ListProducts(t *testing.T) {
	type fields struct {
		supportDal *mocks.Support
	}

	recorder := httptest.NewRecorder()

	awsPlatform := "amazon_web_services"
	awsPlatformWithDash := "amazon-web-services"
	gsuitePlatform := "google_g_suite"
	gsuitePlatformWithDash := "g-suite"

	products := []domain.Product{
		{
			ID:       "aws",
			Name:     "aws",
			Platform: awsPlatformWithDash,
			Summary:  "test summary",
			Version:  89789374857389,
		},
		{
			ID:       "google-g-suite",
			Name:     "google g suite",
			Platform: gsuitePlatformWithDash,
			Summary:  "test summary",
			Version:  89789374857389,
		},
	}

	type args struct {
		platform string
	}

	platforms := []domain.Platform{
		{
			Asset:         awsPlatformWithDash,
			HelperText:    "test helper text",
			Label:         "test label",
			Order:         1,
			Title:         "test title",
			Value:         awsPlatform,
			SaasSupported: true,
		},
		{
			Asset:      gsuitePlatformWithDash,
			HelperText: "test helper text 2",
			Label:      "test label 2",
			Order:      2,
			Title:      "test title 2",
			Value:      gsuitePlatform,
		},
	}

	tests := []struct {
		name      string
		args      args
		ctxValues map[string]interface{}
		fields    fields
		wantErr   error
		on        func(*fields)
		response  *ProductsAPI
	}{
		{
			name: "Success, service returns all products",
			args: args{
				platform: "",
			},
			on: func(f *fields) {
				f.supportDal.
					On("ListProducts", mock.Anything, []string{}).
					Return(products, nil).
					Once()

			},
			response: &ProductsAPI{
				Products: toProductsAPI(products).Products,
			},
		},
		{
			name: "Success, service returns product for requested platform",
			args: args{
				platform: awsPlatform,
			},
			on: func(f *fields) {
				f.supportDal.
					On("ListProducts", mock.Anything, []string{awsPlatformWithDash}).
					Return([]domain.Product{products[0]}, nil).
					Once()
			},
			response: &ProductsAPI{
				Products: toProductsAPI([]domain.Product{products[0]}).Products,
			},
		},
		{
			name: "Success, SAAS customer, service returns all Product Only products",
			args: args{
				platform: "",
			},
			ctxValues: map[string]interface{}{
				"customerType": pkg.ProductOnlyCustomerType,
			},
			on: func(f *fields) {
				f.supportDal.
					On("ListPlatforms", mock.Anything, true).
					Return([]domain.Platform{platforms[0]}, nil).
					Once()
				f.supportDal.
					On("ListProducts", mock.Anything, []string{awsPlatformWithDash}).
					Return([]domain.Product{products[0]}, nil).
					Once()

			},
			response: &ProductsAPI{
				Products: toProductsAPI([]domain.Product{products[0]}).Products,
			},
		},
		{
			name: "Error invalid platform",
			args: args{
				platform: "test_platform1",
			},
			wantErr: ErrInvalidPlatform,
		},
		{
			name: "ListProducts returned error",
			args: args{
				platform: awsPlatform,
			},
			wantErr: errors.New("some error"),
			on: func(f *fields) {
				f.supportDal.
					On("ListProducts", mock.Anything, []string{awsPlatformWithDash}).
					Return(nil, errors.New("some error")).
					Once()

			},
		},
	}

	for _, tt := range tests {
		ctx, _ := gin.CreateTestContext(recorder)

		if tt.ctxValues != nil {
			for k, v := range tt.ctxValues {
				ctx.Set(k, v)
			}
		}

		t.Run(tt.name, func(t *testing.T) {
			tt.fields = fields{
				supportDal: &mocks.Support{},
			}
			s := &SupportService{
				supportDal: tt.fields.supportDal,
			}

			if tt.on != nil {
				tt.on(&tt.fields)
			}

			r, err := s.ListProducts(ctx, tt.args.platform)

			if tt.wantErr != nil {
				assert.Error(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)

				if tt.response != nil {
					assert.Equal(t, tt.response, r)
				}
			}
		})
	}
}

func TestSupportService_ToProductsApi(t *testing.T) {
	products := []domain.Product{
		{
			ID:       "aws",
			Name:     "aws",
			Platform: "amazon-web-services",
			Summary:  "test summary",
			Version:  89789374857389,
		},
		{
			ID:       "google-g-suite",
			Name:     "google g suite",
			Platform: "g-suite",
			Summary:  "test summary",
			Version:  89789374857389,
		},
	}

	expected := ProductsAPI{
		Products: []ProductAPI{
			{
				ID:          "aws",
				DisplayName: "aws",
				Platform:    "amazon_web_services",
			},
			{
				ID:          "google_g_suite",
				DisplayName: "google g suite",
				Platform:    "google_g_suite",
			},
		},
	}

	assert.Equal(t, expected, toProductsAPI(products))
}
