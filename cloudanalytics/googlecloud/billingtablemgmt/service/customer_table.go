package googlecloud

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	metadataDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	metadataTasks "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/tasks"
	tableMgmtDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/tablemanagement/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/googlecloud"
	standaloneDomain "github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/shared"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

// ScheduledBillingAccountsTableUpdate updates a customer's billing accounts tables when one of
// the contracts for the customer is updated in order to recalculate discounts.
func (s *BillingTableManagementService) ScheduledBillingAccountsTableUpdate(ctx context.Context) error {
	fs := s.conn.Firestore(ctx)

	assetPrefixLen := len(common.Assets.GoogleCloud) + 1
	billingAccountsToUpdate := make(map[string]bool)

	docSnaps, err := fs.CollectionGroup("contractUpdates").
		Where("type", "==", common.Assets.GoogleCloud).
		Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	for _, docSnap := range docSnaps {
		contractRef := docSnap.Ref.Parent.Parent

		contractDocSnap, err := contractRef.Get(ctx)
		if err != nil {
			return err
		}

		var contract common.Contract

		if err := contractDocSnap.DataTo(&contract); err != nil {
			return err
		}

		if contract.Customer == nil || contract.Entity == nil {
			continue
		}

		billingAccounts := make([]string, 0)

		if len(contract.Assets) > 0 {
			for _, assetRef := range contract.Assets {
				billingAccounts = append(billingAccounts, assetRef.ID[assetPrefixLen:])
			}
		} else {
			docSnaps, err := fs.Collection("assets").
				Where("type", "==", common.Assets.GoogleCloud).
				Where("customer", "==", contract.Customer).
				Where("entity", "==", contract.Entity).
				Select().Documents(ctx).GetAll()
			if err != nil {
				return err
			}

			for _, docSnap := range docSnaps {
				billingAccounts = append(billingAccounts, docSnap.Ref.ID[assetPrefixLen:])
			}
		}

		now := time.Now().UTC()

		for _, billingAccountID := range billingAccounts {
			if _, prs := billingAccountsToUpdate[billingAccountID]; prs {
				continue
			}

			billingAccountsToUpdate[billingAccountID] = true

			basePath := fmt.Sprintf("/tasks/analytics/google-cloud/accounts/%s", billingAccountID)

			// Recalculate all partitions for customer table
			if _, err := s.conn.CloudTaskClient.CreateAppEngineTask(ctx, (&common.CloudTaskConfig{
				Method:       cloudtaskspb.HttpMethod_POST,
				Path:         basePath + "?allPartitions=true",
				Queue:        common.TaskQueueCloudAnalyticsTablesGCP,
				ScheduleTime: common.TimeToTimestamp(now.Add(time.Minute * 30)),
			}).AppEngineConfig(nil)); err != nil {
				return err
			}

			// Reaggregate all aggregated tables and partitions for customer tables
			// (schedule after the raw table update)
			if _, err := s.conn.CloudTaskClient.CreateAppEngineTask(ctx, (&common.CloudTaskConfig{
				Method:       cloudtaskspb.HttpMethod_POST,
				Path:         basePath + "/aggregate-all?allPartitions=true",
				Queue:        common.TaskQueueCloudAnalyticsTablesGCP,
				ScheduleTime: common.TimeToTimestamp(now.Add(time.Minute * 120)),
			}).AppEngineConfig(nil)); err != nil {
				return err
			}
		}

		if _, err := docSnap.Ref.Delete(ctx); err != nil {
			return err
		}
	}

	return nil
}

// UpdateBillingAccountsTable iterates all GCP assets and create tasks that update the
// billing table, metadata and aggregated tables
func (s *BillingTableManagementService) UpdateBillingAccountsTable(ctx context.Context, input domain.UpdateBillingAccountsTableInput) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	l.Info("Starting UpdateBillingAccountsTable job")

	var (
		docSnaps                  []*firestore.DocumentSnapshot
		err                       error
		shouldUpdateMetadata      bool
		shouldUpdateTable         bool
		shouldAggregate           bool
		shouldUpdateAllPartitions bool
	)

	if input.FromDate != "" && input.FromDateNumPartitions < 0 {
		return errors.New("numPartitions must be not negative when fromDate is set")
	}

	if len(input.Assets) > 0 {
		// use to update specific accounts for testing only
		assetsToUpdate := make(map[string]bool)

		for _, assetID := range input.Assets {
			if _, prs := assetsToUpdate[assetID]; prs {
				continue
			}

			assetsToUpdate[assetID] = true

			docSnap, err := fs.Collection("assets").Doc(assetID).Get(ctx)
			if err != nil && status.Code(err) != codes.NotFound {
				return err
			}

			if docSnap.Exists() {
				docSnaps = append(docSnaps, docSnap)
			}
		}
	} else {
		docSnaps, err = fs.Collection("assets").
			Where("type", common.In, []string{
				common.Assets.GoogleCloud,
				common.Assets.GoogleCloudStandalone,
			}).
			Documents(ctx).GetAll()
		if err != nil {
			return err
		}
	}

	l.Infof("Updating %d accounts", len(docSnaps))

	switch input.Mode {
	case "metadata":
		// updates metadata for all accounts
		shouldUpdateMetadata = true
	case "aggregate":
		// aggregates all partitions for all accounts
		// does not update main tables
		shouldAggregate = true
		shouldUpdateAllPartitions = input.FromDate == ""
	case "tables":
		// updates all main tables for all accounts
		// does not update aggregated tables
		shouldUpdateTable = true
		shouldUpdateAllPartitions = input.FromDate == ""
	default:
		// default behavior will update main table and aggregated table (last partitions)
		shouldUpdateTable = true
		shouldAggregate = true
	}

	l.Infof("Should update table: %v", shouldUpdateTable)
	l.Infof("Should update metadata: %v", shouldUpdateMetadata)
	l.Infof("Should update aggregated tables: %v", shouldAggregate)

	// Sort assets so that the tasks for the newest assets are created first
	sort.Slice(docSnaps, func(i, j int) bool {
		return docSnaps[i].CreateTime.After(docSnaps[j].CreateTime)
	})

	metadataScheduleTime := time.Now().UTC()

	for _, docSnap := range docSnaps {
		var asset googlecloud.Asset
		if err := docSnap.DataTo(&asset); err != nil {
			l.Errorf("failed to convert asset %s data with error %s", docSnap.Ref.ID, err)
			continue
		}

		// skip update of test customer assets
		if slice.Contains(input.ExceptAssets, docSnap.Ref.ID) {
			l.Infof("skipping asset %s", docSnap.Ref.ID)
			continue
		}

		// skip update of standalone assets that import of data is not complete
		if asset.AssetType == common.Assets.GoogleCloudStandalone {
			billingImportStatus, err := s.billingImport.GetBillingImportStatus(ctx, asset.Customer.ID, asset.Properties.BillingAccountID)
			if err != nil {
				l.Errorf("failed to get billing import status for asset %s error: %s", docSnap.Ref.ID, err)
				continue
			}

			l.Infof("Standalone account %s billing import status %s", asset.Properties.BillingAccountID, billingImportStatus.Status)

			if billingImportStatus.Status != standaloneDomain.BillingImportStatusCompleted && billingImportStatus.Status != standaloneDomain.BillingImportStatusEnabled {
				l.Infof("did not update table for standalone asset %s billing import status %s", docSnap.Ref.ID, billingImportStatus.Status)
				continue
			}
		}

		basePath := fmt.Sprintf("/tasks/analytics/%s/accounts/%s", asset.AssetType, asset.Properties.BillingAccountID)

		// Create update metadata task
		if shouldUpdateMetadata {
			body := metadataDomain.MetadataUpdateInput{AssetID: docSnap.Ref.ID, CustomerID: asset.BaseAsset.Customer.ID}

			scheduleTime := metadataScheduleTime

			if shouldUpdateTable {
				// when also updating main table, delay the metadata task by 60 minutes
				// to allow the main table update job to complete
				scheduleTime = scheduleTime.Add(60 * time.Minute)
			}

			if err := metadataTasks.CreateUpdateGCPAccountMetadataTask(ctx, s.conn, body, asset.Properties.BillingAccountID, &scheduleTime); err != nil {
				l.Errorf("failed to create metadata update task for %s with error %s", docSnap.Ref.ID, err)
				continue
			}

			metadataScheduleTime = metadataScheduleTime.Add(1 * time.Second)
		}

		// skip update of main table for e2e test account
		if asset.Properties.BillingAccountID == common.E2ETestBillingAccountID {
			continue
		}

		addQueryParams := func(path string) string {
			if shouldUpdateAllPartitions {
				path += "?allPartitions=true"
			} else if input.FromDate != "" {
				path += "?from=" + input.FromDate

				if input.FromDateNumPartitions > 0 {
					path += "&numPartitions=" + strconv.Itoa(input.FromDateNumPartitions)
				}
			}

			return path
		}

		// Update main account table
		if shouldUpdateTable {
			path := addQueryParams(basePath)

			c := &common.CloudTaskConfig{
				Method: cloudtaskspb.HttpMethod_POST,
				Path:   path,
				Queue:  common.TaskQueueCloudAnalyticsTablesGCP,
			}

			if _, err := s.conn.CloudTaskClient.CreateAppEngineTask(ctx, c.AppEngineConfig(nil)); err != nil {
				l.Errorf("failed to create table update task for %s with error %s", docSnap.Ref.ID, err)
				continue
			}
		}

		// Create aggregate table task
		if shouldAggregate {
			path := addQueryParams(basePath + "/aggregate-all")

			c := &common.CloudTaskConfig{
				Method: cloudtaskspb.HttpMethod_POST,
				Path:   path,
				Queue:  common.TaskQueueCloudAnalyticsTablesGCP,
			}

			if shouldUpdateTable {
				// when also updating main table, delay the aggregation by 90 minutes
				// to allow the main table update job to complete
				c.ScheduleTime = common.TimeToTimestamp(time.Now().Add(90 * time.Minute))
			}

			if _, err := s.conn.CloudTaskClient.CreateAppEngineTask(ctx, c.AppEngineConfig(nil)); err != nil {
				l.Errorf("failed to create table aggregate task for %s with error %s", docSnap.Ref.ID, err)
			}
		}
	}

	return nil
}

func (s *BillingTableManagementService) UpdateBillingAccountTable(
	ctx context.Context,
	uri string,
	billingAccountID string,
	allPartitions bool,
	refreshMetadata bool,
	assetType string,
	fromDate string,
	numPartitions int,
) error {
	product := common.Assets.GoogleCloud

	isStandalone := strings.Contains(uri, common.Assets.GoogleCloudStandalone)
	if isStandalone {
		product = common.Assets.GoogleCloudStandalone
	}

	suffix := strings.Replace(billingAccountID, "-", "_", -1)

	data := tableMgmtDomain.BigQueryTableUpdateRequest{
		ConfigJobID:           fmt.Sprintf("cloud_analytics_%s-%s", product, billingAccountID),
		DefaultProjectID:      domain.BillingProjectProd,
		DefaultDatasetID:      domain.BillingDataset,
		DestinationProjectID:  domain.BillingProjectProd,
		DestinationDatasetID:  domain.GetCustomerBillingDataset(suffix),
		DestinationTableName:  domain.GetCustomerBillingTable(suffix, ""),
		WriteDisposition:      bigquery.WriteTruncate,
		AllPartitions:         allPartitions,
		FromDate:              fromDate,
		FromDateNumPartitions: numPartitions,
		Clustering:            domain.GetBillingTableClustering(),
		IsStandalone:          isStandalone,

		House:   common.HouseAdoption,
		Feature: common.FeatureCloudAnalytics,
		Module:  common.ModuleTableManagement,
	}

	// Wait till table update is done if metadata should be refreshed afterwards
	if refreshMetadata && assetType != "" {
		data.WaitTillDone = true
	}

	if err := s.updateBillingAccountTable(ctx, billingAccountID, &data); err != nil {
		return err
	}

	// Refresh report metadata
	if refreshMetadata && assetType != "" {
		assetID := fmt.Sprintf("%s-%s", assetType, billingAccountID)
		if err := s.gcpMetadataService.UpdateBillingAccountMetadata(ctx, assetID, billingAccountID, nil); err != nil {
			return err
		}
	}

	return nil
}
