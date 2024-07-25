package rippling

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/rippling/dal"
	"github.com/doitintl/hello/scheduled-tasks/rippling/iface"
	"github.com/doitintl/hello/scheduled-tasks/rippling/utils"
)

// RipplingService - integrates data from rippling api into the CMP
type RipplingService struct {
	loggerProvider   logger.Provider
	ripplingDal      dal.IRipplingDAL
	accountManagers  dal.IAccountManagers
	doitEmployeesDal doitemployees.ServiceInterface
}

func NewRipplingService(log logger.Provider, conn *connection.Connection) (iface.IRippling, error) {
	ctx := context.Background()

	ripplingDal, err := dal.NewRipplingDAL(ctx)
	if err != nil {
		return nil, err
	}

	accountManagers, err := dal.NewAccountManagers(ctx, log)
	if err != nil {
		return nil, err
	}

	doitEmployeesDal := doitemployees.NewService(conn)

	return &RipplingService{
		log,
		ripplingDal,
		accountManagers,
		doitEmployeesDal,
	}, nil
}

func (s *RipplingService) getLogger(ctx context.Context, flow string) logger.ILogger {
	return utils.GetRipplingLogger(ctx, s.loggerProvider, flow)
}
