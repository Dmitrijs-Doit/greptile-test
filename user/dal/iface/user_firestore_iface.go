//go:generate mockery --output=../mocks --all
package iface

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/user/domain"
)

//go:generate mockery --name IUserFirestoreDAL --output ../mocks
type IUserFirestoreDAL interface {
	GetUserByEmail(ctx context.Context, email string, customerID string) (*domain.User, error)
	ListUsers(ctx context.Context, customerRef *firestore.DocumentRef, limit int) ([]*domain.User, error)
	Get(ctx context.Context, id string) (*common.User, error)
	GetUsersWithRecentEngagement(ctx context.Context) ([]common.User, error)
	GetLastUserEngagementTimeForCustomer(ctx context.Context, customerID string) (*time.Time, error)
	GetRef(ctx context.Context, ID string) *firestore.DocumentRef
	GetCustomerUsersWithNotifications(ctx context.Context, customerID string, isRestore bool) ([]*pkg.User, error)
	ClearUserNotifications(ctx context.Context, user *pkg.User) error
	RestoreUserNotifications(ctx context.Context, user *pkg.User) error
	GetCustomerUsersWithInvoiceNotification(ctx context.Context, customerID, entityID string) ([]*common.User, error)
	GetCustomerUsersByRoles(ctx context.Context, customerID string, roles []common.PresetRole) ([]*common.User, error)
}
