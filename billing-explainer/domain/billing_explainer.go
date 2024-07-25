package domain

import (
	"fmt"
	"strings"
)

type BillingExplainerInputStruct struct {
	CustomerID   string `json:"customer_id" validate:"required"`
	BillingMonth string `json:"billing_month" validate:"required"`
	EntityID     string `json:"entity_id" validate:"required"`
	IsBackfill   bool   `json:"is_backfill"`
}

type PayerAccountInfoStruct struct {
	PayerID      string
	FriendlyName string
}

type PayerAccountHistoryResult struct {
	PayerID string `bigquery:"payer_id"`
}

type DropDownStruct struct {
	CostType string  `firestore:"cost_type"`
	Cost     float64 `firestore:"cost"`
}

type SupportDropDownStruct struct {
	Cost    float64          `firestore:"cost"`
	Details []SupportDetails `firestore:"details"`
}

type ServiceSummary struct {
	Services     []DropDownStruct      `firestore:"serviceCharges"`
	Discount     []DropDownStruct      `firestore:"discounts"`
	Tax          []DropDownStruct      `firestore:"tax"`
	Support      SupportDropDownStruct `firestore:"supportCharges"`
	Credit       []DropDownStruct      `firestore:"credits"`
	Savings      []DropDownStruct      `firestore:"savings"`
	OtherCharges []DropDownStruct      `firestore:"otherCharges"`
	Refund       []DropDownStruct      `firestore:"refunds"`
	Total        float64               `firestore:"total"`
}

type Summary struct {
	Aws  ServiceSummary `firestore:"aws"`
	Doit ServiceSummary `firestore:"doit"`
}

type CostDetail struct {
	EdpDiscount             float64 `firestore:"edpDiscount,omitempty"`
	PrivateRateDiscount     float64 `firestore:"privateRateDiscount,omitempty"`
	Usage                   float64 `firestore:"usage,omitempty"`
	FlexsaveCharges         float64 `firestore:"flexsaveCharges,omitempty"`
	SavingsPlanCoveredUsage float64 `firestore:"savingsPlanCoveredUsage,omitempty"`
	ReservationAppliedUsage float64 `firestore:"reservationAppliedUsage,omitempty"`
	SavingsPlanNegation     float64 `firestore:"savingsPlanNegation,omitempty"`
	SavingsPlanRecurringFee float64 `firestore:"savingsPlanRecurringFee,omitempty"`
	FlexsaveCoveredUsage    float64 `firestore:"flexsaveCoveredUsage,omitempty"`
	BundledDiscount         float64 `firestore:"bundledDiscount,omitempty"`
	SppDiscount             float64 `firestore:"sppDiscount,omitempty"`
	Credit                  float64 `firestore:"credits,omitempty"`
	FlexsaveSavings         float64 `firestore:"flexsaveSavings,omitempty"`
	ReservationRecurringFee float64 `firestore:"reservationRecurringFee,omitempty"`
	OcbCharges              float64 `firestore:"ocbCharges,omitempty"`
	Refund                  float64 `firestore:"refund,omitempty"`
	Fee                     float64 `firestore:"fee,omitempty"`
	SavingsPlanUpfrontFee   float64 `firestore:"savingsPlanUpfrontFee,omitempty"`
	FlexsaveAdjustment      float64 `firestore:"flexsaveAdjustment,omitempty"`
}

type Providers struct {
	DoiT CostDetail `firestore:"doit"`
	AWS  CostDetail `firestore:"aws"`
}

type Explainer struct {
	Summary Summary              `firestore:"summary"`
	Service map[string]Providers `firestore:"service"`
	Account map[string]Providers `firestore:"account"`
}

type SupportDetails struct {
	Project            string  `bigquery:"project_id" firestore:"project_id"`
	ServiceDescription string  `bigquery:"service_description" firestore:"service_description"`
	Description        string  `bigquery:"description" firestore:"description"`
	Cost               float64 `bigquery:"cost" firestore:"cost"`
	BaseCost           float64 `bigquery:"base_cost" firestore:"base_cost"`
}

type SummaryBQ struct {
	Type     string           `bigquery:"type"`
	Cost     float64          `bigquery:"cost"`
	Source   string           `bigquery:"source"`
	CostType string           `bigquery:"cost_type"`
	Details  []SupportDetails `bigquery:"details"`
}

type BillingExplainerParams struct {
	CustomerID    string
	StartOfMonth  string
	EndOfMonth    string
	InvoiceMonth  string
	CustomerTable string
	DefaultBucket string
	InvoicingMode string
}

type CostBreakdownDetails struct {
	CostType string  `bigquery:"cost_type"`
	Cost     float64 `bigquery:"cost"`
}

type ServiceRecord struct {
	ServiceDescription string                 `bigquery:"service_description"`
	Source             string                 `bigquery:"source"`
	CostBreakdown      []CostBreakdownDetails `bigquery:"cost_breakdown"`
}

type AccountRecord struct {
	AccountID     string                 `bigquery:"account_id"`
	Source        string                 `bigquery:"source"`
	CostBreakdown []CostBreakdownDetails `bigquery:"cost_breakdown"`
}

func ToLowerFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	// Convert the first letter to lowercase and concatenate with the rest of the string
	return strings.ToLower(string(s[0])) + s[1:]
}

func findDropDownByCostType(dropDowns []DropDownStruct, costType string) (int, bool) {
	for i, dropDown := range dropDowns {
		if dropDown.CostType == costType {
			return i, true
		}
	}

	return -1, false
}

func updateOrAppendDropDown(target *ServiceSummary, dropDowns []DropDownStruct, dropDown DropDownStruct, res string) {
	foundIndex, found := findDropDownByCostType(dropDowns, dropDown.CostType)
	if found {
		dropDowns[foundIndex].Cost += dropDown.Cost
	} else {
		// Append the new DropDownStruct based on the target slice
		if res == "Service" {
			target.Services = append(target.Services, dropDown)
		} else if res == "Discount" {
			target.Discount = append(target.Discount, dropDown)
		} else if res == "Savings" {
			target.Savings = append(target.Savings, dropDown)
		} else if res == "Tax" {
			target.Tax = append(target.Tax, dropDown)
		} else if res == "Credit" {
			target.Credit = append(target.Credit, dropDown)
		} else if res == "OtherCharges" {
			target.OtherCharges = append(target.OtherCharges, dropDown)
		} else if res == "Refund" {
			target.Refund = append(target.Refund, dropDown)
		}
	}
}

func updateOrAppendSupportStruct(target *ServiceSummary, supportDropDownStruct SupportDropDownStruct) {
	if len(target.Support.Details) > 0 {
		target.Support.Details = append(target.Support.Details, supportDropDownStruct.Details...)
		target.Support.Cost += supportDropDownStruct.Cost
	} else {
		target.Support = supportDropDownStruct
	}
}

func addDiscountToUsage(explainer Explainer, discount float64) Explainer {
	for i, service := range explainer.Summary.Doit.Services {
		if service.CostType == "usage" {
			explainer.Summary.Doit.Services[i].Cost += discount
			explainer.Summary.Doit.Total += discount

			break
		}
	}

	return explainer
}

func MapResultsToExplainer(results []SummaryBQ, serviceBreakdownResults []ServiceRecord, accountBreakdownResults []AccountRecord) Explainer {
	var explainer Explainer

	var totalDiscount float64

	for _, result := range results {
		var target *ServiceSummary

		switch result.Source {
		case "AWS":
			target = &explainer.Summary.Aws
		case "DoiT":
			target = &explainer.Summary.Doit
		default:
			fmt.Printf("Unknown source: %s\n", result.Source)
			continue
		}

		var dropDown = DropDownStruct{CostType: ToLowerFirst(result.CostType), Cost: result.Cost}

		switch result.Type {
		case "Service":
			updateOrAppendDropDown(target, target.Services, dropDown, "Service")
		case "Discount":
			if result.Source == "DoiT" && result.CostType != "BundledDiscount" && result.CostType != "SppDiscount" {
				if result.Cost < 0.0 {
					totalDiscount += (result.Cost * -1)
				} else {
					totalDiscount += result.Cost
					dropDown.Cost = result.Cost * -1
				}
			}

			updateOrAppendDropDown(target, target.Discount, dropDown, "Discount")

		case "Savings":
			updateOrAppendDropDown(target, target.Savings, dropDown, "Savings")

		case "Tax":
			updateOrAppendDropDown(target, target.Tax, dropDown, "Tax")

		case "Support":
			var supportDropDownStruct = SupportDropDownStruct{Cost: result.Cost, Details: result.Details}

			updateOrAppendSupportStruct(target, supportDropDownStruct)

		case "Credit":
			updateOrAppendDropDown(target, target.Credit, dropDown, "Credit")

		case "OtherCharges":
			updateOrAppendDropDown(target, target.OtherCharges, dropDown, "OtherCharges")

		case "Refund":
			updateOrAppendDropDown(target, target.Refund, dropDown, "Refund")

		default:
			fmt.Printf("unknown type: %s\n", result.Type)
		}

		target.Total += dropDown.Cost
	}

	finalExplainer := addDiscountToUsage(explainer, totalDiscount)

	finalExplainer.Service = MapServiceBreakDownInExplainer(serviceBreakdownResults)

	finalExplainer.Account = MapAccountBreakDownInExplainer(accountBreakdownResults)

	return finalExplainer
}

func MapCostDetailsForServiceAndAccount(costBreakdown []CostBreakdownDetails, source string, costDetail *CostDetail) {
	var totalDiscount float64

	var edpDiscount float64

	for _, detail := range costBreakdown {
		if source == "DoiT" && (detail.CostType == "PrivateRateDiscount" || detail.CostType == "EdpDiscount") {
			if detail.Cost < 0.0 {
				totalDiscount += (detail.Cost * -1)
			} else {
				totalDiscount += detail.Cost
				detail.Cost = (detail.Cost * -1)
			}
		}

		if detail.CostType == "EdpDiscount" {
			if detail.Cost < 0.0 {
				edpDiscount += (detail.Cost * -1)
			} else {
				edpDiscount += detail.Cost
			}
		}

		switch detail.CostType {
		case "Usage", "FlexsaveRDSManagementFee":
			if costDetail.Usage > 0 {
				costDetail.Usage = costDetail.Usage + detail.Cost
			} else {
				costDetail.Usage = detail.Cost
			}

		case "EdpDiscount":
			costDetail.EdpDiscount += detail.Cost
		case "flexsaveCharges":
			costDetail.FlexsaveCharges += detail.Cost
		case "Credit": // Assuming there might be a Credit field
			costDetail.Credit += detail.Cost
		case "SavingsPlanCoveredUsage":
			costDetail.SavingsPlanCoveredUsage += detail.Cost
		case "FlexsaveCoveredUsage":
			costDetail.FlexsaveCoveredUsage += detail.Cost
		case "SavingsPlanRecurringFee":
			costDetail.SavingsPlanRecurringFee += detail.Cost
		case "reservationAppliedUsage":
			costDetail.ReservationAppliedUsage += detail.Cost
		case "SavingsPlanNegation":
			costDetail.SavingsPlanNegation += detail.Cost
		case "FlexsaveSavings":
			costDetail.FlexsaveSavings += detail.Cost
		case "BundledDiscount":
			costDetail.BundledDiscount += detail.Cost
		case "PrivateRateDiscount":
			costDetail.PrivateRateDiscount += detail.Cost
		case "SppDiscount":
			costDetail.SppDiscount += detail.Cost
		case "reservationRecurringFee":
			costDetail.ReservationRecurringFee += detail.Cost
		case "ocbCharges":
			costDetail.OcbCharges += detail.Cost
		case "Refund":
			costDetail.Refund += detail.Cost
		case "Fee":
			costDetail.Fee += detail.Cost
		case "SavingsPlanUpfrontFee":
			costDetail.SavingsPlanUpfrontFee += detail.Cost
		case "FlexsaveAdjustment":
			costDetail.FlexsaveAdjustment += detail.Cost

		default:
			fmt.Printf("unknown type: %s\n", detail.CostType)
		}
	}

	if costDetail.Usage > 0.0 && totalDiscount > 0.0 && source == "DoiT" {
		costDetail.Usage += totalDiscount
	}

	// Add edpDiscount to Fee only when Usage==0 because we already add edpDiscount to Usage in above step
	if costDetail.SavingsPlanRecurringFee > 0.0 && edpDiscount > 0.0 && source == "DoiT" && costDetail.Usage == 0.0 {
		costDetail.SavingsPlanRecurringFee += edpDiscount
	}

	if costDetail.ReservationRecurringFee > 0.0 && edpDiscount > 0.0 && source == "DoiT" && costDetail.Usage == 0.0 {
		costDetail.ReservationRecurringFee += edpDiscount
	}

	if costDetail.OcbCharges > 0.0 && edpDiscount > 0.0 && source == "DoiT" && costDetail.Usage == 0.0 {
		costDetail.OcbCharges += edpDiscount
	}

	if costDetail.FlexsaveCharges > 0.0 && edpDiscount > 0.0 && source == "DoiT" && costDetail.Usage == 0.0 {
		costDetail.FlexsaveCharges += edpDiscount
	}
}

func MapServiceBreakDownInExplainer(serviceBreakdownResults []ServiceRecord) map[string]Providers {
	detailmap := make(map[string]Providers)

	for _, result := range serviceBreakdownResults {
		serviceProvider, exists := detailmap[result.ServiceDescription]

		if !exists {
			serviceProvider = Providers{}
		}

		var costDetail *CostDetail

		switch result.Source {
		case "DoiT":
			costDetail = &serviceProvider.DoiT
		case "AWS":
			costDetail = &serviceProvider.AWS
		}

		MapCostDetailsForServiceAndAccount(result.CostBreakdown, result.Source, costDetail)

		detailmap[result.ServiceDescription] = serviceProvider
	}

	return detailmap
}

func MapAccountBreakDownInExplainer(accountBreakdownResults []AccountRecord) map[string]Providers {
	detailmap := make(map[string]Providers)

	for _, result := range accountBreakdownResults {
		provider, exists := detailmap[result.AccountID]

		if !exists {
			provider = Providers{}
		}

		var costDetail *CostDetail

		switch result.Source {
		case "DoiT":
			costDetail = &provider.DoiT
		case "AWS":
			costDetail = &provider.AWS
		}

		MapCostDetailsForServiceAndAccount(result.CostBreakdown, result.Source, costDetail)

		detailmap[result.AccountID] = provider
	}

	return detailmap
}
