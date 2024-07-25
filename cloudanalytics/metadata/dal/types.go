package dal

import (
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

type MetadataFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}
