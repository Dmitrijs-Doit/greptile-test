package http

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/http"
	"github.com/doitintl/idtoken"
)

const (
	integrationServiceLocalhost = "http://localhost:8085"
	integrationServiceDev       = "https://aws-marketplace-integration-service-yx3ainsm7q-uc.a.run.app"
	integrationServiceProd      = "https://aws-marketplace-integration-service-6owo5qhrqq-uc.a.run.app"
)

func getIntegrationServiceURL() string {
	if common.IsLocalhost {
		return integrationServiceLocalhost
	}

	if common.Production {
		return integrationServiceProd
	}

	return integrationServiceDev
}

func NewIntegrationServiceClient() (http.IClient, error) {
	ctx := context.Background()
	baseURL := getIntegrationServiceURL()

	tokenSource, err := idtoken.New().GetTokenSource(ctx, baseURL)
	if err != nil {
		return nil, err
	}

	client, err := http.NewClient(ctx, &http.Config{
		BaseURL:     baseURL,
		TokenSource: tokenSource,
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}
