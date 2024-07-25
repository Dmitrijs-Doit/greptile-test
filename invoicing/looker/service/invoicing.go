package service

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/domain"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/utils"
	lookerDomain "github.com/doitintl/hello/scheduled-tasks/looker/domain"
	lookerUtils "github.com/doitintl/hello/scheduled-tasks/looker/utils"
)

func (s *InvoicingService) GetInvoiceRows(ctx context.Context, task *domain.CustomerTaskData, customerRef *firestore.DocumentRef, _ map[string]*common.Entity, respChan chan<- *domain.ProductInvoiceRows) {
	l := s.loggerProvider(ctx)

	res := &domain.ProductInvoiceRows{
		Type:  lookerUtils.LookerProductType,
		Rows:  make([]*domain.InvoiceRow, 0),
		Error: nil,
	}

	// Return results
	defer func() {
		respChan <- res
	}()

	contracts, err := s.contractsDAL.GetActiveCustomerContractsForProductTypeAndMonth(ctx, customerRef, task.InvoiceMonth, lookerUtils.LookerProductType)
	if err != nil {
		l.Errorf("failed to fetch looker contracts. error [%s]", err)
		res.Error = err

		return
	}

	if len(contracts) == 0 {
		// Customer has no Looker usage
		return
	}

	sanitizedInvoiceMonth := time.Date(task.InvoiceMonth.Year(), task.InvoiceMonth.Month(), task.InvoiceMonth.Day(), 0, 0, 0, 0, time.UTC)

	for _, contract := range contracts {
		if lookerUtils.IsOldFormat(*contract) {
			l.Infof("old contract format. skipping contract [%s]", contract.ID)
			continue
		}

		if contract.Entity == nil {
			err = fmt.Errorf("invalid entity for looker contract [%s], skipping", contract.ID)
			res.Error = err

			return
		}

		var properties lookerDomain.LookerContractProperties

		properties, err := properties.DecodePropertiesMapIntoStruct(contract.Properties)
		if err != nil {
			l.Errorf("invalid contract properties. error [%s]", err)
			res.Error = err

			return
		}

		rows := extractRowsFromLookerContract(properties, sanitizedInvoiceMonth, *contract)

		res.Rows = append(res.Rows, rows...)
	}
}

func extractRowsFromLookerContract(properties lookerDomain.LookerContractProperties, invoiceMonth time.Time, contract pkg.Contract) []*domain.InvoiceRow {
	rows := make([]*domain.InvoiceRow, 0)

	if properties.Skus != nil {
		for _, sku := range properties.Skus {
			if IsInvoiceMonthBillableForSku(sku, properties, invoiceMonth) {
				monthsToBill := 1
				if properties.InvoiceFrequency != lookerUtils.MonthlyInvoicingFrequency {
					monthsToBill = lookerUtils.GetRemainingMonthsInBillingPeriod(invoiceMonth, sku.StartDate, int(sku.Months), int(properties.InvoiceFrequency))
				}
				adjustedPPU := adjustMonthlyCharge(sku, properties, invoiceMonth) * float64(monthsToBill)

				bucketRef := contract.Entity.Collection("buckets").Doc("looker-bucket")
				row := &domain.InvoiceRow{
					Discount:    contract.Discount,
					Description: "Google Looker",
					Details:     sku.SkuName.Label,
					Quantity:    sku.Quantity,
					PPU:         adjustedPPU,
					Currency:    string(fixer.USD),
					Total:       adjustedPPU * utils.ToProportion(contract.Discount) * float64(sku.Quantity),
					SKU:         sku.SkuName.GoogleSKU,
					Rank:        1,
					Type:        lookerUtils.LookerProductType,
					Final:       true,
					Entity:      contract.Entity,
					Bucket:      bucketRef, // Unsupported, we will need all Looker rows to get into specific bucket
				}

				if properties.InvoiceFrequency != lookerUtils.MonthlyInvoicingFrequency {
					deffStart :=
						time.Date(invoiceMonth.Year(), invoiceMonth.Month(), sku.StartDate.Day(), 0, 0, 0, 0, time.UTC)
					row.DeferredRevenuePeriod = &domain.DeferredRevenuePeriod{
						StartDate: deffStart,
						EndDate:   deffStart.AddDate(0, monthsToBill, 0),
					}
				}

				rows = append(rows, row)
			}
		}
	}

	return rows
}

func adjustMonthlyCharge(sku lookerDomain.LookerContractSKU, properties lookerDomain.LookerContractProperties, invoiceMonth time.Time) float64 {
	if properties.InvoiceFrequency == lookerUtils.MonthlyInvoicingFrequency {
		numOfBillableDaysInMonth := lookerUtils.GetNumOfBillableDaysInMonth(sku, int(sku.Months), invoiceMonth)
		totalDaysInMonth := lookerUtils.GetMonthLength(invoiceMonth)

		return (sku.MonthlySalesPrice / float64(totalDaysInMonth)) * float64(numOfBillableDaysInMonth)
	}

	// for non-monthly invoicing, we simply return the monthly price
	return sku.MonthlySalesPrice
}

func IsInvoiceMonthBillableForSku(sku lookerDomain.LookerContractSKU, properties lookerDomain.LookerContractProperties, invoiceMonth time.Time) bool {
	// billing does not include the end date, so we need to subtract 1 day from the end date
	end := sku.StartDate.AddDate(0, int(sku.Months), -1)

	frequency := int(properties.InvoiceFrequency)
	if frequency == lookerUtils.MonthlyInvoicingFrequency {
		validStart := sku.StartDate.Before(invoiceMonth) || sku.StartDate.Equal(invoiceMonth)
		// we continue to generate an invoice for the month the contract ends in
		validEnd := end.After(invoiceMonth) || end.Equal(invoiceMonth) || (invoiceMonth.Year() == end.Year() && invoiceMonth.Month() == end.Month())

		return validStart && validEnd
	}

	billingIteration := 0
	for {
		billingDate := sku.StartDate.AddDate(0, billingIteration*frequency, 0)

		// if start + some multiple of the invoice frequency is the current invoice month && is before the end date: return true
		if (billingDate.Month() == invoiceMonth.Month() && billingDate.Year() == invoiceMonth.Year()) &&
			(billingDate.Before(end)) {
			return true
		} else if billingDate.After(invoiceMonth) || billingDate.After(end) {
			return false
		}

		billingIteration++
	}
}
