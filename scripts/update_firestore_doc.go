package scripts

import (
	"errors"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
)

type UpdateFirestoreFieldInput struct {
	Project         string      `json:"project"`
	CollectionPath  string      `json:"collection_path"`
	CollectionGroup bool        `json:"collection_group"`
	Field           string      `json:"field"`
	Value           interface{} `json:"value"`
}

// UpdateFirestoreField updates a firestore document field with the value sent.
//
// Example: payload to update the "draft" field of the "report" document:
//
//	{
//	    "project": "doitintl-cmp-dev",
//	    "collection_path": "dashboards/google-cloud-reports/savedReports",
//	    "collection_group": false,
//	    "field": "draft",
//	    "value": false
//	}
//
// will update all the report documents in the dev project of the dashboards/google-cloud-reports/savedReports collection.
func UpdateFirestoreField(ctx *gin.Context) []error {
	var params UpdateFirestoreFieldInput
	if err := ctx.ShouldBindJSON(&params); err != nil {
		return []error{err}
	}

	if params.Project == "" || params.CollectionPath == "" || params.Field == "" {
		err := errors.New("invalid input parameters")
		return []error{err}
	}

	// Trim leading "/"" from paths if they exist
	params.CollectionPath = strings.TrimPrefix(params.CollectionPath, "/")

	fs, err := firestore.NewClient(ctx, params.Project)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	var q firestore.Query

	if params.CollectionGroup {
		q = fs.CollectionGroup(params.CollectionPath).Select(params.Field)
	} else {
		q = fs.Collection(params.CollectionPath).Select(params.Field)
	}

	snapshots, err := q.Documents(ctx).GetAll()
	if err != nil {
		return []error{err}
	}

	batch := fb.NewAutomaticWriteBatch(fs, 250)

	for _, snapshot := range snapshots {
		batch.Update(snapshot.Ref, []firestore.Update{{Path: params.Field, Value: params.Value}})
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		return errs
	}

	return nil
}
