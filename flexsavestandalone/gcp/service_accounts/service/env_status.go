package service

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/service_accounts/dal"
	dataStructures "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/service_accounts/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type EnvStatusService struct {
	loggerProvider logger.Provider
	*connection.Connection
	dal *dal.OnBoardingFirestore
}

func NewEnvStatusService(log logger.Provider, conn *connection.Connection) *EnvStatusService {
	return &EnvStatusService{
		log,
		conn,
		dal.NewOnBoardingFirestoreWithClient(log, conn),
	}
}

func (p *EnvStatusService) GetEnvStatus(ctx context.Context) (*dataStructures.EnvStatus, error) {
	ref := p.dal.GetEnvStatusRef(ctx)

	result, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}

	var envStatus *dataStructures.EnvStatus
	if err = result.DataTo(&envStatus); err != nil {
		return nil, err
	}

	return envStatus, nil
}

func (p *EnvStatusService) SetEnvStatus(ctx context.Context, e *dataStructures.EnvStatus) error {
	return p.dal.SetEnvStatus(ctx, e)
}
