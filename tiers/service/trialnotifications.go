package service

import (
	"context"
	"slices"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/tiers/dal"
	notificationcenter "github.com/doitintl/notificationcenter/pkg"
)

type notificationsData struct {
	customerRef       *firestore.DocumentRef
	customerName      string
	users             []*common.User
	amsAdded          bool
	tierName          string
	tiersData         map[string]*pkg.CustomerTier
	packageType       string
	notifications     []trialNotification
	nIdx              int
	lastNotifications *dal.CustomerTrialNotifications
	dryRun            bool
}

type customerData struct {
	Name  string                       `firestore:"name"`
	Tiers map[string]*pkg.CustomerTier `firestore:"tiers"`
}

func (s *TiersService) SendTrialNotifications(ctx context.Context, dryRun bool) error {
	s.handleAllInactiveCustomerNotifications(ctx, dryRun)

	s.handleAllActiveTrialNotifications(ctx, dryRun)

	return nil
}

func (s *TiersService) handleAllActiveTrialNotifications(ctx context.Context, dryRun bool) {
	logger := s.loggerProvider(ctx)

	customersMap, err := s.getTrialNotificationsCustomers(ctx)
	if err != nil {
		logger.Errorf("%sfailed to read active trial customers\n Error: %v", loggerPrefix, err)
		return
	}

	var wg sync.WaitGroup

	for tierName, customerDocs := range customersMap {
		for _, customerDoc := range customerDocs {
			var customer customerData
			if err := customerDoc.DataTo(&customer); err != nil {
				logger.Errorf("%sfailed to read customer tier data\n Error: %v", loggerPrefix, err)
				continue
			}

			found := false

			for packageType, tier := range customer.Tiers {
				if tier.TrialCanceledDate == nil && (tier.TrialStartDate == nil || tier.TrialEndDate == nil) {
					continue
				}

				if _, ok := activeTrialNotifications[pkg.PackageTierType(packageType)]; ok {
					found = true
					break
				}

				if _, ok := usageNotifications[pkg.PackageTierType(packageType)]; ok {
					found = true
					break
				}
			}

			if !found {
				continue
			}

			wg.Add(1)

			go s.handleCustomerActiveTrialNotifications(ctx, &wg, customerDoc.Ref, &customer, tierName, dryRun)
		}
	}

	wg.Wait()
}

func (s *TiersService) handleAllInactiveCustomerNotifications(ctx context.Context, dryRun bool) {
	logger := s.loggerProvider(ctx)

	customersDocs, err := s.getInactiveCustomers(ctx)
	if err != nil {
		logger.Errorf("%sfailed to read active trial customers\n Error: %v", loggerPrefix, err)
		return
	}

	var wg sync.WaitGroup

	for _, customerDoc := range customersDocs {
		var customer customerData
		if err := customerDoc.DataTo(&customer); err != nil {
			logger.Errorf("%sfailed to read customer tier data\n Error: %v", loggerPrefix, err)
			continue
		}

		wg.Add(1)

		go s.handleInactiveCustomerNotification(ctx, &wg, customerDoc.Ref, &customer, dryRun)
	}

	wg.Wait()
}

func (s *TiersService) handleCustomerActiveTrialNotifications(ctx context.Context, wg *sync.WaitGroup, customerRef *firestore.DocumentRef, customer *customerData, tierName string, dryRun bool) {
	defer wg.Done()

	now := time.Now().UTC()

	logger := s.loggerProvider(ctx)

	data, err := s.getCustomerNotificationData(ctx, customerRef, customer, dryRun)
	if err != nil {
		logger.Error(err)
		return
	} else if data == nil {
		return
	}

	data.tiersData = customer.Tiers
	data.tierName = tierName

	if s.sendCustomerActiveTrialNotification(ctx, now, data) {
		s.updateCustomerNotificationsDoc(ctx, customerRef.ID, data.lastNotifications, dryRun)
	}
}

func (s *TiersService) handleInactiveCustomerNotification(ctx context.Context, wg *sync.WaitGroup, customerRef *firestore.DocumentRef, customer *customerData, dryRun bool) {
	defer wg.Done()

	now := time.Now().UTC()

	logger := s.loggerProvider(ctx)

	data, err := s.getCustomerNotificationData(ctx, customerRef, customer, dryRun)
	if err != nil {
		logger.Error(err)
		return
	} else if data == nil {
		return
	}

	data.packageType = string(pkg.NavigatorPackageTierType)

	if s.sendInactiveCustomerNotification(ctx, now, data) {
		s.updateCustomerNotificationsDoc(ctx, customerRef.ID, data.lastNotifications, dryRun)
	}
}

// sendActiveTrialNotifications sends relevant notifications and updates last sent data inside d.lastNotifications
func (s *TiersService) sendCustomerActiveTrialNotification(ctx context.Context, now time.Time, d *notificationsData) bool {
	var ok bool
	sentNewNotification := false

	for packageType, tier := range d.tiersData {
		if tier.TrialCanceledDate == nil && (tier.TrialStartDate == nil || tier.TrialEndDate == nil) {
			continue
		}

		d.packageType = packageType

		dates := &startEndDates{
			start: tier.TrialStartDate,
			end:   tier.TrialEndDate,
		}
		if tier.TrialCanceledDate != nil {
			dates.end = tier.TrialCanceledDate
		}

		// send trial period notifications
		if d.notifications, ok = activeTrialNotifications[pkg.PackageTierType(packageType)]; ok {
			lastDate := d.lastNotifications.LastSent[packageType]

			if d.nIdx = s.getLatestRelevantNotification(dates, d.notifications, now, lastDate); d.nIdx > -1 {
				trialPeriod, err := s.getTrialPeriod(ctx, d.customerRef, tier, d.packageType)
				if err != nil {
					s.loggerProvider(ctx).Errorf("%sfailed to get trial period for customer %s\n Error: %v", loggerPrefix, d.customerRef.ID, err)

					trialPeriod = defaultTrialPeriod
				}

				if s.sendCourierEmail(ctx, d,
					map[string]interface{}{
						"trialPeriod": trialPeriod,
						"date":        dates.end.Format("Jan 2, 2006"),
					}) {

					d.lastNotifications.LastSent[packageType] = now
					sentNewNotification = true
				}
			}
		}

		if dates.end.Before(now) {
			continue
		}

		// send usage notifications
		if d.notifications, ok = usageNotifications[pkg.PackageTierType(packageType)]; ok {
			if s.sendUsageNotifications(ctx, now, d) {
				sentNewNotification = true
			}
		}
	}

	return sentNewNotification
}

func (s *TiersService) sendUsageNotifications(ctx context.Context, now time.Time, d *notificationsData) bool {
	logger := s.loggerProvider(ctx)

	sentNewNotification := false

	for i, n := range d.notifications {
		if n.usageVerifier == nil || slices.Contains(d.lastNotifications.UsageSent[d.packageType], n.id) {
			continue
		}

		dates := &startEndDates{
			start: d.tiersData[d.packageType].TrialStartDate,
			end:   d.tiersData[d.packageType].TrialEndDate,
		}

		if s.isRelevantNotification(n, dates, now, time.Time{}, n.maxAge) {
			nonZeroUsage, err := n.usageVerifier.hasUsage(ctx, s, d.customerRef)
			if err != nil {
				logger.Errorf("%sfailed to check %s usage for customer %s\n Error: %v", loggerPrefix, n.description, d.customerRef.ID, err)
				continue
			}

			if !nonZeroUsage {
				d.nIdx = i

				if s.sendCourierEmail(ctx, d, map[string]interface{}{}) {
					if d.lastNotifications.UsageSent == nil {
						d.lastNotifications.UsageSent = make(map[string][]string)
					}

					d.lastNotifications.UsageSent[d.packageType] = append(d.lastNotifications.UsageSent[d.packageType], n.id)
					sentNewNotification = true
				}
			}
		}
	}

	return sentNewNotification
}

func (s *TiersService) sendInactiveCustomerNotification(ctx context.Context, now time.Time, d *notificationsData) bool {
	var ok bool
	sentNewNotification := false

	firstUserTimestamp := s.getFirstUserTimestamp(ctx, d.users)
	if firstUserTimestamp.IsZero() {
		return false
	}

	d.notifications, ok = inactiveCustomerNotifications[pkg.PackageTierType(d.packageType)]
	if !ok {
		return false
	}

	lastDate := d.lastNotifications.LastSent[d.packageType]

	dates := &startEndDates{
		start: &firstUserTimestamp,
	}

	if d.nIdx = s.getLatestRelevantNotification(dates, d.notifications, now, lastDate); d.nIdx > -1 {
		if s.sendCourierEmail(ctx, d, map[string]interface{}{}) {
			d.lastNotifications.LastSent[d.packageType] = now
			sentNewNotification = true
		}
	}

	return sentNewNotification
}

func (s *TiersService) sendCourierEmail(ctx context.Context, d *notificationsData, emailData map[string]interface{}) bool {
	logger := s.loggerProvider(ctx)

	sentNewNotification := false

	if err := s.addAMsAndSMETeam(ctx, d); err != nil {
		s.loggerProvider(ctx).Errorf("%sfailed to add AMs for customer %s\n Error: %v", loggerPrefix, d.customerRef.ID, err)
	}

	notification := d.notifications[d.nIdx]

	for _, user := range d.users {
		emailData["name"] = user.FirstName

		dryRunPrefix := "Dry run: "

		if !d.dryRun {
			config := notificationcenter.Notification{
				Email:    []string{user.Email},
				Template: s.getCourierTemplateID(ctx, d.customerRef, notification, d.tierName),
				Data:     emailData,
			}

			_, err := s.notificationClient.Send(ctx, config)
			if err != nil {
				logger.Errorf("%sfailed to send %s notification for customer %s\n Error: %v", loggerPrefix, notification.description, d.customerRef.ID, err)
				continue
			}

			dryRunPrefix = ""
		}

		logger.Debugf("%s%ssent %s notification for customer %s to %s with params %+v", loggerPrefix, dryRunPrefix, notification.description, d.customerRef.ID, user.Email, emailData)

		sentNewNotification = true
	}

	return sentNewNotification
}
