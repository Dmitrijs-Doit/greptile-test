package domain

type SlackRequest struct {
	Challenge string     `json:"challenge"`
	Token     string     `json:"token"`
	TeamID    string     `json:"team_id"`
	Event     SlackEvent `json:"event"`
}

type SlackEvent struct {
	Type      SlackEventType `json:"type"`
	Channel   string         `json:"channel"`
	User      string         `json:"user"`
	MessageTs string         `json:"message_ts"`
	Links     []SlackLink    `json:"links"`
	Tab       string         `json:"tab"`
}

type SlackEventType string

const (
	EventLinkShared   SlackEventType = "link_shared"
	EventHomeOpened   SlackEventType = "app_home_opened"
	EventMemberJoined SlackEventType = "member_joined_channel"
	EventChallenge    SlackEventType = "challenge"
)

type SlackLink struct {
	URL string `json:"url"`
}

// ChartCollaborationValue collaboration action that should be done upon the given chart
type ChartCollaborationValue string

const (
	ChannelViewer   ChartCollaborationValue = "channel_viewer"
	ChannelEditor   ChartCollaborationValue = "channel_editor"
	WorkspaceViewer ChartCollaborationValue = "workspace_viewer"
	WorkspaceEditor ChartCollaborationValue = "workspace_editor"
	CancelAction    ChartCollaborationValue = "cancel"
)

type ChartCollaborationReq struct {
	UnfurlURL   string                  `json:"unfurlUrl"`
	ResponseURL string                  `json:"responseUrl"`
	WorkspaceID string                  `json:"workspaceId"`
	ChannelID   string                  `json:"channelId"`
	Value       ChartCollaborationValue `json:"value"`
	UserID      string                  `json:"userId"`
}
