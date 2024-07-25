package service

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/customerapi"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
	domainAttributions "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/domain/attribution"
	attrService "github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attributions/service"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
)

func toAttributionGroupGetExternal(ctx context.Context, a *attributiongroups.AttributionGroup) (*attributiongroups.AttributionGroupGetExternal, error) {
	internalAttributions := []domainAttributions.Attribution{}

	for _, attr := range a.Attributions {
		docSnap, err := attr.Get(ctx)
		if err != nil {
			return nil, err
		}

		var internalAttr domainAttributions.Attribution
		if err := docSnap.DataTo(&internalAttr); err != nil {
			return nil, err
		}

		internalAttr.ID = attr.ID
		internalAttributions = append(internalAttributions, internalAttr)
	}

	externalAttributions := attrService.ToAttributionList(internalAttributions)

	return &attributiongroups.AttributionGroupGetExternal{
		ID:           a.ID,
		Customer:     a.Customer,
		Name:         a.Name,
		Organization: a.Organization,
		TimeCreated:  a.TimeCreated,
		TimeModified: a.TimeModified,
		Description:  a.Description,
		Attributions: externalAttributions,
		Type:         a.Type,
		Cloud:        a.Cloud,
	}, nil
}

func toAttributionGroupsListExternal(attributionGroups []attributiongroups.AttributionGroup) []customerapi.SortableItem {
	var l []customerapi.SortableItem

	for _, g := range attributionGroups {
		newG := attributiongroups.AttributionGroupListItemExternal{
			ID:          g.ID,
			Name:        g.Name,
			Description: g.Description,
			Type:        g.Type,
			CreateTime:  g.TimeCreated.UnixMilli(),
			UpdateTime:  g.TimeModified.UnixMilli(),
			Cloud:       g.Cloud,
		}

		for _, c := range g.Collaborators {
			if c.Role == collab.CollaboratorRoleOwner {
				newG.Owner = c.Email
				break
			}
		}

		l = append(l, newG)
	}

	return l
}

func (s *AttributionGroupsService) validateAttributions(ctx context.Context, attributionIDs []string, customerID string) ([]*firestore.DocumentRef, error) {
	var attributionRefs []*firestore.DocumentRef

	for _, attributionID := range attributionIDs {
		attribution, err := s.attributionsDAL.GetAttribution(ctx, attributionID)
		if err != nil {
			return nil, err
		}

		if attribution.Type == string(domainAttributions.ObjectTypeManaged) {
			return nil, attributiongroups.ErrManagedAttributionTypeInvalid
		}

		if attribution.Type == string(domainAttributions.ObjectTypeCustom) && attribution.Customer.ID != customerID {
			return nil, attributiongroups.ErrForbiddenAttribution
		}

		attributionRefs = append(attributionRefs, s.attributionsDAL.GetRef(ctx, attributionID))
	}

	return attributionRefs, nil
}
