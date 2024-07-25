package dal

import (
	"context"

	"cloud.google.com/go/firestore"

	doitFS "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/bq-lens/pricebook/domain"
)

//go:generate mockery --name Pricebook --output ./mocks --case=underscore
type Pricebook interface {
	Get(ctx context.Context, edition domain.Edition) (*domain.PricebookDocument, error)
	Set(ctx context.Context, edition domain.Edition, data domain.PricebookDocument) error
}

type PricebookDAL struct {
	firestoreClientFun iface.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

func NewPricebookDALWithClient(fun iface.FirestoreFromContextFun) *PricebookDAL {
	return &PricebookDAL{
		firestoreClientFun: fun,
		documentsHandler:   doitFS.DocumentHandler{},
	}
}

func (d *PricebookDAL) Get(ctx context.Context, edition domain.Edition) (*domain.PricebookDocument, error) {
	doc := d.getEditionDocumentRef(ctx, edition)

	return doitFS.GetDocument[domain.PricebookDocument](ctx, d.documentsHandler, doc)
}

func (d *PricebookDAL) Set(ctx context.Context, edition domain.Edition, data domain.PricebookDocument) error {
	_, err := d.documentsHandler.Set(ctx, d.getEditionDocumentRef(ctx, edition), data)

	return err
}

func (d *PricebookDAL) getEditionDocumentRef(ctx context.Context, edition domain.Edition) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).Collection("superQuery").Doc("pricebook").Collection("edition-pricebook").Doc(string(edition))
}
