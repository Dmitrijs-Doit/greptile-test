package invoicing

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/invoicing/aws"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

type BillingTaskAmazonWebServices struct {
	CustomerID   string    `json:"customer_id"`
	InvoiceMonth time.Time `json:"invoice_month"`
	DryRun       bool      `json:"dry_run"`
}

type BillingTaskAmazonWebServicesAnalytics struct {
	InvoiceMonth         string `json:"invoice_month"`
	Version              string `json:"version"`
	ValidateWithOldLogic bool   `json:"validate_with_old_logic"`
	Dry                  bool   `json:"dry"`
}

func (s *commonAWSInvoicingService) CalculateSpendAndCreditsData(
	invoiceMonthString string,
	accountID string,
	date time.Time,
	cost float64,
	entityRef *firestore.DocumentRef,
	assetRef *firestore.DocumentRef,
	credits []*aws.CustomerCreditAmazonWebServices,
	accountsData map[string]float64,
	creditsData map[string]map[string]float64) {
	var credit *aws.CustomerCreditAmazonWebServices

	for _, _credit := range credits {
		switch {
		// validate that this asset assigned to the same entity as the credit
		case entityRef == nil || entityRef.ID != _credit.Entity.ID:
			continue

		// validate that the asset belongs to this credit
		case _credit.Assets != nil && doitFirestore.FindIndex(_credit.Assets, assetRef) == -1:
			continue

		// validate that the credit has any funds remaining and the spend is not negative
		case cost <= 0 || _credit.Remaining <= 0:
			continue

		// validate that the credit starts or equals to the spend date
		// validate that the credit ends after the spend date
		case date.Before(_credit.StartDate) || !_credit.EndDate.After(date):
			continue

		// found usable credit, but it hasn't enough funds to cover this cost
		// use the remaining credit, and search for another credit
		case cost > _credit.Remaining:
			_credit.Touched = true
			accountsData[accountID] += _credit.Remaining
			creditsData[accountID][_credit.Snapshot.Ref.ID] += _credit.Remaining
			_credit.Utilization[invoiceMonthString][accountID] += _credit.Remaining
			cost -= _credit.Remaining
			_credit.Remaining = 0
			_credit.DepletionDate = &date

			continue
		}

		// it's a match!
		_credit.Touched = true
		credit = _credit

		break
	}

	if credit != nil {
		// this credit has enough funds to cover this cost (i.e. cost <= credit.Remaining)
		accountsData[accountID] += cost
		creditsData[accountID][credit.Snapshot.Ref.ID] += cost
		credit.Utilization[invoiceMonthString][accountID] += cost
		credit.Remaining -= cost
	} else {
		// no credit found
		accountsData[accountID] += cost
	}
}

func (s *commonAWSInvoicingService) GetAmazonWebServicesCredits(ctx context.Context, invoiceMonth time.Time, customerRef *firestore.DocumentRef, accountsWithAssets []string) ([]*aws.CustomerCreditAmazonWebServices, error) {
	invoiceMonthString := invoiceMonth.Format(InvoiceMonthPattern)
	credits := make([]*aws.CustomerCreditAmazonWebServices, 0)

	docSnaps, err := customerRef.Collection("customerCredits").
		Where("type", "==", common.Assets.AmazonWebServices).
		Where("endDate", ">", invoiceMonth).
		OrderBy("endDate", firestore.Asc).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	for _, docSnap := range docSnaps {
		var credit aws.CustomerCreditAmazonWebServices
		if err := docSnap.DataTo(&credit); err != nil {
			return nil, err
		}

		startDate := time.Date(credit.StartDate.Year(), credit.StartDate.Month(), 1, 0, 0, 0, 0, time.UTC)
		credit.Snapshot = docSnap
		credit.Remaining = credit.Amount
		credit.RemainingPreviousMonth = credit.Remaining

		if credit.Utilization == nil {
			credit.Utilization = make(map[string]map[string]float64)
		}

		switch {
		case startDate.Before(invoiceMonth):
			// calculate remainder from previous month, do not modify past utilization
			for _invoiceMonth, customersMonthUtilization := range credit.Utilization {
				utilizationMonth, err := time.Parse(InvoiceMonthPattern, _invoiceMonth)
				if err != nil {
					return nil, err
				}

				if utilizationMonth.Before(invoiceMonth) {
					for _, v := range customersMonthUtilization {
						credit.Remaining -= v
					}
				}
			}

			credit.RemainingPreviousMonth = credit.Remaining

			fallthrough

		case startDate.Equal(invoiceMonth):
			if _, prs := credit.Utilization[invoiceMonthString]; prs {
				for accountID, v := range credit.Utilization[invoiceMonthString] {
					if slice.Contains(accountsWithAssets, accountID) || isValidMarketplaceAccountId(accountsWithAssets, accountID) {
						delete(credit.Utilization[invoiceMonthString], accountID)
					} else {
						credit.Remaining -= v
					}
				}
			} else {
				credit.Utilization[invoiceMonthString] = make(map[string]float64)
			}

			if credit.Remaining < 1e-3 {
				credit.Remaining = 0
			}

			credits = append(credits, &credit)
		default:
			continue
		}
	}

	return credits, nil
}

func isValidMarketplaceAccountId(validAccounts []string, accountId string) bool {
	if strings.Contains(accountId, "_marketplace_") && len(accountId) > 24 {
		return slice.Contains(validAccounts, accountId[0:12])
	}

	return false
}
