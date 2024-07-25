package connection

import (
	"context"
	"errors"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/drive/v2"
	"google.golang.org/api/option"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

var (
	ErrBigqueryInitialization = errors.New("bigquery initialization error")
)

type BigQueryClient struct {
	projectsBQ map[string]*bigquery.Client
	bq         *bigquery.Client
	bqGCP      *bigquery.Client
}

func NewBigQuery(ctx context.Context, log *logger.Logging, projects []string) (*BigQueryClient, error) {
	logger := log.Logger(ctx)

	scopes := option.WithScopes(bigquery.Scope, drive.DriveScope)

	bq, err := bigquery.NewClient(ctx, common.ProjectID, scopes)
	if err != nil {
		logger.Errorf("%s: %s", ErrBigqueryInitialization, err)
		return nil, ErrBigqueryInitialization
	}

	bqGCP, err := bigquery.NewClient(ctx, "doitintl-cmp-gcp-data", scopes)
	if err != nil {
		logger.Errorf("%s: %s", ErrBigqueryInitialization, err)
		return nil, ErrBigqueryInitialization
	}

	// Per-project bq clients.
	projectsBQ := make(map[string]*bigquery.Client)

	for _, project := range projects {
		client, err := bigquery.NewClient(ctx, project, scopes)
		if err != nil {
			logger.Errorf("%s: %s", ErrBigqueryInitialization, err)
			return nil, ErrBigqueryInitialization
		}

		projectsBQ[project] = client
	}

	return &BigQueryClient{
		bq:         bq,
		bqGCP:      bqGCP,
		projectsBQ: projectsBQ,
	}, nil
}
