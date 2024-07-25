package service

import (
	"context"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
	csmAttrDal "github.com/doitintl/hello/scheduled-tasks/csmengagement/dal"
	"github.com/doitintl/hello/scheduled-tasks/csmengagement/domain"
	nc "github.com/doitintl/notificationcenter/pkg"
)

const standardUserRoleName = "Standard User"
const noAttributionsEmailTemplate = "846CCQA5H94255QJY4H8C0FESXBD"
const csmSenderEmailAddress = "csm@doit-intl.com"

func (s *service) GetNoAttributionsEmails(ctx context.Context) ([]domain.NoAttributionEmailParams, error) {

	tracker := s.noAttrsDAL.GetTracker()
	emailsToSend := make([]domain.NoAttributionEmailParams, 0)

	allCustomAttrsMap, err := getAllCustomAttributionsMap(ctx, s.noAttrsDAL)
	if err != nil {
		return emailsToSend, err
	}

	customersEligibleForNotification, err := getAllCustomersEligibleForNotifications(ctx, s.noAttrsDAL, s.csmService, allCustomAttrsMap)
	if err != nil {
		return emailsToSend, err
	}

	requiredPermissionSnaps, err := s.noAttrsDAL.GetRequiredRolePermissions(ctx, standardUserRoleName)
	if err != nil {
		return emailsToSend, err
	}

	requiredPermissions := convertToPermissionFormat(requiredPermissionSnaps)

	emailsSent, err := tracker.GetSent(ctx)
	if err != nil {
		return emailsToSend, err
	}

	// Create a buffered channel to limit the number of goroutines
	sem := make(chan struct{}, 25)

	var wg sync.WaitGroup

	var emailsMutex sync.Mutex

	for _, customerDocSnap := range customersEligibleForNotification {
		wg.Add(1)

		go func(customerDocSnap *firestore.DocumentSnapshot) {
			defer wg.Done()

			sem <- struct{}{}

			defer func() { <-sem }()

			if err := handleUsers(ctx, s, s.noAttrsDAL, customerDocSnap, emailsSent, requiredPermissions, &emailsToSend, &emailsMutex); err != nil {
				s.l.Errorf("error handling users: %v", err)
			}
		}(customerDocSnap)
	}
	// Wait for all goroutines to finish
	wg.Wait()

	return emailsToSend, nil
}

func handleUsers(
	ctx context.Context,
	s *service,
	attrDal csmAttrDal.INoAttributionsEmail,
	customerDocSnap *firestore.DocumentSnapshot,
	emailsSent map[string]string,
	requiredPermissions []common.Permission,
	emailsToSend *[]domain.NoAttributionEmailParams,
	emailsMutex *sync.Mutex,
) error {
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	// get all the users for the customer
	users, err := attrDal.GetEligibleUsersForCustomer(ctx, customerDocSnap.Ref)
	if err != nil {
		return err
	}

	// if there are no users for the customer, skip
	if len(users) == 0 {
		return nil
	}

	// get the account team for the customer
	accountTeam, err := s.customerDAL.GetCustomerAccountTeam(ctx, customerDocSnap.Ref.ID)
	if err != nil {
		s.l.Errorf("failed to get account team for customer %s: %v", customerDocSnap.Ref.ID, err)
		return err
	}

	for _, user := range users {
		// if the user has already been emailed, skip
		if _, ok := emailsSent[user.Email]; ok {
			continue
		}

		if user.FirstName == "" {
			user.FirstName = strings.Split(user.Email, "@")[0]
		}
		// check that user has logged in the past 30 days:
		if user.LastLogin.Before(thirtyDaysAgo) {
			continue
		}

		// 1. Check that the user has a valid role:
		hasPermissions := attrDal.UserHasRequiredPermissions(ctx, user, requiredPermissions)
		if hasPermissions != nil {
			continue
		}

		// 2. Check that the user has no attributions created:
		attrRefs, err := attrDal.GetAttributionsForUser(ctx, collab.Collaborator{
			Email: user.Email,
			Role:  collab.CollaboratorRoleOwner,
		})
		if err != nil || len(attrRefs) > 0 {
			continue
		}

		// Append email parameters to the shared slice, with mutex protection
		emailsMutex.Lock()
		*emailsToSend = append(*emailsToSend, domain.NoAttributionEmailParams{
			RecipientEmail: user.Email,
			RecipientName:  user.FirstName,
			AccountTeam:    accountTeam,
			CustomerName:   user.Customer.Name,
		})
		emailsMutex.Unlock()
	}

	return nil
}

func convertToPermissionFormat(requiredPermissions []*firestore.DocumentRef) []common.Permission {
	permissionsInConstFormat := make([]common.Permission, 0)

	for _, ref := range requiredPermissions {
		// convert the permission id to a common.Permission const
		p := common.Permission(ref.ID)
		permissionsInConstFormat = append(permissionsInConstFormat, p)
	}

	return permissionsInConstFormat

}

func sendNoAttributionEmail(ctx context.Context, client nc.NotificationSender, params domain.NoAttributionEmailParams) (string, error) {
	bcc := make([]string, 0)

	for _, member := range params.AccountTeam {
		if member.Role == common.AccountManagerRoleTAM ||
			member.Role == common.AccountManagerRoleCSM ||
			member.Role == common.AccountManagerRoleSAM {
			bcc = append(bcc, member.Email)
		}
	}

	type accountTeamMember struct {
		Role  string `json:"role"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}

	accountTeam := make([]accountTeamMember, 0)

	for _, member := range params.AccountTeam {

		title := strings.Replace(string(member.Role), "_", " ", -1)
		title = strings.ToUpper(title[:1]) + title[1:]
		accountTeam = append(accountTeam, accountTeamMember{
			Role:  title,
			Email: member.Email,
			Name:  member.Name,
		})
	}

	n := nc.Notification{
		Template: noAttributionsEmailTemplate,
		Email:    []string{params.RecipientEmail},
		BCC:      bcc,
		EmailFrom: nc.EmailFrom{
			Name:  "DoiT International",
			Email: csmSenderEmailAddress,
		},
		Data: map[string]interface{}{
			"userName":     params.RecipientName,
			"customerName": params.CustomerName,
			"accountTeam":  accountTeam,
		},
		Mock: !common.Production,
	}

	return client.Send(ctx, n)
}

func getAllCustomAttributionsMap(ctx context.Context, attrDal csmAttrDal.INoAttributionsEmail) (map[string]attribution.Attribution, error) {
	allCustomAttrsMap := make(map[string]attribution.Attribution, 0)

	allCustomAttributions, err := attrDal.GetAllCustomAttributions(ctx)
	if err != nil {
		return allCustomAttrsMap, err
	}

	for _, atr := range allCustomAttributions {

		if atr.Customer == nil {
			continue
		}

		allCustomAttrsMap[atr.Customer.ID] = atr
	}

	return allCustomAttrsMap, nil
}

func getAllCustomersEligibleForNotifications(
	ctx context.Context,
	attrDal csmAttrDal.INoAttributionsEmail,
	csmService CSMEngagement,
	allCustomAttrsMap map[string]attribution.Attribution) ([]*firestore.DocumentSnapshot, error) {
	var customersEligibleForNotification []*firestore.DocumentSnapshot

	var mu sync.Mutex

	var wg sync.WaitGroup

	allCustomerDocSnaps, err := attrDal.GetCustomersNewerThanThirtyDays(ctx)
	if err != nil {

		return customersEligibleForNotification, err
	}

	sem := make(chan struct{}, 25)
	for _, customerDocSnap := range allCustomerDocSnaps {

		wg.Add(1)
		sem <- struct{}{}

		go func(customerDocSnap *firestore.DocumentSnapshot) {

			defer wg.Done()

			defer func() { <-sem }()

			if _, ok := allCustomAttrsMap[customerDocSnap.Ref.ID]; !ok {

				// check if the customer is resold and has MRR > 20000
				isResold, err := csmService.IsCustomerResold(ctx, customerDocSnap.Ref.ID)
				if err != nil || !isResold {
					return
				}

				MRR, err := csmService.GetCustomerMRR(ctx, customerDocSnap.Ref.ID, true)
				if err != nil {
					return
				}

				if MRR < 20000 {
					return
				}

				mu.Lock()
				customersEligibleForNotification = append(
					customersEligibleForNotification,
					customerDocSnap,
				)
				mu.Unlock()
			}
		}(customerDocSnap)
	}

	wg.Wait()

	return customersEligibleForNotification, nil
}

func (s *service) SendNoAttributionsEmails(
	ctx context.Context,
	emailsToSend []domain.NoAttributionEmailParams) error {

	sentCounter := 0
	tracker := s.noAttrsDAL.GetTracker()

	alreadySent, err := tracker.GetSent(ctx)
	if err != nil {
		return err
	}

	client := s.notificationSender
	for _, emailParams := range emailsToSend {
		id, err := sendNoAttributionEmail(ctx, client, emailParams)
		if err != nil {
			return err
		}
		alreadySent[emailParams.RecipientEmail] = id
		sentCounter++
	}

	if sentCounter > 0 {
		return tracker.UpdateSent(ctx, alreadySent)
	}

	return nil
}
