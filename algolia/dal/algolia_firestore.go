package dal

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/algolia"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

type AlgoliaFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

// NewAlgoliaFirestoreWithClient returns a new AlgoliaFirestore using given client.
func NewAlgoliaFirestoreWithClient(fun connection.FirestoreFromContextFun) *AlgoliaFirestore {
	return &AlgoliaFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *AlgoliaFirestore) GetConfigFromFirestore(ctx context.Context) (*algolia.Config, error) {
	docRef := d.firestoreClientFun(ctx).Collection("app").Doc("algolia")

	snap, err := d.documentsHandler.Get(ctx, docRef)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	var c algolia.FirestoreConfig

	if err = snap.DataTo(&c); err != nil {
		return nil, err
	}

	if !common.Production && c.DevAppID != "" && c.DevSearchKey != "" {
		return &algolia.Config{AppID: c.DevAppID, SearchKey: c.DevSearchKey}, nil
	}

	return &algolia.Config{AppID: c.AppID, SearchKey: c.SearchKey}, nil
}
