package labels

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidLabelID        = errors.New("invalid label ID")
	ErrInvalidObjects        = errors.New("invalid objects")
	ErrEmptyRequest          = errors.New("invalid empty request")
	ErrNoLabelsToAddOrRemove = errors.New("add labels and remove labels arrays are empty")
	ErrInvalidObjectType     = errors.New("invalid object type")
	ErrInvalidObjectID       = errors.New("invalid object id")
	ErrLabelNotFound         = func(labelID string) error { return fmt.Errorf("label %s not found", labelID) }
	ErrNoPermissions         = func(objectType ObjectType, objectID string) error {
		return fmt.Errorf("user doesn't have permissions to edit object: %s %s", objectType, objectID)
	}
	ErrInvalidName               = errors.New("invalid label name")
	ErrInvalidColor              = errors.New("invalid label color")
	ErrInvalidCustomer           = errors.New("invalid customer")
	ErrInvalidUser               = errors.New("invalid user")
	ErrInvalidLabel              = errors.New("invalid label")
	ErrDuplicatedLabelInRequest  = errors.New("duplicated label in request")
	ErrDuplicatedObjectInRequest = errors.New("duplicated object in request")
)
