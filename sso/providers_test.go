package sso

import (
	"context"
	"testing"

	"firebase.google.com/go/v4/auth"

	"github.com/stretchr/testify/assert"

	"github.com/doitintl/hello/scheduled-tasks/common"
	customerMocks "github.com/doitintl/hello/scheduled-tasks/customer/dal/mocks"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/firebase/tenant"
	tenantMocks "github.com/doitintl/hello/scheduled-tasks/firebase/tenant/mocks"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func setupProviderService(t *testing.T) *ProviderService {
	ctx := context.Background()
	log, err := logger.NewLogging(ctx)

	if err != nil {
		t.Error(err)
	}

	conn, err := connection.NewConnection(ctx, log)

	if err != nil {
		t.Error(err)
	}

	tenantService, err := tenant.NewTenantsServiceWithDalClient(&tenantMocks.Tenants{})
	if err != nil {
		t.Error(err)
	}

	return &ProviderService{log, conn, tenantService, &customerMocks.Customers{}}
}

func TestCreateSAMLProvider(t *testing.T) {
	t.Skip("Skipping as this test is not working as expected.") // TODO: remove this line once the test is fixed

	ctx := context.Background()
	firebaseAuth, _ := fb.App.Auth(ctx)
	config := (&auth.TenantToCreate{}).DisplayName("testTenant")
	newTenant, _ := firebaseAuth.TenantManager.CreateTenant(context.Background(), config)

	defer deleteTenant(ctx, t, firebaseAuth, newTenant)

	s := setupProviderService(t)

	tDal := s.Tenants.(*tenantMocks.Tenants)
	cDal := s.Customers.(*customerMocks.Customers)

	tDal.On("GetTenantIDByCustomer", ctx, "customerId").Return(&newTenant.ID, nil)
	cDal.On("GetCustomer", ctx, "customerId").Return(&common.Customer{
		PrimaryDomain: "testDomain.com",
	}, nil)

	resp, err := s.CreateProvider(ctx, "customerId", &ProviderConfig{
		SAML: &SAMLConfig{
			ID:          "saml.testId",
			Enabled:     true,
			IdpEntityID: "idpEntityId",
			SpEntityID:  "spEntityId",
			SsoURL:      "https://test.com/saml/sso",
			Certificate: "-----BEGIN CERTIFICATE-----\\nCERT1...\\n-----END CERTIFICATE-----",
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, resp)
	assert.Equal(t, true, resp.SAML.Enabled)
	assert.Equal(t, "saml.cmp.testDomain.com", resp.SAML.SpEntityID)
	assert.Equal(t, getCallBackURL(), resp.SAML.CallbackURL)
}

func TestCreateOIDCProvider(t *testing.T) {
	t.Skip("Skipping as this test is not working as expected.") // TODO: remove this line once the test is fixed

	ctx := context.Background()
	firebaseAuth, _ := fb.App.Auth(ctx)
	config := (&auth.TenantToCreate{}).DisplayName("testTenant")
	newTenant, _ := firebaseAuth.TenantManager.CreateTenant(context.Background(), config)

	defer deleteTenant(ctx, t, firebaseAuth, newTenant)

	s := setupProviderService(t)

	tDal := s.Tenants.(*tenantMocks.Tenants)
	cDal := s.Customers.(*customerMocks.Customers)

	tDal.On("GetTenantIDByCustomer", ctx, "customerId").Return(&newTenant.ID, nil)
	cDal.On("GetCustomer", ctx, "customerId").Return(&common.Customer{
		PrimaryDomain: "testDomain.com",
	}, nil)

	resp, err := s.CreateProvider(ctx, "customerId", &ProviderConfig{
		OIDC: &OIDCConfig{
			ID:           "oidc.testId",
			Enabled:      true,
			ClientID:     "clientID",
			IssuerURL:    "https://test.okta.com",
			ClientSecret: "clientSecret",
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, resp)
	assert.Equal(t, "clientID", resp.OIDC.ClientID)
	assert.Equal(t, "https://test.okta.com", resp.OIDC.IssuerURL)
	assert.Equal(t, true, resp.OIDC.Enabled)
}

func TestGetAllProviders(t *testing.T) {
	t.Skip("Skipping as this test is not working as expected.") // TODO: remove this line once the test is fixed

	ctx := context.Background()
	firebaseAuth, _ := fb.App.Auth(ctx)
	config := (&auth.TenantToCreate{}).DisplayName("testTenant")
	newTenant, _ := firebaseAuth.TenantManager.CreateTenant(context.Background(), config)

	defer deleteTenant(ctx, t, firebaseAuth, newTenant)

	s := setupProviderService(t)

	tDal := s.Tenants.(*tenantMocks.Tenants)
	tDal.On("GetTenantIDByCustomer", ctx, "customerId").Return(&newTenant.ID, nil)

	newSAMLConfig := (&auth.SAMLProviderConfigToCreate{}).
		Enabled(true).
		DisplayName("saml-test-DisplayName").
		ID("saml.test").
		RPEntityID("saml-test-spID").
		SSOURL("https://test.jumpcloud.com").
		X509Certificates([]string{
			"-----BEGIN CERTIFICATE-----\\nCERT1...\\n-----END CERTIFICATE-----",
		}).
		IDPEntityID("saml-test-idpID").
		CallbackURL(getCallBackURL())

	newOIDCConfig := (&auth.OIDCProviderConfigToCreate{}).
		Enabled(false).
		DisplayName("oidc-test-DisplayName").
		ID("oidc.test").
		ClientID("oidc-test-client-id").
		ClientSecret("oidc--test-client-secret").
		Issuer("https://test.okta.com")

	tAuth, err := firebaseAuth.TenantManager.AuthForTenant(newTenant.ID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = tAuth.CreateSAMLProviderConfig(ctx, newSAMLConfig)
	if err != nil {
		t.Fatal(err)
	}

	_, err = tAuth.CreateOIDCProviderConfig(ctx, newOIDCConfig)
	if err != nil {
		t.Fatal(err)
	}

	pConfig, err := s.GetAllProviders(ctx, "customerId")
	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, pConfig.OIDC)
	assert.NotNil(t, pConfig.SAML)
}

func TestUpdateSAMLProvider(t *testing.T) {
	t.Skip("Skipping as this test is not working as expected.") // TODO: remove this line once the test is fixed

	ctx := context.Background()
	firebaseAuth, _ := fb.App.Auth(ctx)
	config := (&auth.TenantToCreate{}).DisplayName("testTenant")
	newTenant, _ := firebaseAuth.TenantManager.CreateTenant(context.Background(), config)

	defer deleteTenant(ctx, t, firebaseAuth, newTenant)

	s := setupProviderService(t)

	tDal := s.Tenants.(*tenantMocks.Tenants)
	tDal.On("GetTenantIDByCustomer", ctx, "customerId").Return(&newTenant.ID, nil)

	newConfig := (&auth.SAMLProviderConfigToCreate{}).
		Enabled(true).
		DisplayName("saml-test-DisplayName").
		ID("saml.test").
		RPEntityID("saml-test-spID").
		SSOURL("https://test.jumpcloud.com").
		X509Certificates([]string{
			"-----BEGIN CERTIFICATE-----\\nCERT1...\\n-----END CERTIFICATE-----",
		}).
		IDPEntityID("saml-test-idpID").
		CallbackURL(getCallBackURL())

	tAuth, err := firebaseAuth.TenantManager.AuthForTenant(newTenant.ID)
	if err != nil {
		t.Fatal(err)
	}

	created, err := tAuth.CreateSAMLProviderConfig(ctx, newConfig)
	if err != nil {
		t.Fatal(err)
	}

	pu := &ProviderConfig{
		SAML: &SAMLConfig{
			ID:          created.ID,
			Enabled:     false,
			SsoURL:      "https://testupdate.jumpcloud.com",
			IdpEntityID: "updatedIdpEntityID",
		}}

	pConfig, err := s.UpdateProvider(ctx, "customerId", pu)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "updatedIdpEntityID", pConfig.SAML.IdpEntityID)
	assert.Equal(t, "https://testupdate.jumpcloud.com", pConfig.SAML.SsoURL)
	assert.Equal(t, false, pConfig.SAML.Enabled)
}

func TestUpdateOIDCProvider(t *testing.T) {
	t.Skip("Skipping as this test is not working as expected.") // TODO: remove this line once the test is fixed

	ctx := context.Background()
	firebaseAuth, _ := fb.App.Auth(ctx)
	config := (&auth.TenantToCreate{}).DisplayName("testTenant")
	newTenant, _ := firebaseAuth.TenantManager.CreateTenant(context.Background(), config)

	defer deleteTenant(ctx, t, firebaseAuth, newTenant)

	s := setupProviderService(t)

	tDal := s.Tenants.(*tenantMocks.Tenants)
	tDal.On("GetTenantIDByCustomer", ctx, "customerId").Return(&newTenant.ID, nil)

	newConfig := (&auth.OIDCProviderConfigToCreate{}).
		Enabled(false).
		DisplayName("oidc-test-DisplayName").
		ID("oidc.test").
		ClientID("oidc-test-client-id").
		ClientSecret("oidc--test-client-secret").
		Issuer("https://test.okta.com")

	tAuth, err := firebaseAuth.TenantManager.AuthForTenant(newTenant.ID)
	if err != nil {
		t.Fatal(err)
	}

	created, err := tAuth.CreateOIDCProviderConfig(ctx, newConfig)
	if err != nil {
		t.Fatal(err)
	}

	pu := &ProviderConfig{
		OIDC: &OIDCConfig{
			ID:       created.ID,
			Enabled:  false,
			ClientID: "updated-oidc-client-id",
		}}

	pConfig, err := s.UpdateProvider(ctx, "customerId", pu)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "updated-oidc-client-id", pConfig.OIDC.ClientID)
	assert.Equal(t, false, pConfig.OIDC.Enabled)
}

func deleteTenant(ctx context.Context, t *testing.T, firebaseAuth *auth.Client, newTenant *auth.Tenant) {
	func(TenantManager *auth.TenantManager, ctx context.Context, tenantID string) {
		err := TenantManager.DeleteTenant(ctx, tenantID)

		if err != nil {
			t.Error(err)
		}
	}(firebaseAuth.TenantManager, ctx, newTenant.ID)
}
