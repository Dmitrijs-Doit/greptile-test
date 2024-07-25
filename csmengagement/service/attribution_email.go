package service

import (
	"context"
	"strings"

	"time"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
	csmDal "github.com/doitintl/hello/scheduled-tasks/csmengagement/dal"
	customerDomain "github.com/doitintl/hello/scheduled-tasks/customer/domain"
	nc "github.com/doitintl/notificationcenter/pkg"
)

func (s *service) SendNewAttributionEmails(ctx context.Context) error {
	twoDaysAgoStart := time.Now().AddDate(0, 0, -2).Truncate(24 * time.Hour)
	twoDaysAgoEnd := twoDaysAgoStart.Add(24 * time.Hour)
	dal := csmDal.NewAttributionEmail(s.fs)

	atrs, err := dal.GetAttributionsByDateRange(ctx, twoDaysAgoStart, twoDaysAgoEnd)
	if err != nil {
		return err
	}

	tracker := &csmDal.FsTracker{
		DocRef: s.fs.Collection("csmEngagement").Doc("attributionEmail"),
	}

	_, err = s.sendAttributionEmails(ctx, atrs, true, dal, tracker)

	return err
}

func (s *service) sendAttributionEmails(ctx context.Context, atrs []csmDal.AttributionData, onlyFirstAtr bool, dal csmDal.IAttributionEmail, tracker csmDal.SentNotificationsTracker) (int, error) {
	alreadyEmailed, err := tracker.GetSent(ctx)
	if err != nil {
		return 0, err
	}

	recipients, err := s.getFilteredRecipientData(ctx, dal, atrs, alreadyEmailed, onlyFirstAtr)
	if err != nil {
		return 0, err
	}

	for _, r := range recipients {
		accountTeam, err := s.customerDAL.GetCustomerAccountTeam(ctx, r.CustomerID)
		if err != nil {
			s.l.Errorf("failed to get account team for customer %s: %v", r.CustomerID, err)
			continue
		}

		requestID, err := sendEmail(ctx, s.notificationSender, r, accountTeam)
		if err != nil {
			s.l.Errorf("failed to send email to owner: %v", err)
			continue
		}

		s.l.Infof("sent email to owner %s, request id: %s", r.OwnerEmail, requestID)

		alreadyEmailed[r.OwnerEmail] = requestID
	}

	if len(recipients) > 0 {
		return len(recipients), tracker.UpdateSent(ctx, alreadyEmailed)
	}

	return 0, nil
}

type RecipientData struct {
	AttributionID string
	SetFirst      bool
	CustomerID    string
	OwnerEmail    string
	OwnerName     string
}

// getFilteredRecipientData returns a list of recipients that are eligible to receive an email
func (s *service) getFilteredRecipientData(ctx context.Context, dal csmDal.IAttributionEmail, atrs []csmDal.AttributionData, emailedUsers map[string]string, onlyFirstAtr bool) ([]RecipientData, error) {
	var filtered []RecipientData

	seenOwners := map[string]bool{}

	for _, atr := range atrs {
		for _, c := range atr.Collabs {
			if _, ok := emailedUsers[c.Email]; ok {
				continue
			}

			if c.Role != collab.CollaboratorRoleOwner {
				continue
			}

			if _, ok := seenOwners[c.Email]; ok {
				continue
			}

			if strings.HasSuffix(c.Email, "doit.com") || strings.HasSuffix(c.Email, "doit-intl.com") {
				continue
			}

			seenOwners[c.Email] = true

			if onlyFirstAtr {
				isFirstAttribution, err := dal.IsFirstAttribution(ctx, atr.AttributionID, c)
				if err != nil {
					return nil, err
				}

				if !isFirstAttribution {
					continue
				}
			}

			ownerCreatedBudgets, err := dal.HasBudgets(ctx, c)
			if err != nil {
				return nil, err
			}

			ownerCreatedAlerts, err := dal.HasAlerts(ctx, c)
			if err != nil {
				return nil, err
			}

			if ownerCreatedBudgets || ownerCreatedAlerts {
				continue
			}

			MRR, err := s.csmService.GetCustomerMRR(ctx, atr.CustomerID, true)
			if err != nil {
				s.l.Errorf("failed to get customer MRR for customer %s: %v", atr.CustomerID, err)
				continue
			}

			if MRR < 20000 {
				continue
			}

			user, err := s.userDAL.GetUserByEmail(ctx, c.Email, atr.CustomerID)
			if err != nil {
				s.l.Errorf("failed to get user by email %s: %v", c.Email, err)
				continue
			}

			filtered = append(filtered, RecipientData{
				AttributionID: atr.AttributionID,
				SetFirst:      onlyFirstAtr,
				CustomerID:    atr.CustomerID,
				OwnerEmail:    c.Email,
				OwnerName:     user.FirstName,
			})
		}
	}

	return filtered, nil
}

func sendEmail(ctx context.Context, client nc.NotificationSender, atr RecipientData, accountTeam []customerDomain.AccountManagerListItem) (string, error) {
	bcc := make([]string, 0, len(accountTeam))
	for _, member := range accountTeam {
		bcc = append(bcc, member.Email)
	}

	var first string // courier template doesn't support boolean values

	var subject string

	if atr.SetFirst {
		subject = "Congrats on Your First Attribution!"
		first = "true"
	} else {
		subject = "Unlock Advanced Cloud Cost Tracking with the DoiT Console"
		first = "false"
	}

	n := nc.Notification{
		Template: "ZRDJ2MN79041EFQN5XF2MGYBTMY2",
		Email:    []string{atr.OwnerEmail},
		BCC:      bcc,
		EmailFrom: nc.EmailFrom{
			Name:  "DoiT International",
			Email: "csm@doit-intl.com",
		},
		Data: map[string]interface{}{
			"subject": subject,
			"name":    atr.OwnerName,
			"first":   first,
			"link":    getConsoleLink(atr.CustomerID, atr.AttributionID),
		},
		Mock: !common.Production,
	}

	return client.Send(ctx, n)
}

func getConsoleLink(customerID string, atrID string) string {
	sub := "dev-app"
	if common.Production {
		sub = "console"
	}

	return "https://" + sub + ".doit.com/customers/" + customerID + "/analytics/attributions/" + atrID
}
