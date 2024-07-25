package domain

import (
	"fmt"

	slackgo "github.com/slack-go/slack"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
)

// supported chart types to unfurl
type ChartType string

const (
	TypeReport ChartType = "reports"
	TypeBudget ChartType = "budgets"
)

type ShareAudience string

const (
	AudienceChannel   ShareAudience = "channel"
	AudienceWorkspace ShareAudience = "workspace"
)

const (
	ErrorUnfurl string = "unfurl error: "
)

var (
	InvalidLinkError = fmt.Errorf("link has to be either Budget or Report")
)

type UnfurlPayload struct {
	MessageTs  string
	Channel    string
	Unfurl     map[string]slackgo.Attachment
	Ephemeral  []slackgo.Block // used to send private message to the sender user
	TeamID     string
	CustomerID string
	ChartType  ChartType
	ChartID    string
	Email      string
}

type ChartCollaborationPayload struct {
	Role             collab.CollaboratorRole
	Public           bool
	GroupToShareWith ShareAudience
	CancelAction     bool
	CustomerID       string
	ChartType        ChartType
	ChartID          string
	WorkspaceID      string
	ChannelID        string
	ResponseURL      string
	SlackUserID      string
	Ephemeral        []slackgo.Block
}
