package collab

import (
	"context"
)

//go:generate mockery --name Icollab --output ./mocks
type Icollab interface {
	ShareAnalyticsResource(ctx context.Context, oldCollabs, newCollabs []Collaborator, public *PublicAccess, resourceID, requesterEmail string, sharer AnalyticsSharer, isCAOwner bool) error
}
