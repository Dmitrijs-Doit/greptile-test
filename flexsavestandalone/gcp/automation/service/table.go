package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"cloud.google.com/go/bigquery"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/utils"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/utils/consts"
	billingDal "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/dal"
	billingService "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/service"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Table struct {
	loggerProvider logger.Provider
	*connection.Connection
	dal              *dal.Metadata
	billingTable     billingService.Table
	customerBQClient billingService.ExternalBigQueryClient
	table            *billingDal.BQTable
}

func NewTable(log logger.Provider, conn *connection.Connection) *Table {
	return &Table{
		loggerProvider:   log,
		Connection:       conn,
		dal:              dal.NewMetadata(log, conn),
		billingTable:     billingService.NewTable(log, conn),
		customerBQClient: billingService.NewExternalBigQueryClient(log, conn),
	}
}

func (t *Table) CreateDummyTables(ctx context.Context, aom *dataStructures.AutomationOrchestratorMetadata) error {
	logger := t.loggerProvider(ctx)

	customerBQ, err := t.customerBQClient.GetCustomerBQClientWithParams(ctx, consts.ServiceAccount, consts.DummyBQProjectName)
	if err != nil {
		err = fmt.Errorf("unable to GetCustomerBQClient.Caused by %s", err)
		logger.Error(err)

		return err
	}

	defer customerBQ.Close()

	creatingTablesWG := sync.WaitGroup{}
	creatingTablesWG.Add(aom.NumOfDummyUsers)

	for i := 0; i < aom.NumOfDummyUsers; i++ {
		go func(index int) {
			defer creatingTablesWG.Done()

			tableName := utils.GetDummyTableName(aom.Version, index)
			err := t.CreateDummyTable(ctx, tableName, customerBQ)

			if err != nil {
				err = fmt.Errorf("unable to create table %s. Caused by %s", tableName, err)
				logger.Error(err)
			}
		}(i)
	}

	return nil
}

func (t *Table) CreateDummyTable(ctx context.Context, tableName string, customerBQ *bigquery.Client) error {
	logger := t.loggerProvider(ctx)

	md, err := t.billingTable.GetDefaultTemplate(ctx, tableName)
	if err != nil {
		err = fmt.Errorf("unable to GetDefaultTemplate for table %s. Caused by %s", tableName, err)
		logger.Error(err)

		return err
	}

	err = t.table.MustCreateTable(ctx, customerBQ, consts.DummyBQDatasetName, md)
	if err != nil {
		err = fmt.Errorf("unable to Create bq table.Caused by %s", err)
		logger.Error(err)

		return err
	}

	return nil
}

func (t *Table) DeleteDummyTables(ctx context.Context) error {
	logger := t.loggerProvider(ctx)

	customerBQ, err := t.customerBQClient.GetCustomerBQClientWithParams(ctx, consts.ServiceAccount, consts.DummyBQProjectName)
	if err != nil {
		err = fmt.Errorf("unable to GetCustomerBQClient.Caused by %s", err)
		logger.Error(err)

		return err
	}

	defer customerBQ.Close()

	tableIterator := customerBQ.Dataset(consts.DummyBQDatasetName).Tables(ctx)

	for {
		table, err := tableIterator.Next()
		if err != nil {
			if err == iterator.Done {
				return nil
			}

			err = fmt.Errorf("unable to get Tables. Coused by %s", err)

			return err
		}

		if table.TableID == consts.DummyBQTableNameOriginal {
			continue
		}

		if !strings.Contains(table.TableID, utils.GetProjectNameUnderscore()) {
			logger.Infof("skipping deletion of table %s because it belongs to another environment", table.TableID)
			continue
		}

		err = table.Delete(ctx)
		if err != nil {
			if gapiErr, ok := err.(*googleapi.Error); ok && gapiErr.Code == http.StatusNotFound {
				if gapiErr.Code == http.StatusNotFound {
					logger.Infof("skipping deletion of table %s since it's been already deleted.", table.TableID)
					continue
				}
			}

			fmt.Errorf("unable to delete table.Caused by %s", err)
			logger.Error(err)

			return err
		}
	}
}

func (m *Table) grantAccessToDummyTable(ctx context.Context, bqTable *bigquery.Table) error {
	logger := m.loggerProvider(ctx)

	automationBQ, err := m.customerBQClient.GetCustomerBQClientWithParams(ctx, consts.ServiceAccount, consts.DummyBQProjectName)
	if err != nil {
		err = fmt.Errorf("unable to GetCustomerBQClient.Caused by %s", err)
		logger.Error(err)

		return err
	}

	defer automationBQ.Close()

	policy, err := bqTable.IAM().V3().Policy(ctx)
	if err != nil {
		err = fmt.Errorf("unable to get policy from table %s. Caused by %s", bqTable.TableID, err)
		logger.Error(err)

		return err
	}

	logger.Infof("%v", policy)

	return nil
}
