package scripts

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/errorreporting"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
)

type MigrateCollectionFieldInput struct {
	ProjectID      string      `json:"project_id"`
	CollectionPath string      `json:"collection_path"`
	DeleteOldField bool        `json:"delete_old_field"`
	OldField       string      `json:"old_field"`
	OldValue       interface{} `json:"old_value"`
	NewField       string      `json:"new_field"`
	NewValue       interface{} `json:"new_value"`
	Limit          int         `json:"limit"`
}

// MigrateCollectionField migrate a collection's field with one value to another
//
// Example: payload to migrate "isPublic" field of the "report" document to the new "public" field:
// {
//     "project_id": "doitintl-cmp-dev",
//     "collection_path": "dashboards/google-cloud-reports/savedReports",
//     "limit": 1,
//     "delete_old_field": false,
//     "old_field": "isPublic",
//     "old_value": true,
//     "new_field": "public",
//     "new_value": "viewer"
// }

func MigrateCollectionField(ctx *gin.Context) []error {
	var params MigrateCollectionFieldInput
	if err := ctx.ShouldBindJSON(&params); err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	if params.ProjectID == "" || params.CollectionPath == "" || params.OldField == "" || params.NewField == "" {
		err := errors.New("invalid input parameters")
		errorreporting.AbortWithErrorReport(ctx, http.StatusBadRequest, err)

		return []error{err}
	}

	if params.OldField == params.NewField && params.DeleteOldField {
		err := errors.New("cannot delete old field if it has the same key as new")
		errorreporting.AbortWithErrorReport(ctx, http.StatusBadRequest, err)

		return []error{err}
	}

	// Trim leading "/"" from paths if they exist
	params.CollectionPath = strings.TrimPrefix(params.CollectionPath, "/")

	fs, err := firestore.NewClient(ctx, params.ProjectID)
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}
	defer fs.Close()

	query := fs.Collection(params.CollectionPath).Query

	// Comment out the next line if you don't want to filter the docs based on "old_value"
	query = query.Where(params.OldField, "==", params.OldValue)

	// If you want to run on just a few docs to test it out before running on the full collection
	if params.Limit > 0 {
		query = query.Limit(params.Limit)
	}

	docSnaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	batch := fb.NewAutomaticWriteBatch(fs, 250)

	for _, docSnap := range docSnaps {
		if params.Limit > 0 {
			// When running on a fixed amount of docs, print the doc paths that were migrated
			fmt.Println(docSnap.Ref.Path)
		}

		newValue, err := newFieldValue(params.NewValue, docSnap)
		if err != nil {
			errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
			return []error{err}
		}

		updates := []firestore.Update{
			{Path: params.NewField, Value: newValue},
		}

		if params.DeleteOldField {
			updates = append(updates, firestore.Update{Path: params.OldField, Value: firestore.Delete})
		}

		batch.Update(docSnap.Ref, updates)
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, errs[0])
		return errs
	}

	return nil
}

func newFieldValue(defaultValue interface{}, docSnap *firestore.DocumentSnapshot) (interface{}, error) {
	/**
		Advanced new field value example:
		Gets a new value based on the existing docSnap instead of providing a fixed value in params
		Uncomment the code below and edit per your requirements
	**/
	// var attr attribution.Attribution
	// if err := docSnap.DataTo(&attr); err != nil {
	// 	return nil, err
	// }
	// newValue := []report.CloudAnalyticsCollaborator{
	// 	{Email: attr.Owner, Role: collab.CollaboratorRoleOwner},
	// }
	// return newValue, nil
	return defaultValue, nil
}
