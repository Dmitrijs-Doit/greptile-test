package sso

import (
	"context"
	"unicode"
	"fmt"

	"firebase.google.com/go/v4/auth"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"google.golang.org/api/iterator"
)

// getSAMLConfig returns an enabled SAML Configuration or any disabled one
// at most each tenant has two SSO configs a SAML or an OIDC out of each only one is enabled.
func getSAMLConfig(ctx context.Context, tenantAuth *auth.TenantClient) (*SAMLConfig, error) {
	samlIter := tenantAuth.SAMLProviderConfigs(ctx, "")

	var disabledSAML *SAMLConfig

	for {
		saml, err := samlIter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		current := &SAMLConfig{
			ID:          saml.ID,
			Enabled:     saml.Enabled,
			SpEntityID:  saml.RPEntityID,
			SsoURL:      saml.SSOURL,
			Certificate: saml.X509Certificates[0],
			IdpEntityID: saml.IDPEntityID,
			CallbackURL: saml.CallbackURL,
		}

		if saml.Enabled {
			return current, nil
		}

		disabledSAML = current
	}

	return disabledSAML, nil
}

// createSAMLProvider create new SAML Provider config based on idpEntityId, enabled, ssoURL and certificate; return callback url and service provider id
func createSAMLProvider(ctx context.Context, config *SAMLConfig, customer *common.Customer, tAuth *auth.TenantClient) (*SAMLConfig, error) {
	if config == nil {
		return nil, nil
	}

	displayName := fmt.Sprintf("%s.%s", "saml.cmp", customer.PrimaryDomain)
	providerID := fmt.Sprintf("%s.%s", SAML, customer.PrimaryDomain)
	firstChar := rune(customer.PrimaryDomain[0])
	// google identity provider saml_config_id must start with saml. followed by a lower case letter
	if !unicode.IsLower(firstChar){
		providerID = fmt.Sprintf("%s.%s%s", SAML, "a" ,customer.PrimaryDomain)
	}

	newConfig := (&auth.SAMLProviderConfigToCreate{}).
		Enabled(config.Enabled).
		DisplayName(displayName).
		ID(providerID).
		RPEntityID(displayName).
		SSOURL(config.SsoURL).
		X509Certificates([]string{
			config.Certificate,
		}).
		IDPEntityID(config.IdpEntityID).
		CallbackURL(getCallBackURL())

	saml, err := tAuth.CreateSAMLProviderConfig(ctx, newConfig)
	if err != nil {
		return nil, err
	}

	return &SAMLConfig{
		ID:          saml.ID,
		Enabled:     saml.Enabled,
		SpEntityID:  saml.RPEntityID,
		SsoURL:      saml.SSOURL,
		Certificate: saml.X509Certificates[0],
		IdpEntityID: saml.IDPEntityID,
		CallbackURL: saml.CallbackURL,
	}, nil
}

// updateSAMLProvider update an existing SAML Provider
func updateSAMLProvider(ctx context.Context, config *SAMLConfig, tenantAuth *auth.TenantClient) (*SAMLConfig, error) {
	if config == nil || config.ID == "" {
		return nil, nil
	}

	newConfig := (&auth.SAMLProviderConfigToUpdate{}).Enabled(config.Enabled)

	if config.Certificate != "" {
		newConfig.X509Certificates([]string{
			config.Certificate,
		})
	}

	if config.IdpEntityID != "" {
		newConfig.IDPEntityID(config.IdpEntityID)
	}

	if config.SsoURL != "" {
		newConfig.SSOURL(config.SsoURL)
	}

	saml, err := tenantAuth.UpdateSAMLProviderConfig(ctx, config.ID, newConfig)
	if err != nil {
		return nil, err
	}

	return &SAMLConfig{
		ID:          saml.ID,
		Enabled:     saml.Enabled,
		SpEntityID:  saml.RPEntityID,
		SsoURL:      saml.SSOURL,
		Certificate: saml.X509Certificates[0],
		IdpEntityID: saml.IDPEntityID,
		CallbackURL: saml.CallbackURL,
	}, nil
}
