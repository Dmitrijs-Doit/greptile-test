package knownissues

import (
	"context"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/health"
)

var sevLevelRegexp = regexp.MustCompile(`Current severity level: (.*?)\n`)

type AWSKnownIssue struct {
	IssueID           string     `json:"issueId" firestore:"issueId"`
	Product           string     `json:"affectedProduct" firestore:"affectedProduct"`
	Platform          string     `json:"platform" firestore:"platform"`
	Title             string     `json:"title" firestore:"title"`
	OutageDescription string     `json:"outageDescription" firestore:"outageDescription"`
	Status            string     `json:"status" firestore:"status"`
	DateTime          time.Time  `json:"dateTime" firestore:"dateTime"`
	Region            string     `json:"region" firestore:"region"`
	AvailabilityZone  *string    `json:"availabilityZone" firestore:"availabilityZone"`
	LastUpdatedTime   *time.Time `json:"lastUpdatedTime" firestore:"lastUpdatedTime"`
	EndTime           *time.Time `json:"endTime" firestore:"endTime"`
	ExposureLevel     string     `json:"exposureLevel" firestore:"exposureLevel"`
}

func (ki AWSKnownIssue) GetIssueID() string {
	return ki.IssueID
}

func (ki AWSKnownIssue) GetDateTime() time.Time {
	return ki.DateTime
}

func (ki AWSKnownIssue) AddOrUpdateKnownIssue(ctx context.Context, knownIssuesCollection *firestore.CollectionRef, bw *firestore.BulkWriter) error {

	docSnaps, err := knownIssuesCollection.
		Where("issueId", "==", ki.IssueID).
		Limit(1).Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	if len(docSnaps) == 0 {
		knownIssueRef := knownIssuesCollection.NewDoc()
		_, err := bw.Create(knownIssueRef, ki)

		return err
	}

	docSnap := docSnaps[0]

	var existingKnownIssue AWSKnownIssue

	if err := docSnap.DataTo(&existingKnownIssue); err != nil {
		return err
	}

	if existingKnownIssue.Status != "archived" {
		updates := []firestore.Update{
			{Path: "status", Value: ki.Status},
			{Path: "outageDescription", Value: ki.OutageDescription},
		}

		_, err := bw.Update(docSnap.Ref, updates)
		return err
	}

	return nil
}

func getAllEventsAccountFilters(events []*health.OrganizationEvent) []*health.EventAccountFilter {
	var eventsAccountFilters []*health.EventAccountFilter
	for _, event := range events {
		eventsAccountFilters = append(eventsAccountFilters,
			&health.EventAccountFilter{EventArn: event.Arn})
	}

	return eventsAccountFilters
}

func getAwsKnownIssueProduct(awsEventService string) string {
	awsEventService = strings.ToLower(awsEventService)
	return strings.Title(awsEventService)
}

func getAwsKnownIssueTitle(awsEventTypeCode string) string {
	awsEventTypeCode = strings.Replace(awsEventTypeCode, "_", " ", -1)
	awsEventTypeCode = strings.ToLower(awsEventTypeCode)

	return strings.Title(awsEventTypeCode)
}

func getAwsKnownIssueLevel(knownIssueDescription string) string {
	if match := sevLevelRegexp.FindStringSubmatch(knownIssueDescription); len(match) > 0 {
		return match[1]
	}

	return ""
}

func (s *Service) UpdateAwsKnownIssues(ctx context.Context) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	// Why is MPA 6 (872035802921) used for this?
	masterPayerAccount, err := s.mpaDAL.GetMasterPayerAccount(ctx, "872035802921")
	if err != nil {
		return err
	}

	creds, err := masterPayerAccount.NewCredentials("")
	if err != nil {
		return err
	}

	session, err := session.NewSession(&aws.Config{
		Region:      aws.String(endpoints.UsEast1RegionID),
		Credentials: creds,
	})
	if err != nil {
		return err
	}

	healthService := health.New(session)

	req, _ := healthService.EnableHealthServiceAccessForOrganizationRequest(&health.EnableHealthServiceAccessForOrganizationInput{})

	if err = req.Send(); err != nil {
		return err
	}

	describeEventsForOrganizationOutput, err := healthService.DescribeEventsForOrganization(&health.DescribeEventsForOrganizationInput{})
	if err != nil {
		return err
	}

	eventsAccountFilters := getAllEventsAccountFilters(describeEventsForOrganizationOutput.Events)

	describeEventsDetailsOutput, err := healthService.DescribeEventDetailsForOrganization(&health.DescribeEventDetailsForOrganizationInput{
		OrganizationEventDetailFilters: eventsAccountFilters,
	})
	if err != nil {
		return err
	}

	bw := fs.BulkWriter(ctx)
	defer bw.End()

	for _, event := range describeEventsDetailsOutput.SuccessfulSet {
		eventDetails := event.Event

		var (
			issueID  string
			region   string
			dateTime time.Time
		)

		if eventDetails.Arn != nil {
			issueID = *eventDetails.Arn
		}

		if eventDetails.Region != nil {
			region = *eventDetails.Region
		}

		if eventDetails.StartTime != nil {
			dateTime = *eventDetails.StartTime
		}

		status := getKnownIssueStatus(*eventDetails.StatusCode)
		product := getAwsKnownIssueProduct(*eventDetails.Service)
		title := getAwsKnownIssueTitle(*eventDetails.EventTypeCode)
		outageDesc := *event.EventDescription.LatestDescription
		exposureLevel := getAwsKnownIssueLevel(outageDesc)

		knownIssue := AWSKnownIssue{
			IssueID:           issueID,
			Product:           product,
			Title:             title,
			Platform:          awsPlatform,
			OutageDescription: outageDesc,
			Status:            status,
			DateTime:          dateTime,
			Region:            region,
			AvailabilityZone:  eventDetails.AvailabilityZone,
			LastUpdatedTime:   eventDetails.LastUpdatedTime,
			EndTime:           eventDetails.EndTime,
			ExposureLevel:     exposureLevel,
		}

		if err := knownIssue.AddOrUpdateKnownIssue(ctx, s.getKnownIssuesCollection(ctx), bw); err != nil {
			l.Errorf("failed adding/updating aws known issue %s with error: %s", err)
		}
	}

	return nil
}
