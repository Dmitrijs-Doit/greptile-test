package scripts

import (
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/errorreporting"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type RequestJSON struct {
	InvoiceMonth string `json:"invoiceMonth"`
}

// DeleteDuplicateInvoiceAdjustments deletes the stale invoices left during renaming flexSave to flexsave, this should only be relevant for Apr-2022
func DeleteDuplicateInvoiceAdjustments(ctx *gin.Context) []error {
	l := logger.FromContext(ctx)

	var request RequestJSON
	if err := ctx.BindJSON(&request); err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	requestMonth, err := time.Parse("2006-01", request.InvoiceMonth)
	if err != nil {
		return []error{fmt.Errorf("please provide invoice year-month in format YYYY-MM eg: 2022-04 for april-2022")}
	}

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	batch := fb.NewAutomaticWriteBatch(fs, 100)

	customersSnaps, err := fs.Collection("customers").Where("assets", common.ArrayContains, common.Assets.AmazonWebServices).Documents(ctx).GetAll()
	if err != nil {
		return []error{err}
	}

	invoiceMonth := time.Date(requestMonth.Year(), requestMonth.Month(), 5, 0, 0, 0, 0, time.UTC)

	for _, customersSnap := range customersSnaps {
		invoiceAdjustment := customersSnap.Ref.Collection("customerInvoiceAdjustments")

		docs1, err := invoiceAdjustment.Where("type", "==", common.Assets.AmazonWebServices).
			Where("invoiceMonths", common.ArrayContains, time.Date(invoiceMonth.Year(), invoiceMonth.Month(), 1, 0, 0, 0, 0, time.UTC)).
			Where("description", "==", "FlexSave Savings").
			Where("details", "==", "DoiT FlexSave Savings").
			Documents(ctx).GetAll()
		if err != nil {
			l.Errorf("error occurred while process invoiceAdjustment deletion for customer %v", customersSnap.Ref.ID)
			continue
		}

		docs2, err := invoiceAdjustment.Where("type", "==", common.Assets.AmazonWebServices).
			Where("invoiceMonths", common.ArrayContains, time.Date(invoiceMonth.Year(), invoiceMonth.Month(), 1, 0, 0, 0, 0, time.UTC)).
			Where("description", "==", "FlexSave Savings").
			Where("details", "==", "DoiT FlexSave Savings | DA").
			Documents(ctx).GetAll()
		if err != nil {
			l.Errorf("error occurred while process invoiceAdjustment deletion for customer %v", customersSnap.Ref.ID)
			continue
		}

		docs := append(docs1, docs2...)

		for _, doc := range docs {
			l.Infof("deleting invoiceAdjustment %v for customer %v", doc.Ref.ID, customersSnap.Ref.ID)
			batch.Delete(doc.Ref)
		}
	}

	l.Infof("committing all deletes")

	errors := batch.Commit(ctx)
	if len(errors) > 0 {
		l.Errorf("batch commit failed to update some deletes, please check logs and manually verify, error %v", errors)
	}

	l.Infof("completed")

	return nil
}
