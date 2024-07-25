package invoices

import (
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/priority"
	priorityDomain "github.com/doitintl/hello/scheduled-tasks/priority/domain"
	"github.com/doitintl/hello/scheduled-tasks/slice"
	stripeConsts "github.com/doitintl/hello/scheduled-tasks/stripe/consts"
)

type CustomerTask struct {
	EntityID        string                        `json:"entity_id"`
	PriorityID      string                        `json:"priority_id"`
	PriorityCompany string                        `json:"priority_company"`
	OpenInvoices    []*priorityDomain.OpenInvoice `json:"open_invoices"`
	Timestamp       time.Time                     `json:"timestamp"`
}

const pdfDataURLPrefix = "data:application/pdf;base64,"

var (
	baseMergeOptions = []firestore.FieldPath{
		{"IVNUM"},
		{"CUSTNAME"},
		{"CDES"},
		{"IVDATE_STRING"},
		{"PAYDATE_STRING"},
		{"CODE"},
		{"QPRICE"},
		{"VAT"},
		{"TOTPRICE"},
		{"IVTYPE"},
		{"DETAILS"},
		{"COMPANY"},
		{"STATDES"},
		{"ROTL_CMP_NUMBER"},
		{"EXTFILES"},
		{"CANCELED"},
		{"SYMBOL"},
		{"USDEXCH"},
		{"USDTOTAL"},
		{"DEBIT"},
		{"IVDATE"},
		{"PAYDATE"},
		{"ESTPAYDATE"},
		{"PAID"},
		{"PRODUCTS"},
		{"INVOICEITEMS"},
		{"metadata"},
		{"customer"},
		{"entity"},
	}

	invoiceMergeOptions = firestore.Merge(baseMergeOptions...)
	ccFeesMergeOptions  = firestore.Merge(append(baseMergeOptions, firestore.FieldPath{"stripeLocked"})...)
)

func MainHandler(ctx *gin.Context) {
	l := logger.FromContext(ctx)

	now := time.Now().UTC()

	fs := common.GetFirestoreClient(ctx)

	openInvoices := make(map[string][]*priorityDomain.OpenInvoice)

	for _, company := range priority.Companies {
		companyOpenInvoices, err := getOpenInvoices(ctx, string(company), "")
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		for _, openInvoice := range companyOpenInvoices {
			// Filter out receipts from the results
			if openInvoice.ID == "" || strings.HasPrefix(openInvoice.ID, "RC") {
				continue
			}

			k := fmt.Sprintf("%s-%s", company, openInvoice.PriorityID)
			if _, prs := openInvoices[k]; !prs {
				openInvoices[k] = make([]*priorityDomain.OpenInvoice, 0)
			}

			openInvoices[k] = append(openInvoices[k], openInvoice)
		}
	}

	iter := fs.Collection("entities").Select("priorityId", "priorityCompany").Documents(ctx)
	defer iter.Stop()

	for {
		docSnap, err := iter.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			l.Errorf("customers error: %v", err)
			ctx.AbortWithError(http.StatusInternalServerError, err)

			return
		}

		var entity common.Entity
		if err := docSnap.DataTo(&entity); err != nil {
			l.Errorf("entity: %v", err)
			continue
		}

		k := fmt.Sprintf("%s-%s", entity.PriorityCompany, entity.PriorityID)

		var entityOpenInvoices []*priorityDomain.OpenInvoice

		if v, prs := openInvoices[k]; prs {
			entityOpenInvoices = v
		}

		t := CustomerTask{
			EntityID:        docSnap.Ref.ID,
			PriorityID:      entity.PriorityID,
			PriorityCompany: entity.PriorityCompany,
			OpenInvoices:    entityOpenInvoices,
			Timestamp:       now,
		}

		taskBody, err := json.Marshal(t)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		config := common.CloudTaskConfig{
			Method:       cloudtaskspb.HttpMethod_POST,
			Path:         "/tasks/invoices",
			Queue:        common.TaskQueueInvoicesSync,
			Body:         taskBody,
			ScheduleTime: nil,
		}

		_, err = common.CreateCloudTask(ctx, &config)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}
}

// func CustomerWorker(ctx *gin.Context, t CustomerTask) {
func CustomerWorker(ctx *gin.Context) {
	l := logger.FromContext(ctx)

	if !fixer.CurrencyHistoricalTimeseriesInitialized {
		l.Info("currency historical timeseries is not available (invoice)")
		ctx.AbortWithStatus(http.StatusBadRequest)

		return
	}

	var t CustomerTask
	if err := ctx.ShouldBindJSON(&t); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	l.Info(t)

	now := t.Timestamp
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	severity20 := today.AddDate(0, 0, -35)
	severity30 := today.AddDate(0, 0, -60)
	severity40 := today.AddDate(0, 0, -90)

	fs := common.GetFirestoreClient(ctx)

	dashboardsRef := fs.Collection("dashboards")
	collectionRef := fs.Collection("collection")
	customerBatch := BatchWithCounter{fs.Batch(), 0}

	entityRef := fs.Collection("entities").Doc(t.EntityID)

	entityDocSnap, err := entityRef.Get(ctx)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	var entity common.Entity
	if err := entityDocSnap.DataTo(&entity); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	averageAccountReceivablesDays := -1

	metadataDocSnap, err := entityRef.Collection("entityMetadata").Doc("account-receivables").Get(ctx)
	if err != nil {
		l.Warning(err)
	} else {
		if v, err := metadataDocSnap.DataAt("avgDays"); err == nil {
			averageAccountReceivablesDays = int(v.(int64))
		}
	}

	customerDocSnap, err := entity.Customer.Get(ctx)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	var customer common.Customer
	if err := customerDocSnap.DataTo(&customer); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	invoices, err := getEntityInvoices(ctx, &entity)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	l.Infof("total invoices: %d", len(invoices))

	sort.Slice(invoices, func(i, j int) bool {
		return invoices[j].DateString < invoices[i].DateString
	})

	overDueInvoices := make([]*MinimalInvoice, 0)
	reminders := map[int][]*MinimalInvoice{
		1: {},
		2: {},
		3: {},
	}

	wireTransferPayer := entity.Payment != nil && (entity.Payment.Type == common.EntityPaymentTypeWireTransfer ||
		entity.Payment.Type == common.EntityPaymentTypeBillCom)

	collectionItem := &CollectionItem{
		Customer:     entity.Customer,
		Entity:       entityRef,
		PriorityName: entity.Name,
		PriorityID:   entity.PriorityID,
		EntityData:   &entity,
		Strategic:    customer.Classification == common.CustomerClassificationStrategic,
		Severity:     0,
		Products:     []string{},
		Totals:       newTotalPerCurrency(),
	}

	metadata := map[string]interface{}{
		"customer": map[string]string{
			"name":          customer.Name,
			"primaryDomain": customer.PrimaryDomain,
		},
		"entity": map[string]string{
			"name": entity.Name,
		},
	}

	for _, fullIV := range invoices {
		fullIV.Customer = entity.Customer
		fullIV.Entity = entityRef
		fullIV.Metadata = metadata
		fullIV.Company = entity.PriorityCompany
		fullIV.Canceled = fullIV.Status == StatusCanceled

		switch fullIV.Type {
		case "A":
			fullIV.InvoiceItems = fullIV.AInvoiceItemsSubForm
			if len(fullIV.AInvoicesSubForm) > 0 {
				fullIV.PayDateString = fullIV.AInvoicesSubForm[0].PayDate
			}
		case "C":
			fullIV.InvoiceItems = fullIV.CInvoiceItemsSubForm
			if len(fullIV.CInvoicesSubForm) > 0 {
				fullIV.PayDateString = fullIV.CInvoicesSubForm[0].PayDate
			}
		case "F":
			fullIV.InvoiceItems = fullIV.FInvoiceItemsSubForm
			if len(fullIV.CInvoicesSubForm) > 0 {
				fullIV.PayDateString = fullIV.CInvoicesSubForm[0].PayDate
			}
		}

		invoiceRef := fs.Collection("invoices").Doc(fmt.Sprintf("%s-%s-%s", fullIV.Company, fullIV.PriorityID, fullIV.ID))

		if fullIV.PayDateString != "" {
			if t, err := time.Parse(dateTimeFormat, fullIV.PayDateString); err != nil {
				l.Warning(err)
			} else {
				fullIV.PayDate = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			}
		} else {
			// All invoices should have paydate
			l.Warningf("invoice missing paydate %s", invoiceRef.ID)
		}

		var openIVDebit float64

		for _, iv := range t.OpenInvoices {
			if fullIV.ID != iv.ID {
				continue
			}

			openIVDebit += iv.Debit
		}

		if math.Abs(openIVDebit) > 1e-2 {
			fullIV.Paid = false
			fullIV.Debit = openIVDebit
		} else {
			fullIV.Paid = true
		}

		if t, err := time.Parse(dateTimeFormat, fullIV.DateString); err != nil {
			l.Warningf("invalid date %s: %v", invoiceRef.ID, err)
		} else {
			fullIV.Date = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			if !fullIV.Paid && averageAccountReceivablesDays > -1 {
				estPayDate := fullIV.Date.AddDate(0, 0, averageAccountReceivablesDays)
				fullIV.EstimatedPayDate = &estPayDate
			}
		}

		convDay := fixer.CurrencyHistoricalTimeseries[fullIV.Date.Year()][fullIV.Date.Format(dateFormat)]
		fullIV.Symbol = fixer.CodeToLabel(fullIV.Currency)
		fullIV.USDExchangeRate = convDay[fullIV.Symbol]
		fullIV.USDTotal = fullIV.Total / fullIV.USDExchangeRate
		debitWeight := fullIV.Debit / fullIV.USDExchangeRate

		if fullIV.ExternalFilesSubForm != nil {
			for _, externalFile := range fullIV.ExternalFilesSubForm {
				if strings.HasPrefix(externalFile.ExtFileName, pdfDataURLPrefix) {
					msg := fmt.Sprintf("%s-%s", invoiceRef.ID, externalFile.UpdateDate)

					hmac, err := common.Sha256HMAC(msg, []byte(priority.Client.StorageSecret))
					if err != nil {
						ctx.AbortWithError(http.StatusInternalServerError, err)
						return
					}

					// Object path in GCS bucket
					objectPath := fmt.Sprintf("invoices-v2/%s/%s.pdf", fullIV.Customer.ID, hmac)
					externalFile.Storage = &objectPath
					object := InvoicesBucket.Object(objectPath)

					// File name when dowloaded in CMP Client
					objectName := fmt.Sprintf("%s.pdf", fullIV.ID)
					externalFile.Key = &objectName

					// URL to invoice in CMP Client
					invoiceURL := fmt.Sprintf("https://%s/customers/%s/invoices/%s/%s",
						common.Domain,
						fullIV.Customer.ID,
						fullIV.Entity.ID,
						fullIV.ID,
					)
					externalFile.URL = &invoiceURL

					if _, err := object.Attrs(ctx); err == storage.ErrObjectNotExist {
						// The object does not exist, upload it to GCS now
						// Create gzip Writer and set object attributes
						objWriter := object.NewWriter(ctx)
						objWriter.ObjectAttrs.ContentType = "application/pdf"
						objWriter.ObjectAttrs.ContentEncoding = "gzip"
						objWriter.ObjectAttrs.ContentDisposition = fmt.Sprintf(`filename="%s"`, objectName)
						gzipWriter := gzip.NewWriter(objWriter)

						// Get PDF data
						encodedPDF := externalFile.ExtFileName[len(pdfDataURLPrefix):]

						decodedPDF, err := base64.StdEncoding.DecodeString(encodedPDF)
						if err != nil {
							ctx.AbortWithError(http.StatusInternalServerError, err)
							return
						}

						// Upload to GCS
						if _, err := gzipWriter.Write(decodedPDF); err != nil {
							ctx.AbortWithError(http.StatusInternalServerError, err)
							return
						}

						// Close writers
						if err := gzipWriter.Close(); err != nil {
							ctx.AbortWithError(http.StatusInternalServerError, err)
							return
						}

						if err := objWriter.Close(); err != nil {
							ctx.AbortWithError(http.StatusInternalServerError, err)
							return
						}
					} else if err != nil {
						ctx.AbortWithError(http.StatusInternalServerError, err)
						return
					}
				}
			}
		}

		for _, it := range fullIV.InvoiceItems {
			if it.Currency == "" {
				it.Currency = fullIV.Currency
			}

			it.Symbol = fixer.CodeToLabel(it.Currency)
			it.USDExchangeRate = convDay[it.Symbol]

			if v, prs := ProductSKU[it.SKU]; prs && v != "" {
				it.Type = v
			} else {
				it.Type = "other"
			}

			if slice.FindIndex(fullIV.Products, it.Type) == -1 {
				fullIV.Products = append(fullIV.Products, it.Type)
			}
		}

		// The invoice is not fully paid and not canceled
		if !fullIV.Canceled && !fullIV.Paid && !fullIV.PayDate.IsZero() {
			invoice := fullIV.minimize()

			if wireTransferPayer {
				invoiceReminders(ctx, invoiceRef, today, fullIV.PayDate, invoice, reminders)
			}

			if today.After(fullIV.PayDate) {
				overDueInvoices = append(overDueInvoices, invoice)

				if !collectionItem.Date.IsZero() {
					if fullIV.PayDate.Before(collectionItem.Date) {
						collectionItem.Date = fullIV.PayDate
					}
				} else {
					collectionItem.Date = fullIV.PayDate
				}

				switch {
				case collectionItem.Date.Before(severity40):
					collectionItem.Severity = 40
				case collectionItem.Date.Before(severity30):
					collectionItem.Severity = 30
				case collectionItem.Date.Before(severity20):
					collectionItem.Severity = 20
				default:
					collectionItem.Severity = 10
				}

				collectionItem.Weight += debitWeight
				collectionItem.Totals[fullIV.Symbol] += fullIV.Debit

				for _, product := range fullIV.Products {
					if !slice.Contains(collectionItem.Products, product) {
						collectionItem.Products = append(collectionItem.Products, product)
					}
				}
			}
		}

		var mergeOptions firestore.SetOption

		if len(fullIV.InvoiceItems) == 1 &&
			fullIV.InvoiceItems[0].SKU == stripeConsts.FeesSurchargeInvoiceItemSKU {
			// This is a CC Fees invoice, set the stripe locked field to true
			fullIV.StripeLocked = true
			mergeOptions = ccFeesMergeOptions
		} else {
			mergeOptions = invoiceMergeOptions
		}

		if _, err := invoiceRef.Set(ctx, fullIV, mergeOptions); err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}

	latestInvoices := make([]*MinimalInvoice, 0)
	for _, invoice := range invoices {
		if len(latestInvoices) >= MaxLatestInvoices {
			break
		}

		if invoice.Canceled {
			continue
		}

		latestInvoices = append(latestInvoices, invoice.minimize())
	}

	if collectionItem.Weight > 0 {
		customerBatch.Batch.Set(collectionRef.Doc("debt-analytics").Collection("debtAnalytics").Doc(t.EntityID), collectionItem)
	} else {
		customerBatch.Batch.Delete(collectionRef.Doc("debt-analytics").Collection("debtAnalytics").Doc(t.EntityID))
	}

	customerBatch.Count++

	var email *string
	if entity.Contact != nil && entity.Contact.Email != nil {
		email = entity.Contact.Email
	}

	for reminderNum, invoices := range reminders {
		var total float64
		for _, invoice := range invoices {
			total += invoice.Debit
		}

		docID := fmt.Sprintf("%s-%d", t.EntityID, reminderNum)

		ref := collectionRef.Doc("invoice-reminders").Collection("invoiceReminders").Doc(docID)
		if total > 1 {
			customerBatch.Batch.Set(ref, map[string]interface{}{
				"customer":       entity.Customer,
				"entity":         entityRef,
				"invoices":       invoices,
				"contact":        email,
				"date":           today,
				"reminderNumber": reminderNum,
			}, firestore.MergeAll)
		} else {
			customerBatch.Batch.Delete(ref)
		}

		customerBatch.Count++
	}

	customerBatch.Batch.Set(dashboardsRef.Doc("invoices-overdue").Collection("invoicesOverdue").Doc(t.EntityID), map[string]interface{}{
		"customer":   entity.Customer,
		"entity":     entityRef,
		"invoices":   overDueInvoices,
		"totalCount": len(overDueInvoices),
		"timestamp":  firestore.ServerTimestamp,
	}, firestore.MergeAll)

	customerBatch.Count++

	customerBatch.Batch.Set(dashboardsRef.Doc("invoices-latest").Collection("invoicesLatest").Doc(t.EntityID), map[string]interface{}{
		"customer":   entity.Customer,
		"entity":     entityRef,
		"invoices":   latestInvoices,
		"totalCount": len(latestInvoices),
		"timestamp":  firestore.ServerTimestamp,
	}, firestore.MergeAll)

	customerBatch.Count++

	if customerBatch.Count > 0 {
		if _, err := customerBatch.Batch.Commit(ctx); err != nil {
			l.Errorf("customerBatch.Commit: %s", err.Error())
		}
	}
}

func CustomerHandler(ctx *gin.Context, customerID string) {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)

	channelRef, _, err := fs.Collection("channels").Add(ctx, map[string]interface{}{
		"uid":       ctx.GetString("uid"),
		"timestamp": firestore.ServerTimestamp,
		"type":      "customer.invoices.sync",
		"customer":  customerID,
		"complete":  false,
	})
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Mark channel as completed after 1 minute
	defer func(channelID string) {
		go func() {
			timer := time.NewTimer(1 * time.Minute)
			<-timer.C

			if _, err := fs.Collection("channels").Doc(channelID).Update(context.Background(), []firestore.Update{
				{FieldPath: []string{"complete"}, Value: true},
			}); err != nil {
				l.Errorf("failed to update channel: %s", err)
			}
		}()
	}(channelRef.ID)

	customerRef := fs.Collection("customers").Doc(customerID)

	iter := fs.Collection("entities").Where("customer", "==", customerRef).Select("priorityId", "priorityCompany").Documents(ctx)
	defer iter.Stop()

	for {
		docSnap, err := iter.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		var entity common.Entity
		if err := docSnap.DataTo(&entity); err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		openInvoices := make([]*priorityDomain.OpenInvoice, 0)

		if entityOpenInvoices, err := getOpenInvoices(ctx, entity.PriorityCompany, entity.PriorityID); err == nil {
			for _, openInvoice := range entityOpenInvoices {
				if openInvoice.ID == "" || strings.HasPrefix(openInvoice.ID, "RC") {
					continue
				}

				openInvoices = append(openInvoices, openInvoice)
			}
		} else {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		t := CustomerTask{
			EntityID:        docSnap.Ref.ID,
			PriorityID:      entity.PriorityID,
			PriorityCompany: entity.PriorityCompany,
			OpenInvoices:    openInvoices,
			Timestamp:       time.Now().UTC(),
		}

		taskBody, err := json.Marshal(t)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_POST,
			Path:   "/tasks/invoices",
			Queue:  common.TaskQueueDefault,
			Body:   taskBody,
		}

		_, err = common.CreateCloudTask(ctx, &config)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}
}

func getEntityInvoices(ctx *gin.Context, entity *common.Entity) ([]*FullInvoice, error) {
	l := logger.FromContext(ctx)

	forms := [3]map[string]string{
		// Export invoices (No tax)
		{
			"url":    "FINVOICES",
			"select": "IVNUM,DEBIT,IVTYPE,CUSTNAME,CDES,IVDATE,QPRICE,TOTPRICE,CODE,DETAILS,STATDES,ROTL_CMP_NUMBER",
			"expand": "FINVOICEITEMS_SUBFORM($select=PARTNAME,PDES,QUANT,PRICE,PERCENT,DISPRICE,QPRICE,ROTL_EXPPARTDES,FROMDATE,TODATE),CINVOICESCONT_SUBFORM($select=PAYDATE),EXTFILES_SUBFORM($select=EXTFILENAME,UDATE)",
		},

		// Tax invoices
		{
			"url":    "AINVOICES",
			"select": "IVNUM,DEBIT,IVTYPE,CUSTNAME,CDES,IVDATE,QPRICE,TOTPRICE,CODE,DETAILS,STATDES,ROTL_CMP_NUMBER,VAT",
			"expand": "AINVOICEITEMS_SUBFORM($select=PARTNAME,PDES,QUANT,PRICE,PERCENT,DISPRICE,QPRICE,IVTAX,ICODE,EXCH,ROTL_EXPPARTDES,FROMDATE,TODATE),AINVOICESCONT_SUBFORM($select=PAYDATE),EXTFILES_SUBFORM($select=EXTFILENAME,UDATE)",
		},

		// No longer in use by Finance, doesn't have the "ROTL_CMP_NUMBER" value
		{
			"url":    "CINVOICES",
			"select": "IVNUM,DEBIT,IVTYPE,CUSTNAME,CDES,IVDATE,QPRICE,VAT,TOTPRICE,CODE,DETAILS,STATDES",
			"expand": "CINVOICEITEMS_SUBFORM($select=PARTNAME,PDES,QUANT,PRICE,PERCENT,DISPRICE,QPRICE,IVTAX,ICODE,EXCH,ROTL_EXPPARTDES),CINVOICESCONT_SUBFORM($select=PAYDATE),EXTFILES_SUBFORM($select=EXTFILENAME,UDATE)",
		},
	}

	bodies := make([][]byte, 3)
	errors := make([]error, 0)
	wg := &sync.WaitGroup{}
	wg.Add(3)

	for i, form := range forms {
		go func(i int, form map[string]string) {
			defer wg.Done()

			params := make(map[string][]string)
			params["$filter"] = []string{fmt.Sprintf("CUSTNAME eq '%s' and FINAL eq 'Y'", entity.PriorityID)}
			params["$select"] = []string{form["select"]}
			params["$expand"] = []string{form["expand"]}

			path := form["url"]
			if body, err := priority.Client.Get(entity.PriorityCompany, path, params); err != nil {
				errors = append(errors, err)
			} else {
				bodies[i] = body
			}
		}(i, form)
	}

	wg.Wait()

	for _, err := range errors {
		if err != nil {
			l.Errorf("getEntityInvoices failed with error: %s", err)
			return nil, err
		}
	}

	invoices := make([]*FullInvoice, 0)

	for _, body := range bodies {
		result := &FullInvoicesResult{}

		if err := json.Unmarshal(body, result); err != nil {
			return nil, err
		}

		invoices = append(invoices, result.Invoices...)
	}

	return invoices, nil
}

func getOpenInvoices(ctx *gin.Context, company, priorityID string) ([]*priorityDomain.OpenInvoice, error) {
	params := make(map[string][]string)
	if priorityID != "" {
		params["$filter"] = []string{fmt.Sprintf("CUSTNAME eq '%s' and IVNUM gt ''", priorityID)}
	} else {
		params["$filter"] = []string{fmt.Sprintf("IVNUM gt ''")}
	}

	path := "ROTL_OPENIVS"

	body, err := priority.Client.Get(company, path, params)
	if err != nil {
		return nil, err
	}

	var result priorityDomain.OpenInvoicesResponse

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result.OpenInvoices, nil
}
