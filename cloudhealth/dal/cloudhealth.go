package dal

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"

	sharedfs "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
)

const customerNotFoundCode = "customer-not-found"

type CHTError struct {
	customerID string
	code       string
}

func (e *CHTError) Error() string {
	return fmt.Sprintf("no CloudHealth customer doc found for customer: %v", e.customerID)
}

func (e *CHTError) Is(tgt error) bool {
	target, ok := tgt.(*CHTError)
	if !ok {
		return false
	}

	return target.code == customerNotFoundCode
}

// CloudHealthDAL is used to interact with CloudHealth stored on Firestore.
type CloudHealthDAL struct {
	firestoreClient  *firestore.Client
	documentsHandler iface.DocumentsHandler
}

// NewCloudHealthDAL returns a new CloudHealthDAL using given client.
func NewCloudHealthDAL(fs *firestore.Client) *CloudHealthDAL {
	return &CloudHealthDAL{
		firestoreClient:  fs,
		documentsHandler: sharedfs.DocumentHandler{},
	}
}

func (d *CloudHealthDAL) cloudHealthCustomerCollectionRef() *firestore.CollectionRef {
	return d.firestoreClient.Collection("integrations").Doc("cloudhealth").Collection("cloudhealthCustomers")
}

func (d *CloudHealthDAL) GetCustomerCloudHealthID(ctx context.Context, customerRef *firestore.DocumentRef) (string, error) {
	iter := d.cloudHealthCustomerCollectionRef().
		Where("customer", "==", customerRef).
		Where("disabled", "==", false).
		Limit(1).
		Documents(ctx)

	snaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return "", err
	}

	if len(snaps) == 0 {
		return "", &CHTError{customerRef.ID, customerNotFoundCode}
	}

	return snaps[0].ID(), nil
}
