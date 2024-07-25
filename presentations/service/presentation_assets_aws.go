package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
	amazonwebservicesDomain "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	analyticsAWS "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/amazonwebservices/utils"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/presentations/log"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

const batchSetErrorMessage = "batch.Set err: %v"

func (p *PresentationService) createPresentationModeAwsAssets(
	ctx context.Context,
	customerRef *firestore.DocumentRef,
	entity *common.Entity,
	accounts []AssetWithName,
	payerAccountID string,
) {
	logger := p.Logger(ctx)
	logger.SetLabel(log.LabelPresentationUpdateStage.String(), "assets")
	logger.Infof("AWS assets update for account: %s", payerAccountID)

	fs := p.conn.Firestore(ctx)
	batch := doitFirestore.NewBatchProviderWithClient(fs, 250).Provide(ctx)

	for _, account := range accounts {
		assetDocID := strings.Join([]string{common.Assets.AmazonWebServices, account.assetID}, "-")
		assetSettingsRef := p.assetSettingsDAL.GetRef(ctx, assetDocID)
		entityRef := entity.Snapshot.Ref

		tag, err := generateAssetTagFromNumericID(account.assetID)
		if err != nil {
			logger.Errorln(err)
			continue
		}

		tags := []string{tag}

		assetSettings := pkg.AWSAssetSettings{
			BaseAsset: pkg.BaseAsset{
				AssetType: common.Assets.AmazonWebServices,
				Entity:    entityRef,
				Customer:  customerRef,
				Tags:      tags,
			},
			TimeCreated: time.Now().UTC(),
		}
		if err := batch.Set(ctx, assetSettingsRef, assetSettings); err != nil {
			logger.Errorln(err)
			continue
		}

		asset := amazonwebservices.Asset{
			AssetType: common.Assets.AmazonWebServices,
			Customer:  customerRef,
			Properties: &pkg.AWSProperties{
				AccountID:    account.assetID,
				Name:         account.assetName,
				FriendlyName: account.assetName,
				SauronRole:   true,
				OrganizationInfo: &pkg.OrganizationInfo{
					PayerAccount: &amazonwebservicesDomain.PayerAccount{
						AccountID:   payerAccountID,
						DisplayName: "presentation-mode",
					},
					Status: "ACTIVE",
					Email:  "",
				},
				CloudHealth: &pkg.CloudHealthAccountInfo{
					Status: "green",
				},
			},
			Tags:   tags,
			Entity: entityRef,
		}

		assetRef := p.assetsDAL.GetRef(ctx, assetDocID)
		if err := batch.Set(ctx, assetRef, asset); err != nil {
			logger.Errorf(batchSetErrorMessage, err)
			continue
		}

		if err := batch.Set(ctx, assetRef.Collection("assetMetadata").Doc("metadata"), map[string]interface{}{
			"lastUpdated": firestore.ServerTimestamp,
			"type":        asset.AssetType,
		}); err != nil {
			logger.Errorf(batchSetErrorMessage, err)
			continue
		}
	}

	if err := batch.Commit(ctx); err != nil {
		logger.Errorf("batch.Commit err: %v", err)
	}
}

func (p *PresentationService) getAnonymizedFlexsaveAccountIDs(ctx *gin.Context, bq *bigquery.Client, customerID string) ([]string, error) {
	flexsaveAccountIDs, err := p.flexsaveAPI.ListFlexsaveAccountsWithCache(ctx, time.Minute*30)
	if err != nil {
		return nil, err
	}

	flexsaveAccountIDsAsStrings := ""

	for idx, accountID := range flexsaveAccountIDs {
		if idx != 0 {
			flexsaveAccountIDsAsStrings += ","
		}

		flexsaveAccountIDsAsStrings += fmt.Sprintf(`"%s"`, accountID)
	}

	replacer := strings.NewReplacer(
		"{project_id_anonymizer}", createAwsProjectIDAnonymizer(customerID),
		"{flexsave_project_ids}", flexsaveAccountIDsAsStrings,
	)

	query := bq.Query(replacer.Replace(`
		{project_id_anonymizer}
		SELECT AWSProjectIdAnonymizer(pid) as fid FROM UNNEST([{flexsave_project_ids}]) as pid
	`))

	iter, err := query.Read(ctx)
	if err != nil {
		return nil, err
	}

	flexsaveIDs := make([]string, 0)

	for {
		var resultSet struct {
			FlexsaveID string `bigquery:"fid"`
		}

		err := iter.Next(&resultSet)
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		flexsaveIDs = append(flexsaveIDs, resultSet.FlexsaveID)
	}

	return flexsaveIDs, nil
}

func (p *PresentationService) getAwsAssetsFromBillingData(ctx *gin.Context, customerID string) ([]AssetWithName, error) {
	logger := p.Logger(ctx)
	bq := p.conn.Bigquery(ctx)

	flexsaveIDs, err := p.getAnonymizedFlexsaveAccountIDs(ctx, bq, customerID)
	if err != nil {
		return nil, err
	}

	source := getBqTable(bq, analyticsAWS.GetBillingProject(), customerID, analyticsAWS.GetCustomerBillingDataset, analyticsAWS.GetCustomerBillingTable)

	replacer := strings.NewReplacer(
		"{table}", source.FullTableName,
		"{demoAccountId}", awsDemoBillingAccountID,
	)

	query := getQueryWithLabels(ctx, bq, replacer.Replace(`
		SELECT project_id as projectId,
		(SELECT value FROM UNNEST(labels) WHERE key = 'env') AS env,
		(SELECT value FROM UNNEST(labels) WHERE key = 'project') AS projectName
		FROM 	{table}
		WHERE 	billing_account_id = "{demoAccountId}"
				AND project_id IS NOT NULL
				AND cost_type != "Tax"
		GROUP BY projectId, env, projectName
		ORDER BY projectId
	`), customerID)

	logger.Info(query.QueryConfig.Q)

	iter, err := query.Read(ctx)
	if err != nil {
		logger.Errorln(err)
		return nil, err
	}

	assetsWithNames := make([]AssetWithName, 0)

	for {
		var accountResultSet struct {
			ProjectID   string `bigquery:"projectId"`
			ProjectName string `bigquery:"projectName"`
			Env         string `bigquery:"env"`
		}

		err := iter.Next(&accountResultSet)
		if err == iterator.Done {
			break
		}

		if err != nil {
			logger.Errorln(err)
			return nil, err
		}

		if slice.Contains(flexsaveIDs, accountResultSet.ProjectID) {
			continue
		}

		assetName := strings.Join([]string{accountResultSet.Env, accountResultSet.ProjectName}, "-")

		hasSameAccountName := doesSliceHaveItem(assetsWithNames, func(asset AssetWithName) bool {
			return asset.assetName == assetName
		})

		if hasSameAccountName {
			assetName += "-" + accountResultSet.ProjectID[len(accountResultSet.ProjectID)-3:]
		}

		assetsWithNames = append(assetsWithNames,
			AssetWithName{
				assetID:   accountResultSet.ProjectID,
				assetName: assetName,
			},
		)
	}

	return assetsWithNames, nil
}

func (p *PresentationService) UpdateAWSAssets(ctx *gin.Context, customerID string) error {
	customer, err := p.getDemoCustomerFromID(ctx, customerID)
	if err != nil {
		return err
	}

	entitiesRef, err := p.entitiesDAL.GetCustomerEntities(ctx, customer.Snapshot.Ref)
	if err != nil {
		return err
	}

	if len(entitiesRef) == 0 {
		return errors.New("customer does not have any entity")
	}

	assetsWithNames, err := p.getAwsAssetsFromBillingData(ctx, customerID)
	if err != nil {
		return err
	}

	p.createPresentationModeAwsAssets(ctx, customer.Snapshot.Ref, entitiesRef[0], assetsWithNames, awsDemoBillingAccountID)

	return nil
}
