package service

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

func GetBudgetTypeString(budgetType budget.BudgetType) (string, error) {
	switch budgetType {
	case budget.Recurring:
		return "recurring", nil
	case budget.Fixed:
		return "fixed", nil
	default:
		return "", fmt.Errorf("no type found")
	}
}

func mapInternalBudgetToResponseBudget(internalBudget *budget.Budget) (*BudgetAPI, error) {
	if internalBudget.Config == nil {
		return nil, ErrMissingBudgetConfig
	}

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

	budgetType, err := GetBudgetTypeString(internalBudget.Config.Type)
	if err != nil {
		return nil, err
	}

	startPeriod := internalBudget.Config.StartPeriod.UnixMilli()

	var endPeriod *int64

	if internalBudget.Config.Type == budget.Fixed {
		fixedEndTime := internalBudget.Config.EndPeriod.UnixMilli()
		endPeriod = &fixedEndTime
	}
	
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

	b := BudgetAPI{
		ID:                      &internalBudget.ID,
		Name:                    &internalBudget.Name,
		Description:             &internalBudget.Description,
		Public:                  internalBudget.Public,
		Alerts:                  &alerts,
		Collaborators:           internalBudget.Collaborators,
		Recipients:              internalBudget.Recipients,
		Scope:                   scope,
		Amount:                  &internalBudget.Config.Amount,
		Currency:                &currency,
		GrowthPerPeriod:         &internalBudget.Config.GrowthPerPeriod,
		TimeInterval:            &timeInterval,
		CurrentUtilization:      currentUtilization,
		ForecastedUtilization:   forcastedUtilization,
		Metric:                  &metric,
		Type:                    &budgetType,
		StartPeriod:             &startPeriod,
		EndPeriod:               endPeriod,
		CreateTime:              internalBudget.TimeCreated.UnixMilli(),
		UpdateTime:              internalBudget.TimeModified.UnixMilli(),
		UsePrevSpend:            &internalBudget.Config.UsePrevSpend,
		RecipientsSlackChannels: internalBudget.RecipientsSlackChannels,
	}

	return &b, nil
}

func isEmailOnCollaborators(email string, collaborators []collab.Collaborator) bool {
	for _, collaborator := range collaborators {
		if collaborator.Email == email {
			return true
		}
	}

	return false
}

func parseFilter(filterStr string) (*dal.BudgetListFilter, error) {
	filter := dal.BudgetListFilter{
		OrderBy: "createTime",
	}

	if filterStr == "" {
		return &filter, nil
	}

	var owners []string

	lastModified := ""
	filters := strings.Split(filterStr, "|")

	for _, param := range filters {
		splitParam := strings.Split(param, ":")
		if len(splitParam) != 2 {
			return nil, fmt.Errorf(ErrorInvalidFilterKey, param)
		}

		key := splitParam[0]
		value := splitParam[1]

		switch key {
		case "owner":
			owners = append(owners, value)
		case "lastModified":
			if lastModified == "" {
				lastModified = value
			}
		default:
			return nil, fmt.Errorf(ErrorInvalidFilterKey, key)
		}
	}

	if lastModified != "" {
		t, err := common.MsToTime(lastModified)
		if err != nil {
			return nil, fmt.Errorf(ErrorInvalidValue, "lastModified")
		}

		filter.TimeModified = &t
		filter.OrderBy = "updateTime"
	}

	filter.Owners = owners

	return &filter, nil
}

func validateListBudgetsArgs(args *ExternalAPIListArgsReq) (*dal.ListBudgetsArgs, error) {
	filter, err := parseFilter(args.BudgetRequest.Filter)
	if err != nil {
		return nil, err
	}

	listBudgetArgs := dal.ListBudgetsArgs{
		CustomerID:     args.CustomerID,
		Email:          args.Email,
		Filter:         filter,
		MaxResults:     50,
		IsDoitEmployee: args.IsDoitEmployee,
	}

	if args.BudgetRequest.MinCreationTime != "" {
		minCreationTime, err := common.MsToTime(args.BudgetRequest.MinCreationTime)
		if err != nil {
			return nil, fmt.Errorf(ErrorInvalidValue, "minCreationTime")
		}

		listBudgetArgs.MinCreationTime = &minCreationTime
	}

	if args.BudgetRequest.MaxCreationTime != "" {
		maxCreationTime, err := common.MsToTime(args.BudgetRequest.MaxCreationTime)
		if err != nil {
			return nil, fmt.Errorf(ErrorInvalidValue, "maxCreationTime")
		}

		listBudgetArgs.MaxCreationTime = &maxCreationTime
	}

	if args.BudgetRequest.MaxResults != "" {
		maxResults, err := strconv.Atoi(args.BudgetRequest.MaxResults)
		if err != nil {
			return nil, fmt.Errorf(ErrorInvalidValue, "maxResults")
		}

		if maxResults > 250 {
			return nil, ErrorParamMaxResultRange
		}

		if maxResults > 0 {
			listBudgetArgs.MaxResults = maxResults
		}
	}

	if args.BudgetRequest.PageToken != "" {
		listBudgetArgs.PageToken = args.BudgetRequest.PageToken
	}

	return &listBudgetArgs, nil
}

func toListBudgetsAPI(budgets []budget.Budget, customerID string, owners []string, timeModified *time.Time) []customerapi.SortableItem {
	var budgetListItems []customerapi.SortableItem

	for _, b := range budgets {
		var (
			alertThresholds                                []AlertThreshold
			attributions                                   []string
			forecastedTotalAmountDate                      *int64
			amount                                         float64
			currency, timeInterval                         string
			startPeriod, endPeriod, createTime, updateTime int64
		)

		owner := b.GetBudgetOwner()

		if len(owners) > 0 && !slice.Contains(owners, owner) {
			continue
		}

		if timeModified != nil && !b.TimeModified.IsZero() && b.TimeModified.Before(*timeModified) {
			continue
		}

		if b.Config != nil {
			for _, a := range b.Config.Alerts {
				if math.IsInf(a.Percentage, 0) {
					continue
				}

				alertThresholds = append(alertThresholds, AlertThreshold{
					Percentage: a.Percentage,
					Amount:     a.Percentage * b.Config.Amount / 100,
				})
			}

			for _, ref := range b.Config.Scope {
				attributions = append(attributions, ref.ID)
			}

			amount = b.Config.Amount
			startPeriod = b.Config.StartPeriod.UnixMilli()
			endPeriod = b.Config.EndPeriod.UnixMilli()
			currency = string(b.Config.Currency)
			timeInterval = string(b.Config.TimeInterval)
		}

		if b.Utilization.ForecastedTotalAmountDate != nil {
			f := b.Utilization.ForecastedTotalAmountDate.UnixMilli()
			forecastedTotalAmountDate = &f
		}

		if !b.TimeCreated.IsZero() {
			createTime = b.TimeCreated.UnixMilli()
		}

		if !b.TimeModified.IsZero() {
			updateTime = b.TimeModified.UnixMilli()
		}

		item := BudgetListItem{
			ID:                        b.ID,
			BudgetName:                b.Name,
			Owner:                     b.GetBudgetOwner(),
			CreateTime:                createTime,
			UpdateTime:                updateTime,
			URL:                       "https://" + common.Domain + "/customers/" + customerID + "/analytics/budgets/" + b.ID,
			Amount:                    amount,
			Currency:                  currency,
			TimeInterval:              timeInterval,
			StartPeriod:               startPeriod,
			EndPeriod:                 endPeriod,
			ForecastedUtilizationDate: forecastedTotalAmountDate,
			CurrentUtilization:        b.Utilization.Current,
			Scope:                     attributions,
			AlertThresholds:           alertThresholds,
		}
		budgetListItems = append(budgetListItems, item)
	}

	return budgetListItems
}
