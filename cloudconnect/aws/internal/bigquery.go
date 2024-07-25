package internal

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

type BigQuery struct {
	*bigquery.Client
}

func NewBigQueryClient(ctx context.Context) (*BigQuery, error) {
	client, err := bigquery.NewClient(ctx, common.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("could not initialize bigquery client. error %s", err)
	}

	return &BigQuery{
		client,
	}, nil
}
