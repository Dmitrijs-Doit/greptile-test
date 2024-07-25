package knownissues

import (
	"context"
	"strings"

	"cloud.google.com/go/firestore"
)

const (
	defaultGcpKnownIssuesGmail = "gcp-known-issues@cre.doit-intl.com"
	ongoingStatus              = "ongoing"
	archivedStatus             = "archived"
	gcpPlatform                = "google-cloud-project"
	awsPlatform                = "amazon-web-services"
)

func getKnownIssueStatus(status string) string {
	lowercasestatus := strings.ToLower(status)

	if lowercasestatus == "ongoing" || lowercasestatus == "open" {
		return ongoingStatus
	}

	return archivedStatus
}

// GetKnownIssues - fetch known issues from gcp/aws and store them in firestore
func (s *Service) UpdateKnownIssues(ctx context.Context) error {
	l := s.loggerProvider(ctx)

	awsErr := s.UpdateAwsKnownIssues(ctx)
	if awsErr != nil {
		l.Errorf("failed to update AWS known issues with error: %s", awsErr)
	}

	gcpErr := s.UpdateGcpKnownIssues(ctx)
	if gcpErr != nil {
		l.Errorf("failed to update GCP known issues with error: %s", gcpErr)
	}

	if gcpErr != nil {
		return gcpErr
	} else if awsErr != nil {
		return awsErr
	}

	return nil
}

func (s *Service) getKnownIssuesCollection(ctx context.Context) *firestore.CollectionRef {
	return s.conn.Firestore(ctx).Collection("knownIssues")
}
