package sl

import (
	"context"
	"fmt"

	"github.com/slack-go/slack"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

const nonBillingReportSlackChanel = "C03PMMV26E4" //"#spot0-notifications"
const maxSlackBlocks = 44

type NonBillingTagsRow struct {
	PrimaryDomain string
	MPAAccount    string
	Status        string
}

type Spot0Slack struct{}

type ISlack interface {
	PublishToSlack(ctx context.Context, reportData []*NonBillingTagsRow) (*string, error)
	AssembleNonBillingTagsSlackMessage(reportData []*NonBillingTagsRow) (map[string]interface{}, *string)
}

func (s *Spot0Slack) PublishToSlack(ctx context.Context, reportData []*NonBillingTagsRow) (*string, error) {
	message, extra := s.AssembleNonBillingTagsSlackMessage(reportData)
	_, err := common.PublishToSlack(ctx, message, nonBillingReportSlackChanel)

	return extra, err
}

func (s *Spot0Slack) AssembleNonBillingTagsSlackMessage(reportData []*NonBillingTagsRow) (map[string]interface{}, *string) {
	var slackBlocks []slack.Block

	slackHeaderBlocks := []slack.Block{
		slack.NewHeaderBlock(
			&slack.TextBlockObject{
				Type: slack.PlainTextType,
				Text: fmt.Sprintf(`Customers with missing billing information:`),
			},
		),
		slack.NewDividerBlock(),
		slack.NewSectionBlock(nil, []*slack.TextBlockObject{
			{
				Type: slack.MarkdownType,
				Text: "*Primary Domain:*(MPA account)",
			},
			{
				Type: slack.MarkdownType,
				Text: "Status",
			},
		}, nil),
	}

	var count int

	var extraAccounts string

	for _, nonBillingTagsRow := range reportData {
		sectionBlock := slack.NewSectionBlock(nil, []*slack.TextBlockObject{
			{
				Type: slack.MarkdownType,
				Text: fmt.Sprintf("%s(%s)", nonBillingTagsRow.PrimaryDomain, nonBillingTagsRow.MPAAccount),
			},
			{
				Type: slack.MarkdownType,
				Text: nonBillingTagsRow.Status,
			},
		}, nil)

		count++

		if count < maxSlackBlocks {
			slackBlocks = append(slackBlocks, sectionBlock)
		}

		extraAccounts += fmt.Sprintf("%s, %s \t %s\n", nonBillingTagsRow.PrimaryDomain, nonBillingTagsRow.MPAAccount, nonBillingTagsRow.Status)
	}

	attachment := slack.Attachment{
		Blocks: slack.Blocks{
			BlockSet: slackBlocks,
		},
	}
	attachments := []slack.Attachment{
		attachment,
	}

	if count >= maxSlackBlocks {
		allAccountsText := slack.Attachment{
			Text: extraAccounts,
		}
		attachments = append(attachments, allAccountsText)
	}

	message := map[string]interface{}{
		"blocks":      slackHeaderBlocks,
		"attachments": attachments,
	}

	return message, &extraAccounts
}
