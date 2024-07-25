package dal

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

//go:generate mockery --name CSMEngagementDAL --output ./mocks
type CSMEngagementDAL interface {
	GetCustomerEngagementDetailsByCustomerID(ctx context.Context) (map[string]EngagementDetails, error)
	AddLastCustomerEngagementTime(ctx context.Context, customerID string, time time.Time) error
}

type dal struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

func NewCSMEngagementDAL(fs func(ctx context.Context) *firestore.Client) CSMEngagementDAL {
	return withClient(fs)
}

func withClient(fun connection.FirestoreFromContextFun) CSMEngagementDAL {
	return &dal{
		firestoreClientFun: fun,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *dal) GetCustomerEngagementDetailsByCustomerID(ctx context.Context) (map[string]EngagementDetails, error) {
	doc := d.firestoreClientFun(ctx).Collection("csmEngagement").Doc("customers").Collection("engagementDetails").Documents(ctx)

	snaps, err := d.documentsHandler.GetAll(doc)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return map[string]EngagementDetails{}, nil
		}

		return nil, err
	}

	engagementDetails := make(map[string]EngagementDetails)

	for _, snap := range snaps {
		engagementDetail := EngagementDetails{}
		if err := snap.DataTo(&engagementDetail); err != nil {
			return nil, err
		}

		engagementDetails[engagementDetail.CustomerID] = engagementDetail
	}

	return engagementDetails, nil
}

func (d *dal) AddLastCustomerEngagementTime(ctx context.Context, customerID string, notifiedTime time.Time) error {
	ref := d.firestoreClientFun(ctx).Collection("csmEngagement").Doc("customers").Collection("engagementDetails").Doc(customerID)

	_, err := d.documentsHandler.Get(ctx, ref)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			engagementDetails := EngagementDetails{
				CustomerID:    customerID,
				NotifiedDates: []time.Time{notifiedTime},
			}

			_, err = d.documentsHandler.Create(ctx, ref, engagementDetails)

			return err
		}

		return err
	}

	_, err = d.documentsHandler.Update(ctx, ref, []firestore.Update{
		{
			Path:  "NotifiedDates",
			Value: firestore.ArrayUnion(notifiedTime),
		},
	})

	return err
}
