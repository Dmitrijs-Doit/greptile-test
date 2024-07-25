package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/bq_utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"google.golang.org/api/googleapi"

	"cloud.google.com/go/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"google.golang.org/api/iterator"
)

type Table interface {
	TableExists(ctx context.Context, bq *bigquery.Client, datasetID, tableID string) (bool, error)
	GetDefaultTemplate(ctx context.Context, tableName string) (md *bigquery.TableMetadata, err error)
	CreateTmpTable(ctx context.Context, iteration int64) error
	CreateUnifiedTable(ctx context.Context) error
	CreateAlternativeUnifiedTable(ctx context.Context) error
	CreateLocalTable(ctx context.Context, billingAccount string) error
	CreateAlternativeLocalTable(ctx context.Context, billingAccount string) error
	DeleteUnifiedTable(ctx context.Context) error
	DeleteTmpTable(ctx context.Context, bq *bigquery.Client, iteration int64) error
	DeleteAlternativeTmpTable(ctx context.Context, bq *bigquery.Client) error
	DeleteLocalTable(ctx context.Context, billingAccount string) error
	DeleteAnyTmpTable(ctx context.Context, bq *bigquery.Client) error
	DeleteAllLocalTables(ctx context.Context, bq *bigquery.Client) error
	CreateAlternativeTmpTable(ctx context.Context) error
	GetTableLocation(ctx context.Context, bq *bigquery.Client, table *dataStructures.BillingTableInfo) (string, error)
	UpdateSchema(ctx context.Context, bq *bigquery.Client, table string) error
}

type TableImpl struct {
	loggerProvider logger.Provider
	*connection.Connection
	config  *dal.PipelineConfigFirestore
	table   *dal.BQTable
	bqUtils *bq_utils.BQ_Utils
}

func NewTable(log logger.Provider, conn *connection.Connection) *TableImpl {
	return &TableImpl{
		log,
		conn,
		dal.NewPipelineConfigWithClient(conn.Firestore),
		dal.NewBQTable(log, conn),
		bq_utils.NewBQ_UTils(log, conn),
	}
}

func (t *TableImpl) TableExists(ctx context.Context, bq *bigquery.Client, datasetID, tableID string) (bool, error) {
	return t.table.Exists(ctx, bq, datasetID, tableID)
}

func (t *TableImpl) mustCreateTable(ctx context.Context, md *bigquery.TableMetadata, datasetName string) error {
	logger := t.loggerProvider(ctx)

	bq, err := t.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		err = fmt.Errorf("unable to GetBQClientByProjectID. Caused by %s", err)
		logger.Error(err)

		return err
	}

	return t.table.MustCreateTable(ctx, bq, datasetName, md)
}

func (t *TableImpl) createTableIfNecessary(ctx context.Context, md *bigquery.TableMetadata, datasetName string) error {
	logger := t.loggerProvider(ctx)

	err := t.mustCreateTable(ctx, md, datasetName)
	if err == common.ErrTableAlreadyExists {
		logger.Infof("skipping creation of table %s.%s.%s. Caused by table not found", utils.GetProjectName(), datasetName, md.Name)
		return nil
	}

	return err
}

func (t *TableImpl) GetDefaultTemplate(ctx context.Context, tableName string) (md *bigquery.TableMetadata, err error) {
	logger := t.loggerProvider(ctx)

	config, err := t.config.GetPipelineConfig(ctx)
	if err != nil {
		err = fmt.Errorf("unable to GetPipelineConfig. Caused by %s", err)
		logger.Error(err)

		return nil, err
	}

	templateProjBQ, err := t.bqUtils.GetBQClientByProjectID(ctx, config.TemplateBillingDataProjectID)
	if err != nil {
		err = fmt.Errorf("unable to GetBQClientByProjectID. Caused by %s", err)
		logger.Error(err)

		return nil, err
	}

	md, err = t.table.GetTableMetadata(ctx, templateProjBQ, config.TemplateBillingDataDatasetID, config.TemplateBillingDataTableID)
	if err != nil {
		err = fmt.Errorf("unable to GetTableMetadata. Caused by %s", err)
		logger.Error(err)

		return nil, err
	}

	removeUnessesaryFieldsFromMetadata(md, tableName)

	return md, nil
}

func (t *TableImpl) CreateTmpTable(ctx context.Context, iteration int64) error {
	logger := t.loggerProvider(ctx)

	md, err := t.GetDefaultTemplate(ctx, utils.GetUnifiedTempTableName(iteration))
	if err != nil {
		err = fmt.Errorf("unable to GetDefaultTemplate. Caused by %s", err)
		logger.Error(err)

		return err
	}

	editTmpMetadataValues(md, utils.GetUnifiedTempTableName(iteration))
	md.Clustering = &bigquery.Clustering{Fields: []string{"billing_account_id"}}

	return t.mustCreateTable(ctx, md, consts.UnifiedGCPBillingDataset)
}

func (t *TableImpl) CreateUnifiedTable(ctx context.Context) error {
	logger := t.loggerProvider(ctx)

	md, err := t.GetDefaultTemplate(ctx, consts.UnifiedGCPRawTable)
	if err != nil {
		err = fmt.Errorf("unable to GetDefaultTemplate. Caused by %s", err)
		logger.Error(err)

		return err
	}

	editTmpMetadataValues(md, consts.UnifiedGCPRawTable)

	md.Schema = append(md.Schema, &bigquery.FieldSchema{
		Name: "doit_export_time",
		Type: bigquery.TimestampFieldType,
	})

	md.Clustering = &bigquery.Clustering{Fields: []string{"billing_account_id"}}

	return t.mustCreateTable(ctx, md, consts.UnifiedGCPBillingDataset)
}

func (t *TableImpl) CreateAlternativeUnifiedTable(ctx context.Context) error {
	logger := t.loggerProvider(ctx)

	md, err := t.GetDefaultTemplate(ctx, consts.UnifiedAlternativeGCPRawTable)
	if err != nil {
		err = fmt.Errorf("unable to GetDefaultTemplate. Caused by %s", err)
		logger.Error(err)

		return err
	}

	editTmpMetadataValues(md, consts.UnifiedAlternativeGCPRawTable)

	md.Schema = append(md.Schema, &bigquery.FieldSchema{
		Name: "doit_export_time",
		Type: bigquery.TimestampFieldType,
	})

	md.Clustering = &bigquery.Clustering{Fields: []string{"billing_account_id"}}

	return t.createTableIfNecessary(ctx, md, consts.AlternativeUnifiedGCPBillingDataset)
}

func (t *TableImpl) CreateLocalTable(ctx context.Context, billingAccount string) error {
	logger := t.loggerProvider(ctx)

	md, err := t.GetDefaultTemplate(ctx, utils.GetLocalCopyAccountTableName(billingAccount))
	if err != nil {
		err = fmt.Errorf("unable to GetDefaultTemplate. Caused by %s", err)
		logger.Error(err)

		return err
	}

	return t.mustCreateTable(ctx, md, consts.LocalBillingDataset)
}

func (t *TableImpl) CreateAlternativeLocalTable(ctx context.Context, billingAccount string) error {
	logger := t.loggerProvider(ctx)

	md, err := t.GetDefaultTemplate(ctx, utils.GetAlternativeLocalCopyAccountTableName(billingAccount))
	if err != nil {
		err = fmt.Errorf("unable to GetDefaultTemplate. Caused by %s", err)
		logger.Error(err)

		return err
	}

	return t.createTableIfNecessary(ctx, md, consts.AlternativeLocalBillingDataset)
}

func (t *TableImpl) deleteTable(ctx context.Context, datasetName, tableName string) error {
	logger := t.loggerProvider(ctx)

	bq, err := t.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		err = fmt.Errorf("unable to GetBQClientByProjectID. Caused by %s", err)
		logger.Error(err)

		return err
	}

	err = t.table.Delete(ctx, bq, datasetName, tableName)
	if err != nil {
		if err == common.ErrTableNotFound {
			logger.Infof("skipping deletion of table %s.%s.%s. Caused by table not found", utils.GetProjectName(), datasetName, tableName)
			return nil
		}
	}

	return err
}

func (t *TableImpl) deleteAllLTablesFromDS(ctx context.Context, bq *bigquery.Client, datasetName, tablePrefix string) error {
	logger := t.loggerProvider(ctx)
	tablesIterator := bq.Dataset(datasetName).Tables(ctx)

	for {
		table, err := tablesIterator.Next()
		if err != nil {
			if gapiErr, ok := err.(*googleapi.Error); ok && gapiErr.Code == http.StatusNotFound {
				t.loggerProvider(ctx).Infof("skipping deletion of tables since dataset %s doesn't exist", datasetName)
				return nil
			} else if err == iterator.Done {
				break
			}

			return err
		}

		if strings.Contains(table.TableID, tablePrefix) {
			err = t.deleteTable(ctx, datasetName, table.TableID)
			if err != nil {
				err = fmt.Errorf("unable to delete table %s.%s.%s. Caused by %s", utils.GetProjectName(), datasetName, table.TableID, err)
				logger.Error(err)

				return err
			}
		}
	}

	return nil
}

func (t *TableImpl) DeleteUnifiedTable(ctx context.Context) error {
	return t.deleteTable(ctx, consts.UnifiedGCPBillingDataset, consts.UnifiedGCPRawTable)
}

func (t *TableImpl) DeleteTmpTable(ctx context.Context, bq *bigquery.Client, iteration int64) error {
	return t.deleteTable(ctx, consts.UnifiedGCPBillingDataset, utils.GetUnifiedTempTableName(iteration))
}

func (t *TableImpl) DeleteAlternativeTmpTable(ctx context.Context, bq *bigquery.Client) error {
	return t.deleteTable(ctx, consts.AlternativeLocalBillingDataset, consts.UnifiedAlternativeRawBillingTable)
}

func (t *TableImpl) DeleteLocalTable(ctx context.Context, billingAccount string) error {
	return t.deleteTable(ctx, consts.LocalBillingDataset, utils.GetLocalCopyAccountTableName(billingAccount))
}

func (t *TableImpl) DeleteAnyTmpTable(ctx context.Context, bq *bigquery.Client) error {
	return t.deleteAllLTablesFromDS(ctx, bq, consts.UnifiedGCPBillingDataset, consts.UnifiedRawBillingTablePrefix)
}

func (t *TableImpl) DeleteAllLocalTables(ctx context.Context, bq *bigquery.Client) error {
	return t.deleteAllLTablesFromDS(ctx, bq, consts.LocalBillingDataset, consts.LocalBillingTablePrefix)
}

func (t *TableImpl) CreateAlternativeTmpTable(ctx context.Context) error {
	logger := t.loggerProvider(ctx)

	md, err := t.GetDefaultTemplate(ctx, consts.UnifiedAlternativeRawBillingTable)
	if err != nil {
		err = fmt.Errorf("unable to GetDefaultTemplate. Caused by %s", err)
		logger.Error(err)

		return err
	}

	editTmpMetadataValues(md, consts.UnifiedAlternativeRawBillingTable)
	md.Clustering = &bigquery.Clustering{Fields: []string{"billing_account_id"}}

	err = t.mustCreateTable(ctx, md, consts.AlternativeLocalBillingDataset)
	if err != nil {
		err = fmt.Errorf("unable to mustCreateTable. Caused by %s", err)
		logger.Error(err)

		return err
	}

	return nil
}

func (t *TableImpl) GetTableLocation(ctx context.Context, bq *bigquery.Client, table *dataStructures.BillingTableInfo) (string, error) {
	return t.table.GetTableLocation(ctx, bq, table)
}

func (t *TableImpl) UpdateSchema(ctx context.Context, bq *bigquery.Client, table string) error {
	return t.table.UpdateSchema(ctx, bq, table)
}

func removeUnessesaryFieldsFromMetadata(md *bigquery.TableMetadata, tableName string) {
	tmpTableMetadata := md
	tmpTableMetadata.Name = tableName
	tmpTableMetadata.FullID = ""
	tmpTableMetadata.Type = ""
	tmpTableMetadata.CreationTime = time.Time{}
	tmpTableMetadata.LastModifiedTime = time.Time{}
	tmpTableMetadata.NumRows = 0
	tmpTableMetadata.NumBytes = 0
	tmpTableMetadata.NumLongTermBytes = 0
	tmpTableMetadata.ETag = ""
	tmpTableMetadata.Clustering = nil
}

func editTmpMetadataValues(md *bigquery.TableMetadata, tableName string) {
	removeUnessesaryFieldsFromMetadata(md, tableName)
	tmpTableMetadata := md
	tmpTableMetadata.Schema = append(tmpTableMetadata.Schema, &bigquery.FieldSchema{
		Name: "iteration",
		Type: bigquery.IntegerFieldType,
	})
	tmpTableMetadata.Schema = append(tmpTableMetadata.Schema, &bigquery.FieldSchema{
		Name: "verified",
		Type: bigquery.BooleanFieldType,
	})
	tmpTableMetadata.Schema = append(tmpTableMetadata.Schema, &bigquery.FieldSchema{
		Name: consts.CustomerTypeField,
		Type: bigquery.StringFieldType,
	})
}
