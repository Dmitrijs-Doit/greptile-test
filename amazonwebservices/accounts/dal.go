package accounts

import (
	"context"

	"cloud.google.com/go/firestore"

	sharedfs "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

const (
	integrationsCollection    = "integrations"
	amazonWebServicesDocument = "amazon-web-services"
	accountsCollection        = "accounts"
)

//go:generate mockery --name Dal --inpackage
type Dal interface {
	findAccountByID(ctx context.Context, payerID string) (*Account, error)
}

type dal struct {
	firestoreClient  connection.FirestoreFromContextFun
	documentsHandler iface.DocumentsHandler
}

func newDal(fs *firestore.Client) Dal {
	return &dal{
		firestoreClient: func(ctx context.Context) *firestore.Client {
			return fs
		},
		documentsHandler: sharedfs.DocumentHandler{},
	}
}

func (d *dal) accountsCollection(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClient(ctx).Collection(integrationsCollection).Doc(amazonWebServicesDocument).Collection(accountsCollection)
}

func (d *dal) findAccountByID(ctx context.Context, id string) (*Account, error) {
	snap, err := d.documentsHandler.Get(ctx, d.accountsCollection(ctx).Doc(id))
	if err != nil {
		return nil, err
	}

	accountData := &Account{}

	err = snap.DataTo(accountData)
	if err != nil {
		return nil, err
	}

	return accountData, nil
}
