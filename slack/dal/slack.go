package dal

import (
	"context"
	"encoding/json"
	"strings"

	slackgo "github.com/slack-go/slack"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/slack/dal/iface"
	"github.com/doitintl/hello/scheduled-tasks/slack/domain"
	"github.com/doitintl/slackapi"
)

/*
	SlackDAL

Data Access Layer responsible for communication with slack API when additional steps (such as obtaining token) are required
*/
type SlackDAL struct {
	API            *slackapi.SlackAPI
	firestore      iface.IFirestoreDAL
	loggerProvider logger.Provider
}

func NewSlackDAL(ctx context.Context, log logger.Provider, conn *connection.Connection, project string) (*SlackDAL, error) {
	api, err := slackapi.NewSlackAPI(ctx, project)
	if err != nil {
		return nil, err
	}

	firestore := NewFirestoreDAL(ctx, conn)

	return NewSlackDALWithClient(api, firestore, log), nil
}

func NewSlackDALWithClient(api *slackapi.SlackAPI, firestore iface.IFirestoreDAL, log logger.Provider) *SlackDAL {
	return &SlackDAL{api, firestore, log}
}

func (d *SlackDAL) SendEphemeral(ctx context.Context, channelID, userID string, blocks *slackgo.MsgOption) (string, error) {
	threadTS, err := d.API.DoitsyClient.SendEphemeralMessage(channelID, userID, *blocks)
	if err != nil {
		return "", err
	}

	d.loggerProvider(ctx).Infof("ephemeral message posted on channel %s on %s", channelID, threadTS)

	return threadTS, nil
}

func (d *SlackDAL) SendInternalMessage(ctx context.Context, channelID string, blocks *slackgo.MsgOption) (string, error) {
	threadTS, err := d.API.DoitsyClient.SendMessage(channelID, *blocks)
	if err != nil {
		return "", err
	}

	d.loggerProvider(ctx).Infof("message posted on internal channel %s on %s", channelID, threadTS)

	return threadTS, nil
}

func (d *SlackDAL) SendMessage(ctx context.Context, customerID, channelID string, blocks *slackgo.MsgOption) (string, error) {
	_, _, _, botToken, err := d.firestore.GetCustomerWorkspaceDecrypted(ctx, customerID)
	if err != nil {
		return "", err
	}

	threadTS, err := d.API.Client.WithToken(botToken).SendMessage(channelID, *blocks)
	if err != nil {
		return "", err
	}

	d.loggerProvider(ctx).Infof("message posted for customer %s on channel %s on %s", customerID, channelID, threadTS)

	return threadTS, nil
}

func (d *SlackDAL) SendUnfurl(ctx context.Context, unfurlPayload *domain.UnfurlPayload) error {
	logger := d.loggerProvider(ctx)

	unfurlStr, err := json.Marshal(unfurlPayload.Unfurl)
	if err != nil {
		return err
	}

	logger.Infof("unfurl request payload: %s", string(unfurlStr))

	_, _, userToken, _, err := d.firestore.GetWorkspaceDecrypted(ctx, unfurlPayload.TeamID)
	if err != nil {
		return err
	}

	if err := d.API.Client.WithToken(userToken).Unfurl(unfurlPayload.Channel, unfurlPayload.MessageTs, unfurlPayload.Unfurl); err != nil {
		return err
	}

	logger.Infof("unfurl request success")

	return nil
}

func (d *SlackDAL) SendUnfurlWithEphemeral(ctx context.Context, unfurlPayload *domain.UnfurlPayload) error {
	_, _, userToken, _, err := d.firestore.GetWorkspaceDecrypted(ctx, unfurlPayload.TeamID)
	if err != nil {
		return err
	}

	if err := d.API.Client.WithToken(userToken).UnfurlWithEphemeral(unfurlPayload.Channel, unfurlPayload.MessageTs, unfurlPayload.Unfurl, unfurlPayload.Ephemeral); err != nil {
		return err
	}

	d.loggerProvider(ctx).Infof("unfurlWithEphemeral request success")

	return nil
}

func (d *SlackDAL) SendResponse(ctx context.Context, workspaceID, channelID, responseURL string, blocks []slackgo.Block) error {
	_, _, userToken, _, err := d.firestore.GetWorkspaceDecrypted(ctx, workspaceID)
	if err != nil {
		return err
	}

	if err := d.API.Client.WithToken(userToken).SendResponseAndReplace(channelID, responseURL, blocks); err != nil {
		return err
	}

	d.loggerProvider(ctx).Infof("sendResponse request success")

	return nil
}

func (d *SlackDAL) GetUserEmail(ctx context.Context, workspaceID, userID string) (string, error) {
	logger := d.loggerProvider(ctx)

	_, _, userToken, _, err := d.firestore.GetWorkspaceDecrypted(ctx, workspaceID)
	if err != nil {
		return "", err
	}

	user, err := d.API.Client.WithToken(userToken).GetUser(userID)
	if err != nil {
		return "", err
	}

	logger.Infof("User ID: %s, FullName: %s, Email: %s", user.ID, user.Profile.RealName, user.Profile.Email)

	return user.Profile.Email, nil
}

func (d *SlackDAL) GetUserByEmail(ctx context.Context, customerID, email string) (*slackgo.User, error) {
	_, _, userToken, _, err := d.firestore.GetCustomerWorkspaceDecrypted(ctx, customerID)
	if err != nil {
		return nil, err
	}

	return d.API.Client.WithToken(userToken).GetUserByEmail(email)
}

func (d *SlackDAL) GetInternalUserByEmail(ctx context.Context, email string) (*slackgo.User, error) {
	return d.API.DoitsyClient.GetUserByEmail(email)
}

func (d *SlackDAL) GetUser(ctx context.Context, customerID, ID string) (*slackgo.User, error) {
	_, _, userToken, _, err := d.firestore.GetCustomerWorkspaceDecrypted(ctx, customerID)
	if err != nil {
		return nil, err
	}

	return d.API.Client.WithToken(userToken).GetUser(ID)
}

func (d *SlackDAL) GetAllChannelMemberEmails(ctx context.Context, workspaceID, channelID string) ([]string, error) {
	_, _, userToken, botToken, err := d.firestore.GetWorkspaceDecrypted(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	members, err := d.API.Client.WithToken(userToken).GetAllChannelMemberEmails(channelID)
	if err == nil {
		return members, nil
	}

	d.loggerProvider(ctx).Infof("failed to fetch channel members using userToken, trying with botToken; %s", err)

	return d.API.Client.WithToken(botToken).GetAllChannelMemberEmails(channelID)
}

func (d *SlackDAL) GetChannelInfo(ctx context.Context, workspaceID, channelID string) (*slackgo.Channel, error) {
	_, _, userToken, _, err := d.firestore.GetWorkspaceDecrypted(ctx, workspaceID)
	if err != nil {
		return nil, err
	}

	getChannelFun := d.API.Client.WithToken(userToken).GetChannel

	return getChannel(getChannelFun, channelID)
}

func (d *SlackDAL) GetInternalChannelInfo(channelID string) (*slackgo.Channel, error) {
	getChannelFun := d.API.DoitsyClient.GetChannel
	return getChannel(getChannelFun, channelID)
}

func getChannel(getChannelFunc func(string) (*slackgo.Channel, error), channelID string) (*slackgo.Channel, error) {
	channel, err := getChannelFunc(channelID)
	if err != nil {
		// not reliable
		if strings.HasPrefix(channelID, "D") { //	direct message channel ID's always starts with D and has no unique name - thus can be handled with no `im:read` scope
			return &slackgo.Channel{GroupConversation: slackgo.GroupConversation{Name: "directmessage"}}, nil
		}

		return nil, err
	}

	return channel, nil
}

func (d *SlackDAL) CreateChannel(channelName string) (*slackgo.Channel, error) {
	return d.API.DoitsyClient.CreateChannel(channelName)
}

func (d *SlackDAL) CreateChannelWithFallback(ctx context.Context, channelName string) (*slackgo.Channel, error) {
	channel, err := d.CreateChannel(channelName)
	if err != nil && err.Error() == domain.ErrorNameTaken {
		d.loggerProvider(ctx).Infof("channel %s already exist, using fallback - [%s1]", channelName, channelName)
		channelName += "1"
		channel, err = d.CreateChannel(channelName)
	}

	return channel, err
}

func (d *SlackDAL) GetAllCustomerChannels(ctx context.Context, customerID string) ([]slackgo.Channel, error) {
	_, _, _, botToken, err := d.firestore.GetCustomerWorkspaceDecrypted(ctx, customerID)
	if err != nil {
		return nil, err
	}

	return d.API.Client.WithToken(botToken).GetAllChannels([]string{"public_channel"})
}

// GetCustomerPrivateChannels returns the private channels the bot is in.
// Filters out channels that the user is not a member of.
func (d *SlackDAL) GetCustomerPrivateChannelsForUser(ctx context.Context, customerID string, userEmail string) ([]slackgo.Channel, error) {
	_, _, _, botToken, err := d.firestore.GetCustomerWorkspaceDecrypted(ctx, customerID)
	if err != nil {
		return nil, err
	}

	channels, err := d.API.Client.WithToken(botToken).GetAllChannels([]string{"private_channel"})
	if err != nil {
		return nil, err
	}

	userChannels := []slackgo.Channel{}

	for _, channel := range channels {
		emails, err := d.API.Client.WithToken(botToken).GetAllChannelMemberEmails(channel.ID)
		if err != nil {
			return nil, err
		}

		for _, email := range emails {
			if email == userEmail {
				userChannels = append(userChannels, channel)
				break
			}
		}
	}

	return userChannels, err
}

func (d *SlackDAL) InviteUsersToChannel(channelID string, users ...string) (*slackgo.Channel, error) {
	return d.API.DoitsyClient.InviteUsersToChannel(channelID, users...)
}

func (d *SlackDAL) GetChannelMembers(channelID string) ([]string, error) {
	return d.API.DoitsyClient.GetAllChannelMembers(channelID)
}

func (d *SlackDAL) PublishAppHome(ctx context.Context, workspaceID, userID string, viewRequest slackgo.HomeTabViewRequest) error {
	_, botToken, _, _, err := d.firestore.GetWorkspaceDecrypted(ctx, workspaceID)
	if err != nil {
		return err
	}

	return d.API.Client.WithToken(botToken).PublishAppHome(userID, viewRequest)
}
