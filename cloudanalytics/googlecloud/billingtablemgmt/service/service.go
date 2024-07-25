package googlecloud

import (
	"context"
	"fmt"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"

	assetsDal "github.com/doitintl/hello/scheduled-tasks/assets/dal"
	cspTaskReporterService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/csptaskreporter/service"
	cspTaskReporterServiceIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/csptaskreporter/service/iface"
	discountsService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/discounts/service"
	discountsServiceIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/discounts/service/iface"
	cspDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/dal/csp"
	cspDALIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/dal/csp/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	gkeCostAllocationService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/gke_cost_allocation/service/cost_allocation"
	gkeCostAllocationServiceIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/gke_cost_allocation/service/cost_allocation/iface"
	metadataDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/dal"
	gcpMetadata "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/gcp"
	gcpMetadataIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/gcp/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	reportStatus "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/statuses"
	tableMgmtDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/service"
	"github.com/doitintl/hello/scheduled-tasks/common"
	contractDal "github.com/doitintl/hello/scheduled-tasks/contract/dal"
	contractDalIface "github.com/doitintl/hello/scheduled-tasks/contract/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	billingUpdateDal "github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/shared/dal"
	standaloneDomain "github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/shared/domain"
)

type BillingTableManagementService struct {
	loggerProvider           logger.Provider
	conn                     *connection.Connection
	reportStatusService      *reportStatus.ReportStatusesService
	billingUpdate            billingUpdateDal.BillingUpdate
	billingImport            billingUpdateDal.BillingImportStatus
	cspDAL                   cspDALIface.ICSPFirestore
	discountsService         discountsServiceIface.IDiscountsService
	gcpMetadataService       gcpMetadataIface.GCPMetadata
	cspTaskReporter          cspTaskReporterServiceIface.TaskReporter
	assets                   assetsDal.Assets
	gkeCostAllocationService gkeCostAllocationServiceIface.ICostAllocationService
	contractDAL              contractDalIface.ContractFirestore
}

func NewBillingTableManagementService(loggerProvider logger.Provider, conn *connection.Connection) *BillingTableManagementService {
	reportStatusService, err := reportStatus.NewReportStatusesService(loggerProvider, conn)
	if err != nil {
		return nil
	}

	gkeCostAllocation, err := gkeCostAllocationService.NewCostAllocationService(loggerProvider, conn)
	if err != nil {
		return nil
	}

	metadataDAL := metadataDAL.NewMetadataFirestoreWithClient(conn.Firestore)

	return &BillingTableManagementService{
		loggerProvider,
		conn,
		reportStatusService,
		billingUpdateDal.NewBillingUpdateFirestoreWithClient(conn.Firestore),
		billingUpdateDal.NewBillingImportStatusWithClient(conn.Firestore),
		cspDAL.NewCSPFirestoreWithClient(loggerProvider, conn.Firestore),
		discountsService.NewDiscountsService(conn),
		gcpMetadata.NewGCPMetadataService(loggerProvider, conn, metadataDAL),
		cspTaskReporterService.New(loggerProvider),
		assetsDal.NewAssetsFirestoreWithClient(conn.Firestore),
		gkeCostAllocation,
		contractDal.NewContractFirestoreWithClient(conn.Firestore),
	}
}

func (s *BillingTableManagementService) StandaloneBillingUpdateEvents(ctx context.Context) error {
	l := s.loggerProvider(ctx)

	bq, ok := domainOrigin.Bigquery(ctx, s.conn)
	if !ok {
		l.Warningf("no bq client found for origin, using default")
	}

	events, err := s.billingUpdate.ListBillingUpdateEvents(ctx)
	if err != nil {
		l.Errorf("failed to retrieve gcp standalone events error: %s", err)
		return err
	}

	uniqueBillingAccountEvents := make(map[string][]*standaloneDomain.BillingEvent)

	for _, e := range events {
		// we only care about onboarding and backfill events for now
		if e.EventType == standaloneDomain.BillingUpdateEventOnboarding || e.EventType == standaloneDomain.BillingUpdateEventBackfill {
			uniqueBillingAccountEvents[e.BillingAccountID] = append(uniqueBillingAccountEvents[e.BillingAccountID], e)
		}
	}

	product := common.Assets.GoogleCloudStandalone

	data := tableMgmtDomain.BigQueryTableUpdateRequest{
		AllPartitions:        true,
		Clustering:           domain.GetBillingTableClustering(),
		DefaultProjectID:     domain.BillingProjectProd,
		DefaultDatasetID:     domain.BillingDataset,
		DestinationProjectID: domain.BillingProjectProd,
		IsStandalone:         true,
		WaitTillDone:         true,
		WriteDisposition:     bigquery.WriteTruncate,

		House:   common.HouseAdoption,
		Feature: common.FeatureCloudAnalytics,
		Module:  common.ModuleTableManagement,
	}

mainloop:
	for billingAccountID, events := range uniqueBillingAccountEvents {
		assetID := fmt.Sprintf("%s-%s", product, billingAccountID)

		if _, err := s.assets.Get(ctx, assetID); err != nil {
			l.Errorf("failed to fetch gcp standalone asset %s error: %s", assetID, err)
			continue
		}

		data.ConfigJobID = fmt.Sprintf("cloud_analytics_%s-%s", product, billingAccountID)
		data.DestinationDatasetID = domain.GetCustomerBillingDataset(billingAccountID)
		data.DestinationTableName = domain.GetCustomerBillingTable(billingAccountID, "")

		l.Infof("creating gcp standalone customer table update task for billing account id %s", billingAccountID)

		if err := s.updateBillingAccountTable(ctx, billingAccountID, &data); err != nil {
			l.Errorf("failed to update gcp standalone customer table for billing account id %s error: %s", billingAccountID, err)
			continue
		}

		for _, interval := range query.GetAggregatedTableIntervals() {
			data.DestinationTableName = domain.GetCustomerBillingTable(billingAccountID, interval)

			query := domain.GetAggregatedQuery(true, billingAccountID, false, interval)

			if err := service.RunBillingTableUpdateQuery(ctx, bq, query, &data); err != nil {
				l.Errorf("failed to update gcp standalone aggregated table %s for billing account id %s error: %s", interval, billingAccountID, err)
				continue mainloop
			}
		}

		if err := s.assets.UpdateAsset(ctx, assetID, []firestore.Update{
			{
				Path:  "standaloneProperties.billingReady",
				Value: true,
			},
		}); err != nil {
			l.Errorf("failed to update gcp standalone asset %s error: %s", assetID, err)
		}

		var metadataUpdated bool

		for _, event := range events {
			if err := s.billingUpdate.UpdateTimeCompleted(ctx, event.ID()); err != nil {
				return err
			}

			if !metadataUpdated && event.EventType == standaloneDomain.BillingUpdateEventOnboarding {
				_ = s.gkeCostAllocationService.ScheduleInitStandaloneAccounts(ctx, []string{billingAccountID})

				if err := s.gcpMetadataService.UpdateBillingAccountMetadata(ctx, assetID, billingAccountID, nil); err != nil {
					return err
				}
				metadataUpdated = true
			}
		}
	}

	return nil
}
