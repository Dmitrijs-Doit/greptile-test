package dal

import (
	"context"
	"fmt"

	"google.golang.org/api/iterator"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/consts"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils/dataStructures"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

type RowsValidatorFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

func NewRowsValidatorWithClient(fun connection.FirestoreFromContextFun) *RowsValidatorFirestore {
	return &RowsValidatorFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *RowsValidatorFirestore) GetDocRef(ctx context.Context, billingAccountID string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection(consts.IntegrationsCollection).Doc(consts.GCPFlexsaveStandaloneDoc).Collection(consts.RowsValidator).Doc(billingAccountID)
}

func (d *RowsValidatorFirestore) DeleteDocsRef(ctx context.Context) error {
	it := d.firestoreClientFun(ctx).Collection(consts.IntegrationsCollection).Doc(consts.GCPFlexsaveStandaloneDoc).Collection(consts.RowsValidator).DocumentRefs(ctx)

	for {
		doc, err := it.Next()
		if err != nil {
			if err == iterator.Done {
				return nil
			}

			err = fmt.Errorf("unable to get doc. Caused by %s", err)

			return err
		}

		_, err = doc.Delete(ctx)
		if err != nil {
			err = fmt.Errorf("unable to delete doc. Caused by %s", err)
			return err
		}
	}
}

func (d *RowsValidatorFirestore) GetRowsValidatorMetadata(ctx context.Context, billingAccountID string) (*dataStructures.RowsValidatorMetadata, error) {
	snap, err := d.documentsHandler.Get(ctx, d.GetDocRef(ctx, billingAccountID))
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	var md dataStructures.RowsValidatorMetadata

	err = snap.DataTo(&md)
	if err != nil {
		return nil, err
	}

	return &md, nil
}

func (d *RowsValidatorFirestore) SetRowsValidatorMetadata(ctx context.Context, billingAccountID string, config *dataStructures.RowsValidatorMetadata) error {
	_, err := d.documentsHandler.Set(ctx, d.GetDocRef(ctx, billingAccountID), config)
	if err != nil {
		return err
	}

	return nil
}
