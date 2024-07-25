package service

import (
	"context"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"google.golang.org/api/iam/v1"

	"github.com/doitintl/cloudtasks/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/dal"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/utils"
)

type ServiceAccountsPoolService struct {
	loggerProvider logger.Provider
	*connection.Connection
	dal             *dal.ServiceAccountsFirestore
	customerDAL     customerDal.Customers
	cloudTaskClient iface.CloudTaskClient
}

func NewServiceAccountsPoolService(log logger.Provider, conn *connection.Connection, cloudTaskClient iface.CloudTaskClient) *ServiceAccountsPoolService {
	return &ServiceAccountsPoolService{
		log,
		conn,
		dal.NewServiceAccountsFirestoreWithClient(log, conn),
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
		cloudTaskClient,
	}
}

func (s *ServiceAccountsPoolService) AddServiceAccount(ctx context.Context, serviceAccount *iam.ServiceAccount) error {
	return s.dal.AddNewServiceAccount(ctx, serviceAccount.Email)
}

func (s *ServiceAccountsPoolService) GetNextFreeServiceAccount(ctx context.Context, customerID string) (string, error) {
	customerRef := s.customerDAL.GetRef(ctx, customerID)

	serviceAccountEmail, shouldCreateNewSA, err := s.dal.GetReservedServiceAccountEmail(ctx, customerRef, "")
	if err != nil {
		return "", err
	}

	if shouldCreateNewSA {
		if err := s.runCreateNewSASetTask(ctx); err != nil {
			s.loggerProvider(ctx).Error("Failed to create new service accounts task: ", err)
		}
	}

	return serviceAccountEmail, nil
}

func (s *ServiceAccountsPoolService) runCreateNewSASetTask(ctx context.Context) error {
	config := common.CloudTaskConfig{
		Method: cloudtaskspb.HttpMethod_GET,
		Path:   utils.CreateNewServiceAccountsURL,
		Queue:  common.TaskQueueBillingSaaSGCPNewServiceAccounts,
	}

	_, err := s.cloudTaskClient.CreateTask(ctx, config.Config(nil))

	return err
}

func (s *ServiceAccountsPoolService) MarkServiceAccountOnBoardSuccessful(ctx context.Context, customerID, saEmail, billingAccountID string) error {
	return s.dal.MoveReservedServiceAccountToTaken(ctx, customerID, saEmail, billingAccountID)
}
