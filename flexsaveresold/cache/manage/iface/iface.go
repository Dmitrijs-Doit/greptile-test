package iface

import (
	"context"

	fspkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
)

type EmailInterface interface {
	SendWelcomeEmail(
		ctx context.Context,
		params *types.WelcomeEmailParams,
		usersWithPermissions []*common.User,
		accountManagers []*common.AccountManager,
	) error
}

type FlexsaveService interface {
	RunCacheForSingleCustomer(ctx context.Context, customerID string) (*fspkg.FlexsaveSavings, error)
}
