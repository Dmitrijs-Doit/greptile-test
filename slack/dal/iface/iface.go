//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"cloud.google.com/go/firestore"
	slackgo "github.com/slack-go/slack"

	firestorePkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/slack/domain"
	notificationDomain "github.com/doitintl/notificationcenter/domain"
)

type IFirestoreDAL interface {
	GetCustomer(ctx context.Context, customerID string) (*firestore.DocumentRef, *common.Customer, error)
	GetWorkspaceDecrypted(ctx context.Context, workspaceID string) (*firestorePkg.SlackWorkspace, string, string, string, error)        // workspace, ID, decrypted user token, decrypted bot token
	GetCustomerWorkspaceDecrypted(ctx context.Context, customerID string) (*firestorePkg.SlackWorkspace, string, string, string, error) // workspace, ID, decrypted user token, decrypted bot token
	GetCustomerWorkspace(ctx context.Context, customerID string) (*firestorePkg.SlackWorkspace, string, error)                          // workspace, ID
	SetCustomerWorkspace(ctx context.Context, workspaceID string, workspace *firestorePkg.SlackWorkspace) error
	GetDoitEmployee(ctx context.Context, email string) (*firestorePkg.User, error) // trying both domains (doit.com, doit-intl.com)
	GetUser(ctx context.Context, email string) (*firestorePkg.User, error)
	UserHasCloudAnalyticsPermission(ctx context.Context, email string) (bool, error)
	GetSharedChannel(ctx context.Context, channelID string) (*firestorePkg.SharedChannel, error)
	GetCustomerSharedChannel(ctx context.Context, customerID string) (*firestorePkg.SharedChannel, error)
	SetCustomerSharedChannel(ctx context.Context, customerID string, channel *firestorePkg.SharedChannel) error
	CreateNotificationConfig(ctx context.Context, config notificationDomain.NotificationConfig) error
	DeleteCustomerSharedChannel(ctx context.Context, customerID, channelID string) error
}

type ISlackDAL interface {
	// chat
	SendEphemeral(ctx context.Context, channelID, userID string, blocks *slackgo.MsgOption) (string, error)
	SendMessage(ctx context.Context, customerID, channelID string, blocks *slackgo.MsgOption) (string, error)
	SendInternalMessage(ctx context.Context, channelID string, blocks *slackgo.MsgOption) (string, error)
	SendUnfurl(ctx context.Context, unfurlPayload *domain.UnfurlPayload) error
	SendUnfurlWithEphemeral(ctx context.Context, unfurlPayload *domain.UnfurlPayload) error
	SendResponse(ctx context.Context, workspaceID, channelID, responseURL string, blocks []slackgo.Block) error

	// user
	GetUserEmail(ctx context.Context, customerID, userID string) (string, error)
	GetUserByEmail(ctx context.Context, customerID, email string) (*slackgo.User, error)
	GetInternalUserByEmail(ctx context.Context, email string) (*slackgo.User, error)
	GetUser(ctx context.Context, workspaceID, ID string) (*slackgo.User, error)

	// channel
	GetAllChannelMemberEmails(ctx context.Context, workspaceID, channelID string) ([]string, error)
	GetChannelInfo(ctx context.Context, workspaceID, channelID string) (*slackgo.Channel, error)
	GetInternalChannelInfo(channelID string) (*slackgo.Channel, error)
	CreateChannel(channelName string) (*slackgo.Channel, error)
	CreateChannelWithFallback(ctx context.Context, channelName string) (*slackgo.Channel, error)
	GetAllCustomerChannels(ctx context.Context, customerID string) ([]slackgo.Channel, error)
	GetCustomerPrivateChannelsForUser(ctx context.Context, customerID string, userEmail string) ([]slackgo.Channel, error)
	InviteUsersToChannel(channelID string, users ...string) (*slackgo.Channel, error)
	GetChannelMembers(channelID string) ([]string, error)

	// views
	PublishAppHome(ctx context.Context, workspaceID, userID string, viewRequest slackgo.HomeTabViewRequest) error
}
