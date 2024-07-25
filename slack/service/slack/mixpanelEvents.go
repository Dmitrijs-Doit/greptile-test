package slack

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/slack/domain"
)

func (s *SlackService) ParseMixpanelRequest(ctx *gin.Context) (*domain.MixpanelProperties, error) {
	l := s.loggerProvider(ctx)

	var req domain.MixpanelProperties
	if err := ctx.ShouldBindJSON(&req); err != nil {
		return nil, err
	}

	l.Infof("Mixpanel request: %+v", req)

	if req.Event == "" {
		return nil, errors.New("no event received")
	}

	_, err := s.GenerateMixpanelPayload(ctx, &req)

	return &req, err
}

// GenerateMixpanelPayload - to invoke mixpanel event related to an action happened on slack app. also returns true if installed app is outdated
func (s *SlackService) GenerateMixpanelPayload(ctx context.Context, properties *domain.MixpanelProperties) (bool, error) {
	l := s.loggerProvider(ctx)

	var err error

	var updateIsRequired bool

	if properties.Email == "" {
		properties.Email, err = s.slackDAL.GetUserEmail(ctx, properties.WorkspaceID, properties.User)
		if err != nil {
			return false, err
		}
	}

	workspace, _, _, _, err := s.firestoreDAL.GetWorkspaceDecrypted(ctx, properties.WorkspaceID)
	if err != nil {
		return false, err
	}

	if properties.ChannelName == "" {
		channel, err := s.slackDAL.GetChannelInfo(ctx, properties.WorkspaceID, properties.ChannelID)
		if err != nil {
			l.Infof("failed getting channel info with error: %s", err)

			if err.Error() == domain.ErrorMissingScope.Error() {
				// when missing scopes - check if slack app is updated, return error if not
				if err := s.CheckVersionUpdated(ctx, properties.WorkspaceID); err != nil {
					if err.Error() == domain.ErrorAppIsOutdated.Error() { //	backward compatibility - when app version with no channels scopes is installed proceed without channel name
						updateIsRequired = true
					} else {
						return false, err
					}
				}
			}
		}

		if channel != nil {
			properties.ChannelName = channel.Name
		}
	}

	payload := map[string]interface{}{
		"Slack Workspace":    workspace.Name,
		"Slack Workspace ID": properties.WorkspaceID,
		"Slack Channel":      properties.ChannelName,
		"Slack Channel ID":   properties.ChannelID,
	}

	if properties.ChartID != "" { //	ChartID relevant for "slack.CHART_TYPE.unfurl/view/investigate" events
		payload["Chart ID"] = properties.ChartID
	}

	properties.Payload = payload

	l.Infof("slack mixpanel properties: %+v", properties)

	return updateIsRequired, nil
}
