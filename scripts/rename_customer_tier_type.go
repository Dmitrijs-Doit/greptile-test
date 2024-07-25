package scripts

import (
	"errors"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
)

type RenameCustomerTierRequest struct {
	CollectionPath string `json:"collection_path"`
	Field          string `json:"field"`
	NewName        string `json:"newName"`
	Commit         bool   `json:"commit"`
}

// RenameCustomerTierType updates a customers tier field, eg rename tiers.console -> tiers.navigator
//
// Example: payload to rename the "tier.console" field of the "customer" document to "tier.navigator":
//
//	{
//	    "collection_path": "customers",
//	    "field": "tiers.console",
//	    "newName": "tiers.navigator",
//	    "commit": false
//	}
//
// will update all the customer documents in the dev project which has a tier field defined
func RenameCustomerTierType(ctx *gin.Context) []error {
	var params RenameCustomerTierRequest
	if err := ctx.ShouldBindJSON(&params); err != nil {
		return []error{err}
	}

	if params.Field == "" || params.CollectionPath == "" || params.NewName == "" {
		err := errors.New("invalid input parameters")
		return []error{err}
	}

	if !params.Commit {
		fmt.Println("Not committing changed")
	}

	// Trim leading "/"" from paths if they exist
	params.CollectionPath = strings.TrimPrefix(params.CollectionPath, "/")

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	q := fs.Collection(params.CollectionPath).Where("tiers", "!=", "")

	snapshots, err := q.Documents(ctx).GetAll()
	if err != nil {
		return []error{err}
	}

	fmt.Println("Number of customers found", len(snapshots))

	batch := fb.NewAutomaticWriteBatch(fs, 250)

	for _, snapshot := range snapshots {
		data, err := snapshot.DataAt(params.Field)
		if err != nil {
			return []error{err}
		}

		if !params.Commit {
			fmt.Printf("CustomerID %s: Setting field %s to %+v, deleting field %s\n",
				snapshot.Ref.ID,
				params.NewName,
				data,
				params.Field,
			)

			continue
		}

		batch.Update(snapshot.Ref, []firestore.Update{
			{Path: params.NewName, Value: data},
			{Path: params.Field, Value: firestore.Delete},
		})
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		return errs
	}

	return nil
}
