package service

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/attribution-groups/domain/attributiongroups"
)

func (s *AttributionGroupsService) GetAttributionGroups(
	ctx context.Context,
	attributionGroupsIDs []string,
) ([]*attributiongroups.AttributionGroup, error) {
	var docRefs []*firestore.DocumentRef

	for _, attributionGroupID := range attributionGroupsIDs {
		docRef := s.attributionGroupsDAL.GetRef(ctx, attributionGroupID)
		docRefs = append(docRefs, docRef)
	}

	attributionGroups, err := s.attributionGroupsDAL.GetAll(ctx, docRefs)
	if err != nil {
		return nil, err
	}

	return attributionGroups, nil
}
