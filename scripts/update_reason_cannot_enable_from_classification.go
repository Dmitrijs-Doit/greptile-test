package scripts

import (
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	doitFirestore "github.com/doitintl/firestore"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

func UpdateReasonCantEnableForTerminatedCustomers(ctx *gin.Context) []error {
	l := logger.FromContext(ctx)

	errors := []error{}

	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}
	defer fs.Close()

	batch := doitFirestore.NewBatchProviderWithClient(fs, 150).Provide(ctx)

	snaps, err := fs.Collection("customers").Where("assets", common.ArrayContains, common.Assets.AmazonWebServices).Documents(ctx).GetAll()
	if err != nil {
		errors = append(errors, err)
	}

	var terminatedCustomers []string

	for _, snap := range snaps {
		var customer common.Customer

		err := snap.DataTo(&customer)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		customerID := snap.Ref.ID

		//we do not want to update customers that have g-suite or google-cloud
		if len(customer.Assets) == 1 && customer.Classification == "terminated" {
			terminatedCustomers = append(terminatedCustomers, customerID)
		}
	}

	for _, customerID := range terminatedCustomers {
		ref := fs.Collection("integrations").Doc("flexsave").Collection("configuration").Doc(customerID)

		snap, err := ref.Get(ctx)
		if err != nil {
			errors = append(errors, err)
		}

		var configuration pkg.FlexsaveConfiguration

		err = snap.DataTo(&configuration)
		if err != nil {
			errors = append(errors, err)
		}

		if configuration.AWS.Enabled {
			err = batch.Update(ctx, ref, []firestore.Update{
				{Path: "AWS.enabled", Value: false},
				{Path: "AWS.timeDisabled", Value: time.Now().UTC()},
			})
			if err != nil {
				errors = append(errors, err)
				continue
			}

			l.Infof("set enabled = false & timeDisabled for %s", customerID)
		}
	}

	err = batch.Commit(ctx)
	if err != nil {
		errors = append(errors, err)
	}

	return errors
}
