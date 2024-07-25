package slack

import (
	"context"
	"fmt"

	slackgo "github.com/slack-go/slack"

	sharedFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	domainSlack "github.com/doitintl/hello/scheduled-tasks/slack/domain"
)

func (s *SlackService) HandleUserJoinedSharedChannel(ctx context.Context, event domainSlack.SlackEvent) error {
	l := s.loggerProvider(ctx)

	channel, err := s.firestoreDAL.GetSharedChannel(ctx, event.Channel)
	if err != nil {
		if err == sharedFirestore.ErrNotFound {
			l.Warningf("slack channel [%s] is not shared, aborting welcome message flow", event.Channel)
			return nil
		}

		return err
	}

	customerID := channel.Customer.ID
	URLs := map[string]string{
		"settings":     fmt.Sprintf("https://%s/customers/%s/integrations/slack", common.Domain, customerID),
		"incidents":    fmt.Sprintf("https://%s/customers/%s/known-issues", common.Domain, customerID),
		"tickets":      fmt.Sprintf("https://%s/customers/%s/support", common.Domain, customerID),
		"createTicket": fmt.Sprintf("https://%s/customers/%s/support/new", common.Domain, customerID),
	}
	payload := s.WelcomeToChannelPayload(URLs)

	_, err = s.slackDAL.SendEphemeral(ctx, event.Channel, event.User, &payload)

	return err
}

// HandleAppHome - push relevant payload to Home tab on DoiT International Slack app (AF79TTA7N)
func (s *SlackService) HandleAppHome(ctx context.Context, req *domainSlack.SlackRequest) error {
	authenticated, customerID, err := s.isCustomer(ctx, req.TeamID)
	if err != nil {
		return err
	}

	blocks := s.AppHomePayload(authenticated, customerID, req.Event.User)

	return s.slackDAL.PublishAppHome(ctx, req.TeamID, req.Event.User, slackgo.HomeTabViewRequest{
		Type:   slackgo.VTHomeTab,
		Blocks: blocks,
	})
}

// HandleLinkSharedEvent -  link_shared event received by DoiT International Slack app (AF79TTA7N)
func (s *SlackService) HandleLinkSharedEvent(ctx context.Context, req *domainSlack.SlackRequest) (*domainSlack.MixpanelProperties, error) {
	if len(req.Event.Links) < 1 || req.Event.Links[0].URL == "" {
		return nil, fmt.Errorf("no link was received")
	}

	URL := req.Event.Links[0].URL

	unfurlPayload, err := s.generateUnfurlPayload(ctx, req)
	if err != nil {
		return nil, s.handleError(ctx, unfurlPayload, URL, err)
	}

	if unfurlPayload == nil {
		return nil, nil
	}

	unfurlPayload.Unfurl = s.LoadingUnfurlPayload(URL)
	if err := s.slackDAL.SendUnfurl(ctx, unfurlPayload); err != nil {
		return nil, s.handleError(ctx, unfurlPayload, URL, err)
	}

	var (
		chart    interface{}
		chartErr error
	)

	switch unfurlPayload.ChartType {
	case domainSlack.TypeBudget:
		chart, unfurlPayload.Unfurl, chartErr = s.budgetsService.GetUnfurlPayload(ctx, unfurlPayload.ChartID, unfurlPayload.CustomerID, URL)
	case domainSlack.TypeReport:
		chart, unfurlPayload.Unfurl, chartErr = s.reportsService.GetUnfurlPayload(ctx, unfurlPayload.ChartID, unfurlPayload.CustomerID, URL)
	}

	if chartErr != nil {
		return nil, s.handleError(ctx, unfurlPayload, URL, chartErr)
	}

	if err = s.ValidateUnfurlPermissions(ctx, unfurlPayload, chart); err != nil {
		return nil, s.handleError(ctx, unfurlPayload, URL, err)
	}

	if err := s.slackDAL.SendUnfurl(ctx, unfurlPayload); err != nil {
		return nil, err
	}

	if err := s.promptChartCollaboration(ctx, chart, unfurlPayload, URL); err != nil {
		return nil, err
	}

	return s.GenerateUnfurlMixpanelPayload(ctx, unfurlPayload, URL)
}
