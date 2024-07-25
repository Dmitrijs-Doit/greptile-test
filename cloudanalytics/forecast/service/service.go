package forecasts

import (
	"context"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/forecast/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/http"
	"github.com/doitintl/idtoken"
)

const (
	productionHost  = "https://cmp-forecast-api-alqysnpjoq-uc.a.run.app"
	developmentHost = "https://cmp-forecast-api-wsqwprteya-uc.a.run.app"
)

type Service struct {
	loggerProvider logger.Provider
	httpClient     *http.Client
}

func getServiceHost() string {
	if common.Production {
		return productionHost
	}

	return developmentHost
}

func NewService(loggerProvider logger.Provider) (iface.Service, error) {
	ctx := context.Background()

	serviceURL := getServiceHost()

	tokenSource, err := idtoken.New().GetTokenSource(ctx, serviceURL)
	if err != nil {
		return nil, err
	}

	httpClient, err := http.NewClient(ctx, &http.Config{
		BaseURL: serviceURL,
		Headers: map[string]string{
			"Accept":       "application/json",
			"Content-Type": "application/json",
		},
		TokenSource: tokenSource,
		Timeout:     30 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	return &Service{
		loggerProvider,
		httpClient,
	}, nil
}
