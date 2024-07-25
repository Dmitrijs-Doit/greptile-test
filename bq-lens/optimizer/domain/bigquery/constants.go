package bqmodels

type QueryName string

const (
	StorageSavings QueryName = "storageSavings"
	TotalScanPrice QueryName = "totalScan"

	// hybrid
	CostFromTableTypes  QueryName = "costFromTableTypes"
	TableStorageTB      QueryName = "tableStorageTB"
	TableStoragePrice   QueryName = "tableStoragePrice"
	DatasetStorageTB    QueryName = "datasetStorageTB"
	DatasetStoragePrice QueryName = "datasetStoragePrice"
	ProjectStorageTB    QueryName = "projectStorageTB"
	ProjectStoragePrice QueryName = "projectStoragePrice"

	// flat-rate
	ScheduledQueriesMovement      QueryName = "scheduledQueriesMovement"
	SlotsExplorerFlatRate         QueryName = "slotsExplorer-flatRate"
	UserSlots                     QueryName = "userSlots"
	UserSlotsTopQueries           QueryName = "userSlotsTopQueries"
	BillingProjectSlotsTopQueries QueryName = "billingProjectSlotsTopQueries"
	BillingProjectSlotsTopUsers   QueryName = "billingProjectSlotsTopUsers"
	BillingProjectSlots           QueryName = "billingProjectSlots"

	// standard edition
	StandardScheduledQueriesMovement QueryName = "standardScheduledQueriesMovement"
	StandardSlotsExplorer            QueryName = "standardSlotsExplorer"
	StandardUserSlots                QueryName = "standardUserSlots"
	StandardBillingProjectSlots      QueryName = "standardBillingProjectSlots"

	// enterprise edition
	EnterpriseScheduledQueriesMovement QueryName = "enterpriseScheduledQueriesMovement"
	EnterpriseSlotsExplorer            QueryName = "enterpriseSlotsExplorer"
	EnterpriseUserSlots                QueryName = "enterpriseUserSlots"
	EnterpriseBillingProjectSlots      QueryName = "enterpriseBillingProjectSlots"

	// enterprise plus edition
	EnterprisePlusScheduledQueriesMovement QueryName = "enterprisePlusScheduledQueriesMovement"
	EnterprisePlusSlotsExplorer            QueryName = "enterprisePlusSlotsExplorer"
	EnterprisePlusUserSlots                QueryName = "enterprisePlusUserSlots"
	EnterprisePlusBillingProjectSlots      QueryName = "enterprisePlusBillingProjectSlots"

	// on-demand
	LimitingJobsSavings   QueryName = "limitingJobsSavings"
	UsePartitionField     QueryName = "usePartitionField"
	PhysicalStorage       QueryName = "physicalStorage"
	PartitionTables       QueryName = "partitionTables"
	ClusterTables         QueryName = "clusterTables"
	SlotsExplorerOnDemand QueryName = "slotsExplorer-onDemand"

	BillingProject           QueryName = "billingProject"
	BillingProjectTopUsers   QueryName = "billingProjectTopUsers"
	BillingProjectTopQueries QueryName = "billingProjectTopQueries"

	BillingProjectScanPrice           QueryName = "billingProject-scanPrice"
	BillingProjectTopUsersScanPrice   QueryName = "billingProjectTopUsers-scanPrice"
	BillingProjectTopQueriesScanPrice QueryName = "billingProjectTopQueries-scanPrice"
	BillingProjectScanTB              QueryName = "billingProject-scanTB"
	BillingProjectTopUsersScanTB      QueryName = "billingProjectTopUsers-scanTB"
	BillingProjectTopQueriesScanTB    QueryName = "billingProjectTopQueries-scanTB"

	User            QueryName = "user"
	UserTopProjects QueryName = "userTopProjects"
	UserTopDatasets QueryName = "userTopDatasets"
	UserTopTables   QueryName = "userTopTables"
	UserTopQueries  QueryName = "userTopQueries"

	UserScanPrice            QueryName = "user-scanPrice"
	UserTopProjectsScanPrice QueryName = "userTopProjects-scanPrice"
	UserTopDatasetsScanPrice QueryName = "userTopDatasets-scanPrice"
	UserTopTablesScanPrice   QueryName = "userTopTables-scanPrice"
	UserTopQueriesScanPrice  QueryName = "userTopQueries-scanPrice"
	UserScanTB               QueryName = "user-scanTB"
	UserTopProjectsScanTB    QueryName = "userTopProjects-scanTB"
	UserTopDatasetsScanTB    QueryName = "userTopDatasets-scanTB"
	UserTopTablesScanTB      QueryName = "userTopTables-scanTB"
	UserTopQueriesScanTB     QueryName = "userTopQueries-scanTB"

	Project            QueryName = "project"
	ProjectTopUsers    QueryName = "projectTopUsers"
	ProjectTopQueries  QueryName = "projectTopQueries"
	ProjectTopDatasets QueryName = "projectTopDatasets"
	ProjectTopTables   QueryName = "projectTopTables"

	ProjectScanPrice            QueryName = "project-scanPrice"
	ProjectTopUsersScanPrice    QueryName = "projectTopUsers-scanPrice"
	ProjectTopQueriesScanPrice  QueryName = "projectTopQueries-scanPrice"
	ProjectTopDatasetsScanPrice QueryName = "projectTopDatasets-scanPrice"
	ProjectTopTablesScanPrice   QueryName = "projectTopTables-scanPrice"
	ProjectScanTB               QueryName = "project-scanTB"
	ProjectTopUsersScanTB       QueryName = "projectTopUsers-scanTB"
	ProjectTopQueriesScanTB     QueryName = "projectTopQueries-scanTB"
	ProjectTopDatasetsScanTB    QueryName = "projectTopDatasets-scanTB"
	ProjectTopTablesScanTB      QueryName = "projectTopTables-scanTB"

	Dataset           QueryName = "dataset"
	DatasetTopTables  QueryName = "datasetTopTables"
	DatasetTopUsers   QueryName = "datasetTopUsers"
	DatasetTopQueries QueryName = "datasetTopQueries"

	DatasetScanPrice           QueryName = "dataset-scanPrice"
	DatasetTopTablesScanPrice  QueryName = "datasetTopTables-scanPrice"
	DatasetTopUsersScanPrice   QueryName = "datasetTopUsers-scanPrice"
	DatasetTopQueriesScanPrice QueryName = "datasetTopQueries-scanPrice"
	DatasetScanTB              QueryName = "dataset-scanTB"
	DatasetTopTablesScanTB     QueryName = "datasetTopTables-scanTB"
	DatasetTopUsersScanTB      QueryName = "datasetTopUsers-scanTB"
	DatasetTopQueriesScanTB    QueryName = "datasetTopQueries-scanTB"

	Table           QueryName = "table"
	TableTopUsers   QueryName = "tableTopUsers"
	TableTopQueries QueryName = "tableTopQueries"

	TableScanPrice           QueryName = "table-scanPrice"
	TableTopUsersScanPrice   QueryName = "tableTopUsers-scanPrice"
	TableTopQueriesScanPrice QueryName = "tableTopQueries-scanPrice"
	TableScanTB              QueryName = "table-scanTB"
	TableTopUsersScanTB      QueryName = "tableTopUsers-scanTB"
	TableTopQueriesScanTB    QueryName = "tableTopQueries-scanTB"
)

type TimeRange string

const (
	TimeRangeDay   TimeRange = "past-1-day"
	TimeRangeWeek  TimeRange = "past-7-days"
	TimeRangeMonth TimeRange = "past-30-days"
)

func (t TimeRange) String() string {
	return string(t)
}

var DataPeriods = []TimeRange{TimeRangeMonth, TimeRangeWeek, TimeRangeDay}
