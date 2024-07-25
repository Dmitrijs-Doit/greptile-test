package http

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"github.com/doitintl/http"
	"github.com/doitintl/idtoken"
)

const (
	procurementServiceLocalhost = "http://localhost:8080"
	procurementServiceDev       = "https://flexsave-gcp-marketplace-procurement-kaz3welw4q-uc.a.run.app"
	procurementServiceProd      = "https://flexsave-gcp-marketplace-procurement-hkllbpb3gq-uc.a.run.app"
)

func getProcurementServiceURL() string {
	if common.Production {
		return procurementServiceProd
	}

	return procurementServiceDev
}

func NewProcurementClient() (http.IClient, error) {
	ctx := context.Background()
	baseURL := getProcurementServiceURL()

	secret, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretAppEngine)
	if err != nil {
		return nil, err
	}

	tokenSource, err := idtoken.New().GetServiceAccountTokenSource(ctx, baseURL, secret)
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
