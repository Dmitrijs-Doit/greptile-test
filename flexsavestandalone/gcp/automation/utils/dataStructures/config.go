package dataStructures

type PipelineConfig struct {
	RegionsBuckets               map[string]string `json:"regionsBuckets" firestore:"regionsBuckets"`
	DestinationTableFormat       string            `json:"destinationTableFormat" firestore:"destinationTableFormat"`
	DestinationDatasetFormat     string            `json:"destinationDatasetFormat" firestore:"destinationDatasetFormat"`
	DestinationProject           string            `json:"destinationProject" firestore:"destinationProject"`
	TemplateBillingDataProjectID string            `json:"templateBillingDataProjectID" firestore:"templateBillingDataProjectID"`
	TemplateBillingDataDatasetID string            `json:"templateBillingDataDatasetID" firestore:"templateBillingDataDatasetID"`
	TemplateBillingDataTableID   string            `json:"templateBillingDataTableID" firestore:"templateBillingDataTableID"`
	//StorageRole                  string            `json:"storageRole" firestore:"storageRole"`
}
