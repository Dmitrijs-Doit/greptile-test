package utils

const (
	serviceAccountPrefix                     = "doit-saas"
	serviceAccountDescription                = "SaaS Console Service Account"
	saTokenCreateRole                        = "roles/iam.serviceAccountTokenCreator"
	MaxServiceAccountsInProject       int    = 98
	ServiceAccountsInProjectThreshold int    = 20
	FreeServiceAccountsThreshold      int    = 5
	projectPrefix                     string = "doit-saas"
	fallbackBilllingAccountID         string = "016E1A-7AB8D6-8BE61D"
	DevFolderID                       string = "324415374716"
	ProdFolderID                      string = "1087891064495"
	IntegrationsCollection            string = "integrations"
	BillingStandaloneCollection       string = "billing-standalone"
	ServiceAccountsCollection         string = "gcp-service-accounts"
	ServiceAccountsDoc                       = "service-accounts"
	FreeServiceAccountsCollection            = "freeGCPServiceAccounts"
	ServiceAccountsPoolDocument              = "pool"
	ReservedServiceAccountsCollection        = "reservedGCPServiceAccounts"
	AcquiredServiceAccountsCollection        = "acquiredGCPServiceAccounts"
	DevServiceAccountsDoc                    = "dev-service-accounts"
	DevProjectsDoc                           = "dev-projects"
	ProjectsDoc                              = "projects-new"
	EnvStatusDoc                             = "env-config"
	CreateNewServiceAccountsURL              = "/tasks/billing-standalone/google-cloud/onboarding/new-service-accounts"
	cloudResourceManagerAPI                  = "cloudresourcemanager.googleapis.com"
	serviceUsageAPI                          = "serviceusage.googleapis.com"
	bigqueryAPI                              = "bigquery.googleapis.com"
	cloudbillingAPI                          = "cloudbilling.googleapis.com"

	billingPipelineCloudRunServiceAccountEmail    = "gcp-saas-billing@me-doit-intl-com.iam.gserviceaccount.com"
	devBillingPipelineCloudRunServiceAccountEmail = "gcp-saas-billing@doitintl-cmp-dev.iam.gserviceaccount.com"

	devGAEDefaultServiceAccountsGroup = "cmp-sa-dev@doit-intl.com"

	DedicatedCustomersServiceAccountsGroup = "gcp-saas-billing@doit.com"
)
