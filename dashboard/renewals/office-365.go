package renewals

import (
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/microsoft"
	"github.com/gin-gonic/gin"
)

const timeLayout = "2006-01-02T15:04:05Z"

func Office365RenewalsHandler(ctx *gin.Context) {
	const pageSize = 100

	var i = 0

	var startAfter *firestore.DocumentSnapshot

	var docs []*firestore.DocumentSnapshot

	var err error

	fs := common.GetFirestoreClient(ctx)

	query := fs.Collection("assets").Where("type", "==", common.Assets.Office365).Limit(pageSize)

	for {
		docs, err = query.StartAfter(startAfter).Documents(ctx).GetAll()
		if err != nil {
			log.Println(err.Error())
			ctx.Status(http.StatusInternalServerError)

			return
		}

		if len(docs) > 0 {
			startAfter = docs[len(docs)-1]
			i++
		} else {
			ctx.Status(http.StatusOK)
			return
		}

		hasPendingWrites := false
		batch := fs.Batch()

		for _, docSnap := range docs {
			hasPendingWrites = Office365Renewals(ctx, fs, docSnap, batch) || hasPendingWrites
		}

		if hasPendingWrites {
			if resp, err := batch.Commit(ctx); err != nil {
				log.Println(err)
			} else {
				_ = resp
			}
		}
	}
}

func Office365Renewals(ctx *gin.Context, fs *firestore.Client, docSnap *firestore.DocumentSnapshot, batch *firestore.WriteBatch) bool {
	now := time.Now()
	then := common.ToUnixMillis(now.AddDate(0, 3, 0))

	var asset microsoft.Asset
	if err := docSnap.DataTo(&asset); err != nil {
		log.Println(err.Error())
		return false
	}

	endDate, err := time.Parse(timeLayout, asset.Properties.Subscription.EndDate)
	if err != nil {
		log.Println(err.Error())
		return false
	}

	ref := fs.Collection("dashboards").Doc("assetsRenewals").Collection("ids").Doc(docSnap.Ref.ID)
	if asset.Properties.Subscription.Status != "active" {
		batch.Delete(ref)
	} else if then > common.ToUnixMillis(endDate) {
		batch.Set(ref, map[string]interface{}{
			"type":      common.Assets.Office365,
			"customer":  asset.Customer,
			"entity":    asset.Entity,
			"asset":     docSnap.Ref,
			"endTime":   endDate,
			"timestamp": firestore.ServerTimestamp,
			"name":      asset.Properties.Subscription.OfferName,
			"domain":    asset.Properties.CustomerDomain,
		})
	} else {
		batch.Delete(ref)
	}

	return true
}
