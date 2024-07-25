package service

import (
	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type BudgetAPI struct {
	// budget ID, identifying the report
	// in:path
	ID *string `json:"id"`
	// Budget Name
	// required: true
	Name *string `json:"name"`
	// Budget description
	// default: ""
	Description *string `json:"description"`

	Public *collab.PublicAccess `json:"public"`
	// List of up to three thresholds defined as percentage of amount
	// default: []
	Alerts *[3]ExternalBudgetAlert `json:"alerts"`
	// List of permitted users to view/edit the report
	Collaborators []collab.Collaborator `json:"collaborators"`
	// List of emails to notify when reaching alert threshold
	Recipients []string `json:"recipients"`
	// List of slack channels to notify when reaching alert threshold
	// default: []
	RecipientsSlackChannels []common.SlackChannel `json:"recipientsSlackChannels"`
	// List of attributions that defines that budget scope
	// required: true
	Scope []string `json:"scope"`
	// Budget period amount
	// required: true(if usePrevSpend is false)
	Amount *float64 `json:"amount"`
	// Use the last period's spend as the target amount for recurring budgets
	// default: false
	UsePrevSpend *bool `json:"usePrevSpend"`
	// Budget currency can be one of: ["USD","ILS","EUR","GBP","AUD","CAD","DKK","NOK","SEK","BRL","SGD","MXN","CHF","MYR","TWD","EGP","ZAR","JPY","IDR"]
	// required: true
	Currency *string `json:"currency"`
	// Periodical growth percentage in recurring budget
	// default: 0
	GrowthPerPeriod *float64 `json:"growthPerPeriod"`
	// Budget metric - currently fixed to "cost"
	// default: "cost"
	Metric *string `json:"metric"`
	// Recurring budget interval can be on of: ["day", "week", "month", "quarter","year"]
	// required: true
	TimeInterval *string `json:"timeInterval"`
	// budget type can be one of: ["fixed", "recurring"]
	// required: true
	Type *string `json:"type"`
	// Budget start Date (in UNIX timestamp)
	// required: true
	StartPeriod *int64 `json:"startPeriod"`
	// Fixed budget end date (in UNIX timestamp)
	// required: true(if budget type is fixed)
	EndPeriod *int64 `json:"endPeriod"`
	// Creation time (in UNIX timestamp)
	CreateTime int64 `json:"createTime"`
	// Update time (in UNIX timestamp)
	UpdateTime            int64   `json:"updateTime"`
	CurrentUtilization    float64 `json:"currentUtilization"`
	ForecastedUtilization float64 `json:"forecastedUtilization"`
}

type BudgetsRequest struct {
	// The maximum number of results to return in a single page. Leverage the page tokens to iterate through the entire collection.
	// default: 50
	MaxResults string `json:"maxResults"`
	// Page token, returned by a previous call, to request the next page of results
	PageToken string `json:"pageToken,omitempty"`
	// An expression for filtering the results of the request. The syntax is "key:[<value>]". Available keys: owner, lastModified in ms (>lasModified). Multiple filters can be connected using a pipe |. Note that using different keys in the same filter results in “AND,” while using the same key multiple times in the same filter results in “OR”.
	Filter string `json:"filter"`
	// Min value for reports creation time, in milliseconds since the POSIX epoch. If set, only reports created after or at this timestamp are returned.
	MinCreationTime string `json:"minCreationTime"`
	// Max value for reports creation time, in milliseconds since the POSIX epoch. If set, only reports created before or at this timestamp are returned.
	MaxCreationTime string `json:"maxCreationTime"`
}

type ExternalAPIListArgsReq struct {
	BudgetRequest  *BudgetsRequest
	Email          string
	CustomerID     string
	IsDoitEmployee bool
}

type BudgetList struct {
	// Page token, returned by a previous call, to request the next page of results
	PageToken string `json:"pageToken,omitempty"`
	// Budgets rows count
	RowCount int `json:"rowCount"`
	// Array of Budgets
	Budgets []customerapi.SortableItem `json:"budgets"`
}

// swagger
type BudgetListResponse struct {
	// Page token, returned by a previous call, to request the next page of results
	PageToken string `json:"pageToken,omitempty"`
	// Budgets rows count
	RowCount int `json:"rowCount"`
	// Array of Budgets
	Budgets []BudgetListItem `json:"budgets"`
}

type BudgetListItem struct {
	ID                        string           `json:"id" firestore:"id"`
	BudgetName                string           `json:"budgetName" firestore:"name"`
	Owner                     string           `json:"owner" firestore:"owner"`
	CreateTime                int64            `sortKey:"createTime" json:"createTime" firestore:"timeCreated"`  // in ms since the POSIX epoch
	UpdateTime                int64            `sortKey:"updateTime" json:"updateTime" firestore:"timeModified"` // in ms since the POSIX epoch
	Amount                    float64          `json:"amount"`
	Currency                  string           `json:"currency"`
	TimeInterval              string           `json:"timeInterval"`
	StartPeriod               int64            `json:"startPeriod"`               // in ms since the POSIX epoch
	EndPeriod                 int64            `json:"endPeriod"`                 // in ms since the POSIX epoch
	ForecastedUtilizationDate *int64           `json:"forecastedUtilizationDate"` // in ms since the POSIX epoch
	CurrentUtilization        float64          `json:"currentUtilization"`
	Scope                     []string         `json:"scope"`
	AlertThresholds           []AlertThreshold `json:"alertThresholds"`
	URL                       string           `json:"url"`
}

func (b BudgetListItem) GetID() string {
	return b.ID
}

type AlertThreshold struct {
	Percentage float64 `json:"percentage"`
	Amount     float64 `json:"amount"`
}

type ExternalBudgetAlert struct {
	Percentage     float64 `json:"percentage" firestore:"percentage"`
	ForecastedDate *int64  `json:"forecastedDate" firestore:"forecastedDate"`
	Triggered      bool    `json:"triggered" firestore:"triggered"`
}

type DeleteManyBudgetRequest struct {
	IDs        []string `json:"ids"`
	Email      string   `json:"-"`
	CustomerID string   `json:"-"`
}
