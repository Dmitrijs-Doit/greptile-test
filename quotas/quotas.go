package quotas

import (
	"context"
	"errors"
	"log"
	"time"

	"cloud.google.com/go/firestore"

	firestorePkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/slice"
	notificationcenterDomain "github.com/doitintl/notificationcenter/domain"
	notificationcenterClient "github.com/doitintl/notificationcenter/pkg"
)

// EmailLimit ..
type EmailLimit struct {
	Service   string `json:"service"`
	Region    string `json:"region"`
	Status    string `json:"status"`
	Limit     string `json:"limit"`
	AccountID string `json:"accountId"`
}

type QuotaNotificationData struct {
	EmailTo        []string
	EmailCc        []string
	PrimaryDomain  string
	Limits         []EmailLimit
	Platform       string
	Link           string
	Documentation  string
	SlackChannells []notificationcenterClient.Slack
}

// return the cmp-quotas-monitoring channel ID
func getCmpQuotasMonitoringSlackChannelID(ctx context.Context, fs *firestore.Client) (string, error) {
	appChannelsDoc, err := fs.Collection("app").Doc("slack").Get(ctx)
	if err != nil {
		return "", err
	}

	var fsChannels firestorePkg.Channels

	if err := appChannelsDoc.DataTo(&fsChannels); err != nil {
		return "", err
	}

	for _, channel := range fsChannels.Channels {
		if channel.ID == "quotas" {
			return channel.InternalChannel, nil
		}
	}

	// if no channel found, return error
	err = errors.New("quotas channel not found")
	log.Println("error getting cmpQuotasChannelID", err)

	return "", err
}

func getQuotaAccountManagers(ctx context.Context, customerID string, company common.AccountManagerCompany) ([]*common.AccountManager, error) {
	accountManagers, err := common.GetCustomerAccountManagersForCustomerID(ctx, customerID, common.AccountManagerCompanyDoit)
	if err != nil {
		return nil, err
	}

	accountManagersByCompany, err := common.GetCustomerAccountManagersForCustomerID(ctx, customerID, company)
	if err != nil {
		return nil, err
	}

	return append(accountManagers, accountManagersByCompany...), nil
}

func GetQuotaNotificationTargets(ctx context.Context, recipients []notificationcenterDomain.NotificationConfig, customerID string, company common.AccountManagerCompany, fs *firestore.Client) ([]string, []string, []notificationcenterClient.Slack) {
	emailTo := []string{}
	emailCc := []string{}
	slackChannels := []notificationcenterClient.Slack{}

	// adding our cmp-quotas-monitoring slack channel
	cmpQuotasChannelID, err := getCmpQuotasMonitoringSlackChannelID(context.Background(), fs)
	if err == nil && cmpQuotasChannelID != "" {
		slackChannels = append(slackChannels, notificationcenterClient.Slack{
			Channel: cmpQuotasChannelID,
		})
	}

	// adding CCs
	accountManagers, err := getQuotaAccountManagers(ctx, customerID, company)
	if err != nil {
		log.Printf("quota - no account managers: %s", err)
	} else {
		for _, am := range accountManagers {
			if !slice.Contains(emailTo, am.Email) {
				emailCc = append(emailCc, am.Email)
			}
		}
	}

	if !slice.Contains(emailCc, "cmp-quota-alerts@doit-intl.com") {
		emailCc = append(emailCc, "cmp-quota-alerts@doit-intl.com")
	}

	// adding emailTo and slack channels by iterating over the recipients
	for _, recipient := range recipients {
		emailTargetsToAdd := recipient.GetEmailTargets()
		for _, target := range emailTargetsToAdd {
			if !slice.Contains(emailTo, target) {
				emailTo = append(emailTo, target)
			}
		}

		slackTargetsToAdd := recipient.GetSlackTargets()
		for _, target := range slackTargetsToAdd {
			// check if slack channel is already in the list
			exists := false

			for _, channel := range slackChannels {
				if channel.Channel == target.ID {
					exists = true
					break
				}
			}

			if !exists {
				slackChannels = append(slackChannels, notificationcenterClient.Slack{
					Channel:     target.ID,
					AccessToken: target.AccessToken,
				})
			}
		}
	}

	return emailTo, emailCc, slackChannels
}

func CreateQuotaNotification(d QuotaNotificationData) notificationcenterClient.Notification {
	services := ""
	for _, limit := range d.Limits {
		services += limit.Service + " "
	}

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	todayString := today.Format("2006-01-02")
	dateDomain := todayString + "/" + d.PrimaryDomain

	platformTitle := "Project"
	if d.Platform == "AWS" {
		platformTitle = "Account"
	}

	notification := notificationcenterClient.Notification{
		Email:    d.EmailTo,
		Template: "98MW5AJ7P0MXJCN9WPMCN0BES5M8",
		Data: map[string]interface{}{
			"date_domain":    dateDomain,
			"limits":         d.Limits,
			"domain":         d.PrimaryDomain,
			"services_name":  services,
			"platform":       d.Platform,
			"platform_title": platformTitle,
			"link":           d.Link,
			"documentation":  d.Documentation,
		},

		Slack:                   d.SlackChannells,
		SlackDisableLinkPreview: true,
		Mock:                    !common.Production,
	}

	if len(notification.Email) != 0 {
		notification.CC = d.EmailCc
		notification.EmailFrom = notificationcenterClient.NotificationsFrom
	}

	return notification
}
