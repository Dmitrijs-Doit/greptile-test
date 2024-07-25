package internal

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

type Firestore struct {
	*firestore.Client
}

func NewFirestoreClient(ctx context.Context) (*Firestore, error) {
	client, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("could not initialize firestore client. error %s", err)
	}

	return &Firestore{
		client,
	}, nil
}
