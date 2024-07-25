package scripts

import (
	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

// Removes tenantID propery from the auth object in customer collection
// Example...
//
//	{
//		"project": "doitintl-kraken",
//		"customer": "UHUu4R4ORhErDvamyrFH",
//		"auth": {
//			tentantID: "t-UHUu4R4ORhErDvamyrFH"
//		}
//	}
func ModifyCustomerAuth(ctx *gin.Context) []error {
	l := logger.FromContext(ctx)

	l.Infof("ModifyCustomerAuth")

	fs := common.GetFirestoreClient(ctx)
	wb := fb.NewAutomaticWriteBatch(fs, 500)
	docSnaps, err := fs.Collection("customers").Documents(ctx).GetAll()
	l.Infof("%v customers were found", len(docSnaps))

	if err != nil {
		return []error{err}
	}

	for _, docSnap := range docSnaps {
		update := []firestore.Update{{
			Path:  "auth.tenantId",
			Value: firestore.Delete,
		}}
		wb.Update(docSnap.Ref, update)
	}

	if errs := wb.Commit(ctx); len(errs) > 0 {
		for _, err := range errs {
			l.Errorf("wb.Commit err: %v", err)
		}

		return errs
	}

	return nil
}
