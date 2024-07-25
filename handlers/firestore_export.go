package handlers

import (
	"context"
	"fmt"

	firestore "google.golang.org/api/firestore/v1"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

func FirestoreExportHandler(ctx context.Context) error {
	fsService, err := firestore.NewService(ctx)
	if err != nil {
		return err
	}

	bucket := fmt.Sprintf("gs://%s-%s", common.ProjectID, bucketSuffix)
	exportDocsRequest := firestore.GoogleFirestoreAdminV1ExportDocumentsRequest{
		CollectionIds: []string{
			"accountManagers",
			"assetSettings",
			"assetSettingsLogs",
			"assets",
			"attributions",
			"buckets",
			"cloudAnalyticsAlerts",
			"cloudAnalyticsBudgets",
			"cloudAnalyticsMetrics",
			"cloudAnalyticsAttributionGroups",
			"cloudServices",
			"commitmentContracts",
			"cloudhealthCustomers",
			"configuration",
			"customer-savings-plans",
			"customerCredits",
			"customerInvoiceAdjustments",
			"customerNotes",
			"customerOrgs",
			"customers",
			"debtAnalytics",
			"entities",
			"entitlements",
			"entityMetadata",
			"featureDemand",
			"flexibleReservedInstances",
			"flexsave-payer-configs",
			"fs-onboarding",
			"gcpMarketplaceAccounts",
			"gcpMarketplaceEntitlements",
			"gcpMarketplaceAdjustments",
			"gcpStandaloneAccounts",
			"googleCloudBillingSkus",
			"googleKubernetesEngineTables",
			"insightsResults",
			"inventory-g-suite",
			"inventory-office-365",
			"invites",
			"invoices",
			"invoicesOverdue",
			"mpaAccounts",
			"permissions",
			"rampPlans",
			"receipts",
			"roles",
			"savedReports",
			"slackChannel",
			"ticketStatistics",
			"tiers",
			"users",
			"vendorContracts",
		},
		OutputUriPrefix: bucket,
	}

	exportDocsCall := fsService.Projects.Databases.ExportDocuments("projects/"+common.ProjectID+"/databases/(default)", &exportDocsRequest)
	if _, err := exportDocsCall.Do(); err != nil {
		return err
	}

	return nil
}
