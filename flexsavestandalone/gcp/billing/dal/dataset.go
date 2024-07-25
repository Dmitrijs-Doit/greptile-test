package dal

import (
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Dataset struct {
	*logger.Logging
	*connection.Connection
}

func NewDataset(log *logger.Logging, conn *connection.Connection) *Dataset {
	return &Dataset{
		log,
		conn,
	}
}

//func (d *Dataset) Create(ctx context.Context, bq *bigquery.Client, datasetName string) error {
//	bq.Dataset(datasetName).Create(ctx)
//}
