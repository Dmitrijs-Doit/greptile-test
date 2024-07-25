package invoicing

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	billingExplainerDomain "github.com/doitintl/hello/scheduled-tasks/billing-explainer/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/domain"
)

type BillingMonth struct {
	NumInvoices int64                           `firestore:"numInvoices"`
	UpdatedAt   time.Time                       `firestore:"updatedAt"`
	Products    map[string]*ProductBillingMonth `firestore:"products"`
	Stats       map[string]*Stats               `firestore:"stats"`
}

type Stats struct {
	NumCustomersReady    int64 `firestore:"numCustomersReady"`
	NumInvoicesReady     int64 `firestore:"numInvoicesReady"`
	NumCustomersNotReady int64 `firestore:"numCustomersNotReady"`
	NumInvoicesNotReady  int64 `firestore:"numInvoicesNotReady"`
	NumInvoicesIssued    int64 `firestore:"numInvoicesIssued"`
	NumCustomersIssued   int64 `firestore:"numCustomersIssued"`
}

type ProductBillingMonth struct {
	NumAdjustments int64   `firestore:"numAdjustments"`
	NumCustomers   int64   `firestore:"numCustomers"`
	NumCredits     int64   `firestore:"numCredits"`
	NumInvoices    int64   `firestore:"numInvoices"`
	Total          float64 `firestore:"total"`
	Credits        float64 `firestore:"credits"`
	Adjustments    float64 `firestore:"adjustments"`
}

type EntityBillingDescriptor struct {
	Customer  *firestore.DocumentRef  `firestore:"customer"`
	Entity    *firestore.DocumentRef  `firestore:"entity"`
	Timestamp time.Time               `firestore:"timestamp"`
	Stats     map[string]*EntityStats `firestore:"stats"`
}

type EntityStats struct {
	NumIssued   int64 `firestore:"numIssued"`
	NumFinal    int64 `firestore:"numFinal"`
	NumNotFinal int64 `firestore:"numNotFinal"`
}

type ErrorLine struct {
	Customer  *firestore.DocumentRef `firestore:"customer"`
	Error     string                 `firestore:"error"`
	Type      string                 `firestore:"type"`
	Details   interface{}            `firestore:"details"`
	Timestamp time.Time              `firestore:"timestamp"`
}

type Invoice struct {
	Group                     *string                    `firestore:"group"`
	Date                      time.Time                  `firestore:"date"`
	Type                      string                     `firestore:"type"`
	Details                   string                     `firestore:"details"`
	Rows                      []*domain.InvoiceRow       `firestore:"rows"`
	Timestamp                 time.Time                  `firestore:"timestamp"`
	Currency                  string                     `firestore:"currency"`
	Final                     bool                       `firestore:"final"`
	ExpireBy                  *time.Time                 `firestore:"expireBy"`
	Parent                    *firestore.DocumentRef     `firestore:"-"`
	IssuedAt                  *time.Time                 `firestore:"issuedAt"`
	CanceledAt                *time.Time                 `firestore:"canceledAt"`
	Note                      string                     `firestore:"note"`
	ID                        *string                    `firestore:"id"`
	InconclusiveInvoiceReason *inconclusiveInvoiceReason `firestore:"inconclusiveInvoiceReason"`
	CancellationReason        string                     `firestore:"cancellationReason"`
}

type inconclusiveInvoiceReason string

const prodInvoicingProjectID = "me-doit-intl-com"
const devInvoicingProjectID = "doitintl-cmp-dev"

// define all possible reasons for an invoice to be inconclusive
const (
	lowCostInvoice         inconclusiveInvoiceReason = "lowCost"
	currencyErrorInvoice   inconclusiveInvoiceReason = "currencyError"
)

const (
	flexsaveManagementCost                           = "Customer incurred costs on Flexsave accounts"
	flexsaveRDSCharge                                = "Flexsave RDS Charges"
)

const (
	InvoiceMonthPattern                              = "2006-01"
	maxInvoiceSize                                   = 500 // max amount of rows in an invoice
)

type ProcessInvoicesInput struct {
	InvoiceMonth  string
	TimeIndex     string
	PrimaryDomain string
}

type customerProductInvoicingWorker func(context.Context, *domain.CustomerTaskData, *firestore.DocumentRef, map[string]*common.Entity, chan<- *domain.ProductInvoiceRows)

func getInvoicingProjectID() string {
	if common.ProjectID == "me-doit-intl-com" {
		return prodInvoicingProjectID
	}

	return devInvoicingProjectID
}

// ProcessCustomersInvoices updates the invoices for all customers for a given month
func (s *InvoicingService) ProcessCustomersInvoices(ctx context.Context, params *ProcessInvoicesInput, processWithCloudTask bool) error {
	logger := s.Logger(ctx)
	fs := s.Firestore(ctx)

	if params == nil {
		return fmt.Errorf("invalid input params")
	}

	var (
		invoiceMonth time.Time
		ratesDate    time.Time
		timeIndex    int
	)

	now := time.Now().UTC()

	if params.InvoiceMonth != "" {
		parsedDate, err := time.Parse("2006-01-02", params.InvoiceMonth)
		if err != nil {
			return err
		}

		if parsedDate.After(now) {
			return err
		}

		invoiceMonth = time.Date(parsedDate.Year(), parsedDate.Month()+1, 0, 0, 0, 0, 0, time.UTC)
		ratesDate = invoiceMonth

		timeIndex64, err := strconv.ParseInt(params.TimeIndex, 10, 64)
		if err != nil {
			return err
		}

		if timeIndex64 > -1 {
			return err
		}

		timeIndex = int(timeIndex64)
	} else {
		if now.Day() > 10 {
			invoiceMonth = time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, time.UTC)
			ratesDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
			timeIndex = -1
		} else {
			invoiceMonth = time.Date(now.Year(), now.Month(), 0, 0, 0, 0, 0, time.UTC)
			ratesDate = invoiceMonth
			timeIndex = -2
		}
	}

	logger.Infof("Invoice Month %v", invoiceMonth)
	logger.Infof("Rates Date %v", ratesDate)
	logger.Infof("Time Index %v", timeIndex)
	logger.Infof("Invoice batch marked with timestamp %v", now.Format(time.RFC3339))

	historicalRates, err := common.HistoricalRates(ratesDate)
	if err != nil {
		return err
	}

	logBatch := time.Now().UTC().Truncate(time.Hour * 6).Format(time.DateTime)

	var billingMonth = BillingMonth{
		NumInvoices: 0,
		UpdatedAt:   now,
		Products:    make(map[string]*ProductBillingMonth),
		Stats:       make(map[string]*Stats),
	}

	invoicingMonthRef := fs.Collection("billing").Doc("invoicing").Collection("invoicingMonths").Doc(invoiceMonth.Format("2006-01"))

	var customersIterator *firestore.DocumentIterator
	if params.PrimaryDomain == "*" {
		customersIterator = fs.Collection("customers").Select().Documents(ctx)
		// Update the invoicing month doc updatedAt field
		if _, err := invoicingMonthRef.Set(ctx, billingMonth); err != nil {
			return err
		}
	} else {
		customersIterator = fs.Collection("customers").Where("primaryDomain", "in", strings.Split(params.PrimaryDomain, ",")).Documents(ctx)
		updates := []firestore.Update{
			{Path: "updatedAt", Value: now},
		}
		if _, err := invoicingMonthRef.Update(ctx, updates); err != nil {
			return err
		}
	}

	defer customersIterator.Stop()

	for {
		customerDocSnap, err := customersIterator.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return err
		}

		customerID := customerDocSnap.Ref.ID

		t := domain.CustomerTaskData{
			CustomerID:   customerDocSnap.Ref.ID,
			Now:          now,
			InvoiceMonth: invoiceMonth,
			Rates:        historicalRates.Rates,
			TimeIndex:    timeIndex,
		}

		logger.Debugf("AWS-INVOICE-REGRESSION:%v|%v|%v|%v|%v|%v", logBatch, customerID, "NA", 0, "ProcessDraftInvoice-AllCustomers", "creating draft-invoice task with timestamp "+now.Format(time.DateTime))

		if !processWithCloudTask {
			err = s.ProcessCustomerInvoices(ctx, &t)
			if err != nil {
				return err
			}
			continue
		} else {
			config := common.CloudTaskConfig{
				Method: cloudtaskspb.HttpMethod_POST,
				Path:   fmt.Sprintf("/tasks/invoicing/customers/%s", customerID),
				Queue:  common.TaskQueueInvoicing,
			}

			_, err = s.Connection.CloudTaskClient.CreateTask(ctx, config.Config(t))
			if err != nil {
				return err
			}
		}

	}

	if params.PrimaryDomain != "*" {
		return nil
	}

	return nil
}

// ProcessCustomerInvoices updates the invoices for a customer for a given month
func (s *InvoicingService) ProcessCustomerInvoices(ctx context.Context, task *domain.CustomerTaskData) error {
	logger := s.Logger(ctx)
	fs := s.Firestore(ctx)
	logBatch := time.Now().UTC().Truncate(time.Hour * 6).Format(time.DateTime)

	logger.Infof("ProcessCustomerInvoices started customer %v invoiceMonth %v", task.CustomerID, task.InvoiceMonth)
	logger.Info(task)
	logger.Debugf("AWS-INVOICE-REGRESSION:%v|%v|%v|%v|%v|%v", logBatch, task.CustomerID, "NA", 0, "ProcessDraftInvoice-SingleCustomer", fmt.Sprintf("started draft invoice processing for invoice month %v with timestamp %v", task.InvoiceMonth.Format(time.DateTime), task.Now.Format(time.DateTime)))

	workerChan := make(chan *domain.ProductInvoiceRows)
	entitiesRows := make(map[string]map[string][]*domain.InvoiceRow)
	invoices := make(map[string]*Invoice)
	billingMonthRef := fs.Collection("billing").Doc("invoicing").Collection("invoicingMonths").Doc(task.InvoiceMonth.Format("2006-01"))
	billingMonthErrors := billingMonthRef.Collection("monthErrors")
	customerRef := fs.Collection("customers").Doc(task.CustomerID)
	entities := make(map[string]*common.Entity)

	entitiesIter := fs.Collection("entities").Where("customer", "==", customerRef).Documents(ctx)
	defer entitiesIter.Stop()

	entityCount := 0
	for {
		entityDocSnap, err := entitiesIter.Next()
		if err == iterator.Done {
			if entityCount == 0 {
				// no return, asset not linked with entity error will generate monthError in collection, and eventually in invoice sheet
				logger.Errorf("ProcessCustomerInvoices: customer %v no entity found", customerRef.ID)
			}

			break
		}

		var monthError error
		var entity common.Entity

		returnError := false

		if err != nil {
			monthError = fmt.Errorf("ProcessCustomerInvoices: customer %v entitiesIter.Next errored: %v", customerRef.ID, err.Error())
			logger.Errorf(monthError.Error())
			return fmt.Errorf("entitiesIter.Next: %s", err)
		} else {
			if err := entityDocSnap.DataTo(&entity); err != nil {
				monthError = fmt.Errorf("ProcessCustomerInvoices: customer %v entityDocSnap.DataTo errored: %v", customerRef.ID, err.Error())
				logger.Errorf(monthError.Error())
				//return fmt.Errorf("entityDocSnap.DataTo: %s", err)
			}
		}

		if monthError == nil {
			entity.Snapshot = entityDocSnap
			if entity.Invoicing.Mode == "CUSTOM" && entity.Invoicing.Default == nil {
				monthError = fmt.Errorf("entity %s invoicing mode set to CUSTOM but has no default bucket", entity.PriorityID)
				logger.Errorf("ProcessCustomerInvoices: customer %v errored: %v", customerRef.ID, monthError.Error())
				returnError = true
			}
		}

		if monthError != nil {
			if _, _, err := billingMonthRef.Collection("monthErrors").Add(ctx, map[string]interface{}{
				"type":      nil,
				"timestamp": task.Now,
				"customer":  customerRef,
				"error":     monthError.Error(),
				"details":   entity,
			}); err != nil {
				logger.Errorf("ProcessCustomerInvoices: customer %v errored when adding monthError record %v", customerRef.ID, err.Error())
			}

			if returnError {
				return nil
			}
		}

		entities[entityDocSnap.Ref.ID] = &entity

		entityCount++
	}

	assetInvoiceWorkers := []customerProductInvoicingWorker{
		s.customerMicrosoftAzureHandler,
		s.customerAssetInvoice.GetAWSInvoiceRows,
		s.customerAssetInvoice.GetAWSStandaloneInvoiceRows,
		s.customerAssetInvoice.GetGCPStandaloneInvoiceRows,
		s.customerGoogleCloudHandler,
		s.customerGSuiteHandler,
		s.customerOffice365Handler,
		s.lookerInvoicingService.GetInvoiceRows,
		s.customerDoITPackageInvoice.GetDoITNavigatorInvoiceRows,
		s.customerDoITPackageInvoice.GetDoITSolveInvoiceRows,
		s.customerDoITPackageInvoice.GetDoITSolveAcceleratorInvoiceRows,
	}
	for _, fn := range assetInvoiceWorkers {
		go fn(ctx, task, customerRef, entities, workerChan)
	}

	batch := fb.NewAutomaticWriteBatch(fs, 100)

	var awsEntityIDList []string

	for i := 0; i < len(assetInvoiceWorkers); i++ {
		result := <-workerChan
		if result.Error != nil {
			logger.Errorf("customer %s product %s worker failed with error: %s", customerRef.ID, result.Type, result.Error)
			batch.Create(billingMonthErrors.NewDoc(), map[string]interface{}{
				"type":      result.Type,
				"timestamp": task.Now,
				"customer":  customerRef,
				"error":     result.Error.Error(),
			})

			continue // if one product type failed, continue to other types
		}

		productTypeHasError := false
		productTypeEntityRows := make(map[string][]*domain.InvoiceRow)

		for _, row := range result.Rows {
			if row.Entity == nil {
				err := fmt.Errorf("customer %s product %s invoice row not assigned to billing profile", customerRef.ID, result.Type)
				logger.Error(err)
				batch.Create(billingMonthErrors.NewDoc(), map[string]interface{}{
					"type":      result.Type,
					"timestamp": task.Now,
					"customer":  customerRef,
					"error":     err.Error(),
					"details":   row,
				})

				productTypeHasError = true

				break
			}

			entity, prs := entities[row.Entity.ID]
			if !prs {
				err := fmt.Errorf("customer %s product %s invoice row assigned to invalid entity %s", customerRef.ID, result.Type, row.Entity.ID)
				logger.Error(err)
				batch.Create(billingMonthErrors.NewDoc(), map[string]interface{}{
					"type":      result.Type,
					"timestamp": task.Now,
					"customer":  customerRef,
					"error":     err.Error(),
					"details":   row,
				})

				productTypeHasError = true

				break
			}

			if !entity.Active {
				err := fmt.Errorf("customer %s entity %s is disabled", customerRef.ID, row.Entity.ID)
				logger.Error(err)
				batch.Create(billingMonthErrors.NewDoc(), map[string]interface{}{
					"type":      result.Type,
					"timestamp": task.Now,
					"customer":  customerRef,
					"error":     err.Error(),
					"details":   row,
				})

				productTypeHasError = true

				break
			}

			if _, ok := productTypeEntityRows[row.Entity.ID]; !ok {
				productTypeEntityRows[row.Entity.ID] = make([]*domain.InvoiceRow, 0)
			}

			productTypeEntityRows[row.Entity.ID] = append(productTypeEntityRows[row.Entity.ID], row)
		}

		if !productTypeHasError {
			for entityID, rows := range productTypeEntityRows {
				if _, ok := entitiesRows[entityID]; !ok {
					entitiesRows[entityID] = make(map[string][]*domain.InvoiceRow)
				}

				if result.Type == common.Assets.AmazonWebServices {
					awsEntityIDList = append(awsEntityIDList, entityID)
				}

				entitiesRows[entityID][result.Type] = append(entitiesRows[entityID][result.Type], rows...)
			}
		}
	}

	var customerAwsStats EntityStats

	for entityID, entityTypesRows := range entitiesRows {
		entity := entities[entityID]
		entityRef := fs.Collection("entities").Doc(entityID)
		entityBillingDescriptorRef := billingMonthRef.Collection("monthInvoices").Doc(entityID)
		// Get number of issued invoices per entity and add to the total per customer.
		// If the billing descriptor does not exist for the entity, this might be the first time we
		// are generating invoices for this entity. In this case, there are no previously issued invoices.
		dsnap, err := entityBillingDescriptorRef.Get(ctx)
		if err != nil {
			if status.Code(err) != codes.NotFound {
				err2 := fmt.Errorf("ProcessCustomerInvoices: customer %v entityBillingDescriptorRef.Get(ctx): %s", customerRef.ID, err)
				logger.Errorf(err2.Error())

				return err2
			}
		} else {
			var billingDescriptor EntityBillingDescriptor
			if err := dsnap.DataTo(&billingDescriptor); err != nil {
				err2 := fmt.Errorf("ProcessCustomerInvoices: customer %v dsnap.DataTo(&billingDescriptor): %s", customerRef.ID, err)
				logger.Errorf(err2.Error())

				return err2
			}

			if _, ok := billingDescriptor.Stats[common.Assets.AmazonWebServices]; ok {
				customerAwsStats.NumIssued += billingDescriptor.Stats[common.Assets.AmazonWebServices].NumIssued
			}
		}

		batch.Set(entityBillingDescriptorRef, map[string]interface{}{"customer": customerRef, "entity": entityRef, "timestamp": task.Now}, firestore.MergeAll)

		for assetType, rows := range entityTypesRows {
			invoicingMode := entity.Invoicing.Mode

			// as next10 contracts do Not have assetSettings and does not support buckets,
			// change custom invoice mode to group only for next10 assetTypes
			if (assetType == common.Assets.DoiTNavigator || assetType == common.Assets.DoiTSolve) &&
				invoicingMode == "CUSTOM" {
				invoicingMode = "GROUP"
			}

			if assetType == common.Assets.AmazonWebServices {
				batchedRows := batchRowsBasedOnCategory(rows)
				if len(batchedRows) > 1 {
					invoicingMode = "CUSTOM"
				}
			}

			switch invoicingMode {
			case "SINGLE":
				invoiceID := entityID
				if invoice, prs := invoices[invoiceID]; !prs {
					invoices[invoiceID] = &Invoice{
						Parent:    entityBillingDescriptorRef,
						Date:      task.InvoiceMonth,
						Details:   fmt.Sprintf("Covering %s", task.InvoiceMonth.Format("January 2006")),
						Rows:      rows,
						Timestamp: task.Now,
					}
				} else {
					invoice.Rows = append(invoice.Rows, rows...)
				}

			case "GROUP":
				invoiceID := fmt.Sprintf("%s-%s", entityID, assetType)
				if invoice, prs := invoices[invoiceID]; !prs {
					invoices[invoiceID] = &Invoice{
						Parent:    entityBillingDescriptorRef,
						Date:      task.InvoiceMonth,
						Details:   fmt.Sprintf("Covering %s", task.InvoiceMonth.Format("January 2006")),
						Rows:      rows,
						Timestamp: task.Now,
					}
				} else {
					invoice.Rows = append(invoice.Rows, rows...)
				}

			case "CUSTOM":
				for _, row := range rows {
					bucketID := ""
					if entity.Invoicing.Default != nil {
						bucketID = entity.Invoicing.Default.ID
					}

					if row.Bucket != nil {
						bucketID = row.Bucket.ID
					}

					if row.Category != "" && row.Category != "default" {
						bucketID = bucketID + "_" + row.Category
					}

					invoiceID := fmt.Sprintf("%s-%s", entityID, bucketID)

					invoice, ok := invoices[invoiceID]
					if !ok {
						invoice = &Invoice{
							Parent:    entityBillingDescriptorRef,
							Date:      task.InvoiceMonth,
							Details:   fmt.Sprintf("Covering %s", task.InvoiceMonth.Format("January 2006")),
							Rows:      make([]*domain.InvoiceRow, 0),
							Timestamp: task.Now,
						}
						invoices[invoiceID] = invoice

						// Add a row with 0 cost to the start of the invoice items
						// with the name of the invoice bucket
						if bucketDocSnap, err := row.Bucket.Get(ctx); err == nil {
							if v, err := bucketDocSnap.DataAt("name"); err == nil {
								if bucketName, ok := v.(string); ok && bucketName != "" {
									if row.Category != "" {
										bucketName = bucketName + "_" + row.Category
									}
									invoice.Rows = append(invoice.Rows, &domain.InvoiceRow{
										Description: "Invoice Bucket",
										Details:     bucketName,
										SKU:         InvoicingInfoSKU,
										Rank:        -1,
										Currency:    row.Currency,
										Type:        assetType,
										Final:       true,
									})
								}
							}
						}
					}

					invoice.Rows = append(invoice.Rows, row)
				}
			}
		}
	}

	numInvoices := int64(len(invoices))
	nonFinalInvoiceExpireDate := time.Now().UTC().AddDate(0, 0, 45)

	var pushBillingExplainerTask = false

	if numInvoices > 0 {
		products := make(map[string]*ProductBillingMonth)

		for _, invoice := range invoices {
			var totalUSD float64

			var final = true

			var currencyError = false

			var typeError = false

			var hasCredit = false

			var hasAdjustment = false

			invoice.Type = invoice.Rows[0].Type
			invoice.Currency = invoice.Rows[0].Currency

			for _, row := range invoice.Rows {
				final = final && row.Final
				typeError = typeError || (invoice.Type != row.Type)
				currencyError = currencyError || (row.Rank != InvoiceAdjustmentRank && invoice.Currency != row.Currency)

				var rate float64 = 1
				if v, prs := task.Rates[row.Currency]; prs {
					rate = v
				}

				rowValue := row.Total / rate
				totalUSD += rowValue

				if v, prs := products[row.Type]; prs {
					v.Total += rowValue
				} else {
					products[row.Type] = &ProductBillingMonth{
						Total: rowValue,
					}
				}

				switch row.Rank {
				case CreditRank:
					hasCredit = true
					products[row.Type].Credits += rowValue
				case InvoiceAdjustmentRank:
					hasAdjustment = true
					products[row.Type].Adjustments += rowValue
				default:
				}

				if row.Type == common.Assets.AmazonWebServices {
					entityID := "NA"

					if row.Entity != nil {
						entityID = row.Entity.ID
					}

					logger.Debugf("AWS-INVOICE-REGRESSION:%v|%v|%v|%v|%v|%v", logBatch, customerRef.ID, entityID, row.Total, "ProcessDraftInvoice-SingleCustomer", fmt.Sprintf("%s : %s : final:%v", row.Description, row.Details, final))
				}
			}

			// Don't issue  invoices with low total (<= $1) even if those are final
			if totalUSD <= 1 && final && invoice.Type != common.Assets.AmazonWebServices && invoice.Type != common.Assets.GoogleCloud {
				final = false
				invoice.InconclusiveInvoiceReason = lowCostInvoice.Pointer()
			}

			if currencyError {
				// allow the creation of invoice with multiple different currencies but mark it as "inconclusive"
				logger.Warningf("customer %s invoice has multiple currencies", customerRef.ID)

				final = false
				invoice.InconclusiveInvoiceReason = currencyErrorInvoice.Pointer()
			}

			invoice.Final = final
			if !invoice.Final {
				invoice.ExpireBy = &nonFinalInvoiceExpireDate
			}

			products[invoice.Type].NumCustomers = 1
			products[invoice.Type].NumInvoices++

			if hasCredit {
				products[invoice.Type].NumCredits++
			}

			if hasAdjustment {
				products[invoice.Type].NumAdjustments++
			}

			if invoice.Type == common.Assets.AmazonWebServices {
				// Update final invoices counter per customer
				if final {
					pushBillingExplainerTask = true
					customerAwsStats.NumFinal++
				} else {
					customerAwsStats.NumNotFinal++
				}
			}

			if typeError {
				err := fmt.Errorf("customer %s invoice has multiple product types", customerRef.ID)
				logger.Error(err)
				batch.Create(billingMonthErrors.NewDoc(), map[string]interface{}{
					"type":      "invoice",
					"timestamp": task.Now,
					"customer":  customerRef,
					"error":     err.Error(),
					"details":   invoice,
				})

				continue
			}

			groupRef := invoice.Parent.Collection("entityInvoices").NewDoc()
			invoiceSize := len(invoice.Rows)

			if invoiceSize < maxInvoiceSize {
				// Small invoices are saved as they are
				batch.Create(groupRef, invoice)
				logger.Infof("adding invoice to commit batch, customerId: %v - Path: %v", customerRef.ID, groupRef.Path)

				if invoice.Rows[0].Type == common.Assets.AmazonWebServices {
					logger.Debugf("AWS-INVOICE-REGRESSION:%v|%v|%v|%v|%v|%v", logBatch, customerRef.ID, "NA", totalUSD, "ProcessDraftInvoice-SingleCustomer", fmt.Sprintf("Invoice created at Path:%s ", groupRef.Path))
				}
			} else {
				invoice.Group = &groupRef.ID

				// If the invoice has too many rows we may have issues saving it to firestore.
				// We split big invoices into chunks and group them together when reading.
				for i, j := 0, 0; i < invoiceSize; i += maxInvoiceSize {
					j += maxInvoiceSize
					if j > invoiceSize {
						j = invoiceSize
					}

					invoiceChunk := Invoice{
						Group:                     invoice.Group,
						Date:                      invoice.Date,
						Type:                      invoice.Type,
						Details:                   invoice.Details,
						Rows:                      invoice.Rows[i:j],
						Timestamp:                 invoice.Timestamp,
						Currency:                  invoice.Currency,
						Final:                     invoice.Final,
						ExpireBy:                  invoice.ExpireBy,
						Parent:                    invoice.Parent,
						InconclusiveInvoiceReason: invoice.InconclusiveInvoiceReason,
					}

					newDoc := invoice.Parent.Collection("entityInvoices").NewDoc()
					batch.Create(newDoc, invoiceChunk)
					logger.Infof("adding invoice-chunk to commit batch, customerId: %v - Path: %v", customerRef.ID, newDoc.Path)

					if invoice.Rows[0].Type == common.Assets.AmazonWebServices {
						logger.Debugf("AWS-INVOICE-REGRESSION:%v|%v|%v|%v|%v|%v", logBatch, customerRef.ID, "NA", totalUSD, "ProcessDraftInvoice-SingleCustomer", fmt.Sprintf("Invoice-chunk created at Path:%s ", newDoc.Path))
					}
				}
			}
		}

		if err := fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
			docSnap, err := tx.Get(billingMonthRef)
			if err != nil {
				return err
			}

			var billingMonth BillingMonth
			if err := docSnap.DataTo(&billingMonth); err != nil {
				return err
			}

			// Update amount of invoices
			billingMonth.NumInvoices = billingMonth.NumInvoices + numInvoices

			// Update products statistics
			for t, val := range products {
				if _, prs := billingMonth.Products[t]; prs {
					billingMonth.Products[t].Total += val.Total
					billingMonth.Products[t].Credits += val.Credits
					billingMonth.Products[t].Adjustments += val.Adjustments
					billingMonth.Products[t].NumCustomers += val.NumCustomers
					billingMonth.Products[t].NumInvoices += val.NumInvoices
					billingMonth.Products[t].NumCredits += val.NumCredits
					billingMonth.Products[t].NumAdjustments += val.NumAdjustments
				} else {
					billingMonth.Products[t] = val
				}

				if t == common.Assets.AmazonWebServices {
					// First time init
					if _, ok := billingMonth.Stats[t]; !ok {
						billingMonth.Stats[t] = &Stats{}
					}
					// Update AWS invoicing related stats
					numInvoicesReady := customerAwsStats.NumFinal - customerAwsStats.NumIssued
					billingMonth.Stats[t].NumInvoicesIssued += customerAwsStats.NumIssued
					billingMonth.Stats[t].NumInvoicesReady += numInvoicesReady
					billingMonth.Stats[t].NumInvoicesNotReady += customerAwsStats.NumNotFinal
					if numInvoicesReady > 0 {
						// Customer is considered as ready if there are final invoives ready to be issued
						billingMonth.Stats[t].NumCustomersReady++
					} else if customerAwsStats.NumIssued > 0 {
						// If any invoice has been issued we can mark a customer as issued
						billingMonth.Stats[t].NumCustomersIssued++
					} else {
						billingMonth.Stats[t].NumCustomersNotReady++
					}
				}
			}
			return tx.Set(billingMonthRef, billingMonth, firestore.Merge([]string{"numInvoices"}, []string{"products"}, []string{"stats"}))
		}, firestore.MaxAttempts(10)); err != nil {
			err2 := fmt.Errorf("ProcessCustomerInvoices: customer %v invoicing fs.RunTransaction: %s", customerRef.ID, err)
			logger.Errorf(err2.Error())
			return err2
		}
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		err2 := fmt.Errorf("invoicing batch.Commit: %s", errs[0])
		logger.Errorf("ProcessCustomerInvoices: customer %v errored: %v", customerRef.ID, err2.Error())
		return err2
	}

	logger.Infof("ProcessCustomerInvoices completed customer %v invoiceMonth %v", task.CustomerID, task.InvoiceMonth)

	useAnalyticsDataForInvoice, err := s.customerAssetInvoice.IsUseAnalyticsDataForInvoice(ctx, customerRef)

	if err != nil {
		logger.Error(err.Error())
	}

	if len(awsEntityIDList) > 0 && useAnalyticsDataForInvoice && pushBillingExplainerTask {
		for _, entityID := range awsEntityIDList {
			var billingExplainerStruct billingExplainerDomain.BillingExplainerInputStruct
			billingExplainerStruct.BillingMonth = task.InvoiceMonth.Format("200601")
			billingExplainerStruct.CustomerID = task.CustomerID
			billingExplainerStruct.EntityID = entityID

			taskBody, err := json.Marshal(billingExplainerStruct)

			if err != nil {
				logger.Error(err.Error())
				continue
			}

			config := common.CloudTaskConfig{
				Method:       cloudtaskspb.HttpMethod_POST,
				Path:         "/tasks/billing-explainer/data",
				Queue:        common.TaskQueueBillingExplainer,
				Body:         taskBody,
				ScheduleTime: common.TimeToTimestamp(time.Now().UTC().Add(time.Minute * 180)),
			}

			_, err = common.CreateCloudTask(ctx, &config)

			if err != nil {
				logger.Error(err.Error())
				continue
			}
		}
	}

	return nil
}

// Pointer returns pointer of InconclusiveInvoiceReason
func (r inconclusiveInvoiceReason) Pointer() *inconclusiveInvoiceReason {
	return &r
}

func batchRowsBasedOnCategory(rows []*domain.InvoiceRow) map[string][]*domain.InvoiceRow {
	batchedRows := make(map[string][]*domain.InvoiceRow)

	batchedRows["default"] = []*domain.InvoiceRow{}

	for _, row := range rows {
		if row.Category == "" || row.Category == "default" {
			batchedRows["default"] = append(batchedRows["default"], row)
			continue
		}

		_, ok := batchedRows[row.Category]
		if !ok {
			batchedRows[row.Category] = []*domain.InvoiceRow{}
		}

		batchedRows[row.Category] = append(batchedRows[row.Category], row)
	}

	return batchedRows
}
