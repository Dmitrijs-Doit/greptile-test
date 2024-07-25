package service

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"
	slackgo "github.com/slack-go/slack"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	queryDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	originDomain "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

var allModesFields = []string{queryDomain.FieldProjectID, queryDomain.FieldSKUID, queryDomain.FieldCloudProvider}

func updateBudgetAmount(b *budget.Budget) {
	b.Config.Amount = b.Config.OriginalAmount
	startPeriod := b.Config.StartPeriod.Truncate(budget.DayDuration)
	now := time.Now().UTC().Truncate(budget.DayDuration)
	diff := 0.0

	switch b.Config.TimeInterval {
	case report.TimeIntervalDay:
		diff = now.Sub(startPeriod).Hours() / 24
	case report.TimeIntervalWeek:
		nowYear, nowWeek := now.ISOWeek()
		startYear, startWeek := startPeriod.ISOWeek()
		yearDiff := nowYear - startYear
		diff = float64(nowWeek - startWeek + (yearDiff * 52))
	case report.TimeIntervalMonth:
		nowYear := now.Year()
		startYear := startPeriod.Year()
		nowMonth := now.Month()
		startMonth := startPeriod.Month()
		yearDiff := nowYear - startYear
		monthDiff := int(nowMonth) - int(startMonth)

		if monthDiff < 0 {
			monthDiff += 12
			yearDiff--
		}

		diff = float64((yearDiff * 12) + monthDiff)
	case report.TimeIntervalQuarter:
		nowQuarter := (int(now.Month()) / 3) + 1
		startQuarter := (int(startPeriod.Month()) / 3) + 1
		nowYear := int(now.Year())
		startYear := int(startPeriod.Year())
		yearDiff := nowYear - startYear
		diff = float64(nowQuarter - startQuarter + (yearDiff * 4))
	case report.TimeIntervalYear:
		diff = float64(now.Year()) - float64(startPeriod.Year())
	}

	for i := 0.0; i < diff; i++ {
		b.Config.Amount = b.Config.Amount * (100 + b.Config.GrowthPerPeriod) / 100
	}
}

func (s *BudgetsService) RefreshBudgetUsageData(ctx context.Context, customerID, budgetID, email string) error {
	budget, err := s.GetBudget(ctx, budgetID)
	if err != nil {
		return err
	}

	if err = s.validateBudget(budget); err != nil {
		if err == ErrExpiredBudget {
			return nil
		}

		return err
	}

	result, budget, err := s.getBudgetUsageData(ctx, customerID, email, budget)
	if err != nil {
		return err
	}

	if err = s.updateBudgetRecord(ctx, budgetID, result, budget); err != nil {
		return err
	}

	return nil
}

func (s *BudgetsService) validateBudget(b *budget.Budget) error {
	if b.Config == nil {
		return ErrMissingBudgetConfig
	}

	if len(b.Config.Scope) == 0 {
		return ErrMissingBudgetScope
	}

	if b.Config.StartPeriod.IsZero() {
		return ErrMissingBudgetStartPeriod
	}

	if b.Config.Type == budget.Fixed && b.Config.EndPeriod.IsZero() {
		return ErrInvalidBudgetEndPeriod
	}

	if b.Config.Type == budget.Fixed {
		if b.Config.EndPeriod.Before(b.Config.StartPeriod) {
			return ErrInvalidBudgetEndPeriod
		}

		timeSinceEndPeriod := time.Now().UTC().Truncate(budget.DayDuration).Sub(b.Config.EndPeriod.Truncate(budget.DayDuration))

		if timeSinceEndPeriod.Hours() > budget.HoursInWeek {
			return ErrExpiredBudget
		}
	} else if b.Config.AllowGrowth {
		updateBudgetAmount(b)
	}

	return nil
}

func (s *BudgetsService) updateBudgetRecord(ctx context.Context, budgetID string, result *budget.BudgetUsageDataResult, budget *budget.Budget) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	budgetRef := fs.Collection("cloudAnalytics").Doc("budgets").Collection("cloudAnalyticsBudgets").Doc(budgetID)

	l.Infof("Budget cost amount is: %.2f, and the current utilization amount is: %.2f", budget.Config.Amount, result.Utilization)
	l.Infof("Alert thresholds are: %+v", budget.Config.Alerts)

	updateFields := []firestore.Update{
		{
			FieldPath: []string{"utilization", "current"},
			Value:     result.Utilization,
		},
		{
			FieldPath: []string{"utilization", "forecasted"},
			Value:     budget.Utilization.Forecasted,
		},
		{
			FieldPath: []string{"utilization", "forecastedTotalAmountDate"},
			Value:     budget.Utilization.ForecastedTotalAmountDate,
		},
		{
			FieldPath: []string{"utilization", "previousForecastedDate"},
			Value:     budget.Utilization.PreviousForecastedDate,
		},
		{
			FieldPath: []string{"utilization", "lastPeriod"},
			Value:     result.LastPeriod,
		},
		{
			FieldPath: []string{"utilization", "shouldSendForecastAlert"},
			Value:     budget.Utilization.ShouldSendForecastAlert,
		},
		{
			FieldPath: []string{"config", "alerts"},
			Value:     budget.Config.Alerts,
		},
		{
			FieldPath: []string{"config", "amount"},
			Value:     budget.Config.Amount,
		},
		{
			FieldPath: []string{"timeRefreshed"},
			Value:     time.Now(),
		},
	}

	if budget.Config.UsePrevSpend {
		l.Infof("Updating budget with original amount: %f", budget.Config.OriginalAmount)
		updateFields = append(updateFields, firestore.Update{
			FieldPath: []string{"config", "originalAmount"},
			Value:     budget.Config.OriginalAmount,
		})
	}

	if _, err := budgetRef.Update(ctx, updateFields); err != nil {
		return err
	}

	l.Info("Budget refreshed successfully!")

	return nil
}

func (s *BudgetsService) getBudgetUsageData(ctx context.Context, customerID, email string, b *budget.Budget) (*budget.BudgetUsageDataResult, *budget.Budget, error) {
	queryRequest, err := s.getBudgetQueryRequest(ctx, customerID, b)
	if err != nil {
		return nil, nil, err
	}

	queryResult, err := s.cloudAnalytics.GetQueryResult(ctx, queryRequest, customerID, email)
	if err != nil {
		return nil, nil, err
	}

	lastPeriodSpend := s.getLastPeriod(&queryResult, b)
	utilization, startDate := s.getUtilization(&queryResult, b)

	if b.Config.UsePrevSpend {
		if lastPeriodSpend == 0 && utilization != 0 && startDate != nil {
			lastPeriodSpend = s.getSpeculatedLastPeriod(b, *startDate, utilization)
		}

		b.Config.OriginalAmount = lastPeriodSpend
		if b.Config.AllowGrowth {
			b.Config.Amount = lastPeriodSpend * (100 + b.Config.GrowthPerPeriod) / 100
		} else {
			b.Config.Amount = lastPeriodSpend
		}
	}

	var forecastResult []*budget.BudgetForecastPrediction
	if b.Config.TimeInterval != report.TimeIntervalDay {
		forecastResult, err = s.getBudgetForecast(ctx, queryRequest, &queryResult, b)
		if err != nil {
			return nil, nil, err
		}
	}

	result := budget.BudgetUsageDataResult{
		Utilization: utilization,
		LastPeriod:  lastPeriodSpend,
		Forecast:    forecastResult,
	}

	return &result, b, nil
}

func (s *BudgetsService) getRowDate(row []bigquery.Value) (*time.Time, error) {
	rowYear, err := strconv.Atoi(row[0].(string))
	if err != nil {
		return nil, err
	}

	rowMonth, err := strconv.Atoi(row[1].(string))
	if err != nil {
		return nil, err
	}

	rowDay, err := strconv.Atoi(row[2].(string))
	if err != nil {
		return nil, err
	}

	t := time.Date(rowYear, time.Month(rowMonth), rowDay, 0, 0, 0, 0, time.UTC)

	return &t, nil
}

// getQueryTotalByDate gets the totals from a query results between two dates
// assumes the results are ordered by the time series
func (s *BudgetsService) getQueryTotalByDate(result *cloudanalytics.QueryResult, budget *budget.Budget, startDate, endDate *time.Time) float64 {
	var totals float64

	metricIndex := report.MetricEnumLength + budget.Config.Metric

	for _, row := range result.Rows {
		rowDate, err := s.getRowDate(row)
		if err != nil {
			continue
		}

		// Skip result rows that are not between the provided start and end dates
		if startDate.After(*rowDate) {
			continue
		}

		if endDate != nil && endDate.Before(*rowDate) {
			continue
		}

		if v, ok := row[metricIndex].(float64); ok {
			totals += v
		}
	}

	return totals
}

func (s *BudgetsService) getUtilization(result *cloudanalytics.QueryResult, b *budget.Budget) (float64, *time.Time) {
	startDate := b.Config.StartPeriod.Truncate(budget.DayDuration)
	if startDate.Sub(time.Now().UTC()) > 0 {
		return 0.0, nil
	}

	var totals float64

	if b.Config.Type == budget.Recurring {
		now := time.Now().UTC().Truncate(budget.DayDuration)

		switch b.Config.TimeInterval {
		case report.TimeIntervalDay:
			startDate = now.AddDate(0, 0, -1)
		case report.TimeIntervalWeek:
			startDate = now
			for i := 0; i < 7; i++ {
				if startDate.Weekday() == time.Monday {
					break
				}

				startDate = startDate.AddDate(0, 0, -1)
			}
		case report.TimeIntervalMonth:
			startDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		case report.TimeIntervalQuarter:
			startDate = time.Date(now.Year(), now.Month()-(now.Month()-1)%3, 1, 0, 0, 0, 0, time.UTC)
		case report.TimeIntervalYear:
			startDate = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		default:
		}

		totals = s.getQueryTotalByDate(result, b, &startDate, nil)
	} else {
		endDate := b.Config.EndPeriod.Truncate(budget.DayDuration)
		totals = s.getQueryTotalByDate(result, b, &startDate, &endDate)
	}

	return totals, &startDate
}

func (s *BudgetsService) getSpeculatedLastPeriod(b *budget.Budget, startDate time.Time, utilization float64) float64 {
	var periodHours float64

	var currentHours float64

	if b.Config.Type == budget.Recurring {
		now := time.Now().UTC()

		switch b.Config.TimeInterval {
		case report.TimeIntervalDay:
			periodHours = 24
			currentHours = now.Sub(now.Truncate(time.Hour * 24)).Hours()
		case report.TimeIntervalWeek:
			periodHours = budget.HoursInWeek
			currentHours = now.Sub(startDate).Hours()
		case report.TimeIntervalMonth:
			periodHours = float64(startDate.AddDate(0, 0, -1).Day()) * 24
			currentHours = now.Sub(startDate).Hours()
		case report.TimeIntervalQuarter:
			periodHours = startDate.Sub(startDate.AddDate(0, -3, 0)).Hours()
			currentHours = now.Sub(startDate).Hours()
		case report.TimeIntervalYear:
			periodHours = startDate.Sub(startDate.AddDate(-1, 0, 0)).Hours()
			currentHours = now.Sub(startDate).Hours()
		default:
		}
	}

	if currentHours != 0 {
		return (utilization / currentHours) * periodHours
	}

	return 0
}

func (s *BudgetsService) getLastPeriod(result *cloudanalytics.QueryResult, b *budget.Budget) float64 {
	endDate := b.Config.StartPeriod.Truncate(budget.DayDuration).AddDate(0, 0, -1)
	startDate := b.Config.StartPeriod.Truncate(budget.DayDuration)

	if b.Config.Type == budget.Recurring {
		now := time.Now().UTC().Truncate(budget.DayDuration)

		switch b.Config.TimeInterval {
		case report.TimeIntervalDay:
			startDate = now.AddDate(0, 0, -2)
			endDate = now.AddDate(0, 0, -2)
		case report.TimeIntervalWeek:
			endDate = now.AddDate(0, 0, -1)
			for i := 0; i < 7; i++ {
				if endDate.Weekday() == time.Sunday {
					break
				}

				endDate = endDate.AddDate(0, 0, -1)
			}

			startDate = endDate.AddDate(0, 0, -6)
		case report.TimeIntervalMonth:
			endDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
			startDate = time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC)
		case report.TimeIntervalQuarter:
			endDate = time.Date(now.Year(), now.Month()-(now.Month()-1)%3, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
			startDate = endDate.AddDate(0, -3, 1)
		case report.TimeIntervalYear:
			endDate = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
			startDate = time.Date(now.Year()-1, 1, 1, 0, 0, 0, 0, time.UTC)
		default:
		}
	} else {
		diff := b.Config.EndPeriod.Truncate(budget.DayDuration).Sub(b.Config.StartPeriod.Truncate(budget.DayDuration))
		startDate = endDate.Add(-diff)
	}

	return s.getQueryTotalByDate(result, b, &startDate, &endDate)
}

// GetBudget returns pointer to budget
func (s *BudgetsService) GetBudget(ctx context.Context, budgetID string) (*budget.Budget, error) {
	fs := s.conn.Firestore(ctx)

	budgetRef := fs.Collection("cloudAnalytics").Doc("budgets").Collection("cloudAnalyticsBudgets").Doc(budgetID)

	docSnap, err := budgetRef.Get(ctx)
	if err != nil {
		return nil, err
	}

	var b budget.Budget
	if err := docSnap.DataTo(&b); err != nil {
		return nil, err
	}

	return &b, nil
}

func (s *BudgetsService) getBudgetQueryRequestAttributions(ctx context.Context, b *budget.Budget, qr *cloudanalytics.QueryRequest) ([]*queryDomain.QueryRequestX, error) {
	attributions := make([]*queryDomain.QueryRequestX, 0)
	attributionIDs := b.GetAttributionIDs()

	attributionsRaw, err := cloudanalytics.GetAttributionsRawDataByIDs(ctx, s.conn.Firestore(ctx), attributionIDs)
	if err != nil {
		return nil, err
	}

	for _, attributionRaw := range attributionsRaw {
		attributions = append(attributions, attributionRaw.ToQueryRequestX(true))
	}

	return attributions, nil
}

func (s *BudgetsService) getBudgetQueryRequest(ctx context.Context, customerID string, b *budget.Budget) (*cloudanalytics.QueryRequest, error) {
	qr := cloudanalytics.QueryRequest{
		Origin:       originDomain.QueryOriginFromContext(ctx),
		Filters:      getBudgetQueryRequestFilters(b),
		Currency:     b.Config.Currency,
		DataSource:   b.Config.DataSource,
		Forecast:     false,
		Metric:       b.Config.Metric,
		Type:         "report",
		TimeSettings: getBudgetQueryTimeSettings(b),
		Cols:         s.getBudgetQueryRequestCols(),
	}

	var err error

	qr.Accounts, err = s.cloudAnalytics.GetAccounts(ctx, customerID, nil, []*report.ConfigFilter{})
	if err != nil {
		return nil, err
	}

	qr.Attributions, err = s.getBudgetQueryRequestAttributions(ctx, b, &qr)
	if err != nil {
		return nil, err
	}

	qr.Timezone, _, err = cloudanalytics.GetTimezoneCurrency(ctx, s.conn.Firestore(ctx), "", "", customerID)
	if err != nil {
		return nil, err
	}

	qr.IsCSP = b.Customer.ID == queryDomain.CSPCustomerID

	return &qr, nil
}

func (s *BudgetsService) RefreshAllBudgets(ctx context.Context) error {
	l := s.loggerProvider(ctx)

	budgetIDs, err := s.getAllBudgetsRefs(ctx)
	if err != nil {
		return err
	}

	for _, budget := range budgetIDs {
		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_GET,
			Path:   "/tasks/analytics/customers/" + budget.CustomerID + "/budgets/" + budget.BudgetID,
			Queue:  common.TaskQueueCloudAnalyticsBudgets,
		}
		if _, err = common.CreateCloudTask(ctx, &config); err != nil {
			l.Errorf(err.Error())
			continue
		}
	}

	return err
}

func (s *BudgetsService) getAllBudgetDocSnaps(ctx context.Context) ([]*firestore.DocumentSnapshot, error) {
	fs := s.conn.Firestore(ctx)

	recurringBudgetsRef := fs.Collection("cloudAnalytics").Doc("budgets").Collection("cloudAnalyticsBudgets").Where("config.type", "==", budget.Recurring).Where("isValid", "==", true)

	docSnaps1, err := recurringBudgetsRef.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	fixedBudgetsRef := fs.Collection("cloudAnalytics").Doc("budgets").Collection("cloudAnalyticsBudgets").Where("config.type", "==", budget.Fixed).Where("isValid", "==", true).Where("config.endPeriod", ">=", now)

	docSnaps2, err := fixedBudgetsRef.Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	return append(docSnaps1, docSnaps2...), nil
}

func (s *BudgetsService) getAllBudgetsRefs(ctx context.Context) ([]struct {
	CustomerID string `json:"customerId"`
	BudgetID   string `json:"budgetId"`
}, error) {
	docSnaps, err := s.getAllBudgetDocSnaps(ctx)
	if err != nil {
		return nil, err
	}

	budgetIDs := make([]struct {
		CustomerID string `json:"customerId"`
		BudgetID   string `json:"budgetId"`
	}, 0)

	for _, docSnap := range docSnaps {
		customerRef, err := docSnap.DataAt("customer")
		if err != nil {
			return nil, err
		}

		castedCustomerRef, ok := customerRef.(*firestore.DocumentRef)
		if !ok {
			continue
		}

		node := struct {
			CustomerID string `json:"customerId"`
			BudgetID   string `json:"budgetId"`
		}{
			CustomerID: castedCustomerRef.ID,
			BudgetID:   docSnap.Ref.ID,
		}

		budgetIDs = append(budgetIDs, node)
	}

	return budgetIDs, nil
}

func (s *BudgetsService) getBudgetQueryRequestCols() []*queryDomain.QueryRequestX {
	field := "T.usage_date_time"
	cols := make([]*queryDomain.QueryRequestX, 3)
	col0 := queryDomain.QueryRequestX{
		Field:     field,
		ID:        "datetime:year",
		Key:       "year",
		AllowNull: false,
		Position:  queryDomain.QueryFieldPositionCol,
		Type:      "datetime",
	}
	cols[0] = &col0
	col1 := queryDomain.QueryRequestX{
		Field:     field,
		ID:        "datetime:month",
		Key:       "month",
		AllowNull: false,
		Position:  queryDomain.QueryFieldPositionCol,
		Type:      "datetime",
	}
	cols[1] = &col1
	col2 := queryDomain.QueryRequestX{
		Field:     field,
		ID:        "datetime:day",
		Key:       "day",
		AllowNull: false,
		Position:  queryDomain.QueryFieldPositionCol,
		Type:      "datetime",
	}
	cols[2] = &col2

	return cols
}

// GetBudgetSlackUnfurl - generates a payload for a given budget to be unfurled on a budget link shared on Slack
// TODO: add s.cloudAnalytics.GetBudgetImages() after highcharts directory is refactored
func (s *BudgetsService) GetBudgetSlackUnfurl(ctx context.Context, budgetID, customerID, URL, imageURLCurrent, imageURLForecasted string) (*budget.Budget, map[string]slackgo.Attachment, error) {
	b, err := s.GetBudget(ctx, budgetID)
	if err != nil {
		return nil, nil, err
	}

	budgetType, err := GetBudgetTypeString(b.Config.Type)
	if err != nil {
		return nil, nil, err
	}

	msgPrinter := message.NewPrinter(language.English)

	var utilization float64
	if b.Config.Amount != 0 {
		utilization = (b.Utilization.Current / b.Config.Amount) * 100
	}

	currency := b.Config.Currency.Symbol()

	var forecastedTotalAmountDate string
	if b.Utilization.ForecastedTotalAmountDate != nil {
		forecastedTotalAmountDate = b.Utilization.ForecastedTotalAmountDate.Format(DateFormat)
	}

	budgetPeriod := string(b.Config.TimeInterval)
	if budgetPeriod == "day" {
		budgetPeriod = "dai"
	}

	fields := []*slackgo.TextBlockObject{
		{
			Type: slackgo.MarkdownType,
			Text: fmt.Sprintf("%s: %s", TextBoldBudget, b.Name),
		},
		{
			Type: slackgo.MarkdownType,
			Text: fmt.Sprintf("%s: %s", TextBoldType, strings.Title(budgetType)),
		},
		{
			Type: slackgo.MarkdownType,
			Text: fmt.Sprintf("%s: %.f%%", TextBoldUtilization, math.Round(utilization)),
		},
		{
			Type: slackgo.MarkdownType,
			Text: fmt.Sprintf("%s: %sly", TextBoldPeriod, strings.Title(budgetPeriod)),
		},
		{
			Type: slackgo.MarkdownType,
			Text: msgPrinter.Sprintf("%s: %s%.f", TextBoldBudgetAmound, currency, b.Config.Amount),
		},
		{
			Type: slackgo.MarkdownType,
			Text: fmt.Sprintf("%s: %s", TextBoldMaxUtilization, forecastedTotalAmountDate),
		},
	}

	if b.Config.Type == budget.Fixed {
		fields = append(fields[:3], fields[4:]...)
	}

	textDescription := &slackgo.TextBlockObject{
		Type: slackgo.MarkdownType,
		Text: fmt.Sprintf("%s: %s", TextBoldDescription, b.Description),
	}
	textCurrent := &slackgo.TextBlockObject{
		Type:  slackgo.PlainTextType,
		Text:  TextCurrentSpend,
		Emoji: true,
	}
	textForecastad := &slackgo.TextBlockObject{
		Type:  slackgo.PlainTextType,
		Text:  TextForecastedSpend,
		Emoji: true,
	}
	textButtonOpen := &slackgo.TextBlockObject{
		Type:  slackgo.PlainTextType,
		Text:  TextButtonOpen,
		Emoji: true,
	}
	buttonOpen := &slackgo.ButtonBlockElement{
		Type:     slackgo.METButton,
		ActionID: EventBudgetView,
		Text:     textButtonOpen,
		URL:      URL,
	}

	textButtonInvestigate := &slackgo.TextBlockObject{
		Type:  slackgo.PlainTextType,
		Text:  TextButtonInvestigate,
		Emoji: true,
	}
	buttonInvestigate := &slackgo.ButtonBlockElement{
		Type:     slackgo.METButton,
		ActionID: EventBudgetInvestigate,
		Text:     textButtonInvestigate,
		URL:      fmt.Sprintf("%s?action=investigate", URL),
	}

	sectionBlock := slackgo.NewSectionBlock(nil, fields, nil)
	currentImageBlock := slackgo.NewImageBlock(
		imageURLCurrent,
		TextCurrent,
		"",
		textCurrent,
	)
	forecastedImageBlock := slackgo.NewImageBlock(
		imageURLForecasted,
		TextForecasted,
		"",
		textForecastad,
	)
	actionBlock := slackgo.NewActionBlock("", buttonOpen, buttonInvestigate)

	var blockSet []slackgo.Block
	if b.Description == "" {
		blockSet = []slackgo.Block{
			sectionBlock,
			currentImageBlock,
			forecastedImageBlock,
			actionBlock,
		}
	} else {
		blockSet = []slackgo.Block{
			sectionBlock,
			slackgo.NewSectionBlock(textDescription, nil, nil),
			currentImageBlock,
			forecastedImageBlock,
			actionBlock,
		}
	}

	unfurl := map[string]slackgo.Attachment{
		URL: {
			Blocks: slackgo.Blocks{
				BlockSet: blockSet,
			},
		},
	}

	return b, unfurl, nil
}

func (s *BudgetsService) UpdateEnforcedByMeteringField(ctx context.Context, budgetID string, collaborators []collab.Collaborator, recipients []string, public *collab.PublicAccess) error {
	var enforcedByMetering bool

	if public != nil {
		enforcedByMetering = true
	}

	for _, collaborator := range collaborators {
		if !common.IsDoitDomain(collaborator.Email) {
			enforcedByMetering = true
			break
		}
	}

	if !enforcedByMetering {
		for _, recipient := range recipients {
			if !common.IsDoitDomain(recipient) {
				enforcedByMetering = true
				break
			}
		}
	}

	return s.dal.UpdateBudgetEnforcedByMetering(ctx, budgetID, enforcedByMetering)
}

func getBudgetQueryRequestFilters(b *budget.Budget) []*queryDomain.QueryRequestX {
	filters := make([]*queryDomain.QueryRequestX, 1)
	attributionIDs := b.GetAttributionIDs()
	filter := &queryDomain.QueryRequestX{
		AllowNull:       false,
		ID:              "attribution:attribution",
		IncludeInFilter: true,
		Key:             "attribution",
		Position:        queryDomain.QueryFieldPositionUnused,
		Type:            "attribution",
		Values:          &attributionIDs,
	}
	filters[0] = filter

	return filters
}

func getBudgetQueryTimeSettings(b *budget.Budget) *cloudanalytics.QueryRequestTimeSettings {
	qrts := cloudanalytics.QueryRequestTimeSettings{
		Interval: report.TimeIntervalDay,
	}

	b.Config.StartPeriod = b.Config.StartPeriod.Truncate(budget.DayDuration)
	b.Config.EndPeriod = b.Config.EndPeriod.Truncate(budget.DayDuration)

	// fetch at least two months of past data for forecasting
	qrTo := time.Now().UTC().Truncate(budget.DayDuration)
	qrFrom := qrTo

	if b.Config.StartPeriod.After(qrTo) {
		qrFrom = b.Config.StartPeriod
	}

	qrts.From = &qrFrom
	qrts.To = &qrTo

	if b.Config.Type == budget.Fixed {
		diff := b.Config.EndPeriod.Truncate(budget.DayDuration).Sub(b.Config.StartPeriod.Truncate(budget.DayDuration))
		qrFrom = b.Config.StartPeriod.Truncate(budget.DayDuration).Add(-diff).AddDate(0, -2, 0)
		qrTo = b.Config.EndPeriod
	} else {
		switch b.Config.TimeInterval {
		case report.TimeIntervalDay:
			b.Config.EndPeriod = qrFrom
			b.Config.StartPeriod = qrFrom.AddDate(0, 0, -1)
			qrFrom = qrFrom.AddDate(0, -2, 0)
			qrTo = qrTo.AddDate(0, 0, -1)
		case report.TimeIntervalWeek:
			for i := 0; i < 7; i++ {
				if qrFrom.Weekday() == time.Sunday {
					break
				}

				qrFrom = qrFrom.AddDate(0, 0, 1)
			}

			b.Config.EndPeriod = qrFrom
			b.Config.StartPeriod = qrFrom.AddDate(0, 0, -6)
			qrFrom = qrFrom.AddDate(0, -2, 0)
		case report.TimeIntervalMonth:
			b.Config.EndPeriod = time.Date(qrFrom.Year(), qrFrom.Month()+1, 1, 0, 0, 0, 0, time.UTC)
			qrFrom = time.Date(qrFrom.Year(), qrFrom.Month(), 1, 0, 0, 0, 0, time.UTC)
			b.Config.StartPeriod = qrFrom
			qrFrom = qrFrom.AddDate(0, -2, 0)
		case report.TimeIntervalQuarter:
			qrFrom = time.Date(qrFrom.Year(), qrFrom.Month()-(qrFrom.Month()-1)%3, 1, 0, 0, 0, 0, time.UTC)
			b.Config.EndPeriod = qrFrom.AddDate(0, +3, 0)
			b.Config.StartPeriod = qrFrom
			qrFrom = qrFrom.AddDate(0, -3, 0)
		case report.TimeIntervalYear:
			qrFrom = time.Date(qrFrom.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
			b.Config.StartPeriod = qrFrom
			b.Config.EndPeriod = qrFrom.AddDate(+1, 0, 0)
			qrFrom = qrFrom.AddDate(-1, 0, 0)
		default:
		}
	}

	if qrts.From.Before(cloudanalytics.MinDate) {
		qrts.From = &cloudanalytics.MinDate
	}

	return &qrts
}
