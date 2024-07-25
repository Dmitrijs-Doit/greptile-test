package invoicing

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/common"
	googleDrive "github.com/doitintl/hello/scheduled-tasks/customer/drive"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/domain"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/utils"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/mailer"
	"github.com/doitintl/hello/scheduled-tasks/priority"
)

type ExportInvoicesRequest struct {
	CustomerID *string   `json:"customerId"`
	Year       int64    `json:"year"`
	Month      int64    `json:"month"`
	Types      []string `json:"types"`
	Override   bool     `json:"override"`
}

type InvoiceExportParams struct {
	DocID        string
	Year         int64
	Month        int64
	AssetTypes   []string
	Override     bool
	Email        string
	DevMode      bool
	DevDriveName *string
	CustomerID   *string
}

type GcpFlexsaveInvoicingVersion struct {
	AllowDevExport     bool   `firestore:"allowDevExport"`
	InvoiceExportDrive string `firestore:"invoiceExportDrive"`
	Version            string `firestore:"version"`
}

const (
	defaultInvoiceMaxLineItems int     = 10
	minTotal                   float64 = 1e-2

	shortDateFormat string = "020106"

	// Formula: UNITS * PPU_IN_CURRENCY * (1 - DISCOUNT%) / USD_EXCH_RATE
	convRateFormula string = `=INDIRECT("R[0]C[-13]", FALSE) * INDIRECT("R[0]C[-12]", FALSE) * (1 - 0.01 * INDIRECT("R[0]C[-11]", FALSE)) / VLOOKUP(INDIRECT("R[0]C[-10]", FALSE), RATES!A:B, 2, FALSE)`

	sheetISR int64 = iota + 1
	sheetNonISR
	sheetUS
	sheetNonUS
	sheetUK
	sheetNonUK
	sheetAU
	sheetNonAU
	sheetDE
	sheetNonDE
	sheetFR
	sheetNonFR
	sheetNL
	sheetNonNL
	sheetCH
	sheetNonCH
	sheetCA
	sheetNonCA
	sheetSE
	sheetNonSE
	sheetES
	sheetNonES
	sheetIE
	sheetNonIE
	sheetEE
	sheetNonEE
	sheetSG
	sheetNonSG
	sheetJP
	sheetNonJP
	sheetIND
	sheetNonIND
	sheetINCONCLUSIVE
	sheetLowCost
	sheetCurrencyError
	sheetISSUED
	sheetISSUED3
	sheetISSUED4
	sheetISSUED5
	sheetISSUED6
	sheetISSUED7
	sheetISSUED8
	sheetISSUED9
	sheetISSUED10
	sheetISSUED11
	sheetISSUED12
	sheetERROR
	sheetVERIFICATION
	sheetRATES
	sheetOnHold
)

var (
	currencies = fixer.Currencies
)

// DevExportInvoices exports invoices of given products and month to google spreadsheets
// When looping with entitiesBillingIter use one entity located in the beginning of the billing month collection and break after to avoid waiting.
func (s *InvoicingService) DevExportInvoices(ctx context.Context, params *ExportInvoicesRequest, uid, email string) (map[string]string, error) {
	fs := s.Firestore(ctx)

	isAllowed, devDriveName, err := checkDevModeAllowed(ctx, fs)
	if err != nil || !isAllowed {
		return nil, err
	}

	return s.ExportInvoices(ctx, params, uid, email, true, &devDriveName)
}

// ExportInvoices exports invoices of given products and month to google spreadsheets
// If customerId is provided, export only invoices of that customer
// If override is true, export all invoices and don't update issuedAt field
func (s *InvoicingService) ExportInvoices(ctx context.Context, params *ExportInvoicesRequest, uid, email string, devMode bool, devDriveName *string) (map[string]string, error) {
	l := s.Logger(ctx)
	fs := s.Firestore(ctx)

	if params.CustomerID != nil {
		l.Infof("Running Export for customerId: %v", *params.CustomerID)
	} else {
		l.Info("Running Export for all customers")
	}
	l.Infof("ExportInvoices: Params %v", params)

	if len(params.Types) <= 0 || params.Year < 2019 || params.Month < 1 || params.Month > 12 {
		return nil, web.ErrBadRequest
	}

	for _, t := range params.Types {
		switch t {
		case common.Assets.GSuite:
		case common.Assets.Office365:
		case common.Assets.GoogleCloud:
		case common.Assets.GoogleCloudStandalone:
		case common.Assets.AmazonWebServices:
		case common.Assets.AmazonWebServicesStandalone:
		case common.Assets.MicrosoftAzure:
		case utils.LookerType:
		case utils.NavigatorType:
		case utils.SolveType:
		case utils.SolveAcceleratorType:
		default:
			l.Errorf("exportInvoices: %v", web.ErrBadRequest)
			return nil, web.ErrBadRequest
		}
	}

	doc, _, err := fs.Collection("channels").Add(ctx, map[string]interface{}{
		"uid":       uid,
		"timestamp": firestore.ServerTimestamp,
		"state":     "initializing",
		"type":      "billing.invoicing",
		"progress":  0.0,
		"complete":  false,
	})
	if err != nil {
		return nil, web.ErrInternalServerError
	}

	exportParams := InvoiceExportParams{
		DocID:        doc.ID,
		Year:         params.Year,
		Month:        params.Month,
		AssetTypes:   params.Types,
		Override:     params.Override,
		Email:        email,
		DevMode:      devMode,
		DevDriveName: devDriveName,
		CustomerID:   params.CustomerID,
	}

	go s.exportInvoices(ctx, doc.ID, exportParams)

	return map[string]string{
		"channelId": doc.ID,
	}, nil
}

func (s *InvoicingService) getInvoiceSheets(extendedMode bool) []googleDrive.SheetInfo {
	invoiceSheets := make([]googleDrive.SheetInfo, 0)

	invoiceSheets = append(invoiceSheets,
		googleDrive.SheetInfo{sheetISR, "ISR", true, true},
		googleDrive.SheetInfo{sheetNonISR, "NO_ISR", true, true},
		googleDrive.SheetInfo{sheetUS, "US", true, true},
		googleDrive.SheetInfo{sheetNonUS, "NO_US", true, true},
		googleDrive.SheetInfo{sheetUK, "UK", true, true},
		googleDrive.SheetInfo{sheetNonUK, "NO_UK", true, true},
		googleDrive.SheetInfo{sheetAU, "AU", true, true},
		googleDrive.SheetInfo{sheetNonAU, "NO_AU", true, true},
		googleDrive.SheetInfo{sheetDE, "DE", true, true},
		googleDrive.SheetInfo{sheetNonDE, "NO_DE", true, true},
		googleDrive.SheetInfo{sheetFR, "FR", true, true},
		googleDrive.SheetInfo{sheetNonFR, "NO_FR", true, true},
		googleDrive.SheetInfo{sheetNL, "NL", true, true},
		googleDrive.SheetInfo{sheetNonNL, "NO_NL", true, true},
		googleDrive.SheetInfo{sheetCH, "CH", true, true},
		googleDrive.SheetInfo{sheetNonCH, "NO_CH", true, true},
		googleDrive.SheetInfo{sheetCA, "CA", true, true},
		googleDrive.SheetInfo{sheetNonCA, "NO_CA", true, true},
		googleDrive.SheetInfo{sheetSE, "SE", true, true},
		googleDrive.SheetInfo{sheetNonSE, "NO_SE", true, true},
		googleDrive.SheetInfo{sheetES, "ES", true, true},
		googleDrive.SheetInfo{sheetNonES, "NO_ES", true, true},
		googleDrive.SheetInfo{sheetIE, "IE", true, true},
		googleDrive.SheetInfo{sheetNonIE, "NO_IE", true, true},
		googleDrive.SheetInfo{sheetEE, "EE", true, true},
		googleDrive.SheetInfo{sheetNonEE, "NO_EE", true, true},
		googleDrive.SheetInfo{sheetSG, "SG", true, true},
		googleDrive.SheetInfo{sheetNonSG, "NO_SG", true, true},
		googleDrive.SheetInfo{sheetJP, "JP", true, true},
		googleDrive.SheetInfo{sheetNonJP, "NO_JP", true, true},
		googleDrive.SheetInfo{sheetIND, "ID", true, true},
		googleDrive.SheetInfo{sheetNonIND, "NO_ID", true, true},
		googleDrive.SheetInfo{sheetINCONCLUSIVE, "INCONCLUSIVE", true, true},
		googleDrive.SheetInfo{sheetLowCost, "LOW_COST", true, true},
		googleDrive.SheetInfo{sheetCurrencyError, "CURRENCY_ERROR", true, true},
		googleDrive.SheetInfo{sheetOnHold, "ON_HOLD", true, true},
	)

	if extendedMode {
		invoiceSheets = append(invoiceSheets,
			googleDrive.SheetInfo{sheetISSUED, "ISSUED", true, false},
			googleDrive.SheetInfo{sheetISSUED3, "ISSUED3", true, false},
			googleDrive.SheetInfo{sheetISSUED4, "ISSUED4", true, false},
			googleDrive.SheetInfo{sheetISSUED5, "ISSUED5", true, false},
			googleDrive.SheetInfo{sheetISSUED6, "ISSUED6", true, false},
			googleDrive.SheetInfo{sheetISSUED7, "ISSUED7", true, false},
			googleDrive.SheetInfo{sheetISSUED8, "ISSUED8", true, false},
			googleDrive.SheetInfo{sheetISSUED9, "ISSUED9", true, false},
			googleDrive.SheetInfo{sheetISSUED10, "ISSUED10", true, false},
			googleDrive.SheetInfo{sheetISSUED11, "ISSUED11", true, false},
			googleDrive.SheetInfo{sheetISSUED12, "ISSUED12", true, false},
		)
	}

	invoiceSheets = append(invoiceSheets,
		googleDrive.SheetInfo{sheetERROR, "ERROR", false, false},
		googleDrive.SheetInfo{sheetVERIFICATION, "VERIFICATION", false, false},
		googleDrive.SheetInfo{sheetRATES, "RATES", false, false},
	)

	return invoiceSheets
}

func (s *InvoicingService) exportInvoices(ctx context.Context, channelID string, params InvoiceExportParams) {
	l := s.Logger(ctx)
	fs := s.Firestore(ctx)

	issuingTimestamp := time.Now()
	invoiceMonth := fmt.Sprintf("%d-%02d", params.Year, params.Month)

	l.Infof("exportInvoices: started")

	channelRef := fs.Collection("channels").Doc(channelID)

	defer func() {
		// mark channel as completed when function ends
		_, err := channelRef.Update(ctx, []firestore.Update{
			{FieldPath: []string{"complete"}, Value: true},
		})
		if err != nil {
			l.Errorf("exportInvoices error updating channelRef document, error %v", err)
		}
	}()

	var rates map[fixer.Currency]float64

	var period = time.Date(int(params.Year), time.Month(params.Month), 1, 0, 0, 0, 0, time.UTC)
	// Initialize the Google Drive service
	// Get parent folder and team drive for creating the sheet file
	targetFolder, teamDrive, writePermissionsUser := googleDrive.GetInvoicingTargetSheetDestination(params.Email, params.DevMode, params.DevDriveName)
	googleDriveService, err := googleDrive.NewGoogleDriveService(ctx, targetFolder)
	if err != nil {
		l.Errorf("exportInvoices DriveClient error %v", err)
		return
	}
	if params.CustomerID != nil {
		// Create folder for the single invoices
		for _, productType := range params.AssetTypes {
			singleInvoicesFolderID, err := googleDriveService.CreateSingleInvoicesFolder(targetFolder, productType, invoiceMonth)
			if err != nil {
				l.Errorf("exportInvoices - CreateSingleInvoicesFolder - error %v", err)
			} else {
				targetFolder = singleInvoicesFolderID
				l.Infof("exportInvoices: folderID %v for customer %v", targetFolder, *params.CustomerID)
			}
		}
	}

	now := time.Now().UTC()
	d := time.Date(period.Year(), period.Month()+1, 0, 0, 0, 0, 0, time.UTC)

	if now.Before(d) {
		d = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	}

	ratesData := fixer.HistoricalRatesInput{
		Base:    fixer.USD,
		Symbols: currencies,
		Date:    &d,
	}

	result, err := s.fixerService.HistoricalRates(ctx, &ratesData)
	if err != nil {
		l.Errorf("exportInvoices error fetching historicalRates %v", err)
		return
	}

	if result.Success {
		rates = result.Rates
	} else {
		l.Error("exportInvoices: fixer.HistoricalRates request unsuccessful")
		return
	}

	productLabels, extendedMode, err := getProductLabels(params.AssetTypes)
	if err != nil {
		l.Errorf(err.Error())
		return
	}
	// Generate the sheet name
	productsTitle := strings.Join(productLabels, "+")
	sheetName := generateSheetName(period, productsTitle, nil)
	if params.CustomerID != nil {
		customer, err := s.customersDAL.GetCustomer(ctx, *params.CustomerID)
		if err != nil {
			l.Errorf("exportInvoices error reading customer %v with error %v", *params.CustomerID, err)
			return
		}
		primaryDomain := &customer.PrimaryDomain
		sheetName = generateSheetName(period, productsTitle, primaryDomain)
	}

	var file *drive.File
	file, err = googleDriveService.CreateSheet(sheetName, targetFolder, teamDrive)
	fmt.Printf("targetFolder %v\n", targetFolder)
	if err != nil {
		l.Errorf("exportInvoices - CreateSheet error %v", err)
		return
	}

	// Grant write permissions
	err = googleDriveService.AddPermissionsToSheet(writePermissionsUser, file)
	if err != nil {
		l.Errorf("exportInvoices - AddPermissionsToSheet error %v", err)
		return
	}
	l.Infof("exportInvoices: created invoice spreadsheet %v for %v", file.Id, period.Format("January 2006"))

	invoiceSheets := s.getInvoiceSheets(extendedMode)
	// Add sheets and format widths
	spreadsheet, rowData, err := googleDriveService.ProcessInvoiceSheets(file, invoiceSheets, extendedMode)

	// Collect invoicingMonths data from Firestore
	docSnap, err := fs.Collection("billing").Doc("invoicing").Collection("invoicingMonths").Doc(period.Format("2006-01")).Get(ctx)

	if err != nil {
		l.Errorf("exportInvoices error fetching monthInvoice collection %v", err)
		return
	}

	var billingMonth BillingMonth
	if err := docSnap.DataTo(&billingMonth); err != nil {
		l.Errorf("exportInvoices error reading monthInvoice collection %v", err)
		return
	}

	customerAwsIssuedStatus := make(map[string]bool)

	var numCompleted int64

	var numIssuedAWS int64

	var numInvoices int64

	var updatedAt = billingMonth.UpdatedAt

	if params.CustomerID == nil {
		// Only count the number of invoices on full export
		for _, productType := range params.AssetTypes {
			if product, prs := billingMonth.Products[productType]; prs {
				l.Infof("export invoices will run for %v : %d", productType, product.NumInvoices)

				numInvoices += product.NumInvoices
			}
		}
	}

	if numInvoices > 0 {
		if _, err = channelRef.Update(ctx, []firestore.Update{
			{FieldPath: []string{"state"}, Value: "progress"},
		}); err != nil {
			l.Errorf("exportInvoices error updating channelRef %v", err)
			return
		}
	}

	// Get all errors for the current month
	errorDocSnaps, err := getFirestoreMonthErrorsDoc(ctx, fs, period, updatedAt)
	if err != nil {
		l.Errorf("exportInvoices error reading monthError collection %v", err)
		return
	}

	// add all monthly errors to the sheet
	err = addErrorDocSnapsToErrorSheet(errorDocSnaps, rowData)
	if err != nil {
		l.Errorf("exportInvoices addErrorDocSnapsToErrorSheet error: %v", err)
		return
	}

	l.Debugf("Filtering out for timestamp: %v", updatedAt)

	entitiesBillingIter := fs.Collection("billing").
		Doc("invoicing").
		Collection("invoicingMonths").
		Doc(period.Format("2006-01")).
		Collection("monthInvoices").
		Where("timestamp", "==", updatedAt).
		OrderBy("customer", firestore.Asc).
		Documents(ctx)
	defer entitiesBillingIter.Stop()

	for {
		docSnap, err := entitiesBillingIter.Next()

		if err == iterator.Done {
			break
		}

		if err != nil {
			l.Errorf("exportInvoices error reading billingEntities iterator, error: %v", err)
			return
		}

		if docSnap == nil {
			l.Error("docSnap is nil")
			continue
		}

		var billingDescriptor EntityBillingDescriptor
		if err := docSnap.DataTo(&billingDescriptor); err != nil {
			l.Errorf("exportInvoices error reading billing descriptor for entity %v error: %v", docSnap.Ref.ID, err)
			return
		}
		l.Debugf("exportInvoices processing invoices for customer %v and entity %v", billingDescriptor.Customer.ID, docSnap.Ref.ID)

		// Process only the requested params.CustomerID
		if params.CustomerID != nil && *params.CustomerID != billingDescriptor.Customer.ID {
			continue
		}

		if billingDescriptor.Entity == nil {
			l.Errorf("exportInvoices error reading billing descriptor for entity %v error: %v", docSnap.Ref.ID, errors.New("entity is nil"))
			return
		}

		entityDocSnap, err := billingDescriptor.Entity.Get(ctx)
		if err != nil {
			l.Errorf("exportInvoices error fetching billing descriptor entity %v error: %v", billingDescriptor.Entity.ID, err)
			return
		}

		var entity common.Entity
		if err := entityDocSnap.DataTo(&entity); err != nil {
			l.Errorf("exportInvoices error reading billing descriptor entity %v error: %v", entityDocSnap.Ref.ID, err)
			return
		}

		customer, err := s.customersDAL.GetCustomer(ctx, entity.Customer.ID)

		if err != nil {
			l.Errorf("exportInvoices error reading customer%v with error %v", entity.Customer.ID, err)
			return
		}

		// Collect only the entityInvoices of the requested asset types
		invoicesIter := docSnap.Ref.
			Collection("entityInvoices").
			Where("type", "in", params.AssetTypes).
			Where("timestamp", "==", billingDescriptor.Timestamp).
			Documents(ctx)

		// merge invoices with the same group in one
		invoices, err := entityInvoices(invoicesIter)

		l.Debugf("Entity %v - %v invoices - timestamp %v", docSnap.Ref.ID, len(invoices), billingDescriptor.Timestamp)

		if err != nil {
			l.Errorf("exportInvoices entity %v error %v", docSnap.Ref.ID, err)
			return
		}

		numIssuedEntityInvoices := 0
		wb := fb.NewAutomaticWriteBatch(fs, 100)

		// Check if all invoices are final
		allInvoicesFinal := checkIfAllInvoicesAreFinal(invoices)

		for _, invoice := range invoices {
			l.Debugf("processing export-invoice %v for customer %v", docSnap.Ref.ID, entity.Customer.ID)

			// In case not all entity's AWS invoices are final, switch the "final" flag to false for all of them
			if invoice.Type == common.Assets.AmazonWebServices && !allInvoicesFinal {
				invoice.Final = false
			}

			// add invoice rows to rowData for a spreadsheet
			invoicingMonth := fmt.Sprint(period.Format("2006-01"))

			invoiceData := InvoiceData{
				Customer:             customer,
				Entity:               &entity,
				InvoiceData:          invoice,
				EntityInvoicesDocRef: docSnap.Ref,
				IssuingTimestamp:     issuingTimestamp,
				Override:             params. Override,
				InvoicingMonth:       invoicingMonth,
			}

			sheetData := SheetData{
				Spreadsheet: spreadsheet,
				RowData:     rowData,
			}
			// Append invoice to the sheet
			isIssued := appendInvoice(l, fs, wb, rates, sheetData, invoiceData, googleDriveService)

			// Update counter of issued AWS invoices alongside with the customer:issued status mapping
			if invoice.Type == common.Assets.AmazonWebServices {
				val, ok := customerAwsIssuedStatus[entity.Customer.ID]
				if !ok {
					// Add customer to the mapping for the first time with the current isIssued status
					customerAwsIssuedStatus[entity.Customer.ID] = isIssued
				}

				if !isIssued && val {
					// Swap the status to false in case any of customer's invoices has not been issued
					customerAwsIssuedStatus[entity.Customer.ID] = false
				}

				if isIssued {
					// Update the issued invoices counter
					numIssuedAWS++
					numIssuedEntityInvoices++
				}
			}

			numCompleted++
		}

		// Update issued entity invoices counter in Firestore
		wb.Update(docSnap.Ref, []firestore.Update{
			{Path: "stats.amazon-web-services.numIssued", Value: numIssuedEntityInvoices},
		})

		// Commit Firesstore updates
		if err := wb.Commit(ctx); err != nil {
			l.Errorf("exportInvoices error committing firestore transaction for entity %v error %v", docSnap.Ref.ID, err)
			l.Error(err)
		}

		if numCompleted%10 == 0 {
			if _, err = channelRef.Update(ctx, []firestore.Update{
				{FieldPath: []string{"progress"}, Value: float64(numCompleted) / float64(numInvoices)},
			}); err != nil {
				l.Errorf("exportInvoices error updating channelRef document, error %v", err)
			}
		}
	}

	if _, err = channelRef.Update(ctx, []firestore.Update{
		{FieldPath: []string{"state"}, Value: "finalize"},
		{FieldPath: []string{"progress"}, Value: 1},
	}); err != nil {
		l.Errorf("exportInvoices error updating channelRef document, error %v", err)
	}

	// Prepare verification sheet
	numVerificationSheets := 0

	// Add invoice data to the verification sheet
	for _, s := range invoiceSheets {
		if !s.IncludeInVerification {
			continue
		}

		title, err := googleDriveService.GetSheetName(spreadsheet.Sheets, s.Id)
		if err != nil {
			l.Errorf("exportInvoices - GetSheetName error %v", err)
			return
		}
		numVerificationSheets++

		*rowData[sheetVERIFICATION] = append(*rowData[sheetVERIFICATION], []interface{}{
			title,
			fmt.Sprintf("=SUM('%s'!U:U)", title),
			fmt.Sprintf("=SUM('%s'!V:V)", title),
		})
	}

	*rowData[sheetVERIFICATION] = append(*rowData[sheetVERIFICATION], []interface{}{
		"TOTAL",
		fmt.Sprintf("=SUM(B1:B%d)", numVerificationSheets),
		fmt.Sprintf("=SUM(C1:C%d)", numVerificationSheets),
	})

	// prepare currency sheet
	for _, currency := range currencies {
		var rate float64
		if v, prs := rates[currency]; prs {
			rate = v
		} else {
			rate = 1
		}

		*rowData[sheetRATES] = append(*rowData[sheetRATES], []interface{}{
			currency,
			rate,
		})
	}

	// Add data to the sheet
	err = googleDriveService.AddDataToSpreadsheet(spreadsheet, rowData)
	if err != nil {
		l.Errorf("exportInvoices - AddDataToSpreadsheet error %v", err)
		return
	}

	if !params.DevMode {
		sn := &mailer.SimpleNotification{
			Subject:   "Invoices",
			Preheader: period.Format("January 2006"),
			Body: fmt.Sprintf(`Hello,
			<br/><br/>
			The %s invoices (%s) file you requested is ready and <a href="%s">available here</a>.`, productsTitle, period.Format("January 2006"), spreadsheet.SpreadsheetUrl),
			CCs: []string{"dror@doit.com"},
		}
		mailer.SendSimpleNotification(sn, params.Email)
	}

	// Write stats regarding AWS invoices/customers issued to Firestore
	invoicingMonthDoc := fs.Collection("billing").
		Doc("invoicing").
		Collection("invoicingMonths").
		Doc(period.Format("2006-01"))

	if _, err = invoicingMonthDoc.Update(ctx, []firestore.Update{
		{Path: "stats.amazon-web-services.numCustomersIssued", Value: countKeysWithTrueValue(customerAwsIssuedStatus)},
		{Path: "stats.amazon-web-services.numInvoicesIssued", Value: numIssuedAWS},
		{Path: "stats.amazon-web-services.numCustomersReady", Value: 0},
		{Path: "stats.amazon-web-services.numInvoicesReady", Value: 0},
	}); err != nil {
		l.Errorf("exportInvoices error updating invoicingMonthDoc document, error %v", err)
	}

	l.Infof("exportInvoices completed invoice stats %v", invoicingMonthDoc.ID)
}

func entityInvoices(iter *firestore.DocumentIterator) ([]*Invoice, error) {
	defer iter.Stop()

	invoices := make([]*Invoice, 0)
	groups := make(map[string]*Invoice)

	for {
		docSnap, err := iter.Next()
		if err == iterator.Done {
			// appent to result all of the grouped invoices
			for _, invoice := range groups {
				invoices = append(invoices, invoice)
			}

			return invoices, nil
		}

		if err != nil {
			return nil, err
		}

		var invoice Invoice
		if err := docSnap.DataTo(&invoice); err != nil {
			return nil, err
		}

		// don't reissue canceled invoices
		if invoice.CanceledAt != nil {
			continue
		}

		// Set invoice ID if not available to be able to access and update the doc later on
		if invoice.ID == nil {
			invoice.ID = &docSnap.Ref.ID
		}

		if invoice.Group != nil {
			// aggregate all rows of invoices from the same group
			gid := *invoice.Group
			if v, ok := groups[gid]; ok {
				v.Rows = append(v.Rows, invoice.Rows...)
			} else {
				groups[gid] = &invoice
			}
		} else {
			invoices = append(invoices, &invoice)
		}
	}
}

type entityParams struct {
	sheetID          int64
	priorityCompany  priority.CompanyCode
	currency         string
	country          string
	revenueAccount   string
	targetAccount    string
	isExportCustomer bool
}

// generate invoice rows for a spreadsheet in rowData and set which sheet they it should be added to.
func appendInvoice(l logger.ILogger, fs *firestore.Client, wb *fb.AutomaticWriteBatch, rates map[fixer.Currency]float64, spreadsheetData SheetData, invoiceData InvoiceData, googleDriveService googleDrive.Service) (issued bool) {
	ctx := context.Background()

	var rowData = spreadsheetData.RowData
	var spreadsheet = spreadsheetData.Spreadsheet

	var customer = invoiceData.Customer
	var entity = invoiceData.Entity
	var invoice = invoiceData.InvoiceData
	var entityInvoicesDocRef = invoiceData.EntityInvoicesDocRef
	var issuingTimestamp = invoiceData.IssuingTimestamp
	var override = invoiceData.Override
	var invoicingMonth = invoiceData.InvoicingMonth

	l.Debugf("appending invoice into sheets for customer %v invoiceDoc %v", customer.Name, entityInvoicesDocRef.ID)
	maxRowsPerInvoice := defaultInvoiceMaxLineItems
	if customer.Settings != nil && customer.Settings.Invoicing.MaxLineItems > defaultInvoiceMaxLineItems {
		maxRowsPerInvoice = customer.Settings.Invoicing.MaxLineItems
	}

	total := 0.0
	totalUSD := 0.0

	p, err := getBillingProfileParams(entity, invoice.Type)
	if err != nil {
		invoice.Final = false
		invoice.InconclusiveInvoiceReason = currencyErrorInvoice.Pointer()
		l.Debugf("getBillingProfileParams error %v", err)
		return
	}
	sheetID := p.sheetID
	entityCurrency := p.currency
	revenueAccount := p.revenueAccount
	targetAccount := p.targetAccount

	isInvoiceOnHold := customer.InvoicesOnHold[invoice.Type] != nil
	itemRows := make([][]interface{}, 0)
	creditRows := make([][]interface{}, 0)
	invoiceAdjustmentRows := make([][]interface{}, 0)
	isIssued := false

	var lineItemCurrencyMustMatchEntityCurrency bool

	// Every Priority company (IL, US, UK, etc.) has a default currency.
	// If the billing profile's (entity) currency is the same as the default currency of the Priority company
	// the billing profile belongs, then Priority will convert each line item row to the correct
	// currency if needed.
	// However, if they are not the same, then priority will just override the line item currency
	// with the billing profile's currency _without_ using proper exchange rates.
	// If such thing happen we should mark the invoice as not final and put it in the INCONCLUSIVE tab.
	defaultCurrency, ok := priority.DefaultCurrency[priority.CompanyCode(entity.PriorityCompany)]
	if ok {
		if string(defaultCurrency) != entityCurrency {
			lineItemCurrencyMustMatchEntityCurrency = true
		}
	} else {
		invoice.Final = false
		invoice.InconclusiveInvoiceReason = currencyErrorInvoice.Pointer()
		l.Debugf("currencyErrorInvoice %v - InconclusiveInvoiceReason: %v", entity.PriorityCompany, invoice.InconclusiveInvoiceReason)
	}

	var (
		hasCredits              bool
		invoiceCorrectionAmount float64
	)

	for _, row := range invoice.Rows {
		currency := fixer.FromString(row.Currency)
		if rate, prs := rates[currency]; prs {
			totalUSD += row.Total / rate
		} else {
			totalUSD += row.Total
		}

		hasCredits = hasCredits || row.Rank == CreditRank

		// Convert USD 'total' cloud spend and 'unitPrice' of draft invoices
		// to the currency specified in the billing profile
		if entityCurrency != "" && row.Currency == string(fixer.USD) {
			if row.Type == common.Assets.GoogleCloud ||
				row.Type == utils.LookerType ||
				row.Type == common.Assets.GoogleCloudStandalone ||
				row.Type == common.Assets.AmazonWebServices ||
				row.Type == common.Assets.AmazonWebServicesStandalone ||
				row.Type == common.Assets.MicrosoftAzure ||
				row.Type == utils.NavigatorType ||
				row.Type == utils.SolveType ||
				row.Type == utils.SolveAcceleratorType ||
				row.Rank == InvoiceAdjustmentRank {
				if entityCurrency != string(fixer.ILS) && fixer.SupportedCurrency(entityCurrency) { // ILS is converted by Priority
					currency := fixer.FromString(entityCurrency)
					if r, ok := rates[currency]; ok {
						row.PPU *= r
						row.Total *= r
						row.Currency = string(currency)
					}
				}
			}
		}

		if lineItemCurrencyMustMatchEntityCurrency && row.Currency != entityCurrency {
			invoice.Final = false
			invoice.InconclusiveInvoiceReason = currencyErrorInvoice.Pointer()
			l.Debugf("lineItemCurrencyMustMatchEntityCurrency %v - InconclusiveInvoiceReason: %v", entity.PriorityCompany, invoice.InconclusiveInvoiceReason)
		}

		// Total in currency
		total += row.Total
	}

	// Qualify customers with total in range (-0.01, 0.01) for adding correction row
	addInvoiceCorrectionRow := math.Abs(total) < minTotal &&
		(invoice.Type == common.Assets.AmazonWebServices || invoice.Type == common.Assets.GoogleCloud)

	if addInvoiceCorrectionRow {
		invoice.Final = invoice.Final && invoice.InconclusiveInvoiceReason == nil
		exchRate := 1.0

		// ILS is the only currency that does not need conversion
		if entityCurrency != "" && entityCurrency != string(fixer.ILS) && fixer.SupportedCurrency(entityCurrency) {
			currency := fixer.FromString(entityCurrency)
			if r, ok := rates[currency]; ok {
				exchRate = r
			}
		}

		// Ensure that total + correctionAmount is NOT in interval (-0.01, 0.01)
		// If the total is in range (0. 0.01), correctionAmount + total must be equal 0.01
		// Otherwise, it the total is in range (-0.01,0], correctionAmount + total must be equal -0.01
		if hasCredits || total < 0 {
			// Example: total = -0.007, correctionAmount = -0.003
			invoiceCorrectionAmount = -0.01 - total
		} else {
			// Example: total = 0.007, correctionAmount = 0.003
			invoiceCorrectionAmount = 0.01 - total
		}

		total += invoiceCorrectionAmount
		totalUSD += (invoiceCorrectionAmount / exchRate)
	}

	rowsTypeMap := make(map[string][]*domain.InvoiceRow)
	for _, row := range invoice.Rows {
		row.Type = mapAssetTypeToSpreadsheetAssetType(row.Type)
		rowsTypeMap[row.Type] = append(rowsTypeMap[row.Type], row)
	}

	// Change invoice type if it's Navigator or Solve
	invoiceType := mapAssetTypeToSpreadsheetAssetType(invoice.Type)

	itemRows = append(itemRows, []interface{}{
		1,
		entity.PriorityID,
		invoice.Date.Format(shortDateFormat),
		invoice.Details,
		invoice.ID,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		entity.Name,
		total,
		entity.Currency,
		invoiceType,
		totalUSD,
		nil,
		customer.PrimaryDomain,
	})

	// Write line item
	for t, typeRows := range rowsTypeMap {
		sort.Slice(typeRows, func(i, j int) bool {
			if typeRows[i].Type != typeRows[j].Type {
				return typeRows[i].Type < typeRows[j].Type
			}

			if typeRows[i].Rank != typeRows[j].Rank {
				return typeRows[i].Rank < typeRows[j].Rank
			}

			return typeRows[i].Total > typeRows[j].Total
		})

		// generating additional rows for credits, adjustments etc.
		var (
			extras  *domain.InvoiceRow
			numRows int
		)

		AmazonWebServicesSKU, GoogleCloudSKU := getSKUExtras(*entity)
		switch t {
		case common.Assets.GoogleCloud:
			extras = &domain.InvoiceRow{
				Description: "Google Cloud",
				Details:     "Additional projects",
				Quantity:    1,
				PPU:         0.0,
				SKU:         GoogleCloudSKU,
				Rank:        1,
				Type:        common.Assets.GoogleCloud,
			}
		case common.Assets.AmazonWebServices:
			extras = &domain.InvoiceRow{
				Description: "Amazon Web Services",
				Details:     "Additional accounts",
				Quantity:    1,
				PPU:         0.0,
				SKU:         AmazonWebServicesSKU,
				Rank:        1,
				Type:        common.Assets.AmazonWebServices,
			}
		case common.Assets.MicrosoftAzure:
			extras = &domain.InvoiceRow{
				Description: "Microsoft Azure",
				Details:     "Additional Subscriptions",
				Quantity:    1,
				PPU:         0.0,
				SKU:         MicrosoftAzureSKU,
				Rank:        1,
				Type:        common.Assets.MicrosoftAzure,
			}
		default:
		}

		if extras != nil {
			if entityCurrency == string(fixer.ILS) {
				// ILS is converted by Priority
				extras.Currency = string(fixer.USD)
			} else {
				extras.Currency = string(fixer.FromString(entityCurrency))
			}
		}

		for _, row := range typeRows {
			isTaggedRow := row.Tags != nil && len(row.Tags) > 0
			if extras != nil && !isTaggedRow && row.Rank == 1 && numRows >= maxRowsPerInvoice {
				extras.PPU += row.Total
				if extras.DetailsSuffix == nil && row.DetailsSuffix != nil {
					extras.DetailsSuffix = row.DetailsSuffix
				}
			} else {
				// Set SKU for the row
				row.SKU = getSKU(*entity, row.Type, row.SKU)

				var discount *float64
				if row.Discount > 0 {
					discount = &row.Discount
				}

				currency := string(fixer.FromString(row.Currency))

				var details string
				if row.DetailsSuffix != nil {
					details = row.Details + *row.DetailsSuffix
				} else {
					details = row.Details
				}

				if isTaggedRow {
					details = details + " | " + strings.Join(row.Tags, ", ")
				}

				invoiceRow := []interface{}{
					2,
					nil,
					nil,
					nil,
					nil,
					row.SKU,
					row.Description,
					details,
					row.Quantity,
					row.PPU,
					discount,
					currency,
				}

				if row.DeferredRevenuePeriod != nil {
					invoiceRow = append(invoiceRow,
						revenueAccount,
						row.DeferredRevenuePeriod.StartDate.Format("'"+shortDateFormat),
						row.DeferredRevenuePeriod.EndDate.Format("'"+shortDateFormat),
						targetAccount,
					)
				} else {
					invoiceRow = append(invoiceRow,
						nil,
						nil,
						nil,
						nil,
					)
				}

				invoiceRow = append(invoiceRow,
					nil,
					nil,
					nil,
					row.Type,
					nil,
					convRateFormula,
				)

				switch row.Rank {
				case InvoiceAdjustmentRank:
					invoiceAdjustmentRows = append(invoiceAdjustmentRows, invoiceRow)
				case CreditRank:
					creditRows = append(creditRows, invoiceRow)
				case 1:
					if !isTaggedRow {
						numRows++
					}

					fallthrough
				default:
					itemRows = append(itemRows, invoiceRow)
				}
			}
		}

		if extras != nil && math.Abs(extras.PPU) >= minTotal {
			var details string
			if extras.DetailsSuffix != nil {
				details = extras.Details + *extras.DetailsSuffix
			} else {
				details = extras.Details
			}

			if extras.PPU < 0 {
				extras.PPU *= -1
				extras.Quantity = -1
			}

			itemRows = append(itemRows, []interface{}{
				2,
				nil,
				nil,
				nil,
				nil,
				extras.SKU,
				extras.Description,
				details,
				extras.Quantity,
				extras.PPU,
				nil, // GCP & AWS don't have invoice rows discounts
				extras.Currency,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				extras.Type,
				nil,
				convRateFormula,
			})
		}

		if isInvoiceOnHold {
			sheetID = sheetOnHold
		}

		if !invoice.Final {
			if invoice.InconclusiveInvoiceReason != nil {
				switch *invoice.InconclusiveInvoiceReason {
				case lowCostInvoice:
					sheetID = sheetLowCost
				case currencyErrorInvoice:
					sheetID = sheetCurrencyError
				default:
					sheetID = sheetINCONCLUSIVE
				}
				// Add to monthErrors collection
				errorDoc := ErrorDoc{
					CustomerRef:  entity.Customer,
					Error:        fmt.Sprint(*invoice.InconclusiveInvoiceReason),
					Timestamp:    time.Now(),
					Type:         invoice.Type,
					InvoiceMonth: invoicingMonth,
				}
				result, err := addFirestoreMonthErrorsDoc(ctx, fs, errorDoc)
				if err != nil {
					l.Errorf("exportInvoices error adding error to firestore %v", err)
				} else {
					l.Info(result)
				}
			} else {
				sheetID = sheetINCONCLUSIVE
			}
		}

		// Handle batch invoicing for AWS, Navigator and Solve
		if isInvoiceIssuable(*invoice) {
			if invoice.Final && !isInvoiceOnHold {
				isIssued = true
			}
			// If override: false, check if the invoice was already issued, if so, send it to the ISSUED tab
			// set issuedAt timestamp to the invoice and save it to Firestore
			if !override {
				// Loop over all entity invoices and count invoices with existing IssuedAt timestamp
				// Note: we should send invoices to ISSUED sheet even when they are not final
				previousInvoicedIssued, firstIssuingTimestamp, err := countPreviouslyIssuedEntityInvoices(ctx, entityInvoicesDocRef, issuingTimestamp, invoice.Type)
				if err != nil {
					l.Errorf("exportInvoices error appending invoice rows for entity %v error %v", entityInvoicesDocRef.ID, err)
					return
				}

				// Send invoice to ISSUED tab when any of the previously generated invoices has the value of IssuedAt set
				if previousInvoicedIssued > 0 {
					sheetID = getIssuedSheet(issuingTimestamp, firstIssuingTimestamp)
				} else if invoice.Final && invoice.IssuedAt == nil && !isInvoiceOnHold {
					// Save IssuedAt timestamp to Firestore for future reference to avoid issuing the same invoice again (if requesting export on the same day)
					invoiceRef := entityInvoicesDocRef.Collection("entityInvoices").Doc(*invoice.ID)
					wb.Update(invoiceRef, []firestore.Update{
						{FieldPath: []string{"issuedAt"}, Value: issuingTimestamp},
					})
				}
			}
		}

		// if invoice is on hold adding title row with info to the on hold sheet
		if isInvoiceOnHold {
			cloudOnHoldDetails := customer.InvoicesOnHold[invoice.Type]
			note := fmt.Sprintf("Customer %s put on hold by %s on %s. Reason: %s", customer.PrimaryDomain, cloudOnHoldDetails.Email, cloudOnHoldDetails.Timestamp.Format(time.RFC822), cloudOnHoldDetails.Note)

			if sheetID != sheetOnHold {
				sheetName, err := googleDriveService.GetSheetName(spreadsheet.Sheets, sheetID)
				if err != nil {
					l.Errorf("exportInvoices - GetSheetName error %v", err)
					return
				}
				note = fmt.Sprintf("%s, note: invoice %s is on sheet %s", note, *invoice.ID, sheetName)
			}

			*rowData[sheetOnHold] = append(*rowData[sheetOnHold], []interface{}{note})
		}

		*rowData[sheetID] = append(*rowData[sheetID], itemRows...)

		if len(invoiceAdjustmentRows) > 0 {
			*rowData[sheetID] = append(*rowData[sheetID], invoiceAdjustmentRows...)
		}

		if len(creditRows) > 0 {
			*rowData[sheetID] = append(*rowData[sheetID], creditRows...)
		}

		if addInvoiceCorrectionRow {
			if len(itemRows) == 0 {
				// AWS Dedicate payers with full credits may have invoice total of $0
				// and no line items, we will add a custom row showing 0 cost for the AWS accounts.
				accountsRow := []interface{}{
					2,
					nil,
					nil,
					nil,
					nil,
					extras.SKU,
					extras.Description,
					"Accounts",
					1,
					0.0,
					nil,
					extras.Currency,
					nil,
					nil,
					nil,
					nil,
					nil,
					nil,
					nil,
					extras.Type,
					nil,
					convRateFormula,
				}
				*rowData[sheetID] = append(*rowData[sheetID], accountsRow)
			}

			qty, value := utils.GetQuantityAndValue(1, invoiceCorrectionAmount)
			correctionRow := []interface{}{
				2,
				nil,
				nil,
				nil,
				nil,
				extras.SKU,
				extras.Description,
				"Correction",
				qty,
				value,
				nil,
				extras.Currency,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				nil,
				extras.Type,
				nil,
				convRateFormula,
			}
			*rowData[sheetID] = append(*rowData[sheetID], correctionRow)
		}
	}

	sheetName, err := googleDriveService.GetSheetName(spreadsheet.Sheets, sheetID)
	if err != nil {
		l.Errorf("exportInvoices - GetSheetName error %v", err)
		return
	}
	l.Debugf("added entityInvoicesDocRef %s/entityInvoices/%s in Sheet: %s", entityInvoicesDocRef.Path, *invoice.ID, sheetName)

	return isIssued
}

func countPreviouslyIssuedEntityInvoices(ctx context.Context, docRef *firestore.DocumentRef, issuingTimestamp time.Time, invoiceType string) (previouslyInvoicedIssued int, firstIssuingTimestamp time.Time, err error) {
	previouslyInvoicedIssued = 0

	invoicesIter := docRef.
		Collection("entityInvoices").
		Where("type", "==", invoiceType).
		Where("issuedAt", "!=", issuingTimestamp). // don't fetch invoices issued in the current issuingTimestamp
		Documents(ctx)

	invoices, err := entityInvoices(invoicesIter)
	if err != nil {
		return previouslyInvoicedIssued, firstIssuingTimestamp, err
	}

	for _, invoice := range invoices {
		// An invoice is considered as previously issued when IssuedAt exists
		if invoice.IssuedAt != nil {
			previouslyInvoicedIssued++

			if firstIssuingTimestamp.IsZero() || invoice.IssuedAt.Before(firstIssuingTimestamp) {
				firstIssuingTimestamp = *invoice.IssuedAt
			}
		}
	}

	return previouslyInvoicedIssued, firstIssuingTimestamp, nil
}

func checkDevModeAllowed(ctx context.Context, fs *firestore.Client) (isAllowed bool, devExportDrive string, err error) {

	docSnap, err := fs.Doc("app/invoice-dev-mode").Get(ctx)

	if err != nil {
		return false, "", errors.New("missing configuration for dev invoice, please enable dev invoice export and add drive details")
	}

	var gcpFlexsaveInvoicingVersion GcpFlexsaveInvoicingVersion
	if err := docSnap.DataTo(&gcpFlexsaveInvoicingVersion); err != nil {
		return false, "", errors.New("missing configuration for dev invoice, please enable dev invoice export and add drive details")
	}

	if !gcpFlexsaveInvoicingVersion.AllowDevExport {
		return false, "", errors.New("missing configuration for dev invoice, please enable dev invoice export and add drive details")
	}

	if gcpFlexsaveInvoicingVersion.InvoiceExportDrive == "" {
		return false, "", errors.New("missing configuration for dev invoice, please add export drive details")
	}

	return true, gcpFlexsaveInvoicingVersion.InvoiceExportDrive, nil
}

func countKeysWithTrueValue(mapping map[string]bool) int {
	var counter int

	for _, val := range mapping {
		if val {
			counter++
		}
	}

	return counter
}

func checkIfAllInvoicesAreFinal(invoices []*Invoice) bool {
	allFinal := true

	for _, invoice := range invoices {
		if !invoice.Final {
			allFinal = false
		}
	}

	return allFinal
}
