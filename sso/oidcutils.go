package sso

import (
	"context"
	"fmt"

	"firebase.google.com/go/v4/auth"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"google.golang.org/api/iterator"
)

// getOIDCConfig returns an enabled OIDC Configuration or any disabled one
// at most each tenant has two SSO configs a SAML or an OIDC out of each only one is enabled.
func getOIDCConfig(ctx context.Context, tenantAuth *auth.TenantClient) (*OIDCConfig, error) {
	oidcIter := tenantAuth.OIDCProviderConfigs(ctx, "")

	var disabledOIDC *OIDCConfig

	for {
		oidc, err := oidcIter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		current := &OIDCConfig{
			ID:           oidc.ID,
			Enabled:      oidc.Enabled,
			ClientID:     oidc.ClientID,
			IssuerURL:    oidc.Issuer,
			ClientSecret: oidc.ClientSecret,
			CallbackURL:  getCallBackURL(),
		}

		if oidc.Enabled {
			return current, nil
		}

		disabledOIDC = current
	}

	return disabledOIDC, nil
}

// createOIDCProvider create new OIDC Provider config based on enabled, clientId, clientSecret and IssuerUrl; return callback url
func createOIDCProvider(ctx context.Context, config *OIDCConfig, customer *common.Customer, tenantAuth *auth.TenantClient) (*OIDCConfig, error) {
	if config == nil {
		return nil, nil
	}

	displayName := fmt.Sprintf("%s-%s", "oidc-provider", customer.PrimaryDomain)
	providerID := fmt.Sprintf("%s.%s", OIDC, customer.PrimaryDomain)

	newConfig := (&auth.OIDCProviderConfigToCreate{}).
		ID(providerID).
		Enabled(config.Enabled).
		DisplayName(displayName).
		ClientID(config.ClientID).
		ClientSecret(config.ClientSecret).
		Issuer(config.IssuerURL).
		IDTokenResponseType(false).
		CodeResponseType(true)

	oidc, err := tenantAuth.CreateOIDCProviderConfig(ctx, newConfig)
	if err != nil {
		return nil, err
	}

	return &OIDCConfig{
		ID:           oidc.ID,
		Enabled:      oidc.Enabled,
		ClientID:     oidc.ClientID,
		IssuerURL:    oidc.Issuer,
		ClientSecret: oidc.ClientSecret,
		CallbackURL:  getCallBackURL(),
	}, nil
}

// updateOIDCProvider update an existing OIDC Provider
func updateOIDCProvider(ctx context.Context, config *OIDCConfig, tenantAuth *auth.TenantClient) (*OIDCConfig, error) {
	if config == nil || config.ID == "" {
		return nil, nil
	}

	newConfig := (&auth.OIDCProviderConfigToUpdate{}).Enabled(config.Enabled)

	if config.ClientID != "" {
		newConfig.ClientID(config.ClientID)
	}

	if config.ClientSecret != "" {
		newConfig.ClientSecret(config.ClientSecret)
	}

	if config.IssuerURL != "" {
		newConfig.Issuer(config.IssuerURL)
	}

	oidc, err := tenantAuth.UpdateOIDCProviderConfig(ctx, config.ID, newConfig)
	if err != nil {
		return nil, err
	}

	return &OIDCConfig{
		ID:           oidc.ID,
		Enabled:      oidc.Enabled,
		ClientID:     oidc.ClientID,
		IssuerURL:    oidc.Issuer,
		ClientSecret: oidc.ClientSecret,
		CallbackURL:  getCallBackURL(),
	}, nil
}
