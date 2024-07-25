//go:generate mockery --output=../mocks --all
package iface

import (
	"context"

	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	domainResource "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/resource/domain"
)

type AttributionGroupsIface interface {
	ShareAttributionGroup(ctx context.Context, newCollabs []collab.Collaborator, public *collab.PublicAccess, attributionGroupID, userID, requesterEmail string) error
	CreateAttributionGroup(
		ctx context.Context,
		customerID string,
		requesterEmail string,
		attributionGroup *attributiongroups.AttributionGroupRequest,
	) (string, error)
	GetAttributionGroups(ctx context.Context, attributionGroupsIDs []string) ([]*attributiongroups.AttributionGroup, error)
	UpdateAttributionGroup(ctx context.Context, customerID string, attributionGroupID string, requesterEmail string, attributionGroupUpdate *attributiongroups.AttributionGroupUpdateRequest) error
	DeleteAttributionGroup(
		ctx context.Context,
		customerID string,
		requesterEmail string,
		attributionGroupID string,
	) ([]domainResource.Resource, error)
	GetAttributionGroupExternal(ctx context.Context, attributionGroupID string) (*attributiongroups.AttributionGroupGetExternal, error)
	ListAttributionGroupsExternal(ctx context.Context, req *customerapi.Request) (*attributiongroups.AttributionGroupsListExternal, error)
	SyncEntityInvoiceAttributions(ctx context.Context, req service.SyncEntityInvoiceAttributionsRequest) error
}
