package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/cloudtasks/apiv2/cloudtaskspb"
	"cloud.google.com/go/firestore"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	cloudtasks "github.com/doitintl/cloudtasks/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/dal"
	alertsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/domain"
	alerttier "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/service/alerttier"
	alertTierIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/alerts/service/alerttier/iface"
	attributionsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal"
	attributionDALIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/caownerchecker/service"
	caownercheckerIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/caownerchecker/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	configsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/config/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	metadataService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service"
	metadataIface "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/service/iface"
	metricsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/dal"
	metrics "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metrics/dal/iface"
	domainQuery "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/query/domain"
	reportDAL "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/dal"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDAL "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/doitemployees"
	"github.com/doitintl/hello/scheduled-tasks/fixer"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	labelsDal "github.com/doitintl/hello/scheduled-tasks/labels/dal"
	labelsIface "github.com/doitintl/hello/scheduled-tasks/labels/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/mailer"
	userDal "github.com/doitintl/hello/scheduled-tasks/user/dal"
	"github.com/doitintl/hello/scheduled-tasks/user/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/zapier/dispatch"
	tier "github.com/doitintl/tiers/service"
)

var (
	ErrNoAlertID       = errors.New("alert id is missing")
	ErrNoCollaborators = errors.New("collaborators are missing")
	ErrNoAuthorization = errors.New("not authorized")
)

const timestampFormat = "Jan DAY 2006"

type RecipientsBodyMap map[string][]EmailBody

var timeIntervalLabelMap = map[report.TimeInterval]string{
	report.TimeIntervalDay:     "Daily",
	report.TimeIntervalWeek:    "Weekly",
	report.TimeIntervalMonth:   "Monthly",
	report.TimeIntervalQuarter: "Quarterly",
	report.TimeIntervalYear:    "Yearly",
}

type AnalyticsAlertsService struct {
	loggerProvider   logger.Provider
	conn             *connection.Connection
	alertsDal        alertsDal.Alerts
	notificationsDal alertsDal.Notifications
	collab           collab.Icollab
	cloudTaskClient  cloudtasks.CloudTaskClient
	cloudAnalytics   cloudanalytics.CloudAnalytics
	configs          configsDal.Configs
	metrics          metrics.Metrics
	customersDAL     customerDAL.Customers
	attributionsDAL  attributionDALIface.Attributions
	employeeService  doitemployees.ServiceInterface
	metadataService  metadataIface.MetadataIface
	userDal          iface.IUserFirestoreDAL
	caOwnerChecker   caownercheckerIface.CheckCAOwnerInterface
	labelsDal        labelsIface.Labels
	alertTierService alertTierIface.AlertTierService
	eventDispatcher  dispatch.Dispatcher
}

func NewAnalyticsAlertsService(
	ctx context.Context, loggerProvider logger.Provider, conn *connection.Connection, cloudTaskClient cloudtasks.CloudTaskClient) (*AnalyticsAlertsService, error) {
	customerDal := customerDAL.NewCustomersFirestoreWithClient(conn.Firestore)
	reportDal := reportDAL.NewReportsFirestoreWithClient(conn.Firestore)

	cloudAnalytics, err := cloudanalytics.NewCloudAnalyticsService(loggerProvider, conn, reportDal, customerDal)
	if err != nil {
		return nil, err
	}

	tierService := tier.NewTiersService(conn.Firestore)

	doitEmployeesService := doitemployees.NewService(conn)

	alertTierService := alerttier.NewAlertTierService(loggerProvider, tierService, doitEmployeesService)

	return &AnalyticsAlertsService{
		loggerProvider,
		conn,
		dal.NewAlertsFirestoreWithClient(conn.Firestore),
		dal.NewNotificationsFirestoreWithClient(conn.Firestore),
		&collab.Collab{},
		cloudTaskClient,
		cloudAnalytics,
		configsDal.NewConfigsFirestoreWithClient(conn.Firestore),
		metricsDal.NewMetricsFirestoreWithClient(conn.Firestore),
		customerDal,
		attributionsDal.NewAttributionsFirestoreWithClient(conn.Firestore),
		doitemployees.NewService(conn),
		metadataService.NewMetadataService(ctx, loggerProvider, conn),
		userDal.NewUserFirestoreDALWithClient(conn.Firestore),
		service.NewCAOwnerChecker(conn),
		labelsDal.NewLabelsFirestoreWithClient(conn.Firestore),
		alertTierService,
		dispatch.NewEventDispatcher(loggerProvider, conn.Firestore),
	}, nil
}

func (s *AnalyticsAlertsService) ShareAlert(ctx context.Context, newCollabs []collab.Collaborator, public *collab.PublicAccess, alertID, requesterEmail string, userID string, customerID string) error {
	customer, err := s.customersDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	if customer.PresentationMode != nil && customer.PresentationMode.Enabled {
		alert, err := s.alertsDal.GetAlert(ctx, alertID)
		if err != nil {
			return err
		}

		if alert.Customer.ID == customer.PresentationMode.CustomerID {
			return ErrNoAuthorization
		}
	}

	isCAOwner, err := s.caOwnerChecker.CheckCAOwner(ctx, s.employeeService, userID, requesterEmail)
	if err != nil {
		return err
	}

	alert, err := s.alertsDal.GetAlert(ctx, alertID)
	if err != nil {
		return err
	}

	if err := s.collab.ShareAnalyticsResource(ctx, alert.Collaborators, newCollabs, public, alertID, requesterEmail, s.alertsDal, isCAOwner); err != nil {
		return err
	}

	return nil
}

func (s *AnalyticsAlertsService) SendEmails(ctx context.Context) error {
	l := s.loggerProvider(ctx)

	customerIDs, err := s.notificationsDal.GetCustomers(ctx)
	if err != nil {
		return err
	}

	for _, customerID := range customerIDs {
		config := common.CloudTaskConfig{
			Method: cloudtaskspb.HttpMethod_GET,
			Path:   "/tasks/analytics/alerts/send/" + customerID,
			Queue:  common.TaskQueueCloudAnalyticsAlerts,
		}
		conf := config.Config(nil)

		if _, err = s.cloudTaskClient.CreateTask(ctx, conf); err != nil {
			l.Errorf(err.Error())
			continue
		}
	}

	return nil
}

// build email body from
func (s *AnalyticsAlertsService) SendEmailsToCustomer(ctx context.Context, customerID string) error {
	log := s.loggerProvider(ctx)

	recipientsBodyMap := make(RecipientsBodyMap)

	notificationsMap, err := s.notificationsDal.GetAlertDetectedNotifications(ctx, customerID)
	if err != nil {
		return err
	}

	for _, notifications := range notificationsMap {
		alert, err := s.alertsDal.GetAlert(ctx, notifications[0].Alert.ID)
		if err != nil {
			log.Errorf("error getting alert, alert ID: %s error: %s", notifications[0].Alert.ID, err)
			continue
		}

		if err := s.buildAlertEmailBody(ctx, recipientsBodyMap, notifications, alert); err != nil {
			log.Errorf("error building alert email body, alert ID: %s, error: %s", notifications[0].Alert.ID, err)
			continue
		}

		if err = s.alertsDal.UpdateAlertNotified(ctx, notifications[0].Alert.ID); err != nil {
			log.Errorf("error updating alert notified, alert ID: %s, error: %s", notifications[0].Alert.ID, err)
			continue
		}
	}

	for _, body := range recipientsBodyMap {
		body[len(body)-1].LastAlertInEmail = true
	}

	if err := s.sendEmailToEachRecipient(ctx, recipientsBodyMap, customerID); err != nil {
		return err
	}

	return nil
}

func (s *AnalyticsAlertsService) DeleteMany(
	ctx context.Context,
	email string,
	alertIDs []string,
) error {
	alerts := make([]*domain.Alert, 0, len(alertIDs))

	for _, id := range alertIDs {
		a, err := s.alertsDal.GetAlert(ctx, id)
		if err != nil {
			return err
		}

		alerts = append(alerts, a)
	}

	for _, alert := range alerts {
		if !alert.IsOwner(email) {
			return domain.ErrorUnAuthorized
		}
	}

	alertRefs := make([]*firestore.DocumentRef, 0, len(alertIDs))
	for _, id := range alertIDs {
		alertRefs = append(alertRefs, s.alertsDal.GetRef(ctx, id))
	}

	return s.labelsDal.DeleteManyObjectsWithLabels(ctx, alertRefs)
}

func (s *AnalyticsAlertsService) buildAlertEmailBody(ctx context.Context, recipientsBodyMap RecipientsBodyMap, notifications []*domain.Notification, alert *domain.Alert) error {
	var validNotification *domain.Notification

	for _, notification := range notifications {
		if alert.Etag == notification.Etag {
			validNotification = notification
			break
		}
	}

	if validNotification == nil {
		return nil
	}

	condition, err := s.buildAlertCondition(ctx, alert)
	if err != nil {
		return err
	}

	body := EmailBody{
		Condition: condition,
		Name:      alert.Name,
	}

	if len(alert.Config.Rows) > 0 {
		if body.BreakdownLabel, err = s.getBreakdownLabel(alert.Config.Rows); err != nil {
			return err
		}
	}

	if err := s.buildBodyNotifications(ctx, notifications, alert, &body); err != nil {
		return err
	}

	s.createTopHitsText(alert.Config.Operator, &body)

	s.sortBodyNotifications(alert.Config.Operator, &body)

	s.sortNotificationsData(&body)

	for _, recipient := range validNotification.Recipients {
		if _, ok := recipientsBodyMap[recipient]; !ok {
			recipientsBodyMap[recipient] = []EmailBody{}
		}

		recipientsBodyMap[recipient] = append(recipientsBodyMap[recipient], body)
	}

	return nil
}

func (s *AnalyticsAlertsService) addTimestampDataToMap(notificationsData map[string]*TimestampData, timeDetected time.Time, timeInterval report.TimeInterval) *TimestampData {
	timestamp := s.getTimestamp(timeDetected, timeInterval)

	if notificationsData[timestamp] == nil {
		notificationsData[timestamp] = &TimestampData{
			Timestamp: timestamp,
			SortValue: timeDetected,
		}
	}

	return notificationsData[timestamp]
}

func (s *AnalyticsAlertsService) addTimestampItem(timeInterval report.TimeInterval, body *EmailBody, notificationsData map[string]*TimestampData, notification *domain.Notification, formattedValue string) {
	if body.BreakdownLabel == nil {
		if timeInterval == report.TimeIntervalDay {
			timestampData := s.addTimestampDataToMap(notificationsData, notification.TimeDetected, timeInterval)
			timestampData.Value = &formattedValue
		} else {
			body.Value = &formattedValue
		}
	} else {
		notificationBreakdown := "N/A"

		if notification.Breakdown != nil {
			notificationBreakdown = *notification.Breakdown
		}

		timestampData := s.addTimestampDataToMap(notificationsData, notification.TimeDetected, timeInterval)
		timestampData.Items = append(timestampData.Items, EmailBodyItem{
			Value:     formattedValue,
			Label:     notificationBreakdown,
			SortValue: notification.Value,
		})
	}
}

func (s *AnalyticsAlertsService) buildBodyNotifications(ctx context.Context, notifications []*domain.Notification, alert *domain.Alert, body *EmailBody) error {
	log := s.loggerProvider(ctx)
	notificationsData := make(map[string]*TimestampData)

	for _, notification := range notifications {
		if notification.Etag != alert.Etag {
			log.Errorf("notification etag does not match alert etag")
			continue
		}

		formattedValue := s.getFormattedValue(alert.Config, notification.Value)

		s.addTimestampItem(alert.Config.TimeInterval, body, notificationsData, notification, formattedValue)

		err := s.notificationsDal.UpdateNotificationTimeSent(ctx, notification)
		if err != nil {
			log.Errorf("error updating notification time sent %s", err)
			continue
		}
	}

	for _, timestampData := range notificationsData {
		body.NotificationsData = append(body.NotificationsData, *timestampData)
	}

	return nil
}

func (s *AnalyticsAlertsService) getExtendedMetricLabel(ctx context.Context, key string) (string, error) {
	extendedMetrics, err := s.configs.GetExtendedMetrics(ctx)
	if err != nil {
		return "", err
	}

	for _, extendedMetric := range extendedMetrics {
		if extendedMetric.Key == key {
			return extendedMetric.Label, nil
		}
	}

	return "", fmt.Errorf("extended metric not found")
}

func (s *AnalyticsAlertsService) buildAlertCondition(ctx context.Context, alert *domain.Alert) (string, error) {
	var metricLabel string

	var condition string

	var operatorString string

	var err error

	valueString := s.getFormattedValue(alert.Config, alert.Config.Values[0])

	switch alert.Config.Operator {
	case report.MetricFilterGreaterThan:
		operatorString = "greater than"
	case report.MetricFilterLessThan:
		operatorString = "less than"
	}

	switch alert.Config.Metric {
	case report.MetricExtended:
		if metricLabel, err = s.getExtendedMetricLabel(ctx, *alert.Config.ExtendedMetric); err != nil {
			return "", err
		}
	case report.MetricCustom:
		customMetric, err := s.metrics.GetCustomMetric(ctx, alert.Config.CalculatedMetric.ID)
		if err != nil {
			return "", err
		}

		metricLabel = customMetric.Name
	default:
		if metricLabel, err = domainQuery.GetMetricString(alert.Config.Metric); err != nil {
			return "", err
		}
	}

	stringArr := []string{
		timeIntervalLabelMap[alert.Config.TimeInterval],
		metricLabel,
		string(alert.Config.Condition),
		operatorString,
	}

	condition = strings.Join(stringArr, " ")
	condition = fmt.Sprintf("%s %s", condition, valueString)

	return condition, nil
}

func (s *AnalyticsAlertsService) createTopHitsText(operator report.MetricFilter, body *EmailBody) {
	var topHits string

	for _, timestampData := range body.NotificationsData {
		if len(timestampData.Items) >= 10 {
			switch operator {
			case report.MetricFilterGreaterThan:
				topHits = "(below are the top 10 hits)"
			case report.MetricFilterLessThan:
				topHits = "(below are the bottom 10 hits)"
			default:
				topHits = "(below are the top 10 hits)"
			}

			body.TopHits = &topHits

			break
		}
	}
}

func (s *AnalyticsAlertsService) sortBodyNotifications(operator report.MetricFilter, body *EmailBody) {
	if body.BreakdownLabel != nil {
		for _, timestampData := range body.NotificationsData {
			items := timestampData.Items

			switch operator {
			case report.MetricFilterGreaterThan:
				sort.Slice(items, func(i, j int) bool {
					return items[i].SortValue > items[j].SortValue
				})
			case report.MetricFilterLessThan:
				sort.Slice(items, func(i, j int) bool {
					return items[i].SortValue < items[j].SortValue
				})
			}
		}
	}
}

func (s *AnalyticsAlertsService) sortNotificationsData(body *EmailBody) {
	sort.Slice(body.NotificationsData, func(i, j int) bool {
		return body.NotificationsData[i].SortValue.Before(body.NotificationsData[j].SortValue)
	})
}

func (s *AnalyticsAlertsService) getFormattedValue(config *domain.Config, value float64) string {
	var formattedValue string

	msgPrinter := message.NewPrinter(language.English)
	if config.Condition == domain.ConditionPercentage {
		formattedValue = msgPrinter.Sprintf("%.2f%%", value)
	} else if config.Metric == report.MetricUsage {
		formattedValue = msgPrinter.Sprintf("%.2f", value)
	} else {
		formattedValue = fixer.FormatCurrencyAmountFloat(msgPrinter, value, 2, string(config.Currency))
	}

	return formattedValue
}

func (s *AnalyticsAlertsService) getBreakdownLabel(rows []string) (*string, error) {
	labelDocID := rows[0]

	labelTypeString := strings.Split(labelDocID, ":")
	if len(labelTypeString) == 2 {
		labelType := metadata.MetadataFieldType(labelTypeString[0])
		labelID := labelTypeString[1]

		if cloudanalytics.TypeRequiresRawTable(labelType) {
			decoded, err := base64.StdEncoding.DecodeString(labelID)
			if err != nil {
				return nil, fmt.Errorf("failed to decode key: %v", err)
			}

			label := string(decoded)

			return &label, nil
		}

		label := domainQuery.KeyMap[labelID].Label

		return &label, nil
	}

	return nil, fmt.Errorf("label id is not valid")
}

func (s *AnalyticsAlertsService) sendEmailToEachRecipient(ctx context.Context, recipientsBodyMap RecipientsBodyMap, customerID string) error {
	l := s.loggerProvider(ctx)

	primaryDomain, err := s.customersDAL.GetPrimaryDomain(ctx, customerID)
	if err != nil {
		return err
	}

	for recipient, alertsBody := range recipientsBodyMap {
		if !common.Production && !common.IsDoitDomain(recipient) {
			l.Infof(`mail to <"%s"> didn't send while in development`, recipient)
			continue
		}

		l.Debugf("%+v", alertsBody)

		personalizations := make([]*mail.Personalization, 0)
		p := mail.NewPersonalization()
		to := mail.NewEmail("", recipient)
		p.AddTos(to)
		p.SetDynamicTemplateData("notifications", alertsBody)
		p.SetDynamicTemplateData("primaryDomain", primaryDomain)
		p.SetDynamicTemplateData("customerID", customerID)

		personalizations = append(personalizations, p)
		if err := mailer.SendEmailWithPersonalizations(personalizations, mailer.Config.DynamicTemplates.CloudAnalyticsAlertsDigest, []string{}); err != nil {
			continue
		}
	}

	return nil
}

func (s *AnalyticsAlertsService) getTimestamp(timeDetected time.Time, timeInterval report.TimeInterval) string {
	if timeInterval != report.TimeIntervalDay {
		return ""
	}

	timestamp := timeDetected.Format(timestampFormat)

	day := common.GetDayWithSuffix(timeDetected.Day())

	return strings.Replace(timestamp, "DAY", day, 1)
}
