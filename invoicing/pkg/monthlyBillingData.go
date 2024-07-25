package pkg

import (
	"time"

	"cloud.google.com/go/firestore"
)

type CostAndSavingsAwsLineItemKey struct {
	AccountID      string
	PayerAccountID string
	CostType       string
	Label          string
	MarketplaceSD  string
	IsMarketplace  bool
}

type CostAndSavingsAwsLineItem struct {
	Costs                      float64
	Savings                    float64
	FlexsaveComputeNegations   float64
	FlexsaveSagemakerNegations float64
	FlexsaveRDSNegations       float64
	FlexsaveAdjustments        float64
	FlexsaveRDSCharges         float64
}

type MonthlyBillingAwsFlexsave struct {
	FlexsaveComputeNegations   float64 `firestore:"flexsaveComputeNegations"`
	FlexsaveSagemakerNegations float64 `firestore:"flexsaveSagemakerNegations"`
	FlexsaveRDSNegations       float64 `firestore:"flexsaveRDSNegations"`
	FlexsaveRDSCharges         float64 `firestore:"flexsaveRDSCharges"`
	ManagementCosts            float64 `firestore:"managementCosts"`
	FlexsaveSpCredits          float64 `firestore:"flexsaveCredits"`
	FlexsaveAdjustments        float64 `firestore:"flexsaveAdjustments"`
}

type MarketplaceConstituent struct {
	Spend   float64            `firestore:"spend"`
	Credits map[string]float64 `firestore:"credits"` // services to credit map

}

type MonthlyBillingAmazonWebServices struct {
	Customer                   *firestore.DocumentRef            `firestore:"customer"`
	Verified                   bool                              `firestore:"verified"`
	Spend                      float64                           `firestore:"spend"`
	Flexsave                   *MonthlyBillingAwsFlexsave        `firestore:"flexsave"`
	Credits                    map[string]float64                `firestore:"credits"`
	Type                       string                            `firestore:"type"`
	InvoiceMonth               string                            `firestore:"invoiceMonth"`
	Timestamp                  time.Time                         `firestore:"timestamp,serverTimestamp"`
	CustBillingTblSessionID    string                            `firestore:"custBillingTblSessionId"`
	MarketplaceConstituents    map[string]MarketplaceConstituent `firestore:"marketplaceConstituents"`
	MarketplaceConstituentsRef map[string]string                 `firestore:"marketplaceConstituentsRef"`
}

type MonthlyBillingFlexsaveStandalone struct {
	Customer     *firestore.DocumentRef `firestore:"customer"`
	Spend        map[string]float64     `firestore:"spend"`
	Type         string                 `firestore:"type"`
	InvoiceMonth string                 `firestore:"invoiceMonth"`
	Timestamp    time.Time              `firestore:"timestamp,serverTimestamp"`
}
