//go:generate mockery --output=../mocks --all
package iface

import (
	"context"
)

type EntitiesIface interface {
	SyncEntitiesInvoiceAttributions(ctx context.Context, forceUpdate bool) error
}
