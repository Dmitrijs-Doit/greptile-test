package service

import (
	"context"

	"cloud.google.com/go/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/bq_utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type PipelineConfig interface {
	GetPipelineConfig(ctx context.Context) (*dataStructures.PipelineConfig, error)
	GetRegionBucket(ctx context.Context, region string) (string, error)
	SetRegionBucket(ctx context.Context, region string, bucketName string) error
	GetBillingTemplateTableSchema(ctx context.Context) (*bigquery.TableMetadata, error)
	GetDestinationTableAndDatasetFormats(ctx context.Context) (string, string, error)
	DeletePipelineConfigDoc(ctx context.Context) error
	CreatePipelineConfigDoc(ctx context.Context, config *dataStructures.PipelineConfig) error
}
type PipelineConfigImpl struct {
	loggerProvider logger.Provider
	*connection.Connection
	config  *dal.PipelineConfigFirestore
	bqUtils *bq_utils.BQ_Utils
}

func NewPipelineConfig(log logger.Provider, conn *connection.Connection) *PipelineConfigImpl {
	return &PipelineConfigImpl{
		log,
		conn,
		dal.NewPipelineConfigWithClient(conn.Firestore),
		bq_utils.NewBQ_UTils(log, conn),
	}
}

func (a *PipelineConfigImpl) GetPipelineConfig(ctx context.Context) (*dataStructures.PipelineConfig, error) {
	return a.config.GetPipelineConfig(ctx)
}

func (d *PipelineConfigImpl) GetRegionBucket(ctx context.Context, region string) (string, error) {
	config, err := d.GetPipelineConfig(ctx)
	if err != nil {
		return "", err
	}

	if config.RegionsBuckets == nil || config.RegionsBuckets[region] == "" {
		return "", common.ErrBucketNotFound
	}

	return config.RegionsBuckets[region], nil
}

func (d *PipelineConfigImpl) SetRegionBucket(ctx context.Context, region string, bucketName string) error {
	config, err := d.GetPipelineConfig(ctx)
	if err != nil {
		return err
	}

	if config.RegionsBuckets == nil {
		config.RegionsBuckets = make(map[string]string)
	}

	config.RegionsBuckets[region] = bucketName

	err = d.config.SetPipelineConfig(ctx, config)
	if err != nil {
		return err
	}

	return nil
}

func (d *PipelineConfigImpl) GetBillingTemplateTableSchema(ctx context.Context) (*bigquery.TableMetadata, error) {
	config, err := d.GetPipelineConfig(ctx)
	if err != nil {
		return nil, err
	}

	bq, err := d.bqUtils.GetBQClientByProjectID(ctx, config.TemplateBillingDataProjectID)
	if err != nil {
		return nil, err
	}

	md, err := bq.Dataset(config.TemplateBillingDataDatasetID).Table(config.TemplateBillingDataTableID).Metadata(ctx)
	if err != nil {
		return nil, err
	}

	return md, nil
}

func (d *PipelineConfigImpl) GetDestinationTableAndDatasetFormats(ctx context.Context) (string, string, error) {
	config, err := d.GetPipelineConfig(ctx)
	if err != nil {
		return "", "", err
	}

	return config.DestinationDatasetFormat, config.DestinationTableFormat, nil
}

func (d *PipelineConfigImpl) DeletePipelineConfigDoc(ctx context.Context) error {
	err := d.config.DeletePipelineConfigDoc(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (d *PipelineConfigImpl) CreatePipelineConfigDoc(ctx context.Context, config *dataStructures.PipelineConfig) error {
	err := d.config.SetPipelineConfig(ctx, config)
	if err != nil {
		return err
	}

	return nil
}
