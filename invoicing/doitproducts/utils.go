package doitproducts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

var cloudDesc = map[string]string{
	"google-cloud":          "GCP",
	"google-cloud-platform": "GCP",
	"amazon-web-services":   "AWS",
	"azure":                 "Azure",
	"looker":                "Looker",
	"office-365":            "Office-365",
	"google-workplace":      "Google-Workspace",
}

const PaymentTermAnnual = "annual"

func transcode(in, out interface{}) error {
	buf := new(bytes.Buffer)

	err := json.NewEncoder(buf).Encode(in)
	if err != nil {
		return err
	}

	err = json.NewDecoder(buf).Decode(out)
	if err != nil {
		return err
	}

	return nil
}

func reverseConversionRate(rates map[string]float64, currency string) float64 {
	reverseConversion := 1.0
	if actualRate, prs := rates[currency]; prs {
		reverseConversion = actualRate
	}

	return reverseConversion
}

// invoiceMonth is 0 hour 0 min on last day of month; eg: 2024/04/30 00.00.00.000 UTC
func getDetailsForInvoiceRow(contractStartDate, contractEndDate, invoiceMonth *time.Time) string {
	startDateValue := time.Date(invoiceMonth.Year(), invoiceMonth.Month(), 1, 0, 0, 1, 0, time.UTC)
	endDateValue := time.Date(invoiceMonth.Year(), invoiceMonth.Month()+1, 0, 0, 0, 0, 0, time.UTC)

	startDate := &startDateValue
	endDate := &endDateValue

	if contractStartDate.After(startDateValue) {
		startDate = contractStartDate
	}

	if contractEndDate != nil && !contractEndDate.IsZero() && contractEndDate.Before(endDateValue) {
		endDate = contractEndDate
	}

	return fmt.Sprintf("Period of %s to %s", startDate.Format("2006/01/02"), endDate.Format("2006/01/02"))
}

func findAnnualRollingDates(contractStartDate, contractEndDate *time.Time) (annualStartDay, annualEndDay time.Time, baseRow string) {
	annualStartDay = dayBoundary(*contractStartDate)
	annualEndDay = dayBoundary(annualStartDay).AddDate(1, 0, -1)

	if contractEndDateBeforeAnnualEndDate(contractEndDate, annualEndDay) {
		annualEndDay = dayBoundary(*contractEndDate)
	}

	return annualStartDay, annualEndDay, fmt.Sprintf("Period of %s to %s", annualStartDay.Format("2006/01/02"), annualEndDay.Format("2006/01/02"))
}

func dayBoundary(referenceDate time.Time) time.Time {
	return time.Date(referenceDate.Year(), referenceDate.Month(), referenceDate.Day(), 0, 0, 0, 0, time.UTC)
}

func contractEndDateBeforeAnnualEndDate(contractEndDate *time.Time, annualEndDate time.Time) bool {
	if contractEndDate == nil || contractEndDate.IsZero() { // open-ended contract
		return false
	}

	contractEndDay := dayBoundary(*contractEndDate)
	annualEndDay := dayBoundary(annualEndDate)

	return contractEndDay.Before(annualEndDay)
}

func generateBaseDescription(contractTierDesc string, contractDiscount float64, paymentTerm string) string {
	description := "DoiT Cloud " + contractTierDesc + " - Subscription"

	if strings.ToLower(paymentTerm) == PaymentTermAnnual {
		description += " - Annual"
	}

	if contractDiscount > 0 {
		description += fmt.Sprintf(" with %.2f%% discount", contractDiscount)
	}

	return description
}

func generateDescription(contractTierDesc string, eachCloudRowCloud string, monthlyFlatRate float64, paymentTerm string) string {
	description := ""

	switch monthlyFlatRate {
	case 0:
		description = fmt.Sprintf("DoiT Cloud %s - cloud spend (%s)", contractTierDesc, cloudDesc[eachCloudRowCloud])
	default:
		description = fmt.Sprintf("DoiT Cloud %s - %.2f%% cloud spend (%s)", contractTierDesc, monthlyFlatRate, cloudDesc[eachCloudRowCloud])
	}

	if strings.ToLower(paymentTerm) == PaymentTermAnnual {
		description += " - Annual"
	}

	return description
}
