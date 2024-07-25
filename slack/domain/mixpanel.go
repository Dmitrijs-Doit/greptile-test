package domain

// MixpanelProperties - mixpanel properies related to a slack action
type MixpanelProperties struct {
	Event       string                 `json:"event,omitempty"`
	User        string                 `json:"user,omitempty"`
	Email       string                 `json:"email,omitempty"`
	ChannelID   string                 `json:"channelId,omitempty"`
	ChannelName string                 `json:"channelName,omitempty"`
	ChartID     string                 `json:"chartId,omitempty"`
	WorkspaceID string                 `json:"workspaceId,omitempty"`
	Payload     map[string]interface{} `json:"payload,omitempty"`
}

const (
	MixpanelEventInstallApp           string = "slack.app.install"
	MixpanelEventSharedChannelCreated string = "slack.sharedChannel.created"
)
