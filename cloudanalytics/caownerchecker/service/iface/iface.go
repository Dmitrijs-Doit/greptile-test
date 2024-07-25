package iface

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
)

//go:generate mockery --name CheckCAOwnerInterface --output=../mocks
type CheckCAOwnerInterface interface {
	CheckCAOwner(ctx context.Context, es doitemployees.ServiceInterface, userID string, email string) (bool, error)
}
