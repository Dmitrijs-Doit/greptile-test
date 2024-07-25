package service

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/export/dal"
	dalIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/export/dal/iface"
	domainExport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/export/domain"
)

const (
	ProdViewsProject   = "doit-data-views"
	DevViewsProject    = "doit-data-views-dev"
	ProdAWSDataProject = "doitintl-cmp-aws-data"
	DevAWSDataProject  = "cmp-aws-etl-dev"
)

type BillingExportService struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	bigQueryDAL    dalIface.BigQueryDAL
}

func NewBillingExportService(
	loggerProvider logger.Provider,
	conn *connection.Connection) *BillingExportService {

	viewsProject := DevViewsProject
	billingDataProject := DevAWSDataProject

	if common.Production {
		viewsProject = ProdViewsProject
		billingDataProject = ProdAWSDataProject
	}

	return &BillingExportService{
		loggerProvider: loggerProvider,
		conn:           conn,
		bigQueryDAL:    dal.NewBigQueryDAL(context.Background(), viewsProject, billingDataProject),
	}
}

func (s *BillingExportService) ExportBillingData(ctx context.Context, customerID string, inputParams *domainExport.BillingExportInputStruct) error {
	log := s.loggerProvider(ctx)

	if inputParams.Cloud != common.Assets.AmazonWebServices {
		return nil
	}

	log.Info("Checking if view exists for customerId %s", customerID)

	viewExists, err := s.bigQueryDAL.CheckViewExists(ctx, customerID)
	if err != nil {
		return err
	}

	if viewExists {
		log.Info("View already exist")
		return nil
	}

	log.Info("Creating view")

	if err := s.bigQueryDAL.CreateViewAWS(ctx, customerID); err != nil {
		return err
	}

	log.Info("Sharing view with customer")

	if err := s.bigQueryDAL.AuthorizeView(ctx, customerID, inputParams.CustomerEmail); err != nil {
		return err
	}

	return nil
}
