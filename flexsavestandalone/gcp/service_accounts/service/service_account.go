package service

import (
	"context"
	"time"

	"google.golang.org/api/cloudresourcemanager/v2"
	"google.golang.org/api/iam/v1"

	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/service_accounts/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/service_accounts/utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/retry"
	cloudbilling "google.golang.org/api/cloudbilling/v1"
)

type ServiceAccountService struct {
	loggerProvider logger.Provider
	*connection.Connection
	iamService          *iam.Service
	cloudbillingService *cloudbilling.APIService
	foldersService      *cloudresourcemanager.FoldersService
	dal                 *dal.OnBoardingFirestore
	customerDAL         customerDal.Customers
}

func NewServiceAccountService(log logger.Provider, conn *connection.Connection) *ServiceAccountService {
	ctx := context.Background()

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
		iamService,
		cloudbillingService,
		foldersService,
		dal.NewOnBoardingFirestoreWithClient(log, conn),
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

	// TODO: add here binding to custom flexsave role

	_, err = s.iamService.Projects.ServiceAccounts.SetIamPolicy(resourceName, &iam.SetIamPolicyRequest{
		Policy: &iam.Policy{
			Bindings: saIamPolicy.Bindings, Etag: saIamPolicy.Etag,
		},
	}).Do()
	if err != nil {
		logger.Error("Failed to set Service Account IAM policy", err)
		return nil, err
	}

	resourceName = utils.GetFallbackBillingAccountResourceName()

	baIamPolicy, err := s.cloudbillingService.BillingAccounts.GetIamPolicy(resourceName).Context(ctx).Do()
	if err != nil {
		logger.Error("Failed to get Fallback Billing Account IAM policy", err)
		return nil, err
	}

	baIamPolicy.Bindings = append(baIamPolicy.Bindings, utils.GetFallbackBillingAccountBinding(newServiceAccount.Email))

	_, err = s.cloudbillingService.BillingAccounts.SetIamPolicy(resourceName, &cloudbilling.SetIamPolicyRequest{
		Policy: &cloudbilling.Policy{
			Bindings: baIamPolicy.Bindings, Etag: baIamPolicy.Etag,
		},
	}).Do()
	if err != nil {
		logger.Error("Failed to set Fallback Billing Account IAM policy", err)
		return nil, err
	}

	resourceName = utils.GetFlexsaveProjectsFolderResourceName()

	folderIamPolicy, err := s.foldersService.GetIamPolicy(resourceName, &cloudresourcemanager.GetIamPolicyRequest{}).Context(ctx).Do()
	if err != nil {
		logger.Error("Failed to get Flexsave Projects Folder IAM policy", err)
		return nil, err
	}

	folderIamPolicy.Bindings = append(folderIamPolicy.Bindings, utils.GetFlexsaveProjectsFolderBinding(newServiceAccount.Email))
	setIamPolicyRequest := &cloudresourcemanager.SetIamPolicyRequest{
		Policy: &cloudresourcemanager.Policy{
			Bindings: folderIamPolicy.Bindings, Etag: folderIamPolicy.Etag,
		},
	}

	_, err = s.foldersService.SetIamPolicy(resourceName, setIamPolicyRequest).Context(ctx).Do()
	if err != nil {
		logger.Error("Failed to set Flexsave Projects Folder IAM policy", err)
		return nil, err
	}

	logger.Infof("Successfully created service account %s", newServiceAccount.Email)

	return newServiceAccount, nil
}
