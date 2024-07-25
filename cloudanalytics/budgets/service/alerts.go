package service

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	slackgo "github.com/slack-go/slack"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/domain/budget"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/mailer"
	events "github.com/doitintl/hello/scheduled-tasks/zapier/domain"
)

// to be removed (CMP-2399)
type BudgetSlackAlert struct {
	BudgetID    string
	SlackBlocks []slackgo.Block
}

const (
	datePattern string = "Mon, 02 Jan 2006"
)

// Slack payload texts
const (
	TextBoldBudget            string = "*Budget*"
	TextBoldType              string = "*Type*"
	TextBoldUtilization       string = "*Utilization*"
	TextBoldPeriod            string = "*Period*"
	TextBoldBudgetAmound      string = "*Budget Amount*"
	TextBoldMaxUtilization    string = "*Max Utilization*"
	TextBoldDescription       string = "*Description*"
	TextBoldAlertSpend        string = "*Alert spend*"
	TextBoldAlertPercentage   string = "*Alert %*"
	TextBoldBudgetPercentage  string = "*Budget %*"
	TextBoldCurrentSpend      string = "*Current spend*"
	TextCurrentSpend          string = "Current Spend"
	TextForecastedSpend       string = "Forecasted Spend"
	TextBudgetButton          string = ":mag: Open Budget"
	TextCurrent               string = "Current"
	TextForecasted            string = "Forecasted"
	TextForecastedTotalAmount string = "Based on your current spending patterns, we estimate that you’ll reach 100%% of this budget on *%s*"
	TextBudgetURL             string = "https://console.doit.com/customers/%s/analytics/budgets/%s"
	EventBudgetView           string = "slack.budget.view"
	DateFormat                string = "Jan 2, 2006"
	TextButtonOpen            string = ":mag: Open Budget"
	TextButtonInvestigate     string = ":bar_chart: Investigate Budget"
	EventBudgetInvestigate    string = "slack.budget.investigate"
)

func getAlertData(b *budget.Budget) *budget.BudgetAlert {
	selectedAlert := &b.Config.Alerts[0]
	for i, alert := range b.Config.Alerts {
		if alert.Triggered && alert.Percentage > selectedAlert.Percentage {
			selectedAlert = &b.Config.Alerts[i]
		}
	}

	return selectedAlert
}

func isPendingAlertBudget(b *budget.Budget) bool {
	for _, alert := range b.Config.Alerts {
		alertAmount := alert.Percentage * b.Config.Amount / 100
		if !alert.Triggered && b.Config.Amount > 0 && alert.Percentage > 0 && b.Utilization.Current >= alertAmount {
			return true
		}
	}

	return false
}

func (s *BudgetsService) TriggerBudgetsAlerts(ctx context.Context) (map[*BudgetSlackAlert][]common.SlackChannel, error) {
	budgets, err := s.getAllBudgets(ctx)
	if err != nil {
		return nil, err
	}

	return s.processThresholdAlerts(ctx, budgets)
}

func (s *BudgetsService) TriggerForecastedDateAlerts(ctx context.Context) error {
	budgets, err := s.getAllBudgets(ctx)
	if err != nil {
		return err
	}

	if err := s.processForecastedDateAlerts(ctx, budgets); err != nil {
		return err
	}

	return nil
}

func (s *BudgetsService) processForecastedDateAlerts(ctx context.Context, budgets []*budget.Budget) error {
	budgetsWithPendingForecastAlerts, err := s.getBudgetsWithPendingForecastAlerts(ctx, budgets)
	if err != nil {
		return err
	}

	if len(budgetsWithPendingForecastAlerts) > 0 {
		if err := s.sendForecastedDateAlerts(ctx, budgetsWithPendingForecastAlerts); err != nil {
			return err
		}
	}

	return nil
}

func (s *BudgetsService) processThresholdAlerts(ctx context.Context, budgets []*budget.Budget) (map[*BudgetSlackAlert][]common.SlackChannel, error) {
	l := s.loggerProvider(ctx)

	budgetsWithAlerts, err := s.getBudgetsWithPendingAlerts(ctx, budgets)
	if err != nil {
		l.Errorf("Error getting budgets with pending alerts. error [%s]", err)
		return nil, err
	}

	if err := s.resetTriggeredAlerts(ctx, budgets); err != nil {
		l.Errorf("Error resetting triggered alerts. error [%s]", err)
		return nil, err
	}

	if len(budgetsWithAlerts) > 0 {
		slackPersonalizations, err := s.sendThresholdAlerts(ctx, budgetsWithAlerts) //	send email alerts & return slack alerts
		if err != nil {
			l.Errorf("Error sending threshold alerts. error [%s]", err)
			return nil, err
		}

		// return slack alerts to be sent by slack service (analytics handler)
		return slackPersonalizations, nil
	}

	return nil, nil
}

func (s *BudgetsService) getAllBudgets(ctx context.Context) ([]*budget.Budget, error) {
	docSnaps, err := s.getAllBudgetDocSnaps(ctx)
	if err != nil {
		return nil, err
	}

	budgets := make([]*budget.Budget, 0)

	for _, docSnap := range docSnaps {
		var b budget.Budget
		if err := docSnap.DataTo(&b); err != nil {
			return nil, err
		}

		b.ID = docSnap.Ref.ID
		budgets = append(budgets, &b)
	}

	return budgets, nil
}

func (s *BudgetsService) getBudgetsWithPendingForecastAlerts(ctx context.Context, budgets []*budget.Budget) ([]*budget.Budget, error) {
	fs := s.conn.Firestore(ctx)
	wb := fb.NewAutomaticWriteBatch(fs, 250)

	budgetsWithPendingAlerts := make([]*budget.Budget, 0)

	for _, b := range budgets {
		if b.Utilization.ShouldSendForecastAlert {
			budgetsWithPendingAlerts = append(budgetsWithPendingAlerts, b)
			wb.Update(s.getForecastedDateAlertUpdate(ctx, b))
		}
	}

	if len(budgetsWithPendingAlerts) > 0 {
		if errs := wb.Commit(ctx); len(errs) > 0 {
			return nil, errs[0]
		}
	}

	return budgetsWithPendingAlerts, nil
}

func (s *BudgetsService) getBudgetsWithPendingAlerts(ctx context.Context, budgets []*budget.Budget) ([]*budget.Budget, error) {
	fs := s.conn.Firestore(ctx)
	wb := fb.NewAutomaticWriteBatch(fs, 250)

	budgetsWithAlerts := make([]*budget.Budget, 0)

	for _, b := range budgets {
		if isPendingAlertBudget(b) {
			budgetsWithAlerts = append(budgetsWithAlerts, b)
			wb.Update(s.getBudgetUpdateOperation(ctx, b))
		}
	}

	if len(budgetsWithAlerts) > 0 {
		if errs := wb.Commit(ctx); len(errs) > 0 {
			return nil, errs[0]
		}
	}

	return budgetsWithAlerts, nil
}

func (s *BudgetsService) getForecastedDateAlertUpdate(ctx context.Context, budget *budget.Budget) (*firestore.DocumentRef, []firestore.Update) {
	fs := s.conn.Firestore(ctx)

	budgetRef := fs.Collection("cloudAnalytics").Doc("budgets").Collection("cloudAnalyticsBudgets").Doc(budget.ID)
	updateOperation := []firestore.Update{
		{
			FieldPath: []string{"utilization", "shouldSendForecastAlert"},
			Value:     false,
		},
	}

	return budgetRef, updateOperation
}

func (s *BudgetsService) getBudgetUpdateOperation(ctx context.Context, budget *budget.Budget) (*firestore.DocumentRef, []firestore.Update) {
	fs := s.conn.Firestore(ctx)

	for i, alert := range budget.Config.Alerts {
		alertAmount := alert.Percentage * budget.Config.Amount / 100
		if budget.Utilization.Current > alertAmount && alert.Percentage > 0 {
			budget.Config.Alerts[i].Triggered = true
		}
	}

	budgetRef := fs.Collection("cloudAnalytics").Doc("budgets").Collection("cloudAnalyticsBudgets").Doc(budget.ID)
	updateOperation := []firestore.Update{
		{
			FieldPath: []string{"config", "alerts"},
			Value:     budget.Config.Alerts,
		},
	}

	return budgetRef, updateOperation
}

func (s *BudgetsService) resetTriggeredAlerts(ctx context.Context, budgets []*budget.Budget) error {
	fs := s.conn.Firestore(ctx)
	wb := fb.NewAutomaticWriteBatch(fs, 250)

	for _, b := range budgets {
		alertsToBeReset := 0

		for i, alert := range b.Config.Alerts {
			alertAmount := alert.Percentage * b.Config.Amount / 100
			if alert.Triggered && b.Config.Amount > 0 && alert.Percentage > 0 && b.Utilization.Current < alertAmount {
				b.Config.Alerts[i].Triggered = false
				alertsToBeReset++
			}
		}

		if alertsToBeReset > 0 {
			budgetRef := fs.Collection("cloudAnalytics").Doc("budgets").Collection("cloudAnalyticsBudgets").Doc(b.ID)
			updateOperation := []firestore.Update{
				{
					FieldPath: []string{"config", "alerts"},
					Value:     b.Config.Alerts,
				},
			}

			wb.Update(budgetRef, updateOperation)
		}
	}

	if errs := wb.Commit(ctx); len(errs) > 0 {
		return errs[0]
	}

	return nil
}

func (s *BudgetsService) sendForecastedDateAlerts(ctx context.Context, budgets []*budget.Budget) error {
	l := s.loggerProvider(ctx)

	for _, b := range budgets {
		if err := s.dal.SaveNotification(ctx, mapBudgetToNotification(b, budget.BudgetNotificationTypeForecast, time.Now().UTC())); err != nil {
			l.Warningf("unable to save notification for budget %s: %s", b.ID, err)
		}

		personalizations, err := s.getForecastedDateAlertPersonalizations(ctx, b)
		if err != nil {
			l.Errorf("failed to get personalizations for budget %s with error: %s", b.ID, err)
			continue
		}

		if len(personalizations) == 0 {
			l.Info("personalizations is empty")
			continue
		}

		if err := mailer.SendEmailWithPersonalizations(personalizations, mailer.Config.DynamicTemplates.CloudAnalyticsBudgetForecastedDateAlert, []string{}); err != nil {
			l.Errorf("failed to send budget %s alerts with error: %s", b.ID, err)
			continue
		}
	}

	return nil
}

// sendThresholdAlerts sends alert emails, dispatches alert events, saves notifications, and returns slack messages
func (s *BudgetsService) sendThresholdAlerts(ctx context.Context, budgets []*budget.Budget) (map[*BudgetSlackAlert][]common.SlackChannel, error) {
	l := s.loggerProvider(ctx)
	slackPersonalizations := make(map[*BudgetSlackAlert][]common.SlackChannel, 0)

	for _, b := range budgets {
		err := s.handleEmailAlert(ctx, b)
		if err != nil {
			l.Println(err)
		}

		if err := s.dal.SaveNotification(ctx, mapBudgetToNotification(b, budget.BudgetNotificationTypeThreshold, time.Now().UTC())); err != nil {
			l.Warningf("unable to save notification for budget %s: %s", b.ID, err)
		}

		if err := s.dispatchBudget(ctx, b); err != nil {
			l.Warningf("unable to dispatch %s for event %s: %s",
				b.ID,
				events.BudgetThresholdAchieved,
				err,
			)
		}

		blocks, channels, err := s.getSlackPersonalizations(ctx, b)
		if len(channels) > 0 {
			if err != nil {
				l.Errorf("failed to get Slack personalizations for budget %s with error: %s", b.ID, err)
				continue
			}

			slackAlert := &BudgetSlackAlert{BudgetID: b.ID, SlackBlocks: blocks}
			slackPersonalizations[slackAlert] = channels
		}
	}

	return slackPersonalizations, nil
}

func (s *BudgetsService) handleEmailAlert(ctx context.Context, budget *budget.Budget) error {
	emailPersonalizations, err := s.getMailPersonalizations(ctx, budget)
	if err != nil {
		return fmt.Errorf("failed to get email personalizations for budget %s with error: %s", budget.ID, err)
	}

	if len(emailPersonalizations) == 0 {
		return fmt.Errorf("email personalizations is empty")
	}

	if err = mailer.SendEmailWithPersonalizations(emailPersonalizations, mailer.Config.DynamicTemplates.CloudAnalyticsBudgetAlert, []string{}); err != nil {
		return fmt.Errorf("failed to send budget %s alerts with error: %s", budget.ID, err)
	}

	return nil
}

func (s *BudgetsService) getForecastedDateAlertPersonalizations(ctx context.Context, budget *budget.Budget) ([]*mail.Personalization, error) {
	l := s.loggerProvider(ctx)

	personalizations := make([]*mail.Personalization, 0)
	customerID := budget.Customer.ID
	currencySymbol := budget.Config.Currency.Symbol()
	currentSpendAmount := common.FormatNumber(budget.Utilization.Current, 2)

	var currentSpendPercentage float64
	if budget.Config.Amount != 0 {
		currentSpendPercentage = math.Round(budget.Utilization.Current/budget.Config.Amount*10000) / 100
	}

	if budget.Utilization.ForecastedTotalAmountDate == nil ||
		budget.Utilization.PreviousForecastedDate == nil {
		l.Info("mail was not sent - missing forecasted date")
		return personalizations, nil
	}

	tos := make([]*mail.Email, 0)

	for _, recipient := range budget.Recipients {
		if !common.Production && !common.IsDoitDomain(recipient) {
			l.Info("mail to <" + recipient + "> didn't send while in development")
			continue
		}

		tos = append(tos, mail.NewEmail("", recipient))
	}

	if len(tos) == 0 {
		return personalizations, nil
	}

	p := mail.NewPersonalization()
	p.AddTos(tos...)
	p.SetDynamicTemplateData("budget_name", budget.Name)
	p.SetDynamicTemplateData("forecasted_date", budget.Utilization.ForecastedTotalAmountDate.Format(datePattern))
	p.SetDynamicTemplateData("currency_symbol", currencySymbol)
	p.SetDynamicTemplateData("current_amount", currentSpendAmount)
	p.SetDynamicTemplateData("current_percentage", currentSpendPercentage)
	p.SetDynamicTemplateData("customer_id", customerID)
	p.SetDynamicTemplateData("budget_id", budget.ID)
	p.SetDynamicTemplateData("domain", common.Domain)
	p.SetDynamicTemplateData("previous_date", budget.Utilization.PreviousForecastedDate.Format(datePattern))

	l.Infof("sending budget %s forecasted date alert to %s", budget.ID, strings.Join(budget.Recipients, ", "))
	l.Infof("budget forecasted date alert data: %+v ", budget.Utilization)

	personalizations = append(personalizations, p)

	return personalizations, nil
}

func (s *BudgetsService) getMailPersonalizations(ctx context.Context, budget *budget.Budget) ([]*mail.Personalization, error) {
	l := s.loggerProvider(ctx)

	personalizations := make([]*mail.Personalization, 0)
	customerID := budget.Customer.ID
	currentSpendAmount := common.FormatNumber(budget.Utilization.Current, 2)

	var currentSpendPercentage float64
	if budget.Config.Amount != 0 {
		currentSpendPercentage = math.Round(budget.Utilization.Current/budget.Config.Amount*10000) / 100
	}

	currencySymbol := budget.Config.Currency.Symbol()
	subject := s.getSubject(currentSpendPercentage, budget.Name, false)
	selectedAlert := getAlertData(budget)
	selectedAlertAmount := selectedAlert.Percentage * budget.Config.Amount / 100
	alertAmount := common.FormatNumber(selectedAlertAmount, 2)

	tos := make([]*mail.Email, 0)

	for _, recipient := range budget.Recipients {
		if !common.Production && !common.IsDoitDomain(recipient) {
			l.Info("mail to <" + recipient + "> didn't send while in development")
			continue
		}

		tos = append(tos, mail.NewEmail("", recipient))
	}

	if len(tos) == 0 {
		return personalizations, nil
	}

	p := mail.NewPersonalization()
	p.AddTos(tos...)
	p.SetDynamicTemplateData("subject", subject)
	p.SetDynamicTemplateData("budget_name", budget.Name)
	p.SetDynamicTemplateData("customer_id", customerID)
	p.SetDynamicTemplateData("budget_id", budget.ID)
	p.SetDynamicTemplateData("domain", common.Domain)
	p.SetDynamicTemplateData("currency_symbol", currencySymbol)

	if budget.Config.TimeInterval != report.TimeIntervalDay &&
		budget.Utilization.ForecastedTotalAmountDate != nil &&
		currentSpendPercentage < 100 {
		p.SetDynamicTemplateData("forecasted_date", budget.Utilization.ForecastedTotalAmountDate.Format(datePattern))
	}

	p.SetDynamicTemplateData("current_amount", currentSpendAmount)
	p.SetDynamicTemplateData("current_percentage", currentSpendPercentage)
	p.SetDynamicTemplateData("alert_percentage", selectedAlert.Percentage)
	p.SetDynamicTemplateData("alert_amount", alertAmount)
	l.Infof("sending budget %s threshold alert to %s", budget.ID, strings.Join(budget.Recipients, ", "))
	l.Infof("budget threshold alert data: %+v ", budget.Utilization)

	personalizations = append(personalizations, p)

	return personalizations, nil
}

func (s *BudgetsService) getSubject(budgetUtilizationPercent float64, budgetName string, slack bool) string {
	p := strconv.FormatFloat(budgetUtilizationPercent, 'f', 2, 32)
	if slack {
		return "*Budget Alert*: You’ve exceeded *" + p + "%* of your *" + budgetName + "* budget"
	}

	return "Budget Alert: You’ve exceeded " + p + "% of your " + budgetName + " budget"
}

// generates slack alert personalizations along with channel list to post on
func (s *BudgetsService) getSlackPersonalizations(ctx context.Context, b *budget.Budget) ([]slackgo.Block, []common.SlackChannel, error) { //	todo make for single budget
	l := s.loggerProvider(ctx)

	personalizations := make([]common.SlackChannel, 0)
	customerID := b.Customer.ID
	currentSpendAmount := common.FormatNumber(b.Utilization.Current, 2)

	var currentSpendPercentage float64
	if b.Config.Amount != 0 {
		currentSpendPercentage = math.Round(b.Utilization.Current/b.Config.Amount*10000) / 100
	}

	subject := s.getSubject(currentSpendPercentage, b.Name, true)
	selectedAlert := getAlertData(b)
	selectedAlertAmount := selectedAlert.Percentage * b.Config.Amount / 100
	alertAmount := common.FormatNumber(selectedAlertAmount, 2)

	budgetType, err := GetBudgetTypeString(b.Config.Type)
	if err != nil {
		return nil, nil, err
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
			Text: fmt.Sprintf("%s: %s", TextBoldType, strings.Title(budgetType)),
		},
		{
			Type: slackgo.MarkdownType,
			Text: fmt.Sprintf("%s: %sly", TextBoldPeriod, strings.Title(budgetPeriod)),
		},
		{
			Type: slackgo.MarkdownType,
			Text: fmt.Sprintf("%s: %s%s", TextBoldAlertSpend, currency, alertAmount),
		},
		{
			Type: slackgo.MarkdownType,
			Text: fmt.Sprintf("%s: %.f", TextBoldAlertPercentage, selectedAlert.Percentage),
		},
		{
			Type: slackgo.MarkdownType,
			Text: fmt.Sprintf("%s: %s%s", TextBoldCurrentSpend, currency, currentSpendAmount),
		},
		{
			Type: slackgo.MarkdownType,
			Text: fmt.Sprintf("%s: %.f", TextBoldBudgetPercentage, currentSpendPercentage),
		},
	}

	if b.Config.Type == budget.Fixed {
		fields = append(fields[:1], fields[2:]...)
	}

	textSubject := &slackgo.TextBlockObject{
		Type: slackgo.MarkdownType,
		Text: fmt.Sprint(subject),
	}
	textForecast := &slackgo.TextBlockObject{
		Type: slackgo.MarkdownType,
		Text: fmt.Sprintf(TextForecastedTotalAmount, forecastedTotalAmountDate),
	}

	textDescription := &slackgo.TextBlockObject{
		Type: slackgo.MarkdownType,
		Text: fmt.Sprintf("%s: %s", TextBoldDescription, b.Description),
	}
	textButton := &slackgo.TextBlockObject{
		Type:  slackgo.PlainTextType,
		Text:  TextBudgetButton,
		Emoji: true,
	}
	button := &slackgo.ButtonBlockElement{
		Type:     slackgo.METButton,
		ActionID: EventBudgetView,
		Text:     textButton,
		URL:      fmt.Sprintf(TextBudgetURL, customerID, b.ID),
	}

	subgectBlock := slackgo.NewSectionBlock(textSubject, nil, nil)
	forecastBlock := slackgo.NewSectionBlock(textForecast, nil, nil)
	sectionBlock := slackgo.NewSectionBlock(nil, fields, nil)
	actionBlock := slackgo.NewActionBlock("", button)

	var blockSet []slackgo.Block
	if b.Description == "" {
		blockSet = []slackgo.Block{
			subgectBlock,
			sectionBlock,
			forecastBlock,
			actionBlock,
		}
	} else {
		blockSet = []slackgo.Block{
			subgectBlock,
			sectionBlock,
			slackgo.NewSectionBlock(textDescription, nil, nil),
			forecastBlock,
			actionBlock,
		}
	}

	for _, channel := range b.RecipientsSlackChannels {
		if !common.Production && channel.CustomerID != common.DoitCustomerID && channel.CustomerID != "JhV7WydpOlW8DeVRVVNf" { // on development - only send alerts for doit & budgetao workspaces
			l.Info(fmt.Sprintf("slack alert to %s.%s didn't send while in development", channel.Workspace, channel.Name))
			continue
		}

		personalizations = append(personalizations, channel)
	}
	// TODO refactor --> map[*slackgo.MsgOption][]common.SlackChannel same as slack/channel.go (CMP-2399)

	return blockSet, personalizations, nil
}

// to be removed/moved inside getSlackPersonalizations() (CMP-2399)
func (s *BudgetsService) GetSlackFinalBlocks(ctx context.Context, imageURLForecasted string, blocks []slackgo.Block) slackgo.MsgOption {
	size := len(blocks)
	textForecastad := &slackgo.TextBlockObject{
		Type:  slackgo.PlainTextType,
		Text:  TextForecastedSpend,
		Emoji: true,
	}
	forecastedImageBlock := slackgo.NewImageBlock(
		imageURLForecasted,
		TextForecasted,
		"",
		textForecastad,
	)
	blocks = append(blocks[:size-1], forecastedImageBlock, blocks[size-1])

	return slackgo.MsgOptionBlocks(blocks...)
}

func (b *BudgetsService) dispatchBudget(ctx context.Context, budget *budget.Budget) error {
	dispatchBudget, err := mapInternalBudgetToResponseBudget(budget)
	if err != nil {
		return fmt.Errorf("unable to dispatch: error map to external budget: %w", err)
	}

	return b.eventDispatcher.Dispatch(ctx,
		dispatchBudget,
		budget.Customer,
		budget.ID,
		events.BudgetThresholdAchieved,
	)
}

func mapBudgetToNotification(b *budget.Budget, nt budget.BudgetNotificationType, alertDate time.Time) *budget.BudgetNotification {
	var currentSpendPercentage float64
	if b.Config.Amount != 0 {
		currentSpendPercentage = math.Round(b.Utilization.Current/b.Config.Amount*10000) / 100
	}

	selectedAlert := getAlertData(b)
	selectedAlertAmount := selectedAlert.Percentage * b.Config.Amount / 100

	return &budget.BudgetNotification{
		Name:              b.Name,
		Type:              nt,
		BudgetID:          b.ID,
		Customer:          b.Customer,
		AlertDate:         alertDate,
		AlertAmount:       common.FormatNumber(selectedAlertAmount, 2),
		AlertPercentage:   selectedAlert.Percentage,
		CurrencySymbol:    b.Config.Currency.Symbol(),
		CurrentAmount:     common.FormatNumber(b.Utilization.Current, 2),
		CurrentPercentage: currentSpendPercentage,
		ForcastedDate:     b.Utilization.ForecastedTotalAmountDate,
		ExpireBy:          alertDate.AddDate(0, 0, 31),
		Recipients:        b.Recipients,
	}
}
