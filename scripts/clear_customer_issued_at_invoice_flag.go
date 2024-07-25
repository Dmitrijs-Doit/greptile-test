package scripts

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/errorreporting"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

type InvoiceRequestParam struct {
	StartDayHour string   `json:"startDayHour"`
	EndDayHour   string   `json:"endDayHour"`
	Dry          bool     `json:"dry"`
	UpdateCount  int      `json:"updateCount"`
	CustomerIDs  []string `json:"customerIDs"`
}

// ClearInvoiceIssuedAtForSpecificCustomers
// Inputs start time, end time,  dry mode, expected updateCount, list of customers
// Output looks up firestore database collectionGroup 'entityInvoices' (/billing/invoicing/invoicingMonths/2024-01/monthInvoices/<<entity_id>>/entityInvoices)
// within the time range supplied and for the customers supplied and clears out (sets to nil), timestamp value in 'issuedAt' attribute

// example Http POST to
// http://localhost:8082/scripts/clearSpecificEntIssuedAtInvFlag
// {  "startDayHour": "2024-02-06_09_00", "endDayHour": "2024-02-08_17_00", "dry": true, "updateCount": 1, "customerIDs" : ["XXX", "YYY"] }

func ClearInvoiceIssuedAtForSpecificCustomers(ctx *gin.Context) []error {
	const issuedAt = "issuedAt"

	l := logger.FromContext(ctx)

	var params InvoiceRequestParam
	if err := ctx.ShouldBindJSON(&params); err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	l.Infof("processing clear issuedAt invoice flag script with params %v", params)

	dry := true
	if params.Dry == false {
		dry = false
	}

	startTimeParam := params.StartDayHour
	endTimeParam := params.EndDayHour
	customerIDs := params.CustomerIDs
	updateCount := params.UpdateCount

	startTime, err := time.Parse("2006-01-02_15_04", startTimeParam)
	if err != nil {
		return []error{errors.New("invalid start time, please provide start time in format '2006-01-02_15_04")}
	}

	endTime, err := time.Parse("2006-01-02_15_04", endTimeParam)
	if err != nil {
		return []error{errors.New("invalid end time, please provide end time in format '2006-01-02_15_04")}
	}

	l.Infof("runmode=%v, Finding invoices between startTime=%v and endTime=%v", dry, startTime, endTime)

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{fmt.Errorf("error instantiating firestore client: %s", err.Error())}
	}

	defer fs.Close()

	var customerIDCollectionRefs []*firestore.DocumentRef
	for _, eachCustomer := range customerIDs {
		customerIDCollectionRefs = append(customerIDCollectionRefs, fs.Collection("customers").Doc(eachCustomer))
	}

	customerDocList, err := fs.GetAll(ctx, customerIDCollectionRefs)
	if err != nil {
		return []error{fmt.Errorf("error querying customers collection: %s", err.Error())}
	}

	var entityIDs []string

	for _, each := range customerDocList {
		entityID, err := each.DataAt("entities")
		if err != nil {
			return []error{fmt.Errorf("error fetching entities data for %s: %s", each.Ref.ID, err.Error())}
		}

		entityIDList, ok := entityID.([]interface{})
		if !ok {
			return []error{fmt.Errorf("error fetching entities data for %s: %s", each.Ref.ID, err.Error())}
		}

		for _, eachEntityID := range entityIDList {
			entityIDs = append(entityIDs, eachEntityID.(*firestore.DocumentRef).ID)
		}
	}

	invoicesInTimeRange, err := fs.CollectionGroup("entityInvoices").
		Where("type", "==", "amazon-web-services").
		Where(issuedAt, ">=", startTime).Where(issuedAt, "<=", endTime).
		Documents(ctx).GetAll()
	if err != nil {
		return []error{fmt.Errorf("error querying entityInvoices data: %s", err.Error())}
	}

	var entityInvoices []*firestore.DocumentSnapshot

	for _, each := range invoicesInTimeRange {
		if slice.Contains(entityIDs, each.Ref.Parent.Parent.ID) {
			entityInvoices = append(entityInvoices, each)
			l.Infof("dryMode=%v: Identified entityInvoice: %v for update", dry, each.Ref.Path)
		}
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

	l.Infof("dryMode=%v: Committing updates in %d documents", dry, count)

	if dry {
		return nil
	}

	if errs := batch.Commit(ctx); err != nil {
		return errs
	}

	return nil
}
