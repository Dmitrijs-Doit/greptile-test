package slack

import (
	"context"
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	slackChannelProd = "#fsgcp-ops"
	slackChannelDev  = "#fsgcp-ops-dev"
)

type Service struct {
	loggerProvider logger.Provider
}

func NewSlackService(loggerProvider logger.Provider) *Service {
	return &Service{
		loggerProvider: loggerProvider,
	}
}

func (s *Service) PublishEntitlementCancelledMessage(
	ctx context.Context,
	domain string,
	billingAccountID string,
) error {
	l := s.loggerProvider(ctx)

	message := map[string]interface{}{
		"blocks": []map[string]interface{}{
			{
				"type": "section",
				"text": map[string]interface{}{
					"type": "mrkdwn",
					"text": fmt.Sprintf("<!subteam^S04DA2YNXK8> *%s* %s has just opted out from *Flexsave Marketplace*", domain, billingAccountID),
				},
			},
		},
	}

	if _, err := common.PublishToSlack(ctx, message, getSlackChannel()); err != nil {
		l.Errorf("unable to publish notification to slack. Caused by %s", err)
	}

	return nil
}

func getSlackChannel() string {
	if common.Production {
		return slackChannelProd
	}

	return slackChannelDev
}
