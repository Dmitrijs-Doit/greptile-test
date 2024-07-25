package connection

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

var (
	ErrFirestoreInitialization         = errors.New("firestore initialization error")
	ErrDemoModeFirestoreInitialization = errors.New("demo mode firestore initialization error")
)

type FirestoreClient struct {
	// firestore clients
	fs *firestore.Client
}

func NewFirestore(ctx context.Context, log *logger.Logging) (*FirestoreClient, error) {
	logger := log.Logger(ctx)

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		logger.Errorf("%s: %s", ErrFirestoreInitialization, err)
		return nil, ErrFirestoreInitialization
	}

	return &FirestoreClient{
		fs,
	}, nil
}
