package dal

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/domain"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/iam/organizations"
)

const (
	dashboardAccessMetadataCollection = "dashboards/metadata/dashboardsAccessMetadata"

	customerField          = "customerId"
	timeLastRefreshedField = "timeLastRefreshed"
)

type DashboardAccessMetadataFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

func NewDashboardAccessMetadataFirestore(ctx context.Context, projectID string) (*DashboardAccessMetadataFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return NewDashboardAccessMetadataFirestoreWithClient(
		func(_ context.Context) *firestore.Client {
			return fs
		},
	), nil
}

func NewDashboardAccessMetadataFirestoreWithClient(fun connection.FirestoreFromContextFun) *DashboardAccessMetadataFirestore {
	return &DashboardAccessMetadataFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *DashboardAccessMetadataFirestore) dashboardAccessMetadataCollection(ctx context.Context) *firestore.CollectionRef {
	return d.firestoreClientFun(ctx).Collection(dashboardAccessMetadataCollection)
}

func (d *DashboardAccessMetadataFirestore) ListCustomerDashboardAccessMetadata(ctx context.Context, customerID string) ([]*domain.DashboardAccessMetadata, error) {
	if customerID == "" {
		return nil, ErrInvalidCustomerID
	}

	iter := d.dashboardAccessMetadataCollection(ctx).
		Where(customerField, "==", customerID).
		Documents(ctx)

	docSnaps, err := d.documentsHandler.GetAll(iter)
	if err != nil {
		return nil, err
	}

	dashboardAccessMetadata := make([]*domain.DashboardAccessMetadata, len(docSnaps))

	for i, docSnap := range docSnaps {
		var accessMetadata domain.DashboardAccessMetadata
		if err := docSnap.DataTo(&accessMetadata); err != nil {
			return nil, err
		}

		dashboardAccessMetadata[i] = &accessMetadata
	}

	return dashboardAccessMetadata, nil
}

func (d *DashboardAccessMetadataFirestore) GetDashboardAccessMetadata(ctx context.Context,	customerID ,	orgID ,	dashboardID string) (*domain.DashboardAccessMetadata, error) {
	if customerID == "" {
		return nil, ErrInvalidCustomerID
	}

	if orgID == "" {
		return nil, ErrInvalidOrganizationID
	}

	if dashboardID == "" {
		return nil, ErrInvalidDashboardID
	}

	docID := buildDashboardAccessMetadataDocID(customerID, orgID, dashboardID)

	docRef := d.dashboardAccessMetadataCollection(ctx).Doc(docID)

	docSnap, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, ErrDashboardAccessMetadataNotFound
		}

		return nil, err
	}

	var accessMetadata domain.DashboardAccessMetadata

	if err := docSnap.DataTo(&accessMetadata); err != nil {
		return nil, err
	}

	return &accessMetadata, nil
}

func (d *DashboardAccessMetadataFirestore) SaveDashboardAccessMetadata(
	ctx context.Context,
	accessMetadata *domain.DashboardAccessMetadata,
) error {
	if accessMetadata == nil {
		return ErrInvalidDashboardAccessMetadata
	}

	if accessMetadata.CustomerID == "" {
		return ErrInvalidCustomerID
	}

	if accessMetadata.DashboardID == "" {
		return ErrInvalidDashboardID
	}

	if accessMetadata.OrganizationID == "" {
		return ErrInvalidOrganizationID
	}

	docID := buildDashboardAccessMetadataDocID(accessMetadata.CustomerID, accessMetadata.OrganizationID, accessMetadata.DashboardID)

	docRef := d.dashboardAccessMetadataCollection(ctx).Doc(docID)

	_, err := docRef.Set(ctx, accessMetadata)

	return err
}

func buildDashboardAccessMetadataDocID(customerID, orgID, dashboardID string) string {
	if orgID == "" {
		orgID = organizations.RootOrgID
	}

	return fmt.Sprintf("%s_%s_%s", customerID, orgID, dashboardID)
}

func (d *DashboardAccessMetadataFirestore) UpdateTimeLastRefreshed(ctx context.Context, customerID, orgID, dashboardID string) error {
	if customerID == "" {
		return ErrInvalidCustomerID
	}

	if orgID == "" {
		return ErrInvalidOrganizationID
	}

	if dashboardID == "" {
		return ErrInvalidDashboardID
	}

	docID := buildDashboardAccessMetadataDocID(customerID, orgID, dashboardID)

	docRef := d.dashboardAccessMetadataCollection(ctx).Doc(docID)

	_, err := docRef.Update(ctx, []firestore.Update{
		{
			Path:  timeLastRefreshedField,
			Value: firestore.ServerTimestamp,
		},
	})

	return err
}

func (d *DashboardAccessMetadataFirestore) ShouldRefreshDashboard(
	ctx context.Context,
	customerID, orgID, dashboardID string,
	timeRefreshThreshold time.Duration,
) (bool, *domain.DashboardAccessMetadata, error) {
	var (
		shouldRefreshDashboard bool
		accessMetadata         *domain.DashboardAccessMetadata
	)

	if customerID == "" {
		return shouldRefreshDashboard, accessMetadata, ErrInvalidCustomerID
	}

	if orgID == "" {
		return shouldRefreshDashboard, accessMetadata, ErrInvalidOrganizationID
	}

	if dashboardID == "" {
		return shouldRefreshDashboard, accessMetadata, ErrInvalidDashboardID
	}

	err := d.firestoreClientFun(ctx).RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		docID := buildDashboardAccessMetadataDocID(customerID, orgID, dashboardID)
		docRef := d.dashboardAccessMetadataCollection(ctx).Doc(docID)

		docSnap, err := tx.Get(docRef)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				now := time.Now().UTC()

				accessMetadata := &domain.DashboardAccessMetadata{
					CustomerID:        customerID,
					OrganizationID:    orgID,
					DashboardID:       dashboardID,
					TimeLastAccessed:  &now,
					TimeLastRefreshed: &now,
				}

				if err := tx.Set(docRef, accessMetadata); err != nil {
					return err
				}

				shouldRefreshDashboard = true

				return nil
			}

			return err
		}

		if err := docSnap.DataTo(&accessMetadata); err != nil {
			return err
		}

		// If there is no timeLastRefreshed, or the timeLastRefreshed is older than the timeRefreshThreshold, refresh the dashboard
		if accessMetadata.TimeLastRefreshed == nil {
			shouldRefreshDashboard = true
		} else if duration := time.Since(*accessMetadata.TimeLastRefreshed); duration > timeRefreshThreshold {
			shouldRefreshDashboard = true
		}

		if shouldRefreshDashboard {
			if err := tx.Update(docRef, []firestore.Update{
				{
					Path:  timeLastRefreshedField,
					Value: firestore.ServerTimestamp,
				},
			}); err != nil {
				return err
			}
		}

		return nil
	})

	return shouldRefreshDashboard, accessMetadata, err
}
