package microsoft

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

// CopyLicenseToDashboard copies office365 inventory subscriptions to dashboard chart
func CopyLicenseToDashboard(ctx context.Context) error {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)

	type LastUpdate struct {
		OfficeLastUpdate interface{} `firestore:"officeLastUpdate,serverTimestamp"`
	}

	lastDate, err := fs.Collection("dashboards").Doc("licenseChart").Get(ctx)
	if err != nil {
		l.Errorf("failed to get last update date with error: %s", err)
		return err
	}

	var lu LastUpdate

	if err := lastDate.DataTo(&lu); err != nil {
		l.Errorf("failed to get last update date with error: %s", err)
		return err
	}

	l.Infof("last update date: %s", lu.OfficeLastUpdate.(time.Time))

	if err := copyMonth(ctx, lu.OfficeLastUpdate.(time.Time)); err != nil {
		l.Errorf("failed to copy month with error: %s", err)
		return err
	}

	return nil
}

func copyMonth(ctx context.Context, startDate time.Time) error {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)

	t := startDate
	plusFiveDay := t

	for time.Now().After(plusFiveDay) {
		t = plusFiveDay
		plusFiveDay = t.Add(7 * 24 * time.Hour)

		// Update last date
		if _, err := fs.Collection("dashboards").Doc("licenseChart").Set(ctx, map[string]interface{}{
			"officeLastUpdate": t,
		}, firestore.MergeAll); err != nil {
			l.Errorf("failed to update last update date with error: %s", err)
			return err
		}

		maxLic := make(map[string]map[string]int64)
		iter := fs.Collection("inventory").Doc(t.Format("2006-01")+"-office-365").Collection("inventory-office-365").Where("timestamp", ">", t).Where("timestamp", "<", plusFiveDay).Documents(ctx)

		var iterateRetries int

		for {
			doc, err := iter.Next()
			if err != nil {
				if err == iterator.Done {
					break
				}

				l.Errorf("failed to iterate inventory with error: %s", err)

				if iterateRetries > 5 {
					return err
				}

				iterateRetries++

				continue
			}

			var inventoryItem InventoryItem

			if err := doc.DataTo(&inventoryItem); err != nil {
				l.Errorf("failed to populate inventory item %s with error: %s", doc.Ref.ID, err)
				continue
			}

			skuName := strings.ToLower(strings.Replace(inventoryItem.Subscription.OfferName, " ", "-", -1))

			if maxLic[inventoryItem.CustomerDomain][inventoryItem.Subscription.OfferName] < inventoryItem.Subscription.Quantity {
				if _, exist := maxLic[inventoryItem.CustomerDomain]; !exist {
					maxLic[inventoryItem.CustomerDomain] = make(map[string]int64)
				}

				maxLic[inventoryItem.CustomerDomain][inventoryItem.Subscription.OfferName] = inventoryItem.Subscription.Quantity
			}

			weekDate := t.Format("2006-01-02")

			obj := make(map[string]map[string]interface{})
			obj[inventoryItem.CustomerDomain] = make(map[string]interface{})
			obj[inventoryItem.CustomerDomain][skuName] = map[string]interface{}{
				"platform": common.Assets.Office365,
				"domain":   inventoryItem.CustomerDomain,
				"skuName":  inventoryItem.Subscription.OfferName,
				"quantity": map[string]interface{}{
					weekDate: maxLic[inventoryItem.CustomerDomain][inventoryItem.Subscription.OfferName],
				},
			}
			obj["info"] = map[string]interface{}{
				"customer": inventoryItem.Customer,
			}

			licenseChartRef := fs.Collection("dashboards").Doc("licenseChart").Collection("companies").Doc(inventoryItem.Customer.ID)

			if _, err := licenseChartRef.Set(ctx, obj, firestore.MergeAll); err != nil {
				l.Errorf("failed to update license chart for customer %s with error: %s", inventoryItem.Customer.ID, err)
				return err
			}
		}
	}

	return nil
}
