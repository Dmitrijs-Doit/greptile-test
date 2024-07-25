package cloudanalytics

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	slackgo "github.com/slack-go/slack"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/firebase"
)

// Slack unfurl payload texts
const (
	TextBoldType         string = "*Type*"
	TextBoldDescription  string = "*Description*"
	TextBoldReport       string = "*Report*"
	TextBoldOwner        string = "*Owner*"
	TextBoldLastModified string = "*Last Modified*"
	TextReportButton     string = ":mag: Open Report"
	TextReport           string = "Report"
	EventReportView      string = "slack.report.view"
	DateFormat           string = "Jan 2, 2006"
)

// GetReport fetches a report from firestore and set an appropriate customer to the report
func (s *CloudAnalyticsService) GetReport(ctx context.Context, customerID, reportID string, presentationModeEnabled bool) (*report.Report, error) {
	reportRef := s.reportDAL.GetRef(ctx, reportID)

	docSnap, err := reportRef.Get(ctx)
	if err != nil {
		return nil, err
	}

	var report report.Report
	if err := docSnap.DataTo(&report); err != nil {
		return nil, err
	}

	if report.Customer != nil {
		// If not a preset report and not a Doit employee and customer is not a demo customer, validate that the customer ids match
		isDoitEmployee, _ := ctx.Value(common.CtxKeys.DoitEmployee).(bool)
		if !isDoitEmployee && report.Customer.ID != customerID && !presentationModeEnabled {
			return nil, errors.New("bad request")
		}
	} else {
		// Set the current customer ref for preset report
		report.Customer = s.customersDAL.GetRef(ctx, customerID)
	}

	report.ID = docSnap.Ref.ID

	return &report, nil
}

// GetReportSlackUnfurl generates a payload for a given report to be unfurled on a report link shared on Slack
// TODO: add s.GetReportImage() after highcharts directory is refactored
func (s *CloudAnalyticsService) GetReportSlackUnfurl(ctx context.Context, reportID, customerID, URL, imageURL string) (*report.Report, map[string]slackgo.Attachment, error) {
	report, err := s.GetReport(ctx, customerID, reportID, false)
	if err != nil {
		return nil, nil, err
	}

	var reportOwner string

	for _, collaborator := range report.Collaborators {
		if collaborator.Role == collab.CollaboratorRoleOwner {
			reportOwner = collaborator.Email
		}
	}

	fields := []*slackgo.TextBlockObject{
		{
			Type: slackgo.MarkdownType,
			Text: fmt.Sprintf("%s: %s", TextBoldReport, report.Name),
		},
		{
			Type: slackgo.MarkdownType,
			Text: fmt.Sprintf("%s: %s", TextBoldOwner, reportOwner),
		},
		{
			Type: slackgo.MarkdownType,
			Text: fmt.Sprintf("%s: %s", TextBoldLastModified, report.TimeModified.Format(DateFormat)),
		},
		{

			Type: slackgo.MarkdownType,
			Text: fmt.Sprintf("%s: %s", TextBoldType, strings.Title(report.Type)),
		},
	}

	textDescription := &slackgo.TextBlockObject{
		Type: slackgo.MarkdownType,
		Text: fmt.Sprintf("%s: %s", TextBoldDescription, report.Description),
	}
	textImage := &slackgo.TextBlockObject{
		Type:  slackgo.PlainTextType,
		Text:  report.Name,
		Emoji: true,
	}
	textButton := &slackgo.TextBlockObject{
		Type:  slackgo.PlainTextType,
		Text:  TextReportButton,
		Emoji: true,
	}
	button := &slackgo.ButtonBlockElement{
		Type:     slackgo.METButton,
		ActionID: EventReportView,
		Text:     textButton,
		URL:      URL,
	}

	sectionBlock := slackgo.NewSectionBlock(nil, fields, nil)
	imageBlock := slackgo.NewImageBlock(
		imageURL,
		TextReport,
		"",
		textImage,
	)
	actionBlock := slackgo.NewActionBlock("", button)

	var blockSet []slackgo.Block
	if report.Description == "" {
		blockSet = []slackgo.Block{
			sectionBlock,
			imageBlock,
			actionBlock,
		}
	} else {
		blockSet = []slackgo.Block{
			sectionBlock,
			slackgo.NewSectionBlock(textDescription, nil, nil),
			imageBlock,
			actionBlock,
		}
	}

	unfurl := map[string]slackgo.Attachment{
		URL: {
			Blocks: slackgo.Blocks{
				BlockSet: blockSet,
			},
		},
	}

	return report, unfurl, nil
}

// DeleteStaleDraftReports deletes draft reports last modified more than 12 hours ago
func (s *CloudAnalyticsService) DeleteStaleDraftReports(ctx context.Context) error {
	fs := s.conn.Firestore(ctx)

	draftSnaps, err := fs.Collection("dashboards").
		Doc("google-cloud-reports").
		Collection("savedReports").
		Where("draft", "==", true).
		Where("timeModified", "<", time.Now().Add(-12*time.Hour)).
		Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	wb := firebase.NewAutomaticWriteBatch(fs, 500)
	for _, draftSnap := range draftSnaps {
		wb.Delete(draftSnap.Ref)
	}

	if errs := wb.Commit(ctx); len(errs) > 0 {
		return errs[0]
	}

	return nil
}
