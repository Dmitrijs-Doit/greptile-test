package scripts

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/go-multierror"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/scripts/customer"
)

type ScriptFunc func(ctx *gin.Context) []error

type Scripts struct {
	log  logger.Provider
	conn *connection.Connection

	functionsMap map[string]ScriptFunc
}

func NewScripts(log logger.Provider, conn *connection.Connection) *Scripts {
	flexsaveScripts := NewFlexsaveScripts(log, conn)
	slackScripts := NewSlackScripts(log, conn)
	customerScripts := customer.NewCustomerScripts(log, conn)
	tiersScripts := NewTiersScripts(log, conn)

	functionsMap := map[string]ScriptFunc{
		"mergeCustomers":                        customerScripts.MergeCustomers,
		"reassignCustomerGcpProjects":           reassignCustomerGcpProjects,
		"backfillBillingLineItems":              flexsaveScripts.BackfillBillingLineItems,
		"catalogLoader":                         CatalogLoader,
		"copyFirestoreDoc":                      CopyFirestoreDoc,
		"createCustomersTenants":                CreateTenants,
		"createPublicDashboards":                AddPublicDashboard,
		"updateFirestoreField":                  UpdateFirestoreField,
		"removeDeprecatedPermissions":           RemoveDeprecatedPermissions,
		"removeEarlyAccessFeature":              RemoveEarlyAccessFeature,
		"updateFlexSaveCustomerDiscounts":       UpdateFlexSaveCustomerDiscount,
		"updatePriorityCompanies":               UpdatePriorityCompanies,
		"updateAnalyticsCollaborators":          UpdateAnalyticsCollaborators,
		"updateCloudHealthPricebooksOptions":    updateCloudHealthPricebooksOptions,
		"updateCloudConnectPermissions":         updateCloudConnectPermissions,
		"updateReportsToPresetWidgets":          UpdateReportsToPresetWidgetsHandler,
		"addWidgetToDashboard":                  AddWidgetToDashboard,
		"updateFlexsaveConfigTimeEnabled":       UpdateFlexsaveConfigTimeEnabled,
		"updateFlexsaveConfigTimeDisabled":      UpdateFlexsaveConfigTimeDisabled,
		"migrateUsersToTenants":                 MigrateUsersToCustomerTenant,
		"migrateCollectionField":                MigrateCollectionField,
		"deleteKnownIssueFromAllSharedChannels": DeleteKnownIssueFromAllSharedChannels,
		"syncCustomerTenantIDs":                 syncCustomerTenantIDs,
		"deleteDuplicateInvoiceAdjustments":     DeleteDuplicateInvoiceAdjustments,
		"flexsaveCapitalizationFix":             flexsaveCapitalizationFix,
		"deleteAnalyticsFirestoreDocs":          DeleteAnalyticsFirestoreDocs,
		"newDoiTRole":                           NewDoiTRole,
		"updateRolesForFSSA":                    UpdateRolesForFSSA,
		"updateCustomersSAM":                    UpdateCustomersSAM,
		"modifyCustomerAuth":                    ModifyCustomerAuth,
		"updateCustomersAM":                     UpdateCustomersAM,
		"updateZendeskTimeZones":                UpdateZendeskTimeZones,
		"updateCustomerAuth":                    UpdateCustomerSSOSettings,
		"updateAlertsOrgs":                      UpdateAlertsOrgs,
		"removeSuperQueryPermission":            removeSuperQueryPermission,
		"bigqueryJobsCancel":                    bigqueryJobsCancel,
		"updateStandaloneField":                 UpdateStandaloneField,
		"setCustomerClaims":                     SetCustomClaims,
		"duplicateTenantsWithNewDomainEmails":   DuplicateTenantsWithNewDomainEmails,
		"updateCustomerSubscribers":             UpdateCustomerSubscribers,
		"terminateCustomers":                    TerminateCustomers,
		"addGCPCustomerSegment":                 AddGCPCustomerSegment,
		"addDeleteFieldOnPerks":                 AddDeleteFieldOnPerks,
		"finalizeInvoiceAdjustments":            FinalizeInvoiceAdjustments,
		"updateReasonCantEnable":                UpdateReasonCantEnableForTerminatedCustomers,
		"getSlackWorkspace":                     slackScripts.GetSlackWorkspace,
		"subscribeExistingSharedChannel":        slackScripts.SubscribeExistingSharedChannel,
		"addSlackBotToChannels":                 slackScripts.AddBotToChannels,
		"setBudgetsSlackChannels":               slackScripts.SetBudgetsSlackChannels,
		"addEarlyAccessFeature":                 AddEarlyAccessFeature,
		"printCustomersWithAuthPermission":      PrintCustomersWithAuthManagerPermission,
		"stopTerminatedCustomerNotifications":   StopTerminatedCustomerNotifications,
		"addPerksTimestamps":                    AddPerksTimestamps,
		"migrateStripeEUREntities":              MigrateStripeEUREntities,
		"populateQuickLinks":                    PopulateQuickLinks,
		"populateNotificationsDefinitions":      PopulateNotificationsDefinitions,
		"pushDataToAlgolia":                     PushDataToAlgolia,
		"populateDiscoveryTiles":                PopulateDiscoveryTiles,
		"DeleteInvalidRampPlans":                DeleteInvalidRampPlans,
		"fbodStatistics":                        FbodStatistics,
		"removeAccountManagersDuplications":     RemoveAccountManagersDuplications,
		"addLevelKnownIssuesAWS":                AddLevelKnownIssuesAWS,
		"removeWidget":                          RemoveWidget,
		"removeOrphanTenants":                   RemoveOrphanTenants,
		"doc2Json":                              Doc2Json,
		"changeAccountManagerCompany":           ChangeAccountManagerCompany,
		"setAllGCPSaaSCloudConnectNotified":     SetAllGCPSaaSCloudConnectNotified,
		"encryptWithDevKMSKey":                  EncryptWithDevKms,
		"createFlexsaveInvoiceAdjustment":       createFlexsaveInvoiceAdjustment,
		"addExpireByToNonFinalInvoices":         AddExpireByToNonFinalInvoices,
		"checkForMetricInReports":               CheckForMetricInReports,
		"clearIssuedAtInvoiceFlag":              ClearInvoiceIssuedAtBasedOnTimestampRange,
		"clearSpecificEntIssuedAtInvFlag":       ClearInvoiceIssuedAtForSpecificCustomers,
		"importEntitlements":                    ImportEntitlements,
		"copyEntitlements":                      CopyEntitlements,
		"copyTiers":                             CopyTiers,
		"assignEntitlementsToTier":              AssignEntitlementsToTier,
		"renameCustomerTierType":                RenameCustomerTierType,
		"setCustomersTier":                      SetCustomersTier,
		"syncStripeCustomers":                   SyncStripeCustomers,
		"updateOldInvoicesNotificationSent":     UpdateOldInvoicesNotificationSent,
		"backfilTrialTierContracts":             tiersScripts.BackfilTrialTierContracts,
		"setCustomersToHeritageTier":            tiersScripts.SetCustomersToHeritageTier,
		"setCustomersToEmtpyTier":               tiersScripts.SetCustomersToEmptyTier,
		"addEntitlementsToPresetReports":        AddEntitlementsToPresetReports,
		"addEntitlementsToPresetAttributions":   AddEntitlementsToPresetAttributions,
		"migrateSaasRoles":                      MigrateSaasRoles,
		"addBigqueryEditionsPermissionsGroup":   AddBigqueryEditionsPermissionsGroup,
		"emailFromCSV":                          EmailFromCSV,
		"forceSnapshotBillingTables":            ForceSnapshotBillingTables,
	}

	return &Scripts{
		log,
		conn,
		functionsMap,
	}
}

func concatenateErrors(errs []error) error {
	var err *multierror.Error

	for _, e := range errs {
		err = multierror.Append(err, e)
	}

	return err.ErrorOrNil()
}

func (s *Scripts) HandleScript(ctx *gin.Context, scriptName string) error {
	invokedScript, ok := s.functionsMap[scriptName]
	if !ok {
		return errors.New("invalid script name")
	}

	res := invokedScript(ctx)

	if err := concatenateErrors(res); err != nil {
		return err
	}

	return nil
}
