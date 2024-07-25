package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/invoices"
	"github.com/doitintl/hello/scheduled-tasks/priority"
	priorityDomain "github.com/doitintl/hello/scheduled-tasks/priority/domain"
	"github.com/doitintl/hello/scheduled-tasks/slice"
	"github.com/doitintl/hello/scheduled-tasks/stripe/consts"
)

type ProcessingFeeInput struct {
	Amount int64 `json:"amount"`
}

type ProcessingFee struct {
	FeesAllowed       bool    `json:"feesAllowed"`
	Percentage        float64 `json:"percentage"`
	TotalFees         int64   `json:"totalFees"`
	FeesTaxAmount     int64   `json:"feesTaxAmount"`
	FeesTaxPercentage float64 `json:"feesTaxPercentage"`
}

var countriesTaxes = map[string]float64{
	"Australia":      0.10,
	"Estonia":        0.00,
	"France":         0.20,
	"Germany":        0.19,
	"Ireland":        0.23,
	"Israel":         0.17,
	"Netherlands":    0.21,
	"Singapore":      0.00,
	"Spain":          0.21,
	"Sweden":         0.25,
	"Switzerland":    0.077,
	"United Kingdom": 0.20,
	"Japan":          0.10,
	"Indonesia":      0.11,
}

func isExemptFromCreditCardFees(customer *common.Customer, entity *common.Entity, amount int64) (bool, float64) {
	if customer == nil || entity == nil {
		return true, 0
	}

	// CC fees are disabled for customers with the disable CC fees flag
	if slice.Contains(customer.EarlyAccessFeatures, "Disable Credit Card Fees") {
		return true, 0
	}

	if entity.Currency != nil && *entity.Currency == "EGP" {
		// EG has a special CC fees rate, and are not exempt when the amount is below the threshold
		return false, consts.FeesSurchageCreditCardPctEGP
	}

	if entity.BillingAddress.CountryName != nil {
		country := *entity.BillingAddress.CountryName

		// Countries/regions excluded from CC fees
		switch country {
		case "Germany":
			if entity.BillingAddress.StateName != nil && *entity.BillingAddress.StateName == "Bavaria" {
				return true, 0
			}
		case "United States":
			if entity.BillingAddress.StateCode != nil &&
				(*entity.BillingAddress.StateCode == "MA" || *entity.BillingAddress.StateCode == "CT") {
				return true, 0
			}
		}
	}

	if amount <= consts.FeesSurchageMinAmountThreshold {
		return true, 0
	}

	return false, consts.FeesSurchageDefaultCreditCardPct
}

func (s *StripeService) CalculateTotalFees(amount int64, feePct float64) int64 {
	feePercentComplement := (1 - feePct/100)

	amountWithFee := float64(amount) / feePercentComplement

	amountWithFeeRoundedCents := int64(math.Round(amountWithFee))

	return amountWithFeeRoundedCents - amount
}

func (s *StripeService) GetCreditCardProcessingFee(ctx context.Context, customerID string, entity *common.Entity, amount int64) (*ProcessingFee, error) {
	processingFee := &ProcessingFee{}

	customer, err := s.customersDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}

	isExempt, feePct := isExemptFromCreditCardFees(customer, entity, amount)
	if isExempt {
		return processingFee, nil
	}

	var (
		feesTaxAmount     int64
		feesTaxPercentage float64
	)

	feesAmount := s.CalculateTotalFees(amount, feePct)

	if entity.BillingAddress.CountryName != nil {
		country := *entity.BillingAddress.CountryName
		isExportCountry := priority.IsExportCountry(priority.CompanyCode(entity.PriorityCompany), country)

		if !isExportCountry {
			if tax, ok := countriesTaxes[*entity.BillingAddress.CountryName]; ok && tax > 0 {
				feesTaxAmount = int64(math.Round(float64(feesAmount) * tax))
				feesTaxPercentage = tax * 100
			}
		}
	}

	return &ProcessingFee{
		FeesAllowed:       true,
		Percentage:        feePct,
		TotalFees:         feesAmount,
		FeesTaxAmount:     feesTaxAmount,
		FeesTaxPercentage: feesTaxPercentage,
	}, nil
}

func (s *StripeService) createDraftFeesInvoice(
	ctx context.Context,
	originalInvoice *invoices.FullInvoice,
	amount int64,
	invoiceDate time.Time,
) (*priorityDomain.Invoice, error) {
	iv := priorityDomain.Invoice{
		PriorityCustomerID: originalInvoice.PriorityID,
		PriorityCompany:    originalInvoice.Company,
		InvoiceDate:        invoiceDate.Truncate(24 * time.Hour),
		Description:        fmt.Sprintf(consts.FeesSurchargeInvoiceDescriptionFormat, originalInvoice.ID),
		InvoiceItems: []priorityDomain.InvoiceItem{
			{
				SKU:         consts.FeesSurchargeInvoiceItemSKU,
				Description: consts.FeesSurchargeInvoiceItemDescription,
				Details:     fmt.Sprintf(consts.FeesSurchargeInvoiceItemDetailsFormat, originalInvoice.ID),
				Quantity:    1,
				Amount:      float64(amount) / 100,
				Discount:    nil,
				Currency:    originalInvoice.Currency,
			},
		},
	}

	invoice, err := s.priorityService.CreateInvoice(ctx, iv)
	if err != nil {
		return nil, err
	}

	return &invoice, nil
}
