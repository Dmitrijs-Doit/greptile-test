package scripts

import (
	"time"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	invoicing "github.com/doitintl/hello/scheduled-tasks/invoicing"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
)

type ForceSnapshotBillingTablesRequest struct {
	InvoicingMonth string   `json:"invoicingMonth" binding:"required"`
	CustomerIDs    []string `json:"customerIDs" binding:"required"`
}

func ForceSnapshotBillingTables(ctx *gin.Context) []error {
	l := logger.FromContext(ctx)
	monthFmt := "2006-01"

	l.Info("ForceSnapshotBillingTables started")

	var req ForceSnapshotBillingTablesRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		return []error{err}
	}

	log, _ := logger.NewLogging(ctx)
	conn, _ := connection.NewConnection(ctx, log)

	// Create a new BillingData service
	service := invoicing.NewBillingDataService(conn)

	// Parse the invoiceMonth string
	invoiceMonth, err := time.Parse(monthFmt, req.InvoicingMonth)

	if err != nil {
		l.Errorf("Invalid invoice month format: %v", err)
	}

	// Snapshot the billing tables for each customer
	errors := []error{}

	for _, customerID := range req.CustomerIDs {
		err := service.SnapshotCustomerBillingTable(ctx, customerID, invoiceMonth)

		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		l.Errorf("ForceSnapshotBillingTables failed with errors: %v", errors)

		return errors
	}

	l.Info("ForceSnapshotBillingTables finished successfully")

	return nil
}
