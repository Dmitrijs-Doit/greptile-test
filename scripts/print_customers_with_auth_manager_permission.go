package scripts

import (
	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func PrintCustomersWithAuthManagerPermission(ctx *gin.Context) []error {
	l := logger.FromContext(ctx)

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}

	defer fs.Close()

	authManagerPermissionRef := fs.Collection("permissions").Doc("xykbIpbZLOmZnF8fpiZg")

	docSnaps, err := fs.Collection("roles").
		Where("permissions", "array-contains", authManagerPermissionRef).
		Documents(ctx).
		GetAll()

	if err != nil {
		return []error{err}
	}

	for _, docSnap := range docSnaps {
		l.Infof("roleID: %s", docSnap.Ref.ID)

		customerRef, ok := docSnap.Data()["customer"].(*firestore.DocumentRef)
		if ok {
			l.Infof("customerID: %s", customerRef.ID)
		}
	}

	return nil
}
