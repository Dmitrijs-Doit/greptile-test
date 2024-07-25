package sl

import (
	"testing"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

func TestNewSlackService(t *testing.T) {
	var slSvc Spot0Slack

	var row1 = &NonBillingTagsRow{"doit.com", "1234", "NRA"}

	var row2 = &NonBillingTagsRow{"doit-intl.com", "2345", "Updated"}

	reportData := []*NonBillingTagsRow{
		row1, row2,
	}

	message, _ := slSvc.AssembleNonBillingTagsSlackMessage(reportData)

	attachments := message["attachments"].([]slack.Attachment)
	blocks := attachments[0].Blocks.BlockSet
	assert.Equal(t, 2, len(blocks))
	assert.Equal(t, "doit.com(1234)", blocks[0].(*slack.SectionBlock).Fields[0].Text)
	assert.Equal(t, "NRA", blocks[0].(*slack.SectionBlock).Fields[1].Text)
	assert.Equal(t, "doit-intl.com(2345)", blocks[1].(*slack.SectionBlock).Fields[0].Text)
	assert.Equal(t, "Updated", blocks[1].(*slack.SectionBlock).Fields[1].Text)
}
