package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

//go:generate mockery --output=../mocks --all
type IUserPermissionsChecker interface {
	HasRequiredPermissions(ctx context.Context, user *common.User, requiredPermissions []common.Permission) error
	HasUsersPermission(ctx context.Context, user *common.User) bool
	HasInvoicesPermission(ctx context.Context, user *common.User) bool
	HasLicenseManagePermission(ctx context.Context, user *common.User) bool
	HasEntitiesPermission(ctx context.Context, user *common.User) bool
	HasSandboxAdminPermission(ctx context.Context, user *common.User) bool
	HasSandboxUserPermission(ctx context.Context, user *common.User) bool
	HasFlexSaveAdminPermission(ctx context.Context, user *common.User) bool
	HasManageSettingsPermission(ctx context.Context, user *common.User) bool
	HasContractsViewerPermission(ctx context.Context, user *common.User) bool
	HasAnomaliesViewerPermission(ctx context.Context, user *common.User) bool
	HasPerksViewerPermission(ctx context.Context, user *common.User) bool
	HasIssuesPermission(ctx context.Context, user *common.User) bool
	HasBudgetsPermission(ctx context.Context, user *common.User) bool
	HasMetricsPermission(ctx context.Context, user *common.User) bool
	HasAttributionsPermission(ctx context.Context, user *common.User) bool
	HasSupportPermission(ctx context.Context, user *common.User) bool
	HasCloudAnalyticsPermission(ctx context.Context, user *common.User) bool
	HasCAOwnerRoleAssignerPermission(ctx context.Context, user *common.User) bool
	HasLabelsManagerPermission(ctx context.Context, user *common.User) bool
}
