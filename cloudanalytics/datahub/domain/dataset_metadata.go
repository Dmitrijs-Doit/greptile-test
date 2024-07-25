package domain

import (
	"time"

	"cloud.google.com/go/firestore"
)

type DatasetMetadata struct {
	Name        string                 `firestore:"name"`
	Description string                 `firestore:"description"`
	Customer    *firestore.DocumentRef `firestore:"customer"`
	CreatedBy   string                 `firestore:"createdBy"`
	CreatedAt   time.Time              `firestore:"createdAt"`
}

func (d *DatasetMetadata) Validate() error {
	if d.Name == "" {
		return ErrDatasetNameRequired
	}

	if d.CreatedBy == "" {
		return ErrDatasetCreatedByRequired
	}

	if d.CreatedAt.IsZero() {
		return ErrDatasetCreatedAtRequired
	}

	return nil
}

type CreateDatasetRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}
