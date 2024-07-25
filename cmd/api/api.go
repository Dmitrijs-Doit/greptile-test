package api

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/algolia/algoliahandlers"
	awsHandlers "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/handlers"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/handler"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/service"
	plesHandler "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/ples/handlers"
	authHandlers "github.com/doitintl/hello/scheduled-tasks/auth/handlers"
	avaEmbeddingsHandler "github.com/doitintl/hello/scheduled-tasks/ava/handlers"
	awsMarketplace "github.com/doitintl/hello/scheduled-tasks/aws-marketplace/handlers"
	azureHandler "github.com/doitintl/hello/scheduled-tasks/azure/handler"
	billingExplainerHandlers "github.com/doitintl/hello/scheduled-tasks/billing-explainer/handlers"
	bqLensBackfillHandler "github.com/doitintl/hello/scheduled-tasks/bq-lens/backfill/handlers"
	bqLensBackfillService "github.com/doitintl/hello/scheduled-tasks/bq-lens/backfill/service"
	bqLensDiscoveryHandlers "github.com/doitintl/hello/scheduled-tasks/bq-lens/discovery/handlers"
	bqLensOnboardHandlers "github.com/doitintl/hello/scheduled-tasks/bq-lens/onboard/handlers"
	bqLensOptimizerHandlers "github.com/doitintl/hello/scheduled-tasks/bq-lens/optimizer/handlers"
	bqLensPricebookHandler "github.com/doitintl/hello/scheduled-tasks/bq-lens/pricebook/handler"
	alerts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/handlers"
	attributionGroups "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/handlers"
	attributions "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/handlers"
	budgetsHandlers "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/handlers"
	dashboardSubscription "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/dashboardsubscription/handlers"
	datahubHandlers "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/datahub/handlers"
	billingDataExportHandlers "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/export/handlers"
	analyticsMetadata "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/handlers"
	metricsHandlers "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/handlers"
	analyticsAzure "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/microsoftazure/handlers"
	splitting "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/post-processing/splitting/handlers"
	reportsHandlers "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/handlers"
	reportTemplatesHandlers "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/templatelibrary/handlers"
	"github.com/doitintl/hello/scheduled-tasks/cmd/api/handlers"
	"github.com/doitintl/hello/scheduled-tasks/common"
	contractHandlers "github.com/doitintl/hello/scheduled-tasks/contract/handlers"
	courierHandlers "github.com/doitintl/hello/scheduled-tasks/courier/handler"
	customerCredits "github.com/doitintl/hello/scheduled-tasks/credit/handlers"
	csmEngagementHandlers "github.com/doitintl/hello/scheduled-tasks/csmengagement/handlers"
	customerHandlers "github.com/doitintl/hello/scheduled-tasks/customer/handlers"
	publicDashboardHandlers "github.com/doitintl/hello/scheduled-tasks/dashboard/publicdashboards/handlers"
	entities "github.com/doitintl/hello/scheduled-tasks/entity/handlers"
	flexapi "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/flexapi/shared"
	flexsaveStandaloneAutomationGCP "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/automation/handler"
	flexsaveStandaloneGCP "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/handlers"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/mid"
	"github.com/doitintl/hello/scheduled-tasks/framework/mid/permissions"
	permissionsDomain "github.com/doitintl/hello/scheduled-tasks/framework/mid/permissions/domain"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	invoicingFSSA "github.com/doitintl/hello/scheduled-tasks/invoicing/flexsave/handlers"
	invoicingHandlers "github.com/doitintl/hello/scheduled-tasks/invoicing/handlers"
	knownissues "github.com/doitintl/hello/scheduled-tasks/knownissues/handlers"
	labels "github.com/doitintl/hello/scheduled-tasks/labels/handlers"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	looker "github.com/doitintl/hello/scheduled-tasks/looker/handlers"
	marketplace "github.com/doitintl/hello/scheduled-tasks/marketplace/handlers"
	microsoftLicensesHandlers "github.com/doitintl/hello/scheduled-tasks/microsoft/license/handlers"
	perksHandlers "github.com/doitintl/hello/scheduled-tasks/perks/handlers"
	presentationHandlers "github.com/doitintl/hello/scheduled-tasks/presentations/handlers"
	stripe "github.com/doitintl/hello/scheduled-tasks/stripe/handlers"
	supportHandlers "github.com/doitintl/hello/scheduled-tasks/support/handlers"
	webhookSubscriptions "github.com/doitintl/hello/scheduled-tasks/zapier/handlers"
	"github.com/doitintl/mixpanel"
	tiersService "github.com/doitintl/tiers/service"
)

// API constructs an api with the needed functionality.
type API struct {
	shutdown chan os.Signal
	log      *logger.Logging
	conn     *connection.Connection
}

func NewAPI(shutdown chan os.Signal, logging *logger.Logging, conn *connection.Connection) *API {
	return &API{
		shutdown,
		logging,
		conn,
	}
}

// Build builds the api endpoints with the needed middlewares, and returns http.Handler interface.
func (a *API) Build() http.Handler {
	loggerProvider := logger.FromContext
	detailedLoggerProvider := logger.DetailedLoggerFromContext

	permissionsService := permissions.NewService(a.conn)

	backgroundContext := context.Background()

	// Construct the web.App which holds all routes as well as common Middleware.
	app := web.NewApp(a.shutdown, a.conn, mid.Logger(), mid.Errors(), mid.Panics(), mid.Sentry())

	// Keep Flexsave handler at higher scope until we move it under shared API namespace
	flexsaveSageMaker := handlers.NewFlexsaveSageMaker(loggerProvider, a.conn)
	flexsaveRDS := handlers.NewFlexsaveRDS(loggerProvider, a.conn)
	flexsaveAWS := handlers.NewFlexSaveAWS(loggerProvider, a.conn)
	flexsaveMonitoring := handlers.NewFlexsaveMonitoring(loggerProvider, a.conn)
	flexsaveGCP := handlers.NewFlexSaveGCP(loggerProvider, a.conn)
	assets := handlers.NewAssetHandler(loggerProvider, a.conn)
	rampPlan := handlers.NewRampPlan(a.log, a.conn)
	googleCloud := handlers.NewGoogleCloud(loggerProvider, a.conn)
	invoicing := handlers.NewInvoicing(a.log, a.conn)
	invoicingAnalyticsData := invoicingHandlers.NewInvoicingDataAnalytics(a.conn)
	cloudConnect := handlers.NewCloudConnect(loggerProvider, a.conn)
	partnerSales := handlers.NewPartnerSales(loggerProvider, a.conn)
	digest := handlers.NewDigest(loggerProvider, a.conn)
	fixer := handlers.NewFixer(loggerProvider, a.conn)
	cloudAnalytics := handlers.NewCloudAnalytics(loggerProvider, a.conn)
	costAnomaly := handlers.NewCostAnomaly(cloudAnalytics)
	mixpanelHandler := handlers.NewMixpanel(a.log, a.conn)
	stripe := stripe.NewStripe(loggerProvider, a.conn)
	apiV1 := handlers.NewAPIv1(backgroundContext, loggerProvider, a.conn)
	aws := awsHandlers.NewAWS(loggerProvider, a.conn)
	awsAccountAccess := awsHandlers.NewAccountAccess(loggerProvider, a.conn)
	awsServiceCatalog := awsHandlers.NewServiceCatalog(loggerProvider, a.conn)
	masterPayerAccount := handlers.NewMPA(loggerProvider, a.conn)
	awsGeneratedAccounts := handlers.NewAwsGeneratedAccounts(loggerProvider, a.conn)
	credit := customerCredits.NewCredits(loggerProvider, a.conn)
	csmEngagement := csmEngagementHandlers.NewHandler(loggerProvider, a.conn)
	awsStandaloneHandler := handlers.NewAwsStandaloneHandler(loggerProvider, a.conn, a.log)
	eksMetricsHandler := handlers.NewEksMetricsHandler(loggerProvider, a.conn, a.log)
	gcpStandaloneHandler := handlers.NewGcpStandaloneHandler(loggerProvider, a.conn, a.log)
	gcpStandaloneBillingHandler := flexsaveStandaloneGCP.NewFlexsaveStandaloneGCPBilling(detailedLoggerProvider, a.conn)
	gcpStandaloneAutomationBillingHandler := flexsaveStandaloneAutomationGCP.NewAutomationFS_SA_Billing(detailedLoggerProvider, a.conn, a.log)
	gcpSaaSConsoleHandler := handlers.NewGCPSaaSConsoleHandler(loggerProvider, a.conn, a.log)
	awsSaaSConsoleHandler := handlers.NewAWSSaaSConsoleHandler(loggerProvider, a.conn, a.log)
	azureSaaSConsoleHandler := handlers.NewAzureSaaSConsoleHandler(loggerProvider, a.conn, a.log)
	saasConsoleHandler := handlers.NewSaaSConsoleHandler(loggerProvider, a.conn)
	analyticsAlerts := alerts.NewAnalyticsAlerts(backgroundContext, loggerProvider, a.conn)
	analyticsBudgets := budgetsHandlers.NewAnalyticsBudgets(loggerProvider, a.conn)
	auth := authHandlers.NewAuth(loggerProvider, a.conn)
	ssoProviders := handlers.NewSsoProviders(a.log, a.conn)
	salesforce := handlers.NewSalesforce(a.log, a.conn)
	AWSResoldCache := handlers.NewAWSResoldCache(loggerProvider, a.conn)
	microsoftLicensesHandler := microsoftLicensesHandlers.NewLicenseHandler(a.log, a.conn)
	costAllocation := handlers.NewCostAllocation(loggerProvider, a.conn)
	marketplace := marketplace.NewMarketplace(loggerProvider, a.conn, marketplace.TopicHandlerProvider)
	awsMp := awsMarketplace.NewMarketplaceAWS(loggerProvider, a.conn)
	spot0Costs := handlers.NewSpotZeroCosts(loggerProvider, a.conn)
	spot0fbod := handlers.NewSpotZeroFbod(loggerProvider, a.conn)
	spotScalingEmail := handlers.NewASGCustomerEmailHandler(loggerProvider, a.conn)
	notifier := service.NewNotifierService(loggerProvider)
	accountHandler := handler.NewAccountHandler(notifier, a.log)
	testConnectionHandler := handlers.NewTestConnectionHandler(detailedLoggerProvider, a.conn)
	awsAssetsSupportHandler := handlers.NewAWSAssetsSupportHandler(a.log, a.conn)
	looker := looker.NewLookerAssetsHandler(a.log, a.conn)
	onePassword := handlers.New(loggerProvider, a.conn)
	perksHandlers := perksHandlers.NewPerkHandler(a.log, a.conn)
	analyticsAttributionGroups := attributionGroups.NewAnalyticsAttributionGroups(backgroundContext, loggerProvider, a.conn)
	analyticsAttributions := attributions.NewAttributions(backgroundContext, loggerProvider, a.conn)
	reportHandler := reportsHandlers.NewReport(backgroundContext, loggerProvider, a.conn)
	reportTemplateHandler := reportTemplatesHandlers.NewReportTemplate(loggerProvider, a.conn, backgroundContext)
	presentationHandler := presentationHandlers.NewPresentation(loggerProvider, a.conn)
	analyticsMetadataHandler := analyticsMetadata.NewAnalyticsMetadata(backgroundContext, loggerProvider, a.conn)
	priorityHandler := handlers.NewPriority(loggerProvider, a.conn)
	slack := handlers.NewSlack(loggerProvider, a.conn)
	jira := handlers.NewJira(backgroundContext, loggerProvider, a.conn)
	awsSharedPayersCreditsHandler := invoicingHandlers.NewSharedPayersCreditsHandler(a.log, a.conn)
	algolia := algoliahandlers.NewAlgolia(loggerProvider, a.conn)
	dashboardSubscriptionHandler := dashboardSubscription.NewDashboardSubscription(loggerProvider, a.conn)
	supportHandlers := supportHandlers.NewSupport(loggerProvider, a.conn)
	splittingHandler := splitting.NewSplitting(loggerProvider)
	entitiesHandler := entities.NewEntities(loggerProvider, a.conn)
	webhookSubscriptionHandlers := webhookSubscriptions.NewWebhookHandler(loggerProvider, a.conn)
	labelsHandler := labels.NewLabels(loggerProvider, a.conn)
	analyticsMicrosoftAzure := analyticsAzure.NewAnalyticsMicrosoftAzure(loggerProvider, a.conn)
	awsOpsPage := handlers.NewHandler(loggerProvider, a.conn)
	metricsHandler := metricsHandlers.NewMetric(loggerProvider, a.conn)
	billingExplainerHandler := billingExplainerHandlers.NewBillingExplainerHandler(loggerProvider, a.conn)
	billingDataExportHandler := billingDataExportHandlers.NewBillingExportHandler(loggerProvider, a.conn)
	bqLensDiscoveryHandler := bqLensDiscoveryHandlers.NewTableDiscovery(loggerProvider, a.conn)
	bqLensOptimizerHandler := bqLensOptimizerHandlers.NewOptimizer(backgroundContext, loggerProvider, a.conn)
	bqLensOnboardHandler := bqLensOnboardHandlers.NewOnboardHandler(loggerProvider, a.conn)
	backfillScheduler := bqLensBackfillService.NewBackfillScheduler(loggerProvider, a.conn)
	backfillService := bqLensBackfillService.NewBackfillService(loggerProvider, a.conn)
	bqLensBackfillHandler := bqLensBackfillHandler.NewBackfillHandler(loggerProvider, backfillScheduler, backfillService)
	bqLensPricebookHandler := bqLensPricebookHandler.NewPricebook(loggerProvider, a.conn)
	datahubHandler := datahubHandlers.NewDataHub(loggerProvider, a.conn)

	azure := azureHandler.NewHandler(loggerProvider, a.conn)
	tiers := handlers.NewTiersHandler(loggerProvider, a.conn)
	tierService := tiersService.NewTiersService(a.conn.Firestore)
	contractHandler := contractHandlers.NewContractHandler(loggerProvider, a.conn)
	publicdashboardsHandler := publicDashboardHandlers.NewDashboard(loggerProvider, a.conn)
	ples := plesHandler.NewPLES(loggerProvider, a.conn)

	hasEntitlement := mid.HasEntitlementFunc(tierService)

	flexapiTokenSource, err := flexapi.GetTokenSource(backgroundContext)
	if err != nil {
		panic(err)
	}

	flexapiProxyService := handlers.NewProxyService(flexapi.GetFlexAPIURL(), flexapiTokenSource)
	flexapiProxyHandler := handlers.MethodBasedPermissionProxyHandler(flexapiProxyService, a.conn)

	doitRoleFlexsaveAdmin := mid.AuthDoitEmployeeRole(a.conn, permissionsDomain.DoitRoleFlexsaveAdmin)
	doitRoleCustomerSettingsAdmin := mid.AuthDoitEmployeeRole(a.conn, permissionsDomain.DoitRoleCustomerSettingsAdmin)
	doitRoleAwsAccountGenerator := mid.AuthDoitEmployeeRole(a.conn, permissionsDomain.DoitRoleAwsAccountGenerator)
	doitRoleCustomerTieringAdmin := mid.AuthDoitEmployeeRole(a.conn, permissionsDomain.DoitRoleCustomerTieringAdmin)

	customerHandler := customerHandlers.NewCustomer(loggerProvider, a.conn)

	courierHandler := courierHandlers.NewCourier(backgroundContext, loggerProvider, a.conn)
	avaHandler := avaEmbeddingsHandler.NewAvaEmbeddingsHandler(backgroundContext, loggerProvider, a.conn)

	if common.IsLocalhost {
		scripts := handlers.NewScripts(loggerProvider, a.conn)
		app.Post("/scripts/:scriptName", scripts.HandleScript)
	}

	app.Get("/boom", handlers.Boom)

	// SCHEDULED OR CLOUD TASKS
	tasksGroup := web.NewGroup(app, "/tasks",
		mid.AuthServiceAccount(mid.GetAllowedCloudJobsEmails()),
		mid.AddDefaultLoggerLabels(),
	)
	{
		tasks := handlers.NewTasks(loggerProvider, a.conn)
		tasksGroup.Get("/firebase-auth", tasks.RevokeRefreshTokens)
		tasksGroup.Get("/assets-digest", assets.AssetsDigestHandler)
		tasksGroup.Get("/customer-assets", customerHandler.SetCustomerAssetTypes)
		tasksGroup.Post("/assign-billing-profiles", handlers.AssignAllBillingProfiles)
		tasksGroup.Get("/exchange-rates", fixer.SyncHandler)
		tasksGroup.Get("/segment/all-customers", customerHandler.UpdateAllCustomersSegment)
		tasksGroup.Post("/segment/:customerID", customerHandler.UpdateCustomerSegment)

		salesforceGroup := tasksGroup.NewSubgroup("/salesforce")
		{
			salesforceGroup.Get("/sync", salesforce.SyncHandler)
			salesforceGroup.Get("/customer/:customerID", salesforce.SyncCustomer)
		}

		avaGroup := tasksGroup.NewSubgroup("/ava")
		{
			avaGroup.Post("/upsert-firestore-doc-embeddings", avaHandler.UpsertFirestoreDocumentEmbeddings)
			avaGroup.Post("/upsert-preset-doc-embeddings", avaHandler.UpsertPresetAttributionsEmbeddings)
		}

		csmEngagementGroup := tasksGroup.NewSubgroup("/csm-engagement")
		{
			csmEngagementGroup.Post("/send-attribution-emails", csmEngagement.SendAttributionEmails)
			csmEngagementGroup.Post("/send-no-attributions-emails", csmEngagement.SendNoAttributionsEmails)
			csmEngagementGroup.Post("/send-no-customer-engagement-notifications", csmEngagement.SendNoCustomerEngagementNotifications)
		}

		supportSync := handlers.NewSupportSync(loggerProvider, a.conn)
		tasksGroup.Get("/support-sync", supportSync.Sync)

		ripplingHandler := handlers.NewRipplingHandler(loggerProvider, a.conn)
		ripplingGroup := tasksGroup.NewSubgroup("/rippling")
		{
			ripplingGroup.Get("/sync-account-managers", ripplingHandler.SyncAccountManagers)
			ripplingGroup.Get("/sync-fsm-role", ripplingHandler.SyncFieldSalesManagerRole)
			ripplingGroup.Get("/terminated", ripplingHandler.GetTerminated)
		}

		presentationGroup := tasksGroup.NewSubgroup("/presentation")
		{
			presentationGroup.Post("/scheduler/sync-billing-data", presentationHandler.SyncCustomersBillingData)
			presentationGroup.Post("/scheduler/update-gcp-billing-data", presentationHandler.UpdateGCPBillingData)
			presentationGroup.Post("/scheduler/update-aws-billing-data", presentationHandler.UpdateAWSBillingData)
			presentationGroup.Post("/scheduler/update-azure-billing-data", presentationHandler.UpdateAzureBillingData)
			presentationGroup.Post("/scheduler/aggregate-gcp-billing-data", presentationHandler.AggregateBillingDataGCP)
			presentationGroup.Post("/scheduler/aggregate-aws-billing-data", presentationHandler.AggregateBillingDataAWS)
			presentationGroup.Post("/scheduler/aggregate-azure-billing-data", presentationHandler.AggregateBillingDataAzure)
			presentationGroup.Post("/scheduler/update-gcp-assets-metadata", presentationHandler.UpdateAssetsMetadataGCP)
			presentationGroup.Post("/scheduler/update-aws-assets-metadata", presentationHandler.UpdateAssetsMetadataAWS)
			presentationGroup.Post("/scheduler/update-azure-assets-metadata", presentationHandler.UpdateAssetsMetadataAzure)
			presentationGroup.Post("/scheduler/update-flexsave-aws-savings", presentationHandler.UpdateFlexsaveAWSSavings)
			presentationGroup.Post("/scheduler/update-eks-billing", presentationHandler.UpdateEKSBillingData)
			presentationGroup.Post("/scheduler/update-widgets", presentationHandler.UpdateWidgets)
			presentationGroup.Post("/scheduler/copy-bq-lens-data", presentationHandler.CopyBQLensDataToCustomers)
			presentationGroup.Post("/scheduler/copy-spot-scaling-data", presentationHandler.CopySpotScalingDataToCustomers)
			presentationGroup.Post("/scheduler/copy-cost-anomalies", presentationHandler.CopyCostAnomaliesToCustomers)
		}

		tasksGroup.Get("/cloudhealth/customers", handlers.SyncCloudhealthCustomers)

		mpaGroup := tasksGroup.NewSubgroup("/amazon-web-services/master-payer-accounts")
		{
			mpaGroup.Post("/google-group/create", masterPayerAccount.CreateGoogleGroup)
			mpaGroup.Post("/portfolio-shares-sync-state", awsServiceCatalog.SyncStateAllRegions)
		}

		flexsaveSageMakerGroup := tasksGroup.NewSubgroup("/flexsave-sagemaker")
		{
			flexsaveSageMakerGroup.Get("/run-cache/:customerID", flexsaveSageMaker.Cache)
		}

		flexsaveRDSGroup := tasksGroup.NewSubgroup("/flexsave-rds")
		{
			flexsaveRDSGroup.Get("/run-cache/:customerID", flexsaveRDS.Cache)
		}

		flexSaveGroup := tasksGroup.NewSubgroup("/flex-ri")
		{
			flexSaveGroup.Get("/orders/flexsave", flexsaveAWS.UpdateFlexSaveAutopilotOrdersHandler)
			flexSaveGroup.Get("/orders/flexsave/customer/:customerID", flexsaveAWS.UpdateFlexSaveAutopilotOrdersByCustomerHandler)
			flexSaveGroup.Get("/orders/flexsave/create", flexsaveAWS.CreateFlexSaveOrdersHandler)
			flexSaveGroup.Get("/import-pricing", flexsaveAWS.UpdateInstancesPricingHandler)
			flexSaveGroup.Get("/import-potential", flexsaveAWS.FanoutImportPotentialHandler)
			flexSaveGroup.Get("/import-potential/customer/:customerID", flexsaveAWS.ImportPotentialForCustomerHandler)
			flexSaveGroup.Post("/backfill-savings/customer/:customerID", flexsaveAWS.BackfillFlexsaveInvoiceAdjustment)
			flexSaveGroup.Post("/detect-savings-discrepancies", flexsaveMonitoring.DetectSharedPayerSavingsDiscrepancies)

			flexSaveGroup.Get("/cache/:customerID", AWSResoldCache.CreateForSingleCustomer)
			flexSaveGroup.Get("/cache", AWSResoldCache.CreateCacheForAllCustomers)
			flexSaveGroup.Get("/savings-plans-cache/:customerID", AWSResoldCache.CustomerSavingsPlansCache)
			flexSaveGroup.Get("/savings-plans-cache", AWSResoldCache.CreateSavingsPlansCacheForAllCustomers)
			flexSaveGroup.Get("/standalone/cache/all", awsStandaloneHandler.FanoutFSCacheDataForCustomersHandler)
			flexSaveGroup.Post("/standalone/cache/customer/:customerID", awsStandaloneHandler.UpdateStandaloneCustomerSpendSummaryHandler)

			flexSaveGroup.Post("/update-flexsave-statuses", AWSResoldCache.UpdateCustomerStatuses)
		}

		flexSaveSAGroup := tasksGroup.NewSubgroup("/flexsave-standalone")
		{
			gcpGroup := flexSaveSAGroup.NewSubgroup("/google-cloud")
			{
				billing := gcpGroup.NewSubgroup("/billing")
				{
					billing.Delete("", gcpStandaloneBillingHandler.RemoveAll)
					billing.Post("/onboarding", gcpStandaloneBillingHandler.Onboaring)
					billing.Post("/offboarding", gcpStandaloneBillingHandler.RemoveBilling)
					billing.Post("/alternative", gcpStandaloneBillingHandler.RunAlternativeManager)
					billing.Post("/sanity", gcpStandaloneBillingHandler.RunSanity)
					internal := billing.NewSubgroup("/internal")
					{
						internal.Post("/tasks", gcpStandaloneBillingHandler.RunInternalTask)
						internal.Post("/manager", gcpStandaloneBillingHandler.RunInternalManager)
					}

					external := billing.NewSubgroup("/external")
					{
						external.Post("/manager", gcpStandaloneBillingHandler.RunExternalManager)
						external.Post("/tasks/to-bucket", gcpStandaloneBillingHandler.RunToBucketExternalTask)
						external.Post("/tasks/from-bucket", gcpStandaloneBillingHandler.RunFromBucketExternalTask)
					}
					billing.Get("/monitor", gcpStandaloneBillingHandler.RunMonitor)
					billing.Get("/monitor/:billingAccountId", gcpStandaloneBillingHandler.RunMonitorBillingAccount)

					billing.Get("/rows_validate", gcpStandaloneBillingHandler.ValidateRows)
					billing.Get("/rows_validate/:billingAccountId", gcpStandaloneBillingHandler.ValidateCustomerRows)

					testConnection := billing.NewSubgroup("/tests")
					{
						testConnection.Post("/test_billing_connection", testConnectionHandler.TestBillingConnection)
					}

					tableAnalytics := billing.NewSubgroup("/analytics")
					{
						tableAnalytics.Post("/detailed_table_rewrites_mapping", gcpStandaloneBillingHandler.RunDetailedTableRewritesMapping)
						tableAnalytics.Post("/data_freshness_report", gcpStandaloneBillingHandler.RunDataFreshnessReport)
					}
				}

				automation := billing.NewSubgroup("/automation")
				{
					automation.Delete("", gcpStandaloneAutomationBillingHandler.ResetAllAutomation)
					automation.Post("/manager", gcpStandaloneAutomationBillingHandler.RunAutomation)
					automation.Post("/orchestrator", gcpStandaloneAutomationBillingHandler.CreateOrchestration)
					automation.Delete("/orchestrator", gcpStandaloneAutomationBillingHandler.StopOrchestration)
					automation.Post("/task", gcpStandaloneAutomationBillingHandler.RunTask)
				}

				onboarding := gcpGroup.NewSubgroup("/onboarding")
				{
					onboarding.Get("/new-service-accounts", gcpStandaloneHandler.RunCreateServiceAccountsTask)
					onboarding.Get("/init-env", gcpStandaloneHandler.InitEnvironmentTask)
				}
			}
		}

		saasConsoleGroup := tasksGroup.NewSubgroup("/billing-standalone") // TODO danielle: rename saas-console
		{
			saasConsoleGroup.Get("/notify-new-accounts", saasConsoleHandler.NotifyNewAccounts)

			saasConsoleGroup.Get("/validate-billing", saasConsoleHandler.ValidateBillingData)
			saasConsoleGroup.Get("/validate-permissions", saasConsoleHandler.ValidatePermissions)
			saasConsoleGroup.Post("/validate-aws-account", masterPayerAccount.ValidateSaaS)

			saasConsoleGroup.Post("/validate-active-tier", saasConsoleHandler.DeactivateNoActiveTierBilling)

			gcpGroup := saasConsoleGroup.NewSubgroup("/google-cloud")
			{
				onboarding := gcpGroup.NewSubgroup("/onboarding")
				{
					onboarding.Get("/new-service-accounts", gcpSaaSConsoleHandler.RunCreateServiceAccountsTask)
					onboarding.Get("/init-env", gcpSaaSConsoleHandler.InitEnvironmentTask)
				}
			}
		}

		partnerSalesGroup := tasksGroup.NewSubgroup("/partner-sales")
		{
			partnerSalesGroup.Get("/customers", partnerSales.SyncCustomersHandler)
		}

		analyticsGroup := tasksGroup.NewSubgroup("/analytics")
		{
			analyticsGroup.Get("/currencies", cloudAnalytics.UpdateCurrenciesTable)
			analyticsGroup.Get("/update-credits-table", credit.UpdateCustomerCreditsTable)
			analyticsGroup.Post("/update-looker-table", looker.UpdateCustomersLookerTable)
			analyticsGroup.Post("/widgets/all-customers", cloudAnalytics.UpdateAllCustomerDashboardReportWidgetsHandler)
			analyticsGroup.Post("/widgets/customers/:customerID", cloudAnalytics.UpdateCustomerDashboardReportWidgetsHandler)
			analyticsGroup.Post("/widgets/customers/:customerID/singleWidget", cloudAnalytics.RefreshReportWidgetHandler)
			analyticsGroup.Post("/widgets/dashboards", cloudAnalytics.UpdateDashboardsReportWidgetsHandler)
			analyticsGroup.Post("/widgets/dashboards/subscription", dashboardSubscriptionHandler.SendSubscription)
			analyticsGroup.Get("/csp-metadata", cloudAnalytics.UpdateCustomersInfoTableHandler)

			azureGroup := analyticsGroup.NewSubgroup("/microsoft-azure")
			{
				azureGroup.Get("/metadata/customers", analyticsMetadataHandler.UpdateAzureAllCustomersMetadata)
				azureGroup.Post("/metadata/customers/:customerID", analyticsMetadataHandler.UpdateAzureCustomerMetadata)
				azureGroup.Get("/customers-aggregate", analyticsMicrosoftAzure.UpdateAggregateAllAzureCustomers)
				azureGroup.Get("/customers/:customerID/aggregate", analyticsMicrosoftAzure.UpdateAggregatedTableAzure)
				azureGroup.Get("/customers/:customerID/aggregate-all", analyticsMicrosoftAzure.UpdateAllAzureAggregatedTables)

				azureGroup.Get("/csp-accounts", analyticsMicrosoftAzure.UpdateCSPTable)
				azureGroup.Get("/csp-accounts-aggregate", analyticsMicrosoftAzure.UpdateCSPAggregatedTable)
			}

			bqlensGroup := analyticsGroup.NewSubgroup("/bqlens")
			{
				bqlensGroup.Get("/metadata/customers", analyticsMetadataHandler.UpdateBQLensAllCustomersMetadata)
				bqlensGroup.Post("/metadata/customers/:customerID", analyticsMetadataHandler.UpdateBQLensCustomerMetadata)
			}

			datahubGroup := analyticsGroup.NewSubgroup("/datahub")
			{
				datahubGroup.Post("/metadata/customers", analyticsMetadataHandler.UpdateDataHubMetadata)
			}

			awsGroup := analyticsGroup.NewSubgroup("/amazon-web-services")
			{
				awsGroup.Get("/discounts", cloudAnalytics.UpdateDiscountsAWS)
				awsGroup.Get("/shared-payers-credits", awsSharedPayersCreditsHandler.UpdateSharedPayersCredits)

				awsGroup.Post("/metadata/customers", analyticsMetadataHandler.UpdateAWSAllCustomersMetadata)
				awsGroup.Post("/metadata/customers/:customerID", analyticsMetadataHandler.UpdateAWSCustomerMetadata)
				awsGroup.Post("/accounts-aggregate", cloudAnalytics.UpdateAggregateAllAWSAccounts)
				awsGroup.Post("/accounts/:accountID/aggregate", cloudAnalytics.UpdateAggregatedTableAWS)
				awsGroup.Post("/accounts/:accountID/aggregate-all", cloudAnalytics.UpdateAllAWSAggregatedTables)

				awsGroup.Post("/csp-accounts", cloudAnalytics.UpdateCSPAccounts)
				awsGroup.Post("/csp-accounts-aggregate", cloudAnalytics.UpdateCSPAggregatedTableAWS)
			}

			gcpGroup := analyticsGroup.NewSubgroup("/google-cloud")
			{
				gcpGroup.Get("/init-raw-table", cloudAnalytics.InitRawTable)
				gcpGroup.Get("/init-raw-table-resource", cloudAnalytics.InitRawResourceTable)
				gcpGroup.Get("/update-raw-table", cloudAnalytics.UpdateRawTableLastPartition)
				gcpGroup.Get("/update-raw-table-resource", cloudAnalytics.UpdateRawResourceTableLastPartition)
				gcpGroup.Get("/iam-resources", handlers.UpdateOrganizationIAMResources)
				gcpGroup.Get("/iam-resources-table", cloudAnalytics.UpdateIAMResources)
				gcpGroup.Get("/discounts", cloudAnalytics.UpdateDiscountsGCP)
				gcpGroup.Get("/scheduled", cloudAnalytics.ScheduledBillingAccountsTableUpdate)

				gcpGroup.Post("/accounts", cloudAnalytics.UpdateAllBillingAccountsTableHandler)
				gcpGroup.Post("/accounts/:billingAccountID", cloudAnalytics.UpdateCustomerBillingAccountTable)
				gcpGroup.Post("/accounts/:billingAccountID/metadata", analyticsMetadataHandler.UpdateGCPBillingAccountMetadata)
				gcpGroup.Post("/accounts/:billingAccountID/aggregate", cloudAnalytics.UpdateAggregatedTableGCP)
				gcpGroup.Post("/accounts/:billingAccountID/aggregate-all", cloudAnalytics.UpdateAllGCPAggregatedTables)

				gcpGroup.Post("/csp-accounts", cloudAnalytics.UpdateCSPBillingAccounts)
				gcpGroup.Post("/csp-accounts/:billingAccountID", cloudAnalytics.AppendToTempCSPBillingAccountTable)
				gcpGroup.Post("/csp-accounts-finalize", cloudAnalytics.UpdateCSPTableAndDeleteTemp)
				gcpGroup.Post("/csp-accounts-join", cloudAnalytics.JoinCSPTempTable)
				gcpGroup.Post("/csp-accounts-aggregate", cloudAnalytics.UpdateCSPAggregatedTableGCP)

				costAllocationGroup := gcpGroup.NewSubgroup("/costAllocation")
				{
					costAllocationGroup.Post("/update-customers", costAllocation.UpdateActiveCustomersHandler)
					costAllocationGroup.Put("/update-missing-clusters", costAllocation.UpdateMissingClustersHandler)
					costAllocationGroup.Post("/schedule-init-standalone-accounts", costAllocation.ScheduleInitStandaloneAccountsHandler)
					costAllocationGroup.Post("/init-standalone-account/:billingAccountID", costAllocation.InitStandaloneAccountHandler)
				}
			}

			gcpStandaloneGroup := analyticsGroup.NewSubgroup("/google-cloud-standalone")
			{
				gcpStandaloneGroup.Post("/accounts/:billingAccountID", cloudAnalytics.UpdateCustomerBillingAccountTable)
				gcpStandaloneGroup.Post("/accounts/:billingAccountID/metadata", analyticsMetadataHandler.UpdateGCPBillingAccountMetadata)
				gcpStandaloneGroup.Post("/accounts/:billingAccountID/aggregate", cloudAnalytics.UpdateAggregatedTableGCP)
				gcpStandaloneGroup.Post("/accounts/:billingAccountID/aggregate-all", cloudAnalytics.UpdateAllGCPAggregatedTables)
				gcpStandaloneGroup.Get("/billing-updates", cloudAnalytics.StandaloneBillingUpdateEvents)
			}

			reportsGroup := analyticsGroup.NewSubgroup("/reports")
			{
				reportsGroup.Post("/send", cloudAnalytics.SendReportHandler)
				reportsGroup.Delete("/draft", cloudAnalytics.DeleteStaleDraftReportsHandler)
			}

			budgetsGroup := analyticsGroup.NewSubgroup("/budgets")
			{
				budgetsGroup.Get("/refreshBudgets", cloudAnalytics.RefreshAllBudgetsHandler)
				budgetsGroup.Get("/triggerAlerts", cloudAnalytics.TriggerBudgetsAlertsHandler)
				budgetsGroup.Get("/triggerForecastedDateAlerts", cloudAnalytics.TriggerForecastedDateAlertsHandler)
			}

			digestGroup := analyticsGroup.NewSubgroup("/digest")
			{
				digestGroup.Post("/daily", digest.ScheduleDaily)
				digestGroup.Post("/weekly", digest.ScheduleWeekly)
				digestGroup.Post("/generate-worker", digest.GenerateDigest)
				digestGroup.Post("/send-worker", digest.SendDigest)

				digestGroup.Get("/monthly", digest.HandleMonthlyDigest)
			}

			customersGroup := analyticsGroup.NewSubgroup("/customers")
			{
				customersGroup.Get("/:customerID/budgets/:budgetID", cloudAnalytics.RefreshBudgetUsageDataHandler)
				customersGroup.Post("/:customerID/entities/:entityID/sync-invoice-attributions", analyticsAttributionGroups.SyncEntityInvoiceAttributions)
			}

			alertsGroup := analyticsGroup.NewSubgroup("/alerts")
			{
				alertsGroup.Get("/refresh", analyticsAlerts.RefreshAlerts)
				alertsGroup.Get("/:alertID/refresh", analyticsAlerts.RefreshAlert)
				alertsGroup.Get("/send-emails", analyticsAlerts.SendEmails)
				alertsGroup.Get("/send/:customerID", analyticsAlerts.SendEmailsToCustomer)
			}
		}

		invoicingGroup := tasksGroup.NewSubgroup("/invoicing")
		{
			//invoicing draft invoicing step (step2)
			invoicingGroup.Get("", invoicing.ProcessCustomersInvoicesHandler)                       // draft invoice generation - for all customers
			invoicingGroup.Get("/singleCustomer", invoicing.ProcessSingleCustomerInvoicesHandler)   // draft invoice generation - http for single customer (don't need to provide rates)
			invoicingGroup.Post("/customers/:customerID", invoicing.ProcessCustomerInvoicesHandler) // draft invoice generation - cloudtask

			// invoicing step1
			invoicingGroup.Get("/google-cloud/adhoc-invoice", invoicing.GoogleCloudInvoicingForSingleAccountHandler)
			invoicingGroup.Get("/google-cloud", invoicing.GoogleCloudInvoicingHandler)
			invoicingGroup.Post("/google-cloud", invoicing.GoogleCloudInvoicingWorker)

			invoicingGroup.Get("/google-cloud/skus", invoicing.UpdateCloudBillingSkus)
			invoicingGroup.Get("/microsoft-azure", invoicing.MicrosoftAzureInvoicingHandler)

			invoicingGroup.Get("/amazon-web-services", invoicing.AmazonWebServicesInvoicingHandler) // cht all customers mbd step
			invoicingGroup.Post("/amazon-web-services", invoicing.AmazonWebServicesInvoicingWorker) // cht single customer mbd step

			invoicingGroup.Post("/amazon-web-services/all-customers", invoicingAnalyticsData.UpdateAmazonWebServicesInvoicingData)             // aws analytics all customers mbda step
			invoicingGroup.Post("/amazon-web-services/customer/:customerID", invoicingAnalyticsData.AmazonWebServicesAnalyticsInvoicingWorker) // aws analytics single customers mbda step

			// testing export customers
			invoicingGroup.Post("/dev-export-invoices", invoicing.DevExportHandler)

			// testing issue single customer invoice
			invoicingGroup.Post("/dev-issue-single-invoice", invoicing.DevIssueSingleCustomerInvoiceHandler)

			fssa := invoicingFSSA.NewFSSAInvoiceData(loggerProvider, a.conn)
			// aws
			invoicingGroup.Post("/amazon-web-services-standalone/all-customers", fssa.UpdateFlexsaveInvoicingData)
			invoicingGroup.Post("/amazon-web-services-standalone/customer/:customerID", fssa.UpdateFlexsaveInvoicingDataWorker)
			// gcp
			invoicingGroup.Post("/google-cloud-standalone/all-customers", fssa.UpdateFlexsaveInvoicingData)
			invoicingGroup.Post("/google-cloud-standalone/customer/:customerID", fssa.UpdateFlexsaveInvoicingDataWorker)

			invoicingGroup.Post("/draft-invoices-notification", handlers.NewDraftInvoicesNotification(loggerProvider, a.conn).DraftInvoicesCreatedNotification)
		}

		billingExplainerGroup := tasksGroup.NewSubgroup("/billing-explainer")
		{
			billingExplainerGroup.Post("/data", billingExplainerHandler.GetDataFromBigQueryAndStoreInFirestore)
		}

		invoicesGroup := tasksGroup.NewSubgroup("/invoices")
		{
			invoicesGroup.Get("", handlers.InvoicesMainHandler)
			invoicesGroup.Post("", handlers.InvoicesCustomerWorker)
			invoicesGroup.Get("/notifications", handlers.NotificationsHandler)
			invoicesGroup.Post("/notifications", handlers.NotificationsWorker)
			invoicesGroup.Get("/notice-to-remedy", handlers.NoticeToRemedy)
		}

		receiptsGroup := tasksGroup.NewSubgroup("/receipts")
		{
			receiptsGroup.Get("", priorityHandler.SyncReceipts)
			receiptsGroup.Post("", priorityHandler.SyncCustomerReceipts)
			receiptsGroup.Get("/aggregate", handlers.Aggregate)
			receiptsGroup.Get("/account-receivables", handlers.AccountReceiveables)
		}

		firestoreGroup := tasksGroup.NewSubgroup("/firestore")
		{
			firestoreGroup.Get("/export", handlers.FirestoreExportHandler)
			firestoreGroup.Get("/import-bigquery", handlers.FirestoreImportBigQueryHandler)
		}

		hubspotGroup := tasksGroup.NewSubgroup("/hubspot")
		{
			hubspot := handlers.NewHubspot(a.log, a.conn)
			hubspotGroup.Get("", hubspot.SyncHandler)
			hubspotGroup.Get("/companies/:customerID", hubspot.SyncCompanyHandler)
			hubspotGroup.Get("/contacts/:customerID", hubspot.SyncContactsHandler)
		}

		assetsGroup := tasksGroup.NewSubgroup("/assets")
		{
			// Cleans up stale assets
			assetsGroup.Get("/collect-stale", tasks.AssetCleanupHandler)

			// Google Cloud assets discovery
			assetsGroup.Get("/google-cloud", partnerSales.BillingAccountsListHandler)
			assetsGroup.Get("/google-cloud-standalone", handlers.StandaloneBillingAccountsListHandler)

			// Google Cloud assets task queue handler
			assetsGroup.Post("/google-cloud", handlers.BillingAccountsPageHandler)

			// Google Workspace assets discovery
			assetsGroup.Get("/g-suite", handlers.SubscriptionsListHandlerGSuite)

			// Microsoft 365 and resold Microsoft Azure assets discovery
			assetsGroup.Get("/microsoft", handlers.SubscriptionsListHandlerMicrosoft)
			assetsGroup.Post("/microsoft/customers/:customerID/subscriptions/:subscriptionID/syncQuantity", microsoftLicensesHandler.LicenseSyncHandler)

			// AWS assets discovery
			assetsGroup.Get("/amazon-web-services/shared", assets.UpdateAWSAssetsShared)
			assetsGroup.Get("/amazon-web-services/dedicated", aws.UpdateAWSAssetsDedicated)
			assetsGroup.Post("/amazon-web-services/dedicated/:accountID", aws.UpdateAWSAssetDedicated)
			assetsGroup.Post("/amazon-web-services/manual/:accountID", aws.UpdateManualAsset)

			// AWS Flexsave SaaS assets discovery
			assetsGroup.Get("/amazon-web-services-standalone", awsStandaloneHandler.UpdateAllStandAloneAssets)
			assetsGroup.Post("/amazon-web-services-standalone/:customerID", awsStandaloneHandler.UpdateStandAloneAssets)

			// AWS SaaS (billing) assets discovery
			assetsGroup.Get("/amazon-web-services-saas", awsSaaSConsoleHandler.UpdateAllSaaSAssets)
			assetsGroup.Post("/amazon-web-services-saas/:customerID", awsSaaSConsoleHandler.UpdateSaaSAssets)

			assetsGroup.Get("/amazon-web-services/handshakes", aws.UpdateHandshakes)
			assetsGroup.Get("/amazon-web-services/accounts", aws.UpdateAccounts)
			assetsGroup.Get("/amazon-web-services/tags", handlers.TagCustomers)

			assetsGroup.Post("/delete-customer-sa", handlers.DeleteServiceAccount)

			assetsGroup.Get("/google-cloud/services-analytics", handlers.ImportCloudServicesToFS)
			assetsGroup.Get("/amazon-web-services/services-analytics", handlers.ImportAWSCloudServicesToFS)

			assetsGroup.Post("/azure-saas", azureSaaSConsoleHandler.CreateAssetDiscoveryTasks)
			assetsGroup.Post("/azure-saas/:customerID", azureSaaSConsoleHandler.RunAssetDiscoveryTaskHandler)

			// Adds & Updates AWS assets support type in firestore
			assetsGroup.Post("/amazon-web-services/update-assets-support-type", awsAssetsSupportHandler.UpdateAWSSupportAssetsTypeInFS)
		}

		dashboardGroup := tasksGroup.NewSubgroup("/dashboard")
		{
			dashboardGroup.Get("/debt-analytics", handlers.DebtAnalyticsHandler)
			dashboardGroup.Get("/attach-dashboards", publicdashboardsHandler.AttachAllDashboardsHandler)
			dashboardGroup.Get("/commitment-contracts", handlers.UpdateCommitmentContracts)

			// Ramp Plan
			dashboardGroup.Get("/ramp-plan", rampPlan.UpdateAllRampPlans)
			dashboardGroup.Post("/update-ramp-plan", rampPlan.UpdateRampPlanByID)
			dashboardGroup.Post("/create-ramp-plans", rampPlan.CreateRampPlans)

			renewalsGroup := dashboardGroup.NewSubgroup("/renewals")
			{
				renewalsGroup.Get("/g-suite", handlers.GSuiteRenewalsHandler)
				renewalsGroup.Get("/office-365", handlers.Office365RenewalsHandler)
				renewalsGroup.Get("/zendesk", handlers.ZendeskRenewalsHandler)
				renewalsGroup.Get("/bettercloud", handlers.BetterCloudRenewalsHandler)
			}

			licensesGroup := dashboardGroup.NewSubgroup("/licenses")
			{
				licensesGroup.Get("/g-suite", handlers.CopyLicenseToDashboardGSuite)
				licensesGroup.Get("/office-365", handlers.CopyLicenseToDashboardMicrosoft)
			}
		}

		cloudConnectGroup := tasksGroup.NewSubgroup("/cloudconnect")
		{
			cloudConnectGroup.Get("/health", cloudConnect.Health)
			cloudConnectGroup.Get("/aws/health", handlers.AWSPermissionsHandler)
		}

		slackGroup := tasksGroup.NewSubgroup("/slack")
		{
			slackGroup.Get("/channels-info", handlers.GetSlackSharedChannelsInfo)
		}

		knownIssues := tasksGroup.NewSubgroup("/knownissues")
		{
			kw := knownissues.NewKnownIssues(loggerProvider, a.conn)
			knownIssues.Get("", kw.UpdateKnownIssues)
		}

		serviceLimits := tasksGroup.NewSubgroup("/servicelimits")
		{
			serviceLimits.Get("/aws-refresh", aws.GetCustomerServicesLimitsAWS)
			serviceLimits.Get("/gcp-refresh", googleCloud.GetCustomerServicesLimitsGCP)
		}

		recommenderGroup := tasksGroup.NewSubgroup("/recommender")
		{
			recommenderGroup.Get("/update", googleCloud.GetCustomersRecommendations)
			recommenderGroup.Put("/updateCustomer", googleCloud.UpdateCustomerRecommendations)
		}

		stripeGroup := tasksGroup.NewSubgroup("/stripe")
		{
			stripeGroup.Post("/payments", stripe.AutomaticPaymentsHandler)
			stripeGroup.Post("/payments/entities/:entityID", stripe.AutomaticPaymentsEntityWorker)
			stripeGroup.Get("/digest", stripe.PaymentsDigestHandler)
			stripeGroup.Post("/sync-stripe-customer/:entityID", stripe.SyncCustomerData)
		}

		spotScalingGroup := tasksGroup.NewSubgroup("/spot-scaling")
		{
			spotScalingGroup.Post("/daily-costs", spot0Costs.SpotScalingDailyCosts)
			spotScalingGroup.Post("/monthly-costs", spot0Costs.SpotScalingMonthlyCosts)
			spotScalingGroup.Post("/spot-scaling-email", spotScalingEmail.SendMarketingEmail)
			spotScalingGroup.Get("/non-billing-tags-asg", spot0Costs.UpdateCostAllocationTags)
			spotScalingGroup.Get("/fbod-healthcheck", spot0fbod.FbodHealthCheck)
		}

		marketplaceGroup := tasksGroup.NewSubgroup("/marketplace")
		{
			gcpMarketplaceGroup := marketplaceGroup.NewSubgroup("/gcp")
			{
				gcpMarketplaceGroup.Post("/populate-billing-accounts", marketplace.PopulateBillingAccounts)
			}
		}

		priorityGroup := tasksGroup.NewSubgroup("/priority")
		{
			priorityGroup.Post("/sync-customers", priorityHandler.SyncCustomers)
			priorityGroup.Post("/create-invoice", priorityHandler.CreateInvoice)
			priorityGroup.Post("/approve-invoice", priorityHandler.ApproveInvoice)
			priorityGroup.Post("/close-invoice", priorityHandler.CloseInvoice)
			priorityGroup.Post("/print-invoice", priorityHandler.PrintInvoice)
			priorityGroup.Post("/delete-invoice", priorityHandler.DeleteInvoice)
		}

		entitiesGroup := tasksGroup.NewSubgroup("/entities")
		{
			entitiesGroup.Post("/sync-invoice-attributions", entitiesHandler.SyncEntitiesInvoiceAttributions)
		}

		mixpanelTasksGroup := tasksGroup.NewSubgroup("/mixpanel-tasks")
		{
			mixpanelTasksGroup.Post("/export-events-to-bq", mixpanelHandler.ExportEventsToBQ)
		}

		courierTasksGroup := tasksGroup.NewSubgroup("/courier")
		{
			courierTasksGroup.Post("/export-notification-to-bq", courierHandler.ExportNotificationToBQHandler)
			courierTasksGroup.Post("/export-notifications-to-bq", courierHandler.ExportNotificationsToBQHandler)
		}

		bqlensTasksGroup := tasksGroup.NewSubgroup("/bq-lens")
		{
			bqlensTasksGroup.Get("/discovery-scheduler", bqLensDiscoveryHandler.AllCustomersTablesDiscovery)
			bqlensTasksGroup.Post("/discovery/:customerID", bqLensDiscoveryHandler.SingleCustomerTablesDiscovery)

			bqlensTasksGroup.Get("/optimizer-scheduler", bqLensOptimizerHandler.AllCustomersOptimizer)
			bqlensTasksGroup.Post("/optimizer/:customerID", bqLensOptimizerHandler.SingleCustomerOptimizer)

			bqlensTasksGroup.Post("/onboarding", bqLensOnboardHandler.Onboard)

			bqlensTasksGroup.Post("/backfill-scheduler", bqLensBackfillHandler.ScheduleBackfill)
			bqlensTasksGroup.Post("/backfill", bqLensBackfillHandler.Backfill)

			bqlensTasksGroup.Post("/edition-pricebook", bqLensPricebookHandler.SetEditionPricebook)
		}

		tiersGroup := tasksGroup.NewSubgroup("/tiers")
		{
			tiersGroup.Get("/trial-notifications", tiers.SendTrialNotifications)
			tiersGroup.Post("/set-customers-tier", tiers.SetCustomersTier)

			tiersGroup.Get("/customer/:customerID/feature-access", tiers.CustomerCanAccessFeature)
		}

		datahubGroup := tasksGroup.NewSubgroup("/datahub")
		{
			datahubGroup.Delete("/events/customers/hard", datahubHandler.DeleteAllCustomersDataHard)
			datahubGroup.Delete("/events/customers/:customerID/hard", datahubHandler.DeleteCustomerDataHard)
		}
	}

	// INTERNAL ENDPOINTS - Requests coming from other approved GAE, Cloud Run or other GCP services
	internalGroup := web.NewGroup(app, "/internal", mid.AuthInternalServiceAccounts())
	{
		internalGroup.Get("/ping", handlers.Ping)

		customersGroup := internalGroup.NewSubgroup("/customers/:customerID")
		{
			datahubGroup := customersGroup.NewSubgroup("/datahub")
			{
				datahubGroup.Delete("/events", datahubHandler.DeleteCustomerData)
				datahubGroup.Post("/events/delete", datahubHandler.DeleteCustomerSpecificEvents)
			}
		}

		marketplaceGcpGroup := internalGroup.NewSubgroup("/marketplace/gcp")
		{
			marketplaceGcpGroup.Post("/subscribe", marketplace.Subscribe)
			marketplaceGcpGroup.Post("/standalone-approve", marketplace.StandaloneApprove)
		}

		marketplaceAwsGroup := internalGroup.NewSubgroup("/marketplace/aws")
		{
			marketplaceAwsGroup.Post("/contract/:id", contractHandler.UpdateContract)
			marketplaceAwsGroup.Delete("/contract/:id", contractHandler.InternalCancelContract)
		}

		costAnomalyGroup := internalGroup.NewSubgroup("/cost-anomaly")
		{
			costAnomalyGroup.Post("/:customerID/preview", costAnomaly.Query)
		}

		// Used by Concedefy
		concedefyGroup := internalGroup.NewSubgroup("/aws")
		{
			concedefyGroup.Get("/asset", assets.GetAWSAsset)
			concedefyGroup.Get("/mpa", masterPayerAccount.GetMasterPayerAccount)
		}

		tiersGroup := internalGroup.NewSubgroup("/tiers")
		{
			tiersGroup.Post("/customer/:customerID/feature-access", tiers.CustomerCanAccessFeature)
		}
	}

	// EVENTS ENDPOINTS - Pub/Sub push handlers
	eventsGroup := web.NewGroup(app, "/events/v1")
	{
		marketplaceGcpGroup := eventsGroup.NewSubgroup("/marketplace/gcp", mid.AuthServiceAccount(mid.GetAllowedMarketplaceGcpEventsEmails()))
		{
			marketplaceGcpGroup.Post("/cmp-event", marketplace.HandleCmpEvent)
		}
	}

	// WEBHOOKS
	webhooks := web.NewGroup(app, "/webhooks/v1")
	{
		webhooks.Post("/stripe", stripe.WebhookHandler)
		webhooks.Put("/aws/updateAccountRole", handlers.AWSUpdateRoleHandler)
		webhooks.Post("/aws/updateAccountRole", handlers.AWSDeleteRoleHandler)
		webhooks.Put("/aws/updateAwsFeature", handlers.AWSUpdateFeature)
		webhooks.Post("/aws/updateAwsFeature", handlers.AWSUpdateFeature)

		flexsaveStandaloneGroup := webhooks.NewSubgroup("/flexsave-standalone-aws")
		{
			flexsaveStandaloneGroup.Put("/update-recommendations", awsStandaloneHandler.UpdateRecommendations)
			flexsaveStandaloneGroup.Post("/update-recommendations", awsStandaloneHandler.StackDeletion)
			flexsaveStandaloneGroup.Put("/update-billing", awsStandaloneHandler.UpdateBilling)
			flexsaveStandaloneGroup.Post("/update-billing", awsStandaloneHandler.StackDeletion)
		}

		eksMetricsGroup := webhooks.NewSubgroup("/eks-metrics")
		{
			eksMetricsGroup.Put("/update-eks", eksMetricsHandler.StackCreation)
			eksMetricsGroup.Post("/update-eks", eksMetricsHandler.StackDeletion)
			eksMetricsGroup.Post("/terraform-validate", eksMetricsHandler.ValidateTerraformDeployment)
			eksMetricsGroup.Post("/terraform-destroy", eksMetricsHandler.DestroyTerraformDeployment)
		}

		billingStandaloneGroup := webhooks.NewSubgroup("/saas-console-aws")
		{
			billingStandaloneGroup.Put("/onboarding", awsSaaSConsoleHandler.CURDiscovery)
			billingStandaloneGroup.Post("/onboarding", awsSaaSConsoleHandler.StackDeletion)
		}
	}

	slackApp := web.NewGroup(app, "/slack")
	{
		slackApp.Post("/event_callback", slack.AcknowledgeAndHandleEvent)
		slackApp.Get("/oauth2callback", slack.OAuth2callback)
		slackApp.Get("/installApp", slack.InstallApp)
		slackApp.Post("/mixpanelHandler/sendEvent", slack.SendMixpanelEvent)
		slackApp.Post("/update-collaboration", slack.UpdateCollaboration)
	}

	presentation := web.NewGroup(app, "/presentation")
	{
		presentation.Post("/create-customer", presentationHandler.CreateCustomer)
		presentation.Post("/customer/:customerID/delete-assets", presentationHandler.DeletePresentationCustomerAssets, mid.ValidatePathParamNotEmpty("customerID"))

		presentation.Post("/customer/:customerID/update-aws-billing-data", presentationHandler.UpdateCustomerAWSBillingData, mid.ValidatePathParamNotEmpty("customerID"))
		presentation.Post("/customer/:customerID/update-aws-assets", presentationHandler.UpdateCustomerAWSAssets, mid.ValidatePathParamNotEmpty("customerID"))

		presentation.Post("/customer/:customerID/update-azure-billing-data", presentationHandler.UpdateCustomerAzureBillingData, mid.ValidatePathParamNotEmpty("customerID"))
		presentation.Post("/customer/:customerID/update-azure-assets", presentationHandler.UpdateCustomerAzureAssets, mid.ValidatePathParamNotEmpty("customerID"))

		presentation.Post("/customer/:customerID/update-gcp-billing-data", presentationHandler.UpdateCustomerGCPBillingData, mid.ValidatePathParamNotEmpty("customerID"))
		presentation.Post("/customer/:customerID/update-gcp-assets", presentationHandler.UpdateCustomGcpAssets, mid.ValidatePathParamNotEmpty("customerID"))
		presentation.Post("/customer/:customerID/update-eks-lens-billing-data", presentationHandler.UpdateCustomerEKSLensBillingData, mid.ValidatePathParamNotEmpty("customerID"))

		presentation.Post("/customer/:customerID/copy-bq-lens-data", presentationHandler.CopyBQLensDataToCustomer, mid.ValidatePathParamNotEmpty("customerID"))
		presentation.Post("/customer/:customerID/copy-spot-scaling-data", presentationHandler.CopySpotScalingDataToCustomer, mid.ValidatePathParamNotEmpty("customerID"))
		presentation.Post("/customer/:customerID/copy-cost-anomalies", presentationHandler.CopyCostAnomaliesToCustomer, mid.ValidatePathParamNotEmpty("customerID"))
	}

	// FRONTEND ENDPOINTS
	apiGroup := web.NewGroup(app, "/api/v1", mid.AuthRequired(a.conn))
	{
		apiGroup.Proxy(fmt.Sprintf("/flexapi/*%s", handlers.ProxyPath), flexapiProxyHandler, mid.AuthDoitEmployee())

		awsGroup := apiGroup.NewSubgroup("/amazon-web-services", mid.AuthDoitEmployee())
		{
			mpaGroup := awsGroup.NewSubgroup("/master-payer-accounts")
			{
				mpaGoogleGroupSubGroup := mpaGroup.NewSubgroup("/google-group", mid.AuthDoitEmployee())
				{
					mpaGoogleGroupSubGroup.Post("/update", masterPayerAccount.UpdateGoogleGroup)
					mpaGoogleGroupSubGroup.Post("/delete", masterPayerAccount.DeleteGoogleGroup)
					mpaGoogleGroupSubGroup.Post("/init-creation-cloud-task", masterPayerAccount.CreateGoogleGroupCloudTask)
				}
				mpaGroup.Post("/validate", masterPayerAccount.ValidateMPA)
				mpaGroup.Post("/link-mpa", masterPayerAccount.LinkMpaToSauron)
				mpaGroup.Post("/update-assets/:accountID", aws.UpdateAWSAssetDedicated)
				mpaGroup.Post("/create-portfolio-share/:accountID", awsServiceCatalog.CreatePortfolioShare)
				mpaGroup.Post("/retire/:payerID", masterPayerAccount.RetireMPAHandler)
			}

			accountGeneratorGroup := awsGroup.NewSubgroup(
				"/generated-accounts",
				doitRoleAwsAccountGenerator,
			)
			{
				accountGeneratorGroup.Post("/batch-create", awsGeneratedAccounts.CreateAccountsBatch)
			}

			awsGroup.Post("/ples", ples.UpdatePLES)
		}

		eksMetricsGroup := apiGroup.NewSubgroup("/eks-metrics")
		{
			eksMetricsGroup.Post("/eks-deployment-file", eksMetricsHandler.GetEksDeploymentFiles)
			eksMetricsGroup.Post("/validate-deployment", eksMetricsHandler.ValidateDeployment)
			eksMetricsGroup.Post("/sync-cluster", eksMetricsHandler.SyncManualCluster)
			eksMetricsGroup.Post("/terraform-cluster-file", eksMetricsHandler.GetClusterTerraformFile)
			eksMetricsGroup.Post("/terraform-region-file", eksMetricsHandler.GetRegionTerraformFile)
			eksMetricsGroup.Post("/terraform-validate", eksMetricsHandler.ValidateTerraformDeployment)
			eksMetricsGroup.Post("/terraform-destroy", eksMetricsHandler.DestroyTerraformDeployment)
		}

		fullstory := handlers.NewFullstory(a.log, a.conn)
		apiGroup.Get("/fullstory", fullstory.GetUserHMAC)

		apiGroup.Post("/regexp", handlers.ValidateRegexp)

		announcekitHandler := handlers.NewAnnouncekitHandler(loggerProvider)
		apiGroup.Post("/announcekit-token", announcekitHandler.CreateJwtToken)

		superQueryGroup := apiGroup.NewSubgroup("/superquery")
		{
			superQueryGroup.Get("/finops-process", handlers.InvokeBigQueryProcess)
		}

		spotScalingGroup := apiGroup.NewSubgroup("/spot-scaling/:customerID")
		{
			spotscaling := handlers.NewSpotZero(a.log, a.conn)
			spotScalingGroup.Post("/refresh-asgs", spotscaling.RefreshASGs)
			spotScalingGroup.Post("/apply-recommendation", spotscaling.ApplyConfiguration)
			spotScalingGroup.Post("/update-configuration", spotscaling.UpdateAsgConfig)
			spotScalingGroup.Get("/fallback-on-demand/check-config/:accountID/:region", spotscaling.CheckFallbackOnDemandConfig)
			spotScalingGroup.Post("/fallback-on-demand/add-config", spotscaling.AddFallbackOnDemandConfig)
			spotScalingGroup.Post("/average-prices", spotscaling.AveragePrices)
		}

		usersGroup := apiGroup.NewSubgroup("/users")
		{
			usersHandler := handlers.NewUsers(loggerProvider, a.conn)
			usersGroup.Post("/doit-migration", usersHandler.DoitMigration, mid.AuthDoitEmployee())
			usersGroup.Post("/impersonate/start", handlers.StartImpersonate, mid.AuthDoitEmployee())
			usersGroup.Get("/impersonate/stop", handlers.StopImpersonate)
			usersGroup.Post("/generateApiToken", handlers.GenerateAPIToken)
			usersGroup.Post("/deleteApiKey", handlers.DeleteAPIKey)
			usersGroup.Post("/updateDisplayName", handlers.UpdateFSUserDisplayName)
		}

		invoicingGroup := apiGroup.NewSubgroup("/invoicing", mid.AuthDoitOwner())
		{
			invoicingGroup.Post("/export", invoicing.ExportHandler)
			invoicingGroup.Post("/issue-customer-invoices", invoicing.IssueSingleCustomerInvoiceHandler)
			invoicingGroup.Post("/recalculate-customer", invoicing.RecalculateSingleCustomerHandler)
			invoicingGroup.Post("/cancel-issued-invoices", invoicing.CancelIssuedInvoices)
		}

		mixpanelGroup := apiGroup.NewSubgroup("/mixpanel", mid.AuthDoitEmployee())
		{
			mixpanelGroup.Get("/active-users-report", mixpanelHandler.QuerySegmentationReportHandler)
		}

		pricebooksGroup := apiGroup.NewSubgroup("/pricebooks", mid.AuthDoitEmployee())
		{
			pricebooksGroup.Post("/amazon-web-services", aws.CreatePricebook)
			pricebooksGroup.Put("/amazon-web-services", aws.UpdatePricebook)
			pricebooksGroup.Post("/amazon-web-services/assignments", aws.AssignPricebook)
		}

		flexsaveAdminGroup := apiGroup.NewSubgroup("/flexsave", doitRoleFlexsaveAdmin)
		{
			flexsaveAdminGroup.Post("/orders/autopilot/regenerate-all", flexsaveAWS.RegenerateAutopilotOrdersForAllCustomersHandler)
			flexsaveAdminGroup.Post("/orders/autopilot/accept-all", flexsaveAWS.AcceptAutopilotOrdersHandler)
			flexsaveAdminGroup.Post("/orders/autopilot/regenerate/:customerID", flexsaveAWS.RegenerateAutopilotOrdersForCustomerHandler)
			flexsaveAdminGroup.Post("/orders/reactivate", flexsaveAWS.ReactivateOrdersHandler)
			flexsaveAdminGroup.Post("/orders/reactivateAllForCustomer/:customerID", flexsaveAWS.ReactivateAllOrdersForCustomerHandler)
			flexsaveAdminGroup.Post("/orders/autopilotOrderEndDateUpdater/:customerID", flexsaveAWS.AutopilotOrdersEndDateUpdateHandler)
		}

		flexsaveAdminRoleGroup := apiGroup.NewSubgroup("/flexsave", mid.AuthMPAOrFlexsaveAdmin(a.conn))
		{
			flexsaveAdminRoleGroup.Put("/payers/:payerId/ops-update", awsOpsPage.ProcessOpsUpdates)
		}

		flexsaveDoitEmployeeGroup := apiGroup.NewSubgroup("/flexsave", mid.AuthDoitEmployee())
		{
			flexsaveDoitEmployeeGroup.Post("/payers/mpaActivated/:accountNumber", AWSResoldCache.MPAActivatedHandler)
		}

		flexsaveGcpGroup := apiGroup.NewSubgroup("/flexsave-gcp", mid.AuthDoitEmployee())
		{
			flexsaveGcpGroup.Post("/purchase-plan/execute", flexsaveGCP.Execute)
			flexsaveGcpGroup.Get("/purchase-plan/refresh", flexsaveGCP.Refresh)
			flexsaveGcpGroup.Get("/purchase-plan/refresh/:customerID", flexsaveGCP.RefreshCustomer)

			// New Ops page endpoints
			// trigger update workloads aggregation for bulk purchase
			flexsaveGcpGroup.Get("/ops/agg-workloads", flexsaveGCP.Ops2UpdateBulk)
			// Get purchase plan prices for selected workload
			flexsaveGcpGroup.Post("/purchase-plan/v2/prices", flexsaveGCP.GetPurchaseplanPrices)
			// Do manual purchase for selected workloads
			flexsaveGcpGroup.Post("/purchase-plan/v2/manual-purchase", flexsaveGCP.ManualPurchase)
			// trigger purchase for selected workloads or customers plans
			flexsaveGcpGroup.Post("/purchase-plan/v2/execute", flexsaveGCP.Ops2Execute)
			// trigger purchase for selected customers selected workloads plans
			flexsaveGcpGroup.Post("/purchase-plan/v2/approve", flexsaveGCP.Ops2ApproveWorkloads)
			// trigger purchase for selected workloads or customers plans
			flexsaveGcpGroup.Post("/purchase-plan/v2/execute-bulk", flexsaveGCP.Ops2ExecuteBulk)
			// updates all customers stats recommendation and purchase plans
			flexsaveGcpGroup.Get("/purchase-plan/v2/refresh", flexsaveGCP.Ops2Refresh)
			// updates specific customer's stats recommendation and purchase plans
			flexsaveGcpGroup.Get("/purchase-plan/v2/refresh/:customerID", flexsaveGCP.Ops2RefreshCustomer)
		}

		flexsaveStandaloneGCPGroup := apiGroup.NewSubgroup("/flexsave-standalone-gcp")
		{
			flexsaveStandaloneGCPGroup.Get("/init-onboarding/:customerID", gcpStandaloneHandler.InitOnboarding)
			flexsaveStandaloneGCPGroup.Post("/test-estimations-connection", gcpStandaloneHandler.TestEstimationsConnection)
			flexsaveStandaloneGCPGroup.Get("/refresh-estimations/:customerID", gcpStandaloneHandler.RefreshEstimations)
			flexsaveStandaloneGCPGroup.Post("/contract-agreed", gcpStandaloneHandler.AddContract)
			flexsaveStandaloneGCPGroup.Post("/activate", gcpStandaloneHandler.Activate)
			flexsaveStandaloneGCPGroup.Get("/savings-report/:customerID", gcpStandaloneHandler.SavingsReport)
		}

		flexsaveStandaloneAWSGroup := apiGroup.NewSubgroup("/flexsave-standalone-aws")
		{
			flexsaveStandaloneAWSGroup.Get("/init-onboarding/:customerID", awsStandaloneHandler.InitOnboarding)
			flexsaveStandaloneAWSGroup.Get("/refresh-estimations/:customerID", awsStandaloneHandler.RefreshEstimations)
			flexsaveStandaloneAWSGroup.Delete("/estimation/:customerID/:accountID", awsStandaloneHandler.DeleteEstimation, mid.AuthDoitEmployee())
			flexsaveStandaloneAWSGroup.Post("/savings-recommendations/:customerID/:accountID", awsStandaloneHandler.UpdateSavingsAndRecommendationCSV)
			flexsaveStandaloneAWSGroup.Post("/contract-agreed", awsStandaloneHandler.AddContract)
			flexsaveStandaloneAWSGroup.Get("/savings-report/:customerID", awsStandaloneHandler.SavingsReport)
		}

		saasConsoleGCPGroup := apiGroup.NewSubgroup("/saas-console-gcp")
		{
			saasConsoleGCPGroup.Get("/init-onboarding/:customerID", gcpSaaSConsoleHandler.InitOnboarding)
			saasConsoleGCPGroup.Post("/contract-agreed", gcpSaaSConsoleHandler.AddContract)
			saasConsoleGCPGroup.Post("/activate", gcpSaaSConsoleHandler.Activate)
		}

		saasConsoleAWSGroup := apiGroup.NewSubgroup("/saas-console-aws")
		{
			saasConsoleAWSGroup.Post("/init-onboarding/:customerID", awsSaaSConsoleHandler.InitOnboarding)
			saasConsoleAWSGroup.Post("/contract-agreed", awsSaaSConsoleHandler.AddContract)
			saasConsoleAWSGroup.Post("/refresh", awsSaaSConsoleHandler.CURRefresh)
			saasConsoleAWSGroup.Post("/activate", awsSaaSConsoleHandler.Activate)
		}

		salesforceGroup := apiGroup.NewSubgroup("/salesforce")
		{
			salesforceGroup.Post("/composite", salesforce.CompositeRequestHandler)
		}

		analyticsTemplateLibraryGroup := apiGroup.NewSubgroup("/analytics/template-library")
		{
			reportTemplatesGroup := analyticsTemplateLibraryGroup.NewSubgroup("/report-templates")
			{
				reportTemplatesGroup.Post("", reportTemplateHandler.CreateReportTemplateHandler)
				reportTemplatesGroup.Put(
					"/:id/approve",
					reportTemplateHandler.ApproveReportTemplateHandler,
				)
				reportTemplatesGroup.Put(
					"/:id/reject",
					reportTemplateHandler.RejectReportTemplateHandler,
				)
				reportTemplatesGroup.Put(
					"/:id",
					reportTemplateHandler.UpdateReportTemplateHandler,
				)
				reportTemplatesGroup.Delete(
					"/:id",
					reportTemplateHandler.DeleteReportTemplateHandler,
				)
				reportTemplatesGroup.Post("/template-data", reportTemplateHandler.GetTemplateData)
			}
		}

		customerGroup := apiGroup.NewSubgroup("/customers/:customerID", mid.AuthCustomerRequired())
		{
			azureGroup := customerGroup.NewSubgroup("/azure")
			{
				azureGroup.Post("/store-config", azure.StoreBillingConnection)
				azureGroup.Get("/get-storage-account-name", azure.GetStorageAccountNameForOnboarding)
			}

			customerGroup.Delete(
				"",
				customerHandler.DeleteCustomer,
				doitRoleCustomerSettingsAdmin,
			)

			customerGroup.Get("/dashboards", handlers.GetCustomerDashboards)
			customerGroup.Get("/invoices", handlers.CustomerHandler, mid.AuthDoitEmployee())
			customerGroup.Get("/refresh/g-suite", handlers.SubscriptionsListHandler, mid.AuthDoitEmployee())

			usersGroup := customerGroup.NewSubgroup("/users")
			{
				usersGroup.Post("/exists", handlers.Exists)
				usersGroup.Get("/getUserId", handlers.GetUIDByEmail)
				usersGroup.Delete("/delete", handlers.Delete)
			}

			// Slack
			customerGroup.Get("/slack/create-shared-channel", slack.CreateSlackSharedChannel)
			customerGroup.Get("/slack/get-channel-invitation", slack.GetChannelInvitation)
			customerGroup.Get("/slack/channels", slack.GetCustomerChannels)
			customerGroup.Get("/slack/auth-test", slack.AuthTest)

			// Jira
			customerGroup.Post("/jira/instances", jira.CreateInstance)

			// Cloud Connect
			customerGroup.Post("/cloudconnect/google-cloud", cloudConnect.AddGcpServiceAccount)
			customerGroup.Post("/cloudconnect/google-cloud/remove", handlers.RemoveGcpServiceAccount)
			customerGroup.Post("/cloudconnect/amazon-web-services", handlers.AWSAddRoleHandler)
			customerGroup.Get("/cloudconnect/get-missing-permissions", handlers.GetMissingPermissions)
			customerGroup.Post("/cloudconnect/google-cloud/partial", cloudConnect.AddPartialGcpServiceAccount)
			customerGroup.Post("/cloudconnect/google-cloud/workload-identity-federation-check", cloudConnect.CheckWorkloadIdentityFederationConnection)

			presentationMode := customerGroup.NewSubgroup("/presentation-mode", mid.AuthDoitEmployee())
			{
				presentationMode.Post("/toggle", presentationHandler.ChangePresentationMode, doitRoleCustomerSettingsAdmin, mid.ValidatePathParamNotEmpty("customerID"))

				presentationMode.Patch("/turn-off", tiers.TurnOffPresentationMode)
			}

			customerTiers := customerGroup.NewSubgroup("/tier", mid.AuthDoitEmployee(), doitRoleCustomerTieringAdmin)
			{
				customerTiers.Patch("", tiers.SetCustomerTiers)
			}

			flexSaveGroup := customerGroup.NewSubgroup("/flexsave")
			{
				flexSaveGroup.Patch("/orders/:orderID/activate", flexsaveAWS.ActivateFlexsaveOrderHandler, doitRoleFlexsaveAdmin)

				flexsaveGCPGroup := flexSaveGroup.NewSubgroup("/gcp")
				{
					flexsaveGCPGroup.Post("/enable", flexsaveGCP.Enable, mid.AssertCacheEnableAccess(permissionsService))
					flexsaveGCPGroup.Post("/disable", flexsaveGCP.Disable, mid.AssertCacheDisableAccess(permissionsService))
					flexsaveGCPGroup.Get("/can-disable", AWSResoldCache.CanDisable)
				}

				flexsaveAWSGroup := flexSaveGroup.NewSubgroup("/aws")
				{
					flexsaveAWSGroup.Post("/disable", AWSResoldCache.Disable, mid.AssertCacheDisableAccess(permissionsService))
					flexsaveAWSGroup.Get("/can-disable", AWSResoldCache.CanDisable)
				}
			}

			flexsaveSageMakerGroup := customerGroup.NewSubgroup("/flexsave-sagemaker")
			{
				flexsaveSageMakerGroup.Post("/enable", flexsaveSageMaker.Enable)
			}

			flexsaveRDSGroup := customerGroup.NewSubgroup("/flexsave-rds")
			{
				flexsaveRDSGroup.Post("/enable", flexsaveRDS.Enable)
			}

			// Ava metadata embeddings
			customerGroup.Post("/ava/customer-metadata", avaHandler.CreateMetadataTaskHandler)

			// Ramp Plan
			customerGroup.Post("/ramp-plan", rampPlan.UpdateRampPlanByID)
			customerGroup.Post("/create-ramp-plan", rampPlan.CreateRampPlan)

			// Datahub
			datahubGroup := customerGroup.NewSubgroup("/datahub", mid.AssertUserHasPermissions([]string{string(common.PermissionDataHubAdmin)}, a.conn))
			{
				datahubGroup.Post("/dataset", datahubHandler.CreateDataset)
				datahubGroup.Post("/events/raw", datahubHandler.AddRawEvents)
				datahubGroup.Get("/events/datasets", datahubHandler.GetCustomerDatasets)
				datahubGroup.Get("/events/datasets/:datasetName/batches", datahubHandler.GetCustomerDatasetBatches)
				datahubGroup.Delete("/events/datasets", datahubHandler.DeleteCustomerDatasets)
				datahubGroup.Delete("/events/datasets/:datasetName/batches", datahubHandler.DeleteDatasetBatches)
			}

			// Cloud Analytics
			cloudAnalyticsGroup := customerGroup.NewSubgroup("/analytics", mid.AssertUserHasPermissions([]string{string(common.PermissionCloudAnalytics)}, a.conn))
			{
				cloudAnalyticsGroup.Post("/preview", cloudAnalytics.Query)
				cloudAnalyticsGroup.Get("/metadata/:orgID", analyticsMetadataHandler.UpdateOrgsMetadata)
				cloudAnalyticsGroup.Post("/widgets", cloudAnalytics.UpdateCustomerDashboardReportWidgetsHandler)
				cloudAnalyticsGroup.Post("/widgets/dashboards", cloudAnalytics.UpdateDashboardsReportWidgetsHandler)

				splittingGroup := cloudAnalyticsGroup.NewSubgroup("/splitting")
				{
					splittingGroup.Post("/validate", splittingHandler.ValidateSplitRequest)
				}

				cloudAnalyticsReportGroup := cloudAnalyticsGroup.NewSubgroup("/reports")
				{
					cloudAnalyticsReportGroup.Delete("/deleteMany", reportHandler.DeleteManyHandler)

					cloudAnalyticsReportGroupWithID := cloudAnalyticsReportGroup.NewSubgroup("/:reportID")
					{
						cloudAnalyticsReportGroupWithID.Get("/image", cloudAnalytics.ReportImageHandler)

						cloudAnalyticsReportGroupWithID.Post("/query", cloudAnalytics.Query)
						cloudAnalyticsReportGroupWithID.Patch("/share", reportHandler.ShareReportHandler)
						cloudAnalyticsReportGroupWithID.Post("/widget", cloudAnalytics.RefreshReportWidgetHandler)
						cloudAnalyticsReportGroupWithID.Delete("/widget", cloudAnalytics.DeleteReportWidgetHandler)
						cloudAnalyticsReportGroupWithID.Patch("/widget", cloudAnalytics.UpdateReportWidgetHandler)

						cloudAnalyticsReportGroupWithID.Post("/schedule", cloudAnalytics.CreateScheduleHandler)
						cloudAnalyticsReportGroupWithID.Patch("/schedule", cloudAnalytics.UpdateScheduleHandler)
						cloudAnalyticsReportGroupWithID.Delete("/schedule", cloudAnalytics.DeleteScheduleHandler)
					}
				}

				cloudAnalyticsAttributionsGroup := cloudAnalyticsGroup.NewSubgroup("/attributions")
				{
					cloudAnalyticsAttributionsGroup.Post("", analyticsAttributions.CreateAttributionInternalHandler)
					cloudAnalyticsAttributionsGroup.Post("/preview", cloudAnalytics.Query)
					cloudAnalyticsAttributionsGroup.Delete("", analyticsAttributions.DeleteAttributions)
					cloudAnalyticsAttributionsGroup.Patch("/share", analyticsAttributions.UpdateAttributionSharingHandler)
					cloudAnalyticsAttributionsWithIDParam := cloudAnalyticsAttributionsGroup.NewSubgroup("/:id")
					{
						cloudAnalyticsAttributionsWithIDParam.Get("", analyticsAttributions.GetAttributionHandler)
						cloudAnalyticsAttributionsWithIDParam.Patch("", analyticsAttributions.UpdateAttributionInternalHandler)
						cloudAnalyticsAttributionsWithIDParam.Delete("", analyticsAttributions.DeleteAttributionHandler)
					}
				}

				cloudAnalyticsAttributionGroupsGroup := cloudAnalyticsGroup.NewSubgroup("/attribution-groups", mid.AssertUserHasPermissions([]string{string(common.PermissionCloudAnalytics)}, a.conn))
				{
					cloudAnalyticsAttributionGroupsGroup.Post("", analyticsAttributionGroups.CreateAttributionGroup)
					cloudAnalyticsAttributionGroupsGroup.Delete("", analyticsAttributionGroups.DeleteAttributionGroups)
					cloudAnalyticsAttributionGroupsGroup.Get("/metadata", analyticsMetadataHandler.AttributionGroupsMetadata)
					cloudAnalyticsAttributionGroupsWithIDParam := cloudAnalyticsAttributionGroupsGroup.NewSubgroup("/:attributionGroupID")
					{
						cloudAnalyticsAttributionGroupsWithIDParam.Patch("/share", analyticsAttributionGroups.ShareAttributionGroup)
						cloudAnalyticsAttributionGroupsWithIDParam.Put("", analyticsAttributionGroups.UpdateAttributionGroup)
						cloudAnalyticsAttributionGroupsWithIDParam.Delete("", analyticsAttributionGroups.DeleteAttributionGroup)
					}
				}

				cloudAnalyticsAlertsGroup := cloudAnalyticsGroup.NewSubgroup("/alerts")
				{
					cloudAnalyticsAlertsGroup.Delete("/deleteMany", analyticsAlerts.DeleteManyHandler)

					cloudAnalyticsAlertsGroupWithID := cloudAnalyticsAlertsGroup.NewSubgroup("/:alertID")
					{
						cloudAnalyticsAlertsGroupWithID.Patch("/share", analyticsAlerts.UpdateAlertSharingHandler)
						cloudAnalyticsAlertsGroupWithID.Delete("", analyticsAlerts.DeleteAlert)
					}
				}

				cloudAnalyticsBudgetGroup := cloudAnalyticsGroup.NewSubgroup("/budgets")
				{
					cloudAnalyticsBudgetGroup.Delete("/deleteMany", analyticsBudgets.DeleteManyHandler)

					cloudAnalyticsBudgetGroupWithID := cloudAnalyticsBudgetGroup.NewSubgroup("/:budgetID")
					{
						cloudAnalyticsBudgetGroupWithID.Post("/preview", cloudAnalytics.Query)
						cloudAnalyticsBudgetGroupWithID.Get("", cloudAnalytics.RefreshBudgetUsageDataHandler)
						cloudAnalyticsBudgetGroupWithID.Patch("/share", analyticsBudgets.UpdateBudgetSharingHandler)
						cloudAnalyticsBudgetGroupWithID.Patch("/update-enforced-by-metering", analyticsBudgets.UpdateBudgetEnforcedByMeteringHandler)
					}
				}

				cloudAnalyticsMetricsGroup := cloudAnalyticsGroup.NewSubgroup("/metrics")
				{
					cloudAnalyticsMetricsGroup.Post("/preview", cloudAnalytics.Query)
					cloudAnalyticsMetricsGroup.Delete("/", metricsHandler.DeleteMetricsHandler)
				}
			}

			amazonWebServicesGroup := customerGroup.NewSubgroup("/amazon-web-services")
			{
				amazonWebServicesGroup.Get("/service-limits", aws.UpdateCustomerLimitAWS)
				amazonWebServicesGroup.Get("/support-role", aws.GetSupportRole)
			}

			googleCloudGroup := customerGroup.NewSubgroup("/google-cloud")
			{
				googleCloudGroup.Get("/service-limits", googleCloud.UpdateCustomerLimitGCP)
				googleCloudGroup.Get("/folders", handlers.ListOrganizationFolders)
				googleCloudGroup.Post("/sandbox", handlers.CreateSandbox)

				// Transfer billing accounts
				googleCloudGroup.Post("/service-account", handlers.CreateServiceAccountForCustomer)
				googleCloudGroup.Post("/transfer-projects", handlers.TransferProjects)
				googleCloudGroup.Post("/test-iam", handlers.CheckServiceAccountPermissions)
			}

			// Recommendations
			customerGroup.Post("/recommender/update", googleCloud.UpdateCustomerRecommendations)
			customerGroup.Post("/recommender/change-type", googleCloud.ChangeMachineType)
			customerGroup.Post("/recommender/get-instance-status", googleCloud.GetInstanceStatus)
			customerGroup.Post("/recommender/stop-instance", googleCloud.StopInstance)
			customerGroup.Post("/recommender/start-instance", googleCloud.StartInstance)

			// Cloud Connect
			customerGroup.Get("/cloudconnect/health", cloudConnect.Health)
			customerGroup.Get("/cloudconnect/aws/health", handlers.AWSPermissionsHandler)
			customerGroup.Get("/cloudconnect/aws/health/:accountID", handlers.AWSPermissionsHandler)
			customerGroup.Get("/attachDashboard", publicdashboardsHandler.AttachAllDashboardsHandler)

			entityGroup := customerGroup.NewSubgroup("/entities/:entityID", mid.AuthEntityRequired())
			{
				// Payments
				entityGroup.Post("/setup-intents", stripe.CreateSetupIntentHandler)
				entityGroup.Post("/setup-session", stripe.CreateSetupSessionHandler)
				entityGroup.Get("/paymentMethods", stripe.GetPaymentMethodsHandler)
				entityGroup.Patch("/paymentMethods", stripe.PatchPaymentMethodHandler)
				entityGroup.Delete("/paymentMethods", stripe.DetachPaymentMethodHandler)

				// Invoices
				entityGroup.Post("/invoices/:invoiceID", stripe.PayInvoiceHandler)
				entityGroup.Post("/credit-card-fees", stripe.GetCreditCardProcessingFee)

				// Assets
				entityGroup.Post("/google-cloud", partnerSales.CreateBillingAccountHandler)
				entityGroup.Patch("/google-cloud/:billingAccountID", handlers.SetBillingAccountAdmins)
				entityGroup.Post("/google-cloud/:billingAccountID/send", handlers.SendBillingAccountInstructionsHandler)

				entityGroup.Post("/amazon-web-services/invite", aws.InviteAccount)
				entityGroup.Post("/amazon-web-services/create", aws.CreateAccount)

				// Invoice attributions
				entityGroup.Post("/sync-invoice-attributions", analyticsAttributionGroups.SyncEntityInvoiceAttributions)
			}

			iam := customerGroup.NewSubgroup("/iam")
			{
				orgs := iam.NewSubgroup("/orgs")
				{
					organizations := handlers.NewIAMOrganizations(a.log, a.conn)
					orgs.Delete("", organizations.DeleteIAMOrganizations)
				}
			}

			backfillGroup := customerGroup.NewSubgroup("/backfill")
			{
				backfillGroup.Put("/one-time-copy", cloudAnalytics.HandleCustomerBackfill) // Customer task worker
			}

			assetGroup := customerGroup.NewSubgroup("/assets")
			{
				assetGroup.Post("/google-cloud-direct", assets.CreateGoogleCloudDirectAssetHandler)
				assetGroup.Patch("/google-cloud-direct/:id", assets.UpdateGoogleCloudDirectAssetHandler)
				assetGroup.Delete("/google-cloud-direct/:id", assets.DeleteGoogleCloudDirectAssetHandler)

				assetGroup.Post("/amazon-web-services/account-access", awsAccountAccess.GetCreds)
				assetGroup.Get("/amazon-web-services/account-access/roles", awsAccountAccess.GetRoles)

				assetGroup.Post("/settings/:assetID", assets.UpdateAssetSettingsHandler, mid.AuthDoitEmployee())
				assetGroup.Delete("/settings/:assetID", assets.DeleteAssetSettingsHandler, mid.AuthDoitEmployee())
			}

			bigQueryGroup := customerGroup.NewSubgroup("/bigquery")
			{
				bigQueryHandler := handlers.NewBigQueryHandler(loggerProvider, a.conn)
				bigQueryGroup.Get("/query/:location/:project/:jobId", bigQueryHandler.GetQuery)
			}

			ssoProvidersGroup := customerGroup.NewSubgroup("/ssoproviders")
			{
				ssoProvidersGroup.Get("", ssoProviders.GetAllProvidersHandler)
				ssoProvidersGroup.Post("", ssoProviders.CreateProviderHandler)
				ssoProvidersGroup.Patch("", ssoProviders.UpdateProviderHandler)
			}

			licensesMicrosoftGroup := customerGroup.NewSubgroup("/licenses/microsoft/:licenseCustomerID")
			{
				licensesMicrosoftGroup.Post("/orders", microsoftLicensesHandler.LicenseOrderHandler)
				licensesMicrosoftGroup.Post("/subscriptions/:subscriptionID/changeQuantity", microsoftLicensesHandler.LicenseChangeQuantityHandler)
			}

			perksGroup := customerGroup.NewSubgroup("/perks")
			{
				perksGroup.Post("/register-interest-email", perksHandlers.SendRegisterInterestEmail)
			}

			customerGroup.Put("/clear-users-notifications", customerHandler.ClearCustomerUsersNotifications)
			customerGroup.Put("/restore-users-notifications", customerHandler.RestoreCustomerUsersNotifications)

			labelsGroup := customerGroup.NewSubgroup("/labels", mid.AssertUserHasPermissions([]string{string(common.PermissionLabelsManager)}, a.conn))
			{
				labelsGroup.Post("", labelsHandler.CreateLabel)
				labelsGroup.Patch("/:labelID", labelsHandler.UpdateLabel)
				labelsGroup.Delete("/:labelID", labelsHandler.DeleteLabel)
			}

			customerGroup.Put("/labels/assign", labelsHandler.AssignLabels)

			customerGroup.Patch("/customerTier", supportHandlers.ChangeCustomerTier)

			customerGroup.Post("/billing-export", billingDataExportHandler.HandleCustomerBillingExport)
		}

		marketplaceGroup := apiGroup.NewSubgroup("/marketplace")
		{
			marketplaceGCPGroup := marketplaceGroup.NewSubgroup("/gcp")
			{
				accountsGroup := marketplaceGCPGroup.NewSubgroup("/accounts/:accountID")
				{
					accountsGroup.Post("/approve", marketplace.ApproveAccount)
					accountsGroup.Post("/reject", marketplace.RejectAccount)
					accountsGroup.Post("/populate-billing-account", marketplace.PopulateSingleBillingAccount)
				}

				entitlementsGroup := marketplaceGCPGroup.NewSubgroup("/entitlements/:entitlementID")
				{
					entitlementsGroup.Post("/approve", marketplace.ApproveEntitlement)
					entitlementsGroup.Post("/reject", marketplace.RejectEntitlement)
				}
			}

			marketplaceAWSGroup := marketplaceGroup.NewSubgroup("/aws")
			{
				marketplaceAWSGroup.Post("/resolve-customer", awsMp.ResolveCustomer)
				marketplaceAWSGroup.Post("/entitlement-validation", awsMp.ValidateEntitlement)
			}
		}

		onePasswordGroup := apiGroup.NewSubgroup("/onepassword", mid.AuthDoitEmployee())
		{
			onePasswordGroup.Post("/:vaultID", onePassword.Create)
			onePasswordGroup.Patch("/:vaultID/:itemID/title", onePassword.UpdateTitle)
			onePasswordGroup.Patch("/:vaultID/:itemID/username", onePassword.UpdateUsername)
			onePasswordGroup.Delete("/:vaultID/:itemID", onePassword.Delete)
		}

		algoliaGroup := apiGroup.NewSubgroup("/algolia")
		{
			algoliaGroup.Get("/config", algolia.GetAlgoliaConfig)
		}

		tiersGroup := apiGroup.NewSubgroup("/tiers", mid.AuthDoitEmployee())
		{
			tiersGroup.Patch("/:id", tiers.UpdateTier, mid.AuthDoitEmployeeRole(a.conn, permissionsDomain.DoitRoleOwners))
		}
	}

	// EXTERNAL CMP API
	coreV1Group := web.NewGroup(app, "/core/v1", mid.ExternalAPIAuthMiddleware(), hasEntitlement(pkg.TiersFeatureKeyDoiTAPIAccess))
	{
		// Known Issues API group is legacy and exists for backward compatibility, cloudincidents is the new API
		knownIssuesGroup := coreV1Group.NewSubgroup("/knownissues", mid.AssertUserHasPermissions([]string{string(common.PermissionIssuesViewer)}, a.conn))
		{
			knownIssuesGroup.Get("", apiV1.ListKnownIssues, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodList, mixpanel.FeatureKnownIssues))
			knownIssuesGroup.Get("/:id", apiV1.GetKnownIssue, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodGetResults, mixpanel.FeatureKnownIssues))
		}

		cloudIncidentsGroup := coreV1Group.NewSubgroup("/cloudincidents", mid.AssertUserHasPermissions([]string{string(common.PermissionIssuesViewer)}, a.conn))
		{
			cloudIncidentsGroup.Get("", apiV1.ListKnownIssues, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodList, mixpanel.FeatureKnownIssues))
			cloudIncidentsGroup.Get("/:id", apiV1.GetKnownIssue, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodGetResults, mixpanel.FeatureKnownIssues))
		}
	}

	authV1Group := web.NewGroup(app, "/auth/v1", mid.ExternalAPIAuthMiddleware(), hasEntitlement(pkg.TiersFeatureKeyDoiTAPIAccess))
	{
		authV1Group.Get("/validate", auth.Validate)
	}

	billingV1Group := web.NewGroup(app, "/billing/v1", mid.ExternalAPIAuthMiddleware(), hasEntitlement(pkg.TiersFeatureKeyDoiTAPIAccess))
	{
		invoicesGroup := billingV1Group.NewSubgroup("/invoices", mid.AssertUserHasPermissions([]string{string(common.PermissionInvoices)}, a.conn))
		{
			invoicesGroup.Get("", apiV1.ListInvoices, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodList, mixpanel.FeatureInvoices))
			invoicesGroup.Get("/:id", apiV1.GetInvoice, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodGetResults, mixpanel.FeatureInvoices))
		}

		assetsGroup := billingV1Group.NewSubgroup("/createAsset", mid.ExternalAPIAssertCustomerTypeProductOnly(), mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodCreate, mixpanel.FeatureAssets), mid.AssertUserHasPermissions([]string{string(common.PermissionAssetsManager)}, a.conn))
		{
			assetsGroup.Post("", apiV1.CreateAsset)
		}
	}

	anomaliesV1Group := web.NewGroup(app, "/anomalies/v1", mid.ExternalAPIAuthMiddleware(), hasEntitlement(pkg.TiersFeatureKeyDoiTAPIAccess), mid.AssertUserHasPermissions([]string{string(common.PermissionAnomaliesViewer)}, a.conn))
	{
		anomaliesV1Group.Get("", apiV1.ListAnomalies, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodList, mixpanel.FeatureAnomalies))
		anomaliesV1Group.Get("/:id", apiV1.GetAnomaly, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodGetResults, mixpanel.FeatureAnomalies))
	}

	analyticsV1Group := web.NewGroup(app, "/analytics/v1", mid.ExternalAPIAuthMiddleware(), hasEntitlement(pkg.TiersFeatureKeyDoiTAPIAccess), mid.AssertUserHasPermissions([]string{string(common.PermissionCloudAnalytics)}, a.conn))
	{
		reportsV1Group := analyticsV1Group.NewSubgroup("/reports")
		{
			reportsV1Group.Get("", apiV1.ListReports, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodList, mixpanel.FeatureReports))
			reportsV1Group.Get("/:id", apiV1.RunReport, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodGetResults, mixpanel.FeatureReports))
			reportsV1Group.Get("/:id/config", reportHandler.GetReportConfigExternalHandler, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodGetConfig, mixpanel.FeatureReports))
			reportsV1Group.Post("", reportHandler.CreateReportExternalHandler, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodCreate, mixpanel.FeatureReports))
			reportsV1Group.Post("/query", reportHandler.RunReportFromExternalConfig, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodQuery, mixpanel.FeatureReports))
			reportsV1Group.Patch("/:id", reportHandler.UpdateReportExternalHandler, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodUpdate, mixpanel.FeatureReports))
			reportsV1Group.Delete("/:id", reportHandler.DeleteReportExternalHandler, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodDelete, mixpanel.FeatureReports))
		}

		budgetsV1Group := analyticsV1Group.NewSubgroup("/budgets", mid.AssertUserHasPermissions([]string{string(common.PermissionBudgetsManager)}, a.conn))
		{
			budgetsV1Group.Get("", analyticsBudgets.ExternalAPIListBudgets, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodList, mixpanel.FeatureBudgets))
			budgetsV1Group.Get("/:id", analyticsBudgets.ExternalAPIGetBudget, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodGetConfig, mixpanel.FeatureBudgets))
			budgetsV1Group.Post("", apiV1.CreateBudgetHandler, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodCreate, mixpanel.FeatureBudgets))
			budgetsV1Group.Patch("/:id", apiV1.UpdateBudgetHandler, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodUpdate, mixpanel.FeatureBudgets))
			budgetsV1Group.Delete("/:id", apiV1.DeleteBudgetHandler, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodDelete, mixpanel.FeatureBudgets))
		}

		attributionsV1Group := analyticsV1Group.NewSubgroup("/attributions", mid.AssertUserHasPermissions([]string{string(common.PermissionAttributionsManager)}, a.conn))
		{
			attributionsV1Group.Post("", analyticsAttributions.CreateAttributionExternalHandler, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodCreate, mixpanel.FeatureAttributions))
			attributionsV1Group.Patch("/:id", analyticsAttributions.UpdateAttributionExternalHandler, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodUpdate, mixpanel.FeatureAttributions))
			attributionsV1Group.Get("", analyticsAttributions.ListAttributionsExternalHandler, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodList, mixpanel.FeatureAttributions))
			attributionsV1Group.Get("/:id", analyticsAttributions.GetAttributionExternalHandler, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodGetConfig, mixpanel.FeatureAttributions))
			attributionsV1Group.Delete("/:id", analyticsAttributions.DeleteAttributionExternalHandler, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodDelete, mixpanel.FeatureAttributions))
		}

		attributionGroupsV1Group := analyticsV1Group.NewSubgroup("/attributiongroups")
		{
			attributionGroupsV1Group.Get("/:id", analyticsAttributionGroups.ExternalAPIGetAttributionGroup, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodGetConfig, mixpanel.FeatureAttributionGroups))
			attributionGroupsV1Group.Get("", analyticsAttributionGroups.ExternalAPIListAttributionGroups, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodList, mixpanel.FeatureAttributionGroups))
			attributionGroupsV1Group.Post("", analyticsAttributionGroups.ExternalAPICreateAttributionGroup, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodCreate, mixpanel.FeatureAttributionGroups))
			attributionGroupsV1Group.Patch("/:id", analyticsAttributionGroups.ExternalAPIUpdateAttributionGroup, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodUpdate, mixpanel.FeatureAttributionGroups))
			attributionGroupsV1Group.Delete("/:id", analyticsAttributionGroups.ExternalAPIDeleteAttributionGroup, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodDelete, mixpanel.FeatureAttributionGroups))
		}

		dimensionsV1Group := analyticsV1Group.NewSubgroup("/dimensions")
		{
			dimensionsV1Group.Get("", analyticsMetadataHandler.ExternalAPIListDimensions, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodList, mixpanel.FeatureDimensions))
		}

		dimensionV1Group := analyticsV1Group.NewSubgroup("/dimension")
		{
			dimensionV1Group.Get("", analyticsMetadataHandler.ExternalAPIGetDimensions, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodGetConfig, mixpanel.FeatureDimensions))
		}

		alertsV1Group := analyticsV1Group.NewSubgroup("/alerts")
		{
			alertsV1Group.Get("", analyticsAlerts.ExternalAPIListAlerts, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodList, mixpanel.FeatureAlerts))
			alertsV1Group.Get("/:id", analyticsAlerts.ExternalAPIGetAlert, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodGetConfig, mixpanel.FeatureAlerts))
			alertsV1Group.Delete("/:id", analyticsAlerts.ExternalAPIDeleteAlert, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodDelete, mixpanel.FeatureAlerts))
			alertsV1Group.Post("", analyticsAlerts.ExternalAPICreateAlert, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodCreate, mixpanel.FeatureAlerts))
			alertsV1Group.Patch("/:id", analyticsAlerts.ExternalAPIUpdateAlert, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodUpdate, mixpanel.FeatureAlerts))
		}
	}

	supportV1Group := web.NewGroup(app, "/support/v1/metadata", mid.ExternalAPIAuthMiddleware(), hasEntitlement(pkg.TiersFeatureKeyDoiTAPIAccess), mid.AssertUserHasPermissions([]string{string(common.PermissionSupportRequester)}, a.conn))
	{
		supportV1Group.Get("/platforms", supportHandlers.ListPlatforms, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodList, mixpanel.FeatureSupport))
		supportV1Group.Get("/products", supportHandlers.ListProducts, mid.ExternalAPIMixpanel(mixpanelHandler, mixpanel.MethodList, mixpanel.FeatureSupport))
	}

	// Handle Pub/Sub message(s) from Cloud Functions
	pubsubGroup := web.NewGroup(app, "/pubsub/v1")
	{
		pubsubGroup.Post("/notify-aws-account-update", accountHandler.HandleNotifyAccountUpdate)
	}

	zapierWebhookGroup := web.NewGroup(app, "/zapier/v1", mid.ExternalAPIAuthMiddleware(), hasEntitlement(pkg.TiersFeatureKeyZapierIntegration))
	{
		zapierWebhookGroup.Post("/subscribe", webhookSubscriptionHandlers.Create)
		zapierWebhookGroup.Delete("/unsubscribe", webhookSubscriptionHandlers.Delete)

		zapierMocks := zapierWebhookGroup.NewSubgroup("/restHookMocks")
		{
			zapierMocks.Get("/alertNotification", webhookSubscriptionHandlers.GetAlertsMock)
			zapierMocks.Get("/budgetThreshold", webhookSubscriptionHandlers.GetBudgetsMock)
		}
	}

	customersV1Group := web.NewGroup(app, "/customers/v1", mid.ExternalAPIAuthMiddleware(), hasEntitlement(pkg.TiersFeatureKeyDoiTAPIAccess))
	{
		customersV1Group.Get("/accountTeam", customerHandler.ListAccountManagers)
	}

	contractGroup := apiGroup.NewSubgroup("/contract")
	{
		contractGroup.Post("/create", contractHandler.AddContract)
		contractGroup.Post("/cancel/:id", contractHandler.CancelContract)
		contractGroup.Post("/update/:id", contractHandler.UpdateContract)
		contractGroup.Post("/delete/:id", contractHandler.DeleteContract, mid.AuthDoitEmployeeRole(a.conn, permissionsDomain.DoitRoleContractOwner))
	}

	contractV1Group := tasksGroup.NewSubgroup("/contract")
	{
		// Agregate invoice data for solve/navigator
		contractV1Group.Post("/aggregate-invoice-data/all", contractHandler.AggregateAllInvoiceData)
		contractV1Group.Post("/aggregate-invoice-data/:id", contractHandler.AggregateInvoiceData)

		// Refresh solve/navigator contracts
		contractV1Group.Post("/refresh/:customerID", contractHandler.Refresh)
		contractV1Group.Post("/refresh/all", contractHandler.RefreshAll)

		// Export contracts to BQ
		contractV1Group.Get("/export", contractHandler.ExportContracts)

		contractV1Group.Post("/google-cloud/update-support", contractHandler.UpdateGoogleCloudContractsSupport)
	}

	return app
}
