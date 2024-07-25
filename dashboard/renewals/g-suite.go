package renewals

import (
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/assets"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/gsuite"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"
)

func GSuiteRenewalsHandler(ctx *gin.Context) {
	const pageSize = 100

	var i = 0

	var startAfter *firestore.DocumentSnapshot

	var docs []*firestore.DocumentSnapshot

	var err error

	fs := common.GetFirestoreClient(ctx)

	gsuiteSettings := make(map[string]*assets.Settings)
	q := fs.Collection("assetSettings").Where("type", "==", common.Assets.GSuite)

	iter := q.Documents(ctx)
	defer iter.Stop()

	for {
		docSnap, err := iter.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			log.Printf("Error: (Firestore) %s", err.Error())
			break
		}

		var asDoc common.AssetSettings
		if err := docSnap.DataTo(&asDoc); err != nil {
			log.Printf("Error: %s", err.Error())
		}

		if asSettings, ok := asDoc.Settings.(*assets.Settings); ok {
			gsuiteSettings[docSnap.Ref.ID] = asSettings
		} else {
			gsuiteSettings[docSnap.Ref.ID] = nil
		}
	}

	query := fs.Collection("assets").Where("type", "==", common.Assets.GSuite).Limit(pageSize)

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
			if as, ok := gsuiteSettings[docSnap.Ref.ID]; ok {
				GSuiteRenewals(ctx, fs, docSnap, as, batch)
			} else {
				GSuiteRenewals(ctx, fs, docSnap, nil, batch)
			}
		}

		if resp, err := batch.Commit(ctx); err != nil {
			log.Println(err)
		} else {
			_ = resp
		}
	}
}

func GSuiteRenewals(ctx *gin.Context, fs *firestore.Client, docSnap *firestore.DocumentSnapshot, assetSettings *assets.Settings, batch *firestore.WriteBatch) {
	now := time.Now()
	nowMs := common.ToUnixMillis(now)
	then := common.ToUnixMillis(now.AddDate(0, 3, 0))

	var asset gsuite.Asset
	if err := docSnap.DataTo(&asset); err != nil {
		log.Println(err.Error())
		return
	}

	// for _, subscription := range asset.Properties.Subscriptions {
	// if subscription.ID == asset.Properties.MainSubscriptionID {
	ref := fs.Collection("dashboards").Doc("assetsRenewals").Collection("ids").Doc(docSnap.Ref.ID)

	if asset.Properties.Subscription.Status == "ACTIVE" {
		if assetSettings != nil && assetSettings.Plan != nil {
			if assetSettings.Plan.IsCommitmentPlan && then > assetSettings.Plan.CommitmentInterval.EndTime {
				endTime := common.EpochMillisecondsToTime(assetSettings.Plan.CommitmentInterval.EndTime).UTC()
				batch.Set(ref, map[string]interface{}{
					"type":      common.Assets.GSuite,
					"customer":  asset.Customer,
					"entity":    asset.Entity,
					"asset":     docSnap.Ref,
					"endTime":   endTime,
					"timestamp": firestore.ServerTimestamp,
					"name":      asset.Properties.Subscription.SkuName,
					"domain":    asset.Properties.CustomerDomain,
				})

				return
			}
		} else if asset.Properties.Subscription.Plan.IsCommitmentPlan {
			if asset.Properties.Subscription.Plan.CommitmentInterval != nil {
				if then > asset.Properties.Subscription.Plan.CommitmentInterval.EndTime {
					endTime := common.EpochMillisecondsToTime(asset.Properties.Subscription.Plan.CommitmentInterval.EndTime).UTC()
					batch.Set(ref, map[string]interface{}{
						"type":      common.Assets.GSuite,
						"customer":  asset.Customer,
						"entity":    asset.Entity,
						"asset":     docSnap.Ref,
						"endTime":   endTime,
						"timestamp": firestore.ServerTimestamp,
						"name":      asset.Properties.Subscription.SkuName,
						"domain":    asset.Properties.CustomerDomain,
					})

					return
				}
			} else {
				endTime := common.EpochMillisecondsToTime(asset.Properties.Subscription.CreationTime).UTC().AddDate(0, 9, 0)
				if nowMs > common.ToUnixMillis(endTime) {
					batch.Set(ref, map[string]interface{}{
						"type":      common.Assets.GSuite,
						"customer":  asset.Customer,
						"entity":    asset.Entity,
						"asset":     docSnap.Ref,
						"endTime":   endTime,
						"timestamp": firestore.ServerTimestamp,
						"name":      asset.Properties.Subscription.SkuName,
						"domain":    asset.Properties.CustomerDomain,
					})

					return
				}
			}
		}
	}

	batch.Delete(ref)
	// }
	// }
}
