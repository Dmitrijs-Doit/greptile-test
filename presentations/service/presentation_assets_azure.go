package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"

	doitFirestore "github.com/doitintl/firestore"
	analyticsAzure "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/microsoftazure"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/microsoft"
	"github.com/doitintl/hello/scheduled-tasks/presentations/domain"
	"github.com/doitintl/hello/scheduled-tasks/presentations/log"
)

type azureSubscription struct {
	SubscriptionID   string `bigquery:"subscriptionId"`
	SubscriptionName string `bigquery:"subscriptionName"`
	BillingAccountID string `bigquery:"billingAccountId"`
	BillingProfileID string `bigquery:"billingProfileId"`
	TenantID         string `bigquery:"tenantId"`
}

func (p *PresentationService) createPresentationModeAzureAssets(
	ctx context.Context,
	customerRef *firestore.DocumentRef,
	entity *common.Entity,
	subscriptions []azureSubscription,
	customerID string,
) {
	logger := p.Logger(ctx)
	logger.SetLabel(log.LabelPresentationUpdateStage.String(), "assets")
	logger.Infof("Azure assets update for customer: %s", customerID)

	fs := p.conn.Firestore(ctx)
	batch := doitFirestore.NewBatchProviderWithClient(fs, doitFirestore.MaxBatchThreshold).Provide(ctx)
	assetType := common.Assets.MicrosoftAzure

	for _, subscription := range subscriptions {
		assetDocID := strings.Join([]string{assetType, subscription.SubscriptionID}, "-")
		assetSettingsRef := p.assetSettingsDAL.GetRef(ctx, assetDocID)
		entityRef := entity.Snapshot.Ref
		subscriptionNumericID := strconv.FormatInt(domain.Hash(subscription.SubscriptionID), 10)

		tag, err := generateAssetTagFromNumericID(subscriptionNumericID)
		if err != nil {
			logger.Errorln(err)
			continue
		}

		tags := []string{tag}

		assetSettings := common.AssetSettings{
			BaseAsset: common.BaseAsset{
				AssetType: assetType,
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

		providerID := fmt.Sprintf("/providers/Microsoft.Billing/billingAccounts/%s:%s_2024-01-01/",
			subscription.BillingAccountID,
			subscription.BillingAccountID)

		asset := microsoft.AzureAsset{
			AssetType: assetType,
			Customer:  customerRef,
			Properties: &microsoft.AzureAssetProperties{
				CustomerDomain: strings.ToLower(customerID) + ".onmicrosoft.com",
				CustomerID:     customerID,
				Reseller:       "doitintl.onmicrosoft.com",
				Subscription: &microsoft.AzureSubscriptionProperties{
					BillingProfileDisplayName: "DoiT International",
					BillingProfileID:          fmt.Sprintf("%s/billingProfiles/%s", providerID, subscription.BillingProfileID),
					CustomerDisplayName:       customerID,
					CustomerID:                fmt.Sprintf("%s/customers/%s", providerID, subscription.TenantID),
					DisplayName:               subscription.SubscriptionName,
					SkuDescription:            "Microsoft Azure Plan",
					SkuID:                     "0001",
					SubscriptionBillingStatus: "Active",
					SubscriptionID:            subscription.SubscriptionID,
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
			"type":        assetType,
		}); err != nil {
			logger.Errorf(batchSetErrorMessage, err)
			continue
		}
	}

	if err := batch.Commit(ctx); err != nil {
		logger.Errorf("batch.Commit err: %v", err)
	}
}

func (p *PresentationService) getAzureSubscriptionsFromBillingData(ctx *gin.Context, customerID string) ([]azureSubscription, error) {
	logger := p.Logger(ctx)
	bq := p.conn.Bigquery(ctx)

	source := getBqTable(bq, analyticsAzure.GetBillingProject(), customerID, analyticsAzure.GetCustomerBillingDataset, analyticsAzure.GetCustomerBillingTable)

	replacer := strings.NewReplacer(
		"{table}", source.FullTableName,
	)

	query := getQueryWithLabels(ctx, bq, replacer.Replace(`
		SELECT * FROM (
			SELECT 	tenant_id as tenantId,
					billing_account_id as subscriptionId,
			(SELECT value FROM UNNEST(system_labels) WHERE key = 'azure/subscription_name') AS subscriptionName,
			(SELECT value FROM UNNEST(system_labels) WHERE key = 'azure/billing_account_id') AS billingAccountId,
			(SELECT value FROM UNNEST(system_labels) WHERE key = 'azure/billing_profile_id') AS billingProfileId,
			FROM 	{table}
			GROUP BY 1,2,3,4,5)
		WHERE billingAccountId IS NOT NULL AND billingAccountId != '' AND subscriptionName IS NOT NULL
	`), customerID)

	logger.Info(query.QueryConfig.Q)

	iter, err := query.Read(ctx)
	if err != nil {
		logger.Errorln(err)
		return nil, err
	}

	azureSubscriptions := make([]azureSubscription, 0)

	for {
		var azureSubscriptionRow azureSubscription

		err := iter.Next(&azureSubscriptionRow)
		if err == iterator.Done {
			break
		}

		if err != nil {
			logger.Errorln(err)
			return nil, err
		}

		azureSubscriptions = append(azureSubscriptions, azureSubscriptionRow)
	}

	return azureSubscriptions, nil
}

func (p *PresentationService) UpdateAzureAssets(ctx *gin.Context, customerID string) error {
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

	azureSubscriptions, err := p.getAzureSubscriptionsFromBillingData(ctx, customerID)
	if err != nil {
		return err
	}

	p.createPresentationModeAzureAssets(ctx, customer.Snapshot.Ref, entitiesRef[0], azureSubscriptions, customerID)

	return nil
}
