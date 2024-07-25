package invoicing

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/microsoft"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

type BillingTaskMicrosoftAzure struct {
	CustomerID   string    `json:"customer_id"`
	CurrentMonth bool      `json:"current_month"`
	InvoiceMonth time.Time `json:"invoice_month"`
}

type MonthlyBillingMicrosoftAzure struct {
	Customer         *firestore.DocumentRef `firestore:"customer"`
	Verified         bool                   `firestore:"verified"`
	Spend            float64                `firestore:"spend"`
	Credits          map[string]float64     `firestore:"credits"`
	Type             string                 `firestore:"type"`
	SubscriptionID   string                 `firestore:"subscriptionId"`
	SubscriptionName string                 `firestore:"subscriptionName"`
	InvoiceMonth     string                 `firestore:"invoiceMonth"`
	Timestamp        time.Time              `firestore:"timestamp,serverTimestamp"`
}

// MicrosoftAzureInvoicingData processes all customers subscriptions spend for a given month
func (s *InvoicingService) MicrosoftAzureInvoicingData(ctx context.Context) error {
	l := s.Logger(ctx)
	fs := s.Firestore(ctx)

	now := time.Now().UTC()
	invoiceMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	currentMonth := now.Day() > 10

	if !currentMonth {
		invoiceMonth = invoiceMonth.AddDate(0, -1, 0)
	}

	invoiceMonthString := invoiceMonth.Format(times.YearMonthLayout)

	l.Info("Processing Microsoft Azure billing data for ", invoiceMonth)

	input := &microsoft.CostManagementQueryInput{
		Type:      "AmortizedCost",
		Timeframe: "Custom",
		TimePeriod: &microsoft.CostManagementQueryTimePeriod{
			From: invoiceMonth.Format(times.YearMonthDayLayout),
			To:   invoiceMonth.AddDate(0, 1, -1).Format(times.YearMonthDayLayout),
		},
		Dataset: &microsoft.CostManagementQueryDataset{
			Granularity: "None",
			Aggregation: map[string]interface{}{
				"totalCost": map[string]interface{}{
					"name":     "PreTaxCostUSD",
					"function": "Sum",
				},
			},
			Grouping: []map[string]interface{}{
				{
					"type": "Dimension",
					"name": "CustomerTenantId",
				},
				{
					"type": "Dimension",
					"name": "CustomerName",
				},
				{
					"type": "Dimension",
					"name": "SubscriptionId",
				},
				{
					"type": "Dimension",
					"name": "SubscriptionName",
				},
			},
		},
	}

	batch := fb.NewAutomaticWriteBatch(fs, 100)

	for _, asmAccessToken := range microsoft.ASMAccessTokens {
		billingAccount := microsoft.AzureBillingAccounts[asmAccessToken.GetDomain()]

		res, err := microsoft.CostManagementQuery(asmAccessToken, billingAccount, input)
		if err != nil {
			l.Errorf("Error querying cost management API: %s", err)
			return err
		}

		for _, r := range res.Properties.Rows {
			if err := setMonthlyBillingOperation(ctx, r, fs, batch, currentMonth, invoiceMonth, invoiceMonthString); err != nil {
				l.Error(err)
				continue
			}
		}
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func setMonthlyBillingOperation(ctx context.Context, r []interface{}, fs *firestore.Client, batch *fb.AutomaticWriteBatch, currentMonth bool, invoiceMonth time.Time, invoiceMonthString string) error {
	type Row struct {
		Cost             float64
		CustomerID       string
		CustomerName     string
		SubscriptionID   string
		SubscriptionName string
	}

	row := Row{
		Cost:             r[0].(float64),
		CustomerID:       r[1].(string),
		CustomerName:     r[2].(string),
		SubscriptionID:   r[3].(string),
		SubscriptionName: r[4].(string),
	}

	assetID := fmt.Sprintf("%s-%s", common.Assets.MicrosoftAzure, row.SubscriptionID)

	docSnap, err := fs.Collection("assetSettings").Doc(assetID).Get(ctx)
	if err != nil {
		return err
	}

	var as common.AssetSettings
	if err := docSnap.DataTo(&as); err != nil {
		return err
	}

	billingDataRef := fs.Collection("assets").Doc(assetID).Collection("monthlyBillingData").Doc(invoiceMonthString)
	batch.Set(billingDataRef, MonthlyBillingMicrosoftAzure{
		Customer:         as.Customer,
		Verified:         !currentMonth,
		Spend:            row.Cost / microsoft.AzureResellerMarginModifier,
		Credits:          map[string]float64{},
		SubscriptionID:   row.SubscriptionID,
		SubscriptionName: row.SubscriptionName,
		InvoiceMonth:     invoiceMonth.Format(times.YearMonthLayout),
		Type:             common.Assets.MicrosoftAzure,
	})

	return nil
}
