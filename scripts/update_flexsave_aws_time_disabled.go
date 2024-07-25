package scripts

import (
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"github.com/gin-gonic/gin"
)

func UpdateFlexsaveConfigTimeDisabled(ctx *gin.Context) []error {
	fs, err := firestore.NewClient(ctx, common.ProjectID)
	if err != nil {
		return []error{err}
	}

	payerConfigSnaps, err := fs.Collection("integrations").Doc("flexsave").Collection("flexsave-payer-configs").Where("status", "==", "disabled").Documents(ctx).GetAll()
	if err != nil {
		return []error{err}
	}

	for _, snap := range payerConfigSnaps {
		var payerConfig types.PayerConfig

		if err := snap.DataTo(&payerConfig); err != nil {
			return []error{err}
		}

		docRef := fs.Collection("integrations").
			Doc("flexsave").
			Collection("configuration").
			Doc(payerConfig.CustomerID)

		docSnap, err := docRef.Get(ctx)
		if err != nil {
			return []error{err}
		}

		var cache pkg.FlexsaveConfiguration

		if err := docSnap.DataTo(&cache); err != nil {
			return []error{err}
		}

		if cache.AWS.Enabled {
			continue
		}

		if _, err := docRef.Update(ctx, []firestore.Update{{FieldPath: []string{"AWS", "timeDisabled"}, Value: payerConfig.TimeDisabled}}); err != nil {
			return []error{err}
		}

		fmt.Printf("timeDisabled set for customer: %s \n", payerConfig.CustomerID)
	}

	cacheSnaps, err := fs.Collection("integrations").Doc("flexsave").Collection("configuration").
		Where("AWS.enabled", "==", false).
		Documents(ctx).
		GetAll()
	if err != nil {
		return []error{err}
	}

	for _, snap := range cacheSnaps {
		customerRef := fs.Collection("customers").Doc(snap.Ref.ID)

		invoiceDocs, err := fs.CollectionGroup("customerInvoiceAdjustments").
			Where("details", "==", flexsaveresold.FlexSaveInvoiceDetails).
			Where("customer", "==", customerRef).
			OrderBy("invoiceMonths", firestore.Desc).
			Documents(ctx).GetAll()
		if err != nil {
			return []error{err}
		}

		if len(invoiceDocs) == 0 {
			continue
		}

		var invoiceAdjustment common.InvoiceAdjustment
		if err := invoiceDocs[0].DataTo(&invoiceAdjustment); err != nil {
			return []error{err}
		}

		lastInvoiceMonth := invoiceAdjustment.InvoiceMonths[0]

		disabledFrom := lastInvoiceMonth.AddDate(0, 1, 0)

		var cache pkg.FlexsaveConfiguration

		if err := snap.DataTo(&cache); err != nil {
			return []error{err}
		}

		if cache.AWS.TimeDisabled != nil {
			continue
		}

		if _, err := snap.Ref.Update(ctx, []firestore.Update{{FieldPath: []string{"AWS", "timeDisabled"}, Value: disabledFrom}}); err != nil {
			return []error{err}
		}

		fmt.Printf("timeDisabled %v set for customer: %s \n", disabledFrom, snap.Ref.ID)
	}

	return nil
}
