package service

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/dal"
	ds "github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/dataStructures"
)

type EnvStatusService struct {
	loggerProvider logger.Provider
	*connection.Connection
	dal *dal.ServiceAccountsFirestore
}

func NewEnvStatusService(log logger.Provider, conn *connection.Connection) *EnvStatusService {
	return &EnvStatusService{
		log,
		conn,
		dal.NewServiceAccountsFirestoreWithClient(log, conn),
	}
}

func (p *EnvStatusService) GetEnvStatus(ctx context.Context) (*ds.EnvStatus, error) {
	ref := p.dal.GetEnvStatusRef(ctx)

	result, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}

	var envStatus *ds.EnvStatus
	if err = result.DataTo(&envStatus); err != nil {
		return nil, err
	}

	return envStatus, nil
}

func (p *EnvStatusService) SetEnvStatus(ctx context.Context, e *ds.EnvStatus) error {
	return p.dal.SetEnvStatus(ctx, e)
}
