//go:generate mockery --output=./mocks --all

package dal

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/algolia"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type AlgoliaDAL interface {
	GetConfigFromFirestore(ctx context.Context) (*algolia.Config, error)
}

//go:generate mockery --output=./mocks --all

// UserDAL The following interface contains wrappers around user permission methods used for giving the ability to mock results in unit tests
type UserDAL interface {
	GetUser(ctx context.Context, userID string) (*common.User, error)
	HasUsersPermission(ctx context.Context, user *common.User) bool
	HasInvoicesPermission(ctx context.Context, user *common.User) bool
	HasLicenseManagePermission(ctx context.Context, user *common.User) bool
	HasEntitiesPermission(ctx context.Context, user *common.User) bool
	HasCloudAnalyticsPermission(ctx context.Context, user *common.User) bool
	HasMetricsPermission(ctx context.Context, user *common.User) bool
	HasAttributionsPermission(ctx context.Context, user *common.User) bool
	HasBudgetsPermission(ctx context.Context, user *common.User) bool
}
