package flexapi

import (
	"time"
)

type SavingsPlan struct {
	Commitment             string   `json:"commitment"`
	Currency               string   `json:"currency"`
	State                  string   `json:"state"`
	Start                  string   `json:"start"`
	End                    string   `json:"end"`
	OfferingID             string   `json:"offeringId"`
	PaymentOption          string   `json:"paymentOption"`
	RecurringPaymentAmount string   `json:"recurringPaymentAmount"`
	SavingsPlanARN         string   `json:"savingsPlanArn"`
	SavingsPlanType        string   `json:"savingsPlanType"`
	ProductTypes           []string `json:"productTypes"`
}

type RI struct{}

// Organization represents the configuration of a specific AWS organization
type Organization struct {
	ARN                string `json:"Arn"`
	MasterAccountARN   string `json:"MasterAccountArn"`
	MasterAccountEmail string `json:"MasterAccountEmail"`
	MasterAccountID    string `json:"MasterAccountId"`
	ID                 string `json:"Id"`
}

// Account represents a specific AWS account with identifiers and the AWS savings mechanisms attached to it
type Account struct {
	AccountName  string        `json:"accountName"`
	AccountID    string        `json:"awsAccountId"`
	Organization Organization  `json:"organization"`
	SavingsPlans []SavingsPlan `json:"savingsPlans"`
	RIs          []RI          `json:",omitempty"`
}

type TimeWindowType int

const (
	TimeWindow14Days TimeWindowType = 14
	TimeWindow30Days TimeWindowType = 30
)

type RDSBottomUpRecommendationTimeWindow struct {
	TimeWindowType TimeWindowType `json:"time_window_type"`
	StartDateTime  time.Time      `json:"start_date_time"`
	EndDateTime    time.Time      `json:"end_date_time"`

	Baseline   float64 `json:"baseline"`
	UpperBound float64 `json:"upper_bound"`
	LowerBound float64 `json:"lower_bound"`
}

type RDSBottomUpRecommendation struct {
	PayerID string `json:"payer_id,omitempty"`

	Region     string `json:"region,omitempty"`
	FamilyType string `json:"family_type,omitempty"`
	Database   string `json:"database,omitempty"`

	RDSBottomUpRecommendationTimeWindows []RDSBottomUpRecommendationTimeWindow `json:"rds_bottom_up_recommendation_time_windows,omitempty"`
	RecommendationTimeWindow             TimeWindowType                        `json:"recommendation_time_window,omitempty"`

	ProcessID  string    `json:"process_id,omitempty"`
	ExportTime time.Time `json:"export_time"`
}
