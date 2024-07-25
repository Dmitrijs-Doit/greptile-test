package scripts

import (
	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/gin-gonic/gin"
)

func SetAllGCPSaaSCloudConnectNotified(ctx *gin.Context) []error {
	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}

	docSnaps, err := fs.CollectionGroup("cloudConnect").
		Where("type", "==", "google-cloud-standalone").
		Documents(ctx).GetAll()
	if err != nil {
		return []error{err}
	}

	errs := []error{}

	batch := fb.NewAutomaticWriteBatch(fs, 100)

	for _, docSnap := range docSnaps {
		batch.Update(docSnap.Ref, []firestore.Update{
			{Path: "notified", Value: true},
		})
	}

	if commitErrs := batch.Commit(ctx); len(errs) > 0 {
		errs = append(errs, commitErrs...)
		return errs
	}

	return errs
}
