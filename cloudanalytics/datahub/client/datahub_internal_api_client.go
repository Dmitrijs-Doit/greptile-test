package client

import (
	"context"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/http"
	"github.com/doitintl/idtoken"
)

const (
	datahubInternalServiceLocalhost = "http://localhost:8080"
	datahubInternalServiceDev       = "https://ingestion-api-frontend-wsqwprteya-uc.a.run.app"
	datahubInternalServiceProd      = "https://ingestion-api-frontend-alqysnpjoq-uc.a.run.app"

	contentType     string = "Content-Type"
	applicationJSON string = "application/json"
)

func getDatahubInternalAPIServiceURL() string {
	if common.Production {
		return datahubInternalServiceProd
	}

	return datahubInternalServiceDev
}

func NewDatahubInternalAPIClient() (http.IClient, error) {
	ctx := context.Background()
	baseURL := getDatahubInternalAPIServiceURL()

	client, err := getClient(ctx, baseURL)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func getClient(ctx context.Context, baseURL string) (http.IClient, error) {
	tokenSource, err := idtoken.New().GetTokenSource(ctx, baseURL)
	if err != nil {
		return nil, err
	}

	client, err := http.NewClient(ctx, &http.Config{
		BaseURL:     baseURL,
		Timeout:     3 * time.Minute,
		TokenSource: tokenSource,
		Headers: map[string]string{
			contentType: applicationJSON,
		},
	})

	return client, err
}
