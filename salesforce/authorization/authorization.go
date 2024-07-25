package authorization

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"github.com/golang-jwt/jwt"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"github.com/doitintl/http"
)

const (
	grantType      = "urn:ietf:params:oauth:grant-type:jwt-bearer"
	accessTokenURL = "/services/oauth2/token"
)

type authService struct {
	log        *logger.Logging
	httpClient http.IClient
	sfSecret   *Secret
}

type authResponse struct {
	Token       string `json:"access_token"`
	InstanceURL string `json:"instance_url"`
	ID          string `json:"id"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
}

func NewAuthService(log *logger.Logging, client http.IClient) (AuthorizationService, error) {
	ctx := context.Background()
	l := log.Logger(ctx)

	s, err := getSalesforceSecret(ctx)
	if err != nil {
		l.Errorf("Could not read salesforce secret\n Error: %v", err)
		return nil, err
	}

	var c = client

	if common.IsNil(client) {
		c, err = http.NewClient(ctx, &http.Config{
			BaseURL: s.LoginURL,
			Headers: map[string]string{
				"Content-Type": "application/x-www-form-urlencoded",
			},
		})
		if err != nil {
			return nil, err
		}
	}

	return &authService{
		log:        log,
		httpClient: c,
		sfSecret:   s,
	}, nil
}

// GetToken returns the authorization token.
func (response authResponse) GetToken() string { return response.Token }

// GetInstanceURL returns the Salesforce instance URL to use with the authorization information.
func (response authResponse) GetInstanceURL() string { return response.InstanceURL }

// GetID returns the Salesforce ID of the authorization.
func (response authResponse) GetID() string { return response.ID }

// GetTokenType returns the authorization token type.
func (response authResponse) GetTokenType() string { return response.TokenType }

// GetScope returns the scope of the token.
func (response authResponse) GetScope() string { return response.Scope }

// Authenticate will exchange the JWT signed request for access token.
func (h authService) Authenticate(ctx context.Context) (Authorization, error) {
	l := h.log.Logger(ctx)

	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &jwt.StandardClaims{
		ExpiresAt: expirationTime.Unix(),
		Audience:  h.sfSecret.LoginURL,
		Issuer:    h.sfSecret.ConsumerKey,
		Subject:   h.sfSecret.Username,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)

	signKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(h.sfSecret.PrivateKey)) // parse RSA private key

	if err != nil {
		l.Errorf("Could not parse Salesforce private key\n Error: %v", err)
		return nil, err
	}

	tokenString, err := token.SignedString(signKey) // sign claims with RSA private key
	if err != nil {
		return nil, err
	}

	var authResponse authResponse

	queryParams := map[string][]string{
		"assertion":  {tokenString},
		"grant_type": {grantType},
	}

	if _, err := h.httpClient.Post(ctx, &http.Request{
		URL:          accessTokenURL,
		QueryParams:  queryParams,
		ResponseType: &authResponse,
	}); err != nil {
		return nil, err
	}

	if authResponse.InstanceURL != "" && authResponse.InstanceURL != h.sfSecret.InstanceURL {
		return nil, errors.New("token instance URL and secret instance URL don't match")
	}

	return authResponse, nil
}

func getSalesforceSecret(ctx context.Context) (*Secret, error) {
	secretData, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretSalesforce)
	if err != nil {
		return nil, err
	}

	var s Secret

	if err := json.Unmarshal(secretData, &s); err != nil {
		return nil, err
	}

	v := reflect.ValueOf(s)
	typeOfS := v.Type()

	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).Interface() == "" {
			return nil, errors.New("could not find Salesforce " + typeOfS.Field(i).Name)
		}
	}

	return &s, nil
}

func (h authService) GetInstanceURL() string {
	return h.sfSecret.InstanceURL
}
