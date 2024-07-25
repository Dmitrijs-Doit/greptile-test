package connection

import (
	"context"
	"errors"

	"cloud.google.com/go/pubsub"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

var (
	ErrPubsubInitialization = errors.New("pubsub initialization error")
)

type PubsubClient struct {
	pubsub *pubsub.Client
}

func NewPubsubClient(ctx context.Context, log *logger.Logging) (*PubsubClient, error) {
	logger := log.Logger(ctx)

	ps, err := pubsub.NewClient(ctx, common.ProjectID)
	if err != nil {
		logger.Errorf("%s: %s", ErrPubsubInitialization, err)
		return nil, ErrPubsubInitialization
	}

	return &PubsubClient{
		ps,
	}, nil
}
