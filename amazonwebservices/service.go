package amazonwebservices

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/zerobounce"
)

//go:generate mockery --name IAWSService --output ./mocks
type IAWSService interface {
	InviteAccount(ctx context.Context, customerID, entityID, email string, body *InviteAccountBody) (int, error)
	CreateAccount(ctx context.Context, customerID, entityID, email string, body *CreateAccountBody) (string, error)
	UpdateAccounts(ctx context.Context) error
	UpdateHandshakes(ctx context.Context) error
}

type AWSService struct {
	loggerProvider      logger.Provider
	conn                *connection.Connection
	zb                  *zerobounce.Service
	flexsaveAPI         flexapi.FlexAPI
	doitEmployeeService doitemployees.ServiceInterface
}

func NewAWSService(loggerProvider logger.Provider, conn *connection.Connection) (*AWSService, error) {
	zb := zerobounce.New()

	flexAPIService, err := flexapi.NewFlexAPIService()
	if err != nil {
		return nil, err
	}

	doitEmployeesService := doitemployees.NewService(conn)

	return &AWSService{
		loggerProvider,
		conn,
		zb,
		flexAPIService,
		doitEmployeesService,
	}, nil
}
