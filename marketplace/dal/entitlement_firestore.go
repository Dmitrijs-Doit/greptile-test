package dal

import (
	"context"
	"errors"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	firestoreIface "github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/marketplace/domain"
)

const (
	entitlementCollection = "marketplace/gcp-marketplace/gcpMarketplaceEntitlements"
)

var (
	ErrEntitlementNotFound            = errors.New("entitlement not found")
	ErrProcurementEntitlementNotFound = errors.New("procurement entitlement not found")
)

type EntitlementFirestoreDAL struct {
	firestoreClientFun firestoreIface.FirestoreFromContextFun
	documentsHandler   firestoreIface.DocumentsHandler
}

func NewEntitlementFirestoreDAL(ctx context.Context, projectID string) (*EntitlementFirestoreDAL, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewEntitlementFirestoreDALWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		}), nil
}

func NewEntitlementFirestoreDALWithClient(fun firestoreIface.FirestoreFromContextFun) *EntitlementFirestoreDAL {
	return &EntitlementFirestoreDAL{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *EntitlementFirestoreDAL) entitlementsCollection(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(entitlementCollection)
}

func (d *EntitlementFirestoreDAL) getEntitlementRef(ctx context.Context, docID string) *firestore.DocumentRef {
	return d.entitlementsCollection(ctx).Doc(docID)
}

func (d *EntitlementFirestoreDAL) GetEntitlement(ctx context.Context, entitlementID string) (*domain.EntitlementFirestore, error) {
	entitlementRef := d.getEntitlementRef(ctx, entitlementID)

	docSnap, err := d.documentsHandler.Get(ctx, entitlementRef)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrEntitlementNotFound
		}

		return nil, err
	}

	var entitlementFirestore domain.EntitlementFirestore

	if err := docSnap.DataTo(&entitlementFirestore); err != nil {
		return nil, err
	}

	return &entitlementFirestore, nil
}
