package service

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	fsdal "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/dal"
	billingService "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/service"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/bq_utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type TableQuery struct {
	loggerProvider logger.Provider
	*connection.Connection
	bqQuery           *dal.Query
	billingAccountDAL fsdal.FlexsaveStandalone
	bqUtils           *bq_utils.BQ_Utils
	customerBQClient  billingService.ExternalBigQueryClient
}

func NewTableQuery(log logger.Provider, conn *connection.Connection) *TableQuery {
	ctx := context.Background()

	return &TableQuery{
		log,
		conn,
		dal.NewQuery(log, conn),
		fsdal.NewFlexsaveStandaloneDALWithClient(conn.Firestore(ctx)),
		bq_utils.NewBQ_UTils(log, conn),
		billingService.NewExternalBigQueryClient(log, conn),
	}
}

func (t *TableQuery) CopyFromLocalToTmpTable(ctx context.Context, atm *dataStructures.AutomationTaskMetadata) (job *bigquery.Job, err error) {
	logger := t.loggerProvider(ctx)

	customerBQ, err := t.customerBQClient.GetCustomerBQClientWithParams(ctx, consts.ServiceAccount, consts.DummyBQProjectName)
	if err != nil {
		err = fmt.Errorf("unable to GetCustomerBQClient.Caused by %s", err)
		logger.Error(err)

		return nil, err
	}
	defer customerBQ.Close()

	job, err = t.bqQuery.ExecQueryAsync(ctx, customerBQ, &common.BQExecuteQueryData{
		Query:            utils.GetCopyToDummyQuery(atm.BillingAccountID, atm.RowsPerHour),
		DefaultTable:     atm.BQTable,
		DestinationTable: atm.BQTable,
		WriteDisposition: bigquery.WriteAppend,
		WaitTillDone:     false,
		Internal:         true,
		BillingAccountID: atm.BillingAccountID,
		ConfigJobID:      utils.GetCopyToDummyTablejobPrefix(atm),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to execute query %s. Caused by %s", utils.GetCopyToDummyQuery(atm.BillingAccountID, atm.RowsPerHour), err.Error())
	}

	return job, err
}
