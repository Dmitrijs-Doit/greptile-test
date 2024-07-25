package scripts

import (
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/errorreporting"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold"
	"github.com/gin-gonic/gin"
)

// FinalizeInvoiceAdjustments sets the finalize flag to true on last month's invoice adjustments
func FinalizeInvoiceAdjustments(ctx *gin.Context) []error {
	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	lastMonth := time.Now().UTC().AddDate(0, -1, 0)

	batch := fb.NewAutomaticWriteBatch(fs, 250)

	invoiceDocs, err := fs.CollectionGroup("customerInvoiceAdjustments").
		Where("details", "==", flexsaveresold.FlexSaveInvoiceDetails).
		Where("invoiceMonths", "array-contains", time.Date(lastMonth.Year(), lastMonth.Month(), 1, 0, 0, 0, 0, time.UTC)).
		Documents(ctx).GetAll()
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	for _, invoiceDoc := range invoiceDocs {
		batch.Update(invoiceDoc.Ref, []firestore.Update{
			{
				FieldPath: []string{"finalized"},
				Value:     true,
			}})
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		return errs
	}

	return nil
}
