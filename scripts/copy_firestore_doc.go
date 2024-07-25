package scripts

import (
	"errors"
	"net/http"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/errorreporting"
)

type CopyFirestoreDocInput struct {
	SrcProject            string `json:"src_project"`
	SrcPath               string `json:"src_path"`
	DstProject            string `json:"dst_project"`
	DstPath               string `json:"dst_path"`
	IncludeSubCollections bool   `json:"include_sub_collections"`
}

// CopyFirestoreDoc copies a firestore document from one project to another.
//
// Example: payload to copy from one project to another:
//
//	{
//	    "src_project": "src-project-id"
//	    "src_path": "assets/amazon-web-services-000439338805",
//	    "dst_project": "dst-project-id"
//	    "dst_path": "assets/amazon-web-services-111111111111",
//	}
//
// will copy the document at "/assets/amazon-web-services-000439338805" on the <src_project>
// to the specified <dst_project> project to the path of <dst_path>
// Use the same document ID as in path if you do not want to rename the document.
// Use the same project ID if you want to copy within the same project
//
// WARNING: Document reference fields need to be copied manually.
// Copying them will NOT work properly since they contain metadata from the source project
func CopyFirestoreDoc(ctx *gin.Context) []error {
	var params CopyFirestoreDocInput
	if err := ctx.ShouldBindJSON(&params); err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	if params.SrcProject == "" || params.DstProject == "" ||
		!strings.Contains(params.SrcPath, "/") || !strings.Contains(params.DstPath, "/") {
		err := errors.New("invalid input parameters")
		errorreporting.AbortWithErrorReport(ctx, http.StatusBadRequest, err)

		return []error{err}
	}

	// Trim leading "/"" from paths if they exist
	params.SrcPath = strings.TrimPrefix(params.SrcPath, "/")
	params.DstPath = strings.TrimPrefix(params.DstPath, "/")

	srcFs, err := firestore.NewClient(ctx, params.SrcProject)
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}
	defer srcFs.Close()

	dstFs, err := firestore.NewClient(ctx, params.DstProject)
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}
	defer dstFs.Close()

	docSnap, err := srcFs.Doc(params.SrcPath).Get(ctx)
	if err != nil || docSnap == nil || !docSnap.Exists() {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	if _, err := dstFs.Doc(params.DstPath).Set(ctx, docSnap.Data()); err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	if params.IncludeSubCollections {
		subCollections, err := srcFs.Doc(params.SrcPath).Collections(ctx).GetAll()
		if err != nil {
			return []error{errors.New("no sub-collection found")}
		}

		for _, subCollection := range subCollections {
			docSnaps, err := srcFs.Doc(params.SrcPath).Collection(subCollection.ID).Documents(ctx).GetAll()
			if err != nil {
				return []error{err}
			}

			for _, docSnap := range docSnaps {
				data := docSnap.Data()
				if _, err := dstFs.Doc(params.DstPath).Collection(subCollection.ID).Doc(docSnap.Ref.ID).Set(ctx, data); err != nil {
					errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
					return []error{err}
				}
			}
		}
	}

	return nil
}
