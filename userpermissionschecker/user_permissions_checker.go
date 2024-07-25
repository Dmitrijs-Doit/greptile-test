package service

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

type UserPermissionsChecker struct {
}

// Get permissions array from the user roles
func getUserPermissions(ctx context.Context, u *common.User) []string {
	// New role based permissions
	if u.Roles != nil && len(u.Roles) > 0 {
		permissions := make([]string, 0)

		for _, role := range u.Roles {
			roleDocSnap, err := role.Get(ctx)
			if err != nil {
				return nil
			}

			var role common.Role
			if err := roleDocSnap.DataTo(&role); err != nil {
				return nil
			}

			for _, permission := range role.Permissions {
				permissions = append(permissions, permission.ID)
			}
		}

		return permissions
	}

	// Legacy permissions
	return u.Permissions
}

// Check if the array of permission contains the requested permission
func hasPermission(userPermissions []string, requiredPermission common.Permission) bool {
	for _, pID := range userPermissions {
		if pID == string(requiredPermission) {
			return true
		}
	}

	return false
}

func NewUserPermissionsChecker() *UserPermissionsChecker {
	return &UserPermissionsChecker{}
}

func (s *UserPermissionsChecker) HasRequiredPermissions(ctx context.Context, user *common.User, requiredPermissions []common.Permission) error {
	return user.HasRequiredPermissions(ctx, requiredPermissions)
}

func (s *UserPermissionsChecker) HasUsersPermission(ctx context.Context, user *common.User) bool {
	permissions := getUserPermissions(ctx, user)
	return hasPermission(permissions, common.PermissionIAM)
}

func (s *UserPermissionsChecker) HasInvoicesPermission(ctx context.Context, user *common.User) bool {
	permissions := getUserPermissions(ctx, user)
	return hasPermission(permissions, common.PermissionInvoices)
}

func (s *UserPermissionsChecker) HasLicenseManagePermission(ctx context.Context, user *common.User) bool {
	permissions := getUserPermissions(ctx, user)
	return hasPermission(permissions, common.PermissionAssetsManager)
}

func (s *UserPermissionsChecker) HasEntitiesPermission(ctx context.Context, user *common.User) bool {
	permissions := getUserPermissions(ctx, user)
	return hasPermission(permissions, common.PermissionBillingProfiles)
}

func (s *UserPermissionsChecker) HasSandboxAdminPermission(ctx context.Context, user *common.User) bool {
	permissions := getUserPermissions(ctx, user)
	return hasPermission(permissions, common.PermissionSandboxAdmin)
}

func (s *UserPermissionsChecker) HasSandboxUserPermission(ctx context.Context, user *common.User) bool {
	permissions := getUserPermissions(ctx, user)
	return hasPermission(permissions, common.PermissionFlexibleRI)
}

func (s *UserPermissionsChecker) HasFlexSaveAdminPermission(ctx context.Context, user *common.User) bool {
	permissions := getUserPermissions(ctx, user)
	return hasPermission(permissions, common.PermissionFlexibleRI)
}

func (s *UserPermissionsChecker) HasManageSettingsPermission(ctx context.Context, user *common.User) bool {
	permissions := getUserPermissions(ctx, user)
	return hasPermission(permissions, common.PermissionSettings)
}

func (s *UserPermissionsChecker) HasContractsViewerPermission(ctx context.Context, user *common.User) bool {
	permissions := getUserPermissions(ctx, user)
	return hasPermission(permissions, common.PermissionContractsViewer)
}

func (s *UserPermissionsChecker) HasAnomaliesViewerPermission(ctx context.Context, user *common.User) bool {
	permissions := getUserPermissions(ctx, user)
	return hasPermission(permissions, common.PermissionAnomaliesViewer)
}

func (s *UserPermissionsChecker) HasPerksViewerPermission(ctx context.Context, user *common.User) bool {
	permissions := getUserPermissions(ctx, user)
	return hasPermission(permissions, common.PermissionPerksViewer)
}

func (s *UserPermissionsChecker) HasIssuesPermission(ctx context.Context, user *common.User) bool {
	permissions := getUserPermissions(ctx, user)
	return hasPermission(permissions, common.PermissionIssuesViewer)
}

func (s *UserPermissionsChecker) HasBudgetsPermission(ctx context.Context, user *common.User) bool {
	permissions := getUserPermissions(ctx, user)
	return hasPermission(permissions, common.PermissionBudgetsManager)
}

func (s *UserPermissionsChecker) HasMetricsPermission(ctx context.Context, user *common.User) bool {
	permissions := getUserPermissions(ctx, user)
	return hasPermission(permissions, common.PermissionMetricsManager)
}

func (s *UserPermissionsChecker) HasAttributionsPermission(ctx context.Context, user *common.User) bool {
	permissions := getUserPermissions(ctx, user)
	return hasPermission(permissions, common.PermissionAttributionsManager)
}

func (s *UserPermissionsChecker) HasSupportPermission(ctx context.Context, user *common.User) bool {
	permissions := getUserPermissions(ctx, user)
	return hasPermission(permissions, common.PermissionSupportRequester)
}

func (s *UserPermissionsChecker) HasCloudAnalyticsPermission(ctx context.Context, user *common.User) bool {
	permissions := getUserPermissions(ctx, user)
	return hasPermission(permissions, common.PermissionCloudAnalytics)
}

func (s *UserPermissionsChecker) HasCAOwnerRoleAssignerPermission(ctx context.Context, user *common.User) bool {
	permissions := getUserPermissions(ctx, user)
	return hasPermission(permissions, common.PermissionCAOwnerRoleAssigner)
}

func (s *UserPermissionsChecker) HasLabelsManagerPermission(ctx context.Context, user *common.User) bool {
	permissions := getUserPermissions(ctx, user)
	return hasPermission(permissions, common.PermissionLabelsManager)
}
