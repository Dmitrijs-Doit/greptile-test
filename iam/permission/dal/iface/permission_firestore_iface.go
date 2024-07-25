//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/iam/permission/domain"
)

type IPermissionFirestoreDAL interface {
	Get(ctx context.Context, permissionID string) (*domain.Permission, error)
}
