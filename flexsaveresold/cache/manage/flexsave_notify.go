package manage

import (
	"context"
	"fmt"
	monitoringDomain "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/monitoring/domain"
	"time"

	"github.com/slack-go/slack"

	mpaDAL "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	customerDAL "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/cache/manage/iface"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/utils"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	devSlackChannel  = "#test-aws-fs-ops-notifications"
	prodSlackChannel = "#fsaws-ops"
	greenColour      = "#36a64f" // Slack attachment color for successful notifications
	redColour        = "#DD2E44"
)

//go:generate mockery --name FlexsaveManageNotify --output ./mocks --filename flexsaveManageNotify.go --structname FlexsaveManageNotify
type FlexsaveManageNotify interface {
	SendActivatedNotification(ctx context.Context, customerID string, nextMonthHourlyCommitment *float64, accounts []string) error
	SendSageMakerActivatedNotification(ctx context.Context, customerID string, accounts []string) error
	SendRDSActivatedNotification(ctx context.Context, customerID string, accounts []string) error
	NotifyAboutPayerConfigSet(ctx context.Context, primaryDomain string, accountID string) error
	SendWelcomeEmail(ctx context.Context, customerID string) error
	NotifyPayerUnsubscriptionDueToCredits(ctx context.Context, primaryDomain string, accountID string) error
	NotifySharedPayerSavingsDiscrepancies(ctx context.Context, discrepancies monitoringDomain.SharedPayerSavingsDiscrepancies) error
}

type flexsaveManageNotify struct {
	loggerProvider logger.Provider
	customersDAL   customerDAL.Customers
	emailService   iface.EmailInterface
	*connection.Connection
}

func NewFlexsaveManageNotify(log logger.Provider, conn *connection.Connection) FlexsaveManageNotify {
	return &flexsaveManageNotify{
		log,
		customerDAL.NewCustomersFirestoreWithClient(conn.Firestore),
		flexsaveresold.NewMail(log, conn),
		conn,
	}
}

func (s *flexsaveManageNotify) SendActivatedNotification(ctx context.Context, customerID string, nextMonthHourlyCommitment *float64, accounts []string) error {
	log := s.loggerProvider(ctx)

	customer, err := s.customersDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	hasFlexsaveEligibleDedicatedPayerAccount, err := mpaDAL.HasFlexsaveEligibleDedicatedPayerAccount(ctx, s.Firestore(ctx), log, customerID)
	if err != nil {
		return err
	}

	var notificationParams slackNotificationParams

	if hasFlexsaveEligibleDedicatedPayerAccount {
		if nextMonthHourlyCommitment == nil {
			log.Warningf("No hourly commitment found for customer: %v", customerID)
		} else {
			notificationParams.HourlyCommitment = *nextMonthHourlyCommitment
		}

		notificationParams.Name = customer.PrimaryDomain
		if err := publishConfigCreatedSlackNotification(ctx, customerID, notificationParams, accounts); err != nil {
			log.Errorf(err.Error())
		}
	}

	return nil
}

func (s *flexsaveManageNotify) SendSageMakerActivatedNotification(ctx context.Context, customerID string, accounts []string) error {
	customer, err := s.customersDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	return publishFlexsaveTypeActivatedSlackNotification(ctx, customerID, customer.PrimaryDomain, accounts, utils.SageMakerFlexsaveType)
}

func (s *flexsaveManageNotify) SendRDSActivatedNotification(ctx context.Context, customerID string, accounts []string) error {
	customer, err := s.customersDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return err
	}

	return publishFlexsaveTypeActivatedSlackNotification(ctx, customerID, customer.PrimaryDomain, accounts, utils.RDSFlexsaveType)
}

func (s *flexsaveManageNotify) NotifyAboutPayerConfigSet(ctx context.Context, primaryDomain string, accountID string) error {
	channel := getAWSFlexsaveSlackChannel()
	fields := []map[string]interface{}{
		{
			"type":  slack.MarkdownType,
			"value": fmt.Sprintf("Payer config set and activated for %s, Account Number: %s <https://%s/flexsave-aws-operations|Here>", primaryDomain, accountID, common.Domain),
		},
	}

	message := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"ts":     time.Now().Unix(),
				"color":  greenColour,
				"fields": fields,
			},
		},
	}

	if _, err := common.PublishToSlack(ctx, message, channel); err != nil {
		return err
	}

	return nil
}

func (s *flexsaveManageNotify) NotifyPayerUnsubscriptionDueToCredits(ctx context.Context, primaryDomain string, accountID string) error {
	channel := getAWSFlexsaveSlackChannel()
	fields := []map[string]interface{}{
		{
			"type":  slack.MarkdownType,
			"value": fmt.Sprintf("Payer %s, Account Number: %s, was *moved to pending* due to active credits detected", primaryDomain, accountID),
		},
	}

	message := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"ts":     time.Now().Unix(),
				"fields": fields,
			},
		},
	}

	if _, err := common.PublishToSlack(ctx, message, channel); err != nil {
		return err
	}

	return nil
}

func (s *flexsaveManageNotify) SendWelcomeEmail(ctx context.Context, customerID string) error {
	log := s.loggerProvider(ctx)

	customer, err := s.customersDAL.GetCustomer(ctx, customerID)
	if err != nil {
		log.Errorf("GetCustomer error: %v for customer %s", err, customerID)
		return err
	}

	accountManagers, err := common.GetCustomerAccountManagers(ctx, customer, "doit")
	if err != nil {
		log.Errorf("GetCustomerAccountManagers error: %v for customer %s", err, customerID)
		return err
	}

	customerRef := s.customersDAL.GetRef(ctx, customerID)

	users, err := common.GetCustomerUsersWithPermissions(ctx, s.Firestore(ctx), customerRef, []string{string(common.PermissionFlexibleRI)})
	if err != nil {
		log.Errorf("GetCustomerUsersWithPermissions error: %v for customer %s", err, customerID)
		return err
	}

	err = s.emailService.SendWelcomeEmail(ctx, &types.WelcomeEmailParams{
		CustomerID:  customerID,
		Cloud:       common.AWS,
		Marketplace: false,
	}, users, accountManagers)
	if err != nil {
		log.Errorf("SendWelcomeEmail error: %v for customer: %s", err, customerID)
		return err
	}

	log.Infof("aws flexsave activation email sent to customer: %v", customerID)

	return nil
}

func publishConfigCreatedSlackNotification(ctx context.Context, customerID string, notificationParams slackNotificationParams, accounts []string) error {
	channel := getAWSFlexsaveSlackChannel()
	fields := []map[string]interface{}{
		{
			"type":  slack.MarkdownType,
			"value": fmt.Sprintf(":tada: *Flexsave AWS* has just been enabled for <https://console.doit.com/customers/%s|%s> \n Hourly Commitment: *$%v*", customerID, notificationParams.Name, notificationParams.HourlyCommitment),
		},
		{
			"type":  slack.MarkdownType,
			"value": fmt.Sprintf("Payer Accounts: %v", accounts),
		},
		{
			"type":  slack.MarkdownType,
			"value": "New Payer Config Created <https://console.doit.com/flexsave-aws-operations|Here>",
		},
	}

	message := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"ts":     time.Now().Unix(),
				"color":  greenColour,
				"fields": fields,
			},
		},
	}

	if _, err := common.PublishToSlack(ctx, message, channel); err != nil {
		return err
	}

	return nil
}

func createFlexsaveTypeActivatedNotification(customerID string, customerName string, accounts []string, flexsaveType utils.FlexsaveType) map[string]interface{} {
	fields := []map[string]interface{}{
		{
			"type":  slack.MarkdownType,
			"value": fmt.Sprintf(":tada: *Flexsave %s* has just been enabled for <https://console.doit.com/customers/%s|%s> \n", flexsaveType.ToTitle(), customerID, customerName),
		},
		{
			"type":  slack.MarkdownType,
			"value": fmt.Sprintf("Payer Accounts: %v", accounts),
		},
	}

	return map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"ts":     time.Now().Unix(),
				"color":  greenColour,
				"fields": fields,
			},
		},
	}
}

func publishFlexsaveTypeActivatedSlackNotification(ctx context.Context, customerID string, customerName string, accounts []string, flexsaveType utils.FlexsaveType) error {
	message := createFlexsaveTypeActivatedNotification(customerID, customerName, accounts, flexsaveType)
	channel := getAWSFlexsaveSlackChannel()

	if _, err := common.PublishToSlack(ctx, message, channel); err != nil {
		return err
	}

	return nil
}

func getAWSFlexsaveSlackChannel() string {
	if common.ProjectID == "me-doit-intl-com" {
		return prodSlackChannel
	}

	return devSlackChannel
}

func (s *flexsaveManageNotify) NotifySharedPayerSavingsDiscrepancies(ctx context.Context, discrepancies monitoringDomain.SharedPayerSavingsDiscrepancies) error {
	if len(discrepancies) == 0 {
		return sendEmptyDiscrepancyNotification(ctx)
	}

	if err := sendDiscrepancyNotification(ctx, discrepancies); err != nil {
		return err
	}

	return nil
}

func sendEmptyDiscrepancyNotification(ctx context.Context) error {
	message := map[string]interface{}{
		"text": ":white_check_mark: No shared payer savings discrepancies detected for the current month.",
	}

	if _, err := common.PublishToSlack(ctx, message, getAWSFlexsaveSlackChannel()); err != nil {
		return err
	}

	return nil
}

func sendDiscrepancyNotification(ctx context.Context, discrepancies monitoringDomain.SharedPayerSavingsDiscrepancies) error {
	var details string
	for _, d := range discrepancies {
		details += fmt.Sprintf("*Customer ID:* %s\n*Last Month's Savings:* $%.2f  \n*This Month's Savings:* $0.0\n", d.CustomerID, d.LastMonthSavings)
	}

	attachment := map[string]interface{}{
		"ts":    time.Now().Unix(),
		"color": redColour,
		"fields": []map[string]interface{}{
			{
				"type":  slack.MarkdownType,
				"value": "*:warning: Shared Payer Savings Discrepancy Alert!* \n \n",
			},
			{
				"type":  slack.MarkdownType,
				"value": details,
			},
		},
	}

	message := map[string]interface{}{
		"attachments": []map[string]interface{}{attachment},
	}

	if _, err := common.PublishToSlack(ctx, message, getAWSFlexsaveSlackChannel()); err != nil {
		return err
	}

	return nil
}
