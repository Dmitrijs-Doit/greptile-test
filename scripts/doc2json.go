package scripts

import (
	"errors"
	"fmt"

	"encoding/json"
	"net/http"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/errorreporting"
)

type Doc2JsonInput struct {
	SrcProject string `json:"src_project"`
	SrcPath    string `json:"src_path"`
}

func Doc2Json(ctx *gin.Context) []error {
	var params Doc2JsonInput
	if err := ctx.ShouldBindJSON(&params); err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	if params.SrcProject == "" ||
		!strings.Contains(params.SrcPath, "/") {
		err := errors.New("invalid input parameters")
		errorreporting.AbortWithErrorReport(ctx, http.StatusBadRequest, err)

		return []error{err}
	}

	// Trim leading "/"" from paths if they exist
	params.SrcPath = strings.TrimPrefix(params.SrcPath, "/")

	srcFs, err := firestore.NewClient(ctx, params.SrcProject)
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}
	defer srcFs.Close()

	docSnap, err := srcFs.Doc(params.SrcPath).Get(ctx)
	if err != nil || docSnap == nil || !docSnap.Exists() {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	var Obj interface{}
	if err := docSnap.DataTo(&Obj); err != nil {
		return []error{err}
	}

	jsonBytes, err := json.Marshal(Obj)
	if err := docSnap.DataTo(&Obj); err != nil {
		return []error{err}
	}

	fmt.Printf("\n######### %s #########\n%s\n-----------\n\n", params.SrcPath, jsonBytes)

	return nil
}
