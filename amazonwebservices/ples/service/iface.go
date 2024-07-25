//go:generate mockery --output=./mocks --all

package service

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/ples/domain"
)

type PLESIface interface {
	UpdatePLESAccounts(ctx context.Context, accounts []domain.PLESAccount, forceUpdate bool) []error
}
