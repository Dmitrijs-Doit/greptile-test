package sso

import (
	"github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/firebase/tenant"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type ConfigType string

type ProviderConfig struct {
	SAML *SAMLConfig `json:"saml"`
	OIDC *OIDCConfig `json:"oidc"`
}

type SAMLConfig struct {
	ID          string `json:"id"`
	Enabled     bool   `json:"enabled"`
	SpEntityID  string `json:"spEntityId"`
	SsoURL      string `json:"ssoUrl"`
	Certificate string `json:"certificate"`
	IdpEntityID string `json:"idpEntityId"`
	CallbackURL string `json:"callbackUrl"`
}

type OIDCConfig struct {
	ID           string `json:"id"`
	Enabled      bool   `json:"enabled"`
	ClientID     string `json:"clientId"`
	IssuerURL    string `json:"issuerUrl"`
	ClientSecret string `json:"clientSecret"`
	CallbackURL  string `json:"callbackUrl"`
}

type ProviderService struct {
	*logger.Logging
	*connection.Connection
	*tenant.TenantService
	dal.Customers
}
