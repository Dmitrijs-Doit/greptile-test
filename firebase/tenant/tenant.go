package tenant

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/v4/auth"

	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/firebase/tenant/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

type TenantService struct {
	dal.Tenants
}

func NewTenantsService(conn *connection.Connection) (*TenantService, error) {
	return &TenantService{
		dal.NewTenantsFirestoreWithClient(conn.Firestore),
	}, nil
}

func NewTenantsServiceWithDalClient(dal dal.Tenants) (*TenantService, error) {
	return &TenantService{
		dal,
	}, nil
}

// GetTenantAuthClientByCustomer retrieve firebase Tenant client based on customerID
func (s *TenantService) GetTenantAuthClientByCustomer(ctx context.Context, customerID string) (*auth.TenantClient, error) {
	tenantID, err := s.GetTenantIDByCustomer(ctx, customerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant id for customer: %s: %w", customerID, err)
	}

	return GetTenantAuthClientByTenantID(ctx, *tenantID)
}

// GetTenantAuthClientByEmail retrieve firebase Tenant client based on email
func (s *TenantService) GetTenantAuthClientByEmail(ctx context.Context, email string) (*auth.TenantClient, error) {
	tenantID, err := s.GetTenantIDByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get tenant id for email: %s", email)
	}

	return GetTenantAuthClientByTenantID(ctx, *tenantID)
}

// GetTenantAuthClientByTenantID retrieve firebase Tenant client based on tenantID
func GetTenantAuthClientByTenantID(ctx context.Context, tenantID string) (*auth.TenantClient, error) {
	fbAuth, err := fb.App.Auth(ctx)
	if err != nil {
		return nil, err
	}

	tenantAuth, err := fbAuth.TenantManager.AuthForTenant(tenantID)
	if err != nil {
		return nil, err
	}

	return tenantAuth, nil
}

// GetTenantAuthClientByCustomer retrieve firebase Tenant client based on customerID with provided firestore client
func GetTenantAuthClientByCustomer(ctx context.Context, fs *firestore.Client, customerID string) (*auth.TenantClient, error) {
	docSnap, err := fs.Collection("tenants").Doc(common.ProjectID).Collection("customerToTenant").Doc(customerID).Get(ctx)
	if err != nil {
		return nil, err
	}

	tid, err := docSnap.DataAt("tenantId")
	if err != nil {
		return nil, err
	}

	tenantID, ok := tid.(string)
	if !ok {
		return nil, fmt.Errorf("failed casting tenant id to string for customer: %s", customerID)
	}

	return GetTenantAuthClientByTenantID(ctx, tenantID)
}
