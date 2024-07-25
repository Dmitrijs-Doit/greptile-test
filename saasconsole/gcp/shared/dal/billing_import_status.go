package dal

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	doitFirestore "github.com/doitintl/firestore"
	firestorePkg "github.com/doitintl/firestore/pkg"

	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	utils "github.com/doitintl/hello/scheduled-tasks/saasconsole"
	pkg "github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/shared"
)

const (
	customersCollection         = "customers"
	billingstandaloneCollection = "billingStandalone"
	billingImportStatusKey      = "billingImportStatus"
)

type BillingImportStatusDAL struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

/*
DAL BillingImportStatus interacts with customer's SaaS Console billing import status:
	customers/CUSTOMER_ID/saasconsole/google-cloud-standalone-BILLING_ACCOUNT_ID
*/

type BillingImportStatus interface {
	GetBillingImportStatus(ctx context.Context, customerID, billingAccountID string) (*pkg.GCPBillingImportStatus, error)
	UpdateError(ctx context.Context, customerID, billingAccountID, err string) error
	UpdateMaxTimesThresholds(ctx context.Context, customerID, billingAccountID string) error
	SetStatusPending(ctx context.Context, customerID, billingAccountID string) error
	SetStatusStarted(ctx context.Context, customerID, billingAccountID string) error
	SetStatusCompleted(ctx context.Context, customerID, billingAccountID string) error
	SetStatusEnabled(ctx context.Context, customerID, billingAccountID string) error
	SetStatusFailed(ctx context.Context, customerID, billingAccountID string) error
}

// NewBillingImportStatus returns a new BillingImportStatusDAL instance with active project id.
func NewBillingImportStatus(ctx context.Context) (BillingImportStatus, error) {
	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return nil, err
	}

	fsClient := NewBillingImportStatusWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		})

	return fsClient, nil
}
func NewBillingImportStatusWithClient(fun connection.FirestoreFromContextFun) BillingImportStatus {
	return &BillingImportStatusDAL{
		fun,
		doitFirestore.DocumentHandler{},
	}
}

func (d *BillingImportStatusDAL) customerDocRef(ctx context.Context, customerID string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection(customersCollection).Doc(customerID)
}

func (d *BillingImportStatusDAL) billingStandaloneCollection(ctx context.Context, customerID string) *firestore.CollectionRef {
	return d.customerDocRef(ctx, customerID).Collection(billingstandaloneCollection)
}

func (d *BillingImportStatusDAL) billingImportStatusDocRef(ctx context.Context, customerID, billingAccountID string) *firestore.DocumentRef {
	docID := utils.GetAssetID(firestorePkg.GCP, billingAccountID)
	return d.billingStandaloneCollection(ctx, customerID).Doc(docID)
}

func (d *BillingImportStatusDAL) getMaxTimes() (time.Time, time.Time) {
	startTimeThreshold := time.Hour
	singleDay := 24 * time.Hour
	totalExecutionThreshold := 3 * singleDay

	maxStartTime := time.Now().Add(startTimeThreshold)
	maxTotalExecutionTime := time.Now().Add(startTimeThreshold).Add(totalExecutionThreshold)

	return maxStartTime, maxTotalExecutionTime
}

func (d *BillingImportStatusDAL) updateField(ctx context.Context, customerID, billingAccountID, key string, value interface{}) error {
	docRef := d.billingImportStatusDocRef(ctx, customerID, billingAccountID)

	fields := map[string]interface{}{
		billingImportStatusKey: map[string]interface{}{
			key: value,
		},
		"customer": d.customerDocRef(ctx, customerID),
	}

	if _, err := docRef.Set(ctx, fields, firestore.MergeAll); err != nil {
		return err
	}

	return nil
}

// GetBillingImportStatus returns import billing status for a given customer & billing account id
func (d *BillingImportStatusDAL) GetBillingImportStatus(ctx context.Context, customerID, billingAccountID string) (*pkg.GCPBillingImportStatus, error) {
	docSnap, err := d.billingImportStatusDocRef(ctx, customerID, billingAccountID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var billingstandaloneDoc struct {
		BillingImportStatus *pkg.GCPBillingImportStatus
	}

	if err := docSnap.DataTo(&billingstandaloneDoc); err != nil {
		return nil, err
	}

	return billingstandaloneDoc.BillingImportStatus, nil
}

// UpdateError update billing import error
func (d *BillingImportStatusDAL) UpdateError(ctx context.Context, customerID, billingAccountID, err string) error {
	return d.updateField(ctx, customerID, billingAccountID, "error", err)
}

// UpdateMaxTimesThresholds update time thresholds for billing import process steps
func (d *BillingImportStatusDAL) UpdateMaxTimesThresholds(ctx context.Context, customerID, billingAccountID string) error {
	maxStartTime, maxTotalExecutionTime := d.getMaxTimes()
	if err := d.updateField(ctx, customerID, billingAccountID, "maxStartTime", maxStartTime); err != nil {
		return err
	}

	err := d.updateField(ctx, customerID, billingAccountID, "maxTotalExecutionTime", maxTotalExecutionTime)

	return err
}

// SetStatusPending sets billingImportStatus.status = "pending" for a given customer & billing account id
func (d *BillingImportStatusDAL) SetStatusPending(ctx context.Context, customerID, billingAccountID string) error {
	return d.updateStatus(ctx, customerID, billingAccountID, pkg.BillingImportStatusPending)
}

// SetStatusStarted sets billingImportStatus.status = "started" for a given customer & billing account id
func (d *BillingImportStatusDAL) SetStatusStarted(ctx context.Context, customerID, billingAccountID string) error {
	return d.updateStatus(ctx, customerID, billingAccountID, pkg.BillingImportStatusStarted)
}

// SetStatusCompleted sets billingImportStatus.status = "completed" for a given customer & billing account idw
func (d *BillingImportStatusDAL) SetStatusCompleted(ctx context.Context, customerID, billingAccountID string) error {
	return d.updateStatus(ctx, customerID, billingAccountID, pkg.BillingImportStatusCompleted)
}

// SetStatusEnabled sets billingImportStatus.status = "customer-enabled" for a given customer & billing account id
func (d *BillingImportStatusDAL) SetStatusEnabled(ctx context.Context, customerID, billingAccountID string) error {
	return d.updateStatus(ctx, customerID, billingAccountID, pkg.BillingImportStatusEnabled)
}

// SetStatusFailed sets billingImportStatus.status = "failed" for a given customer & billing account id
func (d *BillingImportStatusDAL) SetStatusFailed(ctx context.Context, customerID, billingAccountID string) error {
	return d.updateStatus(ctx, customerID, billingAccountID, pkg.BillingImportStatusFailed)
}

// UpdateStatus updates billingImportStatus.status field for a given customer & billing account id
func (d *BillingImportStatusDAL) updateStatus(ctx context.Context, customerID, billingAccountID string, status pkg.BillingImportStatus) error {
	return d.updateField(ctx, customerID, billingAccountID, "status", status)
}
