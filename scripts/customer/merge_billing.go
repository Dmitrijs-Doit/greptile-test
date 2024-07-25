package customer

import (
	"context"

	"cloud.google.com/go/firestore"
)

// MergeContracts merges contracts from source customer to target customer
func (s *Scripts) mergeContracts(ctx context.Context, tx *firestore.Transaction, sourceCustomerRef, targetCustomerRef *firestore.DocumentRef) ([]txUpdateOperations, error) {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	queryer := fs.Collection("contracts").Where("customer", "==", sourceCustomerRef).Select("customer")

	docSnaps, err := tx.Documents(queryer).GetAll()
	if err != nil {
		return nil, err
	}

	l.Infof("Found %d contracts to merge", len(docSnaps))

	if len(docSnaps) == 0 {
		return nil, nil
	}

	res := make([]txUpdateOperations, 0, len(docSnaps))

	for _, docSnap := range docSnaps {
		res = append(res, txUpdateOperations{
			ref:     docSnap.Ref,
			updates: []firestore.Update{{Path: "customer", Value: targetCustomerRef}},
		})
	}

	return res, nil
}

// mergeRampPlans merges ramp plans from source customer to target customer
func (s *Scripts) mergeRampPlans(ctx context.Context, tx *firestore.Transaction, sourceCustomerRef, targetCustomerRef *firestore.DocumentRef) ([]txUpdateOperations, error) {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	queryer := fs.Collection("rampPlans").Where("customerRef", "==", sourceCustomerRef).Select("customerRef")

	docSnaps, err := tx.Documents(queryer).GetAll()
	if err != nil {
		return nil, err
	}

	l.Infof("Found %d ramp plans to merge", len(docSnaps))

	if len(docSnaps) == 0 {
		return nil, nil
	}

	res := make([]txUpdateOperations, 0, len(docSnaps))

	for _, docSnap := range docSnaps {
		res = append(res, txUpdateOperations{
			ref:     docSnap.Ref,
			updates: []firestore.Update{{Path: "customerRef", Value: targetCustomerRef}},
		})
	}

	return res, nil
}

// mergeBillingProfiles merges billing profiles from source customer to target customer
func (s *Scripts) mergeBillingProfiles(ctx context.Context, tx *firestore.Transaction, sourceCustomerRef, targetCustomerRef *firestore.DocumentRef) ([]txUpdateOperations, error) {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	queryer := fs.Collection("entities").Where("customer", "==", sourceCustomerRef).Select("customer")

	docSnaps, err := tx.Documents(queryer).GetAll()
	if err != nil {
		return nil, err
	}

	l.Infof("Found %d billing profiles to merge", len(docSnaps))

	if len(docSnaps) == 0 {
		return nil, nil
	}

	res := make([]txUpdateOperations, 0, len(docSnaps))

	for _, docSnap := range docSnaps {
		res = append(res, txUpdateOperations{
			ref:     docSnap.Ref,
			updates: []firestore.Update{{Path: "customer", Value: targetCustomerRef}},
		})
	}

	return res, nil
}

// mergeStripeCustomersMapping merges stripe customers from source customer to target customer
func (s *Scripts) mergeStripeCustomersMapping(ctx context.Context, tx *firestore.Transaction, sourceCustomerRef, targetCustomerRef *firestore.DocumentRef) ([]txUpdateOperations, error) {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	queryer := fs.Collection("integrations/stripe/stripeCustomers").Where("metadata.customer_id", "==", sourceCustomerRef.ID).Select("metadata.customer_id")

	docSnaps, err := tx.Documents(queryer).GetAll()
	if err != nil {
		return nil, err
	}

	l.Infof("Found %d stripe customers to merge", len(docSnaps))

	if len(docSnaps) == 0 {
		return nil, nil
	}

	res := make([]txUpdateOperations, 0, len(docSnaps))

	for _, docSnap := range docSnaps {
		l.Infof("Stripe customer: %s", docSnap.Ref.Path)

		res = append(res, txUpdateOperations{
			ref:     docSnap.Ref,
			updates: []firestore.Update{{FieldPath: []string{"metadata", "customer_id"}, Value: targetCustomerRef.ID}},
		})
	}

	return res, nil
}
