package utils

const (
	serviceAccountPrefix                     = "sa-fs-"
	serviceAccountDescription                = "Flexsave Standalone Service Account"
	saTokenCreateRole                        = "roles/iam.serviceAccountTokenCreator"
	billingAccountAdminRole           string = "roles/billing.admin"
	projectBillingManagerRole         string = "roles/billing.projectManager"
	MaxServiceAccountsInProject       int    = 98
	ServiceAccountsInProjectThreshold int    = 20
	FreeServiceAccountsThreshold      int    = 5
	projectPrefix                     string = "doitintl-sa-fs"
	fallbackBilllingAccountID         string = "016E1A-7AB8D6-8BE61D"
	flexsaveProjectsFolderID          string = "1080772429414"
	DevFolderID                       string = "324415374716"
	ProdFolderID                      string = "1087891064495"
	AppCollection                            = "app"
	GCPFlexsaveStandaloneDoc                 = "gcp-flexsave-standalone"
	OnboardingCollection                     = "onboarding"
	ServiceAccountsDoc                       = "service-accounts"
	ProjectsDoc                              = "projects"
	DevServiceAccountsDoc                    = "dev-service-accounts"
	DevProjectsDoc                           = "dev-projects"
	EnvStatusDoc                             = "env-config"
	CreateNewServiceAccountsURL              = "/tasks/flexsave-standalone/google-cloud/onboarding/new-service-accounts"
	cloudResourceManagerAPI                  = "cloudresourcemanager.googleapis.com"
	serviceUsageAPI                          = "serviceusage.googleapis.com"
	recommenderAPI                           = "recommender.googleapis.com"
	bigqueryAPI                              = "bigquery.googleapis.com"
	cloudbillingAPI                          = "cloudbilling.googleapis.com"

	// TODO: @Stav Create API that returns the sa email
	flexsaveCloudRunServiceAccountEmail        = "flexsave-gcp@doitintl-svc-accounts.iam.gserviceaccount.com"
	billingPipelineCloudRunServiceAccountEmail = "fsgcp-sa-pipeline@me-doit-intl-com.iam.gserviceaccount.com"
	dev_flexsaveCloudRunServiceAccountEmail    = "772991852481-compute@developer.gserviceaccount.com"

	dev_GAEDefaultServiceAccountsGroup = "cmp-sa-dev@doit-intl.com"
)
