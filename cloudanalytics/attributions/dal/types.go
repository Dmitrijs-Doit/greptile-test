package dal

import (
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	labelsDalIface "github.com/doitintl/hello/scheduled-tasks/labels/dal/iface"
)

// AttributionsFirestore is used to interact with cloud analytics attributions stored on Firestore.
type AttributionsFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
	labelsDal          labelsDalIface.Labels
}
