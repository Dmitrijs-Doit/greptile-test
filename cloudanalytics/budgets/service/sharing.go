package service

import (
	"context"
	"errors"
	"regexp"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type ShareBudgetRequest struct {
	Collaborators           []collab.Collaborator `json:"collaborators"`
	PublicAccess            *collab.PublicAccess  `json:"public"`
	Recipients              []string              `json:"recipients"`
	RecipientsSlackChannels []common.SlackChannel `json:"recipientsSlackChannels"`
}

func (s *BudgetsService) ShareBudget(ctx context.Context, newShareBudget ShareBudgetRequest, budgetID, userID, requesterEmail string) error {
	if err := validateRecipients(newShareBudget.Recipients, newShareBudget.Collaborators); err != nil {
		return err
	}

	isCAOwner, err := s.caOwnerChecker.CheckCAOwner(ctx, s.employeeService, userID, requesterEmail)
	if err != nil {
		return err
	}

	budget, err := s.dal.GetBudget(ctx, budgetID)
	if err != nil {
		return err
	}

	if err := s.collab.ShareAnalyticsResource(ctx, budget.Collaborators, newShareBudget.Collaborators, newShareBudget.PublicAccess, budgetID, requesterEmail, s.dal, isCAOwner); err != nil {
		return err
	}

	if err := s.dal.UpdateBudgetRecipients(ctx, budgetID, newShareBudget.Recipients, newShareBudget.RecipientsSlackChannels); err != nil {
		return err
	}

	if err := s.UpdateEnforcedByMeteringField(ctx, budgetID, newShareBudget.Collaborators, newShareBudget.Recipients, newShareBudget.PublicAccess); err != nil {
		return err
	}

	return nil
}

// validateRecipients
func validateRecipients(recipients []string, collaborators []collab.Collaborator) error {
	for _, recipient := range recipients {
		slackChannelEmailRegex := regexp.MustCompile(`@[^.\s]{1,100}\.slack\.com$`)
		if slackChannelEmailRegex.MatchString(recipient) {
			// slack.com recipient is valid
			continue
		}

		if recipient == "no-reply@doit.com" {
			// Recipient is valid if is no-reply
			continue
		}
		// Recipient is valid if is collaborator
		isRecipientCollaborator := false

		for _, collaborator := range collaborators {
			if collaborator.Email == recipient {
				isRecipientCollaborator = true
				break
			}
		}

		if !isRecipientCollaborator {
			return errors.New("recipient is not a collaborator")
		}
	}

	return nil
}
