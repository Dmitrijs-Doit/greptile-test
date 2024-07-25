package domain

import "errors"

const (
	// slack errors
	ErrorNotInChannel     string = "not_in_channel"
	ErrorChannelNotFound  string = "channel_not_found"
	ErrorAlreadyInChannel string = "already_in_channel"
	ErrorCantInviteSelf   string = "cant_invite_self"
	ErrorNameTaken        string = "name_taken"

	// slack constants
	APIInvite          string = "conversations.inviteShared"
	ChannelRedirectURL string = "https://slack.com/app_redirect?channel=%s"
	UserToken          string = "userToken"
	BotToken           string = "botToken"

	// IDs
	DoitsyBotUserID   string = "U012A5B4S3E"
	YallaBotUserID    string = "U017R0QBL1K"
	WorkspaceDoit     string = "T2TG0KM5E"
	WorkspaceBudgetao string = "T02AAND1P0A"
	LatestAppScope    string = "chat:write.public" // note - should be updated with each scope update (pay attention to Bot or User token type)
	// InstallationLink - latest installation link (with the most updated scopes)
	InstallationLink string = "https://slack.com/oauth/v2/authorize?client_id=95544667184.517333928260&scope=channels:read,chat:write,chat:write.public,links:read,links:write&user_scope=channels:read,groups:read,links:read,links:write,mpim:read,users:read,users:read.email"

	// errors
	UserNotFound          string = "no user exists with the email:"
	ErrorUnsupportedEvent string = "unsupported event received: %s"
)

var (
	// ErrorMissingScope - mostly given by outdated app version
	ErrorMissingScope = errors.New("missing_scope")
	// ErrorAppIsOutdated - indicating that an update action is required
	ErrorAppIsOutdated = errors.New("slack app version installed on the given workspace is not updated")

	ErrorCustomerID = errors.New("cannot get customerID")
)
