package saasservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/metadata/domain/metadata"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/contract/domain"
	"github.com/doitintl/hello/scheduled-tasks/dashboard"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/publicdashboards"
	"github.com/doitintl/hello/scheduled-tasks/iam/organizations"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole"
	"github.com/doitintl/hello/scheduled-tasks/saasconsole/saasservice/utils"
	notificationcenter "github.com/doitintl/notificationcenter/pkg"
	"github.com/doitintl/retry"
	tiers "github.com/doitintl/tiers/service"
)

const (
	gcpCourierTemplateID string = "SGYKERHFQXMQ3FKH8KHZXEH27M7Q"
	awsCourierTemplateID string = "SE35GRZBX94NJZQ7KQH7220Q73TK"

	teamSaaSEmail string = "team-saas@doit.com"

	logPrefix string = "SaaS Console - Notify CA data ready - "
)

var platforms = []pkg.StandalonePlatform{pkg.AWS, pkg.GCP}

func (s *SaaSConsoleService) NotifyCloudAnalyticsBillingDataReady(ctx context.Context, dryRun bool) error {
	logger := s.loggerProvider(ctx)

	reports, err := s.getReportsIDs(ctx)
	if err != nil {
		logger.Error(err)
	}

	if len(reports[pkg.AWS]) > 0 {
		aws, err := s.cloudConnectDAL.GetAllAWSNotNotifiedCloudConnect(ctx)
		if err != nil {
			return fmt.Errorf("couldn't fetch all standalone cloud connect docs, %s", err)
		}

		for docID, cc := range aws {
			s.handleAccount(ctx, dryRun, reports[pkg.AWS], docID, cc.Customer.ID, common.Assets.AmazonWebServicesStandalone)
		}
	}

	if len(reports[pkg.GCP]) > 0 {
		gcp, err := s.cloudConnectDAL.GetAllGCPNotNotifiedCloudConnect(ctx)
		if err != nil {
			return fmt.Errorf("couldn't fetch all standalone cloud connect docs, %s", err)
		}

		for docID, cc := range gcp {
			s.handleAccount(ctx, dryRun, reports[pkg.GCP], docID, cc.Customer.ID, common.Assets.GoogleCloudStandalone)
		}
	}

	return nil
}

func (s *SaaSConsoleService) getReportsIDs(ctx context.Context) (map[pkg.StandalonePlatform][]string, error) {
	var errs error

	reports := make(map[pkg.StandalonePlatform][]string)

	for _, p := range platforms {
		var err error

		reports[p], err = s.getDashboardReports(ctx, p)
		if err != nil {
			if errs == nil {
				errs = fmt.Errorf("")
			}

			errs = fmt.Errorf("%s. %scouldn't get %s dashboard doc, %s", errs.Error(), logPrefix, p, err)
		}
	}

	return reports, errs
}

func (s *SaaSConsoleService) getDashboardReports(ctx context.Context, platform pkg.StandalonePlatform) ([]string, error) {
	fs := s.conn.Firestore(ctx)

	docSnap, err := fs.Collection("dashboards").Doc("customization").Collection("public-dashboards").Doc(s.getDashboardID(platform)).Get(ctx)
	if err != nil {
		return nil, err
	}

	var dashboard dashboard.Dashboard
	if err := docSnap.DataTo(&dashboard); err != nil {
		return nil, err
	}

	reports := []string{}

	for _, w := range dashboard.Widgets {
		if strings.HasPrefix(w.Name, "cloudReports::") {
			_, id, found := strings.Cut(w.Name, "_")
			if found {
				reports = append(reports, id)
			}
		}
	}

	return reports, nil
}

func (s *SaaSConsoleService) handleAccount(ctx context.Context, dryRun bool, reports []string, assetID, customerID, cloudConnectType string) {
	logger := s.loggerProvider(ctx)

	platform, accountID := s.getPlatformAndAccountID(assetID, cloudConnectType)

	enabled, err := s.customerEnabledSaaSConsole(ctx, platform, customerID)
	if err != nil {
		logger.Errorf("%scouldn't get customer doc for id: %s, %s", logPrefix, customerID, err)
		return
	} else if !enabled {
		return
	}

	ready := s.customerWidgetDataReady(ctx, customerID, reports)

	if dryRun {
		if ready && s.validatorService.AccountHasLatestBillingData(ctx, platform, customerID, accountID) {
			_ = s.sendNotification(ctx, dryRun, platform, s.customersDAL.GetRef(ctx, customerID))
		}

		return
	}

	if !ready {
		metadataCollection := fmt.Sprintf("%s-%s", cloudConnectType, accountID)

		metadataDocs, err := s.metadataDAL.GetCustomerOrgMetadataCollectionRef(ctx, customerID, metadata.RootOrgID, metadataCollection).Limit(1).Documents(ctx).GetAll()
		if err != nil || len(metadataDocs) < 1 {
			return
		}

		if err := s.widgetService.UpdateCustomerDashboardReportWidgetsHandler(ctx, customerID, organizations.RootOrgID); err != nil {
			return
		}

		if !s.customerWidgetDataReady(ctx, customerID, reports) {
			return
		}
	}

	if !s.validatorService.AccountHasLatestBillingData(ctx, platform, customerID, accountID) {
		return
	}

	customerRef := s.customersDAL.GetRef(ctx, customerID)

	endDate, err := s.startTrial(ctx, customerRef)
	if err != nil {
		logger.Error(err)
		_ = saasconsole.PublishTrialSetErrorSlackNotification(ctx, s.customersDAL, customerID, err)
	}

	if err := s.sendNotificationAndUpdateCloudConnectDoc(ctx, platform, customerRef, accountID); err != nil {
		logger.Error(err)
		_ = saasconsole.PublishBillingReadyErrorSlackNotification(ctx, platform, s.customersDAL, customerID, accountID, err)
	}

	_ = saasconsole.PublishBillingReadySuccessSlackNotification(ctx, platform, s.customersDAL, customerID, accountID, endDate)
}

func (s *SaaSConsoleService) customerWidgetDataReady(ctx context.Context, customerID string, reports []string) bool {
	fs := s.conn.Firestore(ctx)

	for _, reportID := range reports {
		_, err := fs.Collection("cloudAnalytics").
			Doc("widgets").
			Collection("cloudAnalyticsWidgets").
			Doc(customerID + "_" + reportID).
			Get(ctx)
		if err != nil {
			continue
		}

		return true
	}

	return false
}

func (s *SaaSConsoleService) customerEnabledSaaSConsole(ctx context.Context, platform pkg.StandalonePlatform, customerID string) (bool, error) {
	enabled := false

	customer, err := s.customersDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return false, err
	}

	if customer.EnabledSaaSConsole != nil {
		switch platform {
		case pkg.GCP:
			enabled = customer.EnabledSaaSConsole.GCP
		case pkg.AWS:
			enabled = customer.EnabledSaaSConsole.AWS
		default:
		}
	}

	return enabled, nil
}

func (s *SaaSConsoleService) sendNotificationAndUpdateCloudConnectDoc(ctx context.Context, platform pkg.StandalonePlatform, customerRef *firestore.DocumentRef, accountID string) error {
	assetType := utils.GetCloudConnectAssetType(platform)

	if err := retry.BackOffDelay(
		func() error {
			if err := s.cloudConnectDAL.SetCloudConnectNotified(ctx, customerRef, assetType, accountID); err != nil {
				return err
			}
			return nil
		}, 5, time.Second,
	); err != nil {
		return fmt.Errorf("%scould not update cloud connect notified field for %s, account id %s, %s", logPrefix, customerRef.ID, accountID, err)
	}

	if err := s.sendNotification(ctx, false, platform, customerRef); err != nil {
		if err := retry.BackOffDelay(
			func() error {
				if err := s.cloudConnectDAL.SetCloudConnectNotNotified(ctx, customerRef, assetType, accountID); err != nil {
					return err
				}
				return nil
			}, 5, time.Second,
		); err != nil {
			return fmt.Errorf("%scouldn't send %s notification and could not update cloud connect not notified field for %s, account id %s, %s", logPrefix, platform, customerRef.ID, accountID, err)
		}

		return fmt.Errorf("couldn't send %s notification for %s, account id %s, %s", platform, customerRef.ID, accountID, err)
	}

	return nil
}

func (s *SaaSConsoleService) sendNotification(ctx context.Context, dryRun bool, platform pkg.StandalonePlatform, customerRef *firestore.DocumentRef) error {
	emails, err := s.getCustomerUsersEmails(ctx, customerRef)
	if err != nil {
		return err
	}

	if !dryRun {
		templateID := s.getCourierTemplateID(platform)

		notification := notificationcenter.Notification{
			Email:    emails,
			Template: templateID,
			BCC:      []string{teamSaaSEmail},
		}

		_, err = s.notificationClient.Send(ctx, notification)
		if err != nil {
			return err
		}
	}

	s.loggerProvider(ctx).Debugf("%ssent %s notification to %s, users: %v", logPrefix, platform, customerRef.ID, emails)

	return nil
}

func (s *SaaSConsoleService) getCustomerUsersEmails(ctx context.Context, customerRef *firestore.DocumentRef) ([]string, error) {
	users, err := s.userDal.ListUsers(ctx, customerRef, 0)
	if err != nil {
		return nil, err
	}

	emails := []string{}
	emailsMap := make(map[string]bool)

	for _, user := range users {
		if !emailsMap[user.Email] {
			emailsMap[user.Email] = true

			emails = append(emails, user.Email)
		}
	}

	return emails, nil
}

func (s *SaaSConsoleService) getCourierTemplateID(platform pkg.StandalonePlatform) string {
	switch platform {
	case pkg.GCP:
		return gcpCourierTemplateID
	case pkg.AWS:
		return awsCourierTemplateID
	default:
	}

	return ""
}

func (s *SaaSConsoleService) getPlatformAndAccountID(assetID, cloudConnectType string) (pkg.StandalonePlatform, string) {
	switch cloudConnectType {
	case common.Assets.GoogleCloudStandalone:
		return pkg.GCP, strings.TrimPrefix(assetID, common.Assets.GoogleCloud+"-")
	case common.Assets.AmazonWebServicesStandalone:
		return pkg.AWS, strings.TrimPrefix(assetID, common.Assets.AmazonWebServices+"-")
	default:
	}

	return "", ""
}

func (s *SaaSConsoleService) getDashboardID(platform pkg.StandalonePlatform) string {
	dashboardType := publicdashboards.SaaSGcpLens
	if platform == pkg.AWS {
		dashboardType = publicdashboards.SaaSAWSLens
	}

	for _, d := range publicdashboards.DashboardsToAttach {
		if d.DashboardType == dashboardType {
			return d.DashboardID
		}
	}

	return ""
}

func (s *SaaSConsoleService) startTrial(ctx context.Context, customerRef *firestore.DocumentRef) (time.Time, error) {
	if active, latestTrialEndDate, err := s.tiersService.IsCustomerOnActiveTier(
		ctx,
		customerRef,
		[]pkg.PackageTierType{pkg.NavigatorPackageTierType},
	); err != nil || active {
		return latestTrialEndDate, err
	}

	now := time.Now()

	trialLength, err := s.tiersService.GetCustomerTrialCustomLength(ctx, customerRef)
	if err != nil {
		s.loggerProvider(ctx).Errorf("%scouldn't get customer custom trial length for %s, %s", logPrefix, customerRef.ID, err)
	}

	if trialLength < 1 {
		trialLength = tiers.DefaultNavigatorTrialDays
	}

	endDate := now.AddDate(0, 0, trialLength)

	for _, packageType := range []pkg.PackageTierType{pkg.NavigatorPackageTierType, pkg.SolvePackageTierType} {
		trialTierRef, err := s.tiersService.GetTrialTierRef(ctx, packageType)
		if err != nil {
			return time.Time{}, err
		}

		if err = s.contractService.CreateContract(ctx, domain.ContractInputStruct{
			CustomerID: customerRef.ID,
			Tier:       trialTierRef.ID,
			StartDate:  now.Format(time.RFC3339),
			EndDate:    endDate.Format(time.RFC3339),
			Type:       string(packageType),
		}); err != nil {
			return time.Time{}, fmt.Errorf("unable to create %s contract for customer %s: %w", packageType, customerRef.ID, err)
		}
	}

	return endDate, nil
}
