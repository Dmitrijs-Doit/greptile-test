package azure

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"google.golang.org/api/iterator"

	asset_pkg "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	pkg "github.com/doitintl/hello/scheduled-tasks/azure/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/microsoft"
)

const (
	activeStatus = "Active"
	azureTaskURL = "/tasks/assets/azure-saas/%s"
)

type AzureAssetRow struct {
	SubscriptionID     string              `bigquery:"subscriptionId"`
	SubscriptionName   string              `bigquery:"subscription_name"`
	SkuID              bigquery.NullString `bigquery:"sku_id"`
	SkuDescription     bigquery.NullString `bigquery:"sku_description"`
	ProjectID          bigquery.NullString `bigquery:"project_id"`
	BillingAccount     bigquery.NullString `bigquery:"billing_account_id"`
	ResourceID         string              `bigquery:"resource_id"`
	CustomerType       bigquery.NullString `bigquery:"customer_type"`
	CustomerID         bigquery.NullString `bigquery:"customer_id"`
	BillingProfileID   bigquery.NullString `bigquery:"billing_profile_id"`
	BillingProfileName bigquery.NullString `bigquery:"billing_profile_name"`
	ProductOrderName   bigquery.NullString `bigquery:"product_order_name"`
}

// Create azure asset discovery cloud tasks jobs
func (s *AzureSaaSConsoleService) CreateAssetDiscoveryTasks(ctx context.Context) error {
	l := s.loggerProvider(ctx)

	azureConnDoc, err := s.azureSaasDal.GetAzureConnectDocs(ctx)
	if err != nil {
		l.Error("Failed to get azure connect docs", err)
		return err
	}

	for _, doc := range azureConnDoc {
		l.Infof("Creating asset discovery task for %s SubscriptionID %s", doc.CustomerID, doc.SubscriptionID)
		path := fmt.Sprintf(azureTaskURL, doc.CustomerID)

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   path,
			Queue:  common.TaskQueueAssetsMicrosoftSaas,
		}

		payload := &pkg.BillingDataConfig{
			CustomerID:     doc.CustomerID,
			Container:      doc.Container,
			Account:        doc.Account,
			ResourceGroup:  doc.ResourceGroup,
			SubscriptionID: doc.SubscriptionID,
		}
		conf := config.Config(payload)

		if _, err = s.cloudTaskClient.CreateTask(ctx, conf); err != nil {
			l.Errorf(err.Error())
			continue
		}
	}

	return nil
}

// RunAssetDiscoveryTask runs asset discovery task
func (s *AzureSaaSConsoleService) RunAssetDiscoveryTask(ctx context.Context, customerID string, azureConfig pkg.BillingDataConfig) error {
	l := s.loggerProvider(ctx)
	l.Infof("Running asset discovery task for customer %s", azureConfig.CustomerID)

	subscriptions, err := s.GetCustomerSubscriptions(ctx, azureConfig.CustomerID)
	if err != nil {
		l.Error("Failed to get customer subscriptions", customerID, err)
		return err
	}

	if len(subscriptions) == 0 {
		l.Info("No subscriptions found for customer", customerID)
		return nil
	}

	for _, subscription := range subscriptions {
		l.Infof("Running asset discovery task for %s SubscriptionID %s", customerID, subscription.SubscriptionID)

		err := s.setAzureSaasAsset(ctx, customerID, subscription)
		if err != nil {
			l.Error("Failed to run asset discovery task for", customerID, err)
			return err
		}
	}

	return nil
}

// getAssetProperties gets asset properties
func (s *AzureSaaSConsoleService) getAssetProperties(customer *common.Customer, subscription AzureAssetRow) *microsoft.AzureAssetProperties {
	skuDescription := subscription.ProductOrderName.StringVal
	if !subscription.ProductOrderName.Valid {
		skuDescription = "Azure plan"
	}

	return &microsoft.AzureAssetProperties{
		CustomerDomain: customer.PrimaryDomain,
		CustomerID:     customer.Snapshot.Ref.ID,
		Subscription: &microsoft.AzureSubscriptionProperties{
			SubscriptionID:            subscription.SubscriptionID,
			SkuID:                     "",
			SkuDescription:            skuDescription,
			CustomerDisplayName:       customer.Name,
			DisplayName:               subscription.SubscriptionName,
			SubscriptionBillingStatus: activeStatus,
		},
		Reseller: "doitintl",
	}
}

// CreateAsset creates Azure Asset on fs given Azure subscription
func (s *AzureSaaSConsoleService) setAzureSaasAsset(ctx context.Context, customerID string, subscription AzureAssetRow) error {
	l := s.loggerProvider(ctx)
	customerRef := s.customersDAL.GetRef(ctx, customerID)

	customer, err := s.customersDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	assetType := common.Assets.MicrosoftAzureStandalone
	docID := fmt.Sprintf("%s-%s", assetType, subscription.SubscriptionID)
	assetRef := s.assetsDAL.GetRef(ctx, docID)
	assetSettingsRef := s.assetSettingsDAL.GetRef(ctx, docID)

	properties := s.getAssetProperties(customer, subscription)

	asset := microsoft.AzureAsset{
		AssetType:  assetType,
		Customer:   customerRef,
		Properties: properties,
		Bucket:     nil,
		Contract:   nil,
		Entity:     nil,
		Tags:       nil,
	}

	l.Infof("Setting asset %s for customer %s", docID, customerID)

	if _, err := assetRef.Set(ctx, asset); err != nil {
		return err
	}

	assetSettings := &asset_pkg.AWSAssetSettings{
		BaseAsset: asset_pkg.BaseAsset{
			AssetType: assetType,
			Customer:  customerRef,
		},
	}

	if _, err := assetSettingsRef.Set(ctx, assetSettings); err != nil {
		return err
	}

	return nil
}

// GetCustomerSubscriptions gets customer subscriptions from billing table
func (s *AzureSaaSConsoleService) GetCustomerSubscriptions(ctx context.Context, customerID string) ([]AzureAssetRow, error) {
	l := s.loggerProvider(ctx)
	l.Infof("Getting subscriptions for customer %s", customerID)

	queryString := s.buildAssetsQuery(customerID, common.Production)
	query := s.bq.Query(queryString)
	lastDay := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	query.Parameters = []bigquery.QueryParameter{
		{Name: "lastDay", Value: lastDay},
	}
	it, err := query.Read(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to read query: %w", err)
	}

	var azureAssetRow []AzureAssetRow

	for {
		var row AzureAssetRow

		err := it.Next(&row)
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("failed to get customer subscriptions: %w", err)
		}

		azureAssetRow = append(azureAssetRow, row)
	}

	return azureAssetRow, nil
}
