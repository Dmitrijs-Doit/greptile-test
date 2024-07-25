package algoliaservice

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/algolia/algoliasearch-client-go/v3/algolia/search"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/doitintl/hello/scheduled-tasks/algolia"
	"github.com/doitintl/hello/scheduled-tasks/algolia/dal/mocks"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

func TestAlgoliaService_GenerateSecuredAPIKey(t *testing.T) {
	ctx := context.Background()

	type fields struct {
		loggerProvider *loggerMocks.ILogger
		userDAL        *mocks.UserDAL
		customerDal    *customerMocks.Customers
		algoliaDAL     *mocks.AlgoliaDAL
		config         *algolia.Config
	}

	tests := []struct {
		name    string
		on      func(*fields)
		wantErr bool
		assert  func(t *testing.T, f *fields, config *algolia.Config)
	}{
		{
			name:    "user has only default alerts permission",
			wantErr: false,
			on: func(f *fields) {
				f.customerDal.On("GetCustomer", ctx, mock.Anything).Return(&common.Customer{EarlyAccessFeatures: []string{"Algolia Search"}}, nil)
				f.userDAL.On("GetUser", ctx, mock.Anything).Return(&common.User{ID: "abc123"}, nil)

				f.userDAL.On("HasUsersPermission", ctx, mock.Anything).Return(false)
				f.userDAL.On("HasInvoicesPermission", ctx, mock.Anything).Return(false)
				f.userDAL.On("HasLicenseManagePermission", ctx, mock.Anything).Return(false)
				f.userDAL.On("HasEntitiesPermission", ctx, mock.Anything).Return(false)
				f.userDAL.On("HasCloudAnalyticsPermission", ctx, mock.Anything).Return(false)
				f.userDAL.On("HasMetricsPermission", ctx, mock.Anything).Return(false)
				f.userDAL.On("HasAttributionsPermission", ctx, mock.Anything).Return(false)
				f.userDAL.On("HasBudgetsPermission", ctx, mock.Anything).Return(false)
			},
			assert: func(t *testing.T, f *fields, config *algolia.Config) {
				decoded, err := base64.StdEncoding.DecodeString(config.SearchKey)
				if err != nil {
					assert.Fail(t, "test failed, could not create secured API key")
				}
				assert.NotNil(t, decoded)
				decodedString := string(decoded)
				assert.Contains(t, decodedString, "test-user-id")
				assert.Contains(t, decodedString, "alerts")
			},
		},
		{
			name:    "key restricted from invoices and assets indices",
			wantErr: false,
			on: func(f *fields) {
				f.customerDal.On("GetCustomer", ctx, mock.Anything).Return(&common.Customer{EarlyAccessFeatures: []string{"Algolia Search"}}, nil)
				f.userDAL.On("GetUser", ctx, mock.Anything).Return(&common.User{ID: "abc123"}, nil)

				f.userDAL.On("HasEntitiesPermission", ctx, mock.Anything).Return(true)
				f.userDAL.On("HasUsersPermission", ctx, mock.Anything).Return(true)

				f.userDAL.On("HasInvoicesPermission", ctx, mock.Anything).Return(false)
				f.userDAL.On("HasLicenseManagePermission", ctx, mock.Anything).Return(false)
				f.userDAL.On("HasCloudAnalyticsPermission", ctx, mock.Anything).Return(false)
				f.userDAL.On("HasMetricsPermission", ctx, mock.Anything).Return(false)
				f.userDAL.On("HasAttributionsPermission", ctx, mock.Anything).Return(false)
				f.userDAL.On("HasBudgetsPermission", ctx, mock.Anything).Return(false)
			},
			assert: func(t *testing.T, f *fields, config *algolia.Config) {
				decoded, err := base64.StdEncoding.DecodeString(config.SearchKey)
				if err != nil {
					assert.Fail(t, "test failed, could not create secured API key")
				}
				assert.NotNil(t, decoded)
				decodedString := string(decoded)
				assert.Contains(t, decodedString, "test-user-id")
				assert.Contains(t, decodedString, "entities")
				assert.Contains(t, decodedString, "users")

				assert.NotContains(t, decodedString, "invoices")
				assert.NotContains(t, decodedString, "assets")
			},
		},
		{
			name:    "key has valid expiration date",
			wantErr: false,
			on: func(f *fields) {
				f.customerDal.On("GetCustomer", ctx, mock.Anything).Return(&common.Customer{EarlyAccessFeatures: []string{"Algolia Search"}}, nil)
				f.userDAL.On("GetUser", ctx, mock.Anything).Return(&common.User{ID: "abc123"}, nil)

				f.userDAL.On("HasEntitiesPermission", ctx, mock.Anything).Return(true)

				f.userDAL.On("HasUsersPermission", ctx, mock.Anything).Return(false)
				f.userDAL.On("HasInvoicesPermission", ctx, mock.Anything).Return(false)
				f.userDAL.On("HasLicenseManagePermission", ctx, mock.Anything).Return(false)
				f.userDAL.On("HasCloudAnalyticsPermission", ctx, mock.Anything).Return(false)
				f.userDAL.On("HasMetricsPermission", ctx, mock.Anything).Return(false)
				f.userDAL.On("HasAttributionsPermission", ctx, mock.Anything).Return(false)
				f.userDAL.On("HasBudgetsPermission", ctx, mock.Anything).Return(false)
			},
			assert: func(t *testing.T, f *fields, config *algolia.Config) {

				client := search.NewClient("", "")
				validUntil, err := client.GetSecuredAPIKeyRemainingValidity(config.SearchKey)
				if err != nil {
					assert.Fail(t, "failed to get remaining validity of key")
				}
				assert.Equal(t, int(validUntil.Hours()), 23)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := fields{
				&loggerMocks.ILogger{},
				&mocks.UserDAL{},
				&customerMocks.Customers{},
				&mocks.AlgoliaDAL{},
				&algolia.Config{SearchKey: "test"},
			}

			service := &Service{
				loggerProvider: fields.loggerProvider,
				userDAL:        fields.userDAL,
				customerDAL:    fields.customerDal,
				algoliaDAL:     fields.algoliaDAL,
				Config:         fields.config,
			}

			if tt.on != nil {
				tt.on(&fields)
			}

			key, err := service.GetAPIKey(ctx, "test-customer-id", "test-user-id")
			if err != nil && !tt.wantErr {
				t.Errorf("Import() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.assert != nil {
				tt.assert(t, &fields, key)
			}
		})
	}
}
