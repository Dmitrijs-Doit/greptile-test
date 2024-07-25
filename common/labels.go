package common

type LabelKey string

func (l LabelKey) String() string {
	return string(l)
}

const (
	LabelKeyEnv      LabelKey = "env"
	LabelKeyHouse    LabelKey = "house"
	LabelKeyFeature  LabelKey = "feature"
	LabelKeyModule   LabelKey = "module"
	LabelKeyCustomer LabelKey = "customer"
	LabelKeyFunction LabelKey = "function"
	LabelKeyService  LabelKey = "service"
)

type Environment string

const (
	EnvProd Environment = "production"
	EnvDev  Environment = "development"
)

func GetEnvironmentLabel() string {
	if Production {
		return string(EnvProd)
	}

	return string(EnvDev)
}

type House string

func (h House) String() string {
	return string(h)
}

const (
	HouseAdoption House = "adoption"
	HouseData     House = "data"
	HouseGrowth   House = "growth"
	HousePlatform House = "platform"
)

type Feature string

func (f Feature) String() string {
	return string(f)
}

const (
	FeatureCloudAnalytics Feature = "cloud-analytics"
	FeatureRampPlans      Feature = "ramp-plans"
	FeatureAnomalies      Feature = "anomalies"
	FeatureBQLens         Feature = "bq-lens"
	FeatureDataHub        Feature = "datahub"
	FeaturePresentation   Feature = "presentation"
)

type Module string

func (m Module) String() string {
	return string(m)
}

const (
	ModuleTableManagement    Module = "table-management"
	ModuleTableManagementCsp Module = "table-management-csp"
	ModuleWidgets            Module = "widgets"
	ModuleOther              Module = "other"
	ModuleUI                 Module = "ui"
	ModuleAPI                Module = "api"
	ModuleSlackUnfurl        Module = "slack-unfurl"
	ModuleScheduledReports   Module = "scheduled"
	ModuleInvoicing          Module = "invoicing"
	ModuleAlerts             Module = "alert"
	ModuleBudgets            Module = "budget"
	ModuleDigest             Module = "digest"
	ModuleRampPlan           Module = "ramp-plan"
	ModuleAnomalies          Module = "anomalies"
	ModuleBQLensOptimizer    Module = "bq-lens-optimizer"
	ModuleDataHub            Module = "datahub"
	ModuleMetadataGcp        Module = "metadata-gcp"
	ModuleMetadataAws        Module = "metadata-aws"
	ModuleMetadataAzure      Module = "metadata-azure"
	ModuleMetadataDatahub    Module = "metadata-datahub"
	ModuleMetadataBqLens     Module = "metadata-bqlens"
)
