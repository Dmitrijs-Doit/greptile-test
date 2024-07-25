package service

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/api/cloudresourcemanager/v2"
	"google.golang.org/api/iam/v1"

	"github.com/doitintl/googleadmin"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/dal"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/utils"
	"github.com/doitintl/retry"
	admin "google.golang.org/api/admin/directory/v1"
	cloudbilling "google.golang.org/api/cloudbilling/v1"
)

type ServiceAccountService struct {
	loggerProvider logger.Provider
	*connection.Connection
	adminService        googleadmin.GoogleAdmin
	iamService          *iam.Service
	cloudbillingService *cloudbilling.APIService
	foldersService      *cloudresourcemanager.FoldersService
	dal                 *dal.ServiceAccountsFirestore
	customerDAL         customerDal.Customers
}

func NewServiceAccountService(log logger.Provider, conn *connection.Connection) *ServiceAccountService {
	ctx := context.Background()

	adminService, err := googleadmin.NewGoogleAdmin(ctx, common.ProjectID)
	if err != nil {
		return nil
	}

	iamService, err := iam.NewService(ctx)
	if err != nil {
		return nil
	}

	cloudbillingService, err := cloudbilling.NewService(ctx)
	if err != nil {
		return nil
	}

	crmService, err := cloudresourcemanager.NewService(ctx)
	if err != nil {
		return nil
	}

	foldersService := cloudresourcemanager.NewFoldersService(crmService)

	return &ServiceAccountService{
		log,
		conn,
		adminService,
		iamService,
		cloudbillingService,
		foldersService,
		dal.NewServiceAccountsFirestoreWithClient(log, conn),
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
	}
}

func (s *ServiceAccountService) CreateServiceAccount(ctx context.Context) (*iam.ServiceAccount, error) {
	logger := s.loggerProvider(ctx)

	projectID, err := s.dal.GetCurrentProject(ctx)
	if err != nil {
		logger.Error("Failed to get current project", err)
		return nil, err
	}

	projectResourceName := utils.GetFullProjectResourceName(projectID)
	request := utils.CreateNewServiceAccountRequest()

	newServiceAccount, err := s.iamService.Projects.ServiceAccounts.Create(projectResourceName, request).Context(ctx).Do()
	if err != nil {
		logger.Error("Failed to create service account", err)
		return nil, err
	}

	resourceName := utils.GetFullServiceAccountResourceName(projectID, newServiceAccount)

	var saIamPolicy *iam.Policy

	err = retry.BackOffDelay(
		func() error {
			saIamPolicy, err = s.iamService.Projects.ServiceAccounts.GetIamPolicy(resourceName).Context(ctx).Do()
			if err != nil {
				logger.Error("Failed to get IAM policy", err)
				return err
			}

			return nil
		},
		5,
		time.Second*1,
	)

	if err != nil {
		logger.Error("Failed to get IAM policy", err)
		return nil, err
	}

	saIamPolicy.Bindings = append(saIamPolicy.Bindings, utils.GetTokenCreatorBinding())

	_, err = s.iamService.Projects.ServiceAccounts.SetIamPolicy(resourceName, &iam.SetIamPolicyRequest{
		Policy: &iam.Policy{
			Bindings: saIamPolicy.Bindings, Etag: saIamPolicy.Etag,
		},
	}).Do()
	if err != nil {
		logger.Error("Failed to set Service Account IAM policy", err)
		return nil, err
	}

	logger.Infof("Successfully created service account %s", newServiceAccount.Email)

	return newServiceAccount, nil
}

func (s *ServiceAccountService) AddServiceAccountToGroup(ctx context.Context, serviceAccountEmail string) error {
	logger := s.loggerProvider(ctx)

	_, err := s.adminService.InsertGroupMember(utils.DedicatedCustomersServiceAccountsGroup,
		&admin.Member{
			Role:  "MEMBER",
			Email: serviceAccountEmail,
		})
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to add %s to a Google Group", serviceAccountEmail), err)
		return err
	}

	return nil
}
