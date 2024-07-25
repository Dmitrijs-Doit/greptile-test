package domain

import (
	"time"

	"cloud.google.com/go/firestore"
	pkg "github.com/doitintl/firestore/pkg"
)

type PaymentTerm string

const (
	PaymentTermMonthly PaymentTerm = "monthly"
	PaymentTermAnnual  PaymentTerm = "annual"
)

const BillingMonthLayout string = "2006-01"

const (
	BillingDataField    = "billingData"
	LastUpdateDateField = "lastUpdateDate"
	FinalFlag           = "final"
)

type ContractType string

const (
	ContractTypeAWS              ContractType = "amazon-web-services"
	ContractTypeGoogleCloud      ContractType = "google-cloud"
	ContractTypeAzure            ContractType = "microsoft-azure"
	ContractTypeGoogleWorkspace  ContractType = "g-suite"
	ContractTypeGoogleCloudPLPS  ContractType = "google-cloud-partner-led-premium-support"
	ContractTypeLooker           ContractType = "looker"
	ContractTypeNavigator        ContractType = "navigator"
	ContractTypeSolve            ContractType = "solve"
	ContractTypeSolveAccelerator ContractType = "solve-accelerator"
)

type ContractInputStruct struct {
	CustomerID       string            `json:"customerID" validate:"required"`
	Tier             string            `json:"tier,omitempty"`
	StartDate        string            `json:"startDate" validate:"required"`
	EndDate          string            `json:"endDate,omitempty"`
	EntityID         string            `json:"entityID,omitempty"`
	Type             string            `json:"type" validate:"required"`
	Discount         float64           `json:"discount,omitempty"`
	AccountManager   string            `json:"accountManager,omitempty"`
	ContractFile     *pkg.ContractFile `json:"contractFile,omitempty"`
	EstimatedValue   float64           `json:"estimatedValue,omitempty"`
	Notes            string            `json:"notes,omitempty"`
	PurchaseOrder    string            `json:"purchaseOrder,omitempty"`
	CommitmentMonths float64           `json:"commitmentMonths,omitempty"`
	PointOfSale      string            `json:"pointOfSale,omitempty"`
	PaymentTerm      string            `json:"paymentTerm,omitempty"`
	ChargePerTerm    float64           `json:"chargePerTerm,omitempty"`
	MonthlyFlatRate  float64           `json:"monthlyFlatRate,omitempty"`
	IsCommitment     bool              `json:"isCommitment"`
	IsAdvantage      bool              `json:"isAdvantage"`
	TypeContext      string            `json:"typeContext,omitempty"`
	EstimatedFunding *float64          `json:"estimatedFunding,omitempty"`
}

type ContractUpdateInputStruct struct {
	Tier             string            `json:"tier,omitempty"`
	StartDate        string            `json:"startDate,omitempty"`
	EndDate          string            `json:"endDate,omitempty"`
	EntityID         string            `json:"entityID,omitempty"`
	Type             string            `json:"type,omitempty"`
	Discount         float64           `json:"discount,omitempty"`
	AccountManager   string            `json:"accountManager,omitempty"`
	ContractFile     *pkg.ContractFile `json:"contractFile,omitempty"`
	EstimatedValue   float64           `json:"estimatedValue,omitempty"`
	Notes            string            `json:"notes,omitempty"`
	PurchaseOrder    string            `json:"purchaseOrder,omitempty"`
	CommitmentMonths float64           `json:"commitmentMonths,omitempty"`
	PointOfSale      string            `json:"pointOfSale,omitempty"`
	PaymentTerm      string            `json:"paymentTerm,omitempty"`
	ChargePerTerm    float64           `json:"chargePerTerm,omitempty"`
	MonthlyFlatRate  float64           `json:"monthlyFlatRate,omitempty"`
	IsCommitment     *bool             `json:"isCommitment"`
	IsAdvantage      *bool             `json:"isAdvantage"`
	TypeContext      string            `json:"typeContext"`
	EstimatedFunding *float64          `json:"estimatedFunding"`
}

type ContractBillingAggregatedData struct {
	BaseFee     float64                 `firestore:"baseFee"`
	Consumption []pkg.ConsumptionStruct `firestore:"consumption,omitempty"`
}

type AggregatedInvoiceInputStruct struct {
	InvoiceMonth string `json:"invoiceMonth,omitempty"`
}

type OriginalGoogleTier string

const (
	StandardTier OriginalGoogleTier = "standard"
	PremiumTier  OriginalGoogleTier = "premium"
	Enhanced     OriginalGoogleTier = "enhanced"
	NoSupport    OriginalGoogleTier = "no-support"
)

type GCPContractSupportInput struct {
	OriginalSupportTier OriginalGoogleTier `firestore:"originalSupportTier"`
	UpdatedAt           time.Time          `firestore:"updatedAt"`
}

type UpdateSupportInput struct {
	Ref     *firestore.DocumentRef
	Support GCPContractSupportInput
}
