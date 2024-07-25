//go:generate mockery --output=../mocks --all
package reports

import (
	"context"

	slackgo "github.com/slack-go/slack"

	firestorePkg "github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	reportPkg "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/reports/domain/report"
)

type IReportsService interface {
	Get(ctx context.Context, customerID, reportID string) (*reportPkg.Report, error)
	GetUnfurlPayload(ctx context.Context, reportID, customerID, URL string) (*reportPkg.Report, map[string]slackgo.Attachment, error)
	UpdateSharing(ctx context.Context, reportID, customerID string, requester *firestorePkg.User, usersToAdd []string, role collab.CollaboratorRole, public bool) error
}
