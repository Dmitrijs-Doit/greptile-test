package scripts

import (
	"errors"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/googlecloud"
)

type reassignCustomerGcpProjectsInput struct {
	ProjectID        string `json:"project_id"`
	BillingAccountID string `json:"billing_account_id"`
	Limit            int    `json:"limit"`
}

var (
	startAfter           *firestore.DocumentSnapshot
	lastBillingAccountID string
)

func reassignCustomerGcpProjects(ctx *gin.Context) []error {
	var params reassignCustomerGcpProjectsInput

	if err := ctx.ShouldBindJSON(&params); err != nil {
		return []error{err}
	}

	if params.ProjectID == "" || params.BillingAccountID == "" {
		err := errors.New("bad request")
		return []error{err}
	}

	if lastBillingAccountID != "" && lastBillingAccountID != params.BillingAccountID {
		startAfter = nil
	}

	lastBillingAccountID = params.BillingAccountID

	fs, err := firestore.NewClient(ctx, params.ProjectID)
	if err != nil {
		return []error{err}
	}

	baDocSnap, err := fs.Collection("assets").Doc(fmt.Sprintf("google-cloud-%s", params.BillingAccountID)).Get(ctx)
	if err != nil {
		return []error{err}
	}

	var ba googlecloud.Asset

	if err := baDocSnap.DataTo(&ba); err != nil {
		return []error{err}
	}

	docSnaps, err := fs.Collection("assets").
		Where("type", "==", common.Assets.GoogleCloudProject).
		Where("properties.billingAccountId", "==", params.BillingAccountID).
		Select().
		StartAfter(startAfter).
		Limit(params.Limit).
		Documents(ctx).
		GetAll()
	if err != nil {
		return []error{err}
	}

	fmt.Println("found ", len(docSnaps), " projects")

	if len(docSnaps) > 0 {
		bw := fs.BulkWriter(ctx)
		startAfter = docSnaps[len(docSnaps)-1]

		for _, docSnap := range docSnaps {
			if _, err := bw.Update(docSnap.Ref, []firestore.Update{
				{FieldPath: []string{"customer"}, Value: ba.Customer},
				{FieldPath: []string{"contract"}, Value: ba.Contract},
				{FieldPath: []string{"entity"}, Value: ba.Entity},
				{FieldPath: []string{"bucket"}, Value: ba.Bucket},
			}); err != nil {
				return []error{err}
			}

			asRef := fs.Collection("assetSettings").Doc(docSnap.Ref.ID)

			if _, err := bw.Update(asRef, []firestore.Update{
				{FieldPath: []string{"customer"}, Value: ba.Customer},
				{FieldPath: []string{"contract"}, Value: ba.Contract},
				{FieldPath: []string{"entity"}, Value: ba.Entity},
				{FieldPath: []string{"bucket"}, Value: ba.Bucket},
			}); err != nil {
				return []error{err}
			}
		}

		bw.End()
	}

	return nil
}
