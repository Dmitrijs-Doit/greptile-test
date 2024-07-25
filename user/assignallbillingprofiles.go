package user

import (
	"net/http"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

// AssignAllBillingProfiles - assign all of a customer's entities (billing profiles) to all users in the customer [temporary workaround until all code that checks for a user's billing profile assignments is removed]
func AssignAllBillingProfiles(ctx *gin.Context) {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)

	// Fetch all users
	userSnaps, err := fs.Collection("users").OrderBy("customer.ref", firestore.Asc).Documents(ctx).GetAll()
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Create a cache of customers so we don't have to repeatedly call to Firestore for the same customer
	custCache := make(map[string]*common.Customer)

	batch := fb.NewAutomaticWriteBatch(fs, 200)

	// For each of the retrieved users:
	for _, userSnap := range userSnaps {
		var user common.User
		if err := userSnap.DataTo(&user); err != nil {
			l.Info(err)
			continue
		}

		// If the user doesn't have a customer, we can't do anything so skip
		if user.Customer.Ref == nil {
			continue
		}

		// Update the cache if the customer ID doesn't exist in the cache yet
		if custCache[user.Customer.Ref.ID] == nil {
			userCustomer, err := common.GetCustomer(ctx, user.Customer.Ref)
			if err != nil {
				l.Info(err)
				continue
			}

			custCache[user.Customer.Ref.ID] = userCustomer
		}

		// Update the user's entities with all entities from their customer
		batch.Update(userSnap.Ref, []firestore.Update{
			{
				Path:  "entities",
				Value: custCache[user.Customer.Ref.ID].Entities,
			},
		})
	}

	if errs := batch.Commit(ctx); len(errs) > 0 {
		for _, err := range errs {
			l.Errorf("batch.Commit err: %v", err)
		}
	}
}
