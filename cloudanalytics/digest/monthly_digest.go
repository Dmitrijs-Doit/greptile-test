package digest

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/firestore"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/announcekit"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	domainOrigin "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain/origin"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	widget "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/widget/domain"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/dashboard"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	spot0 "github.com/doitintl/hello/scheduled-tasks/spot0/api/model"
	"github.com/doitintl/hello/scheduled-tasks/times"
	notificationDomain "github.com/doitintl/notificationcenter/domain"
	notificationcenter "github.com/doitintl/notificationcenter/pkg"
)

// Firestore paths
const (
	widgetsCollectionPath     string = "cloudAnalytics/widgets/cloudAnalyticsWidgets"
	spotScalingCollectionPath string = "spot0/spotApp/asgs"
)

// Platforms
const (
	awsFsPlatform string = "amazon_web_services"
	gcpFsPlatform string = "google_cloud_platform"
	GCP           string = "GCP"
	AWS           string = "AWS"
)

// Colors
const (
	green string = "#4CAF50"
	red   string = "#FF3D00"
	black string = "#000000"
)

// Currency
const (
	defaultCurrency = fixer.USD
)

// URLs
const (
	awsLogoURL    string = "https://storage.googleapis.com/hello-static-assets/logos/amazon-web-services-new.png"
	googleLogoURL string = "https://storage.googleapis.com/hello-static-assets/logos/google-cloud.png"
	flexsaveURL   string = "https://help.doit.com/flexsave/overview"
	contractURL   string = "https://help.doit.com/assets-and-contracts/commitment-contracts"
	spotURL       string = "https://help.doit.com/spot-scaling/overview"
)

var cloudProviderAssets = map[string]string{
	GCP: common.Assets.GoogleCloud,
	AWS: common.Assets.AmazonWebServices,
}

type TrendProperites struct {
	Value string `json:"value"`
	Color string `json:"color"`
}

type MonthlyPlatformDigest struct {
	Platform           string          `json:"platform"`
	LastMonth          float64         `json:"lastMonth"`
	LastMonthFormatted string          `json:"spend"`
	TwoMonthsBefore    float64         `json:"twoMonthsBefore"`
	Trend              TrendProperites `json:"trend"`
	Forecast           float64         `json:"forecastNumber"`
	ForecastFormatted  string          `json:"forecast"`
	ForecastTrend      TrendProperites `json:"fTrend"`
	LogoURL            string          `json:"logoURL"`
}

type MonthlyDigest struct {
	Platforms          []*MonthlyPlatformDigest   `json:"platforms"`
	Flexsave           string                     `json:"flexsave"`
	CustomerID         string                     `json:"customerId"`
	ContractualSavings string                     `json:"contract"`
	SpotScalingSavings string                     `json:"spot"`
	SupportTickets     []*dashboard.TicketSummary `json:"support_tickets"`
	FirstName          string                     `json:"first_name"`
	Customer           *common.Customer           `json:"customer"`
}

func (s *DigestService) GetMonthlyDigest(ctx context.Context) error {
	l := s.loggerProvider(ctx)

	notificationConfigs, err := s.recipientsService.GetNotificationRecipientsForMessageType(ctx, notificationDomain.NotificationMonthlyDigests)
	if err != nil {
		return err
	}

	var errorResponses []error

	// Get last month product updates
	year, lastMonth, _ := time.Now().UTC().AddDate(0, -1, 0).Date()

	changeLogs, err := s.announceKit.GetChangeLogs(ctx, time.Date(year, lastMonth, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		l.Error(err)
	}

	customersNotifications := s.recipientsService.GetCustomersNotifications(notificationConfigs)
	for customerID, notification := range customersNotifications {
		l.Info(customerID)

		monthlyDigest, err := s.getCustomerMonthlyDigest(ctx, customerID)
		if err != nil {
			errorResponses = append(errorResponses, err)
			continue
		}

		if monthlyDigest == nil || len(monthlyDigest.Platforms) == 0 {
			continue
		}

		s.buildNotification(ctx, *monthlyDigest, notification, changeLogs)

		task, err := s.ncClient.CreateSendTask(ctx, *notification)
		if err != nil {
			errorResponses = append(errorResponses, err)
			continue
		}

		l.Infof("Monthly digest notification sent with task name %s", task.GetName())
	}

	if len(errorResponses) > 0 {
		return fmt.Errorf("failed to send monthly digest notifications: %v", errorResponses)
	}

	return nil
}

func (s *DigestService) buildNotification(
	ctx context.Context,
	monthlyDigest MonthlyDigest,
	notification *notificationcenter.Notification,
	changeLogs announcekit.AnnoucekitFeed,
) notificationcenter.Notification {
	year, month, _ := time.Now().UTC().AddDate(0, -1, 0).Date()
	customerCurrency := common.GetCustomerCurrency(monthlyDigest.Customer)

	currencySymbol, ok := fixer.GetCurrencySymbol(customerCurrency)
	if !ok {
		currencySymbol = "$"
	}

	notification.Mock = !common.Production
	notification.Template = "XSTTJSX500MM59M2D08X10STYWQP"
	notification.Data = map[string]interface{}{
		"s":               currencySymbol,
		"d":               monthlyDigest,
		"month":           month.String(),
		"month_short":     month.String()[:3],
		"year":            strconv.Itoa(year),
		"customer_name":   monthlyDigest.Customer.Name,
		"customer_id":     monthlyDigest.Customer.Snapshot.Ref.ID,
		"flexsave_url":    flexsaveURL,
		"contract_url":    contractURL,
		"spot_url":        spotURL,
		"change_logs":     changeLogs,
		"has_change_logs": len(changeLogs.Items) != 0,
	}

	if len(notification.Email) != 0 {
		notification.CC = getAccountManagers(ctx, &monthlyDigest)
		notification.EmailFrom = notificationcenter.NotificationsFrom
	}

	return *notification
}

func (s *DigestService) getCustomerMonthlyDigest(ctx context.Context, customerID string) (*MonthlyDigest, error) {
	l := s.loggerProvider(ctx)
	p := message.NewPrinter(language.English)

	var platforms []*MonthlyPlatformDigest

	customerRef := s.conn.Firestore(ctx).Collection("customers").Doc(customerID)

	customer, err := common.GetCustomer(ctx, customerRef)
	if err != nil {
		return nil, err
	}

	// Skip terminated and inactive customers
	if customer.Classification == common.CustomerClassificationTerminated ||
		customer.Classification == common.CustomerClassificationInactive {
		return nil, nil
	}

	customerCurrency := common.GetCustomerCurrency(customer)

	// Flexsave savings Digest
	savingsNumber, err := s.getFlexsaveSavings(ctx, customer)
	if err != nil {
		l.Warningf("failed to get flexsave savings with error: %s", err)
	}

	savings, err := s.formatCurrency(p, savingsNumber, 0, defaultCurrency, customerCurrency, true)
	if err != nil {
		return nil, err
	}

	// GCP Monthly Digest
	gcpMonthlyDigest, err := s.getCustomersMonthSpendAndTrendByPlatform(ctx, customer, GCP)
	if err == nil && gcpMonthlyDigest.LastMonth > 0 {
		platforms = append(platforms, gcpMonthlyDigest)
	}

	// AWS Monthly Digest
	awsMonthlyDigest, err := s.getCustomersMonthSpendAndTrendByPlatform(ctx, customer, AWS)
	if err == nil && awsMonthlyDigest.LastMonth > 0 {
		platforms = append(platforms, awsMonthlyDigest)
	}

	var cloudProviders []string // Empty slice means all platforms
	if len(platforms) == 1 {
		cloudProviders = []string{cloudProviderAssets[(platforms[0].Platform)]}
	}

	// Contract Discount Monthly Digest
	contractualSavingsNumber, err := s.getContractualSavings(ctx, customer, cloudProviders)
	if err != nil {
		l.Warningf("failed to get contractual savings with error: %s", err)
	}

	contractualSavings, err := s.formatCurrency(p, contractualSavingsNumber, 0, defaultCurrency, customerCurrency, true)
	if err != nil {
		return nil, err
	}

	// Spot Scaling Monthly Digest
	spotSavingsNumber, err := s.getCustomerSpotscaligSavings(ctx, customer)
	if err != nil {
		l.Warningf("failed to get spot scaling savings with error: %s", err)
	}

	spotSavings, err := s.formatCurrency(p, spotSavingsNumber, 0, defaultCurrency, customerCurrency, true)
	if err != nil {
		return nil, err
	}

	// Support Tickets Monthly resolved
	supportTickets, err := s.getCustomerSupportTickets(ctx, customerID)
	if err != nil {
		l.Warningf("failed to get support tickets with error: %s", err)
	}

	return &MonthlyDigest{
		Flexsave:           savings,
		Platforms:          platforms,
		CustomerID:         customerID,
		ContractualSavings: contractualSavings,
		SpotScalingSavings: spotSavings,
		SupportTickets:     supportTickets,
		Customer:           customer,
	}, nil
}

func (s *DigestService) getCustomerSupportTickets(ctx context.Context, customerID string) ([]*dashboard.TicketSummary, error) {
	l := s.loggerProvider(ctx)

	supportTickets, err := s.dashboardDAL.GetCustomerTicketStatistics(ctx, customerID)
	if err != nil {
		l.Error(err)
		return nil, err
	}

	for _, supportTicket := range supportTickets {
		if supportTicket.Score == "offered" || supportTicket.Score == "unoffered" {
			supportTicket.Score = "Rate Now"
		}

		supportTicket.Score = strings.Title(supportTicket.Score)
	}

	return supportTickets, nil
}

func (s *DigestService) getContractualSavings(ctx context.Context, customer *common.Customer, cloudProviders []string) (float64, error) {
	l := s.loggerProvider(ctx)

	year, err := domainQuery.NewCol("year")
	if err != nil {
		return 0, err
	}

	month, err := domainQuery.NewCol("month")
	if err != nil {
		return 0, err
	}

	cols := []*domainQuery.QueryRequestX{year, month}

	costTypeFilter, err := domainQuery.NewFilter(domainQuery.FieldCostType, domainQuery.WithValues([]string{"regular"}))
	if err != nil {
		return 0, err
	}

	filters := []*domainQuery.QueryRequestX{costTypeFilter}

	qr, err := s.getMonthlyReportRequest(ctx, customer.Snapshot.Ref.ID, cols, filters, report.MetricSavings, cloudProviders)
	if err != nil {
		return 0, err
	}

	result, err := s.cloudAnalytics.GetQueryResult(ctx, qr, customer.Snapshot.Ref.ID, "")
	if err != nil {
		return 0, err
	}

	if result.Rows != nil && result.Rows[0] != nil {
		l.Info(result.Rows[0][5])
		return result.Rows[0][5].(float64), nil
	}

	return 0, nil
}

func (s *DigestService) getMonthlyReportRequest(ctx context.Context, customerID string, cols []*domainQuery.QueryRequestX, filters []*domainQuery.QueryRequestX, metric report.Metric, cloudProviders []string) (*cloudanalytics.QueryRequest, error) {
	costType, err := domainQuery.NewRow("cost_type")
	if err != nil {
		return nil, err
	}

	accounts, err := s.cloudAnalytics.GetAccounts(
		ctx,
		customerID,
		&[]string{
			common.Assets.GoogleCloud,
			common.Assets.AmazonWebServices},
		[]*report.ConfigFilter{},
	)
	if err != nil {
		return nil, err
	}

	today := times.CurrentDayUTC()
	ts := report.TimeSettings{
		Mode:           report.TimeSettingsModeLast,
		Amount:         60,
		Unit:           report.TimeSettingsUnitDay,
		IncludeCurrent: true,
	}

	timeSettings, err := cloudanalytics.GetTimeSettings(&ts, report.TimeIntervalDay, nil, today)
	if err != nil {
		return nil, err
	}

	var organization *firestore.DocumentRef

	timezone, currency, err := cloudanalytics.GetTimezoneCurrency(ctx, s.conn.Firestore(ctx), "", "", customerID)
	if err != nil {
		return nil, err
	}

	qr := cloudanalytics.QueryRequest{
		Origin:         domainOrigin.QueryOriginFromContext(ctx),
		CloudProviders: &cloudProviders,
		Accounts:       accounts,
		Filters:        filters,
		Cols:           cols,
		Rows:           []*domainQuery.QueryRequestX{costType},
		Currency:       currency,
		Metric:         metric,
		TimeSettings:   timeSettings,
		Organization:   organization,
		Timezone:       timezone,
		Forecast:       true,
	}

	return &qr, nil
}

func (s *DigestService) getFlexsaveSavings(ctx context.Context, customer *common.Customer) (float64, error) {
	year, month, _ := time.Now().UTC().AddDate(0, -1, 0).Date()
	formattedDate := fmt.Sprintf("%d_%d", month, year)

	fss, err := s.IntegrationsDAL.GetFlexsaveConfigurationCustomer(ctx, customer.Snapshot.Ref.ID)
	if err != nil {
		return 0, err
	}

	var totalSavings float64

	awsSavings := fss.AWS.SavingsHistory[formattedDate]
	gcsSavings := fss.GCP.SavingsHistory[formattedDate]

	if awsSavings != nil {
		totalSavings += awsSavings.Savings
	}

	if gcsSavings != nil {
		totalSavings += gcsSavings.Savings
	}

	return totalSavings, nil
}

func (s *DigestService) getCustomersMonthSpendAndTrendByPlatform(ctx context.Context, customer *common.Customer, platform string) (*MonthlyPlatformDigest, error) {
	l := s.loggerProvider(ctx)

	var reportID string

	switch platform {
	case GCP:
		reportID = "mkzMt3cHTPC14WRH0eER"
	case AWS:
		reportID = "ss2m7rGY0OjuPDyJub6g"
	}

	trendReportWidget, err := s.getSavedWidgetReport(ctx, customer.Snapshot.Ref.ID, "", reportID)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			l.Infof("No saved widget report found for customer %s and report %s", customer.Snapshot.Ref.ID, reportID)
		}

		return nil, err
	}

	customerCurrency := common.GetCustomerCurrency(customer)

	// get year and month for each date
	year, month, _ := time.Now().UTC().AddDate(0, -1, 0).Date()
	beforeTwoMonthYear, beforeTwoMonth, _ := time.Now().UTC().AddDate(0, -2, 0).Date()
	currentYear, currentMonth, _ := time.Now().UTC().Date()

	var platformDigest MonthlyPlatformDigest
	platformDigest.Platform = platform
	p := message.NewPrinter(language.English)

	// spend rows
	for _, row := range trendReportWidget.Data.Rows {
		reportRow := row.([]interface{})

		// last month
		if spend := getMonthlySpendByRow(year, month, reportRow); spend != nil {
			platformDigest.LastMonth = *spend

			platformDigest.LastMonthFormatted, err = s.formatCurrency(p, *spend, 0, trendReportWidget.Config.Currency, customerCurrency, false)
			if err != nil {
				return nil, err
			}
		}

		// before two month
		if spend := getMonthlySpendByRow(beforeTwoMonthYear, beforeTwoMonth, reportRow); spend != nil {
			platformDigest.TwoMonthsBefore = *spend
		}
	}

	// forecast rows
	for _, forecast := range trendReportWidget.Data.ForecastRows {
		forecastRow := forecast.([]interface{})

		// this month forecast
		if spend := getMonthlySpendByRow(currentYear, currentMonth, forecastRow); spend != nil {
			platformDigest.Forecast = *spend

			platformDigest.ForecastFormatted, err = s.formatCurrency(p, *spend, 0, trendReportWidget.Config.Currency, customerCurrency, false)
			if err != nil {
				return nil, err
			}
		}
	}

	// If there are no forecast rows, then get the daily forecast
	if len(trendReportWidget.Data.Rows) > 1 && len(trendReportWidget.Data.ForecastRows) == 0 {
		monthlyForecast, err := s.getDailyForecast(ctx, customer, []string{cloudProviderAssets[platform]})
		if err != nil {
			s.loggerProvider(ctx).Error("failed to get daily forecast: ", err.Error())
		}

		platformDigest.Forecast = monthlyForecast

		platformDigest.ForecastFormatted, err = s.formatCurrency(p, monthlyForecast, 0, trendReportWidget.Config.Currency, customerCurrency, false)
		if err != nil {
			return nil, err
		}
	}

	if platformDigest.ForecastFormatted == "" {
		platformDigest.ForecastFormatted, err = s.formatCurrency(p, 0, 0, trendReportWidget.Config.Currency, customerCurrency, false)
		if err != nil {
			return nil, err
		}
	}

	// trend for month spend
	trend := calculateTrend(platformDigest.LastMonth, platformDigest.TwoMonthsBefore)
	platformDigest.Trend = trendArrow(trend)

	// trend for forecast
	trend = calculateTrend(platformDigest.Forecast, platformDigest.LastMonth)
	platformDigest.ForecastTrend = trendArrow(trend)

	platformDigest.LogoURL = getPlatformLogo(platform)

	return &platformDigest, nil
}

func (s *DigestService) getDailyForecast(ctx context.Context, customer *common.Customer, cloudProviders []string) (float64, error) {
	_, currentMonth, _ := time.Now().UTC().Date()

	year, err := domainQuery.NewCol("year")
	if err != nil {
		return 0, err
	}

	month, err := domainQuery.NewCol("month")
	if err != nil {
		return 0, err
	}

	day, err := domainQuery.NewCol("day")
	if err != nil {
		return 0, err
	}

	cols := []*domainQuery.QueryRequestX{
		year,
		month,
		day,
	}

	filters := []*domainQuery.QueryRequestX{}

	qr, err := s.getMonthlyReportRequest(ctx, customer.Snapshot.Ref.ID, cols, filters, report.MetricCost, cloudProviders)
	if err != nil {
		return 0, err
	}

	result, err := s.cloudAnalytics.GetQueryResult(ctx, qr, customer.Snapshot.Ref.ID, "")
	if err != nil {
		return 0, err
	}

	return sumDailyForecast(result.ForecastRows, currentMonth), nil
}

func sumDailyForecast(forecastRows [][]bigquery.Value, currentMonth time.Month) float64 {
	forecastSpend := 0.0
	if len(forecastRows) == 0 {
		return forecastSpend
	}

	for _, forecastRow := range forecastRows {
		if len(forecastRow) < 5 {
			continue
		}

		monthString := strings.TrimPrefix(fmt.Sprintf("%s", forecastRow[2]), "0")
		if monthString == strconv.Itoa(int(currentMonth)) && len(forecastRow) > 4 && forecastRow[4] != nil {
			forecastSpend += forecastRow[4].(float64)
		}
	}

	return forecastSpend
}

func (s *DigestService) getSavedWidgetReport(ctx context.Context, customerID string, orgID string, reportID string) (*widget.WidgetReport, error) {
	fs := s.conn.Firestore(ctx)

	docID := s.widgetService.BuildWidgetDocID(customerID, orgID, reportID)

	docSnap, err := fs.Collection(widgetsCollectionPath).Doc(docID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var w widget.WidgetReport
	if err := docSnap.DataTo(&w); err != nil {
		return nil, err
	}

	return &w, nil
}

func (d *DigestService) getCustomerSpotscaligSavings(ctx context.Context, customer *common.Customer) (float64, error) {
	fs := d.conn.Firestore(ctx)
	year, month, _ := time.Now().UTC().AddDate(0, -1, 0).Date()
	formattedDate := fmt.Sprintf("%d_%d", year, int(month)-1)

	snaps, err := fs.Collection(spotScalingCollectionPath).Where("customer", "==", customer.Snapshot.Ref).Documents(ctx).GetAll()
	if err != nil {
		return 0, err
	}

	var savings float64

	for _, snap := range snaps {
		var spotAsg spot0.AsgConfiguration
		if err := snap.DataTo(&spotAsg); err != nil {
			continue
		}

		savings += spotAsg.Usage[formattedDate].TotalSavings
	}

	return savings, nil
}

func getMonthlySpendByRow(year int, month time.Month, reportRow []interface{}) *float64 {
	if reportRow == nil || len(reportRow) < 4 {
		return nil
	}

	monthString := strings.TrimPrefix(fmt.Sprintf("%s", reportRow[2]), "0")
	if fmt.Sprintf("%s", reportRow[1]) == strconv.Itoa(year) {
		if monthString == strconv.Itoa(int(month)) && reportRow[3] != nil {
			spend := reportRow[3].(float64)
			return &spend
		}
	}

	return nil
}

func calculateTrend(lastMonth float64, beforeTwoMonths float64) float64 {
	if beforeTwoMonths == 0 {
		return 0
	}

	trend := ((lastMonth - beforeTwoMonths) / beforeTwoMonths) * 100

	return trend
}

func trendArrow(trend float64) TrendProperites {
	if trend > 100 {
		return TrendProperites{
			Value: "↑Over 100",
			Color: green,
		}
	} else if trend > 0 {
		return TrendProperites{
			Value: fmt.Sprintf("↑%.1f", math.Abs(trend)),
			Color: green,
		}
	} else if trend < 0 {
		return TrendProperites{
			Value: fmt.Sprintf("↓%.1f", math.Abs(trend)),
			Color: red,
		}
	}

	return TrendProperites{
		Value: "0",
		Color: black,
	}
}

func (s *DigestService) formatCurrency(p *message.Printer, amount float64, fracDigits int, fromCurrency fixer.Currency, currency string, enableEmpty bool) (string, error) {
	if enableEmpty && amount == 0 {
		return "", nil
	}

	converted, err := s.converter.Convert(string(fromCurrency), currency, amount, time.Now().UTC().AddDate(0, 0, -4))
	if err != nil {
		return "", err
	}

	return fixer.FormatCurrencyAmountFloat(p, converted, fracDigits, currency), nil
}

func getPlatformLogo(platform string) string {
	if platform == AWS {
		return awsLogoURL
	}

	return googleLogoURL
}

func getPlatform(platform string) string {
	if platform == GCP {
		return common.Assets.GoogleCloud
	}

	return common.Assets.AmazonWebServices
}

func getAccountManagers(ctx context.Context, monthlyDigest *MonthlyDigest) []string {
	var accountManagersEmail []string

	accountManagers, err := common.GetCustomerAccountManagers(ctx, monthlyDigest.Customer, common.AccountManagerCompanyDoit)
	if err == nil {
		for _, accountManager := range accountManagers {
			accountManagersEmail = append(accountManagersEmail, accountManager.Email)
		}
	}

	return accountManagersEmail
}
