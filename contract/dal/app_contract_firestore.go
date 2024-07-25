package dal

import (
	"context"

	"cloud.google.com/go/firestore"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

const (
	appCollection              = "app"
	contractsDoc               = "contracts"
	acceleratorTypesCollection = "acceleratorTypes"
)

// ContractFirestore is used to interact with contracts stored on Firestore.
type AppContractsFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

// NewContractFirestore returns a new ContractFirestore instance with given project id.
func NewAppContractsFirestore(ctx context.Context, projectID string) (*ContractFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewContractFirestoreWithClient(
		func(_ context.Context) *firestore.Client {
			return fs
		},
	), nil
}

// NewAppContractFirestoreWithClient returns a new ContractFirestore using given client.
func NewAppContractFirestoreWithClient(fun connection.FirestoreFromContextFun) *AppContractsFirestore {
	return &AppContractsFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *AppContractsFirestore) GetRef(ctx context.Context, id string) *firestore.DocumentRef {
	return d.firestoreClientFun(ctx).
		Collection(appCollection).
		Doc(contractsCollection).
		Collection(acceleratorTypesCollection).
		Doc(id)
}
