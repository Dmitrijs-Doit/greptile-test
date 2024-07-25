package test_connection

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/service"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type TestConnection struct {
	loggerProvider logger.Provider
	*connection.Connection
	customerBQClient service.ExternalBigQueryClient
	bucket           service.Bucket
	dataCopier       *service.BillingDataCopierService
	table            service.Table
}

func NewTestConnection(log logger.Provider, conn *connection.Connection) *TestConnection {
	return &TestConnection{
		log,
		conn,
		service.NewExternalBigQueryClient(log, conn),
		service.NewBucket(log, conn),
		service.NewBillingDataCopierService(log, conn),
		service.NewTable(log, conn),
	}
}

func (s *TestConnection) TestBillingConnection(ctx context.Context, billingAccountID, serviceAccountEmail string, bqTable *pkg.BillingTableInfo) error {
	logger := s.loggerProvider(ctx)

	var testBucketName string

	defer func() {
		if err := s.cleanUp(ctx, billingAccountID, testBucketName); err != nil {
			logger.Error(err)
		}
	}()

	testBucketName, err := s.testCopyToBucket(ctx, billingAccountID, serviceAccountEmail, &dataStructures.BillingTableInfo{
		ProjectID: bqTable.ProjectID,
		DatasetID: bqTable.DatasetID,
		TableID:   bqTable.TableID,
	})
	if err != nil {
		logger.Errorf("error TestCopyFromCustomerToBucket for BA %s billing connection, %s", billingAccountID, err)
		return err
	}

	if err := s.testCopyFromBucketToLocalTable(ctx, billingAccountID, testBucketName); err != nil {
		logger.Errorf("error TestCopyFromBucketToLocalTable for BA %s billing connection, %s", billingAccountID, err)
		return err
	}

	return nil
}

func (s *TestConnection) getFileURI(billingAccountID, testBucketName string) string {
	return fmt.Sprintf("gs://%s/%s/test/*.gzip", testBucketName, billingAccountID)
}

func (s *TestConnection) testCopyToBucket(ctx context.Context, billingAccountID, serviceAccountEmail string, bqTable *dataStructures.BillingTableInfo) (string, error) {
	customerBQ, err := s.customerBQClient.GetCustomerBQClientWithParams(ctx, serviceAccountEmail, bqTable.ProjectID)
	if err != nil {
		return "", err
	}
	defer customerBQ.Close()

	md, err := customerBQ.DatasetInProject(bqTable.ProjectID, bqTable.DatasetID).Metadata(ctx)
	if err != nil {
		return "", err
	}

	location := md.Location

	testBucketName, err := s.bucket.Create(ctx, location, true)
	if err != nil {
		// TODO: our fault
		return "", err
	}

	now := time.Now()
	startTime := now.UTC().Add(-24 * time.Hour)
	endTime := now.UTC()
	Segment := &dataStructures.Segment{
		StartTime: &startTime,
		EndTime:   &endTime,
	}

	copyToBucketData := service.CopyToBucketData{
		BucketName:          testBucketName,
		FileURI:             s.getFileURI(billingAccountID, testBucketName),
		Segment:             Segment,
		ServiceAccountEmail: serviceAccountEmail,
		RunQueryData: &common.BQExecuteQueryData{
			DefaultTable:     bqTable,
			DestinationTable: bqTable,
			BillingAccountID: billingAccountID,
			WriteDisposition: bigquery.WriteAppend,
			WaitTillDone:     true,
		},
	}

	_, err = s.dataCopier.CopyFromCustomerTableToBucket(ctx, customerBQ, &copyToBucketData)
	if err != nil {
		return "", err
	}

	return testBucketName, nil
}

func (s *TestConnection) testCopyFromBucketToLocalTable(ctx context.Context, billingAccountID, testBucketName string) error {
	err := s.table.CreateLocalTable(ctx, billingAccountID)
	if err != nil {
		return err
	}

	dataFromBucket := service.CopyFromBucketData{
		FileURI: s.getFileURI(billingAccountID, testBucketName),
		RunQueryData: &common.BQExecuteQueryData{
			BillingAccountID: billingAccountID,
			WriteDisposition: bigquery.WriteAppend,
			WaitTillDone:     true,
			Internal:         false,
			ConfigJobID:      utils.GetFromBucketJobPrefix(billingAccountID, -1),
		},
	}

	_, err = s.dataCopier.CopyFromBucketToTable(ctx, &dataFromBucket)
	if err != nil {
		return err
	}

	return nil
}

func (s *TestConnection) cleanUp(ctx context.Context, billingAccountID, testBucketName string) error {
	err := s.table.DeleteLocalTable(ctx, billingAccountID)
	if err != nil {
		return fmt.Errorf("error DeleteLocalTable for testing BA %s billing connection, %s", billingAccountID, err)
	}

	if testBucketName != "" {
		err := s.bucket.DeleteBucket(ctx, testBucketName)
		if err != nil {
			return fmt.Errorf("error DeleteBucket for test bucket %s, %s", testBucketName, err)
		}
	}

	return nil
}
