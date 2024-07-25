package scripts

import (
	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

// Add delete field set to false on all perks
// no Payload
func AddDeleteFieldOnPerks(ctx *gin.Context) []error {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)
	wb := fb.NewAutomaticWriteBatch(fs, 500)

	docSnaps, err := fs.Collection("perks").Documents(ctx).GetAll()

	if err != nil {
		return []error{err}
	}

	for _, docSnap := range docSnaps {
		l.Infof("added delete field to perk %s", docSnap.Ref.ID)

		update := []firestore.Update{{
			Path:  "fields.deleted",
			Value: false,
		}}
		wb.Update(docSnap.Ref, update)
	}

	if errs := wb.Commit(ctx); len(errs) > 0 {
		for _, err := range errs {
			l.Errorf("Commit error: %v", err)
		}

		return errs
	}

	return nil
}
