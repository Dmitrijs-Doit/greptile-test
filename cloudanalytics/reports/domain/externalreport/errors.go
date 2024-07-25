package externalreport

import "errors"

var (
	ErrInvalidInternalID = errors.New("invalid internal ID")
	ErrMetadataType      = errors.New("invalid metadata type")
)
