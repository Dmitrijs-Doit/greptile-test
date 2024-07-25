package flexsaveresold

import (
	"context"
	"errors"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

const (
	// barring current, process 2 days late to allow changes settle + further 2 to ensure successful run, +5 from order.EndDate
	usageDataDelay = time.Hour * 24 * 5
	// Usage which we exclude from discovering potential. Due to delay in billing and CHT processing, we cannot trust
	// that data there is valid, which could reduce the real numbers.
	cloudHealthUsageDataDelay = -72 * time.Hour
	// Least relevant samples to be discarded when searching for potential usage.
	discardedUsagePercentile = 0.1
)

var (
	ErrNoContract       = errors.New("no contract")
	ErrNoSpend          = errors.New("no spend")
	ErrLowSpend         = errors.New("low spend")
	ErrNoBillingProfile = errors.New("no billing profile")
	ErrNoAssets         = errors.New("no assets")
	ErrAlreadyEnabled   = errors.New("flexsave already enabled")
	ErrFlexsaveDisabled = errors.New("flexsave gcp disabled")
)

func commitOrder(ctx context.Context, fs *firestore.Client, order FlexRIOrder) error {
	ordersCollection := fs.Collection("integrations").Doc("amazon-web-services").Collection("flexibleReservedInstances")

	if err := fs.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		orderCounterRef := fs.Collection("app").Doc("flexible-ri-orders")
		docSnap, err := tx.Get(orderCounterRef)
		if err != nil {
			return err
		}
		orderID, err := docSnap.DataAt("id")
		if err != nil {
			return err
		}
		order.ID = orderID.(int64)
		if err := tx.Create(ordersCollection.NewDoc(), order); err != nil {
			return err
		}
		if err := tx.Update(orderCounterRef, []firestore.Update{
			{FieldPath: []string{"id"}, Value: firestore.Increment(1)},
		}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func updateRetiredOrderStatus(ctx context.Context, order *FlexRIOrder, today time.Time) error {
	if today.After((*order.Config.EndDate).Add(usageDataDelay)) {
		order.Status = OrderStatusRetired
		if _, err := order.Snapshot.Ref.Update(ctx, []firestore.Update{
			{
				FieldPath: []string{"status"},
				Value:     order.Status,
			},
		}, firestore.LastUpdateTime(order.Snapshot.UpdateTime)); err != nil {
			return err
		}
	}

	return nil
}

// GetCustomerFlexSaveInvoiceAdjustments returns FlexSave customer invoice adjustments (of given detail type) generated for given month
func GetCustomerFlexSaveInvoiceAdjustments(ctx context.Context, customerRef *firestore.DocumentRef, isAllFlexSave bool, invoiceMonth time.Time) ([]*common.InvoiceAdjustment, error) {
	invoiceAdjustments := make([]*common.InvoiceAdjustment, 0)

	var docSnaps []*firestore.DocumentSnapshot

	var err error
	if isAllFlexSave {
		docSnaps, err = customerRef.Collection("customerInvoiceAdjustments").
			Where("type", "==", common.Assets.AmazonWebServices).
			// invoiceMonth should always be first of month, but regenerate the timestamp to be safe
			Where("invoiceMonths", "array-contains", time.Date(invoiceMonth.Year(), invoiceMonth.Month(), 1, 0, 0, 0, 0, time.UTC)).
			Documents(ctx).GetAll()
	} else {
		docSnaps, err = GetCustomerFlexSaveAutopilotInvoiceAdjustments(ctx, customerRef, invoiceMonth)
	}

	if err != nil {
		return nil, err
	}

	// These substrings will filter FlexSave/FlexRI from anything else (when isAllFlexSave is true)
	substrings := []string{flexSaveDetailPrefix, flexRIDetailPrefix}

	for _, docSnap := range docSnaps {
		var invoiceAdjustment common.InvoiceAdjustment
		if err := docSnap.DataTo(&invoiceAdjustment); err != nil {
			return nil, err
		}

		invoiceAdjustment.Snapshot = docSnap
		if !isAllFlexSave {
			invoiceAdjustments = append(invoiceAdjustments, &invoiceAdjustment)
		} else {
			for _, sub := range substrings {
				if strings.Contains(invoiceAdjustment.Details, sub) {
					invoiceAdjustments = append(invoiceAdjustments, &invoiceAdjustment)
				}
			}
		}
	}

	return invoiceAdjustments, nil
}

func GetCustomerFlexSaveAutopilotInvoiceAdjustments(ctx context.Context, customerRef *firestore.DocumentRef, date time.Time) ([]*firestore.DocumentSnapshot, error) {
	dailyAttributionInvoiceDetail := "DoiT Flexsave Savings | DA"
	autopilotInvoiceTypes := []string{FlexSaveInvoiceDetails, dailyAttributionInvoiceDetail}

	return customerRef.Collection("customerInvoiceAdjustments").
		Where("type", "==", common.Assets.AmazonWebServices).
		Where("details", "in", autopilotInvoiceTypes).
		Where("invoiceMonths", "array-contains", time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)).
		Documents(ctx).GetAll()
}
