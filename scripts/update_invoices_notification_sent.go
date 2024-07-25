package scripts

import (
	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func UpdateOldInvoicesNotificationSent(ctx *gin.Context) []error {
	l := logger.FromContext(ctx)

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		l.Errorf("Update invoices - firestore client error: %s", err.Error())
		return []error{err}
	}

	batch := fs.Batch()

	nonUpdatedInvoices, err := fs.Collection("invoices").
		Where("CANCELED", "==", false).
		Where("notification.sent", "==", false).
		Where("PAID", "==", true).
		Documents(ctx).GetAll()

	if err != nil {
		l.Errorf("Update invoices - error getting non-updated invoices: %s", err.Error())
		return []error{err}
	}

	for _, invoice := range nonUpdatedInvoices {
		batch.Update(invoice.Ref, []firestore.Update{
			{FieldPath: []string{"notification", "sent"}, Value: true},
		})
	}

	if _, err := batch.Commit(ctx); err != nil {
		l.Errorf("Update invoices - error committing updates: %s", err.Error())
		return []error{err}
	}

	l.Infof("%d invoices 'sent' flags updated", len(nonUpdatedInvoices))

	return []error{}
}
