package microsoft

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/assets"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type InventoryItem struct {
	Settings       *assets.Settings       `firestore:"settings"`
	Subscription   *Subscription          `firestore:"subscription"`
	Customer       *firestore.DocumentRef `firestore:"customer"`
	Entity         *firestore.DocumentRef `firestore:"entity"`
	Contract       *firestore.DocumentRef `firestore:"contract"`
	CustomerDomain string                 `firestore:"customerDomain"`
	CustomerID     string                 `firestore:"customerId"`
	SubscriptionID string                 `firestore:"subscriptionId"`
	Reseller       CSPDomain              `firestore:"reseller"`
	Date           time.Time              `firestore:"date"`
	Timestamp      time.Time              `firestore:"timestamp,serverTimestamp"`
	ExpireBy       *time.Time             `firestore:"expireBy"`
}

const Inventory string = "inventory"

func persistInventory(ctx context.Context, l logger.ILogger, inventoryChan <-chan *InventoryItem) {
	fs := common.GetFirestoreClient(ctx)

	batch := fs.Batch()
	batchSize := 0

	for {
		select {
		case item := <-inventoryChan:
			if item == nil {
				if batchSize > 0 {
					if _, err := batch.Commit(ctx); err != nil {
						l.Errorf("failed to commit batch with error: %s", err)
					}
				}

				l.Info("[office-365] closing inventory log channel")

				return
			}

			fullDateString := item.Date.Format("2006-01-02")
			monthDateString := item.Date.Format("2006-01")

			parentDocID := fmt.Sprintf("%s-%s", monthDateString, common.Assets.Office365)
			parentCollection := fmt.Sprintf("%s-%s", Inventory, common.Assets.Office365)
			docID := fmt.Sprintf("%s-%s-%s", fullDateString, common.Assets.Office365, item.Subscription.ID)

			docRef := fs.Collection(Inventory).Doc(parentDocID).Collection(parentCollection).Doc(docID)

			docSnap, err := docRef.Get(ctx)
			if err != nil && status.Code(err) != codes.NotFound {
				l.Errorf("failed to get document with error: %s", err)
				break
			}

			if docSnap == nil || !docSnap.Exists() {
				batch.Set(docRef, item)

				batchSize++
			}

			if batchSize >= 50 {
				if _, err := batch.Commit(ctx); err != nil {
					l.Errorf("failed to commit batch with error: %s", err)
				}

				batch = fs.Batch()
				batchSize = 0
			}
		}
	}
}
