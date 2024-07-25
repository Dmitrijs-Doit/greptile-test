package saasconsole

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/doitintl/firestore/pkg"
	customerDal "github.com/doitintl/hello/scheduled-tasks/customer/dal"
	"github.com/doitintl/hello/scheduled-tasks/slice"
	"github.com/slack-go/slack"
	"golang.org/x/exp/slices"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

type msgStatus string

const (
	errorMsgStatus   msgStatus = "error"
	warningMsgStatus msgStatus = "warning"
	successMsgStatus msgStatus = "success"
)

const (
	prodSlackChannel          = "C05S1V7KCHX"
	prodValidatorSlackChannel = "C0677JCPN69"

	devSlackChannel = "C05R60AN73M"

	DataImportIssueMessage   = "billing data is missing or not fully imported"
	NotificationIssueMessage = "billing data was imported, customer wasn't notified"
	HistoryIssueMessage      = "history data is not fully imported"
)

var testCustomerIDs = []string{"xKndpYwFjHVSIVdW8Wzt", "2MUUrfODqshAoChByDEL"}

func getOnboardSlackChannel(customerID string) string {
	if shouldUseProductionSlackChannel(customerID) {
		return prodSlackChannel
	}

	return devSlackChannel
}

func getValidatorSlackChannel(customerID string) string {
	if shouldUseProductionSlackChannel(customerID) {
		return prodValidatorSlackChannel
	}

	return devSlackChannel
}

func shouldUseProductionSlackChannel(customerID string) bool {
	return common.Production && !slices.Contains(testCustomerIDs, customerID)
}

func getCustomerNameWithLink(ctx context.Context, customersDAL customerDal.Customers, customerID string) string {
	var customerName string
	if customer, err := customersDAL.GetCustomer(ctx, customerID); err != nil {
		customerName = customerID
	} else {
		customerName = customer.Name
	}

	return fmt.Sprintf("<https://console.doit.com/customers/%s|%s>", customerID, customerName)
}

func PublishOnboardSuccessSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL customerDal.Customers, customerID, accountID string) error {
	customerLink := getCustomerNameWithLink(ctx, customersDAL, customerID)

	fields := []map[string]interface{}{
		{
			"type":  slack.MarkdownType,
			"value": fmt.Sprintf(":tada: *%s SaaS Console*", platform),
		},
		{
			"type": slack.MBTDivider,
		},
		{
			"type":  slack.MarkdownType,
			"value": fmt.Sprintf("*%s* has just onboarded with Account ID: *%s*", customerLink, accountID),
		},
		{
			"type":  slack.PlainTextType,
			"value": "The billing data is uploading...",
		},
	}

	return publishSlackNotification(ctx, getOnboardSlackChannel(customerID), fields, getMessageColor(successMsgStatus))
}

func PublishBillingReadySuccessSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL customerDal.Customers, customerID, accountID string, trialEndDate time.Time) error {
	customerLink := getCustomerNameWithLink(ctx, customersDAL, customerID)

	fields := []map[string]interface{}{
		{
			"type":  slack.MarkdownType,
			"value": fmt.Sprintf(":tada: *%s SaaS Console*", platform),
		},
		{
			"type": slack.MBTDivider,
		},
		{
			"type":  slack.MarkdownType,
			"value": fmt.Sprintf("*%s* billing data is ready for Account ID: *%s*", customerLink, accountID),
		},
		{
			"type":  slack.PlainTextType,
			"value": "Customer was notified.",
		},
	}

	if !trialEndDate.IsZero() {
		fields = append(fields, map[string]interface{}{
			"type":  slack.PlainTextType,
			"value": fmt.Sprintf("Trial started, end date: %s.", trialEndDate.Format("January 02, 2006")),
		})
	}

	return publishSlackNotification(ctx, getOnboardSlackChannel(customerID), fields, getMessageColor(successMsgStatus))
}

func PublishOnboardErrorSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL customerDal.Customers, customerID, accountID string, originalError error) error {
	customerLink := getCustomerNameWithLink(ctx, customersDAL, customerID)

	fields := []map[string]interface{}{
		{
			"type":  slack.MarkdownType,
			"value": fmt.Sprintf(":x: *%s SaaS Console*", platform),
		},
		{
			"type": slack.MBTDivider,
		},
		{
			"type":  slack.MarkdownType,
			"value": fmt.Sprintf("*%s* has just failed onboarding with account %s.", customerLink, accountID),
		},
		{
			"type":  slack.PlainTextType,
			"value": originalError.Error(),
		},
	}

	return publishSlackNotification(ctx, getOnboardSlackChannel(customerID), fields, getMessageColor(errorMsgStatus))
}

func PublishBillingReadyErrorSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL customerDal.Customers, customerID, accountID string, originalError error) error {
	customerLink := getCustomerNameWithLink(ctx, customersDAL, customerID)

	fields := []map[string]interface{}{
		{
			"type":  slack.MarkdownType,
			"value": fmt.Sprintf(":x: *%s SaaS Console*", platform),
		},
		{
			"type": slack.MBTDivider,
		},
		{
			"type":  slack.MarkdownType,
			"value": fmt.Sprintf("*%s* billing data is ready for Account ID: *%s*", customerLink, accountID),
		},
		{
			"type":  slack.PlainTextType,
			"value": "We had a problem notifying a customer or updating the firestore",
		},
		{
			"type":  slack.PlainTextType,
			"value": originalError.Error(),
		},
	}

	return publishSlackNotification(ctx, getOnboardSlackChannel(customerID), fields, getMessageColor(errorMsgStatus))
}

func PublishTrialSetErrorSlackNotification(ctx context.Context, customersDAL customerDal.Customers, customerID string, originalError error) error {
	customerLink := getCustomerNameWithLink(ctx, customersDAL, customerID)

	fields := []map[string]interface{}{
		{
			"type":  slack.MarkdownType,
			"value": ":x: *Trial set*",
		},
		{
			"type": slack.MBTDivider,
		},
		{
			"type":  slack.MarkdownType,
			"value": fmt.Sprintf("*%s* setting trial failed.", customerLink),
		},
		{
			"type":  slack.PlainTextType,
			"value": originalError.Error(),
		},
	}

	return publishSlackNotification(ctx, getOnboardSlackChannel(customerID), fields, getMessageColor(errorMsgStatus))
}

func PublishBillingDataAlertSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL customerDal.Customers, customerID, accountID, issueMsd string, hours int) error {
	customerLink := getCustomerNameWithLink(ctx, customersDAL, customerID)

	msgStatus := errorMsgStatus
	msgIcon := ":x:"

	if issueMsd == NotificationIssueMessage {
		msgStatus = warningMsgStatus
		msgIcon = ":face_with_diagonal_mouth:"
	}

	fields := []map[string]interface{}{
		{
			"type":  slack.MarkdownType,
			"value": fmt.Sprintf("%s *%s SaaS Console*", msgIcon, platform),
		},
		{
			"type": slack.MBTDivider,
		},
		{
			"type":  slack.MarkdownType,
			"value": fmt.Sprintf("*%s* - *%s*", customerLink, accountID),
		},
		{
			"type":  slack.PlainTextType,
			"value": fmt.Sprintf("%d hours since onboarding...", hours),
		},
		{
			"type":  slack.PlainTextType,
			"value": issueMsd,
		},
	}

	return publishSlackNotification(ctx, getValidatorSlackChannel(customerID), fields, getMessageColor(msgStatus))
}

func PublishPermissionsAlertSlackNotification(ctx context.Context, platform pkg.StandalonePlatform, customersDAL customerDal.Customers, customerID, accountID string, status *pkg.AWSCloudConnectStatus, criticalPermissions []string) error {
	customerLink := getCustomerNameWithLink(ctx, customersDAL, customerID)

	if status == nil || status.InvalidInfo == nil {
		return nil
	}

	msgStatus := errorMsgStatus
	msgIcon := ":x:"

	if status.Status != pkg.AWSCloudConnectStatusCritical {
		msgStatus = warningMsgStatus
		msgIcon = ":face_with_diagonal_mouth:"
	}

	fields := []map[string]interface{}{
		{
			"type":  slack.MarkdownType,
			"value": fmt.Sprintf("%s *%s SaaS Console*", msgIcon, platform),
		},
		{
			"type": slack.MBTDivider,
		},
		{
			"type":  slack.MarkdownType,
			"value": fmt.Sprintf("*%s* - *%s*", customerLink, accountID),
		},
	}

	if status.InvalidInfo.CURError != "" {
		fields = append(fields,
			[]map[string]interface{}{
				{
					"type":  slack.PlainTextType,
					"value": "*Report error:*",
				},
				{
					"type":  slack.PlainTextType,
					"value": fmt.Sprintf("`%s`", status.InvalidInfo.CURError),
				},
			}...,
		)
	}

	if status.InvalidInfo.RoleError != "" {
		fields = append(fields, []map[string]interface{}{
			{
				"type":  slack.PlainTextType,
				"value": "*Role error:*",
			},
			{
				"type":  slack.PlainTextType,
				"value": fmt.Sprintf("`%s`", status.InvalidInfo.RoleError),
			},
		}...,
		)
	}

	if status.InvalidInfo.PolicyError != "" {
		fields = append(fields, []map[string]interface{}{
			{
				"type":  slack.PlainTextType,
				"value": "*Policy error:*",
			},
			{
				"type":  slack.PlainTextType,
				"value": fmt.Sprintf("`%s`", status.InvalidInfo.PolicyError),
			},
		}...,
		)
	}

	if len(status.InvalidInfo.MissingPermissions) > 0 {
		fields = append(fields, map[string]interface{}{
			"type":  slack.PlainTextType,
			"value": "*The following permissions are missing:*",
		})

		for _, permission := range status.InvalidInfo.MissingPermissions {
			pArr := strings.Split(permission, "~")
			pStr := "    "

			if slice.Contains(criticalPermissions, pArr[0]) {
				pStr = "CRITICAL - "
			}

			resourceORPrincipal := ""
			if len(pArr) > 1 {
				resourceORPrincipal = fmt.Sprintf("Resource: `%s`", pArr[1])
			}

			if len(pArr) > 2 {
				resourceORPrincipal = fmt.Sprintf("Principal: `%s`", pArr[2])
			}

			fields = append(fields, map[string]interface{}{
				"type":  slack.PlainTextType,
				"value": fmt.Sprintf("%sAction: `%s` on %s", pStr, pArr[0], resourceORPrincipal),
			})
		}
	}

	return publishSlackNotification(ctx, getValidatorSlackChannel(customerID), fields, getMessageColor(msgStatus))
}

func PublishDeactivatedBillingSlackNotification(ctx context.Context, customersDAL customerDal.Customers, customersAccounts map[string]map[pkg.StandalonePlatform][]string) error {
	var errs []error

	for customerID, billingAccountIDs := range customersAccounts {
		customerLink := getCustomerNameWithLink(ctx, customersDAL, customerID)

		fields := []map[string]interface{}{
			{
				"type":  slack.MarkdownType,
				"value": ":woman-gesturing-no: *SaaS Console - Trial ended*",
			},
			{
				"type": slack.MBTDivider,
			},
			{
				"type":  slack.MarkdownType,
				"value": fmt.Sprintf("*%s* Customer billing import stopped for accounts:", customerLink),
			},
		}

		for platform, billingAccountIDs := range billingAccountIDs {
			fields = append(fields, map[string]interface{}{
				"type":  slack.MarkdownType,
				"value": fmt.Sprintf("*%s - %s*", platform, strings.Join(billingAccountIDs, ", ")),
			})
		}

		if err := publishSlackNotification(ctx, getOnboardSlackChannel(customerID), fields, getMessageColor(warningMsgStatus)); err != nil {
			errs = append(errs, fmt.Errorf("failed for customer %s: %w", customerID, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("failed to publish deactivate billing slack notifications: %w", errors.Join(errs...))
	}

	return nil
}

func publishSlackNotification(ctx context.Context, channel string, fields []map[string]interface{}, color string) error {
	message := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"ts":     time.Now().Unix(),
				"color":  color,
				"fields": fields,
			},
		},
	}

	if _, err := common.PublishToSlack(ctx, message, channel); err != nil {
		return err
	}

	return nil
}

func getMessageColor(status msgStatus) string {
	switch status {
	case errorMsgStatus:
		return "#db1414"
	case successMsgStatus:
		return "#4CAF50"
	case warningMsgStatus:
		return "#FF9800"
	default:
	}

	return ""
}
