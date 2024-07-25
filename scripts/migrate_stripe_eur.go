package scripts

import (
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	stripe "github.com/stripe/stripe-go/v74"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/stripe/domain"
	"github.com/doitintl/hello/scheduled-tasks/stripe/service"
)

func MigrateStripeEUREntities(ctx *gin.Context) []error {
	dryRun := ctx.Query("dryRun") == "true"

	l := logger.FromContext(ctx)

	fs, err := firestore.NewClient(ctx, common.ProjectID)

	stripeDE, err := service.NewStripeClient(domain.StripeAccountDE)
	if err != nil {
		return []error{err}
	}

	if err != nil {
		return []error{err}
	}

	defer fs.Close()

	var errors []error

	stripePMTypes := []common.EntityPaymentType{
		common.EntityPaymentTypeCard,
		// common.EntityPaymentTypeSEPADebit,
	}

	docs := fs.Collection("entities").Where("currency", "==", "EUR").Where("payment.type", "in", stripePMTypes).Documents(ctx)

	for {
		doc, err := docs.Next()
		if err == iterator.Done {
			break
		}

		var entity common.Entity

		err = doc.DataTo(&entity)
		if err != nil {
			errors = append(errors, fmt.Errorf("error can't unmarshal entity %s: %v", doc.Ref.ID, err))
			continue
		}

		l.Infof("EUR entity %s has Stripe paymentMethodID %s", doc.Ref.ID, entity.Payment)

		// add the DE account id to the entity payment
		if !dryRun {
			_, err := doc.Ref.Update(ctx, []firestore.Update{
				{Path: "payment.accountId", Value: "acct_1Myzw1JCzGwjua24"},
			})
			if err != nil {
				errors = append(errors, fmt.Errorf("error updating entity %s: %v", doc.Ref.ID, err))
				continue
			}

			l.Infof("entity %s updated with accountId acct_1Myzw1JCzGwjua24", doc.Ref.ID)
		} else {
			l.Infof("entity %s would be updated with accountId acct_1Myzw1JCzGwjua24", doc.Ref.ID)
		}

		// udpate pm ids
		stripeCustomerDoc, err := fs.Collection("integrations").Doc("stripe").Collection("stripeCustomers").Doc(doc.Ref.ID).Get(ctx)
		if err != nil {
			errors = append(errors, fmt.Errorf("error getting stripe customer %s: %v", doc.Ref.ID, err))
			continue
		}

		pmIter := stripeDE.PaymentMethods.List(&stripe.PaymentMethodListParams{
			Customer: stripe.String(stripeCustomerDoc.Data()["id"].(string))},
		)
		matched := false

		for {
			if !pmIter.Next() {
				break
			}
			// card
			if entity.Payment.Card != nil && pmIter.PaymentMethod().Card != nil && pmIter.PaymentMethod().Card.Last4 == entity.Payment.Card.Last4 {
				matched = true

				if !dryRun {
					_, err := doc.Ref.Update(ctx, []firestore.Update{
						{Path: "payment.card.id", Value: pmIter.PaymentMethod().ID},
					})
					if err != nil {
						errors = append(errors, fmt.Errorf("error updating entity %s: %v", doc.Ref.ID, err))
						continue
					}

					l.Infof("entity %s updated with CC paymentMethodID %s", doc.Ref.ID, pmIter.PaymentMethod().ID)
				} else {
					l.Infof("entity %s would be updated with CC paymentMethodID %s", doc.Ref.ID, pmIter.PaymentMethod().ID)
				}
			}
			// SEPA: run after SEPA migration
			//
			//	} else if entity.Payment.SEPADebit != nil && pmIter.PaymentMethod().SEPADebit != nil && pmIter.PaymentMethod().SEPADebit.Last4 == entity.Payment.SEPADebit.Last4 {
			//		matched = true
			//		if !dryRun {
			//			_, err := doc.Ref.Update(ctx, []firestore.Update{
			//				{Path: "payment.sepaDebit.id", Value: pmIter.PaymentMethod().ID},
			//			})
			//			if err != nil {
			//				errors = append(errors, fmt.Errorf("error updating entity %s: %v", doc.Ref.ID, err))
			//				continue
			//			}
			//			logger.Infof("entity %s updated with SEPA paymentMethodID %s", doc.Ref.ID, pmIter.PaymentMethod().ID)
			//		} else {
			//			logger.Infof("entity %s would be updated with SEPA paymentMethodID %s", doc.Ref.ID, pmIter.PaymentMethod().ID)
			//		}
			//	}
		}

		if !matched {
			errors = append(errors, fmt.Errorf("no matching payment method found for entity %s", doc.Ref.ID))
		}
	}

	return errors
}
