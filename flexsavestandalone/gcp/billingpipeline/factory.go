package billingpipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"github.com/doitintl/http"
)

type service struct {
	loggerProvider logger.Provider
	*connection.Connection
	apiClient *http.Client
}

func NewService(loggerProvider logger.Provider, conn *connection.Connection) ServiceInterface {
	apiService := devAPIService

	if common.Production {
		apiService = prodAPIService
	}

	// FOR LOCAL DEVELOPMENT ONLY
	// apiService = localAPIService

	ctx := context.Background()

	client, err := http.NewClient(ctx, &http.Config{
		BaseURL: apiService,
		Timeout: 360 * time.Second,
	})
	if err != nil {
		panic(err)
	}

	return &service{
		loggerProvider,
		conn,
		client,
	}
}

func (s *service) setBearerToken(ctx context.Context) (context.Context, error) {
	secret, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretAppEngine)
	if err != nil {
		return nil, err
	}

	token, err := common.GetServiceAccountIDToken(ctx, s.apiClient.URL(), secret)
	if err != nil {
		return nil, err
	}

	ctx = http.WithBearerAuth(ctx, &http.BearerAuthContextData{
		Token: fmt.Sprintf("Bearer %s", token.AccessToken),
	})

	return ctx, nil
}
