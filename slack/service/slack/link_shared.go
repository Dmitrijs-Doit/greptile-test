package slack

import (
	"context"
	"fmt"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/slack/domain"
)

// generateUnfurlPayload - initiate UnfurlPayload object
func (s *SlackService) generateUnfurlPayload(ctx context.Context, req *domain.SlackRequest) (*domain.UnfurlPayload, error) {
	l := s.loggerProvider(ctx)

	l.Infof("Slack link_shared payload: %+v", *req)

	URL := req.Event.Links[0].URL

	customerID, chartType, chartID, err := extractChartData(URL)
	if err != nil {
		l.Warningf("failed to extract data from URL [%s] with error: %s", URL, err)
		return nil, nil //	do not return error for data extraction
	}

	l.Infof("customerID %v, chartType %v, chartID %v", customerID, chartType, chartID)

	email, err := s.slackDAL.GetUserEmail(ctx, req.TeamID, req.Event.User)
	if err != nil {
		return nil, err
	}

	return &domain.UnfurlPayload{
		MessageTs:  req.Event.MessageTs,
		Channel:    req.Event.Channel,
		Unfurl:     nil,
		TeamID:     req.TeamID,
		CustomerID: customerID,
		ChartType:  chartType,
		ChartID:    chartID,
		Email:      email,
	}, nil
}

// handleError - send unfurl for given error & return it
func (s *SlackService) handleError(ctx context.Context, unfurlPayload *domain.UnfurlPayload, URL string, err error) error {
	l := s.loggerProvider(ctx)
	l.Error(err)

	if unfurlPayload != nil {
		unfurlPayload.Unfurl = s.ErrorUnfurlPayload(URL, err)

		if err := s.slackDAL.SendUnfurl(ctx, unfurlPayload); err != nil {
			l.Error(domain.ErrorUnfurl, err)
			return err
		}
	}

	return err
}

// GenerateUnfurlMixpanelPayload - returns event type & mixpanel payload to be included
func (s *SlackService) GenerateUnfurlMixpanelPayload(ctx context.Context, unfurlPayload *domain.UnfurlPayload, URL string) (*domain.MixpanelProperties, error) {
	l := s.loggerProvider(ctx)

	event := EventReportUnfurl
	if unfurlPayload.ChartType == domain.TypeBudget {
		event = EventBudgetUnfurl
	}

	mixpanelProperties := &domain.MixpanelProperties{
		Event:       event,
		Email:       unfurlPayload.Email,
		ChannelID:   unfurlPayload.Channel,
		WorkspaceID: unfurlPayload.TeamID,
		ChartID:     unfurlPayload.ChartID,
	}

	updateIsRequired, err := s.GenerateMixpanelPayload(ctx, mixpanelProperties)
	if updateIsRequired { // send unfurl indicating for available update
		unfurlPayload.Unfurl = s.UpdateUnfurlPayload(URL, unfurlPayload.Unfurl)
		if err := s.slackDAL.SendUnfurl(ctx, unfurlPayload); err != nil {
			l.Error(domain.ErrorUnfurl, err)
			return mixpanelProperties, err
		}
	}

	return mixpanelProperties, err
}

/*
	promptChartCollaboration - will suggest chart link sender, to share (extend collaboration to) the given chart

following those logics: (see reference here https://docs.google.com/spreadsheets/d/1KaG8XQBzO9UqAbkuL7TbJuWJZiu2v2Vov3TQ76DbmIU/edit?usp=sharing)

* DO NOTHING:

	** chart is public
	** sender is not a collaborator
	** sender is a Viewer collaborator
	** sender is an Editor & slack channel is private (cannot get channel members)

* PROMPT `share to channel`:

	** sender is an Editor & slack channel is public

* PROMPT `share to workspace`:

	** sender is the Owner & slack channel is private (cannot get channel members)

* PROMPT both `share to channel` & `share to workspace`:

	** sender is the Owner & slack channel is public
*/
func (s *SlackService) promptChartCollaboration(ctx context.Context, chart interface{}, unfurlPayload *domain.UnfurlPayload, URL string) error {
	l := s.loggerProvider(ctx)
	chartFullID := fmt.Sprintf("%s/%s", unfurlPayload.ChartType, unfurlPayload.ChartID)

	if shouldPrompt := shouldPromptChartCollaboration(chart, unfurlPayload.Email); !shouldPrompt {
		// abort prompt for public charts, or when sender is neither owner nor editor
		l.Infof("[%s] is already public, or sender [%s] is not a collaborator who can share. aborting collaboration prompt\n", chartFullID, unfurlPayload.Email)
		return nil
	}

	collaboratorRole := getCollaboratorRole(chart, unfurlPayload.Email)

	missingCollaborators, err := s.getMissingCollaborators(ctx, chart, unfurlPayload.TeamID, unfurlPayload.Channel)
	channelMembersUnknown := len(missingCollaborators) == 0

	if noMissingCollaborators := err == nil && channelMembersUnknown; noMissingCollaborators {
		l.Infof("all members of channel [%s] are collaborators of [%s]\n", unfurlPayload.Channel, chartFullID)
		return nil
	}

	if err != nil {
		l.Infof("could not get channel [%s] members, proceeding without `share to channel` options. error [%s]\n", unfurlPayload.Channel, err)

		if *collaboratorRole == collab.CollaboratorRoleEditor {
			l.Infof("sender [%s] is editor, cannot share to channel. aborting collaboration prompt\n", unfurlPayload.Channel)
			return nil
		}
	}

	_, _, chartName := GetChartFields(chart)

	unfurlPayload.Ephemeral = s.PromptChartCollaborationPayload(unfurlPayload.ChartType, URL, chartName, channelMembersUnknown, collaboratorRole)

	return s.slackDAL.SendUnfurlWithEphemeral(ctx, unfurlPayload)
}

func (s *SlackService) UpdateChartCollaboration(ctx context.Context, req *domain.ChartCollaborationReq) error {
	l := s.loggerProvider(ctx)
	payload := ParseChartCollaborationReq(req)

	chart, err := s.getChart(ctx, payload.ChartType, payload.ChartID, payload.CustomerID)
	if err != nil {
		return s.handleChartCollaborationError(ctx, payload, err)
	}

	if !payload.CancelAction {
		email, err := s.slackDAL.GetUserEmail(ctx, payload.WorkspaceID, payload.SlackUserID)
		if err != nil {
			return s.handleChartCollaborationError(ctx, payload, err)
		}

		user, err := s.firestoreDAL.GetUser(ctx, email)
		if err != nil {
			return err
		}

		l.Infof("user from firestore: id [%s], name [%s], customer [%s]\n", user.ID, user.DisplayName, user.Customer.Name)

		channelMembers := []string{}

		groupID := payload.ChannelID
		if !payload.Public {
			groupID = payload.WorkspaceID

			channelMembers, err = s.getMissingCollaborators(ctx, chart, payload.WorkspaceID, payload.ChannelID)
			if err != nil {
				return s.handleChartCollaborationError(ctx, payload, err)
			}
		}

		var shareErr error

		switch payload.ChartType {
		case domain.TypeBudget:
			shareErr = s.budgetsService.UpdateSharing(ctx, payload.ChartID, user, channelMembers, payload.Role, payload.Public)
		case domain.TypeReport:
			shareErr = s.reportsService.UpdateSharing(ctx, payload.ChartID, payload.CustomerID, user, channelMembers, payload.Role, payload.Public)
		default:
			shareErr = domain.InvalidLinkError
		}

		if shareErr != nil {
			return s.handleChartCollaborationError(ctx, payload, shareErr)
		}

		l.Infof("successfully updated [%s/%s] collaborators to have all [%s-%s] members with role [%s]\n", payload.ChartType, payload.ChartID, payload.GroupToShareWith, groupID, payload.Role)
	}

	_, _, chartName := GetChartFields(chart)

	payload.Ephemeral = s.UpdateChartCollaborationPayload(payload.CancelAction, chartName)
	if err := s.slackDAL.SendResponse(ctx, payload.WorkspaceID, payload.ChannelID, payload.ResponseURL, payload.Ephemeral); err != nil {
		return s.handleChartCollaborationError(ctx, payload, err)
	}

	return nil
}

// HandleError - send ephemeral for the given error and return it
func (s *SlackService) handleChartCollaborationError(ctx context.Context, payload *domain.ChartCollaborationPayload, err error) error {
	logger := s.loggerProvider(ctx)
	logger.Error(err)

	if payload != nil {
		payload.Ephemeral = s.UpdateChartCollaborationPayload(false, "", true)

		if err := s.slackDAL.SendResponse(ctx, payload.WorkspaceID, payload.ChannelID, payload.ResponseURL, payload.Ephemeral); err != nil {
			logger.Error(domain.ErrorUnfurl, err)
			return err
		}
	}

	return err
}

func (s *SlackService) getChart(ctx context.Context, chartType domain.ChartType, ID, customerID string) (interface{}, error) {
	switch chartType {
	case domain.TypeBudget:
		return s.budgetsService.Get(ctx, ID)
	case domain.TypeReport:
		return s.reportsService.Get(ctx, customerID, ID)
	default:
		return "", domain.InvalidLinkError
	}
}

func (s *SlackService) getMissingCollaborators(ctx context.Context, chart interface{}, workspaceID, channelID string) ([]string, error) {
	collaborators, _, _ := GetChartFields(chart)
	collaboratorsMap := map[string]bool{}

	for _, collaborator := range collaborators {
		collaboratorsMap[collaborator.Email] = true
	}

	channelMembers, err := s.slackDAL.GetAllChannelMemberEmails(ctx, workspaceID, channelID)
	if err != nil {
		return nil, err
	}

	nonCollaboratorMembers := []string{}

	for _, member := range channelMembers {
		if !collaboratorsMap[member] {
			nonCollaboratorMembers = append(nonCollaboratorMembers, member)
		}
	}

	return nonCollaboratorMembers, nil
}

func extractChartData(URL string) (string, domain.ChartType, string, error) {
	if URL == "" {
		return "", "", "", fmt.Errorf("no link")
	}

	splitDomain := strings.Split(URL, "doit.com/customers/")
	if len(splitDomain) < 2 {
		splitDomain = strings.Split(URL, "doit-intl.com/customers/") //	support also previous doit domain
		if len(splitDomain) < 2 {
			return "", "", "", domain.InvalidLinkError
		}
	}

	if common.Production {
		if splitDomain[0] == "https://hello." {
			return "", "", "", fmt.Errorf("links of 'https://hello.doit..' are deprecated")
		}

		if splitDomain[0] != "https://app." && splitDomain[0] != "https://console." { //	"https://app." to be deprecated as well
			return "", "", "", fmt.Errorf("can only unfurl production links")
		}
	}

	splitValues := strings.Split(splitDomain[1], "/")
	if len(splitValues) < 4 {
		return "", "", "", domain.InvalidLinkError
	}

	customerID, chartTypeStr, chartID := splitValues[0], splitValues[2], splitValues[3]
	chartID = strings.Split(chartID, "?")[0]

	chartType := domain.ChartType(chartTypeStr)
	if chartType != domain.TypeReport && chartType != domain.TypeBudget {
		return "", "", "", domain.InvalidLinkError
	}

	return customerID, chartType, chartID, nil
}
