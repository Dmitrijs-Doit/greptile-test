package firestoremodels

import (
	"time"

	"cloud.google.com/go/firestore"
)

// hybrid (output)
type CostFromTableTypeDocument struct {
	Data       map[string]CostFromTableType `firestore:"data"`
	LastUpdate time.Time                    `firestore:"lastUpdate"`
}

type StorageSavingsDocument struct {
	StorageSavings StorageSavings `firestore:"storageSavings"`
	LastUpdate     time.Time      `firestore:"lastUpdate"`
}

type TableStorageTBDocument map[string]TableStorageTB
type TableStoragePriceDocument map[string]TableStoragePrice
type DatasetStorageTBDocument map[string]DatasetStorageTB
type DatasetStoragePriceDocument map[string]DatasetStoragePrice
type ProjectStorageTBDocument map[string]ProjectStorageTB
type ProjectStoragePriceDocument map[string]ProjectStoragePrice

// flat-rate and editions
type ScheduledQueriesDocument struct {
	Data       ScheduledQueriesMovement `firestore:"scheduledQueriesMovement"`
	LastUpdate time.Time                `firestore:"lastUpdate"`
}

type ExplorerDocument struct {
	Day        TimeSeriesData `firestore:"day"`
	Hour       TimeSeriesData `firestore:"hour"`
	LastUpdate time.Time      `firestore:"lastUpdate"`
}

type UserSlotsDocument map[string]UserSlots

type BillingProjectDocument map[string]BillingProject

// on-demand
type RecommendationsDocument struct {
	LimitingJobs         *LimitingJobsSavings `firestore:"limitingJobsSavings,omitempty"`
	UsePartition         *UsePartitionField   `firestore:"usePartitionField,omitempty"`
	PartitionTables      *PartitionTable      `firestore:"partitionTables,omitempty"`
	Cluster              *ClusterTable        `firestore:"clusterTables,omitempty"`
	PhysicalStorageTable *PhysicalStorage     `firestore:"physicalStorage,omitempty"`
	LastUpdate           time.Time            `firestore:"lastUpdate,omitempty"`
}

type BillingProjectScanPriceDocument map[string]BillingProjectScanPrice
type BillingProjectScanTBDocument map[string]BillingProjectScanTB
type DatasetScanPriceDocument map[string]DatasetScanPrice
type DatasetScanTBDocument map[string]DatasetScanTB
type ProjectScanPriceDocument map[string]ProjectScanPrice
type ProjectScanTBDocument map[string]ProjectScanTB
type TableScanPriceDocument map[string]TableScanPrice
type TableScanTBDocument map[string]TableScanTB
type UserScanPriceDocument map[string]UserScanPrice
type UserScanTBDocument map[string]UserScanTB

type BackfillDocument struct {
	Customer         *firestore.DocumentRef `firestore:"customer"`
	BackfillDone     bool                   `firestore:"backfillDone"`
	BackfillProgress float64                `firestore:"backfillProgress"`
}
