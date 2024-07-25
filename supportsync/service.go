package supportsync

import (
	"context"

	"github.com/doitintl/gcs"
	appDal "github.com/doitintl/hello/scheduled-tasks/app/dal"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/http"
)

type SupportSyncService struct {
	loggerProvider logger.Provider
	appDAL         appDal.App
	gcsClient      gcs.GCSClient
	httpClient     *http.Client
}

func NewSupportSyncService(log logger.Provider, firestoreFun connection.FirestoreFromContextFun, gcsClient gcs.GCSClient) (*SupportSyncService, error) {
	ctx := context.Background()

	httpClient, err := http.NewClient(ctx, &http.Config{
		BaseURL: gcsBaseURL,
	})
	if err != nil {
		return nil, err
	}

	return &SupportSyncService{
		log,
		appDal.NewAppFirestoreWithClient(firestoreFun),
		gcsClient,
		httpClient,
	}, nil
}
