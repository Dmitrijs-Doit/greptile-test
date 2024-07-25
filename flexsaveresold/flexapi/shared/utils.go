package shared

import (
	"context"

	"golang.org/x/oauth2"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
)

func GetTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	baseURL := GetFlexAPIURL()

	secret, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretAppEngine)
	if err != nil {
		return nil, err
	}

	tokenSource, err := common.GetServiceAccountTokenSource(ctx, baseURL, secret)
	if err != nil {
		return nil, err
	}

	return tokenSource, nil
}

func GetFlexAPIURL() string {
	if common.Production {
		return flexAPIProd
	}

	return flexAPIDev
}
