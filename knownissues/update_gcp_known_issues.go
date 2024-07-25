package knownissues

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
	"google.golang.org/api/gmail/v1"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/knownissues/iface"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
)

type messageAttachment struct {
	TrackingID     string    `json:"tracking_id"`
	UpdateMessage  string    `json:"update_message"`
	UpdateTime     time.Time `json:"update_time"`
	Products       []Product `json:"products"`
	State          string    `json:"state"`
	Summary        string    `json:"summary"`
	Symptoms       string    `json:"symptoms"`
	Workaround     string    `json:"woraround"`
	NextUpdateTime time.Time `json:"next_update_time"`
	ExposureLevel  string    `json:"exposure_level"`
	Locations      []string  `json:"locations"`
}

type GCPKnownIssue struct {
	IssueID           string     `json:"issueId" firestore:"issueId"`
	Products          []string   `json:"products" firestore:"products"`
	Product           string     `json:"affectedProduct" firestore:"affectedProduct"`
	Platform          string     `json:"platform" firestore:"platform"`
	Title             string     `json:"title" firestore:"title"`
	OutageDescription string     `json:"outageDescription" firestore:"outageDescription"`
	Status            string     `json:"status" firestore:"status"`
	DateTime          time.Time  `json:"dateTime" firestore:"dateTime"`
	Summary           string     `json:"summary" firestore:"summary"`
	Symptoms          string     `json:"symptoms" firestore:"symptoms"`
	Workaround        string     `json:"workaround" firestore:"workaround"`
	NextUpdateTime    *time.Time `json:"nextUpdateTime" firestore:"nextUpdateTime"`
	ExposureLevel     string     `json:"exposureLevel" firestore:"exposureLevel"`
	Locations         []string   `json:"locations" firestore:"locations"`
}

type Product struct {
	Name      string   `json:"name" firestore:"name"`
	Locations []string `json:"locations" firestore:"locations"`
}

func (ki GCPKnownIssue) GetIssueID() string {
	return ki.IssueID
}

func (ki GCPKnownIssue) GetDateTime() time.Time {
	return ki.DateTime
}

func (ki GCPKnownIssue) AddOrUpdateKnownIssue(ctx context.Context, knownIssuesCollection *firestore.CollectionRef, bw *firestore.BulkWriter) error {
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

	var existingKnownIssue GCPKnownIssue

	if err := docSnap.DataTo(&existingKnownIssue); err != nil {
		return err
	}

	if existingKnownIssue.Status != "archived" {
		updates := []firestore.Update{
			{Path: "status", Value: ki.Status},
			{Path: "outageDescription", Value: ki.OutageDescription},
			{Path: "title", Value: ki.Title},
		}

		if ki.NextUpdateTime != nil {
			updates = append(updates, firestore.Update{
				Path: "nextUpdateTime", Value: ki.NextUpdateTime},
			)
		}

		_, err := bw.Update(docSnap.Ref, updates)
		return err
	}

	return nil
}

func getGmailService(ctx context.Context, gmailconf *jwt.Config) (*gmail.Service, error) {
	gmailconf.Subject = common.GetEnv("GCP_KNOWN_ISSUES_GMAIL", defaultGcpKnownIssuesGmail)
	return gmail.New(gmailconf.Client(ctx))
}

func getGcpMessageSubject(message *gmail.Message) string {
	messageSubject := ""

	for _, header := range message.Payload.Headers {
		if header.Name == "Subject" {
			messageSubject = strings.Replace(header.Value, "[Confidential] ", "", -1)
		}
	}

	return messageSubject
}

func getKnownIssueNextUpdateTime(nextUpdateTime time.Time) *time.Time {
	if nextUpdateTime.IsZero() {
		return nil
	}

	return &nextUpdateTime
}

func (ki GCPKnownIssue) isLatestKnownIssueMessage(knownIssues []iface.KnownIssue) bool {
	for _, currentKnownIssue := range knownIssues {
		if currentKnownIssue.GetIssueID() == ki.IssueID && currentKnownIssue.GetDateTime().After(ki.DateTime) {
			return false
		}
	}

	return true
}

func getProductsArray(products []Product) ([]string, []string) {
	var productsName []string

	var locations []string

	for _, p := range products {
		productsName = append(productsName, p.Name)
		locations = append(locations, p.Locations...)
	}

	return productsName, locations
}

func (s *Service) UpdateGcpKnownIssues(ctx context.Context) error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	knownIssues := make([]iface.KnownIssue, 0)

	data, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretAppEngine)
	if err != nil {
		return err
	}

	gmailconf, err := google.JWTConfigFromJSON(data, gmail.GmailReadonlyScope)
	if err != nil {
		return err
	}

	gmailService, err := getGmailService(ctx, gmailconf)
	if err != nil {
		return err
	}

	after := time.Now().AddDate(0, 0, -1).Format("2006/01/02")
	filter := fmt.Sprintf("from:google.com after:%s", after)

	list, err := gmailService.Users.Messages.List("me").Q(filter).Do()
	if err != nil {
		return err
	}

	bw := fs.BulkWriter(ctx)
	defer bw.End()

	for _, messageListItem := range list.Messages {
		message, err := gmailService.Users.Messages.Get("me", messageListItem.Id).Do()
		if err != nil {
			l.Warning(err)
			continue
		}

		// Move on to the next message if this message doesn't have any attachment id
		if message.Payload == nil || len(message.Payload.Parts) < 2 || message.Payload.Parts[1].Body.AttachmentId == "" {
			continue
		}

		messageSubject := getGcpMessageSubject(message)

		messageAttachmentID := message.Payload.Parts[1].Body.AttachmentId

		attachment, err := gmailService.Users.Messages.Attachments.Get("me", messageListItem.Id, messageAttachmentID).Do()
		if err != nil {
			l.Warningf("failed getting attachment for message %s with error: %s", messageListItem.Id, err)
			continue
		}

		attachmentDecodedBytes, err := base64.URLEncoding.DecodeString(attachment.Data)
		if err != nil {
			l.Warningf("failed decoding attachment for message %s with error: %s", messageListItem.Id, err)
			continue
		}

		var messageAttachment messageAttachment

		if err := json.Unmarshal(attachmentDecodedBytes, &messageAttachment); err != nil {
			l.Warningf("failed unmarshalling attachment for message %s with error: %s", messageListItem.Id, err)
			continue
		}

		status := getKnownIssueStatus(messageAttachment.State)
		nextUpdateTime := getKnownIssueNextUpdateTime(messageAttachment.NextUpdateTime)
		products, locations := getProductsArray(messageAttachment.Products)

		knownIssue := GCPKnownIssue{
			IssueID:           messageAttachment.TrackingID,
			Products:          products,
			Platform:          gcpPlatform,
			Title:             messageSubject,
			OutageDescription: messageAttachment.UpdateMessage,
			Status:            status,
			DateTime:          messageAttachment.UpdateTime,
			Summary:           messageAttachment.Summary,
			Symptoms:          messageAttachment.Symptoms,
			Workaround:        messageAttachment.Workaround,
			NextUpdateTime:    nextUpdateTime,
			ExposureLevel:     messageAttachment.ExposureLevel,
			Locations:         locations,
		}
		knownIssues = append(knownIssues, knownIssue)

		if knownIssue.isLatestKnownIssueMessage(knownIssues) {
			if err := knownIssue.AddOrUpdateKnownIssue(ctx, s.getKnownIssuesCollection(ctx), bw); err != nil {
				l.Errorf("failed adding/updating gcp known issue %s with error: %s", err)
			}
		}
	}

	return err
}
