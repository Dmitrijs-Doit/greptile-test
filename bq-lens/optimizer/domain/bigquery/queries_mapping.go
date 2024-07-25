package bqmodels

type Mode string

const (
	OnDemand              Mode = "on-demand"
	FlatRate              Mode = "flat-rate"
	StandardEdition       Mode = "standard-edition"
	EnterpriseEdition     Mode = "enterprise-edition"
	EnterprisePlusEdition Mode = "enterprise-plus-edition"
	Hybrid                Mode = "hybrid"
)

var QueriesPerMode = map[Mode]map[QueryName]string{
	OnDemand:              OnDemandQueries,
	Hybrid:                HybridQueries,
	FlatRate:              FlatRateQueries,
	StandardEdition:       StandardQueries,
	EnterpriseEdition:     EnterpriseQueries,
	EnterprisePlusEdition: EnterprisePlusQueries,
}

var HybridQueries = map[QueryName]string{
	CostFromTableTypes:  costFromTableTypes,
	TableStorageTB:      tableStorageTB,
	TableStoragePrice:   tableStoragePrice,
	DatasetStorageTB:    datasetStorageTB,
	DatasetStoragePrice: datasetStoragePrice,
	ProjectStorageTB:    projectStorageTB,
	ProjectStoragePrice: projectStoragePrice,
}

var FlatRateQueries = map[QueryName]string{
	ScheduledQueriesMovement: scheduledQueriesMovement,
	SlotsExplorerFlatRate:    slotsExplorer,
	UserSlots:                userSlots,
	BillingProjectSlots:      billingProjectSlots,
}

var StandardQueries = map[QueryName]string{
	StandardScheduledQueriesMovement: scheduledQueriesMovement,
	StandardSlotsExplorer:            slotsExplorer,
	StandardUserSlots:                userSlots,
	StandardBillingProjectSlots:      billingProjectSlots,
}

var EnterpriseQueries = map[QueryName]string{
	EnterpriseScheduledQueriesMovement: scheduledQueriesMovement,
	EnterpriseSlotsExplorer:            slotsExplorer,
	EnterpriseUserSlots:                userSlots,
	EnterpriseBillingProjectSlots:      billingProjectSlots,
}

var EnterprisePlusQueries = map[QueryName]string{
	EnterprisePlusScheduledQueriesMovement: scheduledQueriesMovement,
	EnterprisePlusSlotsExplorer:            slotsExplorer,
	EnterprisePlusUserSlots:                userSlots,
	EnterprisePlusBillingProjectSlots:      billingProjectSlots,
}

var OnDemandQueries = map[QueryName]string{
	LimitingJobsSavings:   limitingJobsSavings,
	UsePartitionField:     usePartitionField,
	PartitionTables:       partitionTables,
	ClusterTables:         clusterTables,
	SlotsExplorerOnDemand: slotsExplorer,
	BillingProject:        billingProject,
	User:                  user,
	Project:               project,
	Dataset:               dataset,
	Table:                 table,
	TableTopQueries:       tableTopQueries,
	TableTopUsers:         tableTopUsers,
	PhysicalStorage:       physicalStorage,
}

var UserSlotsQueries = map[QueryName]string{
	UserSlots:           userSlots,
	UserSlotsTopQueries: userSlotsTopQueries,
}

var BillingProjectSlotsQueries = map[QueryName]string{
	BillingProjectSlotsTopQueries: billingProjectSlotsTopQueries,
	BillingProjectSlotsTopUsers:   billingProjectSlotsTopUsers,
}

var OnDemandBillingProjectQueries = map[QueryName]string{
	BillingProjectTopQueries: billingProjectTopQueries,
	BillingProjectTopUsers:   billingProjectTopUsers,
}

var OnDemandUserQueries = map[QueryName]string{
	User:            user,
	UserTopQueries:  userTopQueries,
	UserTopTables:   userTopTables,
	UserTopDatasets: userTopDatasets,
	UserTopProjects: userTopProjects,
}

var TableDiscoveryIndependent = []QueryName{
	ScheduledQueriesMovement,
	SlotsExplorerFlatRate,
	UserSlotsTopQueries,
	UserSlots,
	BillingProjectSlotsTopQueries,
	BillingProjectSlotsTopUsers,
	BillingProjectSlots,
	SlotsExplorerOnDemand,
	BillingProject,
	BillingProjectTopQueries,
	BillingProjectTopUsers,
}
