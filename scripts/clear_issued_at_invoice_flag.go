package scripts

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

// ClearInvoiceIssuedAtBasedOnTimestampRange
// Inputs start time, end time,  dry mode, expected updateCount
// Output looks up firestore database collectionGroup 'entityInvoices' (/billing/invoicing/invoicingMonths/2024-01/monthInvoices/<<entity_id>>/entityInvoices)
// within the time range supplied and clears out (sets to nil), timestamp value in 'issuedAt' attribute

// example Http POST to
// http://<<server>>/scripts/clearIssuedAtInvoiceFlag?startDayHour=2024-02-05_09_00&endDayHour=2024-02-05_13_00&dry=true&updateCount=1

func ClearInvoiceIssuedAtBasedOnTimestampRange(ctx *gin.Context) []error {
	const issuedAt = "issuedAt"
	l := logger.FromContext(ctx)
	dry := true

	if ctx.Query("dry") == "false" {
		dry = false
	}

	startTimeParam := ctx.Query("startDayHour")
	endTimeParam := ctx.Query("endDayHour")
	updateCountParam := ctx.Query("updateCount")

	startTime, err := time.Parse("2006-01-02_15_04", startTimeParam)
	if err != nil {
		return []error{errors.New("invalid start time, please provide start time in format '2006-01-02_15_04")}
	}

	endTime, err := time.Parse("2006-01-02_15_04", endTimeParam)
	if err != nil {
		return []error{errors.New("invalid end time, please provide end time in format '2006-01-02_15_04")}
	}

	updateCount, err := strconv.Atoi(updateCountParam)
	if err != nil {
		return []error{errors.New("could not parse updateCountParam")}
	}

	l.Infof("runmode=%v, Finding invoices between startTime=%v and endTime=%v", dry, startTime, endTime)

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{fmt.Errorf("error instantiating firestore client: %s", err.Error())}
	}

	defer fs.Close()

	entityInvoices, err := fs.CollectionGroup("entityInvoices").
		Where("type", "==", "amazon-web-services").
		Where(issuedAt, ">=", startTime).Where(issuedAt, "<=", endTime).
		Documents(ctx).GetAll()
	if err != nil {
		return []error{fmt.Errorf("error querying entityInvoices data: %s", err.Error())}
	}

	if len(entityInvoices) != updateCount {
		l.Errorf("dryMode=%v: Updating entityInvoice failed: expected updateCount %d, found updateCount %d", dry, updateCount, len(entityInvoices))
		return []error{fmt.Errorf("updating entityInvoice failed: expected updateCount %d", updateCount)}
	}

	batch := fb.NewAutomaticWriteBatch(fs, 250)

	count := 0

	for _, eachInvoice := range entityInvoices {
		l.Infof("dryMode=%v: Updating entityInvoice: %v, setting issuedAt to nil", dry, eachInvoice.Ref.Path)

		if dry {
			continue
		}

		batch.Update(eachInvoice.Ref, []firestore.Update{{
			Path:  issuedAt,
			Value: nil,
		}})

		count++
	}

	l.Infof("committing updates in %d documents", count)

	if dry {
		return nil
	}

	if errs := batch.Commit(ctx); err != nil {
		return errs
	}

	return nil
}
