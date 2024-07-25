package service

import (
	"context"
	"errors"
	"reflect"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/cloudanalytics/collab"
	labels "github.com/doitintl/hello/scheduled-tasks/labels/domain"
)

func getLabelUpdates(req UpdateLabelRequest) []firestore.Update {
	var updates []firestore.Update
	if req.Name != "" {
		updates = append(updates, firestore.Update{Path: "name", Value: req.Name})
	}

	if req.Color != "" {
		updates = append(updates, firestore.Update{Path: "color", Value: req.Color})
	}

	return updates
}

func (s *LabelsService) getObjectReference(ctx context.Context, objectID string, objectType labels.ObjectType) (*firestore.DocumentRef, error) {
	switch objectType {
	case labels.AlertType:
		return s.alertsDal.GetRef(ctx, objectID), nil
	case labels.AttributionsGroupType:
		return s.attributionGroupsDal.GetRef(ctx, objectID), nil
	case labels.AttributionType:
		return s.attributionsDal.GetRef(ctx, objectID), nil
	case labels.BudgetType:
		return s.budgetsDal.GetRef(ctx, objectID), nil
	case labels.MetricType:
		return s.metricsDal.GetRef(ctx, objectID), nil
	case labels.ReportType:
		return s.reportsDal.GetRef(ctx, objectID), nil
	default:
		return nil, labels.ErrInvalidObjectType
	}
}

func (s *LabelsService) checkObjectPermissions(ctx context.Context, objectID string, objectType labels.ObjectType) (bool, error) {
	objRef, err := s.getObjectReference(ctx, objectID, objectType)
	if err != nil {
		return false, err
	}

	var accessSettings collab.Access

	var owner string

	switch objectType {
	case labels.AlertType,
		labels.AttributionsGroupType,
		labels.AttributionType,
		labels.BudgetType,
		labels.ReportType:
		oSnap, err := objRef.Get(ctx)
		if err != nil {
			return false, err
		}

		if err := oSnap.DataTo(&accessSettings); err != nil {
			return false, err
		}
	case labels.MetricType:
		oSnap, err := objRef.Get(ctx)
		if err != nil {
			return false, err
		}

		ownerField, err := oSnap.DataAt("owner")
		if err != nil {
			return false, err
		}

		parsedOwner, ok := ownerField.(string)
		if !ok {
			return false, errors.New("couldn't convert metric owner")
		}

		owner = parsedOwner
	default:
		return false, labels.ErrInvalidObjectType
	}

	if err != nil {
		return false, err
	}

	email := ctx.Value("email").(string)

	if !reflect.DeepEqual(accessSettings, collab.Access{}) {
		return accessSettings.CanEdit(email), nil
	}

	if owner != "" {
		return owner == email, nil
	}

	return false, nil
}

func sliceToMap[T any, M comparable](a []T, keyFunction func(T) M) map[M]T {
	n := make(map[M]T, len(a))
	for _, e := range a {
		n[keyFunction(e)] = e
	}

	return n
}
