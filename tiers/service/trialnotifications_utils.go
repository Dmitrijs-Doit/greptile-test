package service

import (
	"context"
	"fmt"
	"slices"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/firestore/pkg"
	budgetsDal "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/budgets/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/tiers/dal"
	"github.com/doitintl/retry"
	tiersDal "github.com/doitintl/tiers/dal"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	loggerPrefix = "TRIAL NOTIFICATIONS: "

	procurementOnly = "procurement-only"

	SMETeamEmail = "team-navigator@doit.com"

	defaultTrialPeriod = 45
)

type NotificationreferencePoint string

const (
	Start NotificationreferencePoint = "start"
	End   NotificationreferencePoint = "end"
)

type trialNotification struct {
	id                 string
	description        string
	reference          NotificationreferencePoint
	daysOffset         int
	maxAge             int
	courierTemplateID  string
	courierTemplateMap map[string]string
	usageVerifier      usageVerifier
}

type usageVerifier interface {
	hasUsage(ctx context.Context, s *TiersService, customerRef *firestore.DocumentRef) (bool, error)
}

type hasAttributions struct{}

func (v *hasAttributions) hasUsage(ctx context.Context, s *TiersService, customerRef *firestore.DocumentRef) (bool, error) {
	return s.attributionsDal.CustomerHasCustomAttributions(ctx, customerRef)
}

type hasAlertsOrBudgets struct{}

func (v *hasAlertsOrBudgets) hasUsage(ctx context.Context, s *TiersService, customerRef *firestore.DocumentRef) (bool, error) {
	if hasAttributions, err := s.attributionsDal.CustomerHasCustomAttributions(ctx, customerRef); err != nil || !hasAttributions {
		return true, err
	}

	if alerts, err := s.alertsDal.GetAllAlertsByCustomer(ctx, customerRef); err != nil || len(alerts) > 0 {
		return true, err
	}

	if budgets, err := s.budgetsDal.ListBudgets(ctx, &budgetsDal.ListBudgetsArgs{
		CustomerID:     customerRef.ID,
		IsDoitEmployee: true,
	}); err != nil || len(budgets) > 0 {
		return true, err
	}

	return false, nil
}

type startEndDates struct {
	start *time.Time
	end   *time.Time
}

var activeTrialNotifications = map[pkg.PackageTierType][]trialNotification{
	pkg.NavigatorPackageTierType: {
		{
			description:       "15 day into trial",
			reference:         Start,
			daysOffset:        15,
			maxAge:            2,
			courierTemplateID: "68GSY7R65DM8QCNTT4WZJB17VZ9X",
		},
		{
			description:       "week before end of trial",
			reference:         End,
			daysOffset:        -7,
			maxAge:            2,
			courierTemplateID: "R9H11NG1G4MFSYQR24M41HSYV06S",
		},
		{
			description: "end of trial",
			reference:   End,
			daysOffset:  0,
			maxAge:      2,
			courierTemplateMap: map[string]string{
				tiersDal.HeritageResoldTierName:   "2AK2J2H3C8MGK2QDVV1RATBZWZYN",
				tiersDal.ZeroEntitlementsTierName: "P7XP17TZ294E03JE208CBGHCPBMK",
				tiersDal.TrialTierName:            "P7XP17TZ294E03JE208CBGHCPBMK",
				procurementOnly:                   "DHQ818AJDX4959K1MC23D7F88DW8",
			},
		},
	},
}

var inactiveCustomerNotifications = map[pkg.PackageTierType][]trialNotification{
	pkg.NavigatorPackageTierType: {
		{
			description:       "2 weeks inactive",
			reference:         Start,
			daysOffset:        14,
			maxAge:            2,
			courierTemplateID: "7V604FX31SMVSEN6H6817530FT6D",
		},
		{
			description:       "4 weeks inactive",
			reference:         Start,
			daysOffset:        28,
			maxAge:            14,
			courierTemplateID: "HVT72SC15VM55TQT606CN98AV6Y4",
		},
	},
}

var usageNotifications = map[pkg.PackageTierType][]trialNotification{
	pkg.NavigatorPackageTierType: {
		{
			id:                "noAttributions",
			description:       "no attributions created 10 days into trial",
			reference:         Start,
			daysOffset:        10,
			maxAge:            -1,
			courierTemplateID: "CVH37NHQY24JZ1MG9QJJW459PM5J",
			usageVerifier:     &hasAttributions{},
		},
		{
			id:                "noAlertsOrBudgets",
			description:       "no alerts or budgets created 10 days into trial",
			reference:         Start,
			daysOffset:        10,
			maxAge:            -1,
			courierTemplateID: "65HZBS8YD6M6HTHKH7FE165XD52J",
			usageVerifier:     &hasAlertsOrBudgets{},
		},
	},
}

func (s *TiersService) getLatestRelevantNotification(dates *startEndDates, notifications []trialNotification, now time.Time, lastDate time.Time) int {
	idx := -1

	for i, n := range notifications {
		if s.isRelevantNotification(n, dates, now, lastDate, n.maxAge) {
			idx = i
		}
	}

	return idx
}

func (s *TiersService) isRelevantNotification(notification trialNotification, dates *startEndDates, now, lastDate time.Time, maxAge int) bool {
	referenceDate := dates.start
	if notification.reference == End {
		referenceDate = dates.end
	}

	if referenceDate == nil {
		return false
	}

	date := referenceDate.UTC().AddDate(0, 0, notification.daysOffset)

	if date.After(now) {
		return false
	}

	// check if notification was already sent
	if !lastDate.IsZero() && date.Before(lastDate) {
		return false
	}

	// don't send more than daysOld old notifications
	if maxAge >= 0 && date.Before(now.AddDate(0, 0, -1*maxAge)) {
		return false
	}

	return true
}

func (s *TiersService) getCustomerNotificationData(ctx context.Context, customerRef *firestore.DocumentRef, customer *customerData, dryRun bool) (*notificationsData, error) {
	lastNotifications, err := s.trialNotificationsDal.GetCustomerTrialNotifications(ctx, customerRef.ID)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			return nil, fmt.Errorf("%sfailed to read customer trial notification for customer %s\n Error: %v", loggerPrefix, customerRef.ID, err)
		}

		lastNotifications = &dal.CustomerTrialNotifications{
			LastSent:  make(map[string]time.Time),
			UsageSent: make(map[string][]string),
		}
	} else if lastNotifications.LastSent == nil {
		lastNotifications.LastSent = make(map[string]time.Time)
	} else if lastNotifications.UsageSent == nil {
		lastNotifications.UsageSent = make(map[string][]string)
	}

	users, err := s.usersDal.GetCustomerUsersByRoles(ctx, customerRef.ID, userRoleFilter)
	if err != nil {
		return nil, fmt.Errorf("%sfailed to get customer Admin users for customer %s\n Error: %v", loggerPrefix, customerRef.ID, err)
	}

	if len(users) == 0 {
		return nil, nil
	}

	data := &notificationsData{
		customerRef:       customerRef,
		customerName:      customer.Name,
		users:             users,
		lastNotifications: lastNotifications,
		dryRun:            dryRun,
	}

	return data, nil
}

func (s *TiersService) getTrialNotificationsCustomers(ctx context.Context) (map[string][]*firestore.DocumentSnapshot, error) {
	customers := make(map[string][]*firestore.DocumentSnapshot)

	for _, tierName := range []string{tiersDal.TrialTierName, tiersDal.ZeroEntitlementsTierName} {
		c, err := s.getCustomersByTierName(ctx, tierName)
		if err != nil {
			return nil, err
		}

		customers[tierName] = c
	}

	for _, tierName := range s.tiersSvc.GetHeritageTierNames() {
		c, err := s.getCustomersByTierName(ctx, tierName)
		if err != nil {
			return nil, err
		}

		customers[tiersDal.HeritageResoldTierName] = c
	}

	return customers, nil
}

func (s *TiersService) getInactiveCustomers(ctx context.Context) ([]*firestore.DocumentSnapshot, error) {
	customers := []*firestore.DocumentSnapshot{}

	for _, tierName := range []string{tiersDal.PresentationTierName} {
		c, err := s.getCustomersByTierName(ctx, tierName)
		if err != nil {
			return nil, err
		}

		customers = append(customers, c...)
	}

	return customers, nil
}

func (s *TiersService) getCustomersByTierName(ctx context.Context, tierName string) ([]*firestore.DocumentSnapshot, error) {
	tierRef, err := s.tiersSvc.GetTierRefByName(ctx, tierName, pkg.NavigatorPackageTierType)
	if err != nil {
		return nil, err
	}

	customers, err := s.customerDal.GetCustomersByTier(ctx, tierRef, pkg.NavigatorPackageTierType)
	if err != nil {
		return nil, err
	}

	return customers, nil
}

func (s *TiersService) getFirstUserTimestamp(ctx context.Context, users []*common.User) time.Time {
	fs := s.Firestore(ctx)

	firstUserTimestamp := time.Time{}

	for _, user := range users {
		inviteSnap, err := fs.Collection("invites").Where("email", "==", user.Email).Documents(ctx).Next()
		if err != nil {
			continue
		}

		var invite struct {
			Timestamp time.Time `firestore:"timestamp"`
		}

		if err := inviteSnap.DataTo(&invite); err != nil {
			continue
		}

		if firstUserTimestamp.IsZero() || invite.Timestamp.Before(firstUserTimestamp) {
			firstUserTimestamp = invite.Timestamp
		}
	}

	return firstUserTimestamp
}

func (s *TiersService) addAMsAndSMETeam(ctx context.Context, d *notificationsData) error {
	if d.amsAdded {
		return nil
	}

	d.amsAdded = true

	d.users = append(d.users, &common.User{
		Email:     SMETeamEmail,
		FirstName: d.customerName,
	})

	ams, err := s.customerDal.GetCustomerAccountTeam(ctx, d.customerRef.ID)
	if err != nil {
		return fmt.Errorf("%sfailed to get customer account team for customer %s\n Error: %v", loggerPrefix, d.customerRef.ID, err)
	}

	for _, am := range ams {
		if am.Role != common.AccountManagerRoleFSR {
			continue
		}

		d.users = append(d.users, &common.User{
			Email:     am.Email,
			FirstName: d.customerName,
		})
	}

	return nil
}

func (s *TiersService) updateCustomerNotificationsDoc(ctx context.Context, customerID string, lastNotifications *dal.CustomerTrialNotifications, dryRun bool) {
	logger := s.loggerProvider(ctx)

	if dryRun {
		logger.Debugf("%sDry run: setting customer trial notification for customer %s with %+v", loggerPrefix, customerID, lastNotifications)
		return
	}

	if err := retry.BackOffDelay(
		func() error {
			return s.trialNotificationsDal.SetCustomerTrialNotification(ctx, customerID, lastNotifications)
		}, 5, time.Second,
	); err != nil {
		logger.Errorf("%sfailed to set customer trial notification for customer %s\n Error: %v", loggerPrefix, customerID, err)
	}
}

func (s *TiersService) getCourierTemplateID(ctx context.Context, customerRef *firestore.DocumentRef, notification trialNotification, tierName string) string {
	if tierName == tiersDal.ZeroEntitlementsTierName {
		assetTypes := []string{
			common.Assets.GoogleCloud,
			common.Assets.AmazonWebServices,
			common.Assets.MicrosoftAzure,
			common.Assets.Office365,
			common.Assets.GSuite,
			common.Assets.Looker,
		}
		assets, err := s.assetsDal.ListBaseAssetsForCustomer(ctx, customerRef, -1)
		if err != nil {
			s.loggerProvider(ctx).Errorf("%sfailed to get customer assets for customer %s\n Error: %v", loggerPrefix, customerRef.ID, err)
		}

		for _, asset := range assets {
			if slices.Contains(assetTypes, asset.AssetType) {
				return notification.courierTemplateMap[procurementOnly]
			}
		}
	}

	id, ok := notification.courierTemplateMap[tierName]
	if ok {
		return id
	}

	return notification.courierTemplateID
}

func (s *TiersService) getTrialPeriod(ctx context.Context, customerRef *firestore.DocumentRef, tier *pkg.CustomerTier, packageType string) (int, error) {
	var period time.Duration

	if tier.TrialStartDate == nil || tier.TrialEndDate == nil {
		contractRef, err := s.contractsDal.GetCustomerContractRef(ctx, customerRef, packageType)

		if err != nil {
			return -1, fmt.Errorf("%sfailed to get contract for customer %s\n Error: %v", loggerPrefix, customerRef.ID, err)
		}

		contract, err := s.contractsDal.Get(ctx, contractRef.ID)
		if err != nil {
			return -1, fmt.Errorf("%sfailed to read contract for customer %s\n Error: %v", loggerPrefix, customerRef.ID, err)
		}

		period = contract.EndDate.Sub(*contract.StartDate)
	} else {
		period = tier.TrialEndDate.Sub(*tier.TrialStartDate)
	}

	return int(period.Hours() / 24), nil
}
