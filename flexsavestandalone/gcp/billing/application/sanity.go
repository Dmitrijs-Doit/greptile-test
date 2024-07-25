package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/service"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/bq_utils"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Sanity struct {
	loggerProvider logger.Provider
	*connection.Connection
	metadata         service.Metadata
	table            service.Table
	assets           *service.Assets
	bqTable          service.Table
	config           service.PipelineConfig
	bucket           service.Bucket
	customerBQClient service.ExternalBigQueryClient
	bqUtils          *bq_utils.BQ_Utils
	dataset          *service.Dataset
	query            service.TableQuery
}

func NewSanity(log logger.Provider, conn *connection.Connection) *Sanity {
	return &Sanity{
		loggerProvider:   log,
		Connection:       conn,
		metadata:         service.NewMetadata(log, conn),
		table:            service.NewTable(log, conn),
		assets:           service.NewAssets(log, conn),
		bqTable:          service.NewTable(log, conn),
		config:           service.NewPipelineConfig(log, conn),
		bucket:           service.NewBucket(log, conn),
		customerBQClient: service.NewExternalBigQueryClient(log, conn),
		bqUtils:          bq_utils.NewBQ_UTils(log, conn),
		dataset:          service.NewDataset(log, conn),
		query:            service.NewTableQuery(log, conn),
	}
}

func (s *Sanity) RunSanity(ctx context.Context) error {
	logger := s.loggerProvider(ctx)

	masterBillingMD, err := s.metadata.GetInternalTaskMetadata(ctx, consts.MasterBillingAccount)
	if err != nil {
		err = fmt.Errorf("unable to GetInternalTaskMetadata. Caused by %s", err)
		logger.Info(err)

		return err
	}

	config, err := s.config.GetPipelineConfig(ctx)
	if err != nil {
		err = fmt.Errorf("unable to GetPipelineConfig. Caused by %s", err)
		logger.Info(err)

		return err
	}

	templateProjBQ, err := s.bqUtils.GetBQClientByProjectID(ctx, config.TemplateBillingDataProjectID)
	if err != nil {
		err = fmt.Errorf("unable to GetPipelineConfig. Caused by %s", err)
		logger.Info(err)

		return err
	}

	bq, err := s.bqUtils.GetBQClientByProjectID(ctx, utils.GetProjectName())
	if err != nil {
		err = fmt.Errorf("unable to GetPipelineConfig. Caused by %s", err)
		logger.Info(err)

		return err
	}

	_ = bq
	_ = templateProjBQ

	return s.runTests(ctx, masterBillingMD)
}

func (s *Sanity) runTests(ctx context.Context, masterBillingMD *dataStructures.InternalTaskMetadata) error {
	logger := s.loggerProvider(ctx)

	startingTime, err := s.query.GetLocalTableOldestRecordTime(ctx, masterBillingMD)
	if err != nil {
		err = fmt.Errorf("unable to GetLocalTableOldestRecordTime. Caused by %s", err)
		logger.Info(err)

		return err
	}

	startPoint := time.Date(startingTime.Year(), startingTime.Month(), 1, 0, 0, 0, 0, time.UTC)
	endPoint := startPoint

	finalTime, err := s.query.GetUnifiedTableOldestRecordByBA(ctx, masterBillingMD)
	if err != nil {
		err = fmt.Errorf("unable to GetLocalTableOldestRecordTime. Caused by %s", err)
		logger.Info(err)

		return err
	}

	buff := strings.Builder{}

	_, err = buff.WriteString("\n********************************************* STARTING SANITY CHECK *********************************************\n")
	if err != nil {
		err = fmt.Errorf("unable to add to buffer. Caused by %s", err)
		logger.Info(err)

		return err
	}

	successful := true
	months := finalTime.Sub(startingTime).Hours() / 24 / 30

	var ratio, persantage float64
	ratio = 1 / months * 100

	for !endPoint.Equal(finalTime) {
		startPoint = endPoint

		endPoint = startPoint.AddDate(0, 1, 0)
		if endPoint.After(finalTime) {
			endPoint = finalTime
		}

		expectedRows, err := s.query.GetLocalRowsCountPerTimeRange(ctx, masterBillingMD.BillingAccount, &startPoint, &endPoint)
		if err != nil {
			err = fmt.Errorf("unable to GetLocalRowsCountPerTimeRange. Caused by %s", err)
			logger.Info(err)

			return err
		}

		foundRows, err := s.query.GetFromUnifiedTableRowsCountPerTimeRange(ctx, masterBillingMD.BillingAccount, &startPoint, &endPoint)
		if err != nil {
			err = fmt.Errorf("unable to GetLocalRowsCountPerTimeRange. Caused by %s", err)
			logger.Info(err)

			return err
		}

		successful = successful && foundRows == expectedRows

		_, err = buff.WriteString(fmt.Sprintf("passed=%t [%v - %v] => expected=%d vs found=%d\n", foundRows == expectedRows, startPoint, endPoint, expectedRows, foundRows))
		if err != nil {
			err = fmt.Errorf("unable to add to buffer. Caused by %s", err)
			logger.Info(err)

			return err
		}

		logger.Infof(".......%.2f%%", persantage)
		persantage = persantage + ratio
		//logger.Infof("passed=%t [%v - %v] => expected=%d vs found=%d", foundRows == expectedRows, startPoint, endPoint, expectedRows, foundRows)
	}

	_, err = buff.WriteString("************************************************ END SANITY CHECK ************************************************")
	if err != nil {
		err = fmt.Errorf("unable to add to buffer. Caused by %s", err)
		logger.Info(err)

		return err
	}

	logger.Info(buff.String())

	if !successful {
		return fmt.Errorf("sanity failed")
	}

	return nil
}
