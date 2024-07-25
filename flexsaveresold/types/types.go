package types

import (
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
)

type SpendSavings = map[string]float64

type CloudEnablementDetails struct {
	OnDemandSpend    *float64 `json:"onDemandSpend,omitempty"`
	Savings          *float64 `json:"savings,omitempty"`
	SavingsRate      *float64 `json:"savingsRate,omitempty"`
	ReasonCantEnable *string  `json:"reasonCantEnable,omitempty"`
}

type CurrentMonthSavingsMetrics struct {
	Month            string  `firestore:"month"`
	ProjectedSavings float64 `firestore:"projectedSavings"`
}

type FlexSaveSavingsMetrics struct {
	SavingsSummary   *SavingsSummary                   `firestore:"savingsSummary"`
	SavingsHistory   map[string]*MonthlySavingsMetrics `firestore:"savingsHistory"`
	Timestamp        *time.Time                        `firestore:"timestamp,serverTimestamp"`
	Enabled          bool                              `firestore:"enabled"`
	ReasonCantEnable string                            `firestore:"reasonCantEnable"`
	TimeEnabled      *time.Time                        `firestore:"timeEnabled"`
}

type SavingsSummary struct {
	CurrentMonth CurrentMonthSavingsMetrics `firestore:"currentMonth"`
	NextMonth    NextMonthSavingsMetrics    `firestore:"nextMonth"`
}

type NextMonthSavingsMetrics struct {
	Savings          float64  `firestore:"savings"`
	OnDemandSpend    float64  `firestore:"onDemandSpend"`
	SavingsRate      float64  `firestore:"savingsRate"`
	HourlyCommitment *float64 `firestore:"savingsPlanHourlyCommitment,omitempty"`
}

type MonthlySavingsMetrics struct {
	Savings       float64 `firestore:"savings"`
	OnDemandSpend float64 `firestore:"onDemandSpend"`
	SavingsRate   float64 `firestore:"savingsRate"`
}

type CloudConfigData struct {
	SavingsSummary    *SavingsSummary                   `firestore:"savingsSummary"`
	SavingsHistory    map[string]*MonthlySavingsMetrics `firestore:"savingsHistory"`
	Timestamp         *time.Time                        `firestore:"timestamp"`
	Enabled           bool                              `firestore:"enabled"`
	TimeEnabled       *time.Time                        `firestore:"timeEnabled"`
	TimeEnabledDaily  *time.Time                        `firestore:"timeEnabledDaily"`
	TimeDisabledDaily *time.Time                        `firestore:"timeDisabledDaily"`
	ReasonCantEnable  string                            `firestore:"reasonCantEnable"`
}

type ConfigData struct {
	AWS CloudConfigData `firestore:"AWS"`
	GCP CloudConfigData `firestore:"GCP"`
}

type Discount struct {
	Criteria      string    `firestore:"criteria" json:"criteria"`
	Discount      float64   `firestore:"discount" json:"discount"`
	EffectiveDate time.Time `firestore:"effectiveDate" json:"effectiveDate"`
}

// PayerConfig represents a payer account configurations to be used for billing recalculations and SP allocations
type PayerConfig struct {
	CustomerID                  string     `json:"customerId" binding:"required"`
	AccountID                   string     `json:"accountId" binding:"required"`
	PrimaryDomain               string     `json:"primaryDomain" binding:"required"`
	FriendlyName                string     `json:"friendlyName" binding:"required"`
	Name                        string     `json:"name" binding:"required"`
	Status                      string     `json:"status" binding:"required"`
	Type                        string     `json:"type"`
	Managed                     string     `json:"managed"`
	TimeEnabled                 *time.Time `json:"timeEnabled"`
	TimeDisabled                *time.Time `json:"timeDisabled"`
	LastUpdated                 *time.Time `json:"lastUpdated"`
	TargetPercentage            *float64   `json:"targetPercentage"`
	MinSpend                    *float64   `json:"minSpend"`
	MaxSpend                    *float64   `json:"maxSpend"`
	DiscountDetails             []Discount `json:"discountDetails"`
	SageMakerStatus             string     `json:"sagemakerStatus" binding:"required"`
	SageMakerTimeEnabled        *time.Time `json:"sagemakerTimeEnabled"`
	SageMakerTimeDisabled       *time.Time `json:"sagemakerTimeDisabled"`
	RDSStatus                   string     `json:"rdsStatus" binding:"required"`
	RDSTimeEnabled              *time.Time `json:"rdsTimeEnabled"`
	RDSTimeDisabled             *time.Time `json:"rdsTimeDisabled"`
	Seasonal                    bool       `json:"seasonal"`
	KeepActiveEvenWhenOnCredits bool       `json:"keepActiveEvenWhenOnCredits"`
	RDSTargetPercentage         *float64   `json:"rdsTargetPercentage"`
}

func (p *PayerConfig) StatusForFlexsaveType(flexsaveType utils.FlexsaveType) string {
	switch flexsaveType {
	case utils.ComputeFlexsaveType:
		return p.Status
	case utils.SageMakerFlexsaveType:
		return p.SageMakerStatus
	case utils.RDSFlexsaveType:
		return p.RDSStatus
	default:
		return ""
	}
}

type PayerConfigCreatePayload struct {
	PayerConfigs []PayerConfig `json:"payerConfigs" binding:"required"`
}

type PayerConfigUpdatePayload struct {
	PayerConfigs []PayerConfig `json:"payerConfigs" binding:"required"`
	ChangedBy    string        `json:"changedBy" binding:"required"`
	Reason       string        `json:"reason" binding:"required"`
}

type PotentialResponse struct {
	Customer  *firestore.DocumentRef `firestore:"customer"`
	Data      []PotentialData        `firestore:"data"`
	Timestamp time.Time              `firestore:"timestamp,serverTimestamp"`
}

type PotentialData struct {
	Region          LocationVal `json:"region"`
	Account         string      `json:"account"`
	OperatingSystem LabelVal    `json:"operatingSystem"`
	NumInstances    float64     `json:"numInstances"`
	InstanceType    string      `json:"instanceType"`
}

type LocationVal struct {
	Location string `json:"location"`
	Value    string `json:"value"`
}

type LabelVal struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type Recommendation struct {
	Region             string   `json:"region"`
	InstanceFamily     string   `json:"instanceFamily"`
	PayerAccountID     string   `json:"payerAccountId"`
	OperatingSystem    string   `json:"operatingSystem"`
	InstanceSize       string   `json:"instanceSize"`
	NumInstances       float64  `json:"numInstances"`
	Savings            *float64 `json:"savings"`
	OnDemand           *float64 `json:"onDemand"`
	LinkedAccountID    string   `json:"linkedAccountID"`
	PriceAfterDiscount *float64 `json:"priceAfterDiscount"`
}

type RecommendationsResultChannel struct {
	Savings            *float64
	OnDemand           *float64
	PriceAfterDiscount *float64
	Pos                int
	Errors             error
	Warning            error
}

type SavingsPlanData struct {
	SavingsPlanID          string    `bigquery:"sp_arn" firestore:"savingsPlanID"`
	PaymentOption          string    `bigquery:"payment_option" firestore:"paymentOption"`
	HourlyUpfrontFee       float64   `bigquery:"hourly_upfront_fee" json:"-" firestore:"-"`
	UpfrontPayment         float64   `firestore:"upfrontPayment"`
	RecurringPayment       float64   `bigquery:"recurring_fee" firestore:"recurringPayment"`
	Commitment             float64   `bigquery:"commitment" firestore:"commitment"`
	Term                   string    `bigquery:"term" firestore:"term"`
	Type                   string    `bigquery:"type" firestore:"type"`
	ExpirationDateString   string    `bigquery:"end_time" json:"-" firestore:"-"`
	ExpirationDate         time.Time `firestore:"expirationDate"`
	StartDateString        string    `bigquery:"start_time" json:"-" firestore:"-"`
	StartDate              time.Time `firestore:"startDate"`
	Savings                float64   `bigquery:"savings" firestore:"mtdSavings"`
	OnDemandCostEquivalent float64   `bigquery:"on_demand_cost" json:"-" firestore:"-"`
	MostRecentHour         time.Time `bigquery:"max_export_time" json:"-" firestore:"-"`
}

type SavingsPlanDoc struct {
	SavingsPlans []SavingsPlanData `firestore:"savingsPlans"`
	LastUpdated  time.Time         `firestore:"lastUpdated,serverTimestamp"`
}

type SharedPayerOndemandMonthlyData struct {
	OndemandCost float64 `bigquery:"ondemand_cost"`
	MonthYear    string  `bigquery:"month_year"`
}

type WelcomeEmailParams struct {
	CustomerID  string
	Cloud       string
	Marketplace bool
}
