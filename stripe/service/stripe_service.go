package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	entityDal "github.com/doitintl/hello/scheduled-tasks/entity/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	priorityDal "github.com/doitintl/hello/scheduled-tasks/priority/dal"
	priority "github.com/doitintl/hello/scheduled-tasks/priority/service"
	priorityIface "github.com/doitintl/hello/scheduled-tasks/priority/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"github.com/doitintl/hello/scheduled-tasks/stripe/dal"
	httpClient "github.com/doitintl/http"
	"github.com/doitintl/idtoken"
)

const (
	// TODO(dror): Set dev function url
	handlePriorityProcedureDevURL  = "https://us-central1-me-doit-intl-com.cloudfunctions.net/handlePriorityProcedure"
	handlePriorityProcedureProdURL = "https://us-central1-me-doit-intl-com.cloudfunctions.net/handlePriorityProcedure"
)

type StripeService struct {
	loggerProvider logger.Provider
	*connection.Connection
	stripeClient     *Client
	priorityService  priorityIface.Service
	integrationDocID string
	stripeDAL        dal.IStripeFirestore
	customersDAL     customerDal.Customers
	entitiesDAL      entityDal.Entites
}

func NewStripeService(loggerProvider logger.Provider, conn *connection.Connection, stripeClient *Client) (*StripeService, error) {
	ctx := context.Background()

	priorityClient, prioritySec, err := getPriorityClientWithSecret(ctx)
	if err != nil {
		return nil, err
	}

	priorityProcedureClient, err := getPriorityProcedureClient(ctx)
	if err != nil {
		return nil, err
	}

	avalaraClient, err := httpClient.NewClient(ctx, &httpClient.Config{
		BaseURL: "https://rest.avatax.com/api/v2",
	})
	if err != nil {
		return nil, err
	}

	priorityFirestoreDal := priorityDal.NewPriorityFirestoreWithClient(conn.Firestore)

	priorityReaderWriter, err := priorityDal.NewPriorityDAL(
		loggerProvider,
		priorityFirestoreDal,
		priorityClient,
		avalaraClient,
		priorityProcedureClient,
		priorityDal.WithPriorityUserName(prioritySec.Username),
		priorityDal.WithPriorityPassword(prioritySec.Password),
	)
	if err != nil {
		return nil, err
	}

	priorityService, err := priority.NewService(loggerProvider, conn, *priorityFirestoreDal, priorityReaderWriter)
	if err != nil {
		return nil, err
	}

	// TODO(yoni): remove this when we have the dal completed
	integrationDocID := stripeClient.integrationDocID

	return &StripeService{
		loggerProvider,
		conn,
		stripeClient,
		priorityService,
		integrationDocID,
		dal.NewStripeFirestoreWithClient(conn.Firestore, integrationDocID),
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
		entityDal.NewEntitiesFirestoreWithClient(conn.Firestore),
	}, nil
}

func (s *StripeService) ValidateUserPermissions(ctx context.Context, customerID, userID string) (bool, error) {
	if customerID == "" {
		return false, ErrCustomerNotFound
	}

	if userID == "" {
		return false, ErrInvalidUser
	}

	userRef := s.Firestore(ctx).Collection("users").Doc(userID)

	user, err := common.GetUser(ctx, userRef)
	if err != nil {
		return false, err
	}

	if !user.HasEntitiesPermission(ctx) {
		return false, err
	}

	if user.Customer.Ref == nil || user.Customer.Ref.ID != customerID {
		return false, ErrCustomerNotFound
	}

	return true, nil
}

type prioritySecret struct {
	URL      string `json:"url"`
	BaseURL  string `json:"api_url"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func getPriorityClientWithSecret(ctx context.Context) (httpClient.IClient, prioritySecret, error) {
	data, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretPriority)
	if err != nil {
		return nil, prioritySecret{}, err
	}

	var secret prioritySecret

	err = json.Unmarshal(data, &secret)
	if err != nil {
		return nil, prioritySecret{}, err
	}

	priorityClient, err := httpClient.NewClient(ctx, &httpClient.Config{
		BaseURL: secret.BaseURL,
		Timeout: 120 * time.Second,
	})
	if err != nil {
		return nil, prioritySecret{}, err
	}

	return priorityClient, secret, nil
}

func getPriorityProcedureClient(ctx context.Context) (httpClient.IClient, error) {
	handlePriorityProcedureURL := handlePriorityProcedureDevURL
	if common.Production {
		handlePriorityProcedureURL = handlePriorityProcedureProdURL
	}

	secret, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretAppEngine)
	if err != nil {
		return nil, err
	}

	tokenSource, err := idtoken.New().GetServiceAccountTokenSource(ctx, handlePriorityProcedureURL, secret)
	if err != nil {
		return nil, err
	}

	priorityProcedureClient, err := httpClient.NewClient(ctx, &httpClient.Config{
		BaseURL:     handlePriorityProcedureURL,
		Timeout:     120 * time.Second,
		TokenSource: tokenSource,
	})
	if err != nil {
		return nil, err
	}

	return priorityProcedureClient, nil
}
