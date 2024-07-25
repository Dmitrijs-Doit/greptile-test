package servicecatalog

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"

	logger "github.com/doitintl/hello/scheduled-tasks/logger"
)

func TestServiceCatalog_syncState(t *testing.T) {
	type fields struct {
		cache   MockCache
		client  MockClient
		tracker MockShareTracker
	}

	type args struct {
		ctx            context.Context
		region         string
		requiredShares map[string]bool
	}

	tests := []struct {
		name    string
		on      func(*fields)
		args    args
		wantErr error
	}{
		{
			name: "create share if missing",
			args: args{
				ctx:    context.Background(),
				region: "us-east-1",
				requiredShares: map[string]bool{
					"123456789012": true,
					"234567890123": true,
				},
			},
			on: func(f *fields) {
				// gets the current state of shares
				f.client.On("GetAllSharesByNamePrefix", portfolioNamePrefix).Return(map[string]string{
					"234567890123": "portfolio-id",
				}, nil)

				// share is created only for the missing account
				f.cache.On("Get", mock.Anything, CacheKey{
					Name:      portfolioNamePrefix,
					AccountID: accountIDFromARN(accountRoleArnDev),
					Region:    "us-east-1",
				}).Return("portfolio-id", nil)
				f.client.On("CreatePortfolioShare", "portfolio-id", "123456789012").Return(nil)
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loggerProvider := func(ctx context.Context) logger.ILogger {
				return &logger.Logger{}
			}

			fields := fields{
				client:  *NewMockClient(t),
				cache:   *NewMockCache(t),
				tracker: *NewMockShareTracker(t),
			}
			if tt.on != nil {
				tt.on(&fields)
			}

			sc := ServiceCatalog{
				accountRoleArn: accountRoleArnDev,
				loggerProvider: loggerProvider,
				cache:          &fields.cache,
				tracker:        &fields.tracker,
			}
			err := sc.syncState(tt.args.ctx, &fields.client, tt.args.region, tt.args.requiredShares)

			if tt.wantErr != nil {
				assert.EqualError(t, err, tt.wantErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
