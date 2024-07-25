package dal

import (
	"context"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	firestoreIface "github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/iam/permission/domain"
)

const (
	permissionCollection = "permissions"
)

type PermissionFirestoreDAL struct {
	firestoreClientFun firestoreIface.FirestoreFromContextFun
	documentsHandler   firestoreIface.DocumentsHandler
}

func NewPermissionFirestoreDAL(ctx context.Context, projectID string) (*PermissionFirestoreDAL, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewPermissionFirestoreDALWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		}), nil
}

func NewPermissionFirestoreDALWithClient(fun firestoreIface.FirestoreFromContextFun) *PermissionFirestoreDAL {
	return &PermissionFirestoreDAL{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *PermissionFirestoreDAL) permissionCollection(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(permissionCollection)
}

func (d *PermissionFirestoreDAL) getPermissionRef(ctx context.Context, docID string) *firestore.DocumentRef {
	return d.permissionCollection(ctx).Doc(docID)
}

func (d *PermissionFirestoreDAL) Get(ctx context.Context, permissionID string) (*domain.Permission, error) {
	if permissionID == "" {
		return nil, ErrMissingPermissionID
	}

	permissionRef := d.getPermissionRef(ctx, permissionID)

	docSnap, err := d.documentsHandler.Get(ctx, permissionRef)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrPermissionNotFound
		}

		return nil, err
	}

	var permission domain.Permission

	if err := docSnap.DataTo(&permission); err != nil {
		return nil, err
	}

	permission.ID = permissionID

	return &permission, nil
}
