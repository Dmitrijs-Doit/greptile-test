package service

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsavestandalone/gcp/billing/utils"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/mailer"

	"github.com/slack-go/slack"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
)

// MailNotificationTarget represents an email target.
type MailNotificationTarget struct {
	// To is the mail recipient.
	To string
	// SimpleNotification is a pointer to a mailer.SimpleNotification struct.
	// The body should not be populated as it is a separate argument of SendNotification.
	SimpleNotification *mailer.SimpleNotification
}

// SlackNotificationTarget represents a slack channel target.
type SlackNotificationTarget struct {
	// Channel is the channel to send the notification to in the form "#mychannel".
	Channel string
}

// Notification represents a notification instance.
type Notification struct {
	loggerProvider  logger.Provider
	timeFunc        func() int64
	projectNameFunc func() string
}

// NewNotification returns a new instance of the notification service.
func NewNotification(log logger.Provider) *Notification {
	return &Notification{
		loggerProvider:  log,
		timeFunc:        time.Now().Unix,
		projectNameFunc: utils.GetProjectName,
	}
}

// SendNotification sends one or more paragraphs of text to one or more targets.
// Text data is a slice of strings that are assembled before the notification is sent.
// In order tu support multiple text encodings, the data is formatted in markdown and
// rendered to HTML if needed, like for the mail notifications.
// When sending an email notification a mailer.Simplenotification struct is passed with
// the fields set except Body, which is rendered and assigned by this method.
// Example:
//
//	 notification := service.NewNotification(log)
//		mailNotificationTarget := &service.MailNotificationTarget{
//			To: "doer1@doit-intl.com",
//			SimpleNotification: &mailer.SimpleNotification{
//				Subject:   fmt.Sprintf("URGENT - %s - Flexsave Billing Data Is Not Updated", utils.GetProjectName()),
//				Preheader: fmt.Sprintf("Billing Data is not updated for %s", billingAccountID),
//				CCs:       []string{"doer2@doit-intl.com", "doer3@doit-intl.com"},
//			},
//		}
//
//		slackNotificationTarget := &service.SlackNotificationTarget{
//			Channel: "#mychannel",
//		}
//		data := []string{"Some information here  \n", "Something else. *This is bold.*  \n"}
//		notification.SendNotification(ctx, data, mailNotificationTarget, slackNotificationTarget)
func (n *Notification) SendNotification(ctx context.Context, data []string, targets ...interface{}) {
	logger := n.loggerProvider(ctx)

	for _, target := range targets {
		switch target := target.(type) {
		case *MailNotificationTarget:
			to := target.To
			sn := target.SimpleNotification
			sn.Body = n.assembleEmail(data)
			mailer.SendSimpleNotification(sn, to)
		case *SlackNotificationTarget:
			message := n.assembleSlack(data)
			channel := target.Channel

			if _, err := common.PublishToSlack(ctx, message, channel); err != nil {
				logger.Errorf("unable to publish notification to slack. Caused by %s", err)
			}
		default:
			logger.Errorf("unable to publish notification to slack. Caused by unsupported target type %v", reflect.TypeOf(target))
		}
	}
}

func (n *Notification) assembleEmail(data []string) string {
	var body string

	formatted := n.toHTML(data)

	for _, f := range formatted {
		body = fmt.Sprintf("%s%s", body, f)
	}

	return body
}

func (n *Notification) assembleSlack(data []string) map[string]interface{} {
	fields := []map[string]interface{}{
		{
			"title": "Environment",
			"value": n.projectNameFunc(),
			"short": true,
		},
	}

	for _, d := range data {
		fields = append(fields, map[string]interface{}{
			"type":  slack.MarkdownType,
			"value": d,
		})
		fields = append(fields, map[string]interface{}{
			"type": slack.MarkdownType,
			// Slack's markdown renderer ignores \n, so we use U + 3164 here to add an empty line.
			"value": "ㅤ\n",
		})
	}

	message := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"ts":     n.timeFunc(),
				"color":  "#4CAF50",
				"fields": fields,
			},
		},
	}

	return message
}

func (n *Notification) toHTML(data []string) []string {
	var renderedData []string

	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	for _, d := range data {
		html := markdown.ToHTML([]byte(d), nil, renderer)
		renderedData = append(renderedData, string(html))
	}

	return renderedData
}
