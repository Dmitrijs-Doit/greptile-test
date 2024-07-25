package iface

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
)

// KnownIssue represent the message details of a known issue email
type KnownIssue interface {
	GetIssueID() string
	GetDateTime() time.Time
	AddOrUpdateKnownIssue(
		ctx context.Context,
		knownIssuesCollection *firestore.CollectionRef,
		bw *firestore.BulkWriter,
	) error
}
