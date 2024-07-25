package dal

import (
	"context"

	"cloud.google.com/go/firestore"
)

type AppContractFirestore interface {
	GetRef(ctx context.Context, id string) *firestore.DocumentRef
}
