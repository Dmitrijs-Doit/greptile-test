package dal

import (
	"context"
	"net/http"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/googleapi"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type BQTable struct {
	loggerProvider logger.Provider
	*connection.Connection
}

func NewBQTable(log logger.Provider, conn *connection.Connection) *BQTable {
	return &BQTable{
		log,
		conn,
	}
}

func (t *BQTable) datasetExists(ctx context.Context, bqClient *bigquery.Client, datasetID string) (bool, error) {
	if _, err := bqClient.Dataset(datasetID).Metadata(ctx); err != nil {
		if gapiErr, ok := err.(*googleapi.Error); ok {
			if gapiErr.Code == http.StatusNotFound {
				return false, common.ErrDatasetNotFound
			}
		}

		return false, err
	}

	return true, nil
}

func (t *BQTable) Exists(ctx context.Context, bqClient *bigquery.Client, datasetID, tableID string) (bool, error) {
	exists, err := t.datasetExists(ctx, bqClient, datasetID)
	if !exists || err != nil {
		return false, err
	}

	if _, err = bqClient.Dataset(datasetID).Table(tableID).Metadata(ctx); err != nil {
		if gapiErr, ok := err.(*googleapi.Error); ok {
			if gapiErr.Code == http.StatusNotFound {
				return false, common.ErrTableNotFound
			}
		}

		return false, err
	}

	return true, nil
}

func (t *BQTable) MustCreateTable(ctx context.Context, bqClient *bigquery.Client, datasetID string, md *bigquery.TableMetadata) error {
	exists, err := t.datasetExists(ctx, bqClient, datasetID)
	if !exists || err != nil {
		return err
	}

	err = bqClient.Dataset(datasetID).Table(md.Name).Create(ctx, md)
	if gapiErr, ok := err.(*googleapi.Error); ok {
		if gapiErr.Code == http.StatusConflict {
			return common.ErrTableAlreadyExists
		}
	}

	return err
}

func (t *BQTable) Delete(ctx context.Context, bqClient *bigquery.Client, datasetID, tableID string) error {
	err := bqClient.Dataset(datasetID).Table(tableID).Delete(ctx)
	if err != nil {
		if gapiErr, ok := err.(*googleapi.Error); ok {
			if gapiErr.Code == http.StatusNotFound {
				return common.ErrTableNotFound
			}
		}

		return err
	}

	return nil
}

func (t *BQTable) GetTableMetadata(ctx context.Context, bqClient *bigquery.Client, datasetID, tableID string) (*bigquery.TableMetadata, error) {
	exists, err := t.datasetExists(ctx, bqClient, datasetID)
	if !exists || err != nil {
		return nil, err
	}

	return bqClient.Dataset(datasetID).Table(tableID).Metadata(ctx)
}

func (t *BQTable) GetTableLocation(ctx context.Context, bqClient *bigquery.Client, table *dataStructures.BillingTableInfo) (string, error) {
	md, err := t.GetTableMetadata(ctx, bqClient, table.DatasetID, table.TableID)
	if err != nil {
		return "", err
	}
	//gcp_billing_export_v1_016E7D_E1049B_D9D923
	return md.Location, err
}

func (t *BQTable) UpdateSchema(ctx context.Context, bqClient *bigquery.Client, table string) error {
	logger := t.loggerProvider(ctx)

	tableRef := bqClient.Dataset(consts.LocalBillingDataset).Table(table)

	md, err := tableRef.Metadata(ctx)
	if err != nil {
		logger.Errorf("unable to get the MD. Caused by %s", err)
		return err
	}

	for _, field := range md.Schema {
		if field.Name == "tags" {
			field.Schema = append(field.Schema, &bigquery.FieldSchema{Name: "namespace", Type: bigquery.StringFieldType})
		}
	}

	updatedMD := bigquery.TableMetadataToUpdate{
		Schema: md.Schema,
	}

	_, err = tableRef.Update(ctx, updatedMD, md.ETag)
	if err != nil {
		logger.Errorf("unable to update the MD. Caused by %s", err)
		return err
	}

	return nil
}

//func (t *Table) CreateTmpTable(ctx context.Context, iteration uint64) {
//	//data := shared.BigQueryTableUpdateRequest{
//	//	DefaultProjectID:     consts.BillingProjectProd,
//	//	DefaultDatasetID:     consts.FlexSaveSA_BillingDatasetAggregated,
//	//	DestinationProjectID: consts.BillingProjectProd,
//	//	DestinationDatasetID: consts.FlexSaveSA_BillingDatasetAggregated,
//	//	DestinationTableName: templates.CreateTmpTableName(iteration),
//	//	AllPartitions:        true,
//	//	WriteDisposition:     bigquery.WriteAppend,
//	//	Clustering:           &bigquery.Clustering{Fields: []string{"billing_account_id"}},
//	//}
//
//	tableMetadata := bigquery.TableMetadata{
//		Name:                   templates.CreateTmpTableName(iteration),
//		Location:               consts.Location,
//		Description:            consts.TmpTableDescription,
//		Schema:                 nil,
//		MaterializedView:       nil,
//		ViewQuery:              "",
//		UseLegacySQL:           false,
//		UseStandardSQL:         false,
//		TimePartitioning:       nil,
//		RangePartitioning:      nil,
//		RequirePartitionFilter: false,
//		Clustering:             &bigquery.Clustering{Fields: []string{"billing_account_id"}},
//		ExpirationTime:         time.Time{},
//		Labels:                 nil,
//		ExternalDataConfig:     nil,
//		EncryptionConfig:       nil,
//		FullID:                 "",
//		Type:                   "",
//		CreationTime:           time.Time{},
//		LastModifiedTime:       time.Time{},
//		NumBytes:               0,
//		NumLongTermBytes:       0,
//		NumRows:                0,
//		SnapshotDefinition:     nil,
//		StreamingBuffer:        nil,
//		ETag:                   "",
//	}
//}
