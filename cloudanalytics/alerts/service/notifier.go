package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/times"
	events "github.com/doitintl/hello/scheduled-tasks/zapier/domain"
)

var (
	ErrInvalidTableCell = errors.New("invalid table cell")
)

const (
	ValueDailyRange      = 4
	PercentageDailyRange = 5
)

var sustainedUsageCreditsFilter = report.ConfigFilter{
	BaseConfigFilter: report.BaseConfigFilter{
		ID:      fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeFixed, metadata.MetadataFieldKeyCredit),
		Key:     metadata.MetadataFieldKeyCredit,
		Type:    metadata.MetadataFieldTypeFixed,
		Values:  &[]string{"Sustained Usage Discount"},
		Inverse: true,
	},
}

// RefreshAlerts refreshs all alerts in the database
func (s *AnalyticsAlertsService) RefreshAlerts(ctx context.Context) error {
	l := s.loggerProvider(ctx)

	alerts, err := s.alertsDal.GetAlerts(ctx)
	if err != nil {
		return err
	}

	// filter out customers who don't have access to alerts entitlement
	customerEntlAccessMap := make(map[string]bool)

	for _, alert := range alerts {
		customerID := alert.Customer.ID

		hasAccess, ok := customerEntlAccessMap[customerID]
		if !ok {
			accessDenied, err := s.alertTierService.CheckAccessToAlerts(ctx, customerID)
			if err != nil {
				l.Errorf(err.Error())
				continue
			}

			hasAccess = accessDenied == nil
			customerEntlAccessMap[customerID] = hasAccess
		}

		if !hasAccess {
			l.Infof("customer %s does not have access to alerts", customerID)
			continue
		}

		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_GET,
			Path:   "/tasks/analytics/alerts/" + alert.ID + "/refresh",
			Queue:  common.TaskQueueCloudAnalyticsAlerts,
		}

		if _, err = s.cloudTaskClient.CreateTask(ctx, config.Config(nil)); err != nil {
			l.Errorf("failed creating task for alert %s with error: %s", alert.ID, err)
			continue
		}
	}

	return nil
}

// RefreshAlert refreshs an alert in the database. Refreshing an alert means:
// 1. getting the report request (from the alert data)
// 2. getting the report result
// 3. checking if the alert met the requirements, if the alert met the requirements create a detected alert
func (s *AnalyticsAlertsService) RefreshAlert(ctx context.Context, alertID string) error {
	l := s.loggerProvider(ctx)

	alert, err := s.alertsDal.GetAlert(ctx, alertID)
	if err != nil {
		return err
	}

	customerID := alert.Customer.ID

	accessDenied, err := s.alertTierService.CheckAccessToAlerts(ctx, customerID)
	if err != nil {
		return err
	}

	if accessDenied != nil {
		l.Infof("customer %s does not have access to alerts tier", customerID)
		return nil
	}

	if err = s.validateAlert(alert); err != nil {
		return err
	}

	//if we have only one recipient and that is no-reply, we don't run the report
	if len(alert.Recipients) == 1 {
		if alert.Recipients[0] == "no-reply@doit.com" {
			return nil
		}
	}

	qr, err := s.getReportRequest(ctx, alert, alertID)
	if err != nil {
		return err
	}

	// qr will be nil if the alert is by breakdown and already refreshed which mean that
	// there are already 10 docs in the database that are ready to be sent by email
	if qr == nil {
		return nil
	}

	result, err := s.cloudAnalytics.GetQueryResult(ctx, qr, alert.Customer.ID, "")
	if err != nil {
		return err
	}

	if err = s.refreshAlert(ctx, alert, result, qr, alertID); err != nil {
		return err
	}

	return nil
}

// getReportRequest generate the report request that holds the alert data
func (s *AnalyticsAlertsService) getReportRequest(ctx context.Context, alert *domain.Alert, alertID string) (*cloudanalytics.QueryRequest, error) {
	filters := s.getAlertConfigFilters(alert.Config)
	rows := alert.Config.Rows

	// if there is a breakdown then limit results of the breakdown
	if len(rows) > 0 {
		breakDownFilter, err := s.buildBreakdownFilter(ctx, alert, alertID, rows[0])
		if err != nil {
			return nil, err
		}

		if breakDownFilter == nil {
			return nil, nil
		}

		filters = append(filters, breakDownFilter)
	}

	if alert.Config.Condition == domain.ConditionPercentage {
		filters = append(filters, &sustainedUsageCreditsFilter)
	}

	cols, timeInterval := s.getColsAndTimeInterval(alert.Config)

	requestFilters, err := cloudanalytics.GetFilters(filters, rows, cols)
	if err != nil {
		return nil, err
	}

	accounts, err := s.cloudAnalytics.GetAccounts(ctx, alert.Customer.ID, nil, filters)
	if err != nil {
		return nil, err
	}

	attributions, err := s.cloudAnalytics.GetAttributions(ctx, requestFilters, rows, cols, "")
	if err != nil {
		return nil, err
	}

	requestCols, err := cloudanalytics.GetRowsOrCols(cols, domainQuery.QueryFieldPositionCol)
	if err != nil {
		return nil, err
	}

	requestRows, err := cloudanalytics.GetRowsOrCols(rows, domainQuery.QueryFieldPositionRow)
	if err != nil {
		return nil, err
	}

	today := times.CurrentDayUTC()
	ts, customTimeRange, comparative := s.getTimeSettings(today, alert.Config)

	timeSettings, err := cloudanalytics.GetTimeSettings(ts, timeInterval, customTimeRange, today)
	if err != nil {
		return nil, err
	}

	isCSP := alert.Customer.ID == domainQuery.CSPCustomerID

	cloudProviders := cloudanalytics.GetCloudProviders(filters)

	extendedMetric, calculatedMetric, err := s.getCustomMetrics(ctx, alert.Config)
	if err != nil {
		return nil, err
	}

	metricFilters := []*domainQuery.QueryRequestMetricFilter{}

	if alert.Config.Condition == domain.ConditionPercentage {
		if alert.Config.IgnoreValuesRange != nil {
			lowerBound := alert.Config.IgnoreValuesRange.LowerBound
			upperBound := alert.Config.IgnoreValuesRange.UpperBound

			metricFilters = append(metricFilters, &domainQuery.QueryRequestMetricFilter{
				Operator: report.MetricFilterNotBetween,
				Values:   []float64{lowerBound, upperBound},
				Metric:   alert.Config.Metric,
			})
		}
	}

	qr := cloudanalytics.QueryRequest{
		Accounts:         accounts,
		Attributions:     attributions,
		CalculatedMetric: calculatedMetric,
		CloudProviders:   cloudProviders,
		Cols:             requestCols,
		Comparative:      comparative,
		Currency:         alert.Config.Currency,
		DataSource:       &alert.Config.DataSource,
		ExtendedMetric:   extendedMetric,
		Filters:          requestFilters,
		Forecast:         alert.Config.Condition == domain.ConditionForecast,
		IsCSP:            isCSP,
		LimitAggregation: report.LimitAggregationNone,
		Metric:           alert.Config.Metric,
		MetricFiltres:    metricFilters,
		Organization:     alert.Organization,
		Origin:           domainOrigin.QueryOriginFromContext(ctx),
		Rows:             requestRows,
		TimeSettings:     timeSettings,
	}

	return &qr, nil
}

func (s *AnalyticsAlertsService) getAlertConfigFilters(config *domain.Config) []*report.ConfigFilter {
	if len(config.Scope) > 0 {
		var values []string
		for _, v := range config.Scope {
			values = append(values, v.ID)
		}

		filter := report.ConfigFilter{
			BaseConfigFilter: report.BaseConfigFilter{
				ID:     fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeAttribution, metadata.MetadataFieldTypeAttribution),
				Key:    string(metadata.MetadataFieldTypeAttribution),
				Type:   metadata.MetadataFieldTypeAttribution,
				Values: &values,
			},
		}

		return []*report.ConfigFilter{&filter}
	}

	return config.Filters
}

func (s *AnalyticsAlertsService) getColsAndTimeInterval(config *domain.Config) ([]string, report.TimeInterval) {
	var cols []string

	var timeInterval report.TimeInterval

	if config.Condition == domain.ConditionForecast {
		cols = report.GetColsFromInterval(report.TimeIntervalDay)
		timeInterval = report.TimeIntervalDay
	} else {
		cols = report.GetColsFromInterval(config.TimeInterval)
		timeInterval = config.TimeInterval
	}

	return cols, timeInterval
}

func (s *AnalyticsAlertsService) getCustomMetrics(ctx context.Context, config *domain.Config) (string, *cloudanalytics.QueryRequestCalculatedMetric, error) {
	if config.Metric == report.MetricCustom {
		calculatedMetric, err := cloudanalytics.GetQueryRequestCalculatedMetric(ctx, s.conn.Firestore(ctx), config.CalculatedMetric.ID)
		if err != nil {
			return "", nil, errors.New("error getting calculated metric")
		}

		return "", calculatedMetric, nil
	} else if config.ExtendedMetric != nil {
		return *config.ExtendedMetric, nil, nil
	}

	return "", nil, nil
}

// getTimeSettings returns the time settings and custom time range
func (s *AnalyticsAlertsService) getTimeSettings(today time.Time, config *domain.Config) (*report.TimeSettings, *report.ConfigCustomTimeRange, *string) {
	var comparative *string

	var customTimeRange *report.ConfigCustomTimeRange

	var ts report.TimeSettings

	if config.Condition == domain.ConditionValue {
		switch config.TimeInterval {
		case report.TimeIntervalDay:
			ts.Mode = report.TimeSettingsModeCustom
			customTimeRange = &report.ConfigCustomTimeRange{
				From: today.AddDate(0, 0, -ValueDailyRange),
				To:   today,
			}
		case report.TimeIntervalWeek:
			ts.Mode = report.TimeSettingsModeLast
			ts.Unit = report.TimeSettingsUnitDay
			ts.Amount = 7
			ts.IncludeCurrent = true
		case report.TimeIntervalMonth:
			ts.Mode = report.TimeSettingsModeCurrent
			ts.Unit = report.TimeSettingsUnitMonth
		case report.TimeIntervalQuarter:
			ts.Mode = report.TimeSettingsModeCurrent
			ts.Unit = report.TimeSettingsUnitQuarter
		case report.TimeIntervalYear:
			ts.Mode = report.TimeSettingsModeCurrent
			ts.Unit = report.TimeSettingsUnitYear
		}
	}

	if config.Condition == domain.ConditionPercentage {
		switch config.TimeInterval {
		case report.TimeIntervalDay:
			ts.Mode = report.TimeSettingsModeCustom
			customTimeRange = &report.ConfigCustomTimeRange{
				From: today.AddDate(0, 0, -PercentageDailyRange),
				To:   today,
			}
		case report.TimeIntervalWeek:
			ts.Mode = report.TimeSettingsModeLast
			ts.Unit = report.TimeSettingsUnitWeek
			ts.Amount = 2
			ts.IncludeCurrent = true
		case report.TimeIntervalMonth:
			ts.Mode = report.TimeSettingsModeLast
			ts.Unit = report.TimeSettingsUnitMonth
			ts.Amount = 2
			ts.IncludeCurrent = true
		case report.TimeIntervalQuarter:
			ts.Mode = report.TimeSettingsModeLast
			ts.Unit = report.TimeSettingsUnitQuarter
			ts.Amount = 2
			ts.IncludeCurrent = true
		case report.TimeIntervalYear:
			ts.Mode = report.TimeSettingsModeCustom
			customTimeRange = &report.ConfigCustomTimeRange{
				From: today.AddDate(-1, 0, 0),
				To:   today,
			}
		}

		percent := report.ComparativePercentageChange
		comparative = &percent
	}

	if config.Condition == domain.ConditionForecast {
		switch config.TimeInterval {
		case report.TimeIntervalWeek, report.TimeIntervalMonth, report.TimeIntervalQuarter:
			ts.Mode = report.TimeSettingsModeLast
			ts.Unit = report.TimeSettingsUnitMonth
			ts.Amount = 3
			ts.IncludeCurrent = true
		case report.TimeIntervalYear:
			ts.Mode = report.TimeSettingsModeLast
			ts.Unit = report.TimeSettingsUnitMonth
			ts.Amount = 12
			ts.IncludeCurrent = true
		}
	}

	return &ts, customTimeRange, comparative
}

func (s *AnalyticsAlertsService) refreshAlert(ctx context.Context, alert *domain.Alert, result cloudanalytics.QueryResult, qr *cloudanalytics.QueryRequest, alertID string) error {
	notificationsToAdd := []*domain.Notification{}
	metricIndex := len(qr.Rows) + len(qr.Cols) + qr.GetMetricIndex()

	if err := s.checkRowsForAlertValue(ctx, alert, result.Rows, metricIndex, alertID, &notificationsToAdd, qr); err != nil {
		return err
	}

	if err := s.checkRowsForAlertPercentatge(ctx, alert, result.Rows, metricIndex, alertID, &notificationsToAdd, qr); err != nil {
		return err
	}

	if alert.Config.Condition == domain.ConditionForecast {
		notification, err := s.checkRowsForAlertForecast(ctx, result.ForecastRows, alert, alertID, qr)
		if err != nil {
			return err
		}

		if notification != nil {
			notificationsToAdd = append(notificationsToAdd, notification)
		}
	}

	if len(notificationsToAdd) == 0 {
		return nil
	}

	// save the alertName and conditionString to use in in-app notifications
	for _, notification := range notificationsToAdd {
		if notification == nil {
			continue
		}

		condition, _ := s.buildAlertCondition(ctx, alert)
		notification.AlertName = alert.Name
		notification.ConditionString = condition
	}

	addedNotifications, err := s.notificationsDal.AddDetectedNotifications(ctx, notificationsToAdd, alert.Etag)
	if err != nil {
		return err
	}

	for _, notification := range addedNotifications {
		if err := s.eventDispatcher.Dispatch(ctx,
			toWebhookEvent(notification, alert),
			notification.Customer,
			notification.Alert.ID,
			events.AlertConditionSatisfied,
		); err != nil {
			s.loggerProvider(ctx).Warningf(
				"error dispatching %s alert for entity %s: %w",
				events.AlertConditionSatisfied,
				notification.Alert.ID,
				err,
			)
		}
	}

	return nil
}

func (s *AnalyticsAlertsService) convertRowStringToInt(bqValue bigquery.Value) (int, error) {
	switch rowString := bqValue.(type) {
	case string:
		rowInt, err := strconv.Atoi(rowString)
		if err != nil {
			return 0, err
		}

		return rowInt, nil
	default:
		return 0, errors.New("invalid row type")
	}
}

// getTimestampForRowValue is a helper function to get the timestamp for a row value
func (s *AnalyticsAlertsService) getTimestampForRowValue(key string, rowValue bigquery.Value, year, month, day *int) error {
	var err error

	switch key {
	// support for quarter and week can be added if needed
	case "year":
		*year, err = s.convertRowStringToInt(rowValue)
		if err != nil {
			return err
		} else if *year < 1 {
			return fmt.Errorf("invalid year value")
		}
	case "month":
		*month, err = s.convertRowStringToInt(rowValue)
		if err != nil {
			return err
		} else if *month < 1 || *month > 12 {
			return fmt.Errorf("invalid month value")
		}
	case "day":
		*day, err = s.convertRowStringToInt(rowValue)
		if err != nil {
			return err
		} else if *day < 1 || *day > 31 {
			return fmt.Errorf("invalid day value")
		}
	default:
		return fmt.Errorf("invalid row date key")
	}

	return nil
}

// getTimestampFromRow returns the timestamp from the row based on the query request
func (s *AnalyticsAlertsService) getTimestampFromRow(qr *cloudanalytics.QueryRequest, row []bigquery.Value) (*time.Time, error) {
	year, month, day := 1, 1, 1
	rowsLength := len(qr.Rows)

	if qr.Forecast {
		rowsLength++
	}

	if len(row) < rowsLength+len(qr.Cols) {
		return nil, fmt.Errorf("invalid row length")
	}

	for i := range qr.Cols {
		err := s.getTimestampForRowValue(qr.Cols[i].Key, row[i+rowsLength], &year, &month, &day)
		if err != nil {
			return nil, err
		}
	}

	timestamp := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	return &timestamp, nil
}

func (s *AnalyticsAlertsService) getRowTimeDetected(timeInterval report.TimeInterval, qr *cloudanalytics.QueryRequest, row []bigquery.Value) (*time.Time, error) {
	if timeInterval == report.TimeIntervalDay {
		rowTimedetected, err := s.getTimestampFromRow(qr, row)
		if err != nil {
			return nil, err
		}

		return rowTimedetected, nil
	}

	return nil, nil
}

func (s *AnalyticsAlertsService) checkRowsForAlertValue(ctx context.Context, alert *domain.Alert, rows [][]bigquery.Value, metricIndex int, alertID string, notificationsToAdd *[]*domain.Notification, qr *cloudanalytics.QueryRequest) error {
	if alert.Config.Condition == domain.ConditionValue {
		for _, row := range rows {
			rowTimeDetected, err := s.getRowTimeDetected(alert.Config.TimeInterval, qr, row)
			if err != nil {
				return err
			}

			notification, err := s.checkRowForAlertValue(ctx, row, alert, metricIndex, alertID, rowTimeDetected)
			if err != nil {
				return err
			}

			if notification != nil {
				*notificationsToAdd = append(*notificationsToAdd, notification)
			}
		}
	}

	return nil
}

func (s *AnalyticsAlertsService) isFirstPercentageDay(rowTimeDetected *time.Time) bool {
	if rowTimeDetected != nil {
		time := times.CurrentDayUTC()

		duration := time.Sub(*rowTimeDetected)
		if duration.Hours() > 24*(PercentageDailyRange-1)+5 {
			// skip the first row because we use it only to compare, we add 5 hours to be sure we don't miss other rows
			return true
		}
	}

	return false
}

func (s *AnalyticsAlertsService) checkRowsForAlertPercentatge(ctx context.Context, alert *domain.Alert, rows [][]bigquery.Value, metricIndex int, alertID string, notificationsToAdd *[]*domain.Notification, qr *cloudanalytics.QueryRequest) error {
	if alert.Config.Condition == domain.ConditionPercentage {
		metricIndex += qr.GetMetricCount()

		for _, row := range rows {
			rowTimeDetected, err := s.getRowTimeDetected(alert.Config.TimeInterval, qr, row)
			if err != nil {
				return err
			}

			if s.isFirstPercentageDay(rowTimeDetected) {
				continue
			}

			notification, err := s.checkRowForAlertPercentage(ctx, row, alert, metricIndex, alertID, rowTimeDetected)
			if err != nil {
				return err
			}

			if notification != nil {
				*notificationsToAdd = append(*notificationsToAdd, notification)
			}
		}
	}

	return nil
}

// checkCondition checks if the alert condition is met
func (s *AnalyticsAlertsService) checkCondition(config *domain.Config, value float64) bool {
	switch config.Operator {
	case report.MetricFilterGreaterThan:
		return value > config.Values[0]
	case report.MetricFilterLessThan:
		return value < config.Values[0]
	}

	return false
}

// validateAlert checks if the alert is valid
func (s *AnalyticsAlertsService) validateAlert(alert *domain.Alert) error {
	if alert.Config == nil {
		return errors.New("config is null")
	}

	if len(alert.Config.Values) < 1 {
		return errors.New("alert config values is empty")
	}

	if len(alert.Config.Scope) < 1 && len(alert.Config.Filters) < 1 {
		return errors.New("alert config scope and filters are empty")
	}

	if len(alert.Config.Rows) > 0 {
		labelType := metadata.MetadataFieldType(strings.Split(alert.Config.Rows[0], ":")[0])
		if labelType == metadata.MetadataFieldTypeGKE || labelType == metadata.MetadataFieldTypeGKELabel {
			return errors.New("alert config rows contains gke dimension, which isn't supported")
		}
	}

	if len(alert.Name) < 1 {
		return errors.New("alert name is empty")
	}

	if len(alert.Recipients) < 1 {
		return errors.New("alert recipients is empty")
	}

	if alert.Etag == "" {
		return errors.New("etag is empty")
	}

	if alert.Collaborators == nil || len(alert.Collaborators) < 1 {
		return errors.New("collaborators is empty")
	}

	if alert.Customer == nil {
		return errors.New("customer is null")
	}

	var str interface{} = alert.Config.Condition
	switch str.(type) {
	case domain.Condition:
		break
	default:
		return errors.New("condition is not a valid type")
	}

	return nil
}

func (s *AnalyticsAlertsService) checkRowForAlertValue(ctx context.Context, row []bigquery.Value, alert *domain.Alert, metricIndex int, alertID string, rowTimeDetected *time.Time) (*domain.Notification, error) {
	var value float64
	switch row[metricIndex].(type) {
	case float64:
		value = row[metricIndex].(float64)
	default:
		return nil, ErrInvalidTableCell
	}

	return s.newNotification(ctx, alert, &row, value, alertID, rowTimeDetected)
}

func (s *AnalyticsAlertsService) getComparativePctChangeForMetric(row []bigquery.Value, metricIndex int) (float64, error) {
	var (
		rowData cloudanalytics.ComparativeColumnValue
		pct     float64
	)

	switch row[metricIndex].(type) {
	case cloudanalytics.ComparativeColumnValue:
		rowData = row[metricIndex].(cloudanalytics.ComparativeColumnValue)
	default:
		return pct, ErrInvalidTableCell
	}

	switch rowData.Pct.(type) {
	case float64:
		pct = rowData.Pct.(float64)
	case int:
		pct = float64(rowData.Pct.(int))
	default:
		return pct, ErrInvalidTableCell
	}

	return pct, nil
}

func (s *AnalyticsAlertsService) checkRowForAlertPercentage(ctx context.Context, row []bigquery.Value, alert *domain.Alert, metricIndex int, alertID string, rowTimeDetected *time.Time) (*domain.Notification, error) {
	l := s.loggerProvider(ctx)

	pct, err := s.getComparativePctChangeForMetric(row, metricIndex)
	if err != nil {
		return nil, err
	}

	// If cost alert is triggered, check if the usage percentage change is greater than 5%
	// before generating a new notification.
	if alert.Config.Metric == report.MetricCost && s.checkCondition(alert.Config, pct) {
		usagePct, err := s.getComparativePctChangeForMetric(row, metricIndex+1)
		if err != nil {
			return nil, err
		}

		if math.Abs(usagePct) <= 5 {
			l.Debugf("condition met for cost pct change %f, but usage pct change is small |%f| <= 5. Skipping notification.", pct, usagePct)
			return nil, nil
		}
	}

	return s.newNotification(ctx, alert, &row, pct, alertID, rowTimeDetected)
}

func (s *AnalyticsAlertsService) checkRowsForAlertForecast(ctx context.Context, rows [][]bigquery.Value, alert *domain.Alert, alertID string, qr *cloudanalytics.QueryRequest) (*domain.Notification, error) {
	var value float64

	if len(rows) == 0 {
		return nil, errors.New("no forecast rows")
	}

	for _, row := range rows {
		var rowLen = len(row) - 1

		rowTimeDetected, err := s.getTimestampFromRow(qr, row)
		if err != nil {
			return nil, err
		}

		// if the row time is not the same as the forecast time, we don't check the row
		if !s.compareForecastTime(*rowTimeDetected, alert.Config.TimeInterval) {
			continue
		}

		if row[rowLen] == nil {
			continue
		}

		switch row[rowLen].(type) {
		case float64:
			value += row[rowLen].(float64)
		default:
			return nil, ErrInvalidTableCell
		}
	}

	return s.newNotification(ctx, alert, nil, value, alertID, nil)
}

// newNotification adds a notification(detected alert) to the alert
func (s *AnalyticsAlertsService) newNotification(ctx context.Context, alert *domain.Alert, row *[]bigquery.Value, value float64, alertID string, rowTimeDetected *time.Time) (*domain.Notification, error) {
	if !s.checkCondition(alert.Config, value) {
		return nil, nil
	}

	now := time.Now().UTC()

	expireBy, err := s.getExpireByTime(alert.Config.TimeInterval, now)
	if err != nil {
		return nil, err
	}

	timeDetected := now

	if alert.Config.TimeInterval == report.TimeIntervalDay && rowTimeDetected != nil {
		timeDetected = *rowTimeDetected
	}

	notification := &domain.Notification{
		Alert:        s.alertsDal.GetRef(ctx, alertID),
		Value:        value,
		Customer:     alert.Customer,
		Recipients:   alert.Recipients,
		TimeDetected: timeDetected,
		Etag:         alert.Etag,
		ExpireBy:     expireBy,
	}

	if len(alert.Config.Rows) > 0 && alert.Config.Condition != domain.ConditionForecast {
		rowBreakdown := (*row)[0]

		breakdown, err := query.BigqueryValueToString(rowBreakdown)
		if err != nil {
			return nil, err
		}

		breakdownLabel, err := s.getBreakdownLabel(alert.Config.Rows)
		if err != nil {
			return nil, err
		}

		notification.Breakdown = &breakdown
		notification.BreakdownLabel = breakdownLabel
	}

	notification.Period = getFormattedDate(alert.Config.TimeInterval, notification.TimeDetected)

	return notification, nil
}

// getExpireByTime returns the expire time for the notification based on the time interval
func (s *AnalyticsAlertsService) getExpireByTime(timeInterval report.TimeInterval, now time.Time) (time.Time, error) {
	truncateTime := now.Truncate(time.Hour * 24)

	switch timeInterval {
	case report.TimeIntervalDay:
		return truncateTime.AddDate(0, 2, 0), nil
	case report.TimeIntervalWeek:
		return truncateTime.AddDate(0, 3, 0), nil
	case report.TimeIntervalMonth:
		return truncateTime.AddDate(0, 6, 0), nil
	case report.TimeIntervalQuarter:
		return truncateTime.AddDate(1, 0, 0), nil
	case report.TimeIntervalYear:
		return truncateTime.AddDate(3, 0, 0), nil
	default:
		return now, errors.New("invalid alert time interval")
	}
}

func (s *AnalyticsAlertsService) buildBreakdownFilter(ctx context.Context, alert *domain.Alert, alertID, row string) (*report.ConfigFilter, error) {
	var (
		err                         error
		excludedValues              []string
		unsentDetectedNotifications int
		queryLimit                  int
	)

	limitOrder := "desc"
	if alert.Config.Operator == report.MetricFilterLessThan {
		limitOrder = "asc"
	}

	period := getFormattedDate(alert.Config.TimeInterval, time.Now().UTC())

	if alert.Config.TimeInterval != report.TimeIntervalDay {
		excludedValues, unsentDetectedNotifications, err = s.notificationsDal.GetDetectedBreakdowns(ctx, alert.Etag, alertID, period)
		if err != nil {
			return nil, err
		}

		queryLimit = domain.BreakdownLimitValue - unsentDetectedNotifications
	} else {
		queryLimit = domain.BreakdownLimitValue
	}

	if queryLimit <= 0 {
		return nil, nil
	}

	limitMetric := int(alert.Config.Metric)

	return &report.ConfigFilter{
		BaseConfigFilter: report.BaseConfigFilter{
			ID:      row,
			Values:  &excludedValues,
			Inverse: true,
		},
		Limit:       queryLimit,
		LimitOrder:  &limitOrder,
		LimitMetric: &limitMetric,
	}, nil
}

func getFormattedDate(timeInterval report.TimeInterval, date time.Time) string {
	switch timeInterval {
	case report.TimeIntervalDay:
		return date.Format("2006-01-02")
	case report.TimeIntervalWeek:
		year, week := date.ISOWeek()
		return fmt.Sprintf("%d-W%02d", year, week)
	case report.TimeIntervalMonth:
		return date.Format("2006-01")
	case report.TimeIntervalQuarter:
		q := (int(date.Month()) / 3) + 1
		return fmt.Sprintf("%s-Q%d", date.Format("2006"), q)
	case report.TimeIntervalYear:
		return date.Format("2006")
	}

	return ""
}

// toWebhookEvent converts a notification and alert so that it can be usable by zapier customers
func toWebhookEvent(n *domain.Notification, a *domain.Alert) any {
	alertAPI, _ := toAlertAPI(a)

	return WebhookAlertNotification{
		Alert:          *alertAPI,
		Breakdown:      n.Breakdown,
		BreakdownLabel: n.BreakdownLabel,
		Etag:           n.Etag,
		TimeDetected:   n.TimeDetected,
		TimeSent:       n.TimeSent,
		Period:         n.Period,
		Value:          n.Value,
	}
}
