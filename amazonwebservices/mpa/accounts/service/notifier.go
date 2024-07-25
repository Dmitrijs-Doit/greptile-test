package service

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	dal2 "github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/service/iface"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	devAWSProjectID                 = "cmp-aws-etl-dev"
	prodAWSProjectID                = "doitintl-cmp-aws-data"
	devSlackChannel                 = "#test-aws-fs-ops-notifications"
	prodSlackChannel                = "#fsaws-ops"
	devNotificationEventsProjectID  = "doitintl-cmp-dev"
	prodNotificationEventsProjectID = "me-doit-intl-com"
	sharedPayerThreshold            = 5.0
	dedicatedPayerThreshold         = 2.0
	dontNotify                      = math.MaxFloat64
)

type NotifierService struct {
	billing                                   iface.Billing
	masterPayerAccounts                       dal2.MasterPayerAccounts
	publisher                                 iface.NotificationPublisher
	loggerProvider                            logger.Provider
	notificationEventsProjectID, slackChannel string
}

func NewNotifierService(log logger.Provider) *NotifierService {
	projectID := devAWSProjectID
	notificationEventsProjectID := devNotificationEventsProjectID
	slackChannel := devSlackChannel

	if common.Production {
		projectID = prodAWSProjectID
		notificationEventsProjectID = prodNotificationEventsProjectID
		slackChannel = prodSlackChannel
	}

	billing, err := NewBillingService(projectID)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()

	masterPayerAccounts, err := dal2.NewMasterPayerAccountDAL(ctx, common.ProjectID)
	if err != nil {
		panic(err)
	}

	return &NotifierService{
		billing:                     billing,
		masterPayerAccounts:         masterPayerAccounts,
		notificationEventsProjectID: notificationEventsProjectID,
		slackChannel:                slackChannel,
		loggerProvider:              log,
	}
}

func (s *NotifierService) NotifyIfNecessary(ctx context.Context, move iface.AccountMove, eventType iface.EventType) error {
	threshold, err := s.notificationThreshold(ctx, move)
	if err != nil {
		return err
	}

	if threshold == dontNotify {
		return nil
	}

	cu, err := s.billing.GetCoveredUsage(ctx, move.AccountID, move.FromPayer)
	if err != nil {
		return err
	}

	log := s.loggerProvider(ctx)
	usage := cu.SPCost + cu.RICost

	if usage < threshold {
		log.Infof(
			"Account %s (%s) covered usage %d was less than threshold %f. Slack notification skipped.",
			move.AccountID,
			move.AccountName,
			usage,
			threshold)

		return nil
	}

	return s.notify(ctx, move, cu.SPCost, cu.RICost, eventType)
}

func (s *NotifierService) notificationThreshold(ctx context.Context, move iface.AccountMove) (float64, error) {
	log := s.loggerProvider(ctx)
	if move.FromPayer.ID == move.ToPayer.ID {
		log.Infof("Payer account is the same")
		return dontNotify, nil
	}

	if strings.HasPrefix(move.AccountName, "fs") {
		log.Infof("DoiT owner account")
		return dontNotify, nil
	}

	account, err := s.masterPayerAccounts.GetMasterPayerAccount(ctx, move.FromPayer.ID)
	if err != nil {
		if err == dal2.ErrorNotFound {
			log.Infof("Master payer account doesn't exist")
			return dontNotify, nil
		}

		return dontNotify, err
	}

	threshold := dedicatedPayerThreshold
	if account.TenancyType == dal2.SharedTenancy {
		threshold = sharedPayerThreshold
	}

	return threshold, nil
}

func (s *NotifierService) notify(
	ctx context.Context,
	move iface.AccountMove,
	spCost, riCost float64,
	eventType iface.EventType,
) error {
	var msg map[string]interface{}

	switch eventType {
	case iface.MovedAccount:
		msg = movedAccountMessage(move, spCost, riCost)
	case iface.LeftAccount:
		msg = deletedAccountMessage(move, spCost, riCost)
	default:
		return fmt.Errorf("unknown eventType %v", eventType)
	}

	publisher, err := s.createPubSubSession(ctx)
	if err != nil {
		return err
	}

	log := s.loggerProvider(ctx)
	log.Infof("Sending slack notification %+v.", msg)

	if err = publisher.PublishSlackNotification(ctx, msg); err != nil {
		return err
	}

	log.Infof("%s notification %+v was sent to the slack channel %s.", eventType, move, s.slackChannel)

	return nil
}

// creates pubsub session with correct credentials in case of missing permissions (eg. misconfigured slack channel)
// protects scheduled-tasks initialisation from crashing
func (s *NotifierService) createPubSubSession(ctx context.Context) (iface.NotificationPublisher, error) {
	if s.publisher == nil {
		pub, err := dal2.NewPubSubEventsDAL(ctx, s.notificationEventsProjectID, s.slackChannel)
		if err != nil {
			return nil, err
		}

		s.publisher = pub
	}

	return s.publisher, nil
}

func movedAccountMessage(move iface.AccountMove, spCoveredUsage, riDiscountedUsage float64) map[string]interface{} {
	fields := []map[string]interface{}{
		{
			"title": "From Payer",
			"value": fmt.Sprintf("%s (%s)", move.FromPayer.ID, move.FromPayer.DisplayName),
			"short": true,
		},
		{
			"title": "To Payer",
			"value": fmt.Sprintf("%s (%s)", move.ToPayer.ID, move.ToPayer.DisplayName),
			"short": true,
		},
		{
			"title": "Savings Plans Covered Usage",
			"value": spCoveredUsage,
			"short": true,
		},
		{
			"title": "Reserved Instances Discounted Usage",
			"value": riDiscountedUsage,
			"short": true,
		},
	}

	return map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"ts":        time.Now().Unix(),
				"color":     "#4CAF50",
				"title":     fmt.Sprintf("AWS account %s (%s) moved", move.AccountID, move.AccountName),
				"thumb_url": "https://storage.googleapis.com/hello-static-assets/logos/amazon-web-services.png",
				"fields":    fields,
			},
		},
	}
}

func deletedAccountMessage(move iface.AccountMove, spCoveredUsage, riCost float64) map[string]interface{} {
	fields := []map[string]interface{}{
		{
			"value": fmt.Sprintf("Payer ID: %s", move.FromPayer.ID),
		},
		{
			"value": fmt.Sprintf("Display name: %s", move.FromPayer.DisplayName),
		},
		{
			"title": "Savings Plans Covered Usage",
			"value": spCoveredUsage,
			"short": true,
		},
		{
			"title": "Reserved Instances Discounted Usage",
			"value": riCost,
			"short": true,
		},
	}

	return map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"ts":     time.Now().Unix(),
				"color":  "#F44336",
				"title":  fmt.Sprintf("Customer %s (%s) has left", move.AccountName, move.AccountID),
				"fields": fields,
			},
		},
	}
}
