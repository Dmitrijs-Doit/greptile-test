package microsoft

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	httpClient "github.com/doitintl/http"
)

type AccessToken struct {
	TokenType    string `json:"token_type"`
	Resource     string `json:"resource"`
	NotBefore    string `json:"not_before"`
	ExpiresOn    string `json:"expires_on"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`

	expiresOnUnix int64
	mutex         *sync.Mutex
	config        *SecureApplicationModelConfig
	secret        secretmanager.SecretName
}

type IAccessToken interface {
	Refresh() error
	GetTokenType() string
	GetResource() string
	GetNotBefore() string
	GetExpiresOn() string
	GetAccessToken() string
	GetRefreshToken() string
	GetDomain() CSPDomain
	GetAuthenticatedCtx(ctx context.Context) (context.Context, error)
}

func (a AccessToken) GetTokenType() string {
	return a.TokenType
}

func (a AccessToken) GetResource() string {
	return a.Resource
}

func (a AccessToken) GetNotBefore() string {
	return a.NotBefore
}

func (a AccessToken) GetExpiresOn() string {
	return a.ExpiresOn
}

func (a AccessToken) GetAccessToken() string {
	return a.AccessToken
}

func (a AccessToken) GetRefreshToken() string {
	return a.RefreshToken
}

func (a *AccessToken) GetDomain() CSPDomain {
	return CSPDomain(a.config.Domain)
}

// Refresh uses a refresh token to obtain a new access token if needed
func (a *AccessToken) Refresh() error {
	if a.expiresOnUnix > time.Now().Add(5*time.Minute).Unix() {
		return nil
	}

	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.expiresOnUnix > time.Now().Add(5*time.Minute).Unix() {
		return nil
	}

	client := http.DefaultClient
	urlStr := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/token", a.config.Domain)
	v := url.Values{}
	v.Set("resource", a.config.Resource)
	v.Set("client_id", a.config.ClientID)
	v.Set("client_secret", a.config.ClientSecret)
	v.Set("grant_type", a.config.GrantType)
	v.Set("refresh_token", a.config.RefreshToken)
	reqBody := strings.NewReader(v.Encode())
	req, _ := http.NewRequest("POST", urlStr, reqBody)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	if resp.StatusCode == http.StatusOK {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if err := json.Unmarshal(respBody, a); err != nil {
			return err
		}

		expiresOnUnix, err := strconv.ParseInt(a.ExpiresOn, 10, 64)
		if err != nil {
			return err
		}

		a.expiresOnUnix = expiresOnUnix

		return nil
	}

	return errors.New("microsoft refresh token operation failed")
}

func (a *AccessToken) GetAuthenticatedCtx(ctx context.Context) (context.Context, error) {
	if err := a.Refresh(); err != nil {
		return nil, err
	}

	return httpClient.WithBearerAuth(ctx, &httpClient.BearerAuthContextData{
		Token: fmt.Sprintf("%s %s", a.TokenType, a.AccessToken),
	}), nil
}
