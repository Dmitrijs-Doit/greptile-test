package service

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	gcpTableMgmtDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/googlecloud/billingtablemgmt/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/googlecloud"
	"github.com/doitintl/hello/scheduled-tasks/presentations/domain"
	"github.com/doitintl/hello/scheduled-tasks/presentations/log"
)

func (p *PresentationService) createPresentationModeGCPAssets(
	ctx context.Context,
	customerRef *firestore.DocumentRef,
	entityRef *firestore.DocumentRef,
	billingAccountID string,
	projectIDs []string,
) error {
	logger := p.Logger(ctx)
	logger.SetLabel(log.LabelPresentationUpdateStage.String(), "assets")
	logger.Infof("GCP assets update for billing account: %s", billingAccountID)

	fs := p.conn.Firestore(ctx)
	batch := doitFirestore.NewBatchProviderWithClient(fs, 250).Provide(ctx)

	billingAccountDisplayName := strings.ToLower(customerRef.ID) + ".doit.com"

	billingAccountAsset := googlecloud.Asset{
		BaseAsset: common.BaseAsset{
			AssetType: common.Assets.GoogleCloud,
			Entity:    entityRef,
			Customer:  customerRef,
		},
		Properties: &googlecloud.AssetProperties{
			BillingAccountID: billingAccountID,
			DisplayName:      billingAccountDisplayName,
			Admins:           []string{"admin@" + billingAccountDisplayName},
			Projects:         projectIDs,
			NumProjects:      int64(len(projectIDs)),
		},
		StandaloneProperties: nil,
		Snapshot:             nil,
	}

	billingAccountAssetSettings := common.AssetSettings{
		BaseAsset: common.BaseAsset{
			AssetType: billingAccountAsset.AssetType,
			Entity:    entityRef,
			Customer:  customerRef,
		},
	}

	billingAccountAssetDocID := strings.Join([]string{common.Assets.GoogleCloud, billingAccountID}, "-")

	if err := p.setAsset(ctx, batch,
		billingAccountAssetDocID,
		&billingAccountAsset,
		&billingAccountAssetSettings,
	); err != nil {
		logger.Errorf(batchSetErrorMessage, err)
		return err
	}

	for _, projectID := range projectIDs {
		projectAsset := googlecloud.ProjectAsset{
			AssetType: common.Assets.GoogleCloudProject,
			Entity:    entityRef,
			Customer:  customerRef,
			Properties: &googlecloud.ProjectAssetProperties{
				BillingAccountID: billingAccountID,
				ProjectID:        projectID,
			},
		}

		projectAssetSettings := common.AssetSettings{
			BaseAsset: common.BaseAsset{
				AssetType: projectAsset.AssetType,
				Entity:    entityRef,
				Customer:  customerRef,
			},
		}

		if err := p.setAsset(ctx, batch,
			strings.Join([]string{common.Assets.GoogleCloudProject, projectID, customerRef.ID}, "-"),
			&projectAsset,
			&projectAssetSettings,
		); err != nil {
			logger.Errorf(batchSetErrorMessage, err)
			return err
		}
	}

	err := batch.Commit(ctx)

	return err
}

func (p *PresentationService) setAsset(
	ctx context.Context,
	batch iface.Batch,
	assetDocID string,
	asset interface{},
	assetSettings *common.AssetSettings,
) error {
	assetRef := p.assetsDAL.GetRef(ctx, assetDocID)
	assetSettingsRef := p.assetSettingsDAL.GetRef(ctx, assetDocID)

	if err := batch.Set(ctx, assetSettingsRef, assetSettings); err != nil {
		return err
	}

	if err := batch.Set(ctx, assetRef, asset); err != nil {
		return err
	}

	if err := batch.Set(ctx, assetRef.Collection("assetMetadata").Doc("metadata"), map[string]interface{}{
		"lastUpdated": firestore.ServerTimestamp,
		"type":        assetSettings.AssetType,
	}); err != nil {
		return err
	}

	return nil
}

func (p *PresentationService) getGcpProjectIDsFromBillingData(ctx *gin.Context, customerID string) ([]string, error) {
	logger := p.Logger(ctx)
	bq := p.conn.Bigquery(ctx)
	billingAccountID := domain.HashCustomerIdIntoABillingAccountId(customerID)

	dataset := bq.DatasetInProject(gcpTableMgmtDomain.GetBillingProject(), gcpTableMgmtDomain.GetCustomerBillingDataset(billingAccountID))

	if _, err := dataset.Metadata(ctx); err != nil {
		return nil, err
	}

	demoTable := dataset.Table(gcpTableMgmtDomain.GetCustomerBillingTable(billingAccountID, ""))

	replacer := strings.NewReplacer(
		"{table}", strings.Join([]string{demoTable.ProjectID, demoTable.DatasetID, demoTable.TableID}, "."),
	)

	query := getQueryWithLabels(ctx, bq, replacer.Replace(`SELECT DISTINCT(project_id) AS pid FROM {table} WHERE project_id IS NOT NULL AND export_time > "2024-01-01" AND DATETIME(export_time) > DATETIME_SUB(CURRENT_DATETIME(), INTERVAL 3 MONTH)`), customerID)

	iter, err := query.Read(ctx)
	if err != nil {
		logger.Errorln(err)
		return nil, err
	}

	projectIDs := make([]string, 0)

	for {
		var accountResultSet struct {
			ProjectID string `bigquery:"pid"`
		}

		err := iter.Next(&accountResultSet)
		if err == iterator.Done {
			break
		}

		if err != nil {
			logger.Errorln(err)
			return nil, err
		}

		projectIDs = append(projectIDs, accountResultSet.ProjectID)
	}

	return projectIDs, nil
}

func (p *PresentationService) UpdateGCPAssets(ctx *gin.Context, customerID string) error {
	customer, err := p.customersDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	if customer.PresentationMode == nil && !customer.PresentationMode.IsPredefined {
		return errors.New("customer is not a demo customer")
	}

	entities, err := p.entitiesDAL.GetCustomerEntities(ctx, customer.Snapshot.Ref)
	if err != nil {
		return err
	}

	if len(entities) == 0 {
		return errors.New("customer does not have any entity")
	}

	billingAccountID := domain.HashCustomerIdIntoABillingAccountId(customerID)

	projectIDs, err := p.getGcpProjectIDsFromBillingData(ctx, customerID)
	if err != nil {
		return err
	}

	return p.createPresentationModeGCPAssets(ctx, customer.Snapshot.Ref, entities[0].Snapshot.Ref, billingAccountID, projectIDs)
}
