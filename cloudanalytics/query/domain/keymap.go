package domain

import (
	"strings"

	attributionConsts "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution/consts"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

const (
	NotApplicable = "[N/A]"

	dateField = "T.usage_date_time"
)

var KeyMap = map[string]metadata.MetadataField{
	"year": {
		Order:  0,
		Field:  dateField,
		Label:  "Year",
		Plural: "years",
		Type:   metadata.MetadataFieldTypeDatetime,
	},
	"quarter": {
		Order:  1,
		Field:  dateField,
		Label:  "Quarter",
		Plural: "quarters",
		Type:   metadata.MetadataFieldTypeDatetime,
	},
	"month": {
		Order:  2,
		Field:  dateField,
		Label:  "Month",
		Plural: "months",
		Type:   metadata.MetadataFieldTypeDatetime,
	},
	"week": {
		Order:  3,
		Field:  dateField,
		Label:  "Week",
		Plural: "weeks",
		Type:   metadata.MetadataFieldTypeDatetime,
		BaseValueMappingFunc: func(weekStringWithMonday string) string {
			return strings.Split(weekStringWithMonday, " ")[0]
		},
	},
	"day": {
		Order:  4,
		Field:  dateField,
		Label:  "Day",
		Plural: "days",
		Type:   metadata.MetadataFieldTypeDatetime,
	},
	"hour": {
		Order:  5,
		Field:  dateField,
		Label:  "Hour",
		Plural: "hours",
		Type:   metadata.MetadataFieldTypeDatetime,
	},
	"week_day": {
		Order:  6,
		Field:  dateField,
		Label:  "Weekday",
		Plural: "weekdays",
		Type:   metadata.MetadataFieldTypeDatetime,
		BaseValueMappingFunc: func(weekDay string) string {
			switch weekDay {
			case "Monday":
				return "1"
			case "Tuesday":
				return "2"
			case "Wednesday":
				return "3"
			case "Thursday":
				return "4"
			case "Friday":
				return "5"
			case "Saturday":
				return "6"
			case "Sunday":
				return "7"
			default:
				return "0"
			}
		},
	},
	"cloud_provider": {
		Order:               9,
		Field:               "T.cloud_provider",
		Label:               "Cloud",
		Plural:              "cloud providers",
		Type:                metadata.MetadataFieldTypeFixed,
		DisableRegexpFilter: true,
	},
	"billing_account_id": {
		Order:  10,
		Field:  "T.billing_account_id",
		Label:  "Billing Account",
		Plural: "billing accounts",
		Type:   metadata.MetadataFieldTypeFixed,
	},
	"project_ancestry_names": {
		Order:        11,
		Field:        "T.project.ancestry_names",
		Label:        "Folder",
		Plural:       "folders",
		NullFallback: common.String("[Folder N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
		Cloud:        common.Assets.GoogleCloud,
	},
	"project_id": {
		Order:        12,
		Field:        "T.project_id",
		Label:        "Project/Account ID",
		Plural:       "Project/Account ids",
		NullFallback: common.String("[Project/Account ID N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"project_number": {
		Order:        13,
		Field:        "T.project_number",
		Label:        "Project/Account number",
		Plural:       "Project/Account numbers",
		NullFallback: common.String("[Project/Account number N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"project_name": {
		Order:        14,
		Field:        "T.project_name",
		Label:        "Project/Account name",
		Plural:       "Project/Account names",
		NullFallback: common.String("[Project/Account name N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"service_description": {
		Order:        15,
		Field:        "T.service_description",
		Label:        "Service",
		Plural:       "services",
		NullFallback: common.String("[Service N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"sku_description": {
		Order:        16,
		Field:        "T.sku_description",
		Label:        "SKU",
		Plural:       "SKUs",
		NullFallback: common.String("[SKU N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"service_id": {
		Order:        17,
		Field:        "T.service_id",
		Label:        "Service ID",
		Plural:       "service ids",
		NullFallback: common.String("[Service ID N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"sku_id": {
		Order:        18,
		Field:        "T.sku_id",
		Label:        "SKU ID",
		Plural:       "SKU ids",
		NullFallback: common.String("[SKU ID N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"operation": {
		Order:        19,
		Field:        "T.operation",
		Label:        "Operation",
		Plural:       "Operations",
		NullFallback: common.String("[Operation N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
		Cloud:        common.Assets.AmazonWebServices,
	},
	"resource_id": {
		Order:        20,
		Field:        "T.resource_id",
		Label:        "Resource",
		Plural:       "Resources",
		NullFallback: common.String("[Resource N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"resource_global_id": {
		Order:        21,
		Field:        "T.resource_global_id",
		Label:        "Global Resource",
		Plural:       "Global Resources",
		NullFallback: common.String("[Global Resource N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"country": {
		Order:        22,
		Field:        "T.location.country",
		Label:        "Country",
		Plural:       "countries",
		NullFallback: common.String("[Country N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
		Cloud:        common.Assets.AmazonWebServices,
	},
	"region": {
		Order:        23,
		Field:        "T.location.region",
		Label:        "Region",
		Plural:       "regions",
		NullFallback: common.String("[Region N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"zone": {
		Order:        24,
		Field:        "T.location.zone",
		Label:        "Zone",
		Plural:       "zones",
		NullFallback: common.String("[Zone N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"cost_type": {
		Order:        25,
		Field:        "T.cost_type",
		Label:        "Cost Type",
		Plural:       "cost types",
		NullFallback: common.String("[Cost Type N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"pricing_unit": {
		Order:        26,
		Field:        "T.usage.pricing_unit",
		Label:        "Unit",
		Plural:       "pricing units",
		NullFallback: common.String("[Unit N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"credit": {
		Order:        27,
		Field:        "report_value.credit",
		Label:        "Credit",
		Plural:       "credits",
		NullFallback: common.String("[Credit N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"savings_description": {
		Order:        28,
		Field:        "report_value.savings_description",
		Label:        "Savings Type",
		Plural:       "Savings types",
		NullFallback: common.String("[Savings Type N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"customer_type": {
		Order:        29,
		Field:        "T.customer_type",
		Label:        "Customer Type",
		Plural:       "customer types",
		NullFallback: common.String("[Customer Type N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"is_marketplace": {
		Order:               30,
		Field:               "T.is_marketplace",
		Label:               "Marketplace",
		Plural:              "Marketplace",
		CastToDBType:        common.String("BOOL"),
		Type:                metadata.MetadataFieldTypeFixed,
		DisableRegexpFilter: true,
	},
	"invoice_month": {
		Order:        31,
		Field:        "T.invoice.month",
		Label:        "Invoice month (experimental)",
		Plural:       "invoice months (experimental)",
		NullFallback: common.String("[Invoice month N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},

	// GKE Cost allocation fields
	"kubernetes_cluster_name": {
		Order:        50,
		Field:        "T.kubernetes_cluster_name",
		Label:        "GKE Cluster",
		Plural:       "GKE Clusters",
		NullFallback: common.String("[GKE Cluster N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"kubernetes_namespace": {
		Order:        51,
		Field:        "T.kubernetes_namespace",
		Label:        "GKE Namespace",
		Plural:       "GKE Namespaces",
		NullFallback: common.String("[GKE Namespace N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},

	// GKE Usage metering fields
	"cluster_name": {
		Order:  100,
		Field:  "GKE.cluster_name",
		Label:  "GKE (Metering) Cluster",
		Plural: "GKE Clusters",
		Type:   metadata.MetadataFieldTypeGKE,
	},
	"cluster_location": {
		Order:  101,
		Field:  "GKE.cluster_location",
		Label:  "GKE (Metering) Region",
		Plural: "GKE Regions",
		Type:   metadata.MetadataFieldTypeGKE,
	},
	"resource_name": {
		Order:  102,
		Field:  "GKE.resource_name",
		Label:  "GKE (Metering) Resource",
		Plural: "GKE Resources",
		Type:   metadata.MetadataFieldTypeGKE,
	},
	"namespace": {
		Order:  103,
		Field:  "GKE.namespace",
		Label:  "GKE (Metering) Namespace",
		Plural: "GKE Namespaces",
		Type:   metadata.MetadataFieldTypeGKE,
	},

	// CSP Fields
	"csp_classification": {
		Order:  200,
		Field:  "T.classification",
		Label:  "Classification",
		Plural: "Classifications",
		Type:   metadata.MetadataFieldTypeFixed,
	},
	"csp_primary_domain": {
		Order:        201,
		Field:        "T.primary_domain",
		Label:        "Customer",
		Plural:       "Customers",
		NullFallback: common.String("[Customer N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"csp_territory": {
		Order:  202,
		Field:  "T.territory",
		Label:  "Territory",
		Plural: "Territories",
		Type:   metadata.MetadataFieldTypeFixed,
	},
	"csp_payee_country": {
		Order:  203,
		Field:  "T.payee_country",
		Label:  "Payee Country",
		Plural: "Payee Countries",
		Type:   metadata.MetadataFieldTypeFixed,
	},
	"csp_payer_country": {
		Order:  204,
		Field:  "T.payer_country",
		Label:  "Payer Country",
		Plural: "Payer Countries",
		Type:   metadata.MetadataFieldTypeFixed,
	},
	"csp_field_sales_representative": {
		Order:        205,
		Field:        "T.field_sales_representative",
		Label:        "FSR",
		Plural:       "Field Sales Representatives",
		NullFallback: common.String(NotApplicable),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"csp_strategic_account_manager": {
		Order:        206,
		Field:        "T.strategic_account_manager",
		Label:        "AM/SAM",
		Plural:       "Strategic Account Managers",
		NullFallback: common.String(NotApplicable),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"csp_technical_account_manager": {
		Order:        207,
		Field:        "T.technical_account_manager",
		Label:        "TAM",
		Plural:       "Technical Account Managers",
		NullFallback: common.String(NotApplicable),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"csp_customer_success_manager": {
		Order:        208,
		Field:        "T.customer_success_manager",
		Label:        "CSM",
		Plural:       "Customer Success Managers",
		NullFallback: common.String(NotApplicable),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"csp_committed": {
		Order:        209,
		Field:        "T.is_commitment",
		Label:        "Committed",
		Plural:       "Committed",
		NullFallback: common.String("[Contract N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},

	// Customer Features
	"feature": {
		Order:        300,
		Field:        "T.feature",
		Label:        "Feature",
		Plural:       "Features",
		NullFallback: common.String("[Feature N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},

	// BQ Lens (BQ Audit Logs fields)
	"job_status": {
		Order:        400,
		Field:        "T.job_status",
		Label:        "Job Status",
		Plural:       "Job Statuses",
		NullFallback: common.String("[Job Status N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"statement_type": {
		Order:        401,
		Field:        "T.statement_type",
		Label:        "Statement Type",
		Plural:       "Statement Types",
		NullFallback: common.String("[Statement Type N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"query_priority": {
		Order:        402,
		Field:        "T.query_priority",
		Label:        "Query Priority",
		Plural:       "Query Priorities",
		NullFallback: common.String("[Query Priority N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"caller_ip": {
		Order:        403,
		Field:        "T.caller_ip",
		Label:        "Caller IP",
		Plural:       "Caller IPs",
		NullFallback: common.String("[Caller IP N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"user": {
		Order:        404,
		Field:        "T.user",
		Label:        "User",
		Plural:       "Users",
		NullFallback: common.String("[User N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"event_name": {
		Order:        405,
		Field:        "T.event_name",
		Label:        "Event Name",
		Plural:       "Event Names",
		NullFallback: common.String("[Event Name N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},
	"reservation": {
		Order:        406,
		Field:        "T.reservation",
		Label:        "Reservation",
		Plural:       "Reservations",
		NullFallback: common.String("[Reservation N/A]"),
		Type:         metadata.MetadataFieldTypeFixed,
	},

	// Attribution and labels
	"attribution": {
		Order:               900,
		Field:               "",
		Label:               "Attribution",
		Plural:              "attributions",
		NullFallback:        common.String(attributionConsts.AttributionNA),
		Type:                metadata.MetadataFieldTypeAttribution,
		DisableRegexpFilter: true,
	},
	"attribution_group": {
		Order:               901,
		Field:               "",
		Label:               "Attribution Group",
		Plural:              "attribution groups",
		NullFallback:        common.String("Unallocated"),
		Type:                metadata.MetadataFieldTypeAttributionGroup,
		DisableRegexpFilter: true,
	},

	"labels": {
		Order:        1000,
		Field:        "T.labels",
		Plural:       "label values",
		Type:         metadata.MetadataFieldTypeLabel,
		NullFallback: common.String("[Label N/A]"),
	},
	"gke_labels": {
		Order:        1001,
		Field:        "GKE.labels",
		Plural:       "GKE label values",
		Type:         metadata.MetadataFieldTypeGKELabel,
		NullFallback: common.String("[Label N/A]"),
	},
	"project_labels": {
		Order:        1002,
		Field:        "T.project.labels",
		Plural:       "project label values",
		Type:         metadata.MetadataFieldTypeProjectLabel,
		NullFallback: common.String("[Label N/A]"),
	},
	"system_labels": {
		Order:        1003,
		Field:        "T.system_labels",
		Plural:       "system label values",
		Type:         metadata.MetadataFieldTypeSystemLabel,
		NullFallback: common.String("[Label N/A]"),
	},
	"tags": {
		Order:        1004,
		Field:        "T.tags",
		Plural:       "Tag values",
		Type:         metadata.MetadataFieldTypeTag,
		NullFallback: common.String("[Tag N/A]"),
	},
	"labels_keys": {
		Order:   2000,
		Field:   "T.labels",
		Label:   "Labels",
		Plural:  "labels",
		Type:    metadata.MetadataFieldTypeOptional,
		SubType: metadata.MetadataFieldTypeLabel,
	},
	"gke_labels_keys": {
		Order:        2001,
		Field:        "GKE.labels",
		Label:        "GKE Labels",
		Plural:       "GKE labels",
		Type:         metadata.MetadataFieldTypeOptional,
		SubType:      metadata.MetadataFieldTypeGKELabel,
		NullFallback: common.String("[Label N/A]"),
	},
	"project_labels_keys": {
		Order:   2002,
		Field:   "T.project.labels",
		Label:   "Project Labels",
		Plural:  "project labels",
		Type:    metadata.MetadataFieldTypeOptional,
		SubType: metadata.MetadataFieldTypeProjectLabel,
	},
	"system_labels_keys": {
		Order:   2003,
		Field:   "T.system_labels",
		Label:   "System Labels",
		Plural:  "system labels",
		Type:    metadata.MetadataFieldTypeOptional,
		SubType: metadata.MetadataFieldTypeSystemLabel,
	},
	"tags_keys": {
		Order:   2004,
		Field:   "T.tags",
		Label:   "Tags",
		Plural:  "tags",
		Type:    metadata.MetadataFieldTypeOptional,
		SubType: metadata.MetadataFieldTypeTag,
	},
}
