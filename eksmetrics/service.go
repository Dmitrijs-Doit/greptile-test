package eksmetrics

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/http"
	"github.com/doitintl/idtoken"
)

const (
	localEksMetricsAPIService = "http://localhost:8086"
	devEksMetricsAPIService   = "https://cmp-aws-k8s-metrics-service-dev-wsqwprteya-uc.a.run.app"
	prodEksMetricsAPIService  = "https://cmp-aws-k8s-metrics-service-alqysnpjoq-uc.a.run.app"
)

type EKSMetricsService struct {
	loggerProvider  logger.Provider
	firestoreClient *firestore.Client
	apiClient       *http.Client
}

func NewEKSMetricsService(log logger.Provider, conn *connection.Connection) (*EKSMetricsService, error) {
	eksMetricsAPIService := devEksMetricsAPIService
	if common.Production {
		eksMetricsAPIService = prodEksMetricsAPIService
	} else if common.IsLocalhost {
		eksMetricsAPIService = localEksMetricsAPIService
	}

	if eksMetricsAPIService == "" {
		return nil, errors.New("empty eksMetricsAPIService url provided")
	}

	ctx := context.Background()

	tokenSource, err := idtoken.New().GetTokenSource(ctx, eksMetricsAPIService)
	if err != nil {
		return nil, err
	}

	apiClient, err := http.NewClient(ctx, &http.Config{
		BaseURL:     eksMetricsAPIService,
		Timeout:     360 * time.Second,
		TokenSource: tokenSource,
	})
	if err != nil {
		panic(err)
	}

	return &EKSMetricsService{
		log,
		conn.Firestore(ctx),
		apiClient,
	}, nil
}
