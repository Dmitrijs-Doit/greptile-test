package slack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	tld "github.com/jpillora/go-tld"
	slackgo "github.com/slack-go/slack"

	sharedFirestore "github.com/doitintl/firestore"
	firestorePkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/slack/dal"
	"github.com/doitintl/hello/scheduled-tasks/slack/domain"
	domainSlack "github.com/doitintl/hello/scheduled-tasks/slack/domain"
	notificationDomain "github.com/doitintl/notificationcenter/domain"
)

func GetSlackSharedChannelsInfo(ctx *gin.Context) {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)

	appChannelsDoc, err := fs.Collection("app").Doc("slack").Get(ctx)
	if err != nil {
		l.Errorf("failed to get slack app channels doc with error: %s", err)
		ctx.AbortWithError(http.StatusInternalServerError, err)

		return
	}

	var fsChannels firestorePkg.Channels

	if err := appChannelsDoc.DataTo(&fsChannels); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	params := make(map[string][]string)
	params["exclude_archived"] = []string{"true"}
	params["limit"] = []string{"9000"}

	respBody, err := Client.Get("conversations.list", params)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	var slackChannels firestorePkg.Channels
	if err := json.Unmarshal(respBody, &slackChannels); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	batch := fs.Batch()
	batchCounter := 0

	for _, channel := range slackChannels.Channels {
		if channel.IsShared {
			domain := strings.Replace(channel.Name, "-", ".", -1)
			u, err := tld.Parse(fmt.Sprintf("https://%s", domain))

			if err == nil {
				// check if domain contains "-"
				if u.Subdomain != "" {
					domain = strings.Replace(channel.Name, fmt.Sprintf("-%s", u.TLD), fmt.Sprintf(".%s", u.TLD), -1)
				}
			}

			docSnaps, err := fs.Collection("customers").Where("primaryDomain", "==", domain).Limit(1).Documents(ctx).GetAll()
			if err != nil {
				ctx.AbortWithError(http.StatusInternalServerError, err)
				continue
			}

			if len(docSnaps) == 0 {
				l.Warningf("customer not found for channel: %s", channel.Name)
				continue
			}

			for _, docSnap := range docSnaps {
				channel.Customer = docSnap.Ref
				paths := []firestore.FieldPath{[]string{"id"}, []string{"name"}, []string{"is_shared"}, []string{"is_archived"}, []string{"num_members"}, []string{"customer"}}

				batch.Set(fs.Collection("customers").Doc(docSnap.Ref.ID).Collection("slackChannel").Doc(channel.ID), channel, firestore.Merge(paths...))

				batchCounter++
			}

			if batchCounter > 300 {
				if _, err := batch.Commit(ctx); err != nil {
					ctx.AbortWithError(http.StatusInternalServerError, err)
				}

				batchCounter = 0
				batch = fs.Batch()
			}
		}
	}

	if batchCounter > 0 {
		if _, err := batch.Commit(ctx); err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
		}
	}
}

// CreateSlackSharedChannel - create and returns channel name + invitation link
func (s *SlackService) CreateSlackSharedChannel(ctx *gin.Context) (*firestorePkg.SharedChannel, *domainSlack.MixpanelProperties, error) {
	logger := s.loggerProvider(ctx)
	userEmail := ctx.GetString("email")

	customerID := ctx.Param("customerID")
	if customerID == "" {
		return nil, nil, errors.New("no customerID")
	}

	_, customer, err := s.firestoreDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return nil, nil, err
	}

	channelName := strings.Replace(customer.PrimaryDomain, ".", "-", -1)
	logger.Info("shared channel name:" + channelName)

	channel, err := s.slackDAL.CreateChannelWithFallback(ctx, channelName)
	if err != nil {
		return nil, nil, err
	}

	logger.Infof("successfully created channel %s with ID %s", channel.Name, channel.ID)

	invitationURL, err := s.generateInvitationURL(ctx, channel.ID, userEmail)
	if err != nil {
		return nil, nil, err
	}

	logger.Info("successfully created invitation link " + invitationURL)

	sharedChannel, err := s.SubscribeSharedChannel(ctx, customerID, channel.ID)
	if err != nil {
		return nil, nil, err
	}

	mixpanelProperties := &domainSlack.MixpanelProperties{
		Event: domainSlack.MixpanelEventSharedChannelCreated,
		Email: userEmail,
		Payload: map[string]interface{}{
			"Slack Channel":    sharedChannel.Name,
			"Slack Channel ID": sharedChannel.ID,
		},
	}

	return sharedChannel, mixpanelProperties, nil
}

// SubscribeSharedChannel - perform last alignments on existing channel and store it in DB
func (s *SlackService) SubscribeSharedChannel(ctx context.Context, customerID, channelID string) (*firestorePkg.SharedChannel, error) {
	channel, err := s.slackDAL.GetInternalChannelInfo(channelID)
	if err != nil {
		return nil, err
	}

	customerRef, customer, err := s.firestoreDAL.GetCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}

	productOnly, err := s.customerTypeDal.IsProductOnlyCustomerType(ctx, customerID)
	if err != nil {
		return nil, err
	}

	if !productOnly {
		if err := s.addRelevantUsersToSharedChannel(ctx, channel.ID, customer); err != nil {
			return nil, err
		}
	}

	if existingChannel, err := s.firestoreDAL.GetCustomerSharedChannel(ctx, customerID); (err != nil && err != sharedFirestore.ErrNotFound) || existingChannel != nil {
		if err != nil {
			return nil, err
		}

		return nil, fmt.Errorf("customer already has shared channel on firestore with ID " + existingChannel.ID)
	}

	sharedChannel := firestorePkg.SharedChannel{
		ID:         channel.ID,
		Name:       channel.Name,
		IsShared:   true,
		IsArchived: false,
		NumMembers: 1,
		Customer:   customerRef,
	}

	if err := s.firestoreDAL.SetCustomerSharedChannel(ctx, customerID, &sharedChannel); err != nil {
		return nil, err
	}

	if err := s.firestoreDAL.CreateNotificationConfig(ctx, notificationDomain.NotificationConfig{
		Name:        channel.Name,
		CustomerRef: customerRef,
		ProviderTarget: map[string][]interface{}{
			string(notificationDomain.SLACK): {
				notificationDomain.SlackTarget{
					CustomerID: customerID,
					ID:         channelID,
					Name:       channel.Name,
					Shared:     true,
				},
			},
		},
		SelectedNotifications: map[string]*notificationDomain.SelectedNotificationSettings{
			strconv.Itoa(int(notificationDomain.NotificationCloudCostAnomalies)): {
				AnomalyAlerts: 2, // >=warning
			},
		},
		SelectedNotificationsKeys: []int{int(notificationDomain.NotificationCloudCostAnomalies)},
		CreatedBy:                 "doit.com",
	}); err != nil {
		return nil, err
	}

	return &sharedChannel, nil
}

// GetChannelInvitation - get a new invitation link for existing slack channel
func (s *SlackService) GetChannelInvitation(ctx *gin.Context) (string, error) {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return "", fmt.Errorf("missing customer id")
	}

	userEmail := ctx.GetString("email")
	if userEmail == "" {
		return "", fmt.Errorf("missing email")
	}

	channel, err := s.firestoreDAL.GetCustomerSharedChannel(ctx, customerID)
	if err != nil {
		if channel == nil {
			return "", fmt.Errorf("no shared channels found for customer %s", customerID)
		}

		return "", err
	}

	if channel.ID == "" {
		return "", fmt.Errorf("no shared channels found for customer %s", customerID) //	todo const
	}

	return s.generateChannelInvitationOrRedirectURL(ctx, channel.ID, userEmail)
}

// generateChannelInvitationOrRedirectURL generates invitation for new members or returns redirect url for existing members
func (s *SlackService) generateChannelInvitationOrRedirectURL(ctx *gin.Context, channelID, email string) (string, error) {
	redirectURL, err := s.getChannelRedirectURL(ctx, channelID, email)
	if err != nil {
		if strings.Contains(err.Error(), domainSlack.ErrorChannelNotFound) || strings.Contains(err.Error(), domainSlack.ErrorNotInChannel) { //	 case where channel is deleted from slack, delete from db
			if err := s.handleDeletedChannel(ctx, channelID); err != nil {
				return "", err
			}
		}

		return "", err
	}

	if redirectURL != "" {
		return redirectURL, nil
	}

	channelInvitation, err := s.generateInvitationURL(ctx, channelID, email)
	if err != nil {
		if strings.Contains(err.Error(), domainSlack.ErrorChannelNotFound) || strings.Contains(err.Error(), domainSlack.ErrorNotInChannel) {
			if err := s.handleDeletedChannel(ctx, channelID); err != nil {
				return "", err
			}
		}

		return "", err
	}

	return channelInvitation, nil
}

// generateInvitationURL returns redirect url for existing members
func (s *SlackService) getChannelRedirectURL(ctx *gin.Context, channelID, email string) (string, error) {
	l := s.loggerProvider(ctx)

	customerID := ctx.Param("customerID")
	if customerID == "" {
		return "", domain.ErrorCustomerID
	}

	user, err := s.slackDAL.GetInternalUserByEmail(ctx, email)
	if err != nil {
		// user does not belong to doitintl slack workspace thus cannot be found using Doitsy
		l.Infof("slack.getChannelRedirectURL failed to use Doitsy bot token (proceeding with workspace token). email: %s, error: %s", email, err)
	}

	if user == nil {
		user, err = s.slackDAL.GetUserByEmail(ctx, customerID, email)
		if err != nil {
			l.Infof("slack.getChannelRedirectURL failed to use workspace token. email: %s, error: %s", email, err)
			return "", nil
		}
	}

	usersInChannel, err := s.slackDAL.GetChannelMembers(channelID)
	if err != nil {
		return "", err
	}

	for _, userInChannel := range usersInChannel {
		if user.ID == userInChannel {
			return fmt.Sprintf(domainSlack.ChannelRedirectURL, channelID), nil
		}
	}

	return "", nil
}

// generateInvitationURL generates invitation for new members
func (s *SlackService) generateInvitationURL(ctx *gin.Context, channelID, email string) (string, error) {
	params := make(map[string][]string)
	params["channel"] = []string{channelID}
	params["emails"] = []string{email}
	params["external_limited"] = []string{"false"}

	respBody, err := Client.Post(ctx, domainSlack.APIInvite, params, nil) // TODO slack api
	if err != nil {
		return "", err
	}

	response := struct {
		OK                    bool   `json:"ok,omitempty"`
		Error                 string `json:"error,omitempty"`
		InviteID              string `json:"invite_id,omitempty"`
		URL                   string `json:"url,omitempty"`
		ConfCode              string `json:"conf_code,omitempty"`
		IsLegacySharedChannel bool   `json:"is_legacy_shared_channel,omitempty"`
	}{}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return "", err
	}

	if !response.OK || response.Error != "" || response.URL == "" {
		return "", fmt.Errorf("could not generate invitation link. error: %s", response.Error)
	}

	return response.URL, nil
}

// handleDeletedChannel wipe out data from firestore for deleted slack channels
func (s *SlackService) handleDeletedChannel(ctx *gin.Context, channelID string) error {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return fmt.Errorf("missing customer id")
	}

	err := s.firestoreDAL.DeleteCustomerSharedChannel(ctx, customerID, channelID)
	if err != nil {
		return err
	}

	s.loggerProvider(ctx).Infof("deleted slackChannel %s for customer %s in firestore", channelID, customerID)

	return nil
}

// addRelevantUsersToSharedChannel by default - each Slack shared channel include Doitsy & Yalla bots, relevant SAM & FSR
func (s *SlackService) addRelevantUsersToSharedChannel(ctx context.Context, channelID string, customer *common.Customer) error {
	logger := s.loggerProvider(ctx)

	usersToAdd := []string{domainSlack.YallaBotUserID, domainSlack.DoitsyBotUserID}

	accountManagers, err := common.GetCustomerAccountManagers(ctx, customer, common.AccountManagerCompanyDoit)
	if err != nil {
		return err
	}

	for _, accountManager := range accountManagers {
		user, err := s.slackDAL.GetInternalUserByEmail(ctx, accountManager.Email)
		if err != nil {
			logger.Errorf("error getting email [%s]: %s\n", accountManager.Email, err)
			continue
		}

		usersToAdd = append(usersToAdd, user.ID)
	}

	for _, userToAdd := range usersToAdd {
		_, err = s.slackDAL.InviteUsersToChannel(channelID, userToAdd)
		if err != nil {
			if err.Error() == domainSlack.ErrorAlreadyInChannel || err.Error() == domainSlack.ErrorCantInviteSelf {
				continue
			}

			return err
		}
	}

	logger.Info("users added to new channel: ", usersToAdd)

	return nil
}

// returns SlackChannel struct for shared channel or nil if none. (for customer with shared slack channel)
func (s *SlackService) getSharedChannel(ctx *gin.Context) (*common.SlackChannel, error) {
	customerID := ctx.Param("customerID")
	if customerID == "" {
		return nil, domainSlack.ErrorCustomerID
	}

	sharedChannel, err := s.firestoreDAL.GetCustomerSharedChannel(ctx, customerID)
	if err != nil {
		if err == sharedFirestore.ErrNotFound {
			return nil, nil
		}

		return nil, err
	}

	return dal.MapSharedToCommonSlackChannel(sharedChannel), nil
}

// getWorkspaceChannels returns SlackChannel structures for each workspace channel or empty list if none. (for customers with installed DoiT International Slack app (AF79TTA7N))
func (s *SlackService) getWorkspaceChannels(ctx *gin.Context) ([]*common.SlackChannel, error) {
	logger := s.loggerProvider(ctx)
	workspaceChannels := make([]*common.SlackChannel, 0)

	customerID := ctx.Param("customerID")
	if customerID == "" {
		return nil, domainSlack.ErrorCustomerID
	}

	workspace, workspaceID, _, _, err := s.firestoreDAL.GetCustomerWorkspaceDecrypted(ctx, customerID)
	if err != nil {
		if err == sharedFirestore.ErrNotFound {
			logger.Infof("cannot find slack workspace token for customer %s. error: %s", customerID, err.Error())
			return workspaceChannels, nil
		}

		return nil, err
	}

	channels, err := s.slackDAL.GetAllCustomerChannels(ctx, customerID)
	if err != nil {
		if err.Error() == domainSlack.ErrorMissingScope.Error() {
			err = s.CheckVersionUpdated(ctx, workspaceID)
		}

		if err != nil && (err.Error() == domainSlack.ErrorAppIsOutdated.Error() || err.Error() == "account_inactive" || err.Error() == "token_revoked") { //	most likely slack app is outdated OR uninstalled
			logger.Infof("cannot use slack workspace token for customer %s. error: %s", customerID, err.Error())
			return workspaceChannels, nil
		}

		return nil, err
	}

	for _, channel := range channels {
		workspaceChannels = append(workspaceChannels, dal.MapToCommonSlackChannel(&channel, workspace))
	}

	userEmail := ctx.GetString("email")
	if userEmail == "" {
		return workspaceChannels, nil
	}

	privateChannels, err := s.slackDAL.GetCustomerPrivateChannelsForUser(ctx, customerID, userEmail)
	if err != nil {
		logger.Infof("cannot get private channels for user %s in customer %s: %s", userEmail, customerID, err)
		return workspaceChannels, nil
	}

	for _, channel := range privateChannels {
		workspaceChannels = append(workspaceChannels, dal.MapToCommonSlackChannel(&channel, workspace))
	}

	return workspaceChannels, nil
}

// GetCustomerChannels - returns list of channels (both shared & workspace)
func (s *SlackService) GetCustomerChannels(ctx *gin.Context) ([]*common.SlackChannel, error) {
	l := s.loggerProvider(ctx)
	customerChannels := make([]*common.SlackChannel, 0)

	if sharedChannel, err := s.getSharedChannel(ctx); sharedChannel != nil || err != nil {
		if err != nil {
			return nil, err
		}

		customerChannels = append(customerChannels, sharedChannel)
	}

	if workspaceChannels, err := s.getWorkspaceChannels(ctx); len(workspaceChannels) > 0 || err != nil {
		if err != nil {
			l.Error(err)
		}

		customerChannels = append(customerChannels, workspaceChannels...)
	}

	return customerChannels, nil
}

// PostMessages on a given channels. messages --> KEY<slack block>:VALUE<channels to post on...> (by DoiT International Slack app (AF79TTA7N))
func (s *SlackService) PostMessages(ctx *gin.Context, messages map[*slackgo.MsgOption][]common.SlackChannel) {
	logger := s.loggerProvider(ctx)

	for blocks, channels := range messages {
		for _, channel := range channels {
			if err := s.PostOnChannel(ctx, channel, blocks); err != nil {
				logger.Error(err)
				continue
			}
		}
	}
}

// PostOnChannel - polymorphic function which posting messages both internally & externally (both A012TR3MK5E & AF79TTA7N apps)
func (s *SlackService) PostOnChannel(ctx *gin.Context, channel common.SlackChannel, blocks *slackgo.MsgOption) error {
	logger := s.loggerProvider(ctx)

	if channel.Shared {
		if _, err := s.slackDAL.SendInternalMessage(ctx, channel.ID, blocks); err != nil {
			if err.Error() == domainSlack.ErrorChannelNotFound {
				logger.Infof("SendInternalMessage error for channel %s. error: %s", channel.ID, err)
				return nil
			}

			return err
		}

		return nil
	}

	if _, err := s.slackDAL.SendMessage(ctx, channel.CustomerID, channel.ID, blocks); err != nil {
		if err.Error() == domainSlack.ErrorChannelNotFound {
			logger.Infof("SendMessage error for channel %s. error: %s", channel.ID, err)
			return nil
		}

		return err
	}

	return nil
}
