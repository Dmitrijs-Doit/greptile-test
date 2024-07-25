package slack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	slackgo "github.com/slack-go/slack"

	sharedFirestore "github.com/doitintl/firestore"
	firestorePkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"github.com/doitintl/hello/scheduled-tasks/slack/domain"
	"github.com/doitintl/hello/scheduled-tasks/slack/service/slack/iface"
	slackapi "github.com/doitintl/slackapi/client"
)

// ParseEventSubscriptionRequest - parse event triggered by DoiT International Slack app (AF79TTA7N)
func (s *SlackService) ParseEventSubscriptionRequest(ctx *gin.Context) ([]byte, *domain.SlackRequest, error) {
	l := s.loggerProvider(ctx)

	body, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		return nil, nil, err
	}

	l.Infof("request from slack %s\n", string(body))

	var req domain.SlackRequest

	if err := json.Unmarshal(body, &req); err != nil {
		return nil, nil, err
	}

	if req.Challenge != "" {
		req.Event.Type = domain.EventChallenge
	}

	return body, &req, nil
}

// GetSlackCredentials - obtain Slack's client_id & client_secret
func (s *SlackService) GetSlackCredentials(ctx context.Context) (string, string, error) {
	data, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretSlackApp)
	if err != nil {
		return "", "", err
	}

	slackSecrets := struct {
		ClientID     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	}{}
	if err := json.Unmarshal(data, &slackSecrets); err != nil {
		return "", "", err
	}

	ClientID, ClientSecret := slackSecrets.ClientID, slackSecrets.ClientSecret
	if ClientID == "" || ClientSecret == "" {
		return "", "", errors.New("could not find Slack secrets")
	}

	return ClientID, ClientSecret, nil
}

// OAuth2callback - installation callback for DoiT International Slack app (AF79TTA7N)
func (s *SlackService) OAuth2callback(ctx *gin.Context, code, customerID string) (string, *domain.MixpanelProperties, error) {
	l := s.loggerProvider(ctx)
	l.Infof("Code accepted from slack request: %s", code)
	l.Infof("CustomerID accepted from slack request: %s", customerID)

	requestFromCMP := customerID != ""
	validCustomer := true

	ClientID, ClientSecret, err := s.GetSlackCredentials(ctx)
	if err != nil {
		l.Error(err)
		return "", nil, err
	}

	res, err := slackgo.GetOAuthV2Response(&http.Client{}, ClientID, ClientSecret, code, "")
	if err != nil {
		return "", nil, err
	}

	if res.SlackResponse.Error != "" {
		err := errors.New(res.SlackResponse.Error)
		l.Error("Slack OAuth2 res: ", err)

		return "", nil, err
	}

	l.Infof("Slack OAuth2 res: %+v", res)

	userTokenEncrypted, err := common.EncryptSymmetric([]byte(res.AuthedUser.AccessToken))
	if err != nil {
		l.Error(err)
		return "", nil, err
	}

	botTokenEncrypted, err := common.EncryptSymmetric([]byte(res.AccessToken))
	if err != nil {
		l.Error(err)
		return "", nil, err
	}

	client, err := slackapi.NewSlackClient(res.AuthedUser.AccessToken)
	if err != nil {
		l.Error(err)
		return "", nil, err
	}

	user, err := client.GetUser(res.AuthedUser.ID)
	if err != nil {
		l.Error(err)
		return "", nil, err
	}

	l.Infof("User ID: %s, Fullname: %s, Email: %s\n", user.ID, user.Profile.RealName, user.Profile.Email)

	if !requestFromCMP {
		customerID, err = s.getCustomerID(ctx, user.Profile.Email)
		if err != nil {
			if !strings.HasPrefix(err.Error(), domain.UserNotFound) {
				l.Error("getCustomerID error: ", err)
				return "", nil, err
			}

			validCustomer = false

			l.Infof("a non doit-customer with email %s has installed Slack app", user.Profile.Email)
		}
	}

	userToken := &firestorePkg.SlackUserToken{
		Email: user.Profile.Email,
		Token: userTokenEncrypted,
	}

	slackWorkspace := &firestorePkg.SlackWorkspace{
		Name:          res.Team.Name,
		BotToken:      botTokenEncrypted,
		UserToken:     userTokenEncrypted,
		Authenticated: validCustomer,
		UsersTokens:   []*firestorePkg.SlackUserToken{userToken},
	}
	redirectURL := fmt.Sprintf("https://%s", common.Domain)

	var mixpanelProperties *domain.MixpanelProperties

	if validCustomer {
		customerRef, _, err := s.firestoreDAL.GetCustomer(ctx, customerID)
		if err != nil {
			if err == sharedFirestore.ErrNotFound {
				return "", nil, fmt.Errorf("cannot authenticate customer " + customerID)
			}

			return "", nil, err
		}

		slackWorkspace.Customer = customerRef
		redirectURL += fmt.Sprintf("/customers/%s/integrations/slack?install=done", customerID)
		mixpanelProperties = &domain.MixpanelProperties{
			Event: domain.MixpanelEventInstallApp,
			Email: user.Profile.Email,
			Payload: map[string]interface{}{
				"Slack Workspace":    res.Team.Name,
				"Slack Workspace ID": res.Team.ID,
			},
		}
	}

	l.Infof("Redirect URL: %s", redirectURL)

	if err := s.firestoreDAL.SetCustomerWorkspace(ctx, res.Team.ID, slackWorkspace); err != nil {
		l.Error("cannot save credentials to fs: ", err)
		return "", nil, err
	}

	l.Infof("Slack workspace credentials was successfully stored on firestore")

	return redirectURL, mixpanelProperties, nil
}

const privateChannelsScopes = "groups:read,users:read,users:read.email"

var (
	ErrInvalidToken = errors.New("invalid token")
)

func (s *SlackService) AuthTest(ctx context.Context, customerID string) (iface.AuthTestResponse, error) {
	_, _, _, botToken, err := s.GetWorkspaceDecrypted(ctx, customerID)
	if err != nil {
		if err == sharedFirestore.ErrNotFound {
			return iface.AuthTestResponse{Ok: false, PrivateChannelsScopes: false}, nil
		}

		return iface.AuthTestResponse{}, err
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	scopes, err := GetTokenScopes(botToken, *client)
	if err != nil {
		if err == ErrInvalidToken {
			return iface.AuthTestResponse{Ok: false, PrivateChannelsScopes: false}, nil
		}

		return iface.AuthTestResponse{}, err
	}

	for _, requiredScope := range strings.Split(privateChannelsScopes, ",") {
		if !strings.Contains(scopes, requiredScope) {
			return iface.AuthTestResponse{Ok: true, PrivateChannelsScopes: false}, nil
		}
	}

	return iface.AuthTestResponse{Ok: true, PrivateChannelsScopes: true}, nil
}

func GetTokenScopes(token string, client http.Client) (string, error) {
	req, err := http.NewRequest("POST", "https://slack.com/api/auth.test", nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()

	// if the request is successful (status code 200), and the header does not exist, the token is invalid
	scopes := res.Header.Get("x-oauth-scopes")
	if scopes == "" {
		return "", ErrInvalidToken
	}

	return scopes, nil
}

// CheckVersionUpdated - check if version installed on a given workspace is up to date
func (s *SlackService) CheckVersionUpdated(ctx context.Context, workspaceID string) error {
	l := s.loggerProvider(ctx)

	var scopesAggregated string

	tokens := make(map[string]string)

	var err error

	_, _, tokens[domain.UserToken], tokens[domain.BotToken], err = s.firestoreDAL.GetWorkspaceDecrypted(ctx, workspaceID)
	if err != nil {
		return err
	}

	l.Infof("checking slack app version installed on workspace: %s", workspaceID)

	client := &http.Client{}

	for tokenType, token := range tokens {
		scopes, err := GetTokenScopes(token, *client)
		if err != nil {
			return err
		}

		l.Infof("%s scopes: %s", tokenType, scopes)
		scopesAggregated += scopes
	}

	if !strings.Contains(scopesAggregated, domain.LatestAppScope) {
		return domain.ErrorAppIsOutdated
	}

	return nil
}

func (s *SlackService) getCustomerID(ctx *gin.Context, email string) (string, error) {
	auth, err := s.tenantService.GetTenantAuthClientByEmail(ctx, email)
	if err != nil {
		return "", err
	}

	user, err := auth.GetUserByEmail(ctx, email)
	if err != nil {
		return "", err
	}

	if doitEmployee, ok := user.CustomClaims[common.DoitEmployee].(bool); ok && doitEmployee {
		return common.DoitCustomerID, nil
	}

	if customerID, ok := user.CustomClaims["customerId"].(string); ok && customerID != "" {
		return customerID, nil
	}

	return "", errors.New("cannot get customer id")
}
