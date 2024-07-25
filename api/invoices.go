package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/auth"
	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/invoices"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

// Firestore invoice fields
const (
	InvoiceFieldDate     = "IVDATE"
	InvoiceFieldDueDate  = "PAYDATE"
	InvoiceFieldCanceled = "CANCELED"
	InvoiceFieldProducts = "PRODUCTS"
)

// swagger:parameters idOfInvoice
type InvoiceRequest struct {
	// invoice number, identifying the invoice
	// in:path
	ID string `json:"id"`
}

type InvoicesList struct {
	// Page token, returned by a previous call, to request the next page of results
	PageToken string `json:"pageToken,omitempty"`
	// Invoices rows count
	RowCount int `json:"rowCount"`
	// Array of Invoices
	Invoices []InvoiceListItem `json:"invoices"`
}

type Entity struct {
	Name string `json:"name" firestore:"name"`
}

type Metadata struct {
	Entity Entity `json:"entity" firestore:"entity"`
}

type InvoiceListItem struct {
	// invoice number, identifying the invoice
	ID string `json:"id"`
	// The time when this invoice was issued, in milliseconds since the epoch.
	Date int64 `json:"invoiceDate"`
	// platform
	// enum:  google-cloud,amazon-web-services,microsoft-azure,g-suite,office-365,superquery
	Platform string `json:"platform"`
	// The last day to pay the invoice, in milliseconds since the epoch
	DueDate int64 `json:"dueDate"`
	// Status
	//enum: OPEN,PAST DUE,PAID
	Status string `json:"status"`
	// Total invoiced amount
	TOTPRICE float64 `json:"totalAmount"`
	// Invoice balance to be paid
	BalanceAmount float64 `json:"balanceAmount"`
	// Invoice currency
	// enum: USD,GBP,AUD,EUR,ILS,CAD,DKK,NOK,SEK,BRL,SGD,MXN,CHF,MYR,TWD,EGP,ZAR,JPY,IDR
	Currency string `json:"currency"`
	// Link to the downloadable document
	URL string `json:"url"`
}

type Invoice struct {
	// invoice number, identifying the invoice (e.g. "IN204005474")
	ID string `json:"id"`
	// The time when this invoice was issued, in milliseconds since the epoch.
	InvoiceDate int64 `json:"invoiceDate"`
	// Platform can be "google-cloud", "amazon-web-services", "microsoft-azure", "g-suite", "office-365", "superquery"
	Platform string `json:"platform"`
	// due date, in milliseconds since the epoch.
	DueDate int64 `json:"dueDate"`
	// can be either OPEN, PAST DUE, or PAID
	Status string `json:"status"`
	// total invoiced amount
	TotalAmount float64 `json:"totalAmount"`
	// invoice balance to be paid
	BalanceAmount float64 `json:"balanceAmount"`
	// Invoice currency, can be USD, GBP, AUD, EUR, ILS, CAD, DKK, NOK, SEK, BRL, SGD, MXN, CHF, MYR, TWD, EGP, ZAR, JPY, IDR
	Currency string `json:"currency"`
	// link to the downloadable document
	URL string `json:"url"`
	// Invoice line items
	LineItems []*ListItem `json:"lineItems"`
}

type ListItem struct {
	Type        string  `json:"type" firestore:"TYPE"`
	Description string  `json:"description" firestore:"PDES"`
	Price       float64 `json:"price" firestore:"PRICE"`
	Quantity    float64 `json:"qty" firestore:"QUANT"`
	Currency    string  `json:"currency" firestore:"ICODE"`
	Details     string  `json:"details" firestore:"DETAILS"`
}

// swagger:parameters idOfInvoices
type InvoicesRequest struct {
	// The maximum number of results to return in a single page. Leverage the page tokens to iterate through the entire collection.
	// default: 50
	MaxResults int `json:"maxResults"`
	// Page token, returned by a previous call, to request the next page of results
	PageToken string `json:"pageToken,omitempty"`
	// An expression for filtering the results of the request. The syntax is "key:[<value>]". Multiple filters can be connected using a pipe |. Note that using different keys in the same filter results in “AND,” while using the same key multiple times in the same filter results in “OR”.
	Filter string `json:"filter"`
	// Min value for the invoice creation time, in milliseconds since the POSIX epoch. If set, only invoices created after or at this timestamp are returned.
	MinCreationTime int64 `json:"minCreationTime"`
	// Max value for the invoice creation time, in milliseconds since the POSIX epoch. If set, only invoices created before or at this timestamp are returned.
	MaxCreationTime int64 `json:"maxCreationTime"`
}

func ListInvoices(ctx *gin.Context, conn *connection.Connection) {
	// Verify customerID path param
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)
	email := ctx.GetString("email")

	now := time.Now().UTC()
	l := logger.FromContext(ctx)

	l.SetLabels(map[string]string{
		logger.LabelEmail: email,
	})

	invoicesRequest := InvoicesRequest{
		MaxResults:      50,
		MinCreationTime: -1,
		MaxCreationTime: -1,
	}

	maxResultsStr := ctx.Request.URL.Query().Get("maxResults")
	if maxResultsStr != "" {
		var err error

		invoicesRequest.MaxResults, err = strconv.Atoi(maxResultsStr)
		if err != nil {
			AbortMsg(ctx, http.StatusBadRequest, errors.New(ErrorParam+"maxResults"), ErrorParam+"maxResults")
			return
		}

		if invoicesRequest.MaxResults > maxResultsLimit {
			l.Error(ErrorParamMaxResultRange)
			AbortMsg(ctx, http.StatusBadRequest, errMaxResultRange, errMaxResultRange.Error())

			return
		}

		if invoicesRequest.MaxResults <= 0 {
			invoicesRequest.MaxResults = maxResultsLimit
		}
	}

	minCreationTimeStr := ctx.Request.URL.Query().Get("minCreationTime")
	if minCreationTimeStr != "" {
		var err error

		invoicesRequest.MinCreationTime, err = strconv.ParseInt(minCreationTimeStr, 10, 64)
		if err != nil {
			AbortMsg(ctx, http.StatusBadRequest, errors.New(ErrorParam+"minResults"), ErrorParam+"minResults")
			return
		}
	}

	maxCreationTimeStr := ctx.Request.URL.Query().Get("maxCreationTime")
	if maxCreationTimeStr != "" {
		var err error

		invoicesRequest.MaxCreationTime, err = strconv.ParseInt(maxCreationTimeStr, 10, 64)
		if err != nil {
			AbortMsg(ctx, http.StatusBadRequest, errors.New(ErrorParam+"maxCreationTime"), ErrorParam+"maxCreationTime")
			return
		}
	}

	encodedPageToken := ctx.Request.URL.Query().Get("pageToken")

	decodedPageToken, err := customerapi.DecodePageToken(encodedPageToken)
	if err != nil {
		AbortMsg(ctx, http.StatusInternalServerError, err, ErrorInternalError)
		return
	}

	pageToken := decodedPageToken
	filterStr := ctx.Request.URL.Query().Get("filter")

	fs := conn.Firestore(ctx)

	customerRef := fs.Collection("customers").Doc(customerID)

	invoicesRef := fs.Collection("invoices")
	orderBy := InvoiceFieldDate
	invoicesQuery := invoicesRef.Where("customer", "==", customerRef).Where(InvoiceFieldCanceled, "==", false).Limit(invoicesRequest.MaxResults)

	checkCreationDateAfterExec := false // since  firestore cannot have < > for two fields, if there are 2 fields and minCreation and maxCreation will be checked after execution.

	var invoicesList InvoicesList
	invoicesList.Invoices = make([]InvoiceListItem, 0)

	if filterStr != "" {
		// parse the filter string and split into array. loop over each key:
		filterArr := strings.Split(filterStr, "|")

		var platformArr []string

		var dueDateArr []string

		var statusArr []string

		for _, param := range filterArr {
			splitParam := strings.Split(param, ":")
			if len(splitParam) == 2 {
				key := splitParam[0]
				value := splitParam[1]

				switch key {
				case "platform":
					platformArr = append(platformArr, value)
				case "dueDate":
					dueDateArr = append(dueDateArr, value)
				case "status":
					statusArr = append(statusArr, value)
				default:
					l.Error(ErrorUnknownFilterKey + param)
					AbortMsg(ctx, 400, errors.New(ErrorUnknownFilterKey+param), ErrorUnknownFilterKey+param)

					return
				}
			} else {
				l.Error(ErrorUnknownFilterKey + param)
				AbortMsg(ctx, 400, errors.New(ErrorUnknownFilterKey+param), ErrorUnknownFilterKey+param)

				return
			}
		}

		if len(platformArr) > 0 {
			invoicesQuery = invoicesQuery.Where(InvoiceFieldProducts, "array-contains-any", platformArr)
		}

		if len(dueDateArr) > 0 {
			t, err := common.MsToTime(dueDateArr[0])
			if err != nil {
				l.Error(err)
				AbortMsg(ctx, 400, err, ErrorParam+"dueDate")

				return
			}

			invoicesQuery = invoicesQuery.Where(InvoiceFieldDueDate, "==", t)
		}

		if len(statusArr) > 0 {
			switch statusArr[0] {
			case "OPEN":
				invoicesQuery = invoicesQuery.Where("PAID", "==", false)
				invoicesQuery = invoicesQuery.Where(InvoiceFieldDueDate, ">=", now)
				orderBy = InvoiceFieldDueDate
				checkCreationDateAfterExec = true
			case "PAID":
				invoicesQuery = invoicesQuery.Where("PAID", "==", true)
			case "PAST DUE":
				invoicesQuery = invoicesQuery.Where("PAID", "==", false)
				invoicesQuery = invoicesQuery.Where(InvoiceFieldDueDate, "<=", now)
				orderBy = InvoiceFieldDueDate
				checkCreationDateAfterExec = true
			default:
				l.Error(errors.New(ErrorParam + "status"))
				AbortMsg(ctx, 400, errors.New(ErrorParam+"status"), ErrorParam+"status")

				return
			}
		}
	}

	invoicesQuery = invoicesQuery.OrderBy(orderBy, firestore.Desc)

	if invoicesRequest.MinCreationTime != -1 && !checkCreationDateAfterExec {
		t, err := common.MsToTime(minCreationTimeStr)
		if err != nil {
			l.Error(err)
			AbortMsg(ctx, 400, err, ErrorParam+"minCreationTime")

			return
		}

		invoicesQuery = invoicesQuery.Where(InvoiceFieldDate, ">=", t)
		l.Infof("minCreationTime: %#v", t)
	}

	if invoicesRequest.MaxCreationTime != -1 && !checkCreationDateAfterExec {
		t, err := common.MsToTime(maxCreationTimeStr)
		if err != nil {
			l.Error(err)
			AbortMsg(ctx, 400, err, ErrorParam+"maxCreationTime")

			return
		}

		invoicesQuery = invoicesQuery.Where(InvoiceFieldDate, "<=", t)
		l.Infof("maxCreationTime: %#v", t)
	}
	//paging
	if pageToken != "" {
		docSnap, err := fs.Collection("invoices").Doc(pageToken).Get(ctx)
		invoicesQuery = invoicesQuery.StartAfter(docSnap)

		if err != nil {
			AbortMsg(ctx, http.StatusNotFound, err, ErrorPageTokenNotFound)
			return
		}
	}

	// invoices := make([]InvoiceListItem, 0)
	docSnaps, err := invoicesQuery.Documents(ctx).GetAll()
	// invoicesOutput := []Invoice{}
	if err != nil {
		l.Error(err)
		AbortMsg(ctx, http.StatusInternalServerError, err, ErrorInternalError)

		return
	}

	for _, docSnap := range docSnaps {
		var fullInvoice invoices.FullInvoice

		if err := docSnap.DataTo(&fullInvoice); err != nil {
			l.Error(err)
			AbortMsg(ctx, http.StatusInternalServerError, err, ErrorInternalError)

			return
		}

		skipInvoice := false

		invoice := InvoiceListItem{
			ID:            fullInvoice.ID, //docSnap.Ref.ID,
			Date:          fullInvoice.Date.Unix() * 1000,
			DueDate:       fullInvoice.PayDate.Unix() * 1000,
			TOTPRICE:      fullInvoice.Total,
			BalanceAmount: fullInvoice.Debit,
			Currency:      fullInvoice.Currency,
		}
		if len(fullInvoice.Products) > 0 {
			invoice.Platform = fullInvoice.Products[0]
		}

		if fullInvoice.Paid {
			invoice.Status = "PAID"
		} else {
			if invoice.DueDate <= now.Unix()*1000 {
				invoice.Status = "PAST DUE"
			} else {
				invoice.Status = "OPEN"
			}
		}

		if len(fullInvoice.ExternalFilesSubForm) > 0 && fullInvoice.ExternalFilesSubForm[0] != nil && fullInvoice.ExternalFilesSubForm[0].URL != nil {
			invoice.URL = *fullInvoice.ExternalFilesSubForm[0].URL
		}

		if checkCreationDateAfterExec {
			if invoicesRequest.MinCreationTime != -1 {
				t, err := common.MsToTime(minCreationTimeStr)
				if err != nil {
					l.Error(err)
					AbortMsg(ctx, http.StatusBadRequest, err, ErrorParam+"MinCreationTime")

					return
				}

				if invoice.Date < t.Unix()*1000 {
					skipInvoice = true
				}
			}

			if invoicesRequest.MaxCreationTime != -1 {
				t, err := common.MsToTime(maxCreationTimeStr)
				if err != nil {
					l.Error(err)
					AbortMsg(ctx, 400, err, ErrorParam+"MaxCreationTime")

					return
				}

				if invoice.Date > t.Unix()*1000 {
					skipInvoice = true
				}
			}
		}

		if !skipInvoice {
			invoicesList.Invoices = append(invoicesList.Invoices, invoice)
		}
	}

	if len(docSnaps) == invoicesRequest.MaxResults {
		invoicesList.PageToken = customerapi.EncodePageToken(docSnaps[len(docSnaps)-1].Ref.ID)
	} else {
		invoicesList.PageToken = ""
	}

	invoicesList.RowCount = len(invoicesList.Invoices)
	ctx.JSON(http.StatusOK, invoicesList)
}

func GetInvoice(ctx *gin.Context, conn *connection.Connection) {
	email := ctx.GetString("email")

	now := time.Now().UTC()

	l := logger.FromContext(ctx)
	l.SetLabels(map[string]string{
		logger.LabelEmail: email,
	})

	// get the issue id
	invoiceID := ctx.Param("id")
	if invoiceID == "" {
		l.Info(ErrorInvoiceNotFound)
		AbortMsg(ctx, http.StatusBadRequest, errors.New(ErrorInvoiceNotFound), ErrorInvoiceNotFound)

		return
	}

	fs := conn.Firestore(ctx)

	// get customerRef
	customerID := ctx.GetString(auth.CtxKeyVerifiedCustomerID)
	customerRef := fs.Collection("customers").Doc(customerID)

	invoicesRef := fs.Collection("invoices")
	invoicesQuery := invoicesRef.Where("customer", "==", customerRef).Where(InvoiceFieldCanceled, "==", false).Where("IVNUM", "==", invoiceID).Limit(1)

	docSnaps, err := invoicesQuery.Documents(ctx).GetAll()
	if docSnaps == nil || err != nil {
		AbortMsg(ctx, 404, err, ErrorNotFound)
		return
	}

	invoiceSnap := docSnaps[0]

	var invoiceInternal invoices.FullInvoice
	if err := invoiceSnap.DataTo(&invoiceInternal); err != nil {
		l.Error(err)
		AbortMsg(ctx, 500, err, ErrorInternalError)

		return
	}
	// fill the output invoice struct from firebase response:
	invoice := Invoice{
		ID:            invoiceInternal.ID, // invoiceSnap.Ref.ID,
		InvoiceDate:   invoiceInternal.Date.Unix() * 1000,
		DueDate:       invoiceInternal.PayDate.Unix() * 1000,
		TotalAmount:   invoiceInternal.Total,
		BalanceAmount: invoiceInternal.Debit,
		Currency:      invoiceInternal.Currency,
	}
	if len(invoiceInternal.Products) > 0 {
		invoice.Platform = invoiceInternal.Products[0]
	}

	if invoiceInternal.Paid {
		invoice.Status = "PAID"
	} else {
		if invoiceInternal.PayDate.Unix() <= now.Unix() {
			invoice.Status = "PAST DUE"
		} else {
			invoice.Status = "OPEN"
		}
	}

	if invoiceInternal.ExternalFilesSubForm != nil && invoiceInternal.ExternalFilesSubForm[0] != nil && invoiceInternal.ExternalFilesSubForm[0].URL != nil {
		invoice.URL = string(*invoiceInternal.ExternalFilesSubForm[0].URL)
	}

	for _, invoiceItem := range invoiceInternal.InvoiceItems {
		invoiceItem := ListItem{
			Type:        invoiceItem.Type,
			Description: invoiceItem.Description,
			Price:       invoiceItem.Price,
			Quantity:    invoiceItem.Quantity,
			Currency:    invoiceItem.Currency,
			Details:     invoiceItem.Details,
		}
		invoice.LineItems = append(invoice.LineItems, &invoiceItem)
	}

	ctx.JSON(http.StatusOK, invoice)
}
