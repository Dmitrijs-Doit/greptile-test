package common

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
)

const (
	MissingPermissionsPrefix = "user is missing required permission:"
)

// User represents a firestore user document
type User struct {
	Customer           UserCustomer             `firestore:"customer"`
	Domain             string                   `firestore:"domain"`
	Entities           []*firestore.DocumentRef `firestore:"entities"`
	Enrichment         interface{}              `firestore:"enrichment"`
	Permissions        []string                 `firestore:"permissions"`
	Notifications      []int64                  `firestore:"userNotifications"`
	Role               interface{}              `firestore:"role"`
	JobFunction        interface{}              `firestore:"jobFunction"`
	Email              string                   `firestore:"email"`
	DisplayName        string                   `firestore:"displayName"`
	FirstName          string                   `firestore:"firstName"`
	LastName           string                   `firestore:"lastName"`
	Roles              []*firestore.DocumentRef `firestore:"roles"`
	AccessKey          string                   `firestore:"accessKey"`
	Organizations      []*firestore.DocumentRef `firestore:"organizations"`
	EmailNotifications []string                 `firestore:"emailNotifications"`
	DailyDigests       []*firestore.DocumentRef `firestore:"dailyDigests"`
	WeeklyDigests      []*firestore.DocumentRef `firestore:"weeklyDigests"`
	TermsOfService     *TermsOfService          `firestore:"termsOfService"`
	LastLogin          time.Time                `firestore:"lastLogin"`
	ID                 string
}

type TermsOfService struct {
	TimeCreated time.Time `firestore:"timeCreated"`
	Type        string    `firestore:"type"`
}

type UserCustomer struct {
	Ref       *firestore.DocumentRef `firestore:"ref"`
	Name      string                 `firestore:"name"`
	LowerName string                 `firestore:"_name"`
}

type Role struct {
	Customer     *firestore.DocumentRef   `firestore:"customer"`
	Name         string                   `firestore:"name"`
	Description  string                   `firestore:"description"`
	Permissions  []*firestore.DocumentRef `firestore:"permissions"`
	InUse        int64                    `firestore:"inUse"`
	TimeCreated  time.Time                `firestore:"timeCreated"`
	TimeModified time.Time                `firestore:"timeModified"`
	Type         string                   `firestore:"type"`
	CustomerType *string                  `firestore:"customerType"`
}

type UserNotification int

const (
	UserNotificationNewInvoices UserNotification = iota + 1
	UserNotificationCostAnomalies
	UserNotificationOverdueInvoices
	UserNotificationCreditsUtilization
	UserNotificationQuotasUtilization
	UserNotificationKnownIssues
	UserNotificationDailyDigest
	UserNotificationMonthlyDigest
	UserNotificationWeeklyDigest
)

func GetUser(ctx context.Context, ref *firestore.DocumentRef) (*User, error) {
	if ref == nil {
		return nil, errors.New("invalid nil user ref")
	}

	docSnap, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}

	var user User
	if err := docSnap.DataTo(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func GetUserByID(ctx context.Context, userID string, fs *firestore.Client) (*User, error) {
	if userID == "" {
		return nil, errors.New("invalid empty user id")
	}

	ref := fs.Collection("users").Doc(userID)

	return GetUser(ctx, ref)
}

// Get permissions array from the user roles
func (u *User) getUserPermissions(ctx context.Context) []string {
	// New role based permissions
	if u.Roles != nil && len(u.Roles) > 0 {
		permissions := make([]string, 0)

		for _, role := range u.Roles {
			roleDocSnap, err := role.Get(ctx)
			if err != nil {
				return nil
			}

			var role Role
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
func hasPermission(userPermissions []string, requiredPermission Permission) bool {
	for _, pID := range userPermissions {
		if pID == string(requiredPermission) {
			return true
		}
	}

	return false
}

func (u *User) HasRequiredPermissions(ctx context.Context, requiredPermissions []Permission) error {
	if len(requiredPermissions) == 0 {
		return nil
	}

	userPermissions := u.getUserPermissions(ctx)

	for _, requiredPermission := range requiredPermissions {
		if !hasPermission(userPermissions, requiredPermission) {
			return errors.New(MissingPermissionsPrefix + string(requiredPermission))
		}
	}

	return nil
}

func (u *User) HasUsersPermission(ctx context.Context) bool {
	permissions := u.getUserPermissions(ctx)
	return hasPermission(permissions, PermissionIAM)
}

func (u *User) HasInvoicesPermission(ctx context.Context) bool {
	permissions := u.getUserPermissions(ctx)
	return hasPermission(permissions, PermissionInvoices)
}

func (u *User) HasLicenseManagePermission(ctx context.Context) bool {
	permissions := u.getUserPermissions(ctx)
	return hasPermission(permissions, PermissionAssetsManager)
}

func (u *User) HasEntitiesPermission(ctx context.Context) bool {
	permissions := u.getUserPermissions(ctx)
	return hasPermission(permissions, PermissionBillingProfiles)
}

func (u *User) HasSandboxAdminPermission(ctx context.Context) bool {
	permissions := u.getUserPermissions(ctx)
	return hasPermission(permissions, PermissionSandboxAdmin)
}

func (u *User) HasSandboxUserPermission(ctx context.Context) bool {
	permissions := u.getUserPermissions(ctx)
	return hasPermission(permissions, PermissionSandboxUser)
}

func (u *User) HasFlexSaveAdminPermission(ctx context.Context) bool {
	permissions := u.getUserPermissions(ctx)
	return hasPermission(permissions, PermissionFlexibleRI)
}

func (u *User) HasManageSettingsPermission(ctx context.Context) bool {
	permissions := u.getUserPermissions(ctx)
	return hasPermission(permissions, PermissionSettings)
}

func (u *User) HasContractsViewerPermission(ctx context.Context) bool {
	permissions := u.getUserPermissions(ctx)
	return hasPermission(permissions, PermissionContractsViewer)
}

func (u *User) HasAnomaliesViewerPermission(ctx context.Context) bool {
	permissions := u.getUserPermissions(ctx)
	return hasPermission(permissions, PermissionAnomaliesViewer)
}

func (u *User) HasPerksViewerPermission(ctx context.Context) bool {
	permissions := u.getUserPermissions(ctx)
	return hasPermission(permissions, PermissionPerksViewer)
}

func (u *User) HasIssuesPermission(ctx context.Context) bool {
	permissions := u.getUserPermissions(ctx)
	return hasPermission(permissions, PermissionIssuesViewer)
}

func (u *User) HasBudgetsPermission(ctx context.Context) bool {
	permissions := u.getUserPermissions(ctx)
	return hasPermission(permissions, PermissionBudgetsManager)
}

func (u *User) HasMetricsPermission(ctx context.Context) bool {
	permissions := u.getUserPermissions(ctx)
	return hasPermission(permissions, PermissionMetricsManager)
}

func (u *User) HasAttributionsPermission(ctx context.Context) bool {
	permissions := u.getUserPermissions(ctx)
	return hasPermission(permissions, PermissionAttributionsManager)
}

func (u *User) HasSupportPermission(ctx context.Context) bool {
	permissions := u.getUserPermissions(ctx)
	return hasPermission(permissions, PermissionSupportRequester)
}

func (u *User) HasCloudAnalyticsPermission(ctx context.Context) bool {
	permissions := u.getUserPermissions(ctx)
	return hasPermission(permissions, PermissionCloudAnalytics)
}

func (u *User) HasCAOwnerRoleAssignerPermission(ctx context.Context) bool {
	permissions := u.getUserPermissions(ctx)
	return hasPermission(permissions, PermissionCAOwnerRoleAssigner)
}

func (u *User) HasLabelsManagerPermission(ctx context.Context) bool {
	permissions := u.getUserPermissions(ctx)
	return hasPermission(permissions, PermissionLabelsManager)
}

func (u *User) NotificationsInvoicesEnabled() bool {
	for _, v := range u.Notifications {
		if v == 1 {
			return true
		}
	}

	return false
}

func (u *User) NotificationsPaymentReminders() bool {
	for _, v := range u.Notifications {
		if v == 3 {
			return true
		}
	}

	return false
}

// ValidateOrganization checks whether the user is allowed to access some org resource.
func (u *User) ValidateOrganization(orgID string) bool {
	// user and resource has no org, user is allowed to access.
	if len(u.Organizations) == 0 && orgID == "" {
		return true
	}
	// user and resource has org, user is allowed to access only if it is a member of the org.
	if len(u.Organizations) > 0 && orgID != "" {
		return u.MemberOfOrganization(orgID)
	}

	// deny access if the user has an org and the resource does not,
	// or if the user doesn't have an org, and the resource does.
	return false
}

// MemberOfOrganization checks whether the user is a member of an organization.
// Supports multiple orgs.
func (u *User) MemberOfOrganization(orgID string) bool {
	if len(u.Organizations) > 0 {
		for _, org := range u.Organizations {
			if org.ID == orgID {
				return true
			}
		}
	}

	return false
}
