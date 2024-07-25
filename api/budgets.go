package api

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	domainAttributions "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	budgetSvc "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/externalapi/domain/errormsg"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainExternalReport "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/externalreport"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

// swagger:parameters idOfBudgets
type BudgetsRequest struct {
	// The maximum number of results to return in a single page. Leverage the page tokens to iterate through the entire collection.
	// default: 50
	MaxResults int `json:"maxResults"`
	// Page token, returned by a previous call, to request the next page of results
	PageToken string `json:"pageToken,omitempty"`
	// An expression for filtering the results of the request. The syntax is "key:[<value>]". e.g: "type:preset". Multiple filters can be connected using a pipe |. Note that using different keys in the same filter results in “AND,” while using the same key multiple times in the same filter results in “OR”.
	// Available filters: owner, lastModified where lastModified means requesting results modified after this date in milliseconds since the POSIX epoch
	Filter string `json:"filter"`
	// Min value for reports creation time, in milliseconds since the POSIX epoch. If set, only reports created after or at this timestamp are returned.
	MinCreationTime int64 `json:"minCreationTime"`
	// Max value for reports creation time, in milliseconds since the POSIX epoch. If set, only reports created before or at this timestamp are returned.
	MaxCreationTime int64 `json:"maxCreationTime"`
}

// swagger:parameters idOfDeletedBudget
type BudgetDeleteRequest struct {
	// budget ID, identifying the report
	// in:path
	ID *string `json:"id"`
}

type ExternalBudgetAlert struct {
	Percentage     float64 `json:"percentage" firestore:"percentage"`
	ForecastedDate *int64  `json:"forecastedDate" firestore:"forecastedDate"`
	Triggered      bool    `json:"triggered" firestore:"triggered"`
}

type BudgetCreateUpdateAlert struct {
	Percentage float64 `json:"percentage" firestore:"percentage"`
}

type Budget struct {
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
	// Fixed budget end date
	// required: true(if budget type is fixed)
	EndPeriod *int64 `json:"endPeriod"`
	// Budget start Date
	// required: true
	StartPeriod *int64 `json:"startPeriod"`
	// swagger:ignore
	CreateTime int64 `json:"createTime"`
	// swagger:ignore
	UpdateTime int64 `json:"updateTime"`
	// swagger:ignore
	CurrentUtilization float64 `json:"currentUtilization"`
	// swagger:ignore
	ForecastedUtilization float64 `json:"forecastedUtilization"`
	// swagger:ignore
	Draft bool `json:"-"`
	// The data source that will be used to query the data from
	// default: "billing"
	DataSource *domainExternalReport.ExternalDataSource `json:"dataSource"`
}

type BudgetCreateUpdateRequest struct {
	// Budget Name
	Name *string `json:"name"`
	// Budget description
	// default: ""
	Description *string `json:"description"`

	Public *collab.CollaboratorRole `json:"public"`
	// List of up to three thresholds defined as percentage of amount
	// default: []
	Alerts *[3]BudgetCreateUpdateAlert `json:"alerts"`
	// List of permitted users to view/edit the report
	Collaborators []collab.Collaborator `json:"collaborators"`
	// List of emails to notify when reaching alert threshold
	Recipients []string `json:"recipients"`
	// List of slack channels to notify when reaching alert threshold
	// default: []
	RecipientsSlackChannels []common.SlackChannel `json:"recipientsSlackChannels"`
	// List of attributions that defines that budget scope
	Scope []string `json:"scope"`
	// Budget period amount
	// required: true(if usePrevSpend is false)
	Amount *float64 `json:"amount"`
	// Use the last period's spend as the target amount for recurring budgets
	// default: false
	UsePrevSpend *bool `json:"usePrevSpend"`
	// Budget currency can be one of: ["USD","ILS","EUR","GBP","AUD","CAD","DKK","NOK","SEK","BRL","SGD","MXN","CHF","MYR","TWD","EGP","ZAR","JPY","IDR"]
	Currency *string `json:"currency"`
	// Periodical growth percentage in recurring budget
	// default: 0
	GrowthPerPeriod *float64 `json:"growthPerPeriod"`
	// Budget metric - currently fixed to "cost"
	// default: "cost"
	Metric *string `json:"metric"`
	// Recurring budget interval can be on of: ["day", "week", "month", "quarter","year"]
	TimeInterval *string `json:"timeInterval"`
	// budget type can be one of: ["fixed", "recurring"]
	Type *string `json:"type"`
	// Fixed budget end date
	// required: true(if budget type is fixed)
	EndPeriod *int64 `json:"endPeriod"`
	// Budget start Date
	StartPeriod *int64 `json:"startPeriod"`
}

// swagger:parameters idOfBudget
type BudgetRequest struct {
	// budget ID, identifying the report
	// in:path
	// required: true
	ID string `json:"id"`
}

type BudgetRequestData struct {
	BudgetRequest      *BudgetsRequest
	Email              string
	BudgetID           string
	Budget             *Budget
	CustomerID         string
	MinCreationTimeStr string
	MaxCreationTimeStr string
	DoitEmployee       bool
}

func (s *APIV1Service) DeleteBudget(ctx context.Context, requestData *BudgetRequestData) *ErrorResponse {
	l := s.loggerProvider(ctx)
	fs := s.Firestore(ctx)

	budgetRef := fs.Collection("cloudAnalytics").Doc("budgets").Collection("cloudAnalyticsBudgets").Doc(requestData.BudgetID)

	docSnap, err := budgetRef.Get(ctx)
	if err != nil {
		l.Error(err)
		return &ErrorResponse{404, err, ErrorNotFound}
	}

	var b budget.Budget

	if err := docSnap.DataTo(&b); err != nil {
		l.Error(err)
		return &ErrorResponse{500, err, ErrorInternalError}
	}

	bOwner, err := s.getOwnerEmailFromCollaborators(b.Collaborators)
	if err != nil {
		l.Error(err)
		return &ErrorResponse{500, err, ErrorInternalError}
	}

	if *bOwner != requestData.Email {
		return &ErrorResponse{403, errors.New(ErrorForbidden), ErrorForbidden}
	}

	err = s.labelsDal.DeleteObjectWithLabels(ctx, budgetRef)
	if err != nil {
		l.Error(err)
		return &ErrorResponse{500, err, ErrorInternalError}
	}

	return nil
}

func (s *APIV1Service) CreateBudget(ctx context.Context, requestData *BudgetRequestData) (*Budget, *ErrorResponse) {
	l := s.loggerProvider(ctx)
	fs := s.Firestore(ctx)

	internalbudget, err := s.mapRequestToInternalBudget(ctx, *requestData.Budget, requestData.CustomerID, requestData.Email)
	if err != nil {
		l.Error(err)
		return nil, &ErrorResponse{500, err, err.Error()}
	}

	budgetCollectionRef := fs.Collection("cloudAnalytics").Doc("budgets").Collection("cloudAnalyticsBudgets")

	budgetRef, _, err := budgetCollectionRef.Add(ctx, internalbudget)
	if err != nil {
		l.Error(err)
		return nil, &ErrorResponse{500, err, ErrorInternalError}
	}

	go func(customerID, budgetID, email string) {
		if err := s.budgets.RefreshBudgetUsageData(ctx, customerID, budgetID, email); err != nil {
			l.Errorf("error refreshing budget %s with error: %s", budgetID, err)
		}
	}(requestData.CustomerID, budgetRef.ID, requestData.Email)

	budgetWithData, err := s.budgets.GetBudget(ctx, budgetRef.ID)
	if err != nil {
		l.Error(err)
		return nil, &ErrorResponse{500, err, ErrorInternalError}
	}

	budgetWithData.ID = budgetRef.ID

	responseBudget, err := s.mapInternalBudgetToResponseBudget(budgetWithData)
	if err != nil {
		l.Error(err)
		return nil, &ErrorResponse{500, err, ErrorInternalError}
	}

	return responseBudget, nil
}

func (s *APIV1Service) mapRequestToInternalBudget(ctx context.Context, b Budget, customerID, email string) (*budget.Budget, error) {
	fs := s.Firestore(ctx)

	if !b.validateAmount() {
		return nil, errors.New(ErrorBudgetInvalidAmount)
	}

	usePrevSpend := false
	if b.UsePrevSpend != nil {
		usePrevSpend = *b.UsePrevSpend
	}

	growthPerPeriod, err := s.validateGrowthPeriod(b.GrowthPerPeriod)
	if err != nil {
		return nil, err
	}

	currency, err := s.validateCurrency(b.Currency)
	if err != nil {
		return nil, err
	}

	budgetType, err := s.validateType(b.Type)
	if err != nil {
		return nil, err
	}

	if budgetType == budget.Fixed && b.EndPeriod == nil {
		return nil, errors.New(ErrorBudgetFixedWithoutEndPeriod)
	}

	allowGrowth := false
	if b.GrowthPerPeriod != nil && *b.GrowthPerPeriod > 0 {
		allowGrowth = true
	}

	var timeInterval report.TimeInterval
	if budgetType == budget.Recurring {
		timeInterval, err = s.validateTimeInterval(b.TimeInterval)
		if err != nil {
			return nil, err
		}
	}

	startPeriod, endPeriod, err := s.getTimeSettings(
		b.StartPeriod,
		b.EndPeriod,
		budgetType,
		timeInterval)
	if err != nil {
		return nil, err
	}

	scope := make([]*firestore.DocumentRef, 0)

	scope, err = s.validateScope(ctx, b.Scope)
	if err != nil {
		return nil, err
	}

	customerRef := fs.Collection("customers").Doc(customerID)

	if err := s.validateCollaborators(ctx, b.Collaborators, customerID, email); err != nil {
		return nil, err
	}

	recipients, err := s.validateRecipients(b.Recipients, b.Collaborators)
	if err != nil {
		return nil, err
	}

	recipientsSlackChannels := make([]common.SlackChannel, 0)
	if b.RecipientsSlackChannels != nil {
		recipientsSlackChannels = b.RecipientsSlackChannels
	}

	description := s.validateDescription(b.Description)

	if b.Name == nil || *b.Name == "" {
		return nil, fmt.Errorf(ErrorBudgetInvalidName)
	}

	var internalAlerts [3]budget.BudgetAlert

	if b.Alerts != nil {
		for i, a := range b.Alerts {
			internalAlerts[i] = budget.BudgetAlert{
				Percentage: a.Percentage,
			}
		}
	}

	var dataSource *report.DataSource

	var dataSourceValidationErrors []errormsg.ErrorMsg

	if b.DataSource == nil {
		dataSourceBilling := report.DataSourceBilling
		dataSource = &dataSourceBilling
	} else if *b.DataSource == domainExternalReport.ExternalDataSourceBQLens {
		return nil, errors.New(ErrorBudgetInvalidDataSource)
	} else {
		dataSource, dataSourceValidationErrors = b.DataSource.ToInternal()
		if len(dataSourceValidationErrors) > 0 {
			return nil, dataSourceValidationErrors[0]
		}
	}

	name := b.Name

	config := budget.BudgetConfig{
		Currency:        currency,
		Metric:          report.MetricCost,
		Type:            budgetType,
		AllowGrowth:     allowGrowth,
		UsePrevSpend:    usePrevSpend,
		GrowthPerPeriod: growthPerPeriod,
		StartPeriod:     *startPeriod,
		EndPeriod:       *endPeriod,
		TimeInterval:    timeInterval,
		Alerts:          internalAlerts,
		Scope:           scope,
		DataSource:      dataSource,
	}
	if !usePrevSpend {
		config.Amount = *b.Amount
		config.OriginalAmount = *b.Amount
	}

	internalBudget := budget.Budget{
		Config: &config,
		Access: collab.Access{
			Collaborators: b.Collaborators,
			Public:        b.Public,
		},
		Name:                    *name,
		Description:             description,
		TimeModified:            time.Now().UTC(),
		Customer:                customerRef,
		Recipients:              recipients,
		IsValid:                 true,
		RecipientsSlackChannels: recipientsSlackChannels,
	}
	if b.CreateTime != 0 {
		internalBudget.TimeCreated = time.Unix(b.CreateTime, 0)
	} else {
		internalBudget.TimeCreated = time.Now().UTC()
	}

	return &internalBudget, nil
}

func (s *APIV1Service) mapInternalBudgetToResponseBudget(internalBudget *budget.Budget) (*Budget, error) {
	scope := make([]string, 0)
	for _, s := range internalBudget.Config.Scope {
		scope = append(scope, s.ID)
	}

	currency := string(internalBudget.Config.Currency)

	var timeInterval string
	if internalBudget.Config.Type == budget.Recurring {
		timeInterval = string(internalBudget.Config.TimeInterval)
	}

	metric, err := domainQuery.GetMetricString(internalBudget.Config.Metric)
	if err != nil {
		return nil, err
	}

	budgetType, err := budgetSvc.GetBudgetTypeString(internalBudget.Config.Type)
	if err != nil {
		return nil, err
	}

	endPeriod := internalBudget.Config.EndPeriod.UnixMilli()
	startPeriod := internalBudget.Config.StartPeriod.UnixMilli()
	currentUtilization := math.Round(internalBudget.Utilization.Current*100) / 100
	forcastedUtilization := math.Round(internalBudget.Utilization.Forecasted*100) / 100

	var alerts [3]ExternalBudgetAlert
	for i, a := range internalBudget.Config.Alerts {
		alerts[i] = ExternalBudgetAlert{
			Percentage: a.Percentage,
			Triggered:  a.Triggered,
		}

		if !common.IsNil(a.ForecastedDate) && !a.ForecastedDate.IsZero() {
			fd := a.ForecastedDate.UnixMilli()
			alerts[i].ForecastedDate = &fd
		}
	}

	if internalBudget.Config.DataSource == nil || *internalBudget.Config.DataSource == "" {
		internalBudget.Config.DataSource = report.DataSourceBilling.Pointer()
	}

	if *internalBudget.Config.DataSource != report.DataSourceBilling &&
		*internalBudget.Config.DataSource != report.DataSourceBillingDataHub {
		return nil, errors.New(ErrorBudgetInvalidDataSource)
	}

	dataSource, dataSourceValidationErrors := domainExternalReport.NewExternalDatasourceFromInternal(*internalBudget.Config.DataSource)
	if len(dataSourceValidationErrors) > 0 {
		return nil, dataSourceValidationErrors[0]
	}

	b := Budget{
		ID:                      &internalBudget.ID,
		Name:                    &internalBudget.Name,
		Description:             &internalBudget.Description,
		Public:                  internalBudget.Public,
		Alerts:                  &alerts,
		Collaborators:           internalBudget.Collaborators,
		Recipients:              internalBudget.Recipients,
		RecipientsSlackChannels: internalBudget.RecipientsSlackChannels,
		Scope:                   scope,
		Amount:                  &internalBudget.Config.Amount,
		Currency:                &currency,
		GrowthPerPeriod:         &internalBudget.Config.GrowthPerPeriod,
		TimeInterval:            &timeInterval,
		CurrentUtilization:      currentUtilization,
		ForecastedUtilization:   forcastedUtilization,
		Metric:                  &metric,
		Type:                    &budgetType,
		EndPeriod:               &endPeriod,
		StartPeriod:             &startPeriod,
		CreateTime:              internalBudget.TimeCreated.UnixMilli(),
		UpdateTime:              internalBudget.TimeModified.UnixMilli(),
		UsePrevSpend:            &internalBudget.Config.UsePrevSpend,
		DataSource:              dataSource,
	}

	return &b, nil
}

func (s *APIV1Service) UpdateBudget(ctx context.Context, requestData *BudgetRequestData) (*Budget, *ErrorResponse) {
	l := s.loggerProvider(ctx)

	currentBudget, err := s.budgets.GetBudget(ctx, requestData.BudgetID)
	if err != nil {
		l.Error(err)

		if status.Code(err) == codes.NotFound {
			return nil, &ErrorResponse{404, errors.New(ErrorNotFound), ErrorNotFound}
		}

		return nil, &ErrorResponse{500, errors.New(ErrorInternalError), ErrorInternalError}
	}

	if !s.isUserPermitted(ctx, currentBudget, requestData.Email) {
		l.Error(fmt.Errorf(ErrorForbidden))
		return nil, &ErrorResponse{403, errors.New(ErrorForbidden), ErrorForbidden}
	}

	if !requestData.Budget.validateAmount() && currentBudget.Config.Amount == 0 {
		l.Error(fmt.Errorf(ErrorBudgetInvalidAmount))
		return nil, &ErrorResponse{400, errors.New(ErrorBudgetInvalidAmount), ErrorBudgetInvalidAmount}
	}

	budgetUpdates, err := s.getBudgetUpdates(ctx, *requestData.Budget, currentBudget, requestData.CustomerID, requestData.Email)
	if err != nil {
		l.Error(err)
		return nil, &ErrorResponse{400, err, ErrorBadRequest}
	}

	if err := s.updateBudget(ctx, requestData.BudgetID, budgetUpdates); err != nil {
		l.Error(err)
		return nil, &ErrorResponse{500, errors.New(ErrorInternalError), ErrorInternalError}
	}

	go func(customerID, budgetID, email string) {
		if err := s.budgets.RefreshBudgetUsageData(ctx, customerID, budgetID, email); err != nil {
			l.Errorf("error refreshing budget %s with error: %s", budgetID, err)
		}
	}(requestData.CustomerID, requestData.BudgetID, requestData.Email)

	updatedBudget, err := s.budgets.GetBudget(ctx, requestData.BudgetID)
	if err != nil {
		l.Error(err)
		return nil, &ErrorResponse{500, errors.New(ErrorInternalError), ErrorInternalError}
	}

	responseBudget, err := s.mapInternalBudgetToResponseBudget(updatedBudget)
	if err != nil {
		l.Error(err)
		return nil, &ErrorResponse{500, errors.New(ErrorInternalError), ErrorInternalError}
	}

	responseBudget.ID = &requestData.BudgetID

	return responseBudget, nil
}

func (s *APIV1Service) updateBudget(ctx context.Context, budgetID string, updates []firestore.Update) error {
	fs := s.Firestore(ctx)

	if len(updates) == 0 {
		return fmt.Errorf(ErrorBudgetNoUpdates)
	}

	budgetRef := fs.Collection("cloudAnalytics").Doc("budgets").Collection("cloudAnalyticsBudgets").Doc(budgetID)
	if _, err := budgetRef.Update(ctx, updates); err != nil {
		return err
	}

	return nil
}

func (s *APIV1Service) isUserPermitted(ctx context.Context, b *budget.Budget, email string) bool {
	for _, collaborator := range b.Collaborators {
		if collaborator.Email == email {
			if collaborator.Role == collab.CollaboratorRoleOwner || collaborator.Role == collab.CollaboratorRoleEditor {
				return true
			}
		}
	}

	return false
}

func (s *APIV1Service) getBudgetUpdates(ctx context.Context, budget Budget, currentBudget *budget.Budget, customerID, email string) ([]firestore.Update, error) {
	updates := make([]firestore.Update, 0)

	if budget.Amount != nil {
		amountUpdate, originalAmountUpdate := s.getAmountUpdates(*budget.Amount)
		updates = append(updates, amountUpdate)
		updates = append(updates, originalAmountUpdate)
	}

	if budget.Amount != nil || budget.Alerts != nil {
		var internalAlerts [3]BudgetCreateUpdateAlert
		for i, a := range budget.Alerts {
			internalAlerts[i] = BudgetCreateUpdateAlert{
				Percentage: a.Percentage,
			}
		}

		alertsUpdate := firestore.Update{
			Path:  "config.alerts",
			Value: internalAlerts,
		}
		updates = append(updates, alertsUpdate)
	}

	if budget.Type != nil {
		typeUpdate, budgetType, err := s.getBudgetTypeUpdate(*budget.Type)
		if err != nil {
			return nil, err
		}

		updates = append(updates, *typeUpdate)
		currentBudget.Config.Type = *budgetType
	}

	if budget.Currency != nil {
		currency, err := s.validateCurrency(budget.Currency)
		if err != nil {
			return nil, err
		}

		currencyUpdate := firestore.Update{
			Path:  "config.currency",
			Value: currency,
		}
		updates = append(updates, currencyUpdate)
	}

	if budget.GrowthPerPeriod != nil {
		growthPerPeriodUpdate := firestore.Update{
			Path:  "config.growthPerPeriod",
			Value: budget.GrowthPerPeriod,
		}
		updates = append(updates, growthPerPeriodUpdate)
	}

	if budget.UsePrevSpend != nil {
		usePrevSpendUpdate := firestore.Update{
			Path:  "config.usePrevSpend",
			Value: budget.UsePrevSpend,
		}
		updates = append(updates, usePrevSpendUpdate)
	}

	if budget.TimeInterval != nil && *budget.TimeInterval != "" {
		period, err := s.validateTimeInterval(budget.TimeInterval)
		if err != nil {
			return nil, err
		}

		periodUpdate := firestore.Update{
			Path:  "config.timeInterval",
			Value: period,
		}
		updates = append(updates, periodUpdate)
		currentBudget.Config.TimeInterval = period
	}

	if budget.Scope != nil {
		scope, err := s.validateScope(ctx, budget.Scope)
		if err != nil {
			return nil, err
		}

		scopeUpdate := firestore.Update{
			Path:  "config.scope",
			Value: scope,
		}
		updates = append(updates, scopeUpdate)
	}

	if budget.Collaborators != nil {
		if err := s.validateUpdateCollaborators(ctx, budget.Collaborators, customerID, email, currentBudget.GetBudgetOwner()); err != nil {
			return nil, errors.New(ErrorBudgetInvalidCollaborator)
		}

		collaboratorsUpdate := firestore.Update{
			Path:  "collaborators",
			Value: budget.Collaborators,
		}
		updates = append(updates, collaboratorsUpdate)
	}

	if budget.Public != nil && (email == currentBudget.GetBudgetOwner() || currentBudget.EmailIsEditor(email)) {
		publicAccessUpdate := firestore.Update{
			Path:  "public",
			Value: budget.Public,
		}
		updates = append(updates, publicAccessUpdate)
	}

	if budget.Name != nil {
		nameUpdate := firestore.Update{
			Path:  "name",
			Value: budget.Name,
		}
		updates = append(updates, nameUpdate)
	}

	if budget.Description != nil {
		descriptionUpdate := firestore.Update{
			Path:  "description",
			Value: budget.Description,
		}
		updates = append(updates, descriptionUpdate)
	}

	if budget.TimeInterval != nil || budget.Type != nil || budget.StartPeriod != nil || budget.EndPeriod != nil {
		startPeriodUpdate, endPeriodUpdate, err := s.getTimeSettingsUpdates(budget, currentBudget)
		if err != nil {
			return nil, err
		}

		updates = append(updates, *startPeriodUpdate)
		updates = append(updates, *endPeriodUpdate)
	}

	if budget.RecipientsSlackChannels != nil {
		updates = append(updates, firestore.Update{
			Path:  "recipientsSlackChannels",
			Value: budget.RecipientsSlackChannels,
		},
		)
	}

	if budget.Recipients != nil {
		recipients, err := s.getRecipientsUpdates(budget, currentBudget)
		if err != nil {
			return nil, err
		}

		recipientsUpdate := firestore.Update{
			Path:  "recipients",
			Value: recipients,
		}
		updates = append(updates, recipientsUpdate)
	}

	if budget.DataSource != nil {
		if *budget.DataSource == domainExternalReport.ExternalDataSourceBQLens || !budget.DataSource.ValidateDataSource() {
			return nil, errors.New(ErrorBudgetInvalidDataSource)
		}

		dataSourceUpdate := firestore.Update{
			Path:  "config.dataSource",
			Value: budget.DataSource,
		}
		updates = append(updates, dataSourceUpdate)
	}

	updates = append(updates, firestore.Update{
		Path:  "timeModified",
		Value: time.Now(),
	})

	return updates, nil
}

func (s *APIV1Service) getTimeSettingsUpdates(b Budget, currentBudget *budget.Budget) (*firestore.Update, *firestore.Update, error) {
	var start, end *int64
	if b.StartPeriod != nil {
		start = b.StartPeriod
	} else {
		s := common.ToUnixMillis(currentBudget.Config.StartPeriod)
		start = &s
	}

	if currentBudget.Config.Type == budget.Fixed {
		if b.EndPeriod != nil {
			end = b.EndPeriod
		} else {
			e := common.ToUnixMillis(currentBudget.Config.EndPeriod)
			end = &e
		}
	}

	startPeriod, endPeriod, err := s.getTimeSettings(start, end, currentBudget.Config.Type, currentBudget.Config.TimeInterval)
	if err != nil {
		return nil, nil, err
	}

	startPeriodUpdate := firestore.Update{
		Path:  "config.startPeriod",
		Value: startPeriod,
	}
	endPeriodUpdate := firestore.Update{
		Path:  "config.endPeriod",
		Value: endPeriod,
	}

	return &startPeriodUpdate, &endPeriodUpdate, nil
}

func (s *APIV1Service) getAmountUpdates(amount float64) (firestore.Update, firestore.Update) {
	amountUpdate := firestore.Update{
		Path:  "config.amount",
		Value: amount,
	}
	originalAmountUpdate := firestore.Update{
		Path:  "config.originalAmount",
		Value: amount,
	}

	return amountUpdate, originalAmountUpdate
}

func (s *APIV1Service) getBudgetTypeUpdate(requestBudgetType string) (*firestore.Update, *budget.BudgetType, error) {
	budgetType, err := s.validateType(&requestBudgetType)
	if err != nil {
		return nil, nil, err
	}

	typeUpdate := firestore.Update{
		Path:  "config.type",
		Value: budgetType,
	}

	return &typeUpdate, &budgetType, nil
}

func (s *APIV1Service) getRecipientsUpdates(budget Budget, currentBudget *budget.Budget) ([]string, error) {
	collaborators := currentBudget.Collaborators
	if budget.Collaborators != nil {
		collaborators = budget.Collaborators
	}

	recipients, err := s.validateRecipients(budget.Recipients, collaborators)
	if err != nil {
		return nil, err
	}

	return recipients, nil
}

func (b *Budget) validateAmount() bool {
	// amount is required if usePrevSpend is nil or false
	return (b.UsePrevSpend != nil && *b.UsePrevSpend) || (b.Amount != nil && *b.Amount > 0)
}

func (s *APIV1Service) validateCurrency(currency *string) (fixer.Currency, error) {
	if currency != nil && fixer.SupportedCurrency(*currency) {
		return fixer.FromString(*currency), nil
	}

	return "", fmt.Errorf(ErrorBudgetInvalidCurrency)
}

func (s *APIV1Service) validateGrowthPeriod(growthPerPeriod *float64) (float64, error) {
	if growthPerPeriod != nil && *growthPerPeriod < 0 {
		return -1, fmt.Errorf(ErrorBudgetInvalidGrowthPerPeriod)
	}

	if growthPerPeriod == nil {
		return 0, nil
	}

	return *growthPerPeriod, nil
}

func (s *APIV1Service) validateType(budgetType *string) (budget.BudgetType, error) {
	if budgetType == nil {
		return -1, fmt.Errorf(ErrorBudgetInvalidType)
	}

	switch *budgetType {
	case "fixed":
		return budget.Fixed, nil
	case "recurring":
		return budget.Recurring, nil
	default:
		return -1, fmt.Errorf(ErrorBudgetInvalidType)
	}
}

func (s *APIV1Service) validateTimeInterval(timeInterval *string) (report.TimeInterval, error) {
	if timeInterval == nil {
		return "", fmt.Errorf(ErrorBudgetInvalidTimeInterval)
	}

	switch *timeInterval {
	case "day":
		return report.TimeIntervalDay, nil
	case "week":
		return report.TimeIntervalWeek, nil
	case "month":
		return report.TimeIntervalMonth, nil
	case "quarter":
		return report.TimeIntervalQuarter, nil
	case "year":
		return report.TimeIntervalYear, nil
	default:
		return "", fmt.Errorf(ErrorBudgetInvalidTimeInterval)
	}
}

func (s *APIV1Service) getTimeSettings(start *int64, end *int64, budgetType budget.BudgetType, period report.TimeInterval) (*time.Time, *time.Time, error) {
	var startPeriod, endPeriod time.Time

	if start != nil {
		startPeriod = common.EpochMillisecondsToTime(*start)
	}

	if budgetType == budget.Fixed {
		if end != nil {
			endPeriod = common.EpochMillisecondsToTime(*end)
		}
	}

	if budgetType == budget.Recurring {
		from := startPeriod

		to := startPeriod

		switch period {
		case report.TimeIntervalDay:
			to = to.AddDate(0, 0, 1)
		case report.TimeIntervalWeek:
			from, to = s.getWeekIntervalTimeSettings(from, to)
		case report.TimeIntervalMonth:
			to = time.Date(to.Year(), to.Month()+1, 1, 0, 0, 0, 0, time.UTC)
			from = time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
		case report.TimeIntervalQuarter:
			from, to = s.getQuarterIntervalTimeSettings(from, to)
		case report.TimeIntervalYear:
			from = time.Date(from.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
			to = from.AddDate(+1, 0, 0)
		default:
		}

		startPeriod = from
		endPeriod = to
	}

	return &startPeriod, &endPeriod, nil
}

func (s *APIV1Service) getWeekIntervalTimeSettings(from time.Time, to time.Time) (time.Time, time.Time) {
	for i := 0; i < 7; i++ {
		if to.Weekday() == time.Sunday {
			break
		}

		to = to.AddDate(0, 0, 1)
	}

	for i := 0; i < 7; i++ {
		if from.Weekday() == time.Monday {
			break
		}

		from = from.AddDate(0, 0, -1)
	}

	return from, to
}

func (s *APIV1Service) getQuarterIntervalTimeSettings(from time.Time, to time.Time) (time.Time, time.Time) {
	from = time.Date(from.Year(), from.Month()-(from.Month()-1)%3, 1, 0, 0, 0, 0, time.UTC)
	to = from.AddDate(0, +3, 0)

	return from, to
}

func (s *APIV1Service) validateScope(ctx context.Context, attributions []string) ([]*firestore.DocumentRef, error) {
	fs := s.Firestore(ctx)

	if len(attributions) < 1 {
		return nil, fmt.Errorf(ErrorBudgetInvalidScope)
	}

	attrCollection := fs.Collection("dashboards").Doc("google-cloud-reports").Collection("attributions")
	scope := make([]*firestore.DocumentRef, 0)

	for _, attribution := range attributions {
		docRef := attrCollection.Doc(attribution)

		doc, err := docRef.Get(ctx)
		if err != nil {
			return nil, fmt.Errorf(ErrorBudgetInvalidScope)
		}

		attributionType, err := doc.DataAt("type")
		if err != nil {
			return nil, fmt.Errorf(ErrorBudgetInvalidScope)
		}

		if attributionType.(string) == string(domainAttributions.ObjectTypeManaged) {
			return nil, fmt.Errorf(ErrorBudgetInvalidScope)
		}

		scope = append(scope, docRef)
	}

	return scope, nil
}

func (s *APIV1Service) validateCollaborators(ctx context.Context, requestCollaborators []collab.Collaborator, customerID, email string) error {
	ownerCollaborator := collab.Collaborator{
		Email: email,
		Role:  collab.CollaboratorRoleOwner,
	}
	if len(requestCollaborators) == 0 {
		requestCollaborators = append(requestCollaborators, ownerCollaborator)
		return nil // Erez Sh, should we return here ??
	}

	for _, requestCollaborator := range requestCollaborators {
		if err := s.validateCollaborator(ctx, customerID, requestCollaborator.Email); err != nil {
			return err
		}

		if err := s.validateRole(requestCollaborator.Role); err != nil {
			return err
		}
	}

	if err := s.validateOwnerPartOfCollaborators(requestCollaborators, ownerCollaborator); err != nil {
		return err
	}

	return nil
}

func (s *APIV1Service) validateDescription(description *string) string {
	if description == nil {
		return ""
	}

	return *description
}

func (s *APIV1Service) validateRecipients(recipients []string, collaborators []collab.Collaborator) ([]string, error) {
	owner, err := s.getOwnerEmailFromCollaborators(collaborators)
	if err != nil {
		return nil, err
	}

	if len(recipients) == 0 {
		defaultRecipients := make([]string, 1)
		defaultRecipients[0] = *owner

		return defaultRecipients, nil
	}

	for _, recipient := range recipients {
		if !s.isEmailOnCollaborators(recipient, collaborators) {
			newCollaborator := collab.Collaborator{
				Email: recipient,
				Role:  "viewer",
			}
			collaborators = append(collaborators, newCollaborator)
		}
	}

	isOwnerInRecipients := false

	for _, recipient := range recipients {
		if recipient == *owner {
			isOwnerInRecipients = true
			break
		}
	}

	if !isOwnerInRecipients {
		recipients = append(recipients, *owner)
	}

	return recipients, nil
}

func (s *APIV1Service) validateUpdateCollaborators(ctx context.Context, requestCollaborators []collab.Collaborator, customerID, email, owner string) error {
	isUserOwner := email == owner
	if isUserOwner {
		return s.validateCollaborators(ctx, requestCollaborators, customerID, email)
	}

	ownerCollaborator := collab.Collaborator{
		Email: owner,
		Role:  collab.CollaboratorRoleOwner,
	}
	if len(requestCollaborators) == 0 {
		requestCollaborators = append(requestCollaborators, ownerCollaborator)
		return nil // Erez Sh, should we return here ??
	}

	for _, requestCollaborator := range requestCollaborators {
		if err := s.validateCollaborator(ctx, customerID, requestCollaborator.Email); err != nil {
			return err
		}

		if err := s.validateUpdateRole(requestCollaborator.Role); err != nil {
			return err
		}
	}

	if err := s.validateOwnerPartOfCollaborators(requestCollaborators, ownerCollaborator); err != nil {
		return err
	}

	return nil
}

func (s *APIV1Service) validateCollaborator(ctx context.Context, customerID, userEmail string) error {
	fs := s.Firestore(ctx)

	if !isEmailValid(userEmail) {
		return fmt.Errorf(ErrorBudgetInvalidCollaborator)
	}

	doitDomains := []string{"doit.com", "doit-intl.com"}
	emailDomain := strings.Split(userEmail, "@")[1]

	if slice.Contains(doitDomains, emailDomain) {
		return nil
	}

	userSnap, err := fs.Collection("users").Where("email", "==", userEmail).Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	if len(userSnap) != 1 {
		return fmt.Errorf(ErrorBudgetInvalidCollaborator)
	}

	collaboratorCustomer, err := userSnap[0].DataAt("customer.ref")
	if err != nil {
		return err
	}

	collaboratorCustomerRef, ok := collaboratorCustomer.(*firestore.DocumentRef)
	if !ok {
		return fmt.Errorf(ErrorBudgetInvalidCollaborator)
	}

	if collaboratorCustomerRef.ID != customerID {
		return fmt.Errorf(userEmail + ErrorBudgetUserNotConnectedToCustomer)
	}

	collaboratorRoles, err := userSnap[0].DataAt("roles")
	if err != nil {
		return err
	}

	collaboratorRolesArr := collaboratorRoles.([]interface{})
	collaboratorAuthorised := false

	for _, role := range collaboratorRolesArr {
		roleRef := role.(*firestore.DocumentRef)

		roleSnap, err := fs.Collection("roles").Doc(roleRef.ID).Get(ctx)
		if err != nil {
			return err
		}

		permissions, err := roleSnap.DataAt("permissions")
		if err != nil {
			return err
		}

		permissionsArr := permissions.([]interface{})
		for _, permission := range permissionsArr {
			permissionRef := permission.(*firestore.DocumentRef)
			if permissionRef.ID == string(common.PermissionCloudAnalytics) {
				collaboratorAuthorised = true
			}
		}
	}

	if !collaboratorAuthorised {
		return fmt.Errorf(ErrorBudgetInvalidCollaborator)
	}

	return nil
}

func (s *APIV1Service) validateRole(role collab.CollaboratorRole) error {
	switch role {
	case collab.CollaboratorRoleOwner, collab.CollaboratorRoleEditor, collab.CollaboratorRoleViewer:
		return nil
	default:
		return fmt.Errorf(ErrorBudgetInvalidRole)
	}
}
func (s *APIV1Service) validateUpdateRole(role collab.CollaboratorRole) error {
	switch role {
	case collab.CollaboratorRoleEditor, collab.CollaboratorRoleViewer:
		return nil
	default:
		return fmt.Errorf(ErrorBudgetInvalidRole)
	}
}
func (s *APIV1Service) validateOwnerPartOfCollaborators(budgetCollaborators []collab.Collaborator, ownerCollaborator collab.Collaborator) error {
	isUserACollaboratorAsOwner := false

	for _, collaborator := range budgetCollaborators {
		if collaborator.Email == ownerCollaborator.Email {
			isUserACollaboratorAsOwner = true

			if collaborator.Role != collab.CollaboratorRoleOwner {
				return fmt.Errorf(ErrorBudgetInvalidOwnerRole)
			}
		}
	}

	if !isUserACollaboratorAsOwner {
		// Erez Sh, budgetCollaborators is not used, remove this?
		budgetCollaborators = append(budgetCollaborators, ownerCollaborator)
	}

	return nil
}

func (s *APIV1Service) getOwnerEmailFromCollaborators(collaborators []collab.Collaborator) (*string, error) {
	for _, collaborator := range collaborators {
		if collaborator.Role == collab.CollaboratorRoleOwner {
			return &collaborator.Email, nil
		}
	}

	return nil, fmt.Errorf(ErrorBudgetMissingOwner)
}

func (s *APIV1Service) isEmailOnCollaborators(email string, collaborators []collab.Collaborator) bool {
	for _, collaborator := range collaborators {
		if collaborator.Email == email {
			return true
		}
	}

	return false
}
