package dal

import (
	"context"

	"cloud.google.com/go/firestore"

	rv "github.com/doitintl/hello/scheduled-tasks/rowsvalidator"
)

type RowsValidatorMetadata interface {
	GetDocRef(ctx context.Context, billingAccountID string) *firestore.DocumentRef
	DeleteDocRef(ctx context.Context, billingAccountID string) error
	DeleteDocsRef(ctx context.Context) error
	GetRowsValidatorMetadata(ctx context.Context, accountID string) (*rv.RowsValidatorMetadata, error)
	SetRowsValidatorMetadata(ctx context.Context, accountID string, config *rv.RowsValidatorMetadata) error
}
