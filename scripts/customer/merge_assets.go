package customer

import (
	"context"

	"cloud.google.com/go/firestore"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

// mergeAssetsOfType merges assets of a specific type from source customer to target customer
func (s *Scripts) mergeAssetsOfType(ctx context.Context, tx *firestore.Transaction, sourceCustomerRef, targetCustomerRef *firestore.DocumentRef, assetType string) ([]txUpdateOperations, error) {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	assetsQueryer := fs.Collection("assets").Where("customer", "==", sourceCustomerRef).Where("type", "==", assetType).Select("customer")
	assetSettingsQueryer := fs.Collection("assetSettings").Where("customer", "==", sourceCustomerRef).Where("type", "==", assetType).Select("customer")

	docSnaps1, err := tx.Documents(assetsQueryer).GetAll()
	if err != nil {
		return nil, err
	}

	docSnaps2, err := tx.Documents(assetSettingsQueryer).GetAll()
	if err != nil {
		return nil, err
	}

	l.Infof("Found %d %s assets to merge", len(docSnaps1), assetType)
	l.Infof("Found %d %s assetSettings to merge", len(docSnaps2), assetType)

	allDocSnaps := append(docSnaps1, docSnaps2...)

	if len(allDocSnaps) == 0 {
		return nil, nil
	}

	res := make([]txUpdateOperations, 0, len(allDocSnaps))

	for _, docSnap := range allDocSnaps {
		res = append(res, txUpdateOperations{
			ref:     docSnap.Ref,
			updates: []firestore.Update{{Path: "customer", Value: targetCustomerRef}},
		})
	}

	return res, nil
}

// mergeGooglePartnerSalesConsoleMapping merges the Google Partner Sales Console (PSC) customer mapping from source customer to target customer
// Google PSC mapping controls to which Navigator customer the GCP Billing Accounts should be linked to.
// It is important to update it otherwise the assets would by synced back to the source customer.
func (s *Scripts) mergeGooglePartnerSalesConsoleMapping(ctx context.Context, tx *firestore.Transaction, sourceCustomerRef, targetCustomerRef *firestore.DocumentRef) ([]txUpdateOperations, error) {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	queryer := fs.Collection("integrations/google-cloud/googlePartnerSalesCustomers").Where("customer", "==", sourceCustomerRef).Select("customer")

	docSnaps, err := tx.Documents(queryer).GetAll()
	if err != nil {
		return nil, err
	}

	l.Infof("Found %d google partner sales customer mappings to merge", len(docSnaps))

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

// mergeAwsMasterPayerAccounts merges the AWS Master Payer Account from source customer to target customer
func (s *Scripts) mergeAwsMasterPayerAccounts(ctx context.Context, tx *firestore.Transaction, sourceCustomerRef, targetCustomerRef *firestore.DocumentRef) ([]txUpdateOperations, error) {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	queryer := fs.Collection("app/master-payer-accounts/mpaAccounts").Where("customerId", "==", sourceCustomerRef.ID).Select("customerId")

	docSnaps, err := tx.Documents(queryer).GetAll()
	if err != nil {
		return nil, err
	}

	l.Infof("Found %d aws master payer accounts to merge", len(docSnaps))

	if len(docSnaps) == 0 {
		return nil, nil
	}

	docSnap, err := tx.Get(targetCustomerRef)
	if err != nil {
		return nil, err
	}

	var targetCustomer common.Customer
	if err := docSnap.DataTo(&targetCustomer); err != nil {
		return nil, err
	}

	res := make([]txUpdateOperations, 0, len(docSnaps))

	for _, docSnap := range docSnaps {
		res = append(res, txUpdateOperations{
			ref: docSnap.Ref,
			updates: []firestore.Update{
				{Path: "domain", Value: targetCustomer.PrimaryDomain},
				{Path: "customerId", Value: targetCustomerRef.ID},
			},
		})
	}

	return res, nil
}

// mergeCostAnomalies merges cost anomalies from source customer to target customer
func (s *Scripts) mergeCostAnomalies(ctx context.Context, tx *firestore.Transaction, sourceCustomerRef, targetCustomerRef *firestore.DocumentRef) ([]txUpdateOperations, error) {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	queryer := fs.CollectionGroup("billingAnomalies").Where("customer", "==", sourceCustomerRef).Select("customer")

	docSnaps, err := tx.Documents(queryer).GetAll()
	if err != nil {
		return nil, err
	}

	l.Infof("Found %d cost anomalies to merge", len(docSnaps))

	if len(docSnaps) == 0 {
		return nil, nil
	}

	res := make([]txUpdateOperations, 0, len(docSnaps))

	for _, docSnap := range docSnaps {
		// Unsure if there is more than one collection called "billingAnomalies" in firestore
		if docSnap.Ref.Parent.Parent.Parent.ID != "assets" {
			continue
		}

		res = append(res, txUpdateOperations{
			ref:     docSnap.Ref,
			updates: []firestore.Update{{Path: "customer", Value: targetCustomerRef}},
		})
	}

	return res, nil
}
