package dal

import (
	"context"

	"cloud.google.com/go/firestore"
	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/credit"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

const (
	customerCreditsCollection = "customerCredits"
)

// CreditsFirestore is used to interact with Credits stored on Firestore.
type CreditsFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

// NewCreditsFirestoreWithClient returns a new CreditsFirestore using given client.
func NewCreditsFirestoreWithClient(fun connection.FirestoreFromContextFun) *CreditsFirestore {
	return &CreditsFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (s *CreditsFirestore) GetCredits(ctx context.Context) (map[*firestore.DocumentRef]credit.BaseCredit, error) {
	fs := s.firestoreClientFun(ctx)
	iter := fs.CollectionGroup(customerCreditsCollection).Select("name", "utilization", "type", "customer").Documents(ctx)

	creditDocsnaps, err := s.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	customerCredits := make(map[*firestore.DocumentRef]credit.BaseCredit, len(creditDocsnaps))

	for _, creditDocsnap := range creditDocsnaps {
		var credit credit.BaseCredit
		if err := creditDocsnap.DataTo(&credit); err != nil {
			return nil, err
		}

		customerCredits[creditDocsnap.Snapshot().Ref] = credit
	}

	return customerCredits, nil
}
