package consts

import "time"

const (
	IntegrationsCollection                    string = "integrations"
	GCPFlexsaveStandaloneDoc                  string = "gcp-flexsave-standalone-pipeline"
	InternalUpdateManagerCollection           string = "internal-update-manager"
	ExternalUpdateManagerCollection           string = "external-update-manager"
	InternalTasksCollection                   string = "internal-update-tasks"
	ExternalTasksCollection                   string = "external-update-tasks"
	RowsValidator                             string = "rows-validator"
	InternalUpdateManagerDoc                  string = "internalUpdateManager"
	ExternalUpdateManagerDoc                  string = "externalUpdateManager"
	CopyToUnifiedTableJobPrefixTemplate       string = "IN-ToUnified-%d"
	FromLocalTableToTmpTableJobPrefixTemplate string = "IN-ToTmp-%s-%d"
	MarkVerifiedTmpTableJobPrefixTemplate     string = "IN-Ver-%s-%d"
	ToBucketJobPrefixTemplate                 string = "EX-ToBuc-%s-%d"
	FromBucketJobPrefixTemplate               string = "EX-FromBuc-%s-%d"
	DeleteRowsOfBATemplate                    string = "DEL-BA-%s"

	ExternalManagerMaxDuration           time.Duration = time.Duration(time.Minute * 10)
	WaitForExternalJobToFinish           time.Duration = time.Duration(time.Minute * 20)
	WaitForExternalJobToFinishOnBoarding time.Duration = time.Duration(time.Hour * 5)

	InternalManagerMaxDuration                     time.Duration = time.Duration(time.Minute * 85)
	InternalTaskMaxDuration                        time.Duration = time.Duration(time.Minute * 40)
	InternalTaskMaxExtentionDuration               time.Duration = time.Duration(time.Minute * 5)
	WaitForJobOnTaskMaxDuration                    time.Duration = time.Duration(time.Minute * 15)
	WaitForInternalFlowToFinishDeletingBillingData time.Duration = time.Duration(time.Hour * 10)

	InternalManagerRecoverMaxDuration time.Duration = time.Duration(time.Minute * 5)
	OnboardingSegmentIntervalInMonths               = 2
	FetchBufferDuration                             = time.Duration(time.Minute * 30)

	BillingProjectProd          string = "doitintl-cmp-gcp-data"
	BillingProjectDev           string = "doitintl-cmp-dev"
	BillingProjectOk8topus      string = "doit-ok8topus"
	BillingProjectCutterfish    string = "cmp-cuttlefish"
	BillingProjectSkuid         string = "doitintl-skuid"
	BillingProjectMeDoitIntlCom string = "me-doit-intl-com"

	LocalBillingDataset                string = "flexsave_standalone_local_gcp_billing"             //"flexsave_gcp_billing_raw"
	AlternativeLocalBillingDataset     string = "flexsave_standalone_alternative_local_gcp_billing" //"flexsave_gcp_billing_raw"
	LocalBillingTablePrefix            string = "doitintl_billing_export_v1"
	AlternativeLocalBillingTablePrefix string = "doitintl_alternative_billing_export_v1"
	UnifiedGCPBillingDataset           string = "unified_gcp_billing"

	UnifiedRawBillingTablePrefix        string = "tmp_gcp_billing"
	UnifiedAlternativeRawBillingTable   string = "alternative_tmp_gcp_billing"
	UnifiedGCPRawTable                  string = "unified_gcp_raw_billing"
	UnifiedAlternativeGCPRawTable       string = "unified_alternative_gcp_raw_billing"
	UnifiedGCPRawTableTemplate          string = "unified_gcp_raw_billing_template"
	AlternativeUnifiedGCPBillingDataset string = "alternative_unified_gcp_billing"

	ResellRawBillingDataset   string = "gcp_billing"
	ResellRawBillingTable     string = "gcp_raw_billing"
	OldestRawBillingPartition string = "2017-10-27"

	ResellRawBillingDetailedTable string = "gcp_raw_billing_v1"

	DoitLocation        string = "US"
	TmpTableDescription string = "tmp billing aggregated table"

	BucketPrefix               string = "flexsave-billing-export"
	PartitionFieldFormat       string = "2006-01-02"
	Partition2FieldFormat      string = "20060102"
	ExportTimeLayout           string = "2006-01-02 15:04:05"
	ExportTimeLayoutWithMillis string = "2006-01-02 15:04:05.000000"
	//2006-01-02T15:04:05.000

	ContextExpirationTimeKey string = "expirationTime"
	IsDebugKey               string = "isDebug"

	CtxInternalManagerTemplate string = "Internal-Manager-#%d"
	CtxExternalManagerTemplate string = "External-Manager-#%d"
	CtxInternalTaskTemplate    string = "Internal-Task-#%d-BA%s"
	CtxExternalTaskTemplate    string = "External-Task-#%d-BA%s"
	CtxDeleteAllTemplate       string = "Delete-All"
	CtxDeleteBATemplate        string = "Delete-BA-#%s"

	CustomerTypeDummy string = "dummy"
	CustomerTypeField string = "customer_type"

	// DedicatedRole is the Flexsave Standalone Billing Pipeline Storage Object Writer role.
	DedicatedRole = "standalone_storage_writer"

	// AnalyticsLoobackInterval is the size of the time window we use for usage data analysis.
	AnalyticsLoobackInterval string = "INTERVAL 1 WEEK"

	AnalyticsDataset                      string = "gcp_billing"
	AnalyticsRewritesMappingTableProd     string = "rewrites_mapping"
	AnalyticsRewritesMappingTableNonProd  string = "rewrites_mapping_nonprod"
	AnaliyticsFreshnessReportTableProd    string = "freshness_report"
	AnaliyticsFreshnessReportTableNonProd string = "freshness_report_nonprod"

	MetadataOperationMaxRetries      = 5
	MetadataOperationFirstRetryDelay = time.Second * 3
)
