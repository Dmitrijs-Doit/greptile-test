package bq_utils

import (
	"context"
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type BQ_Utils struct {
	loggerProvider logger.Provider
	*connection.Connection
}

func NewBQ_UTils(log logger.Provider, conn *connection.Connection) *BQ_Utils {
	return &BQ_Utils{
		loggerProvider: log,
		Connection:     conn,
	}
}

func (b *BQ_Utils) GetBQClientByProjectID(ctx context.Context, projectID string) (*bigquery.Client, error) {
	if common.Production {
		return b.BigqueryGCP(ctx), nil
	}

	switch projectID {
	case consts.BillingProjectProd:
		return b.BigqueryGCP(ctx), nil
	case consts.BillingProjectMeDoitIntlCom:
		return b.BigqueryGCP(ctx), nil
	case consts.BillingProjectDev:
		return b.Bigquery(ctx), nil
	case consts.BillingProjectOk8topus:
		return b.Bigquery(ctx), nil
	case consts.BillingProjectSkuid:
		return b.Bigquery(ctx), nil
	case consts.BillingProjectCutterfish:
		return b.Bigquery(ctx), nil
	default:
		return nil, fmt.Errorf("invalid project %s", projectID)
	}
}
