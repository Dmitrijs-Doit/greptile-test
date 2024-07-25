package servicecatalog

import (
	"context"
	"fmt"
	"testing"

	logger "github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
)

func TestServiceCatalog_CreatePortfolioShare(t *testing.T) {
	type fields struct {
		client MockClient
		cache  MockCache
	}

	type args struct {
		ctx       context.Context
		accountID string
		region    string
	}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		want    string
		wantErr error
	}{
		{
			name: "happpy path",
			args: args{
				ctx:       context.Background(),
				accountID: "123456789012",
				region:    "us-east-1",
			},
			on: func(f *fields) {
				// inits the client for the region
				f.client.On("InitSession", "us-east-1").Return(nil)
				// tries to get the portfolio id from the cache, misses
				f.cache.On("Get", mock.Anything, CacheKey{
					Name:      portfolioNamePrefix,
					AccountID: accountIDFromARN(accountRoleArnDev),
					Region:    "us-east-1",
				}).Return("", ErrCacheMiss)
				// gets the portfolio id from the client
				f.client.On("GetPortfoliosByNamePrefix", mock.Anything, mock.Anything).Return([]Portfolio{
					{
						ID:          "DoiT_id",
						DisplayName: "DoiT-Onboarding",
					},
					{
						ID:          "DoiT1_id",
						DisplayName: "DoiT-Onboarding-1",
					},
				}, nil)
				f.client.On("IsShareQuotaReached", "DoiT_id").Return(true, nil)
				f.client.On("IsShareQuotaReached", "DoiT1_id").Return(false, nil)
				// caches the portfolio id
				f.cache.On("Set", mock.Anything, CacheKey{
					Name:      portfolioNamePrefix,
					AccountID: accountIDFromARN(accountRoleArnDev),
					Region:    "us-east-1",
				}, "DoiT1_id").Return(nil)
				// creates the portfolio share
				f.client.On("CreatePortfolioShare", "DoiT1_id", "123456789012").Return(nil)
			},
			wantErr: nil,
			want:    "DoiT1_id",
		},
		{
			name: "cache hit",
			args: args{
				ctx:       context.Background(),
				accountID: "123456789012",
				region:    "us-east-1",
			},
			on: func(f *fields) {
				f.client.On("InitSession", "us-east-1").Return(nil)
				f.cache.On("Get", mock.Anything, CacheKey{
					Name:      portfolioNamePrefix,
					AccountID: accountIDFromARN(accountRoleArnDev),
					Region:    "us-east-1",
				}).Return("portfolio-id", nil)
				f.client.On("CreatePortfolioShare", "portfolio-id", "123456789012").Return(nil)
			},
			wantErr: nil,
			want:    "portfolio-id",
		},
		{
			name: "invalid cache entry is deleted",
			args: args{
				ctx:       context.Background(),
				accountID: "123456789012",
				region:    "us-east-1",
			},
			on: func(f *fields) {
				f.client.On("InitSession", "us-east-1").Return(nil)
				f.cache.On("Get", mock.Anything, CacheKey{
					Name:      portfolioNamePrefix,
					AccountID: accountIDFromARN(accountRoleArnDev),
					Region:    "us-east-1",
				}).Return("portfolio-id", nil).Once()
				f.client.On("CreatePortfolioShare", "portfolio-id", "123456789012").Return(fmt.Errorf("error"))
				f.cache.On("Del", mock.Anything, CacheKey{
					Name:      portfolioNamePrefix,
					AccountID: accountIDFromARN(accountRoleArnDev),
					Region:    "us-east-1",
				}).Return(nil)
				f.cache.On("Get", mock.Anything, CacheKey{
					Name:      portfolioNamePrefix,
					AccountID: accountIDFromARN(accountRoleArnDev),
					Region:    "us-east-1",
				}).Return("", ErrCacheMiss)
				f.client.On("GetPortfoliosByNamePrefix", mock.Anything, mock.Anything).Return([]Portfolio{
					{
						ID:          "new-portfolio-id",
						DisplayName: "DoiT-Onboarding-1",
					},
				}, nil)
				f.client.On("IsShareQuotaReached", "new-portfolio-id").Return(false, nil)
				f.cache.On("Set", mock.Anything, CacheKey{
					Name:      portfolioNamePrefix,
					AccountID: accountIDFromARN(accountRoleArnDev),
					Region:    "us-east-1",
				}, "new-portfolio-id").Return(nil)
				f.client.On("CreatePortfolioShare", "new-portfolio-id", "123456789012").Return(nil)
			},
			wantErr: nil,
			want:    "new-portfolio-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loggerProvider := func(ctx context.Context) logger.ILogger {
				return &logger.Logger{}
			}

			fields := fields{
				client: *NewMockClient(t),
				cache:  *NewMockCache(t),
			}
			if tt.on != nil {
				tt.on(&fields)
			}

			sc := ServiceCatalog{
				accountRoleArn: accountRoleArnDev,
				loggerProvider: loggerProvider,
				cache:          &fields.cache,
			}
			portfolioID, err := sc.CreatePortfolioShare(tt.args.ctx, &fields.client, tt.args.accountID, tt.args.region)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}

			if tt.want != "" {
				assert.Equal(t, tt.want, portfolioID)
			}
		})
	}
}
