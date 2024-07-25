package connection

import (
	"context"
	"errors"

	kms "cloud.google.com/go/kms/apiv1"

	"github.com/doitintl/hello/scheduled-tasks/logger"
)

var (
	ErrKeyManagementInitialization = errors.New("key management initialization error")
)

type KeyManagementClient struct {
	kms *kms.KeyManagementClient
}

func NewKeyManagement(ctx context.Context, log *logger.Logging) (*KeyManagementClient, error) {
	logger := log.Logger(ctx)

	kms, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		logger.Errorf("%s: %s", ErrKeyManagementInitialization, err)
		return nil, ErrKeyManagementInitialization
	}

	return &KeyManagementClient{
		kms,
	}, nil
}
