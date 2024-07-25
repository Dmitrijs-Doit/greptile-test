package scripts

import (
	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v74"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/stripe/domain"
	"github.com/doitintl/hello/scheduled-tasks/stripe/service"
)

func SyncStripeCustomers(ctx *gin.Context) []error {
	dryRun := ctx.Query("dryRun") == "true"

	l := logger.FromContext(ctx)

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}

	var errors []error

	var integrationDocID string
	if !common.Production {
		integrationDocID = "stripe-test"
	} else {
		integrationDocID = "stripe"
	}

	stripeDE, err := service.NewStripeClient(domain.StripeAccountDE)
	if err == nil {
		if err := syncAccountCustomers(ctx, stripeDE, l, fs, dryRun, domain.StripeAccountDE, integrationDocID); err != nil {
			errors = append(errors, err)
		}
	}

	stripeUS, err := service.NewStripeClient(domain.StripeAccountUS)
	if err == nil {
		if err := syncAccountCustomers(ctx, stripeUS, l, fs, dryRun, domain.StripeAccountUS, integrationDocID); err != nil {
			errors = append(errors, err)
		}
	}

	stripeUK, err := service.NewStripeClient(domain.StripeAccountUKandI)
	if err == nil {
		if err := syncAccountCustomers(ctx, stripeUK, l, fs, dryRun, domain.StripeAccountUKandI, integrationDocID); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

func syncAccountCustomers(ctx *gin.Context, client *service.Client, l logger.ILogger, fs *firestore.Client, dryRun bool, accountID domain.StripeAccountID, integrationDocID string) error {
	l.Infof("Syncing stripe customers for account %s", accountID)

	stripeCustomers := client.Customers.List(nil)
	for stripeCustomers.Next() {
		customer := stripeCustomers.Customer()

		name := customer.Name
		entityID := customer.Metadata["entity_id"]
		entityDoc, err := fs.Collection("entities").Doc(entityID).Get(ctx)

		if err != nil {
			l.Errorf("error getting entity %s", entityID)
			continue
		}

		var entity common.Entity
		err = entityDoc.DataTo(&entity)
		entity.Snapshot = entityDoc

		if err != nil {
			l.Errorf("error getting entity %s", entityID)
			continue
		}

		if name == entity.Name {
			l.Infof("name matches: %s", name)
			continue
		}

		l.Infof("updating customer name for entity %s from %s to %s", entityID, name, entity.Name)

		if !dryRun {
			updated, err := client.Customers.Update(customer.ID, &stripe.CustomerParams{
				Name: &entity.Name,
				Params: stripe.Params{
					Metadata: map[string]string{
						"customer_name": entity.Name,
					},
				},
			})
			if err != nil {
				l.Errorf("error updating customer %s: %v", customer.ID, err)
			}

			if err := service.SaveCustomerInegration(ctx, fs, &entity, updated, accountID, integrationDocID); err != nil {
				l.Errorf("error saving customer %s in integration doc %s: %v", updated.ID, integrationDocID, err)
			}
		}
	}

	return nil
}
