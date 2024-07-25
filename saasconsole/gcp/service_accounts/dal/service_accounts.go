package dal

import (
	"context"

	"cloud.google.com/go/firestore"
	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/service_accounts/utils"
)

// ServiceAccountsFirestore
type ServiceAccountsFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
	tx                 *firestoreTransaction
}

func NewServiceAccountsFirestoreWithClient(log logger.Provider, conn *connection.Connection) *ServiceAccountsFirestore {
	return &ServiceAccountsFirestore{
		firestoreClientFun: conn.Firestore,
		documentsHandler:   doitFirestore.DocumentHandler{},
		tx:                 newFirestoreTransaction(log, conn),
	}
}

// Refs

func (d *ServiceAccountsFirestore) GetOnboardingColRef(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(utils.IntegrationsCollection).Doc(utils.BillingStandaloneCollection).Collection(utils.ServiceAccountsCollection)
}
