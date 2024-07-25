package dal

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/iface"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
)

//go:generate mockery --name FirestoreDAL
type FirestoreDAL interface {
	GetCustomerBillingDataConfigs(ctx context.Context, customerID string) ([]BillingDataConfig, error)
	CreateCustomerBillingDataConfig(ctx context.Context, config BillingDataConfig) error
}

type dal struct {
	firestoreClientFun connection.FirestoreFromContextFun
	documentsHandler   iface.DocumentsHandler
}

func NewFirestoreDAL(fs func(ctx context.Context) *firestore.Client) FirestoreDAL {
	return &dal{
		firestoreClientFun: fs,
		documentsHandler:   doitFirestore.DocumentHandler{},
	}
}

func (d *dal) GetCustomerBillingDataConfigs(ctx context.Context, customerID string) ([]BillingDataConfig, error) {
	doc, err := d.documentsHandler.GetAll(d.firestoreClientFun(ctx).
		Collection("app").
		Doc("azure").
		Collection("standalone").
		Where("customerId", "==", customerID).
		Documents(ctx))
	if err != nil {
		return nil, fmt.Errorf("failed to get customer billing data configs: %w", err)
	}

	documents := make([]BillingDataConfig, 0)

	for _, snap := range doc {
		billingDataConfig := BillingDataConfig{}
		if err := snap.DataTo(&billingDataConfig); err != nil {
			return nil, fmt.Errorf("failed to convert document to billing data config: %w", err)
		}

		documents = append(documents, billingDataConfig)
	}

	return documents, nil
}

func (d *dal) CreateCustomerBillingDataConfig(ctx context.Context, config BillingDataConfig) error {
	_, err := d.documentsHandler.Create(ctx, d.firestoreClientFun(ctx).Collection("app").Doc("azure").Collection("standalone").NewDoc(), config)
	if err != nil {
		return fmt.Errorf("failed to create customer billing data config: %w", err)
	}

	return nil
}
