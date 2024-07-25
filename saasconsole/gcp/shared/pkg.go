package shared

import (
	"time"
)

type GCPBillingImportStatus struct {
	MaxStartTime          time.Time           `firestore:"maxStartTime"`
	MaxTotalExecutionTime time.Time           `firestore:"maxTotalExecutionTime"`
	Status                BillingImportStatus `firestore:"status"`
	Error                 string              `firestore:"error"`
}

type BillingImportStatus string

const (
	BillingImportStatusPending   BillingImportStatus = "pending"
	BillingImportStatusStarted   BillingImportStatus = "started"
	BillingImportStatusCompleted BillingImportStatus = "completed"
	BillingImportStatusEnabled   BillingImportStatus = "customer-enabled"
	BillingImportStatusFailed    BillingImportStatus = "failed"
)
