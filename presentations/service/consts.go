package service

import (
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/presentations/domain"
)

var (
	projectPrefixes = []string{
		"development",
		"production",
		"staging",
		"qa",
		"playground",
		"sandbox",
		"demo",
	}

	projectSuffixes = []string{
		"CloudCosmos",
		"NebulaCraft",
		"AstroCloud",
		"ByteGalaxy",
		"QuantumNimbus",
		"CyberOrbit",
		"SparkCelestial",
		"CloudSwiftX",
		"NebulaInsight",
		"NexusStellar",
	}

	tags = []string{
		"Team Red",
		"Team Blue",
		"Team Green",
	}

	customerLabels = []string{
		"VandelayIndustries",
		"CostanzaArchitects",
		"TopMuffinToYou",
		"KramersChickens",
		"TheMaestroCorp",
		"PuddyIndustries",
		"AVImports",
		"FestivusCorp",
		"JPetermanEnterprises",
		"SerenityNowInnovations",
	}

	projectLabels = []string{
		"GalacticExplorer",
		"NebulaPioneer",
		"StarlightQuest",
		"CelestialNavigator",
		"CosmicDiscovery",
		"InterstellarMission",
		"AstroFrontier",
		"InfinityVoyage",
		"GalacticOdyssey",
		"OrionExpedition",
	}

	azureProductNames = []string{
		"AzureCloudBoost",
		"AzureDataForge",
		"AzureNetConnect",
		"AzureAppSync",
		"AzureInsightPlus",
		"AzureSecureShield",
		"AzureAIAssist",
		"AzureEdgeGuard",
		"AzureScaleMaster",
		"AzureCodeForge",
		"AzureStreamlinePro",
		"AzureHubSync",
		"AzureSmartLink",
		"AzureComputeWave",
		"AzureDataPrime",
		"AzureLogicPulse",
		"AzureBotMaster",
		"AzureSecureVault",
		"AzureAnalyticsPro",
		"AzureConnectX",
	}

	commonNames = []string{
		"NebulaNode",
		"QuantumQuark",
		"CyberByte",
		"TechnoTorus",
		"DataDynamo",
		"VirtualVertex",
		"CodeCruncher",
		"ByteBlast",
		"NanoNexus",
		"DigitalDroid",
		"SynthSphere",
		"LogicLoom",
		"PixelPulse",
		"CloudCrafter",
		"AlphaArray",
		"NexusNode",
		"TerraTech",
		"MatrixMachine",
		"BinaryBeam",
		"InfraSphere",
	}

	owners = []string{
		"Samantha",
		"Jacob",
		"Emily",
		"Liam",
		"Olivia",
		"Ethan",
		"Isabella",
		"Noah",
		"Ava",
		"Alexander",
	}

	playground = []string{
		"TRUE",
		"FALSE",
	}

	customers = []string{
		"Vandelay",
		"Hennigan\\'s",
		"Kruger",
		"Festivus",
		"Puddy",
		"Soupman",
		"J. Peterman",
		"Costanza",
		"Kramerica",
		"Yada",
	}

	cohorts = []string{
		"Early Adopters",
		"Pioneer Pilots",
		"Genesis Group",
		"Founders\\' Circle",
		"Beta Brigade",
		"Launch Legends",
		"Trailblazers",
		"Prime Players",
		"Inception Initiative",
		"Alpha Ambassadors",
	}
)

const (
	// INFO: for MVP, we will use only one demo billing account id
	awsDemoBillingAccountID = domain.AWSDemoBillingAccountID
	sessionIDQueryParamName = domain.SessionIDQueryParamName

	moonActiveCustomerID = "2Gi0e4pPA3wsfJNOOohW"
	connatixCustomerID   = "LcgELbXV21Imef3utMoh"
	taboolaCustomerID    = "ImoC9XkrutBysJvyqlBm"
	aigoAiCustomerID     = "iYXtlBDaHSb3hNANkfrk"
	quinxCustomerID      = "lzQ6rYQdl3dDOA3aTcae"
	takeoffCustomerID    = "fgTSYGR9k7wQjipsidRE"

	awsQueryFields string = `
		billing_account_id,
		project_id,
		service_description,
		service_id,
		sku_description,
		sku_id,
		usage_date_time,
		usage_start_time,
		usage_end_time,
		project,
		labels,
		system_labels,
		location,
		export_time,
		cost,
		currency,
		currency_conversion_rate,
		usage,
		invoice,
		cost_type,
		report,
		resource_id,
		operation,
		customer_type,
		is_marketplace
		`

	azureQueryFields string = `
		billing_account_id,
		project_id,
		service_description,
		service_id,
		sku_description,
		sku_id,
		usage_date_time,
		usage_start_time,
		usage_end_time,
		project,
		labels,
		system_labels,
		location,
		export_time,
		cost,
		currency,
		currency_conversion_rate,
		usage,
		invoice,
		cost_type,
		report,
		resource_id,
		customer_type,
		is_marketplace,
		operation,
		tenant_id`

	gcpQueryFields string = `
		billing_account_id,
		project_id,
		service_description,
		service_id,
		sku_description,
		sku_id,
		usage_date_time,
		usage_start_time,
		usage_end_time,
		project,
		labels,
		system_labels,
		location,
		export_time,
		cost,
		currency,
		currency_conversion_rate,
		usage,
		credits,
		invoice,
		cost_type,
		adjustment_info,
		tags,
		price,
		cost_at_list,
    	transaction_type,
     	seller_name,
     	subscription,
		customer_type,
		gcp_metrics,
    	plps_doit_percent,
		resource_id,
		resource_global_id,
		is_marketplace,
		is_preemptible,
		is_premium_image,
		exclude_discount,
		kubernetes_cluster_name,
		kubernetes_namespace,
		price_book,
		discount,
		report,
		`
)

var (
	presentationCustomers = []string{
		domain.PresentationcustomerAWSAzureGCP,
		domain.PresentationcustomerAWSGCP,
		domain.PresentationcustomerAzureGCP,
		domain.PresentationcustomerAWSAzure,
		domain.PresentationcustomerGCP,
		domain.PresentationcustomerAWS,
		domain.PresentationcustomerAzure,
	}
)

func IsPresentationCustomer(customerID string) bool {
	for _, item := range presentationCustomers {
		if item == customerID {
			return true
		}
	}
	return false
}

const (
	moonactiveBillingID string = "01FE64-BB73DE-4BA1D8"
	takeoffBillingID    string = "018D35-5F6963-05E83A"
)

var (
	gcpDemoBillingIds = map[string]string{
		"moonactive": moonactiveBillingID,
		"takeoff":    takeoffBillingID,
	}
)

var additionalKeys = []string{
	"{resource_anonymizer}",
	"{ancestry_names_anonymizer}",
	"{kubernetes_cluster_name_anonymizer}",
	"{kubernetes_namespace_anonymizer}",
	"{null_tags_field}",
	"{project_id_generator}",
}

// Define additionalValues
var additionalValues = []string{
	resourceAnonymizer,
	AncestryNamesAnonymizer,
	KubernetesClusterNameAnonymizer,
	KubernetesNamespaceAnonymizer,
	domainQuery.NullTags,
	projectIdGenerator,
}
