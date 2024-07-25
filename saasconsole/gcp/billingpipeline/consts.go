package billingpipeline

const (
	localAPIService = "http://localhost:8080"
	devAPIService   = "https://cmp-gcp-billing-standalone-pipeline-wsqwprteya-uc.a.run.app"
	prodAPIService  = "https://doit-gcp-saas-billing-pipeline-alqysnpjoq-uc.a.run.app"
)

type AccountDataStatus string

const (
	AccountDataStatusMissing         AccountDataStatus = "missing"
	AccountDataStatusImportRunning   AccountDataStatus = "import-running"
	AccountDataStatusImportFinished  AccountDataStatus = "import-finished"
	AccountDataStatusHistoryFinished AccountDataStatus = "history-finished"
	AccountDataStatusImportPaused    AccountDataStatus = "import-paused"
)
