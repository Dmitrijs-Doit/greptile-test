package firestoremodels

import "time"

type CommonRecommendation struct {
	Recommendation    string  `firestore:"recommendation"`
	SavingsPercentage float64 `firestore:"savingsPercentage"`
	SavingsPrice      float64 `firestore:"savingsPrice"`
}

type CommonTopQuery struct {
	AvgExecutionTimeSec   float64 `firestore:"avgExecutionTimeSec"`
	AvgSlots              float64 `firestore:"avgSlots"`
	ExecutedQueries       int64   `firestore:"executedQueries"`
	TotalExecutionTimeSec float64 `firestore:"totalExecutionTimeSec"`
	BillingProjectID      string  `firestore:"billingProjectId"`
}

type LimitingJobsSavings struct {
	SumSavingsReducingBy10     float64                   `firestore:"sumSavingsReducingBy10"`
	SumSavingsReducingBy20     float64                   `firestore:"sumSavingsReducingBy20"`
	SumSavingsReducingBy30     float64                   `firestore:"sumSavingsReducingBy30"`
	SumSavingsReducingBy40     float64                   `firestore:"sumSavingsReducingBy40"`
	SumSavingsReducingBy50     float64                   `firestore:"sumSavingsReducingBy50"`
	DetailedTable              []LimitingJobsDetailTable `firestore:"detailedTable"`
	DetailedTableFieldsMapping map[string]FieldDetail    `firestore:"detailedTableFieldsMapping"`
	CommonRecommendation
}

type LimitingJobsDetailTable struct {
	TableFullID       string  `firestore:"tableFullId"`
	JobID             string  `firestore:"jobId"`
	Location          string  `firestore:"location"`
	BillingProjectID  string  `firestore:"billingProjectId"`
	UserID            string  `firestore:"userId"`
	FirstExecution    string  `firestore:"firstExecution"`
	LastExecution     string  `firestore:"lastExecution"`
	AllJobs           int64   `firestore:"allJobs"`
	ScanPricePerQuery float64 `firestore:"scanPricePerQuery"`
	TotalScanPrice    float64 `firestore:"totalScanPrice"`
	ReducingBy10      float64 `firestore:"reducingBy10"`
	ReducingBy20      float64 `firestore:"reducingBy20"`
	ReducingBy30      float64 `firestore:"reducingBy30"`
	ReducingBy40      float64 `firestore:"reducingBy40"`
	ReducingBy50      float64 `firestore:"reducingBy50"`
}

type UsePartitionField struct {
	DetailedTable              []UsePartitionDetailTable `firestore:"detailedTable"`
	DetailedTableFieldsMapping map[string]FieldDetail    `firestore:"detailedTableFieldsMapping"`
	CommonRecommendation
}

type UsePartitionDetailTable struct {
	JobID            string  `firestore:"jobId"`
	Location         string  `firestore:"location"`
	BillingProjectID string  `firestore:"billingProjectId"`
	ScanTB           float64 `firestore:"scanTB"`
	ScanPrice        float64 `firestore:"scanPrice"`
	TableID          string  `firestore:"tableId"`
	PartitionField   string  `firestore:"partitionField"`
	PartitionType    string  `firestore:"partitionType"`
	PotentialSavings float64 `firestore:"potentialSavings"`
}

type PartitionTable struct {
	DetailedTable              []PartitionDetailTable `firestore:"detailedTable"`
	DetailedTableFieldsMapping map[string]FieldDetail `firestore:"detailedTableFieldsMapping"`
	CommonRecommendation
}

type PartitionDetailTable struct {
	DatasetID                string  `firestore:"datasetId"`
	PotentialPartitionFields string  `firestore:"potentialPartitionFields"`
	PotentialSavings         float64 `firestore:"potentialSavings"`
	ProjectID                string  `firestore:"projectId"`
	ScanPrice                float64 `firestore:"scanPrice"`
	ScanTB                   float64 `firestore:"scanTB"`
	TableIDBaseName          string  `firestore:"tableIdBaseName"`
}

type ClusterTable struct {
	DetailedTable              []ClusterDetailTable   `firestore:"detailedTable"`
	DetailedTableFieldsMapping map[string]FieldDetail `firestore:"detailedTableFieldsMapping"`
	CommonRecommendation
}

type ClusterDetailTable struct {
	DatasetID                 string  `firestore:"datasetId"`
	PotentialClusteringFields string  `firestore:"potentialClusteringFields"`
	PotentialSavings          float64 `firestore:"potentialSavings"`
	ProjectID                 string  `firestore:"projectId"`
	ScanPrice                 float64 `firestore:"scanPrice"`
	ScanTB                    float64 `firestore:"scanTB"`
	TableIDBaseName           string  `firestore:"tableIdBaseName"`
}

type PhysicalStorage struct {
	DetailedTable              []PhysicalStorageDetailTable `firestore:"detailedTable"`
	DetailedTableFieldsMapping map[string]FieldDetail       `firestore:"detailedTableFieldsMapping"`
	CommonRecommendation
}

type PhysicalStorageDetailTable struct {
	DatasetID         string  `firestore:"datasetId"`
	ProjectID         string  `firestore:"projectId"`
	TableID           string  `firestore:"tableId"`
	TotalLogicalGB    float64 `firestore:"totalLogicalGB"`
	TotalPhysicalGB   float64 `firestore:"totalPhysicalGB"`
	TotalLogicalCost  float64 `firestore:"totalLogicalCost"`
	TotalPhysicalCost float64 `firestore:"totalPhysicalCost"`
	CompressionRatio  float64 `firestore:"compressionRatio"`
	Savings           float64 `firestore:"savings"`
}

type BillingProjectScanPrice struct {
	BillingProjectID string                                 `firestore:"billingProjectId,omitempty"`
	ScanPrice        float64                                `firestore:"scanPrice,omitempty"`
	TopQueries       map[string]BillingProjectTopQueryPrice `firestore:"topQueries,omitempty"`
	TopUsers         map[string]float64                     `firestore:"topUsers,omitempty"`
	LastUpdate       time.Time                              `firestore:"lastUpdate"`
}

type BillingProjectScanTB struct {
	BillingProjectID string                              `firestore:"billingProjectId,omitempty"`
	ScanTB           float64                             `firestore:"scanTB,omitempty"`
	TopQueries       map[string]BillingProjectTopQueryTB `firestore:"topQueries,omitempty"`
	TopUsers         map[string]float64                  `firestore:"topUsers,omitempty"`
	LastUpdate       time.Time                           `firestore:"lastUpdate"`
}

type BillingProjectTopQueryPrice struct {
	AvgScanPrice   float64 `firestore:"avgScanPrice"`
	Location       string  `firestore:"location"`
	TotalScanPrice float64 `firestore:"totalScanPrice"`
	UserID         string  `firestore:"userId"`
	CommonTopQuery
}

type BillingProjectTopQueryTB struct {
	AvgScanTB   float64 `firestore:"avgScanTB"`
	Location    string  `firestore:"location"`
	TotalScanTB float64 `firestore:"totalScanTB"`
	UserID      string  `firestore:"userId"`
	CommonTopQuery
}

type DatasetScanPrice struct {
	ProjectID  string                          `firestore:"projectId,omitempty"`
	DatasetID  string                          `firestore:"datasetId,omitempty"`
	ScanPrice  float64                         `firestore:"scanPrice,omitempty"`
	TopQuery   map[string]DatasetTopQueryPrice `firestore:"topQueries,omitempty"`
	TopUsers   map[string]float64              `firestore:"topUsers,omitempty"`
	TopTable   map[string]float64              `firestore:"topTables,omitempty"`
	LastUpdate time.Time                       `firestore:"lastUpdate"`
}

type DatasetTopQueryPrice struct {
	AvgScanPrice   float64 `firestore:"avgScanPrice"`
	DatasetID      string  `firestore:"datasetId"`
	Location       string  `firestore:"location"`
	ProjectID      string  `firestore:"projectId"`
	TotalScanPrice float64 `firestore:"totalScanPrice"`
	UserID         string  `firestore:"userId"`
	CommonTopQuery
}

type DatasetScanTB struct {
	ProjectID  string                       `firestore:"projectId,omitempty"`
	DatasetID  string                       `firestore:"datasetId,omitempty"`
	ScanTB     float64                      `firestore:"scanTB,omitempty"`
	TopQuery   map[string]DatasetTopQueryTB `firestore:"topQueries,omitempty"`
	TopUsers   map[string]float64           `firestore:"topUsers,omitempty"`
	TopTable   map[string]float64           `firestore:"topTables,omitempty"`
	LastUpdate time.Time                    `firestore:"lastUpdate"`
}

type DatasetTopQueryTB struct {
	AvgScanTB   float64 `firestore:"avgScanTB"`
	DatasetID   string  `firestore:"datasetId"`
	Location    string  `firestore:"location"`
	ProjectID   string  `firestore:"projectId"`
	TotalScanTB float64 `firestore:"totalScanTB"`
	UserID      string  `firestore:"userId"`
	CommonTopQuery
}

type ProjectScanPrice struct {
	ProjectID  string                          `firestore:"projectId,omitempty"`
	ScanPrice  float64                         `firestore:"scanPrice,omitempty"`
	TopQuery   map[string]ProjectTopQueryPrice `firestore:"topQueries,omitempty"`
	TopUsers   map[string]float64              `firestore:"topUsers,omitempty"`
	TopTable   map[string]float64              `firestore:"topTables,omitempty"`
	TopDataset map[string]float64              `firestore:"topDatasets,omitempty"`
	LastUpdate time.Time                       `firestore:"lastUpdate"`
}

type ProjectTopQueryPrice struct {
	AvgScanPrice   float64 `firestore:"avgScanPrice"`
	Location       string  `firestore:"location"`
	ProjectID      string  `firestore:"projectId"`
	TotalScanPrice float64 `firestore:"totalScanPrice"`
	UserID         string  `firestore:"userId"`
	CommonTopQuery
}

type ProjectScanTB struct {
	ProjectID  string                          `firestore:"projectId,omitempty"`
	ScanTB     float64                         `firestore:"scanTB,omitempty"`
	TopQuery   map[string]ProjectTopQueryPrice `firestore:"topQueries,omitempty"`
	TopUsers   map[string]float64              `firestore:"topUsers,omitempty"`
	TopTable   map[string]float64              `firestore:"topTables,omitempty"`
	TopDataset map[string]float64              `firestore:"topDatasets,omitempty"`
	LastUpdate time.Time                       `firestore:"lastUpdate"`
}

type ProjectTopQueryTB struct {
	AvgScanTB   float64 `firestore:"avgScanTB"`
	Location    string  `firestore:"location"`
	ProjectID   string  `firestore:"projectId"`
	TotalScanTB float64 `firestore:"totalScanTB"`
	UserID      string  `firestore:"userId"`
	CommonTopQuery
}

type TableScanPrice struct {
	ProjectID  string                        `firestore:"projectId,omitempty"`
	DatasetID  string                        `firestore:"datasetId,omitempty"`
	TableID    string                        `firestore:"tableId,omitempty"`
	TopQuery   map[string]TableTopQueryPrice `firestore:"topQueries,omitempty"`
	TopUsers   map[string]float64            `firestore:"topUsers,omitempty"`
	ScanPrice  float64                       `firestore:"scanPrice,omitempty"`
	LastUpdate time.Time                     `firestore:"lastUpdate"`
}

type TableTopQueryPrice struct {
	AvgScanPrice   float64 `firestore:"avgScanPrice"`
	DatasetID      string  `firestore:"datasetId"`
	Location       string  `firestore:"location"`
	ProjectID      string  `firestore:"projectId"`
	TableID        string  `firestore:"tableId"`
	TotalScanPrice float64 `firestore:"totalScanPrice"`
	UserID         string  `firestore:"userId"`
	CommonTopQuery
}

type TableScanTB struct {
	ProjectID  string                        `firestore:"projectId,omitempty"`
	DatasetID  string                        `firestore:"datasetId,omitempty"`
	TableID    string                        `firestore:"tableId,omitempty"`
	TopQuery   map[string]TableTopQueryPrice `firestore:"topQueries,omitempty"`
	TopUsers   map[string]float64            `firestore:"topUsers,omitempty"`
	ScanTB     float64                       `firestore:"scanTB,omitempty"`
	LastUpdate time.Time                     `firestore:"lastUpdate"`
}

type TableTopQueryTB struct {
	AvgScanTB   float64 `firestore:"avgScanTB"`
	DatasetID   string  `firestore:"datasetId"`
	Location    string  `firestore:"location"`
	ProjectID   string  `firestore:"projectId"`
	TableID     string  `firestore:"tableId"`
	TotalScanTB float64 `firestore:"totalScanTB"`
	UserID      string  `firestore:"userId"`
	CommonTopQuery
}

type UserScanPrice struct {
	UserID     string                       `firestore:"userId,omitempty"`
	ScanPrice  float64                      `firestore:"scanPrice,omitempty"`
	TopDataset map[string]float64           `firestore:"topDatasets,omitempty"`
	TopProject map[string]float64           `firestore:"topProjects,omitempty"`
	TopQuery   map[string]UserTopQueryPrice `firestore:"topQueries,omitempty"`
	TopTable   map[string]float64           `firestore:"topTables,omitempty"`
	LastUpdate time.Time                    `firestore:"lastUpdate"`
}

type UserTopQueryPrice struct {
	AvgScanPrice   float64 `firestore:"avgScanPrice"`
	Location       string  `firestore:"location"`
	TotalScanPrice float64 `firestore:"totalScanPrice"`
	UserID         string  `firestore:"userId"`
	CommonTopQuery
}

type UserScanTB struct {
	ScanTB     float64                   `firestore:"scanTB,omitempty"`
	UserID     string                    `firestore:"userId,omitempty"`
	TopDataset map[string]float64        `firestore:"topDatasets,omitempty"`
	TopProject map[string]float64        `firestore:"topProjects,omitempty"`
	TopQuery   map[string]UserTopQueryTB `firestore:"topQueries,omitempty"`
	TopTable   map[string]float64        `firestore:"topTables,omitempty"`
	LastUpdate time.Time                 `firestore:"lastUpdate"`
}

type UserTopQueryTB struct {
	AvgScanTB   float64 `firestore:"avgScanTB"`
	Location    string  `firestore:"location"`
	TotalScanTB float64 `firestore:"totalScanTB"`
	UserID      string  `firestore:"userId"`
	CommonTopQuery
}
