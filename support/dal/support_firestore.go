package dal

import (
	"context"

	"cloud.google.com/go/firestore"

	doitFirestore "github.com/doitintl/firestore"
	firestoreIface "github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/support/domain"
)

const (
	zendeskPlatformsCollection = "integrations/zendesk/ticketFields"
	productsCollection         = "app/support/services"
)

type SupportFirestore struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   firestoreIface.DocumentsHandler
}

func NewSupportFirestore(ctx context.Context, projectID string) (*SupportFirestore, error) {
	fs, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	supportFirestore := NewSupportFirestoreWithClient(
		func(ctx context.Context) *firestore.Client {
			return fs
		})

	return supportFirestore, nil
}

func NewSupportFirestoreWithClient(fun connection.FirestoreFromContextFun) *SupportFirestore {
	return &SupportFirestore{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *SupportFirestore) ListPlatforms(ctx context.Context, isProductOnlySupported bool) ([]domain.Platform, error) {
	docRef := d.firestoreClientFun(ctx).Collection(zendeskPlatformsCollection).Doc("platforms")
	platformsSnap, err := d.documentsHandler.Get(ctx, docRef)

	if err != nil {
		return nil, err
	}

	var platforms domain.Platforms
	if err := platformsSnap.DataTo(&platforms); err != nil {
		return nil, err
	}

	if isProductOnlySupported {
		productOnlyPlatforms := make([]domain.Platform, 0, len(platforms.Values))

		for _, platform := range platforms.Values {
			if platform.SaasSupported {
				productOnlyPlatforms = append(productOnlyPlatforms, platform)
			}
		}

		platforms.Values = productOnlyPlatforms
	}

	return platforms.Values, nil
}

func (d *SupportFirestore) ListProducts(ctx context.Context, platforms []string) ([]domain.Product, error) {
	var iter *firestore.DocumentIterator

	if len(platforms) > 0 {
		iter = d.firestoreClientFun(ctx).Collection(productsCollection).Where("platform", "in", platforms).Documents(ctx)
	} else {
		iter = d.firestoreClientFun(ctx).Collection(productsCollection).Documents(ctx)
	}

	docSnaps, err := iter.GetAll()

	if err != nil {
		return nil, err
	}

	var items []domain.Product

	for _, doc := range docSnaps {
		var item domain.Product
		if err := doc.DataTo(&item); err != nil {
			return nil, err
		}

		item.ID = doc.Ref.ID
		items = append(items, item)
	}

	return items, nil
}
