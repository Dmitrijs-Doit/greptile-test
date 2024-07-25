package handlers

import (
	labels "github.com/doitintl/hello/scheduled-tasks/labels/domain"
	"github.com/doitintl/hello/scheduled-tasks/labels/service"
)

func isValidCreateLabelRequest(req service.CreateLabelRequest) error {
	if req.Name == "" {
		return labels.ErrInvalidName
	}

	if !req.Color.IsValid() {
		return labels.ErrInvalidColor
	}

	if req.CustomerID == "" {
		return labels.ErrInvalidCustomer
	}

	if req.UserEmail == "" {
		return labels.ErrInvalidUser
	}

	return nil
}

func isValidUpdateLabelRequest(req service.UpdateLabelRequest) error {
	if req.LabelID == "" {
		return labels.ErrInvalidLabelID
	}

	if req.Color == "" && req.Name == "" {
		return labels.ErrEmptyRequest
	}

	if req.Color != "" && !req.Color.IsValid() {
		return labels.ErrInvalidColor
	}

	return nil
}

func validateAssignObjectLabelsRequest(req service.AssignLabelsRequest) error {
	if req.CustomerID == "" {
		return labels.ErrInvalidCustomer
	}

	if len(req.Objects) == 0 {
		return labels.ErrInvalidObjects
	}

	duplicatedObjects := make(map[string]struct{})

	for _, o := range req.Objects {
		if o.ObjectID == "" {
			return labels.ErrInvalidObjectID
		}

		if !o.ObjectType.IsValid() {
			return labels.ErrInvalidObjectType
		}

		if _, exists := duplicatedObjects[o.ObjectID]; exists {
			return labels.ErrDuplicatedObjectInRequest
		}

		duplicatedObjects[o.ObjectID] = struct{}{}
	}

	if len(req.AddLabels) == 0 && len(req.RemoveLabels) == 0 {
		return labels.ErrNoLabelsToAddOrRemove
	}

	duplicatedLabels := make(map[string]struct{})
	for _, label := range req.AddLabels {
		if _, exists := duplicatedLabels[label]; exists {
			return labels.ErrDuplicatedLabelInRequest
		}

		duplicatedLabels[label] = struct{}{}
	}

	for _, label := range req.RemoveLabels {
		if _, exists := duplicatedLabels[label]; exists {
			return labels.ErrDuplicatedLabelInRequest
		}

		duplicatedLabels[label] = struct{}{}
	}

	return nil
}
