package dal

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/credit"

	"cloud.google.com/go/firestore"
)

type Credits interface {
	GetCredits(ctx context.Context) (map[*firestore.DocumentRef]credit.BaseCredit, error)
}
