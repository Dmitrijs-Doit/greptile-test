package dal

import (
	slackgo "github.com/slack-go/slack"

	firestorePkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

func decryptWorkspace(workspace *firestorePkg.SlackWorkspace) (string, string, error) {
	userToken, err := decryptToken(workspace.UserToken)
	if err != nil {
		return "", "", err
	}

	botToken, err := decryptToken(workspace.BotToken)
	if err != nil {
		return "", "", err
	}

	return userToken, botToken, nil
}

func decryptToken(tokenEncrypted []byte) (string, error) {
	tokenDecrypted, err := common.DecryptSymmetric([]byte(tokenEncrypted))
	if err != nil {
		return "", err
	}

	return string(tokenDecrypted), nil
}

func MapSharedToCommonSlackChannel(sharedChannel *firestorePkg.SharedChannel) *common.SlackChannel {
	return &common.SlackChannel{
		Name:       sharedChannel.Name,
		ID:         sharedChannel.ID,
		Shared:     true,
		CustomerID: sharedChannel.Customer.ID,
		Type:       "public",
	}
}

func MapToCommonSlackChannel(channel *slackgo.Channel, workspace *firestorePkg.SlackWorkspace) *common.SlackChannel {
	channelType := "public"
	if channel.IsPrivate { // private channels will not be fetched when using botToken
		channelType = "private"
	}

	return &common.SlackChannel{
		Name:       channel.Name,
		ID:         channel.ID,
		Shared:     false,
		CustomerID: workspace.Customer.ID,
		Type:       channelType,
		Workspace:  workspace.Name,
	}
}
