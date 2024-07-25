package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/common"
	csmManagement "github.com/doitintl/hello/scheduled-tasks/csmengagement/dal"
	notificationCenter "github.com/doitintl/notificationcenter/pkg"
)

func (s *service) SendNoCustomerEngagementNotifications(ctx context.Context) error {
	usersWithRecentEngagement, err := s.userDAL.GetUsersWithRecentEngagement(ctx)
	if err != nil {
		return fmt.Errorf("getting users with recent engagement: %w", err)
	}

	allCustomerIDs, err := s.customerDAL.GetAllCustomerIDs(ctx)
	if err != nil {
		return fmt.Errorf("getting all customer IDs: %w", err)
	}

	customersWithNoRecentEngagement := getCustomersWithNoEngagement(allCustomerIDs, usersWithRecentEngagement)

	allCustomerEngagementDetails, err := s.csmEngagementDAL.GetCustomerEngagementDetailsByCustomerID(ctx)
	if err != nil {
		return fmt.Errorf("getting all customer engagement details: %w", err)
	}

	for _, customerID := range customersWithNoRecentEngagement {
		customer, err := s.customerDAL.GetCustomer(ctx, customerID)
		if err != nil {
			s.l.Error(fmt.Errorf("getting customer: %w", err))
			continue
		}

		if customer == nil {
			continue
		}

		if customer.Inactive() || customer.Terminated() || customer.SuspendedForNonPayment() {
			continue
		}

		productOnly, err := s.customerTypeDal.IsProductOnlyCustomerType(ctx, customerID)
		if err != nil {
			s.l.Error(fmt.Errorf("checking if customer is product only: %w", err))
			continue
		}

		if productOnly {
			continue
		}

		customerEngagementDetail, ok := allCustomerEngagementDetails[customerID]
		if ok {
			if customerEngagementDetail.WasNotifiedAboutWithinLastMonth() {
				continue
			}
		}

		mrr, err := s.csmService.GetCustomerMRR(ctx, customerID, true)
		if err != nil {
			s.l.Error(fmt.Errorf("getting customer MRR: %w", err))
			continue
		}

		const mrrEngagementThreshold = 20000
		if mrr < mrrEngagementThreshold {
			continue
		}

		lastUserEngagement, err := s.userDAL.GetLastUserEngagementTimeForCustomer(ctx, customerID)
		if err != nil {
			s.l.Error(fmt.Errorf("getting last user engagement time for customer: %w", err))
			continue
		}

		if lastUserEngagement == nil {
			continue
		}

		if skipDueToNoEngagementSinceEmail(lastUserEngagement, customerID, allCustomerEngagementDetails) {
			continue
		}

		daysSinceLastEngagement := math.Round(time.Since(*lastUserEngagement).Hours() / 24)

		accountTeam, err := s.customerDAL.GetCustomerAccountTeam(ctx, customerID)
		if err != nil {
			s.l.Error(fmt.Errorf("getting customer account team: %w", err))
			continue
		}

		wasSent := false

		for _, accountManager := range accountTeam {
			if accountManager.Role == common.AccountManagerRoleFSR ||
				accountManager.Role == common.AccountManagerRoleTAM ||
				accountManager.Role == common.AccountManagerRoleSAM {
				_, err := s.notificationSender.Send(ctx, notificationCenter.Notification{
					Template: notificationCenter.NoRecentUserActivity,
					Email:    []string{accountManager.Email},
					EmailFrom: notificationCenter.EmailFrom{
						Name:  "DoiT International",
						Email: "csm@doit-intl.com",
					},
					Data: map[string]interface{}{
						"CUSTOMER":            customer.Name,
						"account_team_member": accountManager.Name,
						"number_of":           fmt.Sprintf("%d", int(daysSinceLastEngagement)),
					},
					Mock: !common.Production,
				})
				if err != nil {
					s.l.Error(err)
					continue
				}

				wasSent = true
			}
		}

		if wasSent {
			err = s.csmEngagementDAL.AddLastCustomerEngagementTime(ctx, customerID, time.Now())
			if err != nil {
				s.l.Error(err)
				continue
			}
		}
	}

	return nil
}

func getCustomersWithNoEngagement(allCustomerIDs []string, usersWithRecentEngagement []common.User) []string {
	customersWithEngagement := make(map[string]bool)

	for _, u := range usersWithRecentEngagement {
		if u.Customer.Ref == nil {
			continue
		}

		customersWithEngagement[u.Customer.Ref.ID] = true
	}

	customerIDs := make([]string, 0)

	for _, id := range allCustomerIDs {
		if _, ok := customersWithEngagement[id]; !ok {
			customerIDs = append(customerIDs, id)
		}
	}

	return customerIDs
}

func skipDueToNoEngagementSinceEmail(lastUserEngagement *time.Time, customerID string, allEngagementDetails map[string]csmManagement.EngagementDetails) bool {
	if lastUserEngagement == nil || allEngagementDetails == nil {
		return false
	}

	if _, ok := allEngagementDetails[customerID]; !ok {
		return false
	}

	for _, notifiedDate := range allEngagementDetails[customerID].NotifiedDates {
		if lastUserEngagement.After(notifiedDate) {
			return false
		}
	}

	return true
}
