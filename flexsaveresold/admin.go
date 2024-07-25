package flexsaveresold

import (
	"context"
	"errors"
	"github.com/doitintl/hello/scheduled-tasks/slice"
	"time"

	"cloud.google.com/go/firestore"
	fspkg "github.com/doitintl/firestore/pkg"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
)

// ReactivateAllOrdersForCustomer regenerates pricing information for all active orders
// for specific customer without changing any capacity information.
// Main use case is to update orders if external properties (like margin) were
// updated or if capacity was manually changed.
// Note that we don't filter to this month only, as we generally want to update all orders.
// This might or might not be desired during first 5 days of the month when previous month is still active.
func (s *Service) ReactivateAllOrdersForCustomer(ctx context.Context, customerID string) error {
	fs := s.Firestore(ctx)

	customerRef := fs.Collection("customers").Doc(customerID)

	orders, err := fs.Collection("integrations").
		Doc("amazon-web-services").
		Collection("flexibleReservedInstances").
		Where("customer", "==", customerRef).
		Where("status", "==", OrderStatusActive).
		Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	s.Logger(ctx).Infof("found %v to reactivate for customer %v", len(orders), customerID)

	return s.reactivateOrders(ctx, orders)
}

// ReactivateOrders regenerates pricing information for provided orders without
// changing any capacity information.
// Main use case is to update orders if external properties (like margin) were
// updated or if capacity was manually changed.
// orderIds represent id field inside order rather than firestore ID as those
// are visible via UI.
func (s *Service) ReactivateOrders(ctx context.Context, orderIds []int) error {
	fs := s.Firestore(ctx)
	log := s.Logger(ctx)

	orders, err := fs.Collection("integrations").
		Doc("amazon-web-services").
		Collection("flexibleReservedInstances").
		Where("id", "in", orderIds).
		Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	log.Infof("found match for %v out of %v requested orders", len(orderIds), len(orders))

	return s.reactivateOrders(ctx, orders)
}

func (s *Service) reactivateOrders(ctx context.Context, orders []*firestore.DocumentSnapshot) error {
	for _, orderSnap := range orders {
		var order FlexRIOrder
		if err := orderSnap.DataTo(&order); err != nil {
			return err
		}

		err := s.ActivateFlexRIOrder(ctx, order.Customer.ID, orderSnap.Ref.ID, true)
		if err != nil {
			return err
		}
	}

	return nil
}

// AcceptAutopilotOrders accepts all orders in a 'pending' state.
// This will transition them into 'active' state and fill in pricing information.
// This is called manually at the begining of the month when we verify all
// monthly potential or when we manually regenerate orders.
func (s *Service) AcceptAutopilotOrders(ctx context.Context) error {
	fs := s.Firestore(ctx)
	log := s.Logger(ctx)

	docSnaps, err := fs.Collection("integrations").
		Doc("amazon-web-services").
		Collection("flexibleReservedInstances").
		Where("status", "==", OrderStatusPending).
		Where("execution", "==", OrderExecAutopilot).
		Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	log.Infof("accepting %v orders", len(docSnaps))

	for _, docSnap := range docSnaps {
		var order FlexRIOrder
		if err := docSnap.DataTo(&order); err != nil {
			log.Errorf("could not parse doc '%s' - skipping... error: %s", docSnap.Ref.ID, err)
			continue
		}

		if err := s.ActivateFlexRIOrder(ctx, order.Customer.ID, docSnap.Ref.ID, false); err != nil {
			log.Errorf("could not activate order %s. err - %s", docSnap.Ref.ID, err)
			continue
		}
	}

	return nil
}

// RegenerateAutopilotOrdersForCustomer delete and then create new orders for a specific customer.
// You can provide offset relative to current time allowing you to regenerate current, past or future months orders.
// All orders are created in 'pending' state and need to be manually accepted.
func (s *Service) RegenerateAutopilotOrdersForCustomer(ctx context.Context, customerID string, monthOffset int) error {
	fs := s.Firestore(ctx)

	docRef := fs.Collection("customers").Doc(customerID)

	customer, err := common.GetCustomer(ctx, docRef)
	if err != nil {
		return err
	}

	flexsaveData, err := s.Firestore(ctx).Collection("integrations").Doc("flexsave").Collection("configuration").Doc(customer.Snapshot.Ref.ID).Get(ctx)
	if err != nil && status.Code(err) != codes.NotFound {
		return err
	}

	var config fspkg.FlexsaveConfiguration
	if err := flexsaveData.DataTo(&config); err != nil {
		return err
	}

	if !config.AWS.Enabled {
		return NewServiceError("not autopilot customer", web.ErrBadRequest)
	}

	if err := s.regenerateOrders(ctx, customerID, monthOffset); err != nil {
		return err
	}

	return nil
}

// UpdateFlexsaveDailyOrderEndTimeByCustomerAndMonth updates endDate in orders for given month with given endDate
func (s *Service) UpdateFlexsaveDailyOrderEndTimeByCustomerAndMonth(ctx context.Context, customerID, month string, endDate string) error { // eg: month = "2021-10"
	fs := s.Firestore(ctx)
	log := s.Logger(ctx)

	customerRef := fs.Collection("customers").Doc(customerID)

	startTimeStamp, endTimeStamp, err := extractStartAndEndTimeStamps(month, endDate)
	if err != nil {
		return err
	}

	orderSnaps, err := fs.Collection("integrations").Doc("amazon-web-services").Collection("flexibleReservedInstances").
		Where("status", "in", []OrderStatus{
			OrderStatusActive,
		}).
		Where("customer", "==", customerRef).
		Where("config.startDate", "==", startTimeStamp).
		Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	for _, doc := range orderSnaps {
		if _, err := doc.Ref.Update(ctx,
			[]firestore.Update{
				{Path: "config.endDate", Value: endTimeStamp}}); err != nil {
			return err
		} else {
			log.Debugf("order-doc %s updated with config.endDate %v for customer %s", doc.Ref.ID, endTimeStamp, customerID)
		}
	}

	return nil
}

func extractStartAndEndTimeStamps(month, endDate string) (startTimeStamp, endTimeStamp time.Time, err error) {
	zeroTime := time.Time{}
	startTimeStamp, err = time.Parse("2006-01", month)

	if err != nil {
		return zeroTime, zeroTime, err
	}

	endTimeStamp, err = time.Parse("2006-01-02_15", endDate)
	if err != nil {
		return zeroTime, zeroTime, err
	} else if startTimeStamp.Month() != endTimeStamp.Month() || startTimeStamp.Year() != endTimeStamp.Year() {
		return zeroTime, zeroTime, errors.New("incorrect order month-endTime dates, endTime must be within same month of orders being amended")
	}

	return startTimeStamp, endTimeStamp, nil
}

// RegenerateAutopilotOrdersForCustomer deletes orders (if they exist) and then creates new orders for all customers.
// You can provide offset relative to current time allowing you to regenerate current, past or future months orders.
// A list of customer IDs to exclude from regeneration can be provided in the request body.
// All orders are created in 'pending' state and need to be manually accepted.
func (s *Service) RegenerateAutopilotOrdersForAllCustomers(ctx context.Context, monthOffset int, customerOffset string, customerIDsToExclude []string) error {
	log := s.Logger(ctx)

	docSnaps, err := s.Firestore(ctx).Collection("integrations").Doc("flexsave").Collection("configuration").Where("AWS.enabled", "==", true).Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	successfulCustomers := 0
	totalAutopilotCustomers := 0

	for _, docSnap := range docSnaps {
		if slice.Contains(customerIDsToExclude, docSnap.Ref.ID) {
			log.Infof("skipping orders for customer %v", docSnap.Ref.ID)
			continue
		}

		customerRef := s.customersDAL.GetRef(ctx, docSnap.Ref.ID)

		hasSharedPayerAssets, err := s.assets.HasSharedPayerAWSAssets(ctx, customerRef)
		if err != nil {
			log.Errorf("could not determine shared payer status for customer '%s' - skipping. error: %v", docSnap.Ref.ID, err)
			continue
		}

		if !hasSharedPayerAssets {
			continue
		}

		if customerOffset != "" && docSnap.Ref.ID <= customerOffset {
			log.Infof("skipping orders for customer %v", docSnap.Ref.ID)
			continue
		} else {
			log.Infof("regenerating orders for customer %v", docSnap.Ref.ID)
		}

		totalAutopilotCustomers += 1

		if err := s.regenerateOrders(ctx, docSnap.Ref.ID, monthOffset); err != nil {
			log.Errorf("could not regenerate customer %s. err - %s", docSnap.Ref.ID, err)
		}

		successfulCustomers += 1
	}

	log.Infof("Regenerated orders for %v customers. Failed for %v customers", successfulCustomers, totalAutopilotCustomers-successfulCustomers)

	return nil
}

func (s *Service) regenerateOrders(ctx context.Context, customerID string, offset int) error {
	fs := s.Firestore(ctx)

	today := time.Now().UTC()
	month := time.Date(today.Year(), today.Month()+time.Month(offset), 1, 0, 0, 0, 0, time.UTC)

	if err := s.deleteOrdersForCustomer(ctx, customerID, month); err != nil {
		return err
	}

	docRef := fs.Collection("customers").Doc(customerID)

	customer, err := common.GetCustomer(ctx, docRef)
	if err != nil {
		return err
	}

	if err := s.createFlexsaveOrdersForCustomer(ctx, customer, offset); err != nil {
		return err
	}

	return nil
}

func (s *Service) deleteOrdersForCustomer(ctx context.Context, customerID string, month time.Time) error {
	log := s.Logger(ctx)
	fs := s.Firestore(ctx)
	customerRef := fs.Collection("customers").Doc(customerID)

	orderSnaps, err := fs.Collection("integrations").Doc("amazon-web-services").Collection("flexibleReservedInstances").
		Where("status", "in", []OrderStatus{
			OrderStatusNew,
			OrderStatusActive,
			OrderStatusPending,
		}).
		Where("customer", "==", customerRef).
		Where("config.startDate", "==", month).
		Documents(ctx).GetAll()
	if err != nil {
		if status.Code(err) == codes.NotFound {
			log.Warningf("no flexsave orders found to delete for customer: %s", customerID)
			return nil
		}

		return err
	}

	for _, doc := range orderSnaps {
		if _, err := doc.Ref.Delete(ctx); err != nil {
			return err
		}
	}

	return nil
}
