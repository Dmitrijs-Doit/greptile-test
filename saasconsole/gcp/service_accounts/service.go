package service_accounts

import (
	"context"
	"time"

	fsdal "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/service"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/utils"
	"github.com/doitintl/retry"
)

type GCPSaaSConsoleServiceAccountsService struct {
	loggerProvider logger.Provider
	saasConsoleDAL fsdal.SaaSConsoleOnboard
	saPool         *service.ServiceAccountsPoolService
	serviceAccount *service.ServiceAccountService
	envStatus      *service.EnvStatusService
	project        *service.ProjectService
}

func NewGCPSaaSConsoleServiceAccountsService(log logger.Provider, conn *connection.Connection) *GCPSaaSConsoleServiceAccountsService {
	ctx := context.Background()

	return &GCPSaaSConsoleServiceAccountsService{
		log,
		fsdal.NewSaaSConsoleOnboardDALWithClient(conn.Firestore(ctx)),
		service.NewServiceAccountsPoolService(log, conn, conn.CloudTaskClient),
		service.NewServiceAccountService(log, conn),
		service.NewEnvStatusService(log, conn),
		service.NewProjectService(log, conn),
	}
}

func (s *GCPSaaSConsoleServiceAccountsService) CreateServiceAccounts(ctx context.Context) error {
	for i := 0; i < utils.ServiceAccountsInProjectThreshold; i++ {
		err := retry.BackOffDelay(
			func() error {
				serviceAccount, err := s.serviceAccount.CreateServiceAccount(ctx)
				if err != nil {
					return err
				}

				err = s.saPool.AddServiceAccount(ctx, serviceAccount)
				if err != nil {
					return err
				}

				if err := s.serviceAccount.AddServiceAccountToGroup(ctx, serviceAccount.Email); err != nil {
					return err
				}

				err = s.project.AddServiceAccountToProjects(ctx, serviceAccount.ProjectId)
				if err != nil {
					return err
				}

				return nil
			},
			5,
			time.Second*30,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *GCPSaaSConsoleServiceAccountsService) InitEnvironment(ctx context.Context) error {
	logger := a.loggerProvider(ctx)

	envStatus, err := a.envStatus.GetEnvStatus(ctx)
	if err != nil {
		return err
	}

	if envStatus.Initiated {
		logger.Infof("Environment already initialized")
		return nil
	}

	folder := utils.GetFolderID()

	_, err = a.project.CreateNewProject(ctx, folder)
	if err != nil {
		return err
	}

	err = a.CreateServiceAccounts(ctx)
	if err != nil {
		return err
	}

	envStatus.Initiated = true
	err = a.envStatus.SetEnvStatus(ctx, envStatus)

	if err != nil {
		return err
	}

	return nil
}

func (s *GCPSaaSConsoleServiceAccountsService) GetNextFreeServiceAccount(ctx context.Context, customerID string) (string, error) {
	return s.saPool.GetNextFreeServiceAccount(ctx, customerID)
}

func (s *GCPSaaSConsoleServiceAccountsService) MarkServiceAccountOnBoardSuccessful(ctx context.Context, customerID, saEmail, billingAccountID string) error {
	return s.saPool.MarkServiceAccountOnBoardSuccessful(ctx, customerID, saEmail, billingAccountID)
}
