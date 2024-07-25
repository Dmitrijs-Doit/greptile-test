//go:generate mockery --output=./mocks --all

package dal

import (
	"bytes"
	"context"
)

type PlesBigQueryDalIface interface {
	UpdatePlesAccounts(ctx context.Context, accounts *bytes.Buffer, monthPartition string) error
}
