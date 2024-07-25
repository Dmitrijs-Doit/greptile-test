package slack

import (
	"fmt"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/slack/domain"
	slackgo "github.com/slack-go/slack"
)

const (
	// app home payloads
	AppHomeHeader               string = "Connect your CMP account"
	AppHomeContent              string = "To proceed, please <https://help.doit.com/general/slack|connect your CMP Account>. To try out the CMP please <https://console.doit.com/signup|click here to take a test drive>!"
	AppHomeAuthenticatedHeader  string = "All Set!"
	AppHomeAuthenticatedContent string = "Your <https://console.doit.com/customers/%s/integrations/slack|CMP account> is connected."
	AppHomeButton               string = "Connect Account"

	// welcome message payload
	WelcomeMessageSectionText1 string = "Thank you for joining your first shared channel with DoiT."
	WelcomeMessageSectionText2 string = "Take some time to <%s|view and configure> the notifications you can expect to receive."
	WelcomeMessageSectionText3 string = "You can view <%s|open tickets>, <%s|cloud incidents>, and if you have an urgent request open a <%s|support ticket>."
	WelcomeMessageSectionText4 string = "Also, your essential contacts are added to this channel."

	// unfurl payloads

	// Errors to preview on the unfurl response
	ErrorDefault         string = "Arghh. Something went plain wrong. Sorry!"
	ErrorNotFound        string = "Can’t find this! Please check the URL."
	ErrorUserNotFound    string = "Hmm, we are unable to identify your email address."
	ErrorUserPermissions string = "It seems you don’t have access to this object, do you?"

	// Texts bold
	TextPreviewError string = "*Preview error*"

	// Texts
	TextLoadingLink       string = "Loading link preview..."
	TextLoading           string = "Loading..."
	TextUpdate            string = "Update"
	TextUpdateDescription string = "It seems like a newer version of DoiT International app exist on <https://slack.com/apps/AF79TTA7N-doit-international|Slack App directory>"

	// Mixpanel events
	EventBudgetInvestigate         string = "slack.budget.investigate"
	EventBudgetView                string = "slack.budget.view"
	EventBudgetUnfurl              string = "slack.budget.unfurl"
	EventReportView                string = "slack.report.view"
	EventReportUnfurl              string = "slack.report.unfurl"
	EventBudgetCollaborationUpdate string = "slack.budget.collaboration-update"
	EventBudgetCollaborationCancel string = "slack.budget.collaboration-cancel"
	EventReportCollaborationUpdate string = "slack.report.collaboration-update"
	EventReportCollaborationCancel string = "slack.report.collaboration-cancel"
	EventUpdate                    string = "slack.app.update"

	// Other
	ImageLoadingGif string = "https://storage.googleapis.com/hello-static-assets/images/doitintl-unfurl-loader.gif"
)

func (s *SlackService) UpdateChartCollaborationPayload(cancelAction bool, chartName string, err ...bool) []slackgo.Block {
	text := fmt.Sprintf("Access to *%s* has not been updated.", chartName)
	if !cancelAction {
		text = fmt.Sprintf("Access to *%s* has been updated.", chartName)
	}

	if len(err) != 0 {
		text = "Failed to update access. You can update access to this report via the DoiT Console."
	}

	return []slackgo.Block{
		slackgo.NewSectionBlock(
			slackgo.NewTextBlockObject(
				"mrkdwn",
				text,
				false,
				false,
			),
			nil,
			nil,
		),
	}
}

func (s *SlackService) PromptChartCollaborationPayload(chartType domain.ChartType, unfurlURL, chartName string, excludeChannelOptions bool, collaboratorRole *collab.CollaboratorRole) []slackgo.Block {
	updateAction := EventBudgetCollaborationUpdate
	cancelAction := EventBudgetCollaborationCancel

	if chartType == domain.TypeReport {
		updateAction = EventReportCollaborationUpdate
		cancelAction = EventReportCollaborationCancel
	}

	block1 := slackgo.NewSectionBlock(
		slackgo.NewTextBlockObject(
			"mrkdwn",
			fmt.Sprintf("Not everyone in this channel has access to view *%s*. You can edit access from the dropdown below.", chartName),
			false,
			false,
		),
		nil,
		nil,
	)

	channelOptions := []*slackgo.OptionBlockObject{
		slackgo.NewOptionBlockObject("channel_viewer", slackgo.NewTextBlockObject("plain_text", "View", false, false), nil),
		slackgo.NewOptionBlockObject("channel_editor", slackgo.NewTextBlockObject("plain_text", "Edit", false, false), nil),
	}

	workspaceOptions := []*slackgo.OptionBlockObject{
		slackgo.NewOptionBlockObject("workspace_viewer", slackgo.NewTextBlockObject("plain_text", "View", false, false), nil),
		slackgo.NewOptionBlockObject("workspace_editor", slackgo.NewTextBlockObject("plain_text", "Edit", false, false), nil),
	}

	channelGroup := slackgo.NewOptionGroupBlockElement(
		slackgo.NewTextBlockObject("plain_text", "Give channel members permission to...", false, false),
		channelOptions...,
	)

	workspaceGroup := slackgo.NewOptionGroupBlockElement(
		slackgo.NewTextBlockObject("plain_text", "Give org members permission to...", false, false),
		workspaceOptions...,
	)

	selectText := &slackgo.TextBlockObject{
		"plain_text",
		"Select one...",
		false,
		false,
	}

	var staticSelect *slackgo.SelectBlockElement

	switch *collaboratorRole {
	case collab.CollaboratorRoleOwner:
		staticSelect = slackgo.NewOptionsGroupSelectBlockElement( // both channel & workspace sections
			"static_select",
			selectText,
			updateAction,
			channelGroup,
			workspaceGroup,
		)
		if excludeChannelOptions {
			staticSelect = slackgo.NewOptionsGroupSelectBlockElement( // only workspace section
				"static_select",
				selectText,
				updateAction,
				workspaceGroup,
			)
		}
	case collab.CollaboratorRoleEditor:
		staticSelect = slackgo.NewOptionsGroupSelectBlockElement( // only channel section
			"static_select",
			selectText,
			updateAction,
			channelGroup,
		)
	}

	cancelButton := slackgo.NewButtonBlockElement(
		cancelAction,
		"cancel",
		slackgo.NewTextBlockObject(
			"plain_text",
			"Cancel",
			false,
			false,
		),
	)

	block2 := slackgo.NewActionBlock(
		unfurlURL,
		staticSelect,
		cancelButton,
	)

	return []slackgo.Block{
		block1,
		block2,
	}
}

func (s *SlackService) WelcomeToChannelPayload(URLs map[string]string) slackgo.MsgOption {
	blocks := []slackgo.Block{
		slackgo.NewSectionBlock(
			slackgo.NewTextBlockObject(slackgo.MarkdownType, WelcomeMessageSectionText1, false, false),
			nil,
			nil,
		),
		slackgo.NewSectionBlock(
			slackgo.NewTextBlockObject(slackgo.MarkdownType, fmt.Sprintf(WelcomeMessageSectionText2, URLs["settings"]), false, false),
			nil,
			nil,
		),
		slackgo.NewSectionBlock(
			slackgo.NewTextBlockObject(slackgo.MarkdownType, fmt.Sprintf(WelcomeMessageSectionText3, URLs["tickets"], URLs["incidents"], URLs["createTicket"]), false, false),
			nil,
			nil,
		),
		slackgo.NewSectionBlock(
			slackgo.NewTextBlockObject(slackgo.MarkdownType, WelcomeMessageSectionText4, false, false),
			nil,
			nil,
		),
	}

	return slackgo.MsgOptionBlocks(blocks...)
}

func (s *SlackService) AppHomePayload(authenticated bool, customerID, user string) slackgo.Blocks {
	header := AppHomeHeader
	content := AppHomeContent

	if authenticated && customerID != "" {
		header = AppHomeAuthenticatedHeader
		content = fmt.Sprintf(AppHomeAuthenticatedContent, customerID)
	}

	headerBlock := slackgo.NewHeaderBlock(&slackgo.TextBlockObject{
		Type: slackgo.PlainTextType,
		Text: header,
	})

	textHi := slackgo.NewContextBlock("1", &slackgo.TextBlockObject{
		Type: slackgo.MarkdownType,
		Text: fmt.Sprintf("Hi <@%s>!", user),
	})
	textContent := slackgo.NewContextBlock("2", &slackgo.TextBlockObject{
		Type: slackgo.MarkdownType,
		Text: content,
	})
	textButton := &slackgo.TextBlockObject{
		Type:  slackgo.PlainTextType,
		Text:  AppHomeButton,
		Emoji: true,
	}

	button := &slackgo.ButtonBlockElement{
		Type:     slackgo.METButton,
		ActionID: domain.MixpanelEventInstallApp,
		Text:     textButton,
		URL:      domain.InstallationLink,
		Style:    slackgo.StylePrimary,
	}
	action := slackgo.NewActionBlock(domain.MixpanelEventInstallApp, button)

	blocks := slackgo.Blocks{
		BlockSet: []slackgo.Block{
			headerBlock,
			textHi,
			textContent,
		},
	}

	if !authenticated {
		blocks.BlockSet = append(blocks.BlockSet, action)
	}

	return blocks
}

func (s *SlackService) UpdateUnfurlPayload(URL string, currentPayload map[string]slackgo.Attachment) map[string]slackgo.Attachment {
	textDescription := &slackgo.TextBlockObject{
		Type: slackgo.MarkdownType,
		Text: TextUpdateDescription,
	}
	textButton := &slackgo.TextBlockObject{
		Type:  slackgo.PlainTextType,
		Text:  TextUpdate,
		Emoji: true,
	}

	button := &slackgo.ButtonBlockElement{
		Type:     slackgo.METButton,
		ActionID: EventUpdate,
		Text:     textButton,
		URL:      domain.InstallationLink,
	}
	accessory := slackgo.NewAccessory(button)
	sectionBlock := slackgo.NewSectionBlock(textDescription, nil, accessory)
	currentBlockSet := currentPayload[URL].Blocks.BlockSet

	unfurl := map[string]slackgo.Attachment{
		URL: slackgo.Attachment{
			Blocks: slackgo.Blocks{
				BlockSet: append(currentBlockSet, slackgo.NewDividerBlock(), sectionBlock),
			},
		},
	}

	return unfurl
}

func (s *SlackService) ErrorUnfurlPayload(URL string, err error) map[string]slackgo.Attachment {
	errorMessage := ErrorDefault
	if strings.Contains(err.Error(), "not found") {
		errorMessage = ErrorNotFound
	}

	for _, senderPermission := range [5]string{"user does not have cloudanalytics", "user does not belong to the organization", "sender does not have sufficent", "chart cannot be shared", "cant unfurl the given link on this shared channel"} {
		if strings.Contains(err.Error(), senderPermission) {
			errorMessage = ErrorUserPermissions
		}
	}

	if strings.Contains(err.Error(), "user_not_found") {
		errorMessage = ErrorUserNotFound
	}

	fields := []*slackgo.TextBlockObject{
		&slackgo.TextBlockObject{
			Type: slackgo.MarkdownType,
			Text: fmt.Sprintf("%s: %s", TextPreviewError, errorMessage),
		},
	}

	unfurl := map[string]slackgo.Attachment{
		URL: slackgo.Attachment{
			Blocks: slackgo.Blocks{
				BlockSet: []slackgo.Block{
					slackgo.NewSectionBlock(nil, fields, nil),
				},
			},
		},
	}

	return unfurl
}

func (s *SlackService) LoadingUnfurlPayload(URL string) map[string]slackgo.Attachment {
	title := &slackgo.TextBlockObject{
		Type:  slackgo.PlainTextType,
		Text:  TextLoadingLink,
		Emoji: true,
	}

	unfurl := map[string]slackgo.Attachment{
		URL: slackgo.Attachment{
			Blocks: slackgo.Blocks{
				BlockSet: []slackgo.Block{
					slackgo.NewImageBlock(
						ImageLoadingGif,
						TextLoading,
						"",
						title,
					),
				},
			},
		},
	}

	return unfurl
}
