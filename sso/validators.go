package sso

import (
	"errors"
	"strings"
)

func ValidateOIDCConfigOnCreate(oidc *OIDCConfig) error {
	if oidc == nil {
		return nil
	}

	if oidc.ClientID == "" {
		return errors.New("client id can not be empty")
	}

	if oidc.IssuerURL == "" {
		return errors.New("issuer url can not be empty")
	}

	if oidc.ClientSecret == "" {
		return errors.New("client secret can not be empty")
	}

	return nil
}

func ValidateSAMLConfigOnCreate(saml *SAMLConfig) error {
	if saml == nil {
		return nil
	}

	if saml.IdpEntityID == "" {
		return errors.New("identity provider id can not be empty")
	}

	if saml.SsoURL == "" {
		return errors.New("SSO Url can not be empty")
	}

	if saml.Certificate == "" {
		return errors.New("certificate can not be empty")
	}

	err := validateSAMLCertificate(saml)
	if err != nil {
		return err
	}

	err = validateSAMLSSOUrl(saml)
	if err != nil {
		return err
	}

	return nil
}

func ValidateSAMLConfigOnUpdate(saml *SAMLConfig) error {
	if saml == nil {
		return nil
	}

	err := validateSAMLCertificate(saml)
	if err != nil {
		return err
	}

	err = validateSAMLSSOUrl(saml)
	if err != nil {
		return err
	}

	return nil
}

func validateSAMLCertificate(saml *SAMLConfig) error {
	if saml.Certificate == "" {
		return nil
	}

	prefix := strings.HasPrefix(saml.Certificate, "-----BEGIN CERTIFICATE-----")

	if !prefix {
		return errors.New("certificate must start with -----BEGIN CERTIFICATE-----")
	}

	suffix := strings.HasSuffix(saml.Certificate, "-----END CERTIFICATE-----")

	if !suffix {
		return errors.New("certificate must end with -----END CERTIFICATE-----")
	}

	return nil
}

func validateSAMLSSOUrl(saml *SAMLConfig) error {
	if saml.SsoURL == "" {
		return nil
	}

	httpsPrefix := strings.HasPrefix(saml.SsoURL, "https://")

	if !httpsPrefix {
		return errors.New("SSO URL must use HTTPS")
	}

	return nil
}
