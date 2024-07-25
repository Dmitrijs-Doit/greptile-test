package service

import (
	"context"
	"net/http"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/googleapi"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/bq_utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Dataset struct {
	Logger logger.Provider
	*connection.Connection
	bqUtils *bq_utils.BQ_Utils
}

func NewDataset(loggerProvider logger.Provider, conn *connection.Connection) *Dataset {
	return &Dataset{
		loggerProvider,
		conn,
		bq_utils.NewBQ_UTils(loggerProvider, conn),
	}
}

func (d *Dataset) CreateLocalDataset(ctx context.Context) (err error) {
	return d.createDataset(ctx, consts.LocalBillingDataset)
}

func (d *Dataset) CreateAlternativeLocalDataset(ctx context.Context) (err error) {
	return d.createDataset(ctx, consts.AlternativeLocalBillingDataset)
}

func (d *Dataset) CreateUnifiedDataset(ctx context.Context) (err error) {
	return d.createDataset(ctx, consts.UnifiedGCPBillingDataset)
}

func (d *Dataset) CreateAlternativeUnifiedDataset(ctx context.Context) (err error) {
	return d.createDataset(ctx, consts.AlternativeUnifiedGCPBillingDataset)
}

func (d *Dataset) createDataset(ctx context.Context, datasetName string) (err error) {
	logger := d.Logger(ctx)
	logger.Infof("creating dataset %s", datasetName)

	bq, err := d.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		//TODO handle error
		return err
	}

	err = bq.Dataset(datasetName).Create(ctx, &bigquery.DatasetMetadata{
		Name:     datasetName,
		Location: consts.DoitLocation,
	})
	if err != nil {
		if gapiErr, ok := err.(*googleapi.Error); ok {
			if gapiErr.Code == http.StatusConflict {
				logger.Infof("skipping creating of dataset %s. Dataset already exists", datasetName)
				return nil
			}
		} else {
			logger.Errorf("unable to create dataset %s. Caused by %s", datasetName, err)
			return err
		}
	}

	return nil
}
