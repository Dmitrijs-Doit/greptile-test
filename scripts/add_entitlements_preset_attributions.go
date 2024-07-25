package scripts

import (
	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
)

func AddEntitlementsToPresetAttributions(ctx *gin.Context) []error {
	var params Params
	if err := ctx.ShouldBindJSON(&params); err != nil {
		return []error{err}
	}

	presetAttributions := map[string]string{
		"VI4do9dMowbl4XSMKyTA": eks,
		"fpp0awvNqlfPQVTebKys": eks,
	}

	fs, err := firestore.NewClient(ctx, params.ProjectID)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	batch := fb.NewAutomaticWriteBatch(fs, 500)

	for attributionID, entitlement := range presetAttributions {
		ref := fs.Collection("dashboards").Doc("google-cloud-reports").Collection("attributions").Doc(attributionID)

		batch.Set(ref, map[string]interface{}{
			"entitlements": []string{entitlement},
		}, firestore.MergeAll)
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		return errs
	}

	return nil
}
