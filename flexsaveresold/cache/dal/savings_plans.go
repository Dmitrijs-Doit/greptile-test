package dal

import (
	"context"

	"cloud.google.com/go/firestore"

	sharedfs "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
)

// SavingsPlansDAL is used to interact with SavingsPlans stored in Firestore.
type SavingsPlansDAL struct {
	firestoreClient   *firestore.Client
	documentsHandler  iface.DocumentsHandler
	transactionRunner iface.TransactionRunner
}

// NewSavingsPlansDAL returns a new SavingsPlansDAL using given client.
func NewSavingsPlansDAL(fs *firestore.Client) *SavingsPlansDAL {
	return &SavingsPlansDAL{
		firestoreClient:   fs,
		documentsHandler:  sharedfs.DocumentHandler{},
		transactionRunner: sharedfs.NewTransactionRunnerWithClient(fs),
	}
}

func (d *SavingsPlansDAL) savingsPlansCollectionRef() *firestore.CollectionRef {
	return d.firestoreClient.Collection("integrations").Doc("flexsave").Collection("customer-savings-plans")
}

// CreateCustomerSavingsPlansCache creates or recreates the savings plan document in flexsave/customer-savings-plans collection.
func (d *SavingsPlansDAL) CreateCustomerSavingsPlansCache(ctx context.Context, customerID string, savingsPlans []types.SavingsPlanData) error {
	var ref = d.savingsPlansCollectionRef().Doc(customerID)

	savingsPlanDoc := types.SavingsPlanDoc{
		SavingsPlans: savingsPlans,
	}

	_, err := d.documentsHandler.Set(ctx, ref, savingsPlanDoc)
	if err != nil {
		return err
	}

	return nil
}
