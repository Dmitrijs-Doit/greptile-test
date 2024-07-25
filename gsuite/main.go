package gsuite

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"
	reseller "google.golang.org/api/reseller/v1"

	"github.com/doitintl/hello/scheduled-tasks/assets"
	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Asset struct {
	AssetType  string                 `firestore:"type"`
	Properties *AssetProperties       `firestore:"properties"`
	Bucket     *firestore.DocumentRef `firestore:"bucket"`
	Contract   *firestore.DocumentRef `firestore:"contract"`
	Customer   *firestore.DocumentRef `firestore:"customer"`
	Entity     *firestore.DocumentRef `firestore:"entity"`
	Tags       []string               `firestore:"tags"`
}

type AssetProperties struct {
	CustomerDomain string           `firestore:"customerDomain"`
	CustomerID     string           `firestore:"customerId"`
	Reseller       string           `firestore:"reseller"`
	Subscription   *Subscription    `firestore:"subscription"`
	Settings       *assets.Settings `firestore:"settings"`
}

type Subscription struct {
	BillingMethod   string                   `firestore:"billingMethod,omitempty"`
	CreationTime    int64                    `firestore:"creationTime,omitempty"`
	PurchaseOrderID string                   `firestore:"purchaseOrderId,omitempty"`
	ResourceUIURL   string                   `firestore:"resourceUiUrl,omitempty"`
	SkuID           string                   `firestore:"skuId"`
	SkuName         string                   `firestore:"skuName,omitempty"`
	Status          string                   `firestore:"status,omitempty"`
	ID              string                   `firestore:"subscriptionId"`
	Plan            *assets.SubscriptionPlan `firestore:"plan"`
	RenewalSettings *RenewalSettings         `firestore:"renewalSettings,omitempty"`
	Seats           *Seats                   `firestore:"seats"`
	TrialSettings   *TrialSettings           `firestore:"trialSettings"`
}

type RenewalSettings struct {
	RenewalType string `firestore:"renewalType"`
}

type Seats struct {
	LicensedNumberOfSeats int64 `firestore:"licensedNumberOfSeats"`
	MaximumNumberOfSeats  int64 `firestore:"maximumNumberOfSeats"`
	NumberOfSeats         int64 `firestore:"numberOfSeats"`
}

type TrialSettings struct {
	IsInTrial    bool  `firestore:"isInTrial"`
	TrialEndTime int64 `firestore:"trialEndTime"`
}

const (
	maxResults = 100
)

func (a Asset) GetCacheKey() string {
	return fmt.Sprintf("%s-%s-%s", a.AssetType, a.Entity.ID, a.Properties.Subscription.ID)
}

func (a Asset) ModifyContractQuery(query *firestore.Query) firestore.Query {
	return query.Where("type", "==", a.AssetType)
}

func (a Asset) ContractPredicate(contract *common.Contract) (bool, bool) {
	if len(contract.Assets) > 0 {
		docID := fmt.Sprintf("%s-%s", common.Assets.GSuite, a.Properties.Subscription.ID)
		for _, ref := range contract.Assets {
			if ref != nil && ref.ID == docID {
				return true, false
			}
		}

		return false, false
	}

	return true, true
}

func SubscriptionsListHandler(ctx *gin.Context) {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)

	inventoryChan := make(chan *InventoryItem)

	now := time.Now().UTC()
	y, m, d := now.Date()
	date := time.Date(y, m, d-1, 0, 0, 0, 0, time.UTC)
	inventoryExpireBy := time.Date(y+2, m, 1, 0, 0, 0, 0, time.UTC)

	domains := []string{""}

	customerID := ctx.Param("customerID")
	if customerID != "" {
		docSnap, err := fs.Collection("customers").Doc(customerID).Get(ctx)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		var customer common.Customer
		if err := docSnap.DataTo(&customer); err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		domains = customer.Domains
	}

	assetType := common.Assets.GSuite
	assetSettings := make(map[string]*assets.AssetSettings)

	query := fs.Collection("assetSettings").Where("type", "==", assetType)
	if customerID != "" {
		query = query.Where("customer", "==", fs.Collection("customers").Doc(customerID))
	}

	assetSettingsIter := query.Documents(ctx)
	defer assetSettingsIter.Stop()

	for {
		docSnap, err := assetSettingsIter.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			log.Printf("Error: (Firestore) %s", err.Error())
			return
		}

		var asDoc assets.AssetSettings
		if err := docSnap.DataTo(&asDoc); err != nil {
			log.Printf("Error: %s", err.Error())
		} else {
			assetSettings[docSnap.Ref.ID] = &asDoc
		}
	}

	go persistInventory(ctx, l, inventoryChan)

	defer func() {
		inventoryChan <- nil
	}()

	for index, resellerService := range Resellers {
		customersSubscriptionsMap := make(map[string][]*reseller.Subscription)
		listCall := resellerService.Subscriptions.List().MaxResults(maxResults)

		for _, domain := range domains {
			if domain != "" {
				listCall = listCall.CustomerNamePrefix(domain)
			}

			if err := listCall.Pages(ctx, func(response *reseller.Subscriptions) error {
				for _, subscription := range response.Subscriptions {
					customersSubscriptionsMap[subscription.CustomerId] = append(customersSubscriptionsMap[subscription.CustomerId], subscription)
				}
				time.Sleep(time.Millisecond * time.Duration(250))
				return nil
			}); err != nil {
				log.Println(err)
				ctx.Status(http.StatusInternalServerError)

				continue
			}
		}

		entitiesCache := make(map[string][]*firestore.DocumentRef)
		contractCache := make(map[string]*firestore.DocumentRef)

		batch := fs.Batch()
		batchSize := 0

		for _, subscriptions := range customersSubscriptionsMap {
			domain := subscriptions[0].CustomerDomain
			q := fs.Collection("customers").Where("domains", "array-contains", strings.ToLower(domain))

			iter := q.Limit(1).Documents(ctx)
			defer iter.Stop()

			for {
				customerRef := fb.Orphan

				docSnap, err := iter.Next()
				if err == iterator.Done {
					log.Printf("Error: No customer found for: %s", domain)
				} else if err != nil {
					log.Printf("Error: (Firestore %s) %s", domain, err.Error())
					break
				} else {
					customerRef = docSnap.Ref
				}

				for _, sub := range subscriptions {
					docID := fmt.Sprintf("%s-%s", assetType, sub.SubscriptionId)
					assetRef := fs.Collection("assets").Doc(docID)
					assetSettingsRef := fs.Collection("assetSettings").Doc(docID)

					// Remove non-active assets
					if !SubscriptionActive(sub.Status) && !SubscriptionSuspended(sub.Status) {
						batch.Delete(assetRef)

						batchSize++

						continue
					}

					// Filter out Drive with 0 seats
					if strings.HasPrefix(sub.SkuId, "Google-Drive") || sub.SkuId == GoogleVault {
						if sub.Plan.IsCommitmentPlan {
							if sub.Seats.LicensedNumberOfSeats == 0 && sub.Seats.NumberOfSeats == 0 {
								batch.Delete(assetRef)

								batchSize++

								continue
							}
						} else {
							if sub.Seats.LicensedNumberOfSeats == 0 && sub.Seats.MaximumNumberOfSeats == 0 {
								batch.Delete(assetRef)

								batchSize++

								continue
							}
						}
					}

					var (
						commitmentInterval *assets.SubscriptionPlanCommitmentInterval
						renewalSettings    *RenewalSettings
						trialSettings      *TrialSettings
						seats              *Seats
					)

					if sub.Plan.IsCommitmentPlan {
						if sub.Plan.CommitmentInterval != nil {
							commitmentInterval = &assets.SubscriptionPlanCommitmentInterval{
								StartTime: sub.Plan.CommitmentInterval.StartTime,
								EndTime:   sub.Plan.CommitmentInterval.EndTime,
							}
						}

						if sub.RenewalSettings != nil {
							renewalSettings = &RenewalSettings{
								RenewalType: sub.RenewalSettings.RenewalType,
							}
						}
					}

					if sub.TrialSettings != nil {
						trialSettings = &TrialSettings{
							IsInTrial:    sub.TrialSettings.IsInTrial,
							TrialEndTime: sub.TrialSettings.TrialEndTime,
						}
					}

					if sub.Seats != nil {
						seats = &Seats{
							LicensedNumberOfSeats: sub.Seats.LicensedNumberOfSeats,
							MaximumNumberOfSeats:  sub.Seats.MaximumNumberOfSeats,
							NumberOfSeats:         sub.Seats.NumberOfSeats,
						}
					}

					props := &AssetProperties{
						CustomerDomain: domain,
						CustomerID:     sub.CustomerId,
						Reseller:       Subjects[index],
						Subscription: &Subscription{
							BillingMethod: sub.BillingMethod,
							CreationTime:  sub.CreationTime,
							Plan: &assets.SubscriptionPlan{
								CommitmentInterval: commitmentInterval,
								IsCommitmentPlan:   sub.Plan.IsCommitmentPlan,
								PlanName:           sub.Plan.PlanName,
							},
							RenewalSettings: renewalSettings,
							PurchaseOrderID: sub.PurchaseOrderId,
							ResourceUIURL:   sub.ResourceUiUrl,
							Seats:           seats,
							SkuID:           sub.SkuId,
							SkuName:         sub.SkuName,
							Status:          sub.Status,
							TrialSettings:   trialSettings,
							ID:              sub.SubscriptionId,
						},
					}
					asset := Asset{
						AssetType:  assetType,
						Customer:   customerRef,
						Properties: props,
					}

					paths := []firestore.FieldPath{[]string{"type"}, []string{"properties"}, []string{"customer"}}

					if customerRef.ID != fb.Orphan.ID {
						// Found existing asset settings
						as := assetSettings[docID]
						if as == nil {
							as = &assets.AssetSettings{
								AssetType: assetType,
							}
						}

						if as.Settings != nil {
							asset.Properties.Settings = as.Settings

							// Alert assets past their end date
							if as.Settings.Plan != nil && as.Settings.Plan.IsCommitmentPlan && as.Settings.Plan.CommitmentInterval != nil {
								endTime := common.EpochMillisecondsToTime(as.Settings.Plan.CommitmentInterval.EndTime)
								if now.After(endTime) && now.Hour() > 10 {
									l.Errorf("%s invalid asset settings for %s (%s)", docID, props.Subscription.SkuName, props.CustomerDomain)
								}
							}
						}

						asset.Tags = as.Tags
						asset.Bucket = as.Bucket

						paths = append(paths, []string{"bucket"}, []string{"tags"})

						// Check if entity assignment needs to be updated on asset and asset settings doc
						if entityRef, update := common.GetAssetEntity(ctx, fs, as.Customer, as.Entity, customerRef, entitiesCache); update {
							assetSettingsUpdate := map[string]interface{}{
								"type":     assetType,
								"customer": customerRef,
								"entity":   entityRef,
							}
							asset.Entity = entityRef

							paths = append(paths, []string{"entity"})

							if contractRef, update := common.GetAssetContract(ctx, fs, asset, customerRef, entityRef, contractCache); update {
								assetSettingsUpdate["contract"] = contractRef
								asset.Contract = contractRef

								paths = append(paths, []string{"contract"})
							}

							batch.Set(assetSettingsRef, assetSettingsUpdate, firestore.MergeAll)

							batchSize++
						} else if as.Customer != nil && as.Entity != nil {
							asset.Entity = as.Entity

							paths = append(paths, []string{"entity"})

							if contractRef, update := common.GetAssetContract(ctx, fs, asset, as.Customer, as.Entity, contractCache); update {
								assetSettingsUpdate := map[string]interface{}{
									"type":     assetType,
									"contract": contractRef,
								}

								paths = append(paths, []string{"contract"})
								asset.Contract = contractRef

								batch.Set(assetSettingsRef, assetSettingsUpdate, firestore.MergeAll)

								batchSize++
							}
						}
					} else {
						// Could not find customer, update settings to reference orphan customer and reset entity, contract
						batch.Set(assetSettingsRef, map[string]interface{}{
							"customer": customerRef,
							"entity":   nil,
							"contract": nil,
							"bucket":   nil,
							"type":     assetType,
						}, firestore.MergeAll)

						asset.Entity = nil
						asset.Contract = nil
						asset.Bucket = nil

						paths = append(paths, []string{"entity"}, []string{"contract"}, []string{"bucket"})
						batchSize++
					}

					batch.Set(assetRef, asset, firestore.Merge(paths...))
					batch.Set(assetRef.Collection("assetMetadata").Doc("metadata"), map[string]interface{}{
						"lastUpdated": firestore.ServerTimestamp,
						"type":        asset.AssetType,
					})

					batchSize += 2

					if now.Hour() < 3 {
						inventoryChan <- &InventoryItem{
							Settings:       asset.Properties.Settings,
							Subscription:   asset.Properties.Subscription,
							Customer:       asset.Customer,
							Entity:         asset.Entity,
							Contract:       asset.Contract,
							CustomerID:     sub.CustomerId,
							CustomerDomain: domain,
							SubscriptionID: asset.Properties.Subscription.ID,
							Timestamp:      time.Time{},
							Date:           date,
							Reseller:       Subjects[index],
							ExpireBy:       &inventoryExpireBy,
						}
					}
				}

				break
			}

			if batchSize >= 100 {
				if _, err := batch.Commit(ctx); err != nil {
					log.Println(err)
				}

				batch = fs.Batch()
				batchSize = 0
			}
		}

		if batchSize > 0 {
			if _, err := batch.Commit(ctx); err != nil {
				log.Println(err)
			}
		}
	}

	ctx.Status(http.StatusOK)
}

func SubscriptionActive(status string) bool {
	return status == "ACTIVE"
}

func SubscriptionSuspended(status string) bool {
	return status == "SUSPENDED"
}
