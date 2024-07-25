package scripts

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/errorreporting"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type FlexsaveScripts struct {
	*connection.Connection

	service *flexsaveresold.Service
}

type BackFillRequest struct {
	Month          string
	NumberOfMonths int
	CustomerID     *string
}

func NewFlexsaveScripts(log logger.Provider, conn *connection.Connection) *FlexsaveScripts {
	service := flexsaveresold.NewService(log, conn)

	return &FlexsaveScripts{
		conn,
		service,
	}
}

// BackfillBillingLineItems creates historic FlexSave billing line items from the given month for the number of months specified
// using FlexSave savings values from relevant invoice adjustments for corresponding month.
// If customer ID is provided we will backfill for only this customer.
// Example request....
//
//	{
//		"Month": "2021-11-02",
//		"NumberOfMonths": 1
//		"CustomerID": "CFFE2847383470" (OPTIONAL)
//	}
func (h *FlexsaveScripts) BackfillBillingLineItems(ctx *gin.Context) []error {
	fs := h.Firestore(ctx)

	var requestBody BackFillRequest

	if err := ctx.BindJSON(&requestBody); err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	firstMonth := requestBody.Month
	numberOfMonths := requestBody.NumberOfMonths
	customerID := requestBody.CustomerID

	if firstMonth == "" || numberOfMonths == 0 {
		err := fmt.Errorf("missing request value, have %v but require first month and month", requestBody)
		errorreporting.AbortWithErrorReport(ctx, http.StatusBadRequest, err)

		return []error{err}
	}

	layout := "2006-01-02"

	firstMonthInstance, err := time.Parse(layout, firstMonth)
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return []error{err}
	}

	firstOfMonth := time.Date(firstMonthInstance.Year(), firstMonthInstance.Month(), 1, 0, 0, 0, 0, time.UTC)
	firstOfMonthAfterLast := firstOfMonth.AddDate(0, numberOfMonths, 0)
	customerMonthTotals := make(map[string]map[string]float64)

	if customerID == nil {
		docSnaps, err := fs.Collection("integrations").
			Doc("amazon-web-services").
			Collection("flexibleReservedInstances").
			Where("execution", "==", "autopilot-v1").
			Where("config.startDate", ">=", firstOfMonth).
			Where("config.startDate", "<", firstOfMonthAfterLast).
			Documents(ctx).
			GetAll()

		if err != nil {
			return []error{err}
		}

		for _, docSnap := range docSnaps {
			customerRef := docSnap.Data()["customer"].(*firestore.DocumentRef)
			customerMonthTotals[customerRef.ID] = make(map[string]float64)
		}
	} else {
		customerMonthTotals[*customerID] = make(map[string]float64)
	}

	var isAllFlexSave bool

	for customer := range customerMonthTotals {
		customerRef := fs.Collection("customers").Doc(customer)

		for i := int(firstMonthInstance.Month()); i < int(firstMonthInstance.Month())+numberOfMonths; i++ {
			monthString := strconv.Itoa(int(firstOfMonth.Month())) + "-" + strconv.Itoa(firstOfMonth.Year())
			adjustments, _ := flexsaveresold.GetCustomerFlexSaveInvoiceAdjustments(ctx, customerRef, isAllFlexSave, firstOfMonth)

			total := 0.0

			for _, adjustment := range adjustments {
				total += adjustment.Amount
			}

			customerMonthTotals[customer][monthString] = total
		}
	}

	for customer, months := range customerMonthTotals {
		for date, savings := range months {
			month, err := strconv.Atoi(strings.Split(date, "-")[0])
			if err != nil {
				errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
				return []error{err}
			}

			year, err := strconv.Atoi(strings.Split(date, "-")[1])
			if err != nil {
				errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
				return []error{err}
			}

			firstOfMonthAfter := time.Date(year, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC)
			lastOfMonth := firstOfMonthAfter.AddDate(0, 0, -1)
			timeInstance := lastOfMonth.Truncate(time.Hour * 24)

			if savings < 0 {
				h.service.CreateBillingLineItems(ctx, savings, timeInstance, customer, true)
			}
		}
	}

	return nil
}
