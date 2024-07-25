package dal

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

const (
	tenantsCollection = "tenants"
)

// TenantsFirestore is used to interact with tenants stored on Firestore.
type TenantsFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

// NewTenantsFirestore returns a new TenantsFirestore instance
func NewTenantsFirestore(ctx context.Context) (*TenantsFirestore, error) {
	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return nil, err
	}

	return NewTenantsFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		},
	), nil
}

// NewTenantsFirestoreWithClient returns a new TenantsFirestore using given client.
func NewTenantsFirestoreWithClient(fun connection.FirestoreFromContextFun) *TenantsFirestore {
	return &TenantsFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *TenantsFirestore) GetTenantsDocRef(ctx context.Context) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection(tenantsCollection).Doc(common.ProjectID)
}

func (d *TenantsFirestore) GetTenantIDByCustomer(ctx context.Context, customerID string) (*string, error) {
	doc := d.GetTenantsDocRef(ctx)
	customerToTenantDoc := doc.Collection("customerToTenant").Doc(customerID)
	snap, err := d.documentsHandler.Get(ctx, customerToTenantDoc)

	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	tenant, err := snap.DataAt("tenantId")
	if err != nil {
		return nil, err
	}

	tID, ok := tenant.(string)
	if !ok {
		return nil, fmt.Errorf("failed casting tenant id to string for customer: %s", customerID)
	}

	return &tID, nil
}

func (d *TenantsFirestore) GetTenantIDByEmail(ctx context.Context, email string) (*string, error) {
	doc := d.GetTenantsDocRef(ctx)
	customerToTenantDoc := doc.Collection("emailToTenant").Doc(email)
	snap, err := d.documentsHandler.Get(ctx, customerToTenantDoc)

	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	tenant, err := snap.DataAt("tenantId")
	if err != nil {
		return nil, err
	}

	tID, ok := tenant.(string)
	if !ok {
		return nil, fmt.Errorf("failed casting tenant id to string for email: %s", email)
	}

	return &tID, nil
}

func (d *TenantsFirestore) GetCustomerIDByTenant(ctx context.Context, tenantID string) (*string, error) {
	doc := d.GetTenantsDocRef(ctx)
	customerToTenantDoc := doc.Collection("tenantToCustomer").Doc(tenantID)
	snap, err := d.documentsHandler.Get(ctx, customerToTenantDoc)

	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	customer, err := snap.DataAt("customerId")
	if err != nil {
		return nil, err
	}

	cID, ok := customer.(string)

	if !ok {
		return nil, fmt.Errorf("failed casting customer id to string for tenant: %s", tenantID)
	}

	return &cID, nil
}
