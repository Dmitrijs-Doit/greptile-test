//go:generate mockery --output=../mocks --all
package iface

import (
	"cloud.google.com/go/firestore"
	"context"
	"github.com/doitintl/firestore/pkg"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/user/domain"
)

type IUserService interface {
	GetUserByEmail(ctx context.Context, email string, customerID string) (*domain.User, error)
	Get(ctx context.Context, id string) (*common.User, error)
	GetRef(ctx context.Context, ID string) *firestore.DocumentRef
	GetCustomerUsersWithNotifications(ctx context.Context, customerID string, isRestore bool) ([]*pkg.User, error)
	ClearUserNotifications(ctx context.Context, user *pkg.User) error
	RestoreUserNotifications(ctx context.Context, user *pkg.User) error
}
