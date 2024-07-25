package service

import (
	"context"
	"errors"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/iam/v1"

	"github.com/doitintl/cloudtasks/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/service_accounts/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/service_accounts/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/service_accounts/utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type ServiceAccountsPoolService struct {
	loggerProvider logger.Provider
	*connection.Connection
	dal             *dal.OnBoardingFirestore
	customerDAL     customerDal.Customers
	cloudTaskClient iface.CloudTaskClient
}

func NewServiceAccountsPoolService(log logger.Provider, conn *connection.Connection, cloudTaskClient iface.CloudTaskClient) *ServiceAccountsPoolService {
	return &ServiceAccountsPoolService{
		log,
		conn,
		dal.NewOnBoardingFirestoreWithClient(log, conn),
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
		cloudTaskClient,
	}
}

func (s *ServiceAccountsPoolService) GetNextFreeServiceAccount(ctx context.Context, customerID string) (string, error) {
	saEmail, err := s.GetCustomerReservedServiceAccount(ctx, customerID, "")
	if err != nil {
		return "", err
	}

	if saEmail != "" {
		return saEmail, nil
	}

	customerRef := s.customerDAL.GetRef(ctx, customerID)

	saEmail, shouldCreateNewSA, err := s.dal.GetNextFreeServiceAccount(ctx, customerRef)
	if err != nil {
		return "", err
	}

	if shouldCreateNewSA {
		if err := s.runCreateNewSASetTask(ctx); err != nil {
			return "", err
		}
	}

	return saEmail, nil
}

func (s *ServiceAccountsPoolService) runCreateNewSASetTask(ctx context.Context) error {
	config := common.CloudTaskConfig{
		Method: cloudtaskspb.HttpMethod_GET,
		Path:   utils.CreateNewServiceAccountsURL,
		Queue:  common.TaskQueueFlexsaveStandaloneGCPNewServiceAccounts,
	}

	_, err := s.cloudTaskClient.CreateTask(ctx, config.Config(nil))

	return err
}

type onboardSuccessfulData struct {
	saEmail          string
	customerID       string
	billingAccountID string
}

func (s *ServiceAccountsPoolService) MarkServiceAccountOnBoardSuccessful(ctx context.Context, saEmail, customerID, billingAccountID string) error {
	_, err := s.dal.SetServiceAccountsPool_w_Transaction(ctx, markServiceAccountTaken, &onboardSuccessfulData{saEmail, customerID, billingAccountID})
	return err
}

func markServiceAccountTaken(docSnap *firestore.DocumentSnapshot, aux interface{}) (interface{}, error) {
	var pool dataStructures.ServiceAccountsPool
	if err := docSnap.DataTo(&pool); err != nil {
		return "", err
	}

	saEmail := aux.(*onboardSuccessfulData).saEmail

	if saMetadata, ok := pool.Reserved[saEmail]; ok {
		delete(pool.Reserved, saEmail)

		if len(pool.Taken) == 0 {
			pool.Taken = make(map[string]dataStructures.ServiceAccountMetadata)
		}

		saMetadata.BillingAccountID = aux.(*onboardSuccessfulData).billingAccountID
		pool.Taken[saEmail] = saMetadata
	} else {
		customerID := aux.(*onboardSuccessfulData).customerID
		billingAccountID := aux.(*onboardSuccessfulData).billingAccountID

		if saMetadata, ok := pool.Taken[saEmail]; ok && saMetadata.BillingAccountID == billingAccountID && saMetadata.Customer.ID == customerID {
			return pool, nil
		}

		return "", errors.New("no service account was reserved for this billing account")
	}

	return pool, nil
}

func (s *ServiceAccountsPoolService) GetCustomerReservedServiceAccount(ctx context.Context, customerID, billingAccountID string) (string, error) {
	if customerID == "" && billingAccountID == "" {
		return "", errors.New("customerID and billingAccountID cannot be empty")
	}

	pool, err := s.dal.GetServiceAccountsPool(ctx)
	if err != nil {
		return "", err
	}

	if key := s.getServiceAccountFromPool(pool.Reserved, customerID, billingAccountID); key != "" {
		return key, nil
	}

	return s.getServiceAccountFromPool(pool.Taken, customerID, billingAccountID), nil
}

func (s *ServiceAccountsPoolService) getServiceAccountFromPool(pool map[string]dataStructures.ServiceAccountMetadata, customerID, billingAccountID string) string {
	for key, value := range pool {
		if billingAccountID != "" {
			if value.BillingAccountID == billingAccountID {
				return key
			}
		} else {
			if value.Customer.ID == customerID {
				return key
			}
		}
	}

	return ""
}

func (s *ServiceAccountsPoolService) AddServiceAccount(ctx context.Context, sa *iam.ServiceAccount) error {
	_, err := s.dal.SetServiceAccountsPool_w_Transaction(ctx, addNewServiceAccount, sa)
	if err != nil {
		return err
	}

	return nil
}

func addNewServiceAccount(docSnap *firestore.DocumentSnapshot, aux interface{}) (interface{}, error) {
	var pool dataStructures.ServiceAccountsPool
	if err := docSnap.DataTo(&pool); err != nil {
		return "", err
	}

	sa := aux.(*iam.ServiceAccount)

	if pool.Free == nil {
		pool.Free = make(map[string]dataStructures.ServiceAccountMetadata, 0)
	}

	pool.Free[sa.Email] = dataStructures.ServiceAccountMetadata{
		ProjectID: sa.ProjectId,
	}

	return &pool, nil
}
