package invoicing

import (
	// "github.com/doitintl/auth/firebaseauth/domain"
	"context"
	"errors"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/utils"
	"github.com/doitintl/hello/scheduled-tasks/priority"
	"github.com/doitintl/hello/scheduled-tasks/slice"
	"google.golang.org/api/sheets/v4"
)

type InvoiceData struct {
	Customer             *common.Customer
	Entity               *common.Entity
	InvoiceData          *Invoice
	EntityInvoicesDocRef *firestore.DocumentRef
	IssuingTimestamp     time.Time
	Override             bool
	InvoicingMonth       string
}
type SheetData struct {
	Spreadsheet *sheets.Spreadsheet
	RowData     map[int64]*[][]interface{}
}
type ErrorDoc struct {
	CustomerRef  *firestore.DocumentRef
	Error        string
	Timestamp    time.Time
	Type         string
	InvoiceMonth string
}
type InvoiceDefaultsLedger struct {
	TargetAccount  string
	RevenueAccount string
}

var defaultLedgerMap = map[string]InvoiceDefaultsLedger{
	utils.SolveType: {
		TargetAccount:  "429",
		RevenueAccount: "108-0",
	},
	utils.SolveAcceleratorType: {
		TargetAccount:  "431",
		RevenueAccount: "108-0",
	},
	utils.NavigatorType: {
		TargetAccount:  "430",
		RevenueAccount: "108-0",
	},
	"default": {
		TargetAccount:  "404-1",
		RevenueAccount: "108-0",
	},
}

// mapPriorityCompanyToSheetID maps priority company codes to sheet IDs
func mapPriorityCompanyToSheetID(companyCode priority.CompanyCode) int64 {
	switch companyCode {
	case priority.CompanyCodeUSA:
		return sheetUS
	case priority.CompanyCodeUK:
		return sheetUK
	case priority.CompanyCodeAUS:
		return sheetAU
	case priority.CompanyCodeDE:
		return sheetDE
	case priority.CompanyCodeFR:
		return sheetFR
	case priority.CompanyCodeNL:
		return sheetNL
	case priority.CompanyCodeCH:
		return sheetCH
	case priority.CompanyCodeCA:
		return sheetCA
	case priority.CompanyCodeSE:
		return sheetSE
	case priority.CompanyCodeES:
		return sheetES
	case priority.CompanyCodeIE:
		return sheetIE
	case priority.CompanyCodeEE:
		return sheetEE
	case priority.CompanyCodeSG:
		return sheetSG
	case priority.CompanyCodeJP:
		return sheetJP
	case priority.CompanyCodeID:
		return sheetIND
	case priority.CompanyCodeISR:
		return sheetISR
	default:
		return sheetINCONCLUSIVE
	}
}

// getDefaultAccounts returns default revenue and target accounts based on invoice type
func getDefaultRevenueAndTargetAccounts(invoiceType string) (string, string) {
	if defaults, ok := defaultLedgerMap[invoiceType]; ok {
		return defaults.RevenueAccount, defaults.TargetAccount
	}
	// If the invoice type is not found in the map, use the defaults for "default" key
	return defaultLedgerMap["default"].RevenueAccount, defaultLedgerMap["default"].TargetAccount
}

/*
Function getBillingProfileParams allocates the correct sheet ID
based on the priority company code. It also sets the revenue and target accounts
for ledger mapping in annual invoices.
*/
func getBillingProfileParams(entity *common.Entity, invoiceType string) (entityParams, error) {
	p := entityParams{
		sheetID:          sheetINCONCLUSIVE,
		priorityCompany:  "",
		currency:         "",
		country:          "",
		revenueAccount:   "",
		targetAccount:    "",
		isExportCustomer: true,
	}
	// Check if entity data is available
	if entity == nil || entity.Country == nil || entity.Currency == nil {
		errorMsg := "entity data is missing or incomplete - country: " + *entity.Country + ", currency: " + *entity.Currency
		return p, errors.New(errorMsg)
	}

	// If entity data is available, set parameters based on entity data
	p.currency = *entity.Currency
	p.country = *entity.Country
	p.priorityCompany = priority.CompanyCode(entity.PriorityCompany)
	p.isExportCustomer = priority.IsExportCountry(p.priorityCompany, p.country)
	// Map priority company codes to sheet IDs
	p.sheetID = mapPriorityCompanyToSheetID(p.priorityCompany)

	// Handle specific cases based on priority company
	if p.priorityCompany == priority.CompanyCodeISR {
		p.revenueAccount, p.targetAccount = getISRRevenueAndTargetAccounts(p.isExportCustomer, invoiceType)
	} else {
		// For non-ISR companies, use default ledger mapping
		p.revenueAccount, p.targetAccount = getDefaultRevenueAndTargetAccounts(invoiceType)
	}

	// Move to next sheet for non-inconclusive cases
	if p.sheetID != sheetINCONCLUSIVE && p.isExportCustomer {
		// e.g NO_ISR
		p.sheetID++
	}

	return p, nil
}

// Firestore functions
func getFirestoreMonthErrorsDoc(ctx context.Context, fs *firestore.Client, period time.Time, updatedAt time.Time) ([]*firestore.DocumentSnapshot, error) {
	errorDocSnaps, err := fs.Collection("billing").
		Doc("invoicing").
		Collection("invoicingMonths").
		Doc(period.Format("2006-01")).
		Collection("monthErrors").
		Where("timestamp", "==", updatedAt).
		OrderBy("customer", firestore.Asc).
		Documents(ctx).GetAll()
	if err != nil {
		errorMsg := fmt.Sprintf("error getting error documents for period %s and updated at %s: %v", period.Format("2006-01"), updatedAt, err)
		return nil, errors.New(errorMsg)
	}
	return errorDocSnaps, nil
}

func addFirestoreMonthErrorsDoc(ctx context.Context, fs *firestore.Client, errorDoc ErrorDoc) (string, error) {
	// Create a reference to the billing month document
	billingMonthRef := fs.Collection("billing").Doc("invoicing").Collection("invoicingMonths").Doc(errorDoc.InvoiceMonth)

	// Create a reference to the monthErrors collection within the billing month
	billingMonthErrors := billingMonthRef.Collection("monthErrors")

	// Create a map containing the data for the error document
	errorData := map[string]interface{}{
		"customer":  errorDoc.CustomerRef,
		"error":     errorDoc.Error,
		"timestamp": time.Now(),
		"type":      errorDoc.Type,
	}

	// Add the error document to the monthErrors collection
	newErrorDoc, _, err := billingMonthErrors.Add(ctx, errorData)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Error adding document to firestore: %v", err))
	}

	return fmt.Sprintf("Error document %v/%v added to firestore", billingMonthErrors, newErrorDoc.ID), nil

}

func addErrorDocSnapsToErrorSheet(errorDocSnaps []*firestore.DocumentSnapshot, rowData map[int64]*[][]interface{}) error {

	for _, docSnap := range errorDocSnaps {
		var errLine ErrorLine
		if err := docSnap.DataTo(&errLine); err != nil {
			errorMsg := fmt.Sprintf("processErrorDocSnaps error reading monthErrorLine document %v error %v", docSnap.Ref.ID, err)
			return errors.New(errorMsg)
		}

		*rowData[sheetERROR] = append(*rowData[sheetERROR], []interface{}{
			fmt.Sprintf("https://console.doit.com/customers/%s", errLine.Customer.ID),
			errLine.Type,
			errLine.Error,
		})
	}
	return nil
}

func generateSheetName(period time.Time, productsTitle string, primaryDomain *string) string {
	if primaryDomain != nil {
		// Add primary domain to the sheet name - Single customer export
		return fmt.Sprintf("[%s] %s Invoices - %s", productsTitle, period.Format("January 2006"), *primaryDomain)
	}
	// Default sheet name - Multiple customers export
	return fmt.Sprintf("[%s] %s Invoices", productsTitle, period.Format("January 2006"))
}

func isInvoiceIssuable(invoice Invoice) bool {
	return slice.Contains([]string{common.Assets.AmazonWebServices, utils.NavigatorType, utils.SolveType, utils.SolveAcceleratorType}, invoice.Type)
}

func getIssuedSheet(issuingTimestamp time.Time, firstIssuingTimestamp time.Time) int64 {
	var (
		sheetID         int64
		firstIssuingDay int
	)

	// first issuing day is either current issuingTimestamp's day or the day of firstIssuingTimestamp if exists
	if !firstIssuingTimestamp.IsZero() {
		firstIssuingDay = firstIssuingTimestamp.Day()
	} else {
		firstIssuingDay = issuingTimestamp.Day()
	}

	switch {
	case firstIssuingDay == 3:
		sheetID = sheetISSUED3
	case firstIssuingDay == 4:
		sheetID = sheetISSUED4
	case firstIssuingDay == 5:
		sheetID = sheetISSUED5
	case firstIssuingDay == 6:
		sheetID = sheetISSUED6
	case firstIssuingDay == 7:
		sheetID = sheetISSUED7
	case firstIssuingDay == 8:
		sheetID = sheetISSUED8
	case firstIssuingDay == 9:
		sheetID = sheetISSUED9
	case firstIssuingDay == 10:
		sheetID = sheetISSUED10
	case firstIssuingDay == 11:
		sheetID = sheetISSUED11
	case firstIssuingDay == 12:
		sheetID = sheetISSUED12
	default:
		sheetID = sheetISSUED
	}

	return sheetID
}

func getProductLabels(assetTypes []string) ([]string, bool, error) {
	productLabels := make([]string, 0)
	extendedMode := false // ISSUED tabs are added to spreadsheet in the extended mode

	for _, t := range assetTypes {
		switch t {
		case common.Assets.GSuite:
			productLabels = append(productLabels, "GSuite")
		case common.Assets.Office365:
			productLabels = append(productLabels, "Office")
		case common.Assets.GoogleCloud:
			productLabels = append(productLabels, "GCP")
		case common.Assets.GoogleCloudStandalone:
			productLabels = append(productLabels, "Flexsave GCP")
		case common.Assets.AmazonWebServicesStandalone:
			productLabels = append(productLabels, "Flexsave AWS")
		case common.Assets.MicrosoftAzure:
			productLabels = append(productLabels, "Azure")
		case utils.LookerType:
			productLabels = append(productLabels, "Looker")
		case common.Assets.AmazonWebServices:
			productLabels = append(productLabels, "AWS")
			extendedMode = true
		case utils.NavigatorType:
			productLabels = append(productLabels, "Navigator")
			extendedMode = true
		case utils.SolveType:
			productLabels = append(productLabels, "Solve")
			extendedMode = true
		case utils.SolveAcceleratorType:
			productLabels = append(productLabels, "Solve Accelerator")
			extendedMode = true
		default:
			return nil, false, fmt.Errorf("exportInvoices failed, invalid product type %s", t)
		}
	}

	return productLabels, extendedMode, nil
}

func getSKUExtras(entity common.Entity) (string, string) {
	isChicagoCustomer := getIsChicagoBillingProfile(entity)

	if isChicagoCustomer {
		return ChicagoAmazonWebServicesSKU, ChicagoGoogleCloudSKU
	}
	return AmazonWebServicesSKU, GoogleCloudSKU
}

func getSKU(entity common.Entity, assetType string, currentSKU string) string {
	awsSKU, gcpSKU := getSKUExtras(entity)

	switch assetType {
	case common.Assets.AmazonWebServices:
		return awsSKU
	case common.Assets.GoogleCloud:
		return gcpSKU
	}
	return currentSKU
}
