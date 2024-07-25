package service

import (
	"context"
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/dataStructures"
	billingService "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/service"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/bq_utils"
	serviceAccountUtils "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/service_accounts/utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/retry"
	cloudresourcemanager "google.golang.org/api/cloudresourcemanager/v1"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"

	"strings"
	"sync"
	"time"
)

type ServiceAccount struct {
	loggerProvider logger.Provider
	*connection.Connection
	bqUtils          *bq_utils.BQ_Utils
	lock             *sync.Mutex
	metadataDal      *dal.Metadata
	customerBQClient billingService.ExternalBigQueryClient
}

func NewServiceAccount(log logger.Provider, conn *connection.Connection) *ServiceAccount {
	return &ServiceAccount{
		loggerProvider:   log,
		Connection:       conn,
		bqUtils:          bq_utils.NewBQ_UTils(log, conn),
		lock:             &sync.Mutex{},
		metadataDal:      dal.NewMetadata(log, conn),
		customerBQClient: billingService.NewExternalBigQueryClient(log, conn),
	}
}

func (s *ServiceAccount) CreateServiceAccountsMetadata(ctx context.Context) (err error) {
	logger := s.loggerProvider(ctx)

	sas, err := s.completeTheRemainingSAinProject(ctx)
	if err != nil {
		err = fmt.Errorf("unable to completeTheRemainingSAinProject. Caused by %s", err)
		logger.Error(err)

		return err
	}

	err = s.metadataDal.CreateServiceAccountsMetadata(ctx, sas)
	if err != nil {
		err = fmt.Errorf("unable to CreateServiceAccountsMetadata. Caused by %s", err)
		logger.Error(err)

		return err
	}

	return nil
}

func (s *ServiceAccount) getIamClient(ctx context.Context) (iamClient *iam.Service, err error) {
	logger := s.loggerProvider(ctx)

	ts, err := impersonate.CredentialsTokenSource(ctx, impersonate.CredentialsConfig{
		TargetPrincipal: consts.ServiceAccount,
		Scopes:          []string{"https://www.googleapis.com/auth/cloud-platform"},
	})
	if err != nil {
		err = fmt.Errorf("failed to impersonate.CredentialsTokenSource. Caused by %s", err)
		logger.Error(err)

		return nil, err
	}

	iamClient, err = iam.NewService(context.Background(), option.WithTokenSource(ts))
	if err != nil {
		err = fmt.Errorf("failed to create iamClient service. Caused by %s", err)
		logger.Error(err)

		return nil, err
	}

	return iamClient, nil
}

func (s *ServiceAccount) DeleteServiceAccountsMetadata(ctx context.Context) error {
	return s.metadataDal.DeleteAllServiceAccountsMetadata(ctx)
}

func (s *ServiceAccount) completeTheRemainingSAinProject(ctx context.Context) (updatedSAs []*iam.ServiceAccount, err error) {
	logger := s.loggerProvider(ctx)

	iamClient, err := s.getIamClient(ctx)
	if err != nil {
		err = fmt.Errorf("unable to getIamClient. Caused by %s", err)
		logger.Error(err)

		return nil, err
	}

	serviceList, err := iamClient.Projects.ServiceAccounts.List(serviceAccountUtils.GetFullProjectResourceName(consts.DummyBQProjectName)).PageSize(int64(consts.MaxAllowedSAinProject)).Do()
	if err != nil {
		err = fmt.Errorf("unable to ServiceAccounts.List. Caused by %s", err)
		logger.Error(err)

		return nil, err
	}

	if len(serviceList.Accounts) < consts.MaxAllowedSAinProject {
		for i := 0; i < consts.MaxAllowedSAinProject-len(serviceList.Accounts); i++ {
			err = s.createBA(ctx)
			if err != nil {
				err = fmt.Errorf("unable to createBA. Caused by %s", err)
				logger.Error(err)

				return nil, err
			}

			time.Sleep(time.Second)
		}
	}

	serviceList, err = iamClient.Projects.ServiceAccounts.List(serviceAccountUtils.GetFullProjectResourceName(consts.DummyBQProjectName)).PageSize(int64(consts.MaxAllowedSAinProject)).Do()
	if err != nil {
		err = fmt.Errorf("unable to ServiceAccounts.List. Caused by %s", err)
		logger.Error(err)

		return nil, err
	}

	for _, sa := range serviceList.Accounts {
		if strings.Contains(sa.Name, consts.DummySAPrefix) {
			updatedSAs = append(updatedSAs, sa)
		}
	}

	return updatedSAs, nil
}

func (s *ServiceAccount) createBA(ctx context.Context) (err error) {
	logger := s.loggerProvider(ctx)

	iamClient, err := s.getIamClient(ctx)
	if err != nil {
		err = fmt.Errorf("unable to getIamClient. Caused by %s", err)
		logger.Error(err)

		return err
	}

	sa, err := iamClient.Projects.ServiceAccounts.Create(serviceAccountUtils.GetFullProjectResourceName(consts.DummyBQProjectName), &iam.CreateServiceAccountRequest{
		AccountId:      fmt.Sprintf(consts.DummySATemplate, common.RandomSequenceN(15)),
		ServiceAccount: &iam.ServiceAccount{},
	}).Context(ctx).Do()
	if err != nil {
		err = fmt.Errorf("unable to ServiceAccounts.Create. Caused by %s", err)
		logger.Error(err)

		return err
	}

	logger.Infof("SA %s created", sa.Name)

	return nil
}

func (s *ServiceAccount) GetServiceAccountForBillingAccount(ctx context.Context, billingAccount string, tableName string) (sa *dataStructures.ServiceAccount, err error) {
	logger := s.loggerProvider(ctx)
	//sa, err = s.metadataDal.GetNextNonFullServiceAccount(ctx, billingAccount)
	//if err != nil {
	//	err = fmt.Errorf("unable to get GetNextNonFullServiceAccount. Caused by %s", err)
	//	logger.Error(err)
	//	return nil, err
	//}
	//err = s.grantPermissionsToTable(ctx, tableName, sa.ServiceAccountID)
	//if err != nil {
	//	err = fmt.Errorf("unable to get grantPermissionsToTable. Caused by %s", err)
	//	logger.Error(err)
	//	return nil, err
	//}
	//return sa, nil
	//retryableFunc func() error, retries int, delay time.Duration
	err = retry.FixedDelay(func() error {
		sa, err = s.metadataDal.GetNextNonFullServiceAccount(ctx, billingAccount)
		if err != nil {
			err = fmt.Errorf("unable to GetNextNonFullServiceAccount. Caused by %s", err)
			logger.Error(err)

			return err
		}

		return nil
	}, 100, time.Second)

	return sa, err
}

//func (s *ServiceAccount) grantPermissionsToTable(ctx context.Context, tableName string, sa string) error {
//	logger := s.loggerProvider(ctx)
//
//	customerBQ, err := s.customerBQClient.GetCustomerBQClientByServiceAccount(ctx, consts.ServiceAccount, consts.DummyBQProjectName)
//	if err != nil {
//		err = fmt.Errorf("unable to GetCustomerBQClient.Caused by %s", err)
//		logger.Error(err)
//		return err
//	}
//
//	defer customerBQ.Close()
//	policy, err := customerBQ.Dataset(consts.DummyBQDatasetName).Table(tableName).IAM().Policy(ctx)
//	if err != nil {
//		err = fmt.Errorf("unable to get policies.Caused by %s", err)
//		logger.Error(err)
//		return err
//	}
//
//	policy.Add(fmt.Sprintf("serviceAccount:%s", sa), iamproto.Owner)
//	err = customerBQ.Dataset(consts.DummyBQDatasetName).Table(tableName).IAM().SetPolicy(ctx, policy)
//	if err != nil {
//		err = fmt.Errorf("unable to set policy. Caused by %s", err)
//		logger.Error(err)
//		return err
//	}
//
//	//for _, role := range policy.Roles() {
//	//	if role == "roles/bigquery.admin" {
//	//
//	//	}
//	//}
//
//	//err = customerBQ.Dataset(consts.DummyBQDatasetName).Table(tableName).V3().SetPolicy(ctx, policy)
//	//if err != nil {
//	//	err = fmt.Errorf("unable to set policy. Caused by %s", err)
//	//	logger.Error(err)
//	//	return err
//	//}
//	return nil
//}

func (s *ServiceAccount) getCrmClient(ctx context.Context) (crmClient *cloudresourcemanager.Service, err error) {
	logger := s.loggerProvider(ctx)

	ts, err := impersonate.CredentialsTokenSource(ctx, impersonate.CredentialsConfig{
		TargetPrincipal: consts.ServiceAccount,
		Scopes:          []string{"https://www.googleapis.com/auth/cloud-platform"},
	})
	if err != nil {
		err = fmt.Errorf("failed to impersonate.CredentialsTokenSource. Caused by %s", err)
		logger.Error(err)

		return nil, err
	}

	crmClient, err = cloudresourcemanager.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		err = fmt.Errorf("failed to create cloudresourcemanager service. Caused by %s", err)
		logger.Error(err)

		return nil, err
	}

	return crmClient, nil
}

func (s *ServiceAccount) GrantPermissionsToSAs(ctx context.Context) error {
	logger := s.loggerProvider(ctx)

	sas, err := s.metadataDal.GetServiceAccountsMetadata(ctx)
	if err != nil {
		err = fmt.Errorf("unable to get GetServiceAccountsMetadata. Caused by %s", err)
		logger.Error(err)

		return err
	}

	iamClient, err := s.getIamClient(ctx)
	if err != nil {
		err = fmt.Errorf("unable to getIamClient. Caused by %s", err)
		logger.Error(err)

		return err
	}

	for _, sa := range sas {
		iamPolicy, err := iamClient.Projects.ServiceAccounts.GetIamPolicy(sa.Name).Context(ctx).Do()
		if err != nil {
			err = fmt.Errorf("failed to get IAM policy. Caused by %s", err)
			logger.Error(err)

			return err
		}

		roleExists := false

		for _, binding := range iamPolicy.Bindings {
			if binding.Role == "roles/iam.serviceAccountTokenCreator" {
				roleExists = true

				binding.Members = append(append(append(binding.Members, fmt.Sprintf("user:%s", consts.LionelSA)),
					fmt.Sprintf("serviceAccount:%s", consts.ServiceAccount)),
					fmt.Sprintf("serviceAccount:%s@appspot.gserviceaccount.com", common.ProjectID))
				//
			}
		}

		if !roleExists {
			iamPolicy.Bindings = append(iamPolicy.Bindings, &iam.Binding{
				Role: "roles/iam.serviceAccountTokenCreator",
				Members: []string{
					//consts.ServiceAccount,
					fmt.Sprintf("user:%s", consts.LionelSA),
					fmt.Sprintf("serviceAccount:%s@appspot.gserviceaccount.com", common.ProjectID),
					fmt.Sprintf("serviceAccount:%s", consts.ServiceAccount),
				},
			})
		}

		_, err = iamClient.Projects.ServiceAccounts.SetIamPolicy(sa.Name, &iam.SetIamPolicyRequest{
			Policy: &iam.Policy{
				Bindings: iamPolicy.Bindings, Etag: iamPolicy.Etag,
			},
		}).Do()
		if err != nil {
			err = fmt.Errorf("failed to set IAM policy. Caused by %s", err)
			logger.Error(err)

			return err
		}
	}

	crmClient, err := s.getCrmClient(ctx)
	if err != nil {
		err = fmt.Errorf("failed to getCrmClient. Caused by %s", err)
		logger.Error(err)

		return err
	}

	projectPolicies, err := crmClient.Projects.GetIamPolicy(consts.DummyBQProjectName, &cloudresourcemanager.GetIamPolicyRequest{}).Do()
	if err != nil {
		err = fmt.Errorf("failed to get project GetIamPolicy. Caused by %s", err)
		logger.Error(err)

		return err
	}

	roleExists := false

	for _, binding := range projectPolicies.Bindings {
		if binding.Role == "roles/bigquery.admin" {
			roleExists = true
			binding.Members = mergeListWithoutDuplicates(binding.Members, sas)

			break
		}
	}

	if !roleExists {
		projectPolicies.Bindings = append(projectPolicies.Bindings, &cloudresourcemanager.Binding{
			Role:    "roles/bigquery.admin",
			Members: mergeListWithoutDuplicates([]string{}, sas),
		})
	}

	_, err = crmClient.Projects.SetIamPolicy(consts.DummyBQProjectName, &cloudresourcemanager.SetIamPolicyRequest{
		Policy: &cloudresourcemanager.Policy{
			Bindings: projectPolicies.Bindings, Etag: projectPolicies.Etag,
		},
	}).Do()

	if err != nil {
		err = fmt.Errorf("failed to update project policy. Caused by %s", err)
		logger.Error(err)

		return err
	}

	return nil
}

func mergeListWithoutDuplicates(bindingMembers []string, sas []*dataStructures.ServiceAccount) (set []string) {
	for _, sa := range sas {
		bindingMembers = append(bindingMembers, fmt.Sprintf("serviceAccount:%s", sa.ServiceAccountID))
	}

	m := map[string]string{}
	for _, binding := range bindingMembers {
		m[binding] = binding
	}

	for unique := range m {
		set = append(set, unique)
	}

	return set
}
