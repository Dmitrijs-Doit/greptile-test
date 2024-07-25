package consts

import "time"

var DummySAPoolProjects = []string{DummyBQProjectName}

const (
	AutomationManagerCollection  string        = "automation-manager"
	AutomationManagerDoc         string        = "automationManager"
	AutomationOrchestratorDoc    string        = "automationOrchestrator"
	AutomationTasksCollection    string        = "automation-tasks"
	ServiceAccountPoolCollection string        = "service-account-pool"
	OrchestrationMaxDuration     time.Duration = time.Duration(time.Hour * 5)
	TaskMaxDuration              time.Duration = time.Duration(time.Minute * 10)

	DummyBillingAccountTemplate  string = "DUMMY-%05d-%05d"
	CopyToDummyJobPrefixTemplate string = "AU-ToDum-%s-%d-%d"

	AutomationManagerMaxDuration  time.Duration = time.Duration(time.Minute * 25)
	AutomationTaskMaxDuration     time.Duration = time.Duration(time.Minute * 25)
	AutomationJobMaxDuration      time.Duration = time.Duration(time.Minute * 20)
	DummyTaskMaxExtentionDuration time.Duration = time.Duration(time.Minute * 4)

	MaxAllowedSAinProject    int    = 100
	DummyBQProjectName       string = "danielle-project-2-345913"
	DummySAPrefix            string = "dummy-fs-sa"
	DummySATemplate          string = DummySAPrefix + "-%s"
	DummyCustomerIDTemplate  string = "customerID-%s"
	DummySAPoolProject_1     string = "dummy-fs-sa-project-5596x0kajv"
	DummyBQDatasetName       string = "gcp_billing_dummy"
	DummyBQTableNameTemplate string = "%s_gcp_billing_export_v1_DUMMY_%05d_%05d"
	DummyBQTableNameOriginal string = "gcp_billing_export_v1_DUMMY"
	ServiceAccount           string = "automation@danielle-project-2-345913.iam.gserviceaccount.com"
	LionelSA                 string = "lionel@doit-intl.com"
	//"lionel@doit-intl.com"
)
