package domain

import (
	"context"

	"cloud.google.com/go/bigquery"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

type QueryOrigin = string

const (
	QueryOriginCtxKey = "queryRequestOrigin"

	QueryOriginClient               QueryOrigin = "ui"
	QueryOriginClientReservation    QueryOrigin = "ui-reservation"
	QueryOriginReportsAPI           QueryOrigin = "api"
	QueryOriginSlackUnfurl          QueryOrigin = "slack-unfurl"
	QueryOriginScheduledReports     QueryOrigin = "scheduled"
	QueryOriginWidgets              QueryOrigin = "widget"
	QueryOriginOthers               QueryOrigin = "other"
	QueryOriginInvoicingGcp         QueryOrigin = "invoicing-gcp"
	QueryOriginInvoicingAws         QueryOrigin = "invoicing-aws"
	QueryOriginInvoicingAzure       QueryOrigin = "invoicing-azure"
	QueryOriginAlerts               QueryOrigin = "alert"
	QueryOriginBudgets              QueryOrigin = "budget"
	QueryOriginDigest               QueryOrigin = "digest"
	QueryOriginRampPlan             QueryOrigin = "ramp-plan"
	QueryOriginAnomalies            QueryOrigin = "anomalies"
	QueryOriginTablesMgmtAws        QueryOrigin = "tables-mgmt-aws"
	QueryOriginTablesMgmtGcp        QueryOrigin = "tables-mgmt-gcp"
	QueryOriginTablesMgmtAzure      QueryOrigin = "tables-mgmt-azure"
	QueryOriginPresentationModeSync QueryOrigin = "demo-mode-sync"
	QueryOriginBigQueryLens         QueryOrigin = "bq-lens"
	QueryOriginDataHub              QueryOrigin = "datahub"
	QueryOriginMetadataGcp          QueryOrigin = "metadata-gcp"
	QueryOriginMetadataAws          QueryOrigin = "metadata-aws"
	QueryOriginMetadataAzure        QueryOrigin = "metadata-azure"
	QueryOriginMetadataDataHub      QueryOrigin = "metadata-datahub"
	QueryOriginMetadataBqLens       QueryOrigin = "metadata-bqlens"

	OnDemandProdProject         = "doitintl-cmp-online-reports"
	uiReservationProdProject    = "doit-bq-analytics-client"
	uiReservationDevProject     = "doit-bq-analytics-client-dev"
	widgetsProdProject          = "doit-bq-analytics-widgets"
	widgetsDevProject           = "doit-bq-analytics-widgets-dev"
	apiProdProject              = "doit-bq-analytics-api"
	apiDevProject               = "doit-bq-analytics-api-dev"
	alertsProdProject           = "doit-bq-analytics-alerts"
	alertsDevProject            = "doit-bq-analytics-alerts-dev"
	digestProdProject           = "doit-bq-analytics-digests"
	digestDevProject            = "doit-bq-analytics-digests-dev"
	scheduledReportsProdProject = "doit-bq-analytics-schedule"
	scheduledReportsDevProject  = "doit-bq-analytics-schedule-dev"
	metadataProdProject         = "doit-bq-analytics-metadata"
	metadataDevProject          = "doit-bq-analytics-metadata-dev"
	rampPlansProdProject        = "doit-bq-ramp-plans"
	rampPlansDevProject         = "doit-bq-ramp-plans-dev"
	invoicingAwsProdProject     = "doit-bq-invoicing-aws"
	invoicingAwsDevProject      = "doit-bq-invoicing-aws-dev"
	invoicingGcpProdProject     = "doit-bq-invoicing-gcp"
	invoicingGcpDevProject      = "doit-bq-invoicing-gcp-dev"
	invoicingAzureProdProject   = "doit-bq-invoicing-azure"
	invoicingAzureDevProject    = "doit-bq-invoicing-azure-dev"
	tablesAwsProdProject        = "doit-bq-ca-tables-aws"
	tablesAwsDevProject         = "doit-bq-ca-tables-aws-dev"
	tablesGcpProdProject        = "doit-bq-ca-tables-gcp"
	tablesGcpDevProject         = "doit-bq-ca-tables-gcp-dev"
	tablesAzureProdProject      = "doit-bq-ca-tables-azure"
	tablesAzureDevProject       = "doit-bq-ca-tables-azure-dev"
	anomaliesProdProject        = "doitintl-cmp-anomaly-detection"
	anomaliesDevProject         = "doitintl-cmp-anomaly-dev"
)

var (
	prodProjects = []string{
		OnDemandProdProject,
		uiReservationProdProject,
		apiProdProject,
		alertsProdProject,
		digestProdProject,
		scheduledReportsProdProject,
		widgetsProdProject,
		rampPlansProdProject,
		invoicingAwsProdProject,
		invoicingGcpProdProject,
		invoicingAzureProdProject,
		tablesAwsProdProject,
		tablesGcpProdProject,
		tablesAzureProdProject,
		anomaliesProdProject,
		metadataProdProject,
	}

	devProjects = []string{
		uiReservationDevProject,
		apiDevProject,
		alertsDevProject,
		digestDevProject,
		scheduledReportsDevProject,
		widgetsDevProject,
		rampPlansDevProject,
		invoicingAwsDevProject,
		invoicingGcpDevProject,
		invoicingAzureDevProject,
		tablesAwsDevProject,
		tablesGcpDevProject,
		tablesAzureDevProject,
		anomaliesDevProject,
		metadataDevProject,
	}

	prodProjectPerOrigin = map[QueryOrigin]string{
		QueryOriginClient:            OnDemandProdProject,
		QueryOriginClientReservation: uiReservationProdProject,
		QueryOriginWidgets:           widgetsProdProject,
		QueryOriginReportsAPI:        apiProdProject,
		QueryOriginSlackUnfurl:       apiProdProject,
		QueryOriginAlerts:            alertsProdProject,
		QueryOriginBudgets:           alertsProdProject,
		QueryOriginDigest:            digestProdProject,
		QueryOriginScheduledReports:  scheduledReportsProdProject,
		QueryOriginRampPlan:          rampPlansProdProject,
		QueryOriginInvoicingAws:      invoicingAwsProdProject,
		QueryOriginInvoicingGcp:      invoicingGcpProdProject,
		QueryOriginInvoicingAzure:    invoicingAzureProdProject,
		QueryOriginTablesMgmtAws:     tablesAwsProdProject,
		QueryOriginTablesMgmtGcp:     tablesGcpProdProject,
		QueryOriginTablesMgmtAzure:   tablesAzureProdProject,
		QueryOriginAnomalies:         anomaliesProdProject,
		QueryOriginMetadataGcp:       metadataProdProject,
		QueryOriginMetadataAws:       metadataProdProject,
		QueryOriginMetadataAzure:     metadataProdProject,
		QueryOriginMetadataDataHub:   metadataProdProject,
		QueryOriginMetadataBqLens:    metadataProdProject,
	}

	devProjectPerOrigin = map[QueryOrigin]string{
		QueryOriginClient:            uiReservationDevProject,
		QueryOriginClientReservation: uiReservationDevProject,
		QueryOriginWidgets:           widgetsDevProject,
		QueryOriginReportsAPI:        apiDevProject,
		QueryOriginSlackUnfurl:       apiDevProject,
		QueryOriginAlerts:            alertsDevProject,
		QueryOriginBudgets:           alertsDevProject,
		QueryOriginDigest:            digestDevProject,
		QueryOriginScheduledReports:  scheduledReportsDevProject,
		QueryOriginRampPlan:          rampPlansDevProject,
		QueryOriginInvoicingAws:      invoicingAwsDevProject,
		QueryOriginInvoicingGcp:      invoicingGcpDevProject,
		QueryOriginInvoicingAzure:    invoicingAzureDevProject,
		QueryOriginTablesMgmtAws:     tablesAwsDevProject,
		QueryOriginTablesMgmtGcp:     tablesGcpDevProject,
		QueryOriginTablesMgmtAzure:   tablesAzureDevProject,
		QueryOriginAnomalies:         anomaliesDevProject,
		QueryOriginMetadataGcp:       metadataDevProject,
		QueryOriginMetadataAws:       metadataDevProject,
		QueryOriginMetadataAzure:     metadataDevProject,
		QueryOriginMetadataDataHub:   metadataDevProject,
		QueryOriginMetadataBqLens:    metadataDevProject,
	}
)

// QueryOriginFromContext returns the query origin stored in the context.
// If not set or is not valid, it defaults to "other"
func QueryOriginFromContext(ctx context.Context) QueryOrigin {
	if origin, ok := ctx.Value(QueryOriginCtxKey).(QueryOrigin); ok {
		return origin
	}

	return QueryOriginOthers
}

func QueryOriginFromContextOrFallback(ctx context.Context, fallback QueryOrigin) QueryOrigin {
	if origin, ok := ctx.Value(QueryOriginCtxKey).(QueryOrigin); ok {
		return origin
	}

	return fallback
}

func QueryProjectForOrigin(origin QueryOrigin) (string, bool) {
	var (
		projectID string
		ok        bool
	)

	if common.Production {
		projectID, ok = prodProjectPerOrigin[origin]
	} else {
		projectID, ok = devProjectPerOrigin[origin]
	}

	return projectID, ok
}

// CloudAnalyticsBQProjects returns a slice of projects for which we want a bigquery client.
func CloudAnalyticsBQProjects() []string {
	if common.Production {
		return prodProjects
	}

	return devProjects
}

// Bigquery returns a bigquery client for the corresponding origin stored in the context.
// If a project-specific bq client is not avilable the default is returned and the
// flag is set to false.
func Bigquery(ctx context.Context, conn *connection.Connection) (*bigquery.Client, bool) {
	origin := QueryOriginFromContext(ctx)
	return BigqueryForOrigin(ctx, origin, conn)
}

// BigqueryForOrigin returns a bigquery client for the corresponding origin.
// If a project-specific bq client is not avilable the default is returned and the
// flag is set to false.
func BigqueryForOrigin(ctx context.Context, origin string, conn *connection.Connection) (*bigquery.Client, bool) {
	if projectID, ok := QueryProjectForOrigin(origin); ok {
		return conn.BigqueryForProject(projectID)
	}

	return conn.Bigquery(ctx), false
}

func mapOriginToHouse(origin QueryOrigin) common.House {
	switch origin {
	case QueryOriginPresentationModeSync:
		return common.HouseGrowth
	case QueryOriginInvoicingAws, QueryOriginInvoicingGcp, QueryOriginInvoicingAzure, QueryOriginAnomalies:
		return common.HouseData
	case QueryOriginDigest:
		return common.HousePlatform
	default:
		return common.HouseAdoption
	}
}

func mapOriginToFeature(origin QueryOrigin) common.Feature {
	switch origin {
	case QueryOriginRampPlan:
		return common.FeatureRampPlans
	case QueryOriginAnomalies:
		return common.FeatureAnomalies
	case QueryOriginBigQueryLens:
		return common.FeatureBQLens
	default:
		return common.FeatureCloudAnalytics
	}
}

func mapOriginToModule(origin QueryOrigin) common.Module {
	switch origin {
	case QueryOriginTablesMgmtAws, QueryOriginTablesMgmtGcp, QueryOriginTablesMgmtAzure:
		return common.ModuleTableManagement
	case QueryOriginAnomalies:
		return common.ModuleAnomalies
	case QueryOriginRampPlan:
		return common.ModuleRampPlan
	case QueryOriginInvoicingAws, QueryOriginInvoicingGcp, QueryOriginInvoicingAzure:
		return common.ModuleInvoicing
	case QueryOriginAlerts:
		return common.ModuleAlerts
	case QueryOriginBudgets:
		return common.ModuleBudgets
	case QueryOriginDigest:
		return common.ModuleDigest
	case QueryOriginScheduledReports:
		return common.ModuleScheduledReports
	case QueryOriginReportsAPI:
		return common.ModuleAPI
	case QueryOriginSlackUnfurl:
		return common.ModuleSlackUnfurl
	case QueryOriginClientReservation, QueryOriginClient:
		return common.ModuleUI
	case QueryOriginWidgets:
		return common.ModuleWidgets
	case QueryOriginBigQueryLens:
		return common.ModuleBQLensOptimizer
	case QueryOriginDataHub:
		return common.ModuleDataHub
	case QueryOriginMetadataGcp:
		return common.ModuleMetadataGcp
	case QueryOriginMetadataAws:
		return common.ModuleMetadataAws
	case QueryOriginMetadataAzure:
		return common.ModuleMetadataAzure
	default:
		return common.ModuleOther
	}

}

func MapOriginToHouseFeatureModule(origin QueryOrigin) (common.House, common.Feature, common.Module) {
	house := mapOriginToHouse(origin)
	feature := mapOriginToFeature(origin)
	module := mapOriginToModule(origin)

	return house, feature, module
}
