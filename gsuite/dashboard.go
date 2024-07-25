package gsuite

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/times"
)

const (
	pageSize             = 200
	dashboardsCollection = "dashboards"
	companiesCollection  = "companies"
	inventoryCollection  = "inventory"
	inventoryGSuite      = "inventory-g-suite"
	licenseChartDoc      = "licenseChart"
)

// CopyLicenseToDashboard - copy inventory to dashboard
func CopyLicenseToDashboard(ctx *gin.Context) {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)

	docSnap, err := fs.Collection(dashboardsCollection).Doc(licenseChartDoc).Get(ctx)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	lastUpdateField, err := docSnap.DataAt("gSuiteLastUpdate")
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	lastUpdate, ok := lastUpdateField.(time.Time)
	if !ok {
		ctx.AbortWithError(http.StatusInternalServerError, fmt.Errorf("last update type assert failed"))
		return
	}

	l.Debugf("last update: %v", lastUpdate)

	if err := copyMonth(ctx, fs, l, lastUpdate); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
}

func copyMonth(ctx context.Context, fs *firestore.Client, l logger.ILogger, startDate time.Time) error {
	today := times.CurrentDayUTC()

	t := startDate
	endDate := t

	licenseChartDocRef := fs.Collection(dashboardsCollection).Doc(licenseChartDoc)
	companiesCollectionRef := licenseChartDocRef.Collection(companiesCollection)

	for today.After(endDate) {
		t = endDate
		endDate = t.Add(7 * 24 * time.Hour)

		batch := fb.NewAutomaticWriteBatch(fs, 250)

		maxLic := make(map[string]map[string]int64)
		query := fs.Collection(inventoryCollection).Doc(t.Format("2006-01")+"-g-suite").
			Collection(inventoryGSuite).
			Where("timestamp", ">", t).
			Where("timestamp", "<", endDate).
			Limit(pageSize)

		var startAfter *firestore.DocumentSnapshot

		for {
			// in order to avoid network issues we had when we use iterator and deal with a lot of documents,
			// now we are reading all documents at once by using startAfter paginator & pageSize
			docSnaps, err := query.StartAfter(startAfter).Documents(ctx).GetAll()
			if err != nil {
				l.Errorf("error getting g-suite docs: %s", err)
				return err
			}

			if len(docSnaps) > 0 {
				startAfter = docSnaps[len(docSnaps)-1]
			} else {
				l.Infof("all documents in this timeframe have been processed: %v - %v", t, endDate)
				break
			}

			for _, docSnap := range docSnaps {
				var inventoryItem InventoryItem
				if err := docSnap.DataTo(&inventoryItem); err != nil {
					l.Error(err)
					continue
				}

				skuName := strings.ToLower(strings.Replace(inventoryItem.Subscription.SkuName, " ", "-", -1))

				if _, ok := maxLic[inventoryItem.CustomerDomain]; !ok {
					maxLic[inventoryItem.CustomerDomain] = make(map[string]int64)
				}

				if maxLic[inventoryItem.CustomerDomain][inventoryItem.Subscription.SkuName] < getQuantity(inventoryItem) {
					maxLic[inventoryItem.CustomerDomain][inventoryItem.Subscription.SkuName] = getQuantity(inventoryItem)
				}

				weekDate := t.Format("2006-01-02")

				obj := make(map[string]map[string]interface{})
				obj[inventoryItem.CustomerDomain] = make(map[string]interface{})
				obj[inventoryItem.CustomerDomain][skuName] = map[string]interface{}{
					"platform": "g-suite",
					"domain":   inventoryItem.CustomerDomain,
					"skuName":  inventoryItem.Subscription.SkuName,
					"quantity": map[string]interface{}{
						weekDate: maxLic[inventoryItem.CustomerDomain][inventoryItem.Subscription.SkuName],
					},
				}
				obj["info"] = map[string]interface{}{
					"customer": inventoryItem.Customer,
				}

				batch.Set(companiesCollectionRef.Doc(inventoryItem.Customer.ID), obj, firestore.MergeAll)
			}
		}

		errs := batch.Commit(ctx)
		if len(errs) > 0 {
			for _, err := range errs {
				l.Errorf("batch.Commit err: %v", err)
			}

			return errs[0]
		}

		if _, err := licenseChartDocRef.Update(ctx,
			[]firestore.Update{
				{Path: "gSuiteLastUpdate", Value: endDate},
			},
		); err != nil {
			return err
		}

		l.Infof("gSuiteLastUpdate batch committed: %s", endDate.String())
	}

	return nil
}

func getQuantity(it InventoryItem) int64 {
	var quantity int64

	if it.Subscription != nil && it.Subscription.Seats != nil {
		if it.Subscription.Plan.IsCommitmentPlan {
			quantity = it.Subscription.Seats.NumberOfSeats
		} else {
			quantity = it.Subscription.Seats.MaximumNumberOfSeats
		}

		if quantity >= 25000 {
			quantity = it.Subscription.Seats.LicensedNumberOfSeats
		}
	}

	return quantity
}
