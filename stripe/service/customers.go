package service

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/stripe/stripe-go/v74"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/stripe/domain"
)

// GetOrCreateStripeCustomer returns the stripe customer ID for the given entity
func (s *StripeService) GetOrCreateStripeCustomer(ctx context.Context, entity *common.Entity) (string, error) {
	fs := s.Firestore(ctx)
	logger := s.loggerProvider(ctx)

	stripeCus, err := getEntityCustomerIfExists(ctx, logger, s.stripeClient, fs, s.integrationDocID, entity)
	if err != nil {
		return "", err
	}

	if stripeCus != nil {
		return stripeCus.ID, nil
	}

	stripeCust, err := createEntityCustomer(ctx, logger, s.stripeClient, fs, s.stripeClient.accountID, s.integrationDocID, entity)
	if err != nil {
		return "", err
	}

	return stripeCust.ID, nil
}

func (s *StripeService) SyncCustomerData(ctx context.Context, entity *common.Entity) error {
	fs := s.Firestore(ctx)
	logger := s.loggerProvider(ctx)

	customer, err := getEntityCustomerIfExists(ctx, logger, s.stripeClient, fs, s.integrationDocID, entity)
	if err != nil {
		return err
	}

	if customer == nil {
		return nil // customer not found, nothing to sync
	}

	customerParams := &stripe.CustomerParams{
		Params: stripe.Params{
			Metadata: map[string]string{
				"customer_name": entity.Name,
			},
		},
		Name: &entity.Name,
	}

	updatedCustomer, err := s.stripeClient.Customers.Update(customer.ID, customerParams)
	if err != nil {
		return err
	}

	return SaveCustomerInegration(ctx, fs, entity, updatedCustomer, s.stripeClient.accountID, s.integrationDocID)
}

// createEntityCustomer creates a new stripe customer for the given entity, setting the metadata, returning the customer object
func createEntityCustomer(ctx context.Context, logger logger.ILogger, client *Client, firestore *firestore.Client, accountID domain.StripeAccountID, integrationDocId string, entity *common.Entity) (*stripe.Customer, error) {
	entityID := entity.Snapshot.Ref.ID
	params := &stripe.CustomerParams{Name: &entity.Name, Description: &entity.Name, Email: entity.Contact.Email}
	params.AddMetadata("customer_id", entity.Customer.ID)
	params.AddMetadata("customer_name", entity.Name)
	params.AddMetadata("priority_id", entity.PriorityID)
	params.AddMetadata("entity_id", entityID)
	params.SetIdempotencyKey(entityID)

	stripeCustomer, err := client.Customers.New(params)
	if err != nil {
		return nil, ErrCreateStripeCustomer
	}

	if err := SaveCustomerInegration(ctx, firestore, entity, stripeCustomer, accountID, integrationDocId); err != nil {
		logger.Errorf("error saving customer %s in integration doc %s: %v", stripeCustomer.ID, integrationDocId, err)

		_, err = client.Customers.Del(stripeCustomer.ID, nil)
		if err != nil {
			logger.Errorf("error deleting stripe customer %s: %v", stripeCustomer.ID, err)
		}

		return nil, ErrCreateStripeCustomer
	}

	return stripeCustomer, nil
}

// getEntityCustomerIfExists returns the stripe customer for the given entity if exists. Syncs integration doc and stripe account if needed
func getEntityCustomerIfExists(ctx context.Context, logger logger.ILogger, client *Client, fs *firestore.Client, integrationDocID string, entity *common.Entity) (*stripe.Customer, error) {
	SCIDocSnap, err := fs.Collection("integrations").Doc(integrationDocID).Collection("stripeCustomers").Doc(entity.Snapshot.Ref.ID).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return handleCustomerInegrationNotFound(ctx, fs, logger, client, integrationDocID, entity)
		}

		return nil, err
	}

	return getCustomer(ctx, logger, SCIDocSnap, client)
}

func handleCustomerInegrationNotFound(ctx context.Context, fs *firestore.Client, logger logger.ILogger, client *Client, integrationDocID string, entity *common.Entity) (*stripe.Customer, error) {
	// search for customer on the stripe account, if found, save it in the integration doc and return it
	customer, searchErr := searchCustomer(ctx, logger, client, entity.PriorityID)
	if customer == nil {
		logger.Infof("customer not found on stripe account with priority ID %s", entity.PriorityID)
		return nil, searchErr
	}

	logger.Infof("customer %s found on stripe account, saving in integration doc %s", customer.ID, integrationDocID)

	if err := SaveCustomerInegration(ctx, fs, entity, customer, client.accountID, integrationDocID); err != nil {
		logger.Errorf("error saving customer %s in integration doc %s: %v", customer.ID, integrationDocID, err)
		return nil, err
	}

	return customer, nil
}

// searchCustomer searches for a stripe customer with the given priority ID on the stripe account
func searchCustomer(ctx context.Context, logger logger.ILogger, client *Client, matchPriorityID string) (*stripe.Customer, error) {
	customerIter := client.Customers.Search(&stripe.CustomerSearchParams{
		SearchParams: stripe.SearchParams{
			Query: fmt.Sprintf("metadata[%q]:%q", "priority_id", matchPriorityID),
			Limit: stripe.Int64(1),
		},
	})

	if !customerIter.Next() {
		return nil, nil
	}

	customer := customerIter.Customer()

	if customerIter.Next() {
		return nil, fmt.Errorf("more than one customer found with priority ID %s", matchPriorityID)
	}

	return customer, nil
}

// getCustomer gets the Stripe customer matching the integration doc, if it does not exist, delete the integration doc
func getCustomer(ctx context.Context, logger logger.ILogger, SCIDocSnap *firestore.DocumentSnapshot, client *Client) (*stripe.Customer, error) {
	var sci domain.Customer
	if err := SCIDocSnap.DataTo(&sci); err != nil {
		return nil, err
	}

	customer, getErr := client.Customers.Get(sci.ID, nil)
	if stripeErr, ok := getErr.(*stripe.Error); ok && stripeErr.Code == stripe.ErrorCodeResourceMissing {
		logger.Warningf("customer %s not found, deleting %s from integration doc", sci.ID, SCIDocSnap.Ref.ID)

		if _, delErr := SCIDocSnap.Ref.Delete(ctx); delErr != nil {
			logger.Errorf("error deleting integration doc: %v", SCIDocSnap.Ref.ID, delErr)
		}

		return nil, ErrCustomerNotFound
	}

	return customer, getErr
}

func SaveCustomerInegration(ctx context.Context, fs *firestore.Client, entity *common.Entity, stripeCustomer *stripe.Customer, accountID domain.StripeAccountID, integrationDocID string) error {
	entityID := entity.Snapshot.Ref.ID

	sci := domain.Customer{
		ID:        stripeCustomer.ID,
		AccountID: accountID,
		Email:     stripeCustomer.Email,
		LiveMode:  stripeCustomer.Livemode,
		Metadata: domain.CustomerMetadata{
			CustomerID: entity.Customer.ID,
			Name:       entity.Name,
			PriorityID: entity.PriorityID,
			EntityID:   entityID,
		},
		Timestamp: time.Now(),
	}

	if _, err := fs.Collection("integrations").Doc(integrationDocID).
		Collection("stripeCustomers").Doc(entityID).
		Set(ctx, sci); err != nil {
		return err
	}

	return nil
}
