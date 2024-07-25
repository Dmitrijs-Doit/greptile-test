package dal

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	doitFirestore "github.com/doitintl/firestore"
	firestorePkg "github.com/doitintl/firestore/pkg"

	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	utils "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone"
	pkg "github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/shared"
)

const (
	customersCollection          = "customers"
	flexsaveStandaloneCollection = "flexsaveStandalone"
	billingImportStatusKey       = "billingImportStatus"
)

type BillingImportStatusDAL struct {
	firestoreClient  *firestore.Client
	documentsHandler iface.DocumentsHandler
}

/*
DAL BillingImportStatus interacts with customer's flexsave standalone billing import status:
	customers/CUSTOMER_ID/flexsaveStandalone/google-cloud-standalone-BILLING_ACCOUNT_ID
*/

type BillingImportStatus interface {
	GetBillingImportStatus(ctx context.Context, customerID, billingAccountID string) (*pkg.GCPBillingImportStatus, error)
	UpdateError(ctx context.Context, customerID, billingAccountID, err string) error
	UpdateMaxTimesThresholds(ctx context.Context, customerID, billingAccountID string) error
	UpdateStatus(ctx context.Context, customerID, billingAccountID string, status pkg.BillingImportStatus) error
	SetStatusPending(ctx context.Context, customerID, billingAccountID string) error
	SetStatusStarted(ctx context.Context, customerID, billingAccountID string) error
	SetStatusCompleted(ctx context.Context, customerID, billingAccountID string) error
	SetStatusEnabled(ctx context.Context, customerID, billingAccountID string) error
	SetStatusFailed(ctx context.Context, customerID, billingAccountID string) error
	ListCustomersCompletedBillingImport(ctx context.Context) (map[string]*firestore.DocumentRef, error)
}

// NewBillingImportStatus returns a new BillingImportStatusDAL instance with active project id.
func NewBillingImportStatus(ctx context.Context) (BillingImportStatus, error) {
	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return nil, err
	}

	return NewBillingImportStatusWithClient(fs), nil
}

func NewBillingImportStatusWithClient(firestoreClient *firestore.Client) BillingImportStatus {
	return &BillingImportStatusDAL{
		firestoreClient,
		doitFirestore.DocumentHandler{},
	}
}

func (d *BillingImportStatusDAL) customerDocRef(customerID string) *firestore.DocumentRef {
	return d.firestoreClient.Collection(customersCollection).Doc(customerID)
}

func (d *BillingImportStatusDAL) flexsaveStandaloneCollection(customerID string) *firestore.CollectionRef {
	return d.customerDocRef(customerID).Collection(flexsaveStandaloneCollection)
}

func (d *BillingImportStatusDAL) flexsaveStandaloneCollectionGroup() *firestore.CollectionGroupRef {
	return d.firestoreClient.CollectionGroup(flexsaveStandaloneCollection)
}

func (d *BillingImportStatusDAL) billingImportStatusDocRef(customerID, billingAccountID string) *firestore.DocumentRef {
	docID := utils.GetAssetID(firestorePkg.GCP, billingAccountID)
	return d.flexsaveStandaloneCollection(customerID).Doc(docID)
}

// getMaxTimes calculates times thresholds starting now
func (d *BillingImportStatusDAL) getMaxTimes() (time.Time, time.Time) {
	startTimeThreshold := time.Hour
	singleDay := 24 * time.Hour
	totalExecutionThreshold := 3 * singleDay

	maxStartTime := time.Now().Add(startTimeThreshold)
	maxTotalExecutionTime := time.Now().Add(startTimeThreshold).Add(totalExecutionThreshold)

	return maxStartTime, maxTotalExecutionTime
}

func (d *BillingImportStatusDAL) updateField(ctx context.Context, customerID, billingAccountID, key string, value interface{}) error {
	docRef := d.billingImportStatusDocRef(customerID, billingAccountID)

	fields := map[string]interface{}{
		billingImportStatusKey: map[string]interface{}{
			key: value,
		},
		"customer": d.customerDocRef(customerID),
	}

	if _, err := docRef.Set(ctx, fields, firestore.MergeAll); err != nil {
		return err
	}

	return nil
}

// GetBillingImportStatus returns import billing status for a given customer & billing account id
func (d *BillingImportStatusDAL) GetBillingImportStatus(ctx context.Context, customerID, billingAccountID string) (*pkg.GCPBillingImportStatus, error) {
	docSnap, err := d.billingImportStatusDocRef(customerID, billingAccountID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var flexsaveStandaloneDoc struct {
		BillingImportStatus *pkg.GCPBillingImportStatus
	}

	if err := docSnap.DataTo(&flexsaveStandaloneDoc); err != nil {
		return nil, err
	}

	return flexsaveStandaloneDoc.BillingImportStatus, nil
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

// UpdateStatus updates billingImportStatus.status field for a given customer & billing account id
func (d *BillingImportStatusDAL) UpdateStatus(ctx context.Context, customerID, billingAccountID string, status pkg.BillingImportStatus) error {
	return d.updateField(ctx, customerID, billingAccountID, "status", status)
}

// SetStatusPending sets billingImportStatus.status = "pending" for a given customer & billing account id
func (d *BillingImportStatusDAL) SetStatusPending(ctx context.Context, customerID, billingAccountID string) error {
	return d.UpdateStatus(ctx, customerID, billingAccountID, pkg.BillingImportStatusPending)
}

// SetStatusStarted sets billingImportStatus.status = "started" for a given customer & billing account id
func (d *BillingImportStatusDAL) SetStatusStarted(ctx context.Context, customerID, billingAccountID string) error {
	return d.UpdateStatus(ctx, customerID, billingAccountID, pkg.BillingImportStatusStarted)
}

// SetStatusCompleted sets billingImportStatus.status = "completed" for a given customer & billing account idw
func (d *BillingImportStatusDAL) SetStatusCompleted(ctx context.Context, customerID, billingAccountID string) error {
	return d.UpdateStatus(ctx, customerID, billingAccountID, pkg.BillingImportStatusCompleted)
}

// SetStatusEnabled sets billingImportStatus.status = "customer-enabled" for a given customer & billing account id
func (d *BillingImportStatusDAL) SetStatusEnabled(ctx context.Context, customerID, billingAccountID string) error {
	return d.UpdateStatus(ctx, customerID, billingAccountID, pkg.BillingImportStatusEnabled)
}

// SetStatusFailed sets billingImportStatus.status = "failed" for a given customer & billing account id
func (d *BillingImportStatusDAL) SetStatusFailed(ctx context.Context, customerID, billingAccountID string) error {
	return d.UpdateStatus(ctx, customerID, billingAccountID, pkg.BillingImportStatusFailed)
}

func (d *BillingImportStatusDAL) ListCustomersCompletedBillingImport(ctx context.Context) (map[string]*firestore.DocumentRef, error) {
	snaps, err := d.flexsaveStandaloneCollectionGroup().
		Where("billingImportStatus.status", "==", pkg.BillingImportStatusCompleted).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	customersCompletedBillingImport := make(map[string]*firestore.DocumentRef)

	for _, snap := range snaps {
		doc := struct {
			Customer *firestore.DocumentRef
		}{}
		if err := snap.DataTo(&doc); err != nil {
			return nil, err
		}

		customersCompletedBillingImport[doc.Customer.ID] = snap.Ref
	}

	return customersCompletedBillingImport, err
}
