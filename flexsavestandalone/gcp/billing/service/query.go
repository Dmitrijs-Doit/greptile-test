package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"

	googleCloudConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/bq_utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type TableQuery interface {
	DeleteRowsFromUnifiedByBA(ctx context.Context, billingAccount string) (job *bigquery.Job, err error)
	TruncateUnifiedTableContent(ctx context.Context) (err error)
	GetLocalTableOldestRecordTime(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (startingPoint time.Time, err error)
	GetUnifiedTableOldestRecordByBA(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (startingPoint time.Time, err error)
	GetUnifiedTableNewestRecordByBA(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (finishingPoint time.Time, err error)
	GetCustomersTableOldestRecordTime(ctx context.Context, customerBQ *bigquery.Client, t *dataStructures.BillingTableInfo) (startingPoint time.Time, err error)
	GetCustomersTableNewestRecordTime(ctx context.Context, customerBQ *bigquery.Client, t *dataStructures.BillingTableInfo) (finishingPoint time.Time, err error)
	GetCustomersTableOldestRecordTimeNewerThan(ctx context.Context, customerBQ *bigquery.Client, t *dataStructures.BillingTableInfo, minExportTime time.Time) (oldestPoint time.Time, err error)
	GetUnifiedTableOldestRecordTimeNewerThan(ctx context.Context, minExportTime time.Time) (oldestPoint time.Time, err error)
	GetRawBillingNewestRecordTime(ctx context.Context) (finishingPoint time.Time, err error)
	GetRawBillingOldestRecordTime(ctx context.Context) (startingPoint time.Time, err error)
	GetLocalTableNewestRecordTime(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (finishingPoint time.Time, err error)
	GetFromUnifiedTableRowsCountPerTimeRange(ctx context.Context, billingAccount string, startTime, endTime *time.Time) (rowsCount int64, err error)
	GetLocalRowsCountPerTimeRange(ctx context.Context, billingAccount string, startTime, endTime *time.Time) (rowsCount int64, err error)
	CopyFromTmpTableAllRows(ctx context.Context, imm *dataStructures.InternalManagerMetadata, itms []*dataStructures.InternalTaskMetadata) (job *bigquery.Job, err error)
	CopyFromLocalToTmpTable(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (job *bigquery.Job, err error)
	CopyFromAlternativeLocalToTmpTable(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (job *bigquery.Job, err error)
	MarkTmpTableBillingRowsAsVerified(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (job *bigquery.Job, err error)
	GetLocalTableLatestExportTime(ctx context.Context, bq *bigquery.Client, billingAccountID string) (*time.Time, error)
	GetCustomerRowsCount(ctx context.Context, billingAccountID string, segment *dataStructures.Segment) (map[dataStructures.HashableSegment]int, error)
	GetCustomerRowsCountByExportTime(ctx context.Context, billingAccountID string, segment *dataStructures.Segment) (map[string]int64, error)
	GetLocalRowsCount(ctx context.Context, billingAccountID string, segment *dataStructures.Segment) (map[dataStructures.HashableSegment]int, error)
	GetLocalRowsCountByExportTime(ctx context.Context, billingAccountID string, segment *dataStructures.Segment) (map[string]int64, error)
	GetLUnifiedRowsCount(ctx context.Context, billingAccountID string, segment *dataStructures.Segment) (map[dataStructures.HashableSegment]int, error)
	GetLUnifiedRowsCountByExportTime(ctx context.Context, billingAccountID string, segment *dataStructures.Segment) (map[string]int64, error)
	RunDetailedTableRewritesMapping(ctx context.Context) error
	RunDataFreshnessReport(ctx context.Context) error
	GetCustomerRowsCountPerTimeRange(ctx context.Context, billingAccountID string, startTime, endTime *time.Time) (int64, error)
}

type TableQueryImpl struct {
	loggerProvider logger.Provider
	*connection.Connection
	bqQuery          *dal.Query
	metadata         *dal.Metadata
	bqUtils          *bq_utils.BQ_Utils
	customerBQClient ExternalBigQueryClient
}

func NewTableQuery(log logger.Provider, conn *connection.Connection) *TableQueryImpl {
	return &TableQueryImpl{
		log,
		conn,
		dal.NewQuery(log, conn),
		dal.NewMetadata(log, conn),
		bq_utils.NewBQ_UTils(log, conn),
		NewExternalBigQueryClient(log, conn),
	}
}

func (s *TableQueryImpl) DeleteRowsFromUnifiedByBA(ctx context.Context, billingAccount string) (job *bigquery.Job, err error) {
	query := utils.GetDeleteRowsFromUnifiedByBA(billingAccount)
	bq, err := s.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())

	if err != nil {
		return nil, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	job, err = s.bqQuery.ExecQueryAsync(ctx, bq, &common.BQExecuteQueryData{
		Query: query,
		DefaultTable: &dataStructures.BillingTableInfo{
			ProjectID: utils.GetProjectName(),
			TableID:   utils.GetUnifiedTableFullName(),
			DatasetID: consts.UnifiedGCPBillingDataset,
		},
		WriteDisposition: bigquery.WriteAppend,
		WaitTillDone:     false,
		Internal:         false,
		Export:           true,
		ConfigJobID:      utils.GetDeleteRowsOfBA(billingAccount),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	return job, err
}

func (s *TableQueryImpl) TruncateUnifiedTableContent(ctx context.Context) (err error) {
	logger := s.loggerProvider(ctx)

	prodBQClient, err := s.bqUtils.GetBQClientByProjectID(ctx, consts.BillingProjectProd)
	if err != nil {
		err = fmt.Errorf("unable to GetBQClientByProjectID. Caused by %s", err)
		logger.Error(err)

		return err
	}

	bqClient, err := s.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		err = fmt.Errorf("unable to GetBQClientByProjectID. Caused by %s", err)
		logger.Error(err)

		return err
	}

	copier := bqClient.Dataset(consts.UnifiedGCPBillingDataset).Table(consts.UnifiedGCPRawTable).
		CopierFrom(prodBQClient.Dataset(consts.UnifiedGCPBillingDataset).Table(consts.UnifiedGCPRawTableTemplate))
	copier.WriteDisposition = bigquery.WriteTruncate
	copier.CreateDisposition = bigquery.CreateIfNeeded

	job, err := copier.Run(ctx)
	if err != nil {
		err = fmt.Errorf("unable to start bqjob. Caused by %s", err)
		logger.Error(err)

		return err
	}

	status, err := job.Wait(ctx)
	if err != nil {
		err = fmt.Errorf("unable to wait until bqjob is done. Caused by %s", err)
		logger.Error(err)

		return err
	}

	if err := status.Err(); err != nil {
		err = fmt.Errorf("bqjob execution failure. Caused by %s", err)
		logger.Error(err)

		return err
	}

	return nil
}

func (s *TableQueryImpl) GetLocalTableOldestRecordTime(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (startingPoint time.Time, err error) {
	query, err := utils.GetTableOldestRecordTimeQuery(itm)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	bq, err := s.bqUtils.GetBQClientByProjectID(ctx, itm.BQTable.ProjectID)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	md, err := bq.Dataset(itm.BQTable.DatasetID).Table(itm.BQTable.TableID).Metadata(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	if md.NumRows == 0 {
		return time.Time{}, fmt.Errorf("unable to execute query %s. Caused by empty table", query)
	}

	it, err := s.bqQuery.ReadQueryResult(ctx, bq, query)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	var row dataStructures.ExportTimeRow
	err = it.Next(&row)

	if err != nil && err != iterator.Done {
		return time.Time{}, fmt.Errorf("error iterating through results of query %s. Caused by %s", query, err.Error())
	}

	return row.Export_time, nil
}

func (s *TableQueryImpl) GetUnifiedTableOldestRecordByBA(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (startingPoint time.Time, err error) {
	query := utils.GetUnifiedTableOldestRecordByBA(itm)
	return s.getUnifiedTableExtremeRecordByBA(ctx, itm, query)
}

func (s *TableQueryImpl) GetUnifiedTableNewestRecordByBA(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (finishingPoint time.Time, err error) {
	query := utils.GetUnifiedTableNewestRecordByBA(itm)
	return s.getUnifiedTableExtremeRecordByBA(ctx, itm, query)
}

func (s *TableQueryImpl) getUnifiedTableExtremeRecordByBA(ctx context.Context, itm *dataStructures.InternalTaskMetadata, query string) (finishingPoint time.Time, err error) {
	bq, err := s.bqUtils.GetBQClientByProjectID(ctx, itm.BQTable.ProjectID)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	it, err := s.bqQuery.ReadQueryResult(ctx, bq, query)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	var row dataStructures.ExportTimeRow
	err = it.Next(&row)

	if err != nil && err != iterator.Done {
		return time.Time{}, fmt.Errorf("error iterating through results of query %s. Caused by %s", query, err.Error())
	}

	return row.Export_time, nil
}

func (s *TableQueryImpl) GetCustomersTableOldestRecordTime(ctx context.Context, customerBQ *bigquery.Client, t *dataStructures.BillingTableInfo) (startingPoint time.Time, err error) {
	query, err := utils.GetCustomerTableOldestRecordTimeQuery(t)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	md, err := customerBQ.Dataset(t.DatasetID).Table(t.TableID).Metadata(ctx)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	if md.NumRows == 0 {
		return time.Time{}, common.NewEmptyBillingTableError(t.TableID)
	}

	it, err := s.bqQuery.ReadQueryResult(ctx, customerBQ, query)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	var row dataStructures.ExportTimeRow
	err = it.Next(&row)

	if err != nil {
		if err == iterator.Done {
			return time.Time{}, common.NewEmptyBillingTableError(t.TableID)
		}

		return time.Time{}, fmt.Errorf("error iterating through results of query %s. Caused by %s", query, err.Error())
	}

	return row.Export_time, nil
}

func (s *TableQueryImpl) executeExportTimeCustomerQuery(ctx context.Context, customerBQ *bigquery.Client, t *dataStructures.BillingTableInfo, query string) (finishingPoint time.Time, err error) {
	it, err := s.bqQuery.ReadQueryResult(ctx, customerBQ, query)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	var row dataStructures.ExportTimeRow
	err = it.Next(&row)

	if err != nil {
		if err == iterator.Done {
			return time.Time{}, common.NewEmptyBillingTableError(t.TableID)
		}

		return time.Time{}, fmt.Errorf("error iterating through results of query %s. Caused by %s", query, err.Error())
	}

	return row.Export_time, nil
}

func (s *TableQueryImpl) GetCustomersTableNewestRecordTime(ctx context.Context, customerBQ *bigquery.Client, t *dataStructures.BillingTableInfo) (finishingPoint time.Time, err error) {
	query, err := utils.GetCustomerTableNewestRecordTime(t)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	s.loggerProvider(ctx).Infof("query:%s", query)

	return s.executeExportTimeCustomerQuery(ctx, customerBQ, t, query)
}

func (s *TableQueryImpl) GetCustomersTableOldestRecordTimeNewerThan(ctx context.Context, customerBQ *bigquery.Client, t *dataStructures.BillingTableInfo, minExportTime time.Time) (oldestPoint time.Time, err error) {
	query, err := utils.GetTableOldestRecordTimeNewerThan(t, minExportTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	return s.executeExportTimeCustomerQuery(ctx, customerBQ, t, query)
}

func (s *TableQueryImpl) GetUnifiedTableOldestRecordTimeNewerThan(ctx context.Context, minExportTime time.Time) (oldestPoint time.Time, err error) {
	t := &dataStructures.BillingTableInfo{
		ProjectID: utils.GetProjectName(),
		DatasetID: consts.UnifiedGCPBillingDataset,
		TableID:   consts.UnifiedGCPRawTable,
	}

	query, err := utils.GetTableOldestRecordTimeNewerThan(t, minExportTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	return s.getLocalTableOldestRecordTimeNewerThan(ctx, query, t)
}

func (s *TableQueryImpl) getLocalTableOldestRecordTimeNewerThan(ctx context.Context, query string, t *dataStructures.BillingTableInfo) (oldestPoint time.Time, err error) {
	bq, err := s.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	return s.executeExportTimeCustomerQuery(ctx, bq, t, query)
}

func (s *TableQueryImpl) GetRawBillingNewestRecordTime(ctx context.Context) (finishingPoint time.Time, err error) {
	query := utils.GetRawBillingNewestRecordTime()

	bq, err := s.bqUtils.GetBQClientByProjectID(ctx, consts.BillingProjectProd)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable fetching project bq client %s. Caused by %s", query, err.Error())
	}

	it, err := s.bqQuery.ReadQueryResult(ctx, bq, query)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	var row dataStructures.ExportTimeRow
	err = it.Next(&row)

	if err != nil && err != iterator.Done {
		return time.Time{}, fmt.Errorf("error iterating through results of query %s. Caused by %s", query, err.Error())
	}

	return row.Export_time, nil
}

func (s *TableQueryImpl) GetRawBillingOldestRecordTime(ctx context.Context) (finishingPoint time.Time, err error) {
	useNowAsUpperLimit := false
	query := utils.GetRawBillingOldestRecordTime(useNowAsUpperLimit)

	bq, err := s.bqUtils.GetBQClientByProjectID(ctx, consts.BillingProjectProd)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable fetching project bq client %s. Caused by %s", query, err.Error())
	}

	it, err := s.bqQuery.ReadQueryResult(ctx, bq, query)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	var row dataStructures.ExportTimeRow

	err = it.Next(&row)
	if err == iterator.Done {
		useNowAsUpperLimit = true
		query = utils.GetRawBillingOldestRecordTime(useNowAsUpperLimit)
		it, err := s.bqQuery.ReadQueryResult(ctx, bq, query)

		if err != nil {
			return time.Time{}, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
		}

		err = it.Next(&row)
	}

	if err != nil && err != iterator.Done {
		return time.Time{}, fmt.Errorf("error iterating through results of query %s. Caused by %s", query, err.Error())
	}

	return row.Export_time, nil
}
func (s *TableQueryImpl) GetLocalTableNewestRecordTime(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (finishingPoint time.Time, err error) {
	query, err := utils.GetTableNewestRecordTime(itm)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	bq, err := s.bqUtils.GetBQClientByProjectID(ctx, itm.BQTable.ProjectID)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	it, err := s.bqQuery.ReadQueryResult(ctx, bq, query)
	if err != nil {
		return time.Time{}, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	var row dataStructures.ExportTimeRow
	err = it.Next(&row)

	if err != nil && err != iterator.Done {
		return time.Time{}, fmt.Errorf("error iterating through results of query %s. Caused by %s", query, err.Error())
	}

	return row.Export_time, nil
}

func (s *TableQueryImpl) GetFromUnifiedTableRowsCountPerTimeRange(ctx context.Context, billingAccount string, startTime, endTime *time.Time) (rowsCount int64, err error) {
	query, err := utils.GetFromUnifiedRowCountQuery(billingAccount, startTime, endTime)
	if err != nil {
		return 0, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	bq, err := s.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		return 0, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	it, err := s.bqQuery.ReadQueryResult(ctx, bq, query)
	if err != nil {
		return 0, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	var row dataStructures.RowsCountRow
	err = it.Next(&row)

	if err != nil && err != iterator.Done {
		return 0, fmt.Errorf("error iterating through results of query %s. Caused by %s", query, err.Error())
	}

	return row.Rows_count, nil
}

func (s *TableQueryImpl) GetLocalRowsCountPerTimeRange(ctx context.Context, billingAccount string, startTime, endTime *time.Time) (rowsCount int64, err error) {
	query, err := utils.GetLocalRowCountQuery(billingAccount, startTime, endTime)
	if err != nil {
		return 0, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	bq, err := s.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		return 0, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	it, err := s.bqQuery.ReadQueryResult(ctx, bq, query)
	if err != nil {
		return 0, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	var row dataStructures.RowsCountRow
	err = it.Next(&row)

	if err != nil && err != iterator.Done {
		return 0, fmt.Errorf("error iterating through results of query %s. Caused by %s", query, err.Error())
	}

	return row.Rows_count, nil
}

func (s *TableQueryImpl) GetAlternativeLocalRowsCountPerTimeRange(ctx context.Context, billingAccount string, startTime, endTime *time.Time) (rowsCount int64, err error) {
	query, err := utils.GetAlternativeLocalRowCountQuery(billingAccount, startTime, endTime)
	if err != nil {
		return 0, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	bq, err := s.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		return 0, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	it, err := s.bqQuery.ReadQueryResult(ctx, bq, query)
	if err != nil {
		return 0, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	var row dataStructures.RowsCountRow
	err = it.Next(&row)

	if err != nil && err != iterator.Done {
		return 0, fmt.Errorf("error iterating through results of query %s. Caused by %s", query, err.Error())
	}

	return row.Rows_count, nil
}

func (s *TableQueryImpl) GetCustomerRowsCountPerTimeRange(ctx context.Context, billingAccountID string, startTime, endTime *time.Time) (int64, error) {
	logger := s.loggerProvider(ctx)

	etm, err := s.metadata.GetExternalTaskMetadata(ctx, billingAccountID)
	if err != nil {
		return 0, err
	}

	query, err := utils.GetCustomerRowCountQuery(etm.BQTable, startTime, endTime)
	if err != nil {
		return 0, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	customerBQ, err := s.customerBQClient.GetCustomerBQClientWithParams(ctx, etm.ServiceAccountEmail, etm.BQTable.ProjectID)
	if err != nil {
		logger.Errorf("Error getting customer BQ client: %v", err)
		return 0, err
	}
	defer customerBQ.Close()

	it, err := s.bqQuery.ReadQueryResult(ctx, customerBQ, query)
	if err != nil {
		return 0, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	var row dataStructures.RowsCountRow
	err = it.Next(&row)

	if err != nil && err != iterator.Done {
		return 0, fmt.Errorf("error iterating through results of query %s. Caused by %s", query, err.Error())
	}

	return row.Rows_count, nil
}

func (s *TableQueryImpl) CopyFromTmpTableAllRows(ctx context.Context, imm *dataStructures.InternalManagerMetadata, itms []*dataStructures.InternalTaskMetadata) (job *bigquery.Job, err error) {
	query := utils.GetCopyFromTmpTableAllRowsQuery(imm.Iteration, itms)
	bq, err := s.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())

	if err != nil {
		return nil, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	ctx, cancelF := context.WithTimeout(ctx, time.Until(*imm.CopyToUnifiedTableJob.WaitToStartTimeout))
	defer cancelF()

	job, err = s.bqQuery.ExecQueryAsync(ctx, bq, &common.BQExecuteQueryData{
		Query:      query,
		Clustering: &bigquery.Clustering{Fields: []string{"billing_account_id"}},
		DefaultTable: &dataStructures.BillingTableInfo{
			ProjectID: utils.GetProjectName(),
			TableID:   utils.GetUnifiedTempTableName(imm.Iteration),
			DatasetID: consts.UnifiedGCPBillingDataset,
		},
		DestinationTable: &dataStructures.BillingTableInfo{
			ProjectID: utils.GetProjectName(),
			TableID:   consts.UnifiedGCPRawTable,
			DatasetID: consts.UnifiedGCPBillingDataset,
		},
		WriteDisposition: bigquery.WriteAppend,
		WaitTillDone:     false,
		ConfigJobID:      utils.GetCopyToUnifiedTableJobPrefix(imm.Iteration),
		Internal:         true,
		Export:           false,
	})
	if err != nil {
		return job, fmt.Errorf("unable to execute query %s. Caused by %s", query, err)
	}

	return job, nil
}

func (t *TableQueryImpl) CopyFromLocalToTmpTable(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (job *bigquery.Job, err error) {
	logger := t.loggerProvider(ctx)

	bq, err := t.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		logger.Errorf("unable to GetBQClientByProjectID. Caused by %s", err)
		return nil, err
	}

	query, err := utils.GetInternalUpdateQuery(itm)
	if err != nil {
		logger.Errorf("unable to GetInternalUpdateQuery. Caused by %s", err)
		return nil, err
	}

	logger.Infof("running query %s", query)

	job, err = t.bqQuery.ExecQueryAsync(ctx, bq, &common.BQExecuteQueryData{
		Query:      query,
		Clustering: &bigquery.Clustering{Fields: []string{"billing_account_id"}},
		DefaultTable: &dataStructures.BillingTableInfo{
			ProjectID: itm.BQTable.ProjectID,
			DatasetID: itm.BQTable.DatasetID,
			TableID:   itm.BQTable.TableID,
		},
		DestinationTable: &dataStructures.BillingTableInfo{
			ProjectID: utils.GetProjectName(),
			DatasetID: consts.UnifiedGCPBillingDataset,
			TableID:   utils.GetUnifiedTempTableName(itm.Iteration),
		},
		WriteDisposition: bigquery.WriteAppend,
		WaitTillDone:     false,
		Internal:         true,
		BillingAccountID: itm.BillingAccount,
		ConfigJobID:      utils.GetFromLocalTableToTmpTableJobPrefix(itm.BillingAccount, itm.Iteration),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	return job, err
}

func (t *TableQueryImpl) CopyFromAlternativeLocalToTmpTable(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (job *bigquery.Job, err error) {
	logger := t.loggerProvider(ctx)

	bq, err := t.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		logger.Errorf("unable to GetBQClientByProjectID. Caused by %s", err)
		return nil, err
	}

	query, err := utils.GetAlternativeInternalUpdateQuery(itm)
	if err != nil {
		logger.Errorf("unable to GetInternalUpdateQuery. Caused by %s", err)
		return nil, err
	}

	logger.Infof("query: %s", query)

	job, err = t.bqQuery.ExecQueryAsync(ctx, bq, &common.BQExecuteQueryData{
		Query:      query,
		Clustering: &bigquery.Clustering{Fields: []string{"billing_account_id"}},
		DefaultTable: &dataStructures.BillingTableInfo{
			ProjectID: utils.GetProjectName(),
			DatasetID: consts.AlternativeLocalBillingDataset,
			TableID:   consts.UnifiedAlternativeRawBillingTable,
		},
		DestinationTable: &dataStructures.BillingTableInfo{
			ProjectID: utils.GetProjectName(),
			DatasetID: consts.AlternativeLocalBillingDataset,
			TableID:   consts.UnifiedAlternativeRawBillingTable,
		},
		WriteDisposition: bigquery.WriteAppend,
		WaitTillDone:     true,
		Internal:         true,
		BillingAccountID: itm.BillingAccount,
		//ConfigJobID:      utils.GetFromLocalTableToTmpTableJobPrefix(itm.BillingAccount, itm.Iteration),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	return job, err
}

func (s *TableQueryImpl) MarkTmpTableBillingRowsAsVerified(ctx context.Context, itm *dataStructures.InternalTaskMetadata) (job *bigquery.Job, err error) {
	query := utils.GetMarkTmpTableBillingRowsAsVerifiedQuery(itm)
	bq, err := s.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())

	if err != nil {
		return nil, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	job, err = s.bqQuery.ExecQueryAsync(ctx, bq, &common.BQExecuteQueryData{
		Query:      query,
		Clustering: &bigquery.Clustering{Fields: []string{"billing_account_id"}},
		DefaultTable: &dataStructures.BillingTableInfo{
			ProjectID: utils.GetProjectName(),
			TableID:   utils.GetUnifiedTempTableName(itm.Iteration),
			DatasetID: consts.UnifiedGCPBillingDataset,
		},
		//DestinationTable: &dataStructures.BillingTableInfo{
		//	ProjectID: utils.GetProjectName(),
		//	TableID:   utils.GetUnifiedTempTableName(itm.Iteration),
		//	DatasetID: consts.UnifiedGCPBillingDataset,
		//},
		WriteDisposition: bigquery.WriteAppend,
		WaitTillDone:     false,
		Internal:         false,
		Export:           true,
		ConfigJobID:      utils.GetMarkVerifiedTmpTableJobPrefix(itm.BillingAccount, itm.Iteration),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to execute query %s. Caused by %s", query, err.Error())
	}

	return job, err
}

func (s *TableQueryImpl) GetLocalTableLatestExportTime(ctx context.Context, bq *bigquery.Client, billingAccountID string) (*time.Time, error) {
	var lastExportTime time.Time

	rows, err := s.bqQuery.ReadQueryResult(ctx, bq, utils.GetLocalLatestExportTimeQuery(billingAccountID))
	if err != nil {
		if gapiErr, ok := err.(*googleapi.Error); ok {
			if gapiErr.Code != http.StatusNotFound {
				return nil, err
			}
		}

		lastExportTime = time.Time{}
	} else {
		var row struct {
			ExportTime time.Time `bigquery:"export_time"`
		}

		err = rows.Next(&row)
		if err == iterator.Done {
			return &lastExportTime, nil
		}

		if err != nil {
			// TODO: Handle error.
			return nil, err
		}

		lastExportTime = row.ExportTime
	}

	return &lastExportTime, nil
}

func (s *TableQueryImpl) GetCustomerRowsCount(ctx context.Context, billingAccountID string, segment *dataStructures.Segment) (map[dataStructures.HashableSegment]int, error) {
	logger := s.loggerProvider(ctx)

	etm, err := s.metadata.GetExternalTaskMetadata(ctx, billingAccountID)
	if err != nil {
		return nil, err
	}

	customerBQ, err := s.customerBQClient.GetCustomerBQClientWithParams(ctx, etm.ServiceAccountEmail, etm.BQTable.ProjectID)
	if err != nil {
		logger.Errorf("Error getting customer BQ client: %v", err)
		return nil, err
	}
	defer customerBQ.Close()

	return s.getRowsCount(ctx, customerBQ, etm.BQTable, "", segment)
}

func (s *TableQueryImpl) GetCustomerRowsCountByExportTime(ctx context.Context, billingAccountID string, segment *dataStructures.Segment) (map[string]int64, error) {
	logger := s.loggerProvider(ctx)

	etm, err := s.metadata.GetExternalTaskMetadata(ctx, billingAccountID)
	if err != nil {
		return nil, err
	}

	customerBQ, err := s.customerBQClient.GetCustomerBQClientWithParams(ctx, etm.ServiceAccountEmail, etm.BQTable.ProjectID)
	if err != nil {
		logger.Errorf("Error getting customer BQ client: %v", err)
		return nil, err
	}
	defer customerBQ.Close()

	return s.getRowsCountByExportTime(ctx, customerBQ, etm.BQTable, "", segment)
}

func (s *TableQueryImpl) GetLocalRowsCount(ctx context.Context, billingAccountID string, segment *dataStructures.Segment) (map[dataStructures.HashableSegment]int, error) {
	bq, err := s.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		return nil, err
	}

	project := utils.GetProjectName()
	dataset := consts.LocalBillingDataset
	table := utils.GetLocalCopyAccountTableName(billingAccountID)

	if billingAccountID == googleCloudConsts.MasterBillingAccount {
		project = consts.BillingProjectProd
		dataset = consts.ResellRawBillingDataset
		table = consts.ResellRawBillingTable
	}

	return s.getRowsCount(ctx, bq, &dataStructures.BillingTableInfo{
		ProjectID: project,
		DatasetID: dataset,
		TableID:   table,
	}, "", segment)
}

func (s *TableQueryImpl) GetLocalRowsCountByExportTime(ctx context.Context, billingAccountID string, segment *dataStructures.Segment) (map[string]int64, error) {
	bq, err := s.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		return nil, err
	}

	project := utils.GetProjectName()
	dataset := consts.LocalBillingDataset
	table := utils.GetLocalCopyAccountTableName(billingAccountID)

	if billingAccountID == googleCloudConsts.MasterBillingAccount {
		project = consts.BillingProjectProd
		dataset = consts.ResellRawBillingDataset
		table = consts.ResellRawBillingTable
	}

	return s.getRowsCountByExportTime(ctx, bq, &dataStructures.BillingTableInfo{
		ProjectID: project,
		DatasetID: dataset,
		TableID:   table,
	}, "", segment)
}

func (s *TableQueryImpl) GetLUnifiedRowsCount(ctx context.Context, billingAccountID string, segment *dataStructures.Segment) (map[dataStructures.HashableSegment]int, error) {
	bq, err := s.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		return nil, err
	}

	return s.getRowsCount(ctx, bq, &dataStructures.BillingTableInfo{
		ProjectID: utils.GetProjectName(),
		DatasetID: consts.UnifiedGCPBillingDataset,
		TableID:   consts.UnifiedGCPRawTable,
	}, billingAccountID, segment)
}

func (s *TableQueryImpl) GetLUnifiedRowsCountByExportTime(ctx context.Context, billingAccountID string, segment *dataStructures.Segment) (map[string]int64, error) {
	bq, err := s.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		return nil, err
	}

	return s.getRowsCountByExportTime(ctx, bq, &dataStructures.BillingTableInfo{
		ProjectID: utils.GetProjectName(),
		DatasetID: consts.UnifiedGCPBillingDataset,
		TableID:   consts.UnifiedGCPRawTable,
	}, billingAccountID, segment)
}

func (s *TableQueryImpl) getRowsCount(ctx context.Context, bq *bigquery.Client, table *dataStructures.BillingTableInfo, billingAccountID string, segment *dataStructures.Segment) (map[dataStructures.HashableSegment]int, error) {
	query, segmentLength, err := utils.GetRowsCountQuery(table, billingAccountID, segment)
	if err != nil {
		return nil, err
	}

	type row struct {
		TimeStamp time.Time `bigquery:"time_stamp"`
		RowsCount int       `bigquery:"rows_count"`
	}

	allRows := []row{}
	rowsCount := make(map[dataStructures.HashableSegment]int)

	rows, err := s.bqQuery.ReadQueryResult(ctx, bq, query)
	if err != nil {
		return nil, err
	} else {
		for {
			var r row

			err = rows.Next(&r)
			if err == iterator.Done {
				break
			}

			if err != nil {
				// TODO: Handle error.
				return nil, err
			}

			allRows = append(allRows, r)
		}
	}

	if len(allRows) == 0 {
		return rowsCount, nil
	}

	for _, r := range allRows {
		startTime := r.TimeStamp
		endTime := r.TimeStamp

		switch segmentLength {
		case utils.SegmentLengthHour:
			endTime = endTime.Add(time.Hour)
		case utils.SegmentLengthDay:
			endTime = endTime.Add(24 * time.Hour)
		case utils.SegmentLengthMonth:
			endTime = endTime.AddDate(0, 1, 0)
		}

		rowsCount[dataStructures.HashableSegment{
			StartTime: startTime,
			EndTime:   endTime,
		}] = r.RowsCount
	}

	return rowsCount, nil
}

func (s *TableQueryImpl) getRowsCountByExportTime(ctx context.Context, bq *bigquery.Client, table *dataStructures.BillingTableInfo, billingAccountID string, segment *dataStructures.Segment) (map[string]int64, error) {
	query, err := utils.GetRowsCountByExportTimeQuery(table, billingAccountID, segment)
	if err != nil {
		return nil, err
	}

	type row struct {
		ExportTime time.Time `bigquery:"export_time"`
		RowsCount  int64     `bigquery:"rows_count"`
	}

	allRows := []row{}
	//rowsCount := make(map[dataStructures.HashableSegment]int)

	rows, err := s.bqQuery.ReadQueryResult(ctx, bq, query)
	if err != nil {
		return nil, err
	} else {
		for {
			var r row

			err = rows.Next(&r)
			if err == iterator.Done {
				break
			}

			if err != nil {
				// TODO: Handle error.
				return nil, err
			}

			allRows = append(allRows, r)
		}
	}

	rowsCountByExportTime := make(map[string]int64)
	if len(allRows) == 0 {
		return rowsCountByExportTime, nil
	}

	for _, r := range allRows {
		rowsCountByExportTime[r.ExportTime.Format(consts.ExportTimeLayoutWithMillis)] = r.RowsCount
		//startTime := r.TimeStamp
		//endTime := r.TimeStamp
		//
		//switch segmentLength {
		//case utils.SegmentLengthHour:
		//	endTime = endTime.Add(time.Hour)
		//case utils.SegmentLengthDay:
		//	endTime = endTime.Add(24 * time.Hour)
		//case utils.SegmentLengthMonth:
		//	endTime = endTime.AddDate(0, 1, 0)
		//}
		//rowsCount[dataStructures.HashableSegment{
		//	StartTime: startTime,
		//	EndTime:   endTime,
		//}] = r.RowsCount
	}

	return rowsCountByExportTime, nil
}

func (s *TableQueryImpl) runQueryWithDestinationTable(ctx context.Context, query string, dstTable *dataStructures.BillingTableInfo) error {
	bq, err := s.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		return err
	}

	q := bq.Query(query)
	q.WriteDisposition = bigquery.WriteAppend
	q.CreateDisposition = bigquery.CreateIfNeeded
	q.Dst = &bigquery.Table{
		ProjectID: dstTable.ProjectID,
		DatasetID: dstTable.DatasetID,
		TableID:   dstTable.TableID,
	}

	job, err := q.Run(ctx)
	if err != nil {
		return err
	}

	st, err := job.Wait(ctx)
	if err != nil {
		return err
	}

	if st.Err() != nil {
		return err
	}

	return nil
}

func (s *TableQueryImpl) RunDetailedTableRewritesMapping(ctx context.Context) error {
	query := utils.GetDetailedTableRewritesMappingQuery()
	dstTable := utils.GetDetailedTableAnalyticsTableName()

	return s.runQueryWithDestinationTable(ctx, query, dstTable)
}

func (s *TableQueryImpl) RunDataFreshnessReport(ctx context.Context) error {
	query := utils.GetDetailedTableUsageAndExportTimeDifferential()
	dstTable := utils.GetFreshnessReportTableName()

	return s.runQueryWithDestinationTable(ctx, query, dstTable)
}
