package notification

import (
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/slack-go/slack"
)

func TestToHTML(t *testing.T) {
	testData := []struct {
		name string
		data []string
		want []string
	}{
		{
			name: "Markdown becomes HTML",
			data: []string{"A bit of **markdown** with a [link](https://www.example.com).  \n`   `  \n`  `"},
			want: []string{"<p>A bit of <strong>markdown</strong> with a <a href=\"https://www.example.com\" target=\"_blank\">link</a>.<br>\n<br>\n</p>\n"},
		},
	}

	for _, test := range testData {
		t.Run(test.name, func(t *testing.T) {
			n := &Notification{}

			got := n.toHTML(test.data)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("toHTML() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAssembleEmail(t *testing.T) {
	testData := []struct {
		name string
		data []string
		want string
	}{
		{
			name: "Correctly formatted email body",
			data: []string{"Billing account **12345678**  \n", "Mising rows in local table  \n"},
			want: "<p>Billing account <strong>12345678</strong></p>\n<p>Mising rows in local table</p>\n",
		},
	}

	for _, test := range testData {
		t.Run(test.name, func(t *testing.T) {
			n := &Notification{}

			got := n.assembleEmail(test.data)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("assembleEmail() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestAssembleSlack(t *testing.T) {
	testData := []struct {
		name string
		data []string
		want map[string]interface{}
	}{
		{
			name: "Correctly formatted slack message",
			data: []string{"Billing account **12345678**  \n", "Mising rows in local table  \n"},
			want: map[string]interface{}{
				"attachments": []map[string]interface{}{
					{
						"color": SeverityMediumColor,
						"fields": []map[string]interface{}{
							{"short": bool(true), "title": string("Environment"), "value": string("TEST")},
							{"type": string(slack.MarkdownType), "value": string("Billing account **12345678**  \n")},
							{
								"type": slack.MarkdownType,
								// Slack's markdown renderer ignores \n, so we use U + 3164 here to add an empty line.
								"value": "ㅤ\n",
							},
							{"type": string(slack.MarkdownType), "value": string("Mising rows in local table  \n")},
							{
								"type": slack.MarkdownType,
								// Slack's markdown renderer ignores \n, so we use U + 3164 here to add an empty line.
								"value": "ㅤ\n",
							},
						},
						"ts": int64(12345),
					},
				},
			},
		},
	}

	for _, test := range testData {
		t.Run(test.name, func(t *testing.T) {
			n := &Notification{
				timeFunc:        func() int64 { return 12345 },
				projectNameFunc: func() string { return "TEST" },
			}

			got := n.assembleSlack(test.data, SeverityMedium)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("assembleSlack() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
