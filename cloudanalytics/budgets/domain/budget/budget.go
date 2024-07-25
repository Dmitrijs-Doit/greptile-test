package budget

import (
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
)

type BudgetType int

// Budget types
const (
	Fixed BudgetType = iota
	Recurring
)

type Budget struct {
	collab.Access
	Recipients              []string                 `json:"recipients" firestore:"recipients"`
	RecipientsSlackChannels []common.SlackChannel    `json:"recipientsSlackChannels" firestore:"recipientsSlackChannels"`
	Customer                *firestore.DocumentRef   `json:"customer" firestore:"customer"`
	Description             string                   `json:"description" firestore:"description"`
	Name                    string                   `json:"name" firestore:"name"`
	TimeCreated             time.Time                `json:"timeCreated" firestore:"timeCreated"`
	TimeModified            time.Time                `json:"timeModified" firestore:"timeModified"`
	TimeRefreshed           time.Time                `json:"timeRefreshed" firestore:"timeRefreshed"`
	Config                  *BudgetConfig            `json:"config" firestore:"config"`
	ID                      string                   `json:"id" firestore:"-"`
	Utilization             BudgetUtilization        `json:"utilization" firestore:"utilization"`
	IsValid                 bool                     `json:"isValid" firestore:"isValid"`
	Draft                   bool                     `json:"-" firestore:"draft"`
	Labels                  []*firestore.DocumentRef `json:"labels" firestore:"labels"`
}

type BudgetAlert struct {
	Percentage     float64    `json:"percentage" firestore:"percentage"`
	ForecastedDate *time.Time `json:"forecastedDate" firestore:"forecastedDate"`
	Triggered      bool       `json:"triggered" firestore:"triggered"`
}

type BudgetConfig struct {
	Amount          float64                  `json:"amount" firestore:"amount"`
	Currency        fixer.Currency           `json:"currency" firestore:"currency"`
	StartPeriod     time.Time                `json:"startPeriod" firestore:"startPeriod"`
	EndPeriod       time.Time                `json:"endPeriod" firestore:"endPeriod"`
	Metric          report.Metric            `json:"metric" firestore:"metric"`
	TimeInterval    report.TimeInterval      `json:"timeInterval" firestore:"timeInterval"`
	Type            BudgetType               `json:"type" firestore:"type"`
	GrowthPerPeriod float64                  `json:"growthPerPeriod" firestore:"growthPerPeriod"`
	OriginalAmount  float64                  `json:"originalAmount" firestore:"originalAmount"`
	UsePrevSpend    bool                     `json:"usePrevSpend" firestore:"usePrevSpend"`
	AllowGrowth     bool                     `json:"allowGrowth" firestore:"allowGrowth"`
	Scope           []*firestore.DocumentRef `json:"scope" firestore:"scope"`
	Alerts          [3]BudgetAlert           `json:"alerts" firestore:"alerts"`
	DataSource      *report.DataSource       `json:"dataSource" firestore:"dataSource"`
}

type BudgetUtilization struct {
	Current                   float64    `json:"current" firestore:"current"`
	Forecasted                float64    `json:"forecasted" firestore:"forecasted"`
	ForecastedTotalAmountDate *time.Time `json:"forecastedTotalAmountDate" firestore:"forecastedTotalAmountDate"`
	ShouldSendForecastAlert   bool       `json:"shouldSendForecastAlert" firestore:"shouldSendForecastAlert"`
	PreviousForecastedDate    *time.Time `json:"previousForecastedDate" firestore:"previousForecastedDate"`
}

type BudgetCustomerPair struct {
	CustomerID string `json:"customerId"`
	BudgetID   string `json:"budgetId"`
}

type BudgetNotificationType string

const (
	BudgetNotificationTypeForecast  BudgetNotificationType = "forecast"
	BudgetNotificationTypeThreshold BudgetNotificationType = "threshold"
)

type BudgetNotification struct {
	Name              string                 `json:"name" firestore:"name"`
	Type              BudgetNotificationType `json:"type" firestore:"type"`
	BudgetID          string                 `json:"budgetId" firestore:"budgetId"`
	Customer          *firestore.DocumentRef `json:"customer" firestore:"customer"`
	AlertDate         time.Time              `json:"alertDate" firestore:"alertDate"`
	AlertAmount       string                 `json:"alertAmount" firestore:"alertAmount"`
	AlertPercentage   float64                `json:"alertPercentage" firestore:"alertPercentage"`
	CurrencySymbol    string                 `json:"currencySymbol" firestore:"currencySymbol"`
	CurrentAmount     string                 `json:"currentAmount" firestore:"currentAmount"`
	CurrentPercentage float64                `json:"currentPercentage" firestore:"currentPercentage"`
	ForcastedDate     *time.Time             `json:"forecastedDate" firestore:"forecastedDate"`
	ExpireBy          time.Time              `json:"expireBy" firestore:"expireBy"`
	Created           *time.Time             `json:"created" firestore:"created"`
	Recipients        []string               `json:"recipients" firestore:"recipients"`
}

const (
	DayDuration  = 24 * time.Hour
	WeekDuration = 7 * DayDuration
	HoursInWeek  = 24 * 7
)

// GetBudgetOwner : return owner email from collaborators list
func (b *Budget) GetBudgetOwner() string {
	for _, c := range b.Collaborators {
		if c.Role == collab.CollaboratorRoleOwner {
			return c.Email
		}
	}

	return ""
}

func (b *Budget) GetAttributionIDs() []string {
	attributionIDs := make([]string, 0)
	for _, attr := range b.Config.Scope {
		attributionIDs = append(attributionIDs, attr.ID)
	}

	return attributionIDs
}

func (b *Budget) EmailIsEditor(email string) bool {
	for _, collaborator := range b.Collaborators {
		if collaborator.Email == email && collaborator.Role == collab.CollaboratorRoleEditor {
			return true
		}
	}

	return false
}
