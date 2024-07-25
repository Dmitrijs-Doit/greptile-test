package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
	"firebase.google.com/go/v4/auth"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/assets"

	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/framework/web"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

const (
	dayDuration = 24 * time.Hour

	MaxValidRefreshTokenDuration = 2 * dayDuration
)

type Tasks struct {
	loggerProvider logger.Provider
	*connection.Connection
	awsAssetsService assets.IAWSAssetsService
}

func NewTasks(log logger.Provider, conn *connection.Connection) *Tasks {
	awsAssetsService, err := assets.NewAWSAssetsService(log, conn, conn.CloudTaskClient)
	if err != nil {
		panic(err)
	}

	return &Tasks{
		log,
		conn,
		awsAssetsService,
	}
}

func (h *Tasks) RevokeRefreshTokens(ctx *gin.Context) error {
	logger := h.loggerProvider(ctx)

	auth, err := fb.App.Auth(ctx)
	if err != nil {
		return web.NewRequestError(err, http.StatusInternalServerError)
	}

	tenantsIter := auth.TenantManager.Tenants(ctx, "")

	for {
		t, err := tenantsIter.Next()
		if err != nil {
			if err == iterator.Done {
				break
			}

			return web.NewRequestError(err, http.StatusInternalServerError)
		}

		if err := h.revokeTenantRefreshTokens(ctx, auth, t.ID); err != nil {
			logger.Errorf("revokeTenantRefreshTokens for tenant %s failed with error: %s", t.ID, err)
		}
	}

	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Tasks) revokeTenantRefreshTokens(ctx *gin.Context, auth *auth.Client, tenantID string) error {
	logger := h.loggerProvider(ctx)
	fs := h.Firestore(ctx)

	tenantAuth, err := auth.TenantManager.AuthForTenant(tenantID)
	if err != nil {
		return err
	}

	batch := fb.NewAutomaticWriteBatch(fs, 100)
	iter := tenantAuth.Users(ctx, "")

	for {
		userRecord, err := iter.Next()
		if err != nil {
			if err == iterator.Done {
				if errs := batch.Commit(ctx); len(errs) > 0 {
					return errs[0]
				}

				return nil
			}

			return err
		}

		lastLoginTimestamp := common.EpochMillisecondsToTime(userRecord.UserMetadata.LastLogInTimestamp)

		providerID := "no-provider"
		if len(userRecord.ProviderUserInfo) > 0 {
			providerID = userRecord.ProviderUserInfo[0].ProviderID
		}

		if providerID == "google.com" || providerID == "microsoft.com" {
			// we don't need to revoke google tokens, as we can disable the account
			continue
		}

		if time.Since(lastLoginTimestamp) < dayDuration {
			// If the user logged-in in the last day, don't do anything
			continue
		}

		tokenValidAfterTime := common.EpochMillisecondsToTime(userRecord.TokensValidAfterMillis)
		if time.Since(tokenValidAfterTime) >= MaxValidRefreshTokenDuration {
			// Revoke the user's refresh token and force him to log in again if his
			// refresh token is valid more than the max allowed duration
			if err := tenantAuth.RevokeRefreshTokens(ctx, userRecord.UID); err != nil {
				l := fmt.Sprintf("revoke refresh token email [%s] provider [%s] uid [%s] tenant [%s]", userRecord.Email, providerID, userRecord.UID, tenantID)
				logger.Errorf("%s failed with error: %s", l, err)
			}
		}
	}
}
func (h *Tasks) cleanFlexsaveAsssetsForAWS(ctx context.Context) {
	log.Printf("Cleaning assets for aws flexsave")
	err := h.awsAssetsService.ClearAllFlexsaveAssets(ctx)
	if err != nil {
		log.Printf("fail to clean flexsave assets for aws")
		log.Println(err)

		return
	}
}

func (h *Tasks) AssetCleanupHandler(ctx *gin.Context) error {
	go verifyEntityAssetSettingsIntegrity(ctx)
	go h.cleanAssetsRenewals(ctx)

	// 3 days ago
	t1 := time.Now().Add(time.Duration(-72) * time.Hour)
	go h.cleanAssetsByType(common.Assets.GSuite, t1)
	go h.cleanAssetsByType(common.Assets.Office365, t1)
	go h.cleanAssetsByType(common.Assets.MicrosoftAzure, t1)
	go h.cleanAssetsByType(common.Assets.GoogleCloud, t1)

	// 4 hours ago
	t2 := time.Now().Add(time.Duration(-4) * time.Hour)
	go h.cleanAssetsByType(common.Assets.GoogleCloudProject, t2)

	// 45 days ago (to complete invoicing before removal)
	t3 := time.Now().Add(time.Duration(-1080) * time.Hour)
	go h.cleanAssetsByType(common.Assets.AmazonWebServices, t3)
	go h.cleanFlexsaveAsssetsForAWS(ctx)
	return web.Respond(ctx, nil, http.StatusOK)
}

func (h *Tasks) cleanAssetsRenewals(ctx context.Context) {
	maxAge := time.Now().Add(time.Duration(-24*7) * time.Hour)

	fs := h.Firestore(ctx)

	bulkWriter := fs.BulkWriter(ctx)

	assetIterator := fs.Collection("dashboards").Doc("assetsRenewals").Collection("ids").Where("timestamp", "<", maxAge).Select().Documents(ctx)
	defer assetIterator.Stop()

	for {
		docSnap, err := assetIterator.Next()
		if err == iterator.Done {
			bulkWriter.End()
			log.Println("[COMPLETE] Cleaning assets renewals")
			return
		}

		if err != nil {
			log.Printf("Error [%s]: %s", docSnap.Ref.ID, err.Error())
			log.Println(err)

			return
		}

		if _, err := bulkWriter.Delete(docSnap.Ref); err != nil {
			log.Println(err)
			return
		}
	}
}

func (h *Tasks) cleanAssetsByType(assetType string, maxAge time.Time) {
	ctx := context.Background()

	log.Printf("Cleaning assets by type %s", assetType)

	fs := h.Firestore(ctx)

	bulkWriter := fs.BulkWriter(ctx)

	assetIterator := fs.CollectionGroup("assetMetadata").Where("type", "==", assetType).Where("lastUpdated", "<", maxAge).Select().Documents(ctx)
	defer assetIterator.Stop()

	for {
		docSnap, err := assetIterator.Next()
		if err != nil {
			if err == iterator.Done {
				bulkWriter.End()
				log.Printf("[COMPLETE] Cleaning assets by type %s", assetType)
				return
			}

			log.Println(err)

			return
		}

		assetRef := docSnap.Ref.Parent.Parent
		log.Printf("Removing stale %s asset [%s]", assetType, assetRef.ID)
		if _, err := bulkWriter.Delete(docSnap.Ref); err != nil {
			log.Println(err)
			return
		}

		if _, err := bulkWriter.Delete(assetRef); err != nil {
			log.Println(err)
			return
		}

	}
}

func verifyEntityAssetSettingsIntegrity(ctx context.Context) {
	var pageSize = 200

	var startAfter *firestore.DocumentSnapshot

	fs := common.GetFirestoreClient(ctx)

	entities, err := selectActiveEntities(ctx, fs)
	if err != nil {
		log.Println(err)
		return
	}

	if len(entities) < 800 {
		log.Println("could not fetch all entities")
		return
	}

	batch := fs.Batch()
	hasPendingWrites := false

	query := fs.Collection("assetSettings").Limit(pageSize).Select("entity")

	for {
		docs, err := query.StartAfter(startAfter).Documents(ctx).GetAll()
		if err != nil {
			log.Println(err.Error())
			return
		}

		if len(docs) > 0 {
			startAfter = docs[len(docs)-1]

			for _, docSnap := range docs {
				if e, err := docSnap.DataAt("entity"); err != nil {
					// Asset setting has no entity field
					// log.Printf(" * Asset %s - (%s)", docSnap.Ref.ID, err)
				} else {
					if entity, ok := e.(*firestore.DocumentRef); ok {
						// Entity field is set up correctly
						if _, ok := entities[entity.ID]; ok {
						} else {
							log.Printf(" ** Asset %s Entity %s (Entity does not exist or disabled)", docSnap.Ref.ID, entity.ID)
							batch.Update(docSnap.Ref, []firestore.Update{
								{
									Path:  "entity",
									Value: nil,
								},
								{
									Path:  "contract",
									Value: nil,
								},
								{
									Path:  "bucket",
									Value: nil,
								},
							})

							assetRef := fs.Collection("assets").Doc(docSnap.Ref.ID)
							batch.Update(assetRef, []firestore.Update{
								{
									Path:  "entity",
									Value: nil,
								},
								{
									Path:  "contract",
									Value: nil,
								},
								{
									Path:  "bucket",
									Value: nil,
								},
							})

							hasPendingWrites = true
						}
					}
				}
			}

			if hasPendingWrites {
				if _, err := batch.Commit(ctx); err != nil {
					log.Println(err.Error())
				}

				batch = fs.Batch()
				hasPendingWrites = false
			}
		} else {
			return
		}
	}
}

func selectActiveEntities(ctx context.Context, fs *firestore.Client) (map[string]*firestore.DocumentSnapshot, error) {
	var pageSize = 500

	var startAfter *firestore.DocumentSnapshot

	entitiesMap := make(map[string]*firestore.DocumentSnapshot)

	query := fs.Collection("entities").Where("active", "==", true).Limit(pageSize).Select()

	for {
		docs, err := query.StartAfter(startAfter).Documents(ctx).GetAll()
		if err != nil {
			return nil, err
		}

		if len(docs) > 0 {
			startAfter = docs[len(docs)-1]

			for _, docSnap := range docs {
				if docSnap.Exists() {
					entitiesMap[docSnap.Ref.ID] = docSnap
				}
			}
		} else {
			return entitiesMap, nil
		}
	}
}
