package slack

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	slackgo "github.com/slack-go/slack"

	"github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"github.com/doitintl/hello/scheduled-tasks/slack/domain"
)

// validateRequest used to authenticate request origin using headers, raw body and signing secret
func (s *SlackService) ValidateRequest(ctx *gin.Context, body []byte, appVerificationToken string) error {
	l := s.loggerProvider(ctx)

	if common.IsLocalhost {
		return nil
	}

	secret, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretSlackSigning)
	if err != nil {
		return err
	}

	slackApplicationsSigningMap := map[string]string{} //	App's Verification Token <--> App's Signing Secret
	if err := json.Unmarshal(secret, &slackApplicationsSigningMap); err != nil {
		return err
	}

	signingSecret := slackApplicationsSigningMap[appVerificationToken]

	slackTimestamp := ctx.GetHeader("X-Slack-Request-Timestamp")
	slackSignature := ctx.GetHeader("X-Slack-Signature")
	l.Infof("request timestamp: %s\nrequest signature: %s\n", slackTimestamp, slackSignature) // print for later debug of slack calls to app engine

	secretVerifier, err := slackgo.NewSecretsVerifier(ctx.Request.Header, string(signingSecret))
	if err != nil {
		return err
	}

	_, err = secretVerifier.Write(body)
	if err != nil {
		return err
	}

	err = secretVerifier.Ensure()
	if err != nil {
		return err
	}

	l.Infof("request verification succeeded")

	return nil
}

// ValidatePermissions - validate all required permissions for link unfurling
func (s *SlackService) ValidateUnfurlPermissions(ctx context.Context, unfurlPayload *domain.UnfurlPayload, chart interface{}) error {
	l := s.loggerProvider(ctx)

	isDoitEmployee, err := s.validateDoitEmployee(ctx, unfurlPayload.Email)
	if err != nil {
		l.Error(err)
		return err
	}

	if !isDoitEmployee {
		if err := s.validateSenderPermission(ctx, unfurlPayload.Email, unfurlPayload.CustomerID); err != nil {
			l.Error(err)
			return fmt.Errorf("sender does not have sufficient permission to unfurl the given link")
		}

		if budgetIsShareable := validateChartPermissions(chart, unfurlPayload.Email); !budgetIsShareable {
			return fmt.Errorf("chart cannot be shared by the sender")
		}
	}

	if err := s.validateSharedChannelPermission(ctx, unfurlPayload.CustomerID, unfurlPayload.Channel); err != nil {
		l.Error(err)
		return fmt.Errorf("cant unfurl the given link on this shared channel: " + unfurlPayload.Channel)
	}

	l.Infof("All required permissions are met")

	return nil
}

func (s *SlackService) validateDoitEmployee(ctx context.Context, email string) (bool, error) {
	l := s.loggerProvider(ctx)

	if strings.Contains(email, "doit-intl.com") || strings.Contains(email, "doit.com") { // allow 2 of the possible doit's email domains
		doitEmployee, err := s.firestoreDAL.GetDoitEmployee(ctx, email)
		if err != nil && err != firestore.ErrNotFound {
			return false, err
		}

		if doitEmployee != nil {
			l.Infof("Sender [%s] is a doit employee - has share permissions\n", doitEmployee.DisplayName)
			return true, nil
		}
	}

	return false, nil
}

func (s *SlackService) validateSenderPermission(ctx context.Context, email, customerID string) error {
	l := s.loggerProvider(ctx)

	user, err := s.firestoreDAL.GetUser(ctx, email)
	if err != nil {
		return err
	}

	// validate the user belongs to the link's customer
	if customerID != "" && user.Customer.Ref != nil && user.Customer.Ref.ID != "" {
		if user.Customer.Ref.ID != customerID {
			return fmt.Errorf("user does not belong to the organization which the chart belongs to")
		}
	}

	// validate that the user has cloudanalytics permission
	userHasCloudAnalyticsPermission, err := s.firestoreDAL.UserHasCloudAnalyticsPermission(ctx, email)
	if err != nil {
		return err
	}

	if !userHasCloudAnalyticsPermission {
		return fmt.Errorf("user does not have cloudanalytics permission")
	}

	l.Infof("Sender has share permissions")

	return nil
}

func (s *SlackService) validateSharedChannelPermission(ctx context.Context, customerID, channelID string) error {
	l := s.loggerProvider(ctx)

	sharedChannels, err := s.firestoreDAL.GetSharedChannel(ctx, channelID)
	if err == firestore.ErrNotFound {
		l.Infof("Channel is not shared with external customer, can unfurl")
		return nil
	}

	if err != nil {
		return err
	}

	if customerID != "" && sharedChannels.Customer != nil {
		if sharedChannels.Customer.ID != customerID {
			return fmt.Errorf("shared channel's customerID does not correlate link's customerID")
		}
	}

	l.Infof("Shared channel's customerID correlates link's customerID, can unfurl")

	return nil
}

// validateChartPermissions - either chart is public or email is collaborator
func validateChartPermissions(chart interface{}, email string) bool {
	collaborators, isPublic, _ := GetChartFields(chart)
	if isPublic {
		return true
	}

	for _, collaborator := range collaborators {
		if collaborator.Email == email {
			return true
		}
	}

	return false
}

// shouldPromptChartCollaboration - only prompt collaboration if sender is Owner or Editor & chart is not public
func shouldPromptChartCollaboration(chart interface{}, email string) bool {
	collaborators, isPublic, _ := GetChartFields(chart)
	if isPublic {
		return false
	}

	for _, collaborator := range collaborators {
		if email == collaborator.Email {
			return collaborator.Role == collab.CollaboratorRoleOwner || collaborator.Role == collab.CollaboratorRoleEditor
		}
	}

	return false
}
