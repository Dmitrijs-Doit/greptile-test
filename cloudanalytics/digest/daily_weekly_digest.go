package digest

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/times"
	notificationDomain "github.com/doitintl/notificationcenter/domain"
	notificationCenter "github.com/doitintl/notificationcenter/pkg"
	notificationService "github.com/doitintl/notificationcenter/service"
)

const (
	dailyDigestEps               float64 = 0.001
	lastSpendThresholdPercentage float64 = 0.6
	maxPartialDataDaysSkipped    int     = 3
)

type Data struct {
	AttributionData *AttributionData                `firestore:"data"`
	Customer        string                          `firestore:"customer"`
	Organization    string                          `firestore:"organization"`
	Attribution     string                          `firestore:"attribution"`
	AttributionName string                          `firestore:"attributionName"`
	DigestDate      time.Time                       `firestore:"digestDate"`
	Timestamp       time.Time                       `firestore:"timestamp,serverTimestamp"`
	Currency        fixer.Currency                  `firestore:"currency"`
	Notification    notificationCenter.Notification `firestore:"notification"`
	Frequency       Frequency                       `firestore:"frequency"`
}

type AttributionData struct {
	LastMonthTotal    *float64  `firestore:"lastMonthTotal"`
	CurrentMonthTotal *float64  `firestore:"currentMonthTotal"`
	LastDayTotal      *float64  `firestore:"lastDayTotal"`
	WeekToLastDay     *float64  `firestore:"weekToLastDay"`
	LastMonthToDate   *float64  `firestore:"lastMonthToDate"`
	Trend             *float64  `firestore:"trend"`
	ImagePath         *string   `firestore:"imagePath"`
	ImagePathWeekly   *string   `firestore:"imagePathWeekly"`
	LastDayFullData   time.Time `firestore:"lastDayFullData"`
}

type Frequency string

const (
	FrequencyDaily  Frequency = "Daily"
	FrequencyWeekly           = "Weekly"
)

type GenerateTaskRequest struct {
	Identifier
	Frequency    Frequency                       `json:"frequency"`
	Notification notificationCenter.Notification `json:"notification"`
}

type Identifier struct {
	CustomerID     string `json:"customerId"`
	AttributionID  string `json:"attributionId"`
	OrganizationID string `json:"organizationId"`
}

func (i *Identifier) String() string {
	if i.OrganizationID == "" {
		return fmt.Sprintf("%s_%s", i.CustomerID, i.AttributionID)
	}

	return fmt.Sprintf("%s_%s_%s", i.CustomerID, i.AttributionID, i.OrganizationID)
}

type RowValues struct {
	year  int
	month int
	day   int
	cost  float64
	err   error
}

// ScheduleDaily is triggered by a scheduled job
// gets all of the recipients that opted in for a daily digest
// schedules a cloud task for each customer-attribution-organization combination
func (s *DigestService) ScheduleDaily(ctx context.Context) error {
	l := s.loggerProvider(ctx)

	// skip daily digest on the first day of the month
	dayOfMonth := time.Now().UTC().Day()
	if dayOfMonth == 1 {
		return nil
	}

	notificationConfigs, err := s.recipientsService.GetNotificationRecipientsForMessageType(
		ctx,
		notificationDomain.NotificationDailyDigests,
	)
	if err != nil {
		return err
	}

	for k, p := range s.getDigestTaskPayloads(ctx, notificationConfigs, FrequencyDaily) {
		if err := s.setGenerateTask(ctx, k, p); err != nil {
			l.Errorf("error scheduling daily digest task with error: %s", err)
		}
	}

	return nil
}

func (s *DigestService) ScheduleWeekly(ctx context.Context) error {
	l := s.loggerProvider(ctx)

	// skip weekly digest on the first week of the month
	dayOfMonth := time.Now().UTC().Day()
	if dayOfMonth <= 7 {
		return nil
	}

	notificationConfigs, err := s.recipientsService.GetNotificationRecipientsForMessageType(
		ctx,
		notificationDomain.NotificationWeeklyDigests,
	)
	if err != nil {
		return err
	}

	for k, p := range s.getDigestTaskPayloads(ctx, notificationConfigs, FrequencyWeekly) {
		if err := s.setGenerateTask(ctx, k, p); err != nil {
			l.Error("error scheduling weekly digest task with error: %s", err)
		}
	}

	return nil
}

func (s *DigestService) getDigestTaskPayloads(ctx context.Context, configs []notificationDomain.NotificationConfig, f Frequency) map[string]*GenerateTaskRequest {
	l := s.loggerProvider(ctx)
	digestPayloads := make(map[string]*GenerateTaskRequest)

	var notification string

	switch f {
	case FrequencyDaily:
		notification = fmt.Sprint(notificationDomain.NotificationDailyDigests)
	case FrequencyWeekly:
		notification = fmt.Sprint(notificationDomain.NotificationWeeklyDigests)
	default:
		l.Errorf("invalid frequency")
		return digestPayloads
	}

	for _, c := range configs {
		notificationSettings, ok := c.SelectedNotifications[notification]
		if !ok || len(notificationSettings.Attributions) == 0 || c.CustomerRef == nil {
			l.Errorf("can't schedule digest for config, missing settings")
			continue
		}

		orgID := ""

		if c.UserRef != nil {
			user, err := common.GetUser(ctx, c.UserRef)
			if err != nil {
				continue
			}

			if len(user.Organizations) > 0 {
				orgID = user.Organizations[0].ID
			}
		}

		for _, a := range notificationSettings.Attributions {
			identifier := Identifier{
				CustomerID:     c.CustomerRef.ID,
				AttributionID:  a.ID,
				OrganizationID: orgID,
			}

			emails := c.GetEmailTargets()
			slacks := []notificationCenter.Slack{}

			for _, slack := range c.GetSlackTargets() {
				slacks = append(slacks, notificationCenter.Slack{
					Channel:     slack.ID,
					AccessToken: slack.AccessToken,
				})
			}

			// collect all recipients for the same digest into one notification
			if payload, ok := digestPayloads[identifier.String()]; ok {
				notificationService.MergeConfigTargets(c, &payload.Notification)
				continue
			}

			notification := notificationCenter.Notification{
				Email: emails,
				Slack: slacks,
			}

			if len(notification.Email) != 0 {
				notification.EmailFrom = notificationCenter.NotificationsFrom
			}

			digestPayloads[identifier.String()] = &GenerateTaskRequest{
				Identifier:   identifier,
				Frequency:    f,
				Notification: notification,
			}
		}
	}

	return digestPayloads
}

func (s *DigestService) setGenerateTask(ctx context.Context, identifier string, req *GenerateTaskRequest) error {
	config := common.CloudTaskConfig{
		Method: cloudtaskspb.HttpMethod_POST,
		Path:   "/tasks/analytics/digest/generate-worker",
		Queue:  common.TaskQueueCloudAnalyticsDigest,
	}

	if _, err := s.conn.CloudTaskClient.CreateTask(ctx, config.Config(req)); err != nil {
		return fmt.Errorf("creating digest genereate task for %s failed with error %s", identifier, err)
	}

	return nil
}

func (s *DigestService) setSendTask(ctx context.Context, data *Data) error {
	if !hasSpend(data.AttributionData) {
		return nil
	}

	config := common.CloudTaskConfig{
		Method: cloudtaskspb.HttpMethod_POST,
		Path:   "/tasks/analytics/digest/send-worker",
		Queue:  common.TaskQueueSendgrid,
	}

	if _, err := s.conn.CloudTaskClient.CreateTask(ctx, config.Config(data)); err != nil {
		return fmt.Errorf("creating digest send task for %s failed with error %s", data.Attribution, err)
	}

	return nil
}

func (s *DigestService) Generate(ctx context.Context, dayParam int, req *GenerateTaskRequest) error {
	qr, err := s.getReportRequest(ctx, req)
	if err != nil {
		return err
	}

	result, err := s.cloudAnalytics.GetQueryResult(ctx, qr, req.CustomerID, "")
	if err != nil {
		return err
	}

	d, err := s.getData(ctx, dayParam, &result, req, qr)
	if err != nil {
		return err
	}

	return s.setSendTask(ctx, &d)
}

func (s *DigestService) getReportRequest(ctx context.Context, req *GenerateTaskRequest) (*cloudanalytics.QueryRequest, error) {
	fs := s.conn.Firestore(ctx)
	filter := report.ConfigFilter{
		BaseConfigFilter: report.BaseConfigFilter{
			ID:     fmt.Sprintf("%s:%s", metadata.MetadataFieldTypeAttribution, metadata.MetadataFieldTypeAttribution),
			Key:    string(metadata.MetadataFieldTypeAttribution),
			Type:   metadata.MetadataFieldTypeAttribution,
			Values: &[]string{req.AttributionID},
		},
	}
	filters := []*report.ConfigFilter{&filter}
	rows := []string{"datetime:year", "datetime:month"}
	cols := []string{"datetime:day"}
	requestFilters, err := cloudanalytics.GetFilters(filters, rows, cols)

	if err != nil {
		return nil, err
	}

	accounts, err := s.cloudAnalytics.GetAccounts(ctx, req.CustomerID, nil, filters)
	if err != nil {
		return nil, err
	}

	attributions, err := s.cloudAnalytics.GetAttributions(ctx, requestFilters, rows, cols, req.CustomerID)
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
	ts := report.TimeSettings{
		Mode:           report.TimeSettingsModeLast,
		Amount:         2,
		Unit:           report.TimeSettingsUnitMonth,
		IncludeCurrent: true,
	}

	timeSettings, err := cloudanalytics.GetTimeSettings(&ts, report.TimeIntervalDay, nil, today)
	if err != nil {
		return nil, err
	}

	var organization *firestore.DocumentRef
	if req.OrganizationID != "" {
		organization = fs.Collection("customers").Doc(req.CustomerID).Collection("customerOrgs").Doc(req.OrganizationID)
	}

	timezone, currency, err := cloudanalytics.GetTimezoneCurrency(ctx, s.conn.Firestore(ctx), "", "", req.CustomerID)
	if err != nil {
		return nil, err
	}

	qr := cloudanalytics.QueryRequest{
		Origin:       domainOrigin.QueryOriginFromContext(ctx),
		Accounts:     accounts,
		Filters:      requestFilters,
		Attributions: attributions,
		Cols:         requestCols,
		Rows:         requestRows,
		Currency:     currency,
		Metric:       report.MetricCost,
		TimeSettings: timeSettings,
		Organization: organization,
		Timezone:     timezone,
	}
	qr.IsCSP = req.CustomerID == domainQuery.CSPCustomerID

	return &qr, nil
}

func (s *DigestService) getData(
	ctx context.Context,
	dayParam int,
	result *cloudanalytics.QueryResult,
	req *GenerateTaskRequest,
	qr *cloudanalytics.QueryRequest,
) (Data, error) {
	dDate := time.Now().UTC().Truncate(24 * time.Hour)
	if dayParam > 0 {
		dDate = time.Date(dDate.Year(), dDate.Month(), dayParam, 0, 0, 0, 0, time.UTC)
	}

	data, err := s.getValues(ctx, req, result.Rows, dayParam)
	if err != nil {
		return Data{}, err
	}

	d := Data{
		AttributionData: data,
		Customer:        req.CustomerID,
		Attribution:     req.AttributionID,
		AttributionName: qr.Attributions[0].Key,
		Organization:    req.OrganizationID,
		DigestDate:      dDate,
		Currency:        qr.Currency,
		Notification:    req.Notification,
		Frequency:       req.Frequency,
	}

	return d, nil
}

// getValues generates the cost values and image path
func (s *DigestService) getValues(ctx context.Context, req *GenerateTaskRequest, rows [][]bigquery.Value, dayParam int) (*AttributionData, error) {
	now := time.Now().UTC()
	year, m, day := now.Date()
	month := int(m)

	if dayParam > 0 {
		day = dayParam
	} else if day > 1 {
		day--
	}

	p := time.Date(now.Year(), now.Month(), 0, 0, 0, 0, 0, time.UTC)
	prevYear, prevM, _ := p.Date()
	prevMonth := int(prevM)

	lastMonthTotal, err := getTotal(rows, prevYear, prevMonth, 0, 0)
	if err != nil {
		return nil, err
	}

	var lastDayTotal *float64

	lastDayTotal, lastDay, err := getLastDayTotal(rows, lastMonthTotal, year, month, day)
	if err != nil {
		return nil, err
	}

	day = lastDay.Day()

	currentMonthTotal, err := getTotal(rows, year, month, day, 0)
	if err != nil {
		return nil, err
	}

	lastMonthToDate, err := getTotal(rows, prevYear, prevMonth, day, 0)
	if err != nil {
		return nil, err
	}

	weekToLastDay, err := getTotal(rows, year, month, day, day-7)
	if err != nil {
		return nil, err
	}

	var trend *float64
	if currentMonthTotal != nil && lastMonthToDate != nil && *lastMonthToDate != 0 {
		trend = common.Float(*currentMonthTotal / *lastMonthToDate - 1)
	}

	d := &AttributionData{
		CurrentMonthTotal: currentMonthTotal,
		LastMonthTotal:    lastMonthTotal,
		Trend:             trend,
		LastMonthToDate:   lastMonthToDate,
		WeekToLastDay:     weekToLastDay,
		LastDayTotal:      lastDayTotal,
		LastDayFullData:   lastDay,
	}
	if isDataEmpty(d) {
		return nil, nil
	}

	switch req.Frequency {
	case FrequencyDaily:
		imagePath, err := s.getDigestHighchartsImagePath(ctx, req, d, FrequencyDaily)
		if err != nil {
			return nil, err
		}

		d.ImagePath = imagePath
	case FrequencyWeekly:
		imagePathWeekly, err := s.getDigestHighchartsImagePath(ctx, req, d, FrequencyWeekly)
		if err != nil {
			return nil, err
		}

		d.ImagePathWeekly = imagePathWeekly
	default:
		return nil, fmt.Errorf("invalid frequency: %s", req.Frequency)
	}

	return d, nil
}

// getTotal calculate cost sum per date
func getTotal(rows [][]bigquery.Value, y, m, d, prevD int) (*float64, error) {
	total := 0.0
	// this var help us identify wether we got results for a month
	monthMatchFound := false

	for _, row := range rows {
		rowResult := getRowValues(row)
		if rowResult.err != nil {
			return nil, rowResult.err
		}

		if rowResult.year == y && rowResult.month == m {
			monthMatchFound = true

			if d == 0 {
				total += rowResult.cost
			}

			if rowResult.day > prevD && rowResult.day <= d {
				total += rowResult.cost
			}
		}
	}

	if !monthMatchFound {
		return nil, nil
	}

	return &total, nil
}

// getLastDayTotal calculate last day total with non zero cost
func getLastDayTotal(rows [][]bigquery.Value, lastMonthTotal *float64, y, m, d int) (*float64, time.Time, error) {
	var lastDayTotal *float64

	var err error

	lastDay := time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)

	for {
		lastDayTotal, err = getTotal(rows, y, m, d, d-1)
		if err != nil {
			return nil, lastDay, err
		}

		if d <= 1 || (lastDayTotal != nil && *lastDayTotal > dailyDigestEps) {
			lastDay = time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
			break
		}

		d--
	}

	if lastMonthTotal != nil {
		daysInLastMonth := time.Date(y, time.Month(m), 0, 0, 0, 0, 0, time.UTC).Day()
		lastMonthAvg := *lastMonthTotal / float64(daysInLastMonth)

		skipUpTo := d - maxPartialDataDaysSkipped + 1
		if skipUpTo < 1 {
			skipUpTo = 1
		}

		for {
			if d <= skipUpTo || (lastDayTotal != nil &&
				lastMonthAvg-*lastDayTotal < lastMonthAvg*lastSpendThresholdPercentage) {
				break
			}

			d--
			lastDay = lastDay.AddDate(0, 0, -1)
			lastDayTotal, err = getTotal(rows, y, m, d, d-1)

			if err != nil {
				return nil, lastDay, err
			}
		}
	}

	return lastDayTotal, lastDay, nil
}

// getRowValues parses bigquery values to ints, floats and strings
func getRowValues(row []bigquery.Value) *RowValues {
	var year, month, day int

	var err error

	if len(row) != 4 {
		invalidRowResult(nil)
	}

	yearString, ok := row[0].(string)
	if !ok {
		return invalidRowResult(nil)
	}

	year, err = strconv.Atoi(yearString)
	if err != nil {
		return invalidRowResult(err)
	}

	monthString, ok := row[1].(string)
	if !ok {
		return invalidRowResult(nil)
	}

	month, err = strconv.Atoi(monthString)
	if err != nil {
		return invalidRowResult(err)
	}

	dayString, ok := row[2].(string)
	if !ok {
		return invalidRowResult(nil)
	}

	day, err = strconv.Atoi(dayString)
	if err != nil {
		return invalidRowResult(err)
	}

	cost, ok := row[3].(float64)
	if !ok {
		return invalidRowResult(nil)
	}

	return &RowValues{
		year:  year,
		month: month,
		day:   day,
		cost:  cost,
		err:   nil,
	}
}

func invalidRowResult(err error) *RowValues {
	defaultErr := errors.New("invalid digest query result")
	if err != nil {
		defaultErr = err
	}

	return &RowValues{
		year:  0,
		month: 0,
		day:   0,
		cost:  0.0,
		err:   defaultErr,
	}
}

func isDataEmpty(d *AttributionData) bool {
	return d.CurrentMonthTotal == nil &&
		d.LastMonthTotal == nil &&
		d.LastDayTotal == nil &&
		d.LastMonthToDate == nil &&
		d.Trend == nil
}

func hasSpend(d *AttributionData) bool {
	if d != nil && d.CurrentMonthTotal != nil && *d.CurrentMonthTotal > dailyDigestEps {
		return true
	}

	return false
}

func (s *DigestService) Send(ctx context.Context, d *Data) error {
	l := s.loggerProvider(ctx)

	if !hasSpend(d.AttributionData) {
		l.Infof("no spend for %s", d.Attribution)
		return nil
	}

	switch d.Frequency {
	case FrequencyDaily:
		if d.AttributionData.ImagePath == nil {
			l.Errorf("digest send error: no daily image path or no data")
			return nil
		}
	case FrequencyWeekly:
		if d.AttributionData.ImagePathWeekly == nil {
			l.Errorf("digest send error: no weekly image path or no data")
			return nil
		}
	default:
		return fmt.Errorf("invalid frequency: %s", d.Frequency)
	}

	notificationToSend := buildNotification(d)

	_, err := s.ncClient.CreateSendTask(ctx, notificationToSend)
	if err != nil {
		return err
	}

	return nil
}

func buildNotification(d *Data) notificationCenter.Notification {
	currencySymbol := d.Currency.Symbol()
	if currencySymbol == "" {
		currencySymbol = fixer.USD.Symbol()
	}

	msgPrinter := message.NewPrinter(language.English)

	currentYear, currentMonth, _ := d.DigestDate.Date()
	lastDayFullData := d.AttributionData.LastDayFullData.Day()
	lastDayWithSuffix := common.GetDayWithSuffix(lastDayFullData)
	lastMonth := d.DigestDate.AddDate(0, -1, 0).Month()
	aboveOrBelow := "above"
	trend := "0"
	trendWithArrow := "0"

	if d.AttributionData.Trend != nil {
		trend = fmt.Sprintf("%.2f", math.Abs(*d.AttributionData.Trend*100))
		trendWithArrow = fmt.Sprintf("↑%s", trend)

		if *d.AttributionData.Trend < 0 {
			aboveOrBelow = "below"
			trendWithArrow = fmt.Sprintf("↓%s", trend)
		}
	}

	currentMonthTotal := 0.0
	if d.AttributionData.CurrentMonthTotal != nil {
		currentMonthTotal = *d.AttributionData.CurrentMonthTotal
	}

	weekToLastDayTotal := 0.0
	if d.AttributionData.WeekToLastDay != nil {
		weekToLastDayTotal = *d.AttributionData.WeekToLastDay
	}

	lastDayToal := 0.0
	if d.AttributionData.CurrentMonthTotal != nil {
		lastDayToal = *d.AttributionData.LastDayTotal
	}

	unsubscribeURL := fmt.Sprintf("https://%s/customers/%s/notifications", common.Domain, d.Customer)
	attributionLink := fmt.Sprintf("https://%s/customers/%s/analytics/attributions/%s", common.Domain, d.Customer, d.Attribution)

	d.Notification.Mock = !common.Production
	d.Notification.Template = notificationCenter.CADailyDigestTemplate
	data := map[string]interface{}{
		"current_month":       currentMonth.String(),
		"current_year":        currentYear,
		"current_month_total": getCurrencyFormat(msgPrinter, currencySymbol, currentMonthTotal),
		"week_to_last_day":    getCurrencyFormat(msgPrinter, currencySymbol, weekToLastDayTotal),
		"last_date_total":     getCurrencyFormat(msgPrinter, currencySymbol, lastDayToal),
		"last_date":           lastDayWithSuffix,
		"last_month":          lastMonth.String(),
		"trend":               trend,
		"trend_with_arrow":    trendWithArrow,
		"above_below":         aboveOrBelow,
		"image_url":           d.AttributionData.ImagePath,
		"image_url_weekly":    d.AttributionData.ImagePathWeekly,
		"attribution_name":    d.AttributionName,
		"attribution_link":    attributionLink,
		"unsubscribe_url":     unsubscribeURL,
		"frequency":           d.Frequency,
	}
	d.Notification.Data = data
	d.Notification.EmailFrom = notificationCenter.NotificationsFrom

	return d.Notification
}

func getCurrencyFormat(msgPrinter *message.Printer, currencySymbol string, amount float64) string {
	return msgPrinter.Sprintf("%s%.2f", currencySymbol, amount)
}
