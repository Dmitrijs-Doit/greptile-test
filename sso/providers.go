package sso

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"firebase.google.com/go/v4/auth"

	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/firebase/tenant"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	SAML ConfigType = "saml"
	OIDC ConfigType = "oidc"
)

func NewSSOProviderService(log *logger.Logging, conn *connection.Connection) *ProviderService {
	tenantService, err := tenant.NewTenantsService(conn)

	if err != nil {
		panic(err)
	}

	return &ProviderService{
		log,
		conn,
		tenantService,
		customerDal.NewCustomersFirestoreWithClient(conn.Firestore),
	}
}

// GetAllProviders get first two enabled SSO provider configuration, throw error if both are enabled
func (h *ProviderService) GetAllProviders(ctx context.Context, customerID string) (*ProviderConfig, error) {
	providerConfig := new(ProviderConfig)
	l := h.Logger(ctx)
	tenantAuth, err := h.GetTenantAuthClientByCustomer(ctx, customerID)

	if err != nil {
		l.Errorf("Failed to initialized auth client\n Error: %v", err)
		return nil, err
	}

	var wg sync.WaitGroup

	pTypes := [2]ConfigType{SAML, OIDC}
	done := make(chan bool)
	errs := make(chan error)

	for i := range pTypes {
		wg.Add(1)

		go fillProviderConfigWorker(ctx, &wg, errs, tenantAuth, providerConfig, pTypes[i])
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		break
	case err := <-errs:
		close(errs)
		return nil, err
	}

	if providerConfig.OIDC != nil && providerConfig.OIDC.Enabled && providerConfig.SAML != nil && providerConfig.SAML.Enabled {
		return nil, errors.New("can not have two SSO providers enabled")
	}

	return providerConfig, nil
}

// CreateProvider create a SAML or OIDC configuration; returns spProviderId, callbackURL for SAML; returns callbackURL for OIDC
func (h *ProviderService) CreateProvider(ctx context.Context, customerID string, providerConfig *ProviderConfig) (*ProviderConfig, error) {
	l := h.Logger(ctx)

	tenantAuth, err := h.GetTenantAuthClientByCustomer(ctx, customerID)
	if err != nil {
		l.Errorf("Failed to initialized auth client\n Error: %v", err)
		return nil, err
	}

	customer, err := h.GetCustomer(ctx, customerID)
	if err != nil {
		l.Errorf("Failed to read customer\n Error: %v", err)
		return nil, err
	}

	saml, err := createSAMLProvider(ctx, providerConfig.SAML, customer, tenantAuth)
	if err != nil {
		return nil, err
	}

	oidc, err := createOIDCProvider(ctx, providerConfig.OIDC, customer, tenantAuth)
	if err != nil {
		return nil, err
	}

	return &ProviderConfig{
		SAML: saml,
		OIDC: oidc,
	}, nil
}

// UpdateProvider updates a SAML or OIDC provider configuration
func (h *ProviderService) UpdateProvider(ctx context.Context, customerID string, providerConfig *ProviderConfig) (*ProviderConfig, error) {
	l := h.Logger(ctx)

	tenantAuth, err := h.GetTenantAuthClientByCustomer(ctx, customerID)

	if err != nil {
		l.Errorf("Failed to initialized auth client\n Error: %v", err)
		return nil, err
	}

	saml, err := updateSAMLProvider(ctx, providerConfig.SAML, tenantAuth)

	if err != nil {
		return nil, err
	}

	oidc, err := updateOIDCProvider(ctx, providerConfig.OIDC, tenantAuth)
	if err != nil {
		return nil, err
	}

	return &ProviderConfig{
		SAML: saml,
		OIDC: oidc,
	}, nil
}

func getCallBackURL() string {
	return fmt.Sprintf("https://%s.firebaseapp.com/__/auth/handler", common.ProjectID)
}

func fillProviderConfigWorker(ctx context.Context, wg *sync.WaitGroup, errs chan<- error, t *auth.TenantClient, config *ProviderConfig, pType ConfigType) {
	defer wg.Done()

	switch pType {
	case "oidc":
		oidcConfig, err := getOIDCConfig(ctx, t)
		if err != nil {
			errs <- err
		}

		config.OIDC = oidcConfig

	case "saml":
		samlConfig, err := getSAMLConfig(ctx, t)
		if err != nil {
			errs <- err
		}

		config.SAML = samlConfig
	}
}
