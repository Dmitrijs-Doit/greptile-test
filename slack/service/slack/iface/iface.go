package iface

import (
	"context"

	"github.com/gin-gonic/gin"
	slackgo "github.com/slack-go/slack"

	firestorePkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/slack/domain"
)

type AuthTestResponse struct {
	Ok                    bool `json:"ok"`                      // token is valid, app is installed
	PrivateChannelsScopes bool `json:"private_channels_scopes"` // app has access to private channels
}

type Slack interface {
	// events subscription
	HandleLinkSharedEvent(ctx context.Context, req *domain.SlackRequest) (*domain.MixpanelProperties, error)
	HandleAppHome(ctx context.Context, req *domain.SlackRequest) error
	HandleUserJoinedSharedChannel(ctx context.Context, event domain.SlackEvent) error
	UpdateChartCollaboration(ctx context.Context, req *domain.ChartCollaborationReq) error

	// workspace
	GetWorkspaceDecrypted(ctx context.Context, customerID string) (*firestorePkg.SlackWorkspace, string, string, string, error)

	// validate
	ValidateRequest(ctx *gin.Context, body []byte, appVerificationToken string) error

	// mixpanel
	ParseMixpanelRequest(ctx *gin.Context) (*domain.MixpanelProperties, error)

	// main - TODO (slack refactor) reorder to different files
	ParseEventSubscriptionRequest(ctx *gin.Context) ([]byte, *domain.SlackRequest, error)
	OAuth2callback(ctx *gin.Context, code, customerID string) (string, *domain.MixpanelProperties, error)

	// channels
	SubscribeSharedChannel(ctx context.Context, customerID, channelID string) (*firestorePkg.SharedChannel, error)
	CreateSlackSharedChannel(ctx *gin.Context) (*firestorePkg.SharedChannel, *domain.MixpanelProperties, error)
	GetChannelInvitation(ctx *gin.Context) (string, error)
	GetCustomerChannels(ctx *gin.Context) ([]*common.SlackChannel, error)
	PostMessages(ctx *gin.Context, messages map[*slackgo.MsgOption][]common.SlackChannel)
	PostOnChannel(ctx *gin.Context, channel common.SlackChannel, blocks *slackgo.MsgOption) error

	// auth
	AuthTest(ctx context.Context, customerID string) (AuthTestResponse, error)
}
