package scripts

import (
	"errors"
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/errorreporting"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
)

type RemoveEarlyAccessFeatureInput struct {
	Value string `json:"value"`
}

// RemoveEarlyAccessFeature removes a deprecated early access feature from "cutomers"
func RemoveEarlyAccessFeature(ctx *gin.Context) []error {
	var params RemoveEarlyAccessFeatureInput
	if err := ctx.ShouldBindJSON(&params); err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	if params.Value == "" {
		err := errors.New("missing early access feature to remove")
		errorreporting.AbortWithErrorReport(ctx, http.StatusBadRequest, err)

		return []error{err}
	}

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	batch := fb.NewAutomaticWriteBatch(fs, 250)

	docSnaps, err := fs.Collection("customers").
		Where("earlyAccessFeatures", "array-contains", params.Value).
		Documents(ctx).
		GetAll()
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	if len(docSnaps) > 0 {
		for _, docSnap := range docSnaps {
			batch.Update(docSnap.Ref, []firestore.Update{
				{FieldPath: []string{"earlyAccessFeatures"}, Value: firestore.ArrayRemove(params.Value)},
			})
		}

		if errs := batch.Commit(ctx); len(errs) > 0 {
			return errs
		}
	}

	return nil
}
