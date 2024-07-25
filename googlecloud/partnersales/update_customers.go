package partnersales

import (
	"context"
	"errors"

	"cloud.google.com/go/channel/apiv1/channelpb"
	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
)

// SyncCustomers updates firestore channel customers data according to current customers
// list in Partner Sales Console
func (s *GoogleChannelService) SyncCustomers(ctx context.Context) []error {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	l.Info("ChannelServices - Update Customers")

	wb := fb.NewAutomaticWriteBatch(fs, 250)

	partnerSalesCustomersCollection := fs.Collection("integrations").
		Doc("google-cloud").
		Collection("googlePartnerSalesCustomers")

	lastFSChannelCustomersDataMap, err := s.getLastFSChannelCustomersDataMap(ctx, partnerSalesCustomersCollection)
	if err != nil {
		return []error{err}
	}

	customers := s.cloudChannel.ListCustomers(ctx, &channelpb.ListCustomersRequest{
		Parent:   partnerAccountName,
		PageSize: 50,
	})

	for {
		customer, err := customers.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			l.Error("Failed getting next customer")
			return []error{err}
		}

		customerID, err := s.updateCustomer(ctx, wb, partnerSalesCustomersCollection, customer)
		if err != nil {
			return []error{err}
		}

		// if customer is present in current firestore data, delete it from the map
		delete(lastFSChannelCustomersDataMap, customerID)
	}

	// at this point lastFSChannelCustomersDataMap contains only documents that are no longer
	// present in Partner Sales Console
	for _, snap := range lastFSChannelCustomersDataMap {
		wb.Delete(snap)
	}

	if errs := wb.Commit(ctx); len(errs) > 0 {
		l.Error("Failed batch commit")
		return errs
	}

	return nil
}

// returns current firestore data in map structure for quick data search
func (s *GoogleChannelService) getLastFSChannelCustomersDataMap(
	ctx context.Context,
	partnerSalesCustomersCollection *firestore.CollectionRef,
) (map[string]*firestore.DocumentRef, error) {
	l := s.loggerProvider(ctx)

	docSnaps, err := partnerSalesCustomersCollection.Documents(ctx).GetAll()
	if err != nil {
		l.Error("Failed reading all existing channel customers")
		return nil, err
	}

	snapsMap := make(map[string]*firestore.DocumentRef)
	for _, docSnap := range docSnaps {
		snapsMap[docSnap.Ref.ID] = docSnap.Ref
	}

	return snapsMap, nil
}

// returns updated channel customer id
func (s *GoogleChannelService) updateCustomer(
	ctx context.Context,
	wb *fb.AutomaticWriteBatch,
	partnerSalesCustomersCollection *firestore.CollectionRef,
	customer *channelpb.Customer,
) (string, error) {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	customerID := s.getChannelCustomerID(customer)

	docSnaps, err := fs.Collection("customers").
		Where("domains", "array-contains", customer.Domain).
		Limit(1).Documents(ctx).GetAll()
	if err != nil {
		l.Error("Failed reading customer" + customer.Domain)
		return customerID, err
	}

	if len(docSnaps) == 0 {
		return customerID, errors.New("Missing customer id " + customerID + ", domain " + customer.Domain)
	}

	wb.Set(partnerSalesCustomersCollection.Doc(customerID), ChannelServicesCustomer{
		Customer:       docSnaps[0].Ref,
		Domain:         customer.Domain,
		OrgDisplayName: customer.OrgDisplayName,
	})

	return customerID, nil
}
