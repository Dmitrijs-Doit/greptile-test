package firestore

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	rdsIface "github.com/doitintl/hello/scheduled-tasks/flexsaveresold/rds/iface"
)

//go:generate mockery --name Service --output ./mocks
type Service interface {
	Get(ctx context.Context, customerID string) (*rdsIface.FlexsaveRDSCache, error)
	Update(ctx context.Context, customerID string, data interface{}) error
	AddReasonCantEnable(ctx context.Context, customerID string, reason rdsIface.FlexsaveRDSReasonCantEnable) error
	Create(ctx context.Context, customerID string) error
	Exists(ctx context.Context, customerID string) (bool, error)
	Enable(ctx context.Context, customerID string, timeEnabled time.Time) error
}

type dal struct {
	firestoreClient  *firestore.Client
	documentsHandler iface.DocumentsHandler
}

func NewService(fs *firestore.Client) Service {
	return &dal{
		firestoreClient:  fs,
		documentsHandler: doitFirestore.DocumentHandler{},
	}
}

func (d *dal) collection() *firestore.CollectionRef {
	return d.firestoreClient.Collection("integrations").Doc("flexsave").Collection("configuration-rds")
}

func (d *dal) Get(ctx context.Context, customerID string) (*rdsIface.FlexsaveRDSCache, error) {
	doc := d.collection().Doc(customerID)

	snap, err := d.documentsHandler.Get(ctx, doc)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, doitFirestore.ErrNotFound
		}

		return nil, err
	}

	var s rdsIface.FlexsaveRDSCache
	if err := snap.DataTo(&s); err != nil {
		return nil, err
	}

	return &s, nil
}

func (d *dal) Update(ctx context.Context, customerID string, data interface{}) error {
	doc := d.collection().Doc(customerID)

	_, err := d.documentsHandler.Set(ctx, doc, data, firestore.MergeAll)
	if err != nil {
		return err
	}

	return nil
}

func (d *dal) AddReasonCantEnable(ctx context.Context, customerID string, reason rdsIface.FlexsaveRDSReasonCantEnable) error {
	doc, err := d.Get(ctx, customerID)
	if err != nil {
		return err
	}

	return d.Update(ctx, customerID, map[string]interface{}{
		"reasonCantEnable": append(doc.ReasonCantEnable, reason),
	})
}

func (d *dal) Exists(ctx context.Context, customerID string) (bool, error) {
	doc := d.collection().Doc(customerID)

	snap, err := d.documentsHandler.Get(ctx, doc)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}

		return false, err
	}

	return snap.Exists(), nil
}

func (d *dal) Create(ctx context.Context, customerID string) error {
	ref := d.collection().Doc(customerID)

	_, err := d.documentsHandler.Create(ctx, ref, rdsIface.FlexsaveRDSCache{
		ReasonCantEnable: []rdsIface.FlexsaveRDSReasonCantEnable{},
		TimeEnabled:      nil,
		SavingsSummary: rdsIface.FlexsaveSavingsSummary{
			CurrentMonth:     "",
			NextMonthSavings: 0,
		},
		SavingsHistory:      map[string]rdsIface.MonthSummary{},
		DailySavingsHistory: map[string]rdsIface.MonthSummary{},
	})

	if err != nil {
		return err
	}

	return nil
}

func (d *dal) Enable(ctx context.Context, customerID string, timeEnabled time.Time) error {
	return d.Update(ctx, customerID, map[string]interface{}{
		"timeEnabled": timeEnabled,
	})
}
