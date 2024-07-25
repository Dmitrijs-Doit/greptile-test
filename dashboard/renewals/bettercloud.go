package renewals

import (
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/bettercloud"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

func BetterCloudRenewalsHandler(ctx *gin.Context) {
	const pageSize = 100

	var i = 0

	var startAfter *firestore.DocumentSnapshot

	var docs []*firestore.DocumentSnapshot

	var err error

	fs := common.GetFirestoreClient(ctx)

	query := fs.Collection("assets").Where("type", "==", common.Assets.BetterCloud).Where("properties.subscription.isCommitment", "==", true).Limit(pageSize)

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

		batch := fs.Batch()
		for _, docSnap := range docs {
			bettercloudRenewals(ctx, fs, docSnap, batch)
		}

		if resp, err := batch.Commit(ctx); err != nil {
			log.Println(err)
		} else {
			_ = resp
		}
	}
}

func bettercloudRenewals(ctx *gin.Context, fs *firestore.Client, docSnap *firestore.DocumentSnapshot, batch *firestore.WriteBatch) {
	then := common.ToUnixMillis(time.Now().AddDate(0, 3, 0))

	var asset bettercloud.Asset
	if err := docSnap.DataTo(&asset); err != nil {
		log.Println(err.Error())
		return
	}

	ref := fs.Collection("dashboards").Doc("assetsRenewals").Collection("ids").Doc(docSnap.Ref.ID)
	if asset.Properties.Subscription.EndDate != nil && then > common.ToUnixMillis(*asset.Properties.Subscription.EndDate) {
		batch.Set(ref, map[string]interface{}{
			"type":      common.Assets.BetterCloud,
			"customer":  asset.Customer,
			"entity":    asset.Entity,
			"asset":     docSnap.Ref,
			"endTime":   *asset.Properties.Subscription.EndDate,
			"timestamp": firestore.ServerTimestamp,
			"name":      asset.Properties.Subscription.SkuName,
			"domain":    asset.Properties.CustomerDomain,
		})
	} else {
		batch.Delete(ref)
	}
}
