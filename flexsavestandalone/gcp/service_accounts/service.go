package service_accounts

import (
	"context"

	fsdal "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/service_accounts/service"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/service_accounts/utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type GcpStandaloneServiceAccountsService struct {
	loggerProvider        logger.Provider
	flexsaveStandaloneDAL fsdal.FlexsaveStandalone
	saPool                *service.ServiceAccountsPoolService
	serviceAccount        *service.ServiceAccountService
	envStatus             *service.EnvStatusService
	project               *service.ProjectService
}

func NewGcpStandaloneServiceAccountsService(log logger.Provider, conn *connection.Connection) *GcpStandaloneServiceAccountsService {
	ctx := context.Background()

	return &GcpStandaloneServiceAccountsService{
		log,
		fsdal.NewFlexsaveStandaloneDALWithClient(conn.Firestore(ctx)),
		service.NewServiceAccountsPoolService(log, conn, conn.CloudTaskClient),
		service.NewServiceAccountService(log, conn),
		service.NewEnvStatusService(log, conn),
		service.NewProjectService(log, conn),
	}
}

func (s *GcpStandaloneServiceAccountsService) CreateServiceAccounts(ctx context.Context) error {
	for i := 0; i < utils.ServiceAccountsInProjectThreshold; i++ {
		sa, err := s.serviceAccount.CreateServiceAccount(ctx)
		if err != nil {
			return err
		}

		err = s.saPool.AddServiceAccount(ctx, sa)
		if err != nil {
			return err
		}

		err = s.project.AddServiceAccountToProjects(ctx, sa.ProjectId)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *GcpStandaloneServiceAccountsService) InitEnvironment(ctx context.Context) error {
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

func (s *GcpStandaloneServiceAccountsService) GetNextFreeServiceAccount(ctx context.Context, customerID string) (string, error) {
	return s.saPool.GetNextFreeServiceAccount(ctx, customerID)
}

func (s *GcpStandaloneServiceAccountsService) MarkServiceAccountOnBoardSuccessful(ctx context.Context, saEmail, customerID, billingAccountID string) error {
	return s.saPool.MarkServiceAccountOnBoardSuccessful(ctx, saEmail, customerID, billingAccountID)
}
