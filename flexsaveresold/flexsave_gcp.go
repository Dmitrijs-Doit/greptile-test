package flexsaveresold

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"github.com/doitintl/http"
)

const (
	localGcpAPIService = "http://localhost:8086"
	devGcpAPIService   = "https://cmp-flexsave-gcp-api-wsqwprteya-uc.a.run.app"
	prodGcpAPIService  = "https://cmp-flexsave-gcp-api-alqysnpjoq-uc.a.run.app"
)

//go:generate mockery --output=./mocks --name FlexsaveGCPServiceInterface
type FlexsaveGCPServiceInterface interface {
	EnableFlexsaveGCP(ctx context.Context, customerID string, userID string, doitEmployee bool, email string) error
}

type GCPService struct {
	loggerProvider logger.Provider
	*connection.Connection
	gcpAPIClient  *http.Client
	flexRIService *Service
	email         EmailInterface
}

func NewGCPService(loggerProvider logger.Provider, conn *connection.Connection) *GCPService {
	gcpAPIService := devGcpAPIService
	flexRIService := NewService(loggerProvider, conn)
	email := NewMail(loggerProvider, conn)

	if common.Production {
		gcpAPIService = prodGcpAPIService
	}

	// FOR LOCAL DEVELOPMENT ONLY
	// gcpAPIService = localGcpAPIService

	ctx := context.Background()

	c, err := http.NewClient(ctx, &http.Config{
		BaseURL: gcpAPIService,
		Timeout: 360 * time.Second,
	})
	if err != nil {
		panic(err)
	}

	return &GCPService{
		loggerProvider,
		conn,
		c,
		flexRIService,
		email,
	}
}

func (s *GCPService) setBearerToken(ctx context.Context) (context.Context, error) {
	secret, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretAppEngine)
	if err != nil {
		return nil, err
	}

	token, err := common.GetServiceAccountIDToken(ctx, s.gcpAPIClient.URL(), secret)
	if err != nil {
		return nil, err
	}

	ctx = http.WithBearerAuth(ctx, &http.BearerAuthContextData{
		Token: fmt.Sprintf("Bearer %s", token.AccessToken),
	})

	return ctx, nil
}

func (s *GCPService) Execute(ctx context.Context, email string, customerIDs []string, workloads interface{}) error {
	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return err
	}

	req := struct {
		ApprovedBy  string      `json:"approvedBy"`
		CustomerIDs []string    `json:"customerIds"`
		Workloads   interface{} `json:"workloads,omitempty"`
	}{
		email,
		customerIDs,
		workloads,
	}

	_, err = s.gcpAPIClient.Post(ctx, &http.Request{
		URL:     "/purchase-plan/execute",
		Payload: req,
	})
	if err != nil {
		return err
	}

	return nil
}

// purchase for a single customer's specific workloads plans
func (s *GCPService) Ops2ExecuteCustomerPlans(ctx context.Context, email string, customerID string, workloads interface{}, dryRun bool) error {
	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return err
	}

	req := struct {
		ApprovedBy string      `json:"approved_by"`
		Workloads  interface{} `json:"workloads,omitempty"`
		DryRun     bool        `json:"dry_run"`
	}{
		email,
		workloads,
		dryRun,
	}

	_, err = s.gcpAPIClient.Post(ctx, &http.Request{
		URL:     fmt.Sprintf("/purchase/customers/%s/workloads", customerID),
		Payload: req,
	})
	if err != nil {
		return err
	}

	return nil
}

type CustomerPurchasePlans struct {
	CustomerID string        `json:"customer_id"`
	Workloads  []interface{} `json:"workloads"`
}

// purchase for multipe customer's multiple plans
func (s *GCPService) Ops2ExecuteMultipleCustomerPlans(ctx context.Context, email string, plans []CustomerPurchasePlans, dryRun bool) error {
	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return err
	}

	// create request dto
	req := struct {
		ApprovedBy string                  `json:"approved_by"`
		Approved   []CustomerPurchasePlans `json:"approved"`
		DryRun     bool                    `json:"dry_run"`
	}{
		email,
		plans,
		dryRun,
	}

	_, err = s.gcpAPIClient.Post(ctx, &http.Request{
		URL:     fmt.Sprintf("/purchase/customers/workloads"),
		Payload: req,
	})
	if err != nil {
		return err
	}

	return nil
}

// purchase for a single customer (purchase customer's all workloads)
func (s *GCPService) Ops2ExecuteCustomer(ctx context.Context, email string, customerID string, dryRun bool) error {
	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return err
	}

	req := struct {
		ApprovedBy string `json:"approved_by"`
		DryRun     bool   `json:"dry_run"`
	}{
		email,
		dryRun,
	}

	_, err = s.gcpAPIClient.Post(ctx, &http.Request{
		URL:     fmt.Sprintf("/purchase/customers/%s", customerID),
		Payload: req,
	})
	if err != nil {
		return err
	}

	return nil
}

// bulk purchase for multiple customers (purchase multiple customers all workloads)
func (s *GCPService) Ops2ExecuteCustomers(ctx context.Context, email string, customerIDs []string, dryRun bool) error {
	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return err
	}

	req := struct {
		ApprovedBy  string   `json:"approved_by"`
		CustomerIDs []string `json:"customer_ids"`
		DryRun      bool     `json:"dry_run"`
	}{
		email,
		customerIDs,
		dryRun,
	}

	_, err = s.gcpAPIClient.Post(ctx, &http.Request{
		URL:     "/purchase/customers",
		Payload: req,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *GCPService) GetPurchaseplanPrices(ctx context.Context, purchases []interface{}) (interface{}, error) {
	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return nil, err
	}

	req := struct {
		Purchases []interface{} `json:"purchases"`
	}{
		purchases,
	}

	res := struct {
		Purchases []interface{} `json:"purchases"`
	}{}

	_, err = s.gcpAPIClient.Post(ctx, &http.Request{
		URL:          "/purchase-plan/translate",
		Payload:      req,
		ResponseType: &res,
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *GCPService) ManualPurchase(ctx context.Context, email string, cuds []interface{}, dryRun bool) error {
	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return err
	}

	req := struct {
		ApprovedBy string        `json:"approved_by"`
		Cuds       []interface{} `json:"cuds"`
		DryRun     bool          `json:"dry_run"`
	}{
		ApprovedBy: email,
		Cuds:       cuds,
		DryRun:     dryRun,
	}

	_, err = s.gcpAPIClient.Post(ctx, &http.Request{
		URL:     "/purchase/manual",
		Payload: req,
	})
	if err != nil {
		return err
	}

	return nil
}

// puchase by workloads for all customers
func (s *GCPService) Ops2ExecuteBulk(ctx context.Context, email string, workloads interface{}, dryRun bool) error {
	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return err
	}

	req := struct {
		ApprovedBy string      `json:"approved_by"`
		Workloads  interface{} `json:"workloads,omitempty"`
		DryRun     bool        `json:"dry_run"`
	}{
		email,
		workloads,
		dryRun,
	}

	_, err = s.gcpAPIClient.Post(ctx, &http.Request{
		URL:     "/purchase/workloads",
		Payload: req,
	})
	if err != nil {
		return err
	}

	return nil
}

// Update bulk purchase risks
func (s *GCPService) Ops2UpdateBulk(ctx context.Context) error {
	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return err
	}

	_, err = s.gcpAPIClient.Get(ctx, &http.Request{
		URL: "/ops/agg-workloads",
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *GCPService) Refresh(ctx context.Context) error {
	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return err
	}

	_, err = s.gcpAPIClient.Get(ctx, &http.Request{
		URL: "/purchase-plan/refresh",
	})
	if err != nil {
		return err
	}

	return nil
}

// updates all customers stats recommendation and purchase plans
func (s *GCPService) Ops2Refresh(ctx context.Context) error {
	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return err
	}

	// QUESTION: its take long time to refresh all customers, GO rutine?
	_, err = s.gcpAPIClient.Get(ctx, &http.Request{
		URL: "/ops/update",
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *GCPService) RefreshCustomer(ctx context.Context, customerID string) error {
	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return err
	}

	_, err = s.gcpAPIClient.Get(ctx, &http.Request{
		URL: fmt.Sprintf("/purchase-plan/refresh/%s", customerID),
	})
	if err != nil {
		return err
	}

	return nil
}

// updates specific customer's stats recommendation and purchase plans
func (s *GCPService) Ops2RefreshCustomer(ctx context.Context, customerID string) error {
	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return err
	}

	_, err = s.gcpAPIClient.Get(ctx, &http.Request{
		URL: fmt.Sprintf("/ops/update/customer/%s", customerID),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *GCPService) Create(ctx context.Context, customerID, email string) error {
	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return err
	}

	req := struct {
		OptInBy string `json:"optInBy"`
	}{
		email,
	}

	_, err = s.gcpAPIClient.Post(ctx, &http.Request{
		URL:     fmt.Sprintf("/purchase-plan/create/%s", customerID),
		Payload: req,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *GCPService) DisableCustomer(ctx context.Context, customerID string) error {
	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return err
	}

	dryRun := false
	req := struct {
		DryRun *bool `json:"dryRun"`
	}{
		&dryRun,
	}

	_, err = s.gcpAPIClient.Post(ctx, &http.Request{
		URL:     fmt.Sprintf("/customers/disable/%s", customerID),
		Payload: req,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *GCPService) TestIamPermissions(ctx context.Context, orgID, serviceAccountEmail string, permissions []string, dryRun bool) error {
	if dryRun {
		return nil
	}

	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return err
	}

	req := struct {
		OrganizationID      string   `json:"organization_id"`
		ServiceAccountEmail string   `json:"service_account_email"`
		Permissions         []string `json:"permissions,omitempty"`
	}{
		orgID,
		serviceAccountEmail,
		permissions,
	}

	res := struct {
		Healthy bool   `json:"healthy,omitempty"`
		Error   string `json:"error_message,omitempty"`
	}{}

	_, err = s.gcpAPIClient.Post(ctx, &http.Request{
		URL:          "/test-permissions",
		Payload:      req,
		ResponseType: &res,
	})
	if err != nil {
		return err
	}

	if res.Error != "" {
		return fmt.Errorf("IAM permissions error: %s", res.Error)
	}

	if !res.Healthy {
		return fmt.Errorf("IAM permissions are not valid for service account %s", serviceAccountEmail)
	}

	return nil
}

func (s *GCPService) TestAllocation(ctx context.Context, serviceAccountEmail, billingAccountID string) error {
	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return err
	}

	req := struct {
		BillingAccountID    string `json:"billing_account_id"`
		ServiceAccountEmail string `json:"service_account_email"`
	}{
		billingAccountID,
		serviceAccountEmail,
	}

	res := struct {
		Healthy bool   `json:"healthy,omitempty"`
		Error   string `json:"error_message,omitempty"`
	}{}

	_, err = s.gcpAPIClient.Post(ctx, &http.Request{
		URL:          "/test-permissions/allocation",
		Payload:      req,
		ResponseType: &res,
	})
	if err != nil {
		return err
	}

	if res.Error != "" {
		return fmt.Errorf("test allocation error: %s", res.Error)
	}

	if !res.Healthy {
		return fmt.Errorf("test allocation failed for service account %s", serviceAccountEmail)
	}

	return nil
}

func (s *GCPService) GetBillingAccountDisplayName(ctx context.Context, serviceAccountEmail, billingAccountID string) (string, error) {
	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return "", err
	}

	req := struct {
		BillingAccountID    string `json:"billing_account_id"`
		ServiceAccountEmail string `json:"service_account_email"`
	}{
		billingAccountID,
		serviceAccountEmail,
	}

	res := struct {
		DisplayName string `json:"DisplayName"`
		Name        string `json:"name"`
	}{}

	_, err = s.gcpAPIClient.Post(ctx, &http.Request{
		URL:          "/inventory/billing-accounts/get",
		Payload:      req,
		ResponseType: &res,
	})
	if err != nil {
		return "", err
	}

	return res.DisplayName, nil
}

func (s *GCPService) GetEstimatedSavings(ctx context.Context, serviceAccountEmail string, billingAccountIDs []string) (float64, float64, error) {
	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return 0, 0, err
	}

	req := struct {
		BillingAccountsIDs  []string `json:"billing_accounts_ids"`
		ServiceAccountEmail string   `json:"service_account_email"`
	}{
		billingAccountIDs,
		serviceAccountEmail,
	}

	res := struct {
		MonthlyEstimatedSavings float64 `json:"monthly_estimated_savings"`
		AnnualEstimatedSavings  float64 `json:"annual_estimated_savings"`
	}{}

	_, err = s.gcpAPIClient.Post(ctx, &http.Request{
		URL:          "/recommender/estimated-savings",
		Payload:      req,
		ResponseType: &res,
	})
	if err != nil {
		return 0, 0, err
	}

	return res.MonthlyEstimatedSavings, res.AnnualEstimatedSavings, nil
}

func (s *GCPService) EnableFlexsaveGCP(ctx context.Context, customerID string, userID string, doitEmployee bool, email string) error {
	fs := s.Firestore(ctx)
	log := s.loggerProvider(ctx)

	canEnableFlexSaveGCP, err := s.flexRIService.CanEnableFlexsaveGCP(ctx, customerID, userID, doitEmployee)
	if err != nil {
		return err
	}

	if canEnableFlexSaveGCP.ReasonCantEnable != nil {
		return errors.New(*canEnableFlexSaveGCP.ReasonCantEnable)
	}

	customerRef := fs.Collection("customers").Doc(customerID)

	if err := s.Create(ctx, customerID, email); err != nil {
		return err
	}

	if err := fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		customerConfigRef := fs.Collection("integrations").Doc("flexsave").Collection("configuration").Doc(customerRef.ID)
		data, err := tx.Get(customerConfigRef)
		if err != nil {
			return err
		}

		var config types.ConfigData
		if err := data.DataTo(&config); err != nil {
			return err
		}

		var customer common.Customer
		customerData, err := customerRef.Get(ctx)
		if err != nil {
			return err
		}

		if err := customerData.DataTo(&customer); err != nil {
			return err
		}

		accountManagers, err := common.GetCustomerAccountManagers(ctx, &customer, "doit")
		if err != nil {
			log.Errorf("GetCustomerAccountManagers: %s", err)
		}

		users, err := common.GetCustomerUsersWithPermissions(ctx, fs, customerRef, []string{string(common.PermissionFlexibleRI)})
		if err != nil {
			return err
		}

		marketplace := customer.Marketplace != nil && customer.Marketplace.GCP != nil && customer.Marketplace.GCP.AccountExists

		if err != s.email.SendWelcomeEmail(ctx, &types.WelcomeEmailParams{
			Cloud:       common.GCP,
			CustomerID:  customerID,
			Marketplace: marketplace,
		}, users, accountManagers) {
			log.Error(err)
		}

		err = tx.Update(customerConfigRef, []firestore.Update{
			{
				FieldPath: []string{"GCP", "enabled"},
				Value:     true,
			},
		})
		if err != nil {
			return err
		}

		log.Infof("FlexSave GCP enabled for customer: %s. by %s", customerID, email)

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (s *GCPService) DisableFlexSave(ctx context.Context, customerID string, userID string, doitEmployee bool) error {
	fs := s.Firestore(ctx)
	log := s.loggerProvider(ctx)

	if !doitEmployee {
		if err := assertPermissions(ctx, fs, userID); err != nil {
			return err
		}
	}

	if err := s.DisableCustomer(ctx, customerID); err != nil {
		return err
	}

	log.Infof("FlexSave GCP disabled for customer: %s", customerID)

	return nil
}
