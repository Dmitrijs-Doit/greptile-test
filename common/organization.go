package common

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
)

type Organization struct {
	Name                  string                   `firestore:"name"`
	Description           string                   `firestore:"description"`
	Scope                 []*firestore.DocumentRef `firestore:"scope"`
	Parent                *firestore.DocumentRef   `firestore:"parent"`
	TimeCreated           time.Time                `firestore:"timeCreated"`
	TimeModified          time.Time                `firestore:"timeModified"`
	Customer              *firestore.DocumentRef   `firestore:"customer"`
	Dashboards            []string                 `firestore:"dashboards"`
	AllowCustomDashboards bool                     `firestore:"allowCustomDashboards"`
	LastAccessed          time.Time                `firestore:"lastAccessed"`

	Snapshot *firestore.DocumentSnapshot `firestore:"-"`
	ID       string                      `firestore:"-"`
}

func GetOrganization(ctx context.Context, ref *firestore.DocumentRef) (*Organization, error) {
	if ref == nil {
		return nil, errors.New("invalid nil organization ref")
	}

	docSnap, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}

	var org Organization
	if err := docSnap.DataTo(&org); err != nil {
		return nil, err
	}

	org.Snapshot = docSnap

	return &org, nil
}
