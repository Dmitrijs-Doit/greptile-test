package dal

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type firestoreTransaction struct {
	loggerProvider logger.Provider
	*connection.Connection
}

func newFirestoreTransaction(log logger.Provider, conn *connection.Connection) *firestoreTransaction {
	return &firestoreTransaction{
		log,
		conn,
	}
}

func (s *firestoreTransaction) executeTransaction(ctx context.Context, docRef *firestore.DocumentRef, fn utils.TransactionFunc, aux interface{}) (interface{}, error) {
	var returnValue interface{}

	err := s.Firestore(ctx).RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		ref, err := tx.Get(docRef)
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}

		returnValue, err = fn(ref, aux)
		if err != nil {
			return err
		}

		return tx.Set(docRef, returnValue)
	}, firestore.MaxAttempts(20))
	if err != nil {
		return nil, err
	}

	return returnValue, nil
}
