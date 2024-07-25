package validator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/gcp/billingpipeline"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/saasservice/utils"
)

const (
	billingLogPrefix           string = "SaaS Console - Billing Data Validator - "
	billingSlackErrorFormat    string = "%scouldn't send billing data alert slack notification for customer %s, %s"
	firestoreUpdateErrorFormat string = "%scouldn't update cloud connect firestore doc for customer %s, %s"

	awsBillingTimeout    = 24 * time.Hour
	gcpBillingTimeout    = 3 * time.Hour
	gcpHistoryTimeout    = 48 * time.Hour
	minTimeBetweenAlerts = 12 * time.Hour
)

func (s *SaaSConsoleValidatorService) ValidateBillingData(ctx context.Context) error {
	if err := s.validateGCPBilling(ctx); err != nil {
		return err
	}

	if err := s.validateAWSBilling(ctx); err != nil {
		return err
	}

	return nil
}

func (s *SaaSConsoleValidatorService) validateGCPBilling(ctx context.Context) error {
	logger := s.loggerProvider(ctx)

	cloudConnects, err := s.cloudConnectDAL.GetAllGCPStillOnboardingCloudConnect(ctx)
	if err != nil {
		logger.Errorf("%scouldn't fetch all standalone gcp cloud connect docs, %s", billingLogPrefix, err)
	}

	now := time.Now().UTC()
	assetType := utils.GetCloudConnectAssetType(pkg.GCP)

	var wg sync.WaitGroup

	for _, cloudConnect := range cloudConnects {
		wg.Add(1)

		go func(ctx context.Context, cloudConnect *pkg.GCPCloudConnect, assetType string, now time.Time) {
			defer wg.Done()

			timeSinceOnboard := now.Sub(cloudConnect.TimeCreated)

			status, update, err := s.sendGCPAlert(ctx, cloudConnect, now, timeSinceOnboard)
			if err != nil {
				logger.Error(err)
				return
			}

			if update {
				if err := s.cloudConnectDAL.SetCloudConnectBillingStatus(ctx, cloudConnect.Customer, assetType, cloudConnect.BillingAccountID, status); err != nil {
					logger.Error(err)
				}
			}
		}(ctx, cloudConnect, assetType, now)
	}

	wg.Wait()

	return nil
}

func (s *SaaSConsoleValidatorService) sendGCPAlert(ctx context.Context, cloudConnect *pkg.GCPCloudConnect, now time.Time, timeSinceOnboard time.Duration) (*pkg.CloudConnectBillingStatus, bool, error) {
	if timeSinceOnboard < gcpBillingTimeout {
		return nil, false, nil
	}

	platform := pkg.GCP

	pipelineStatus, err := s.billingPipelineService.GetAccountBillingDataStatus(ctx, cloudConnect.Customer.ID, cloudConnect.BillingAccountID)
	if err != nil {
		return nil, false, fmt.Errorf("%scouldn't get account pipeline status for %s, %s", billingLogPrefix, cloudConnect.BillingAccountID, err)
	}

	hoursSinceOnboard := int(timeSinceOnboard / time.Hour)

	update := false
	status := &pkg.CloudConnectBillingStatus{
		AlertSentAt:     now,
		OnboardFinished: cloudConnect.BillingStatus.OnboardFinished,
	}

	if pipelineStatus == billingpipeline.AccountDataStatusImportFinished || pipelineStatus == billingpipeline.AccountDataStatusHistoryFinished {
		if !cloudConnect.Notified { // alert customer not notified
			if s.shouldSendAlert(&cloudConnect.CloudConnect, now, pkg.CloudConnectStatusNotNotified) {
				if err = saasconsole.PublishBillingDataAlertSlackNotification(ctx, platform, s.customersDAL, cloudConnect.Customer.ID, cloudConnect.BillingAccountID, saasconsole.NotificationIssueMessage, hoursSinceOnboard); err != nil {
					return nil, false, fmt.Errorf(billingSlackErrorFormat, billingLogPrefix, cloudConnect.Customer.ID, err)
				}

				update = true
				status.AlertSentStatus = pkg.CloudConnectStatusNotNotified
			}
		} else if pipelineStatus == billingpipeline.AccountDataStatusImportFinished { // alert history is not fully imported
			if cloudConnect.BillingStatus != nil && !cloudConnect.BillingStatus.OnboardFinished &&
				timeSinceOnboard > gcpHistoryTimeout && s.shouldSendAlert(&cloudConnect.CloudConnect, now, pkg.CloudConnectStatusMissingHistoryBillingData) {
				if err = saasconsole.PublishBillingDataAlertSlackNotification(ctx, platform, s.customersDAL, cloudConnect.Customer.ID, cloudConnect.BillingAccountID, saasconsole.HistoryIssueMessage, hoursSinceOnboard); err != nil {
					return nil, false, fmt.Errorf(billingSlackErrorFormat, billingLogPrefix, cloudConnect.Customer.ID, err)
				}

				update = true
				status.AlertSentStatus = pkg.CloudConnectStatusMissingHistoryBillingData
			}
		} else { // notified && pipelineStatus == billingpipeline.AccountDataStatusHistoryFinished
			update = true
			status.OnboardFinished = true
		}
	} else if pipelineStatus == billingpipeline.AccountDataStatusImportRunning { // alert initial import is not finished
		if s.shouldSendAlert(&cloudConnect.CloudConnect, now, pkg.CloudConnectStatusMissingBillingData) {
			if err = saasconsole.PublishBillingDataAlertSlackNotification(ctx, platform, s.customersDAL, cloudConnect.Customer.ID, cloudConnect.BillingAccountID, saasconsole.DataImportIssueMessage, hoursSinceOnboard); err != nil {
				return nil, false, fmt.Errorf(billingSlackErrorFormat, billingLogPrefix, cloudConnect.Customer.ID, err)
			}

			update = true
			status.AlertSentStatus = pkg.CloudConnectStatusMissingBillingData
		}
	}

	return status, update, nil
}

func (s *SaaSConsoleValidatorService) validateAWSBilling(ctx context.Context) error {
	logger := s.loggerProvider(ctx)

	cloudConnects, err := s.cloudConnectDAL.GetAllAWSNotNotifiedCloudConnect(ctx)
	if err != nil {
		logger.Errorf("%scouldn't fetch all standalone aws cloud connect docs, %s", billingLogPrefix, err)
	}

	now := time.Now().UTC()

	platform := pkg.AWS
	assetType := utils.GetCloudConnectAssetType(platform)

	var wg sync.WaitGroup

	for _, cloudConnect := range cloudConnects {
		wg.Add(1)

		go func(ctx context.Context, cloudConnect *pkg.AWSCloudConnect, assetType string, now time.Time) {
			defer wg.Done()

			timeSinceOnboard := now.Sub(cloudConnect.TimeCreated)

			status, err := s.sendAWSAlert(ctx, cloudConnect, now, timeSinceOnboard)
			if err != nil {
				logger.Error(err)
				return
			}

			if status != nil {
				if err := s.cloudConnectDAL.SetCloudConnectBillingStatus(ctx, cloudConnect.Customer, assetType, cloudConnect.AccountID, status); err != nil {
					logger.Error(err)
				}
			}
		}(ctx, cloudConnect, assetType, now)
	}

	wg.Wait()

	return nil
}

func (s *SaaSConsoleValidatorService) sendAWSAlert(ctx context.Context, cloudConnect *pkg.AWSCloudConnect, now time.Time, timeSinceOnboard time.Duration) (*pkg.CloudConnectBillingStatus, error) {
	if timeSinceOnboard < awsBillingTimeout {
		return nil, nil
	}

	platform := pkg.AWS

	hoursSinceOnboard := int(timeSinceOnboard / time.Hour)

	status := &pkg.CloudConnectBillingStatus{
		AlertSentAt:     now,
		OnboardFinished: true,
	}

	if len(cloudConnect.BillingEtl.ManifestFileHistory) == 0 {
		if s.shouldSendAlert(&cloudConnect.CloudConnect, now, pkg.CloudConnectStatusMissingBillingData) {
			if err := saasconsole.PublishBillingDataAlertSlackNotification(ctx, platform, s.customersDAL, cloudConnect.Customer.ID, cloudConnect.AccountID, saasconsole.DataImportIssueMessage, hoursSinceOnboard); err != nil {
				return nil, fmt.Errorf(billingSlackErrorFormat, billingLogPrefix, cloudConnect.Customer.ID, err)
			}

			status.AlertSentStatus = pkg.CloudConnectStatusMissingBillingData
		}
	} else if s.shouldSendAlert(&cloudConnect.CloudConnect, now, pkg.CloudConnectStatusNotNotified) {
		if err := saasconsole.PublishBillingDataAlertSlackNotification(ctx, platform, s.customersDAL, cloudConnect.Customer.ID, cloudConnect.AccountID, saasconsole.NotificationIssueMessage, hoursSinceOnboard); err != nil {
			return nil, fmt.Errorf(billingSlackErrorFormat, billingLogPrefix, cloudConnect.Customer.ID, err)
		}

		status.AlertSentStatus = pkg.CloudConnectStatusNotNotified
	}

	return status, nil
}

func (s *SaaSConsoleValidatorService) shouldSendAlert(cloudConnect *pkg.CloudConnect, now time.Time, alertType string) bool {
	if cloudConnect.BillingStatus == nil ||
		cloudConnect.BillingStatus.AlertSentStatus != alertType ||
		now.Sub(cloudConnect.BillingStatus.AlertSentAt) > minTimeBetweenAlerts {
		return true
	}

	return false
}
