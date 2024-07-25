package microsoft

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/assets"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/google/uuid"
)

func (a Asset) GetCacheKey() string {
	return fmt.Sprintf("%s-%s", a.BaseAsset.AssetType, a.Entity.ID)
}

func (a Asset) ModifyContractQuery(query *firestore.Query) firestore.Query {
	return query.Where("type", "==", a.BaseAsset.AssetType)
}

func (a Asset) ContractPredicate(*common.Contract) (bool, bool) {
	return true, true
}

func SubscriptionsListHandler(ctx *gin.Context) {
	l := logger.FromContext(ctx)
	fs := common.GetFirestoreClient(ctx)

	now := time.Now().UTC()

	assetSettings := make(map[string]*assets.AssetSettings)
	iter := fs.Collection("assetSettings").Where("type", "==", common.Assets.Office365).Documents(ctx)

	for {
		docSnap, err := iter.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			log.Printf("Error: (Firestore) %s", err.Error())
			break
		}

		var asDoc assets.AssetSettings
		if err := docSnap.DataTo(&asDoc); err != nil {
			log.Printf("Error: %s", err.Error())
		}

		assetSettings[docSnap.Ref.ID] = &asDoc
	}

	for i, accessToken := range MPCAccessTokens {
		customers, err := accessToken.listCustomers()
		if err != nil {
			l.Errorf("error while listing %s csp customers", accessToken.GetDomain())
			ctx.AbortWithError(http.StatusInternalServerError, err)

			continue
		}

		if err := handleMicrosoft365Assets(ctx, fs, accessToken, now, customers, assetSettings); err != nil {
			l.Errorf("error while updating %s csp subscriptions", accessToken.GetDomain())
			ctx.AbortWithError(http.StatusInternalServerError, err)
		}

		handleMicrosoftAzureAssets(ctx, ASMAccessTokens[i], AzureBillingAccounts[accessToken.GetDomain()], customers.Items)
	}
}

func handleMicrosoft365Assets(ctx *gin.Context, fs *firestore.Client, mpcAccessToken *AccessToken, now time.Time, customers *Customers, assetSettings map[string]*assets.AssetSettings) error {
	l := logger.FromContext(ctx)

	reseller := mpcAccessToken.GetDomain()
	y, m, d := now.Date()
	date := time.Date(y, m, d-1, 0, 0, 0, 0, time.UTC)
	inventoryExpireBy := time.Date(y+2, m, 1, 0, 0, 0, 0, time.UTC)

	inventoryChan := make(chan *InventoryItem)
	entitiesCache := make(map[string][]*firestore.DocumentRef)
	contractCache := make(map[string]*firestore.DocumentRef)

	go persistInventory(ctx, l, inventoryChan)

	defer func() {
		inventoryChan <- nil
	}()

	assetType := common.Assets.Office365
	batch := fb.NewAutomaticWriteBatch(fs, 250)

	for _, customer := range customers.Items {
		domain := strings.ToLower(strings.TrimSpace(customer.CompanyProfile.Domain))

		subscriptions, err := mpcAccessToken.ListCustomerSubscriptions(customer)
		if err != nil {
			l.Errorf("error fetching microsoft 365 subscriptions for %s: %v", customer.CompanyProfile.Domain, err)
			continue
		}

		l.Debugf("domain: %-32s\tsubscriptions: %d", domain, subscriptions.TotalCount)

		if subscriptions.TotalCount <= 0 {
			continue
		}

		if signed, err := mpcAccessToken.signedCustomerAgreements(customer); err != nil {
			l.Error(err)
		} else {
			if !signed {
				batch.Set(fs.Collection("integrations").Doc("microsoft").Collection("unsignedMicrosoftAgreements").Doc(customer.ID), map[string]interface{}{}, firestore.MergeAll)
			}
		}

		customerRef, err := getCustomerRefByDomain(ctx, fs, domain)
		if err != nil {
			ctx.Error(err)
			continue
		}

		for _, subscription := range subscriptions.Items {
			docID := fmt.Sprintf("%s-%s", assetType, subscription.ID)
			assetRef := fs.Collection("assets").Doc(docID)
			assetSettingsRef := fs.Collection("assetSettings").Doc(docID)

			if subscription.OfferID == common.MicrosoftAzureOfferID ||
				strings.HasPrefix(subscription.OfferID, common.MicrosoftAzurePlanOfferIDPrefix) ||
				subscription.OfferName == "Azure plan" {
				batch.Delete(assetRef)
				continue
			}

			if subscription.Status == "active" {
				props := &AssetProperties{
					CustomerDomain: domain,
					CustomerID:     customer.ID,
					Subscription:   subscription,
					Reseller:       reseller,
				}

				asset := Asset{
					BaseAsset: pkg.BaseAsset{
						AssetType: assetType,
						Customer:  customerRef,
					},
					Properties: props,
				}

				paths := []firestore.FieldPath{[]string{"type"}, []string{"properties"}, []string{"customer"}}

				// Found real customer
				if customerRef.ID != fb.Orphan.ID {
					// Found existing asset settings
					if as, ok := assetSettings[docID]; ok {
						if as.Settings != nil {
							asset.Tags = as.Tags
							asset.Properties.Settings = as.Settings
						}

						// Copy bucket on asset
						asset.Bucket = as.Bucket

						paths = append(paths, []string{"bucket"})

						// Check if entity assignment needs to be updated on asset and asset settings doc
						if entityRef, update := common.GetAssetEntity(ctx, fs, as.Customer, as.Entity, customerRef, entitiesCache); update {
							assetSettingsUpdate := map[string]interface{}{
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
						} else if as.Customer != nil && as.Entity != nil {
							asset.Entity = as.Entity

							paths = append(paths, []string{"entity"})

							if contractRef, update := common.GetAssetContract(ctx, fs, asset, as.Customer, as.Entity, contractCache); update {
								assetSettingsUpdate := map[string]interface{}{
									"contract": contractRef,
								}

								paths = append(paths, []string{"contract"})
								asset.Contract = contractRef

								batch.Set(assetSettingsRef, assetSettingsUpdate, firestore.MergeAll)
							}
						} else {
							// log.Printf("Not updating contract")
						}
					} else {
						// Could not find asset settings doc, then initialize it with type only
						batch.Set(assetSettingsRef, map[string]interface{}{
							"type": assetType,
						}, firestore.MergeAll)
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
				}

				batch.Set(assetRef, asset, firestore.Merge(paths...))
				batch.Set(assetRef.Collection("assetMetadata").Doc("metadata"), map[string]interface{}{
					"lastUpdated": firestore.ServerTimestamp,
					"type":        asset.AssetType,
				})

				if assetType == common.Assets.Office365 && now.Hour() < 3 {
					inventoryChan <- &InventoryItem{
						Settings:       asset.Properties.Settings,
						Subscription:   asset.Properties.Subscription,
						Customer:       asset.Customer,
						Entity:         asset.Entity,
						Contract:       asset.Contract,
						CustomerID:     customer.ID,
						CustomerDomain: domain,
						SubscriptionID: asset.Properties.Subscription.ID,
						Reseller:       reseller,
						Timestamp:      time.Time{},
						Date:           date,
						ExpireBy:       &inventoryExpireBy,
					}
				}
			} else {
				batch.Delete(assetRef)
			}
		}

		if errs := batch.Commit(ctx); len(errs) > 0 {
			for _, err := range errs {
				l.Errorf("batch.Commit err: %v", err)
			}
		}
	}

	return nil
}

// List customers of Microsoft Partner Center
func (a *AccessToken) listCustomers() (*Customers, error) {
	if err := a.Refresh(); err != nil {
		return nil, err
	}

	client := http.DefaultClient
	urlStr := fmt.Sprintf("%s/v1/customers", a.Resource)
	req, _ := http.NewRequest("GET", urlStr, nil)
	requestID, _ := uuid.NewRandom()
	correlationID, _ := uuid.NewRandom()

	req.Header.Set("Authorization", fmt.Sprintf("%s %s", a.TokenType, a.AccessToken))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("MS-RequestId", requestID.String())
	req.Header.Set("MS-CorrelationId", correlationID.String())

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		var customers Customers
		if err := json.Unmarshal(stripBOM(respBody), &customers); err != nil {
			return nil, err
		}

		return &customers, nil
	}

	return nil, errors.New("failed to list customers")
}

func (a *AccessToken) ListCustomerSubscriptions(customer *Customer) (*Subscriptions, error) {
	if err := a.Refresh(); err != nil {
		return nil, err
	}

	client := http.Client{
		Timeout: 300 * time.Second,
	}
	urlStr := fmt.Sprintf("%s/v1/customers/%s/subscriptions", a.Resource, customer.ID)
	req, _ := http.NewRequest("GET", urlStr, nil)
	requestID, _ := uuid.NewRandom()
	correlationID, _ := uuid.NewRandom()

	req.Header.Set("Authorization", fmt.Sprintf("%s %s", a.TokenType, a.AccessToken))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("MS-RequestId", requestID.String())
	req.Header.Set("MS-CorrelationId", correlationID.String())

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		var subscriptions Subscriptions
		if err := json.Unmarshal(stripBOM(respBody), &subscriptions); err != nil {
			return nil, err
		}

		return &subscriptions, nil
	}

	// log.Println(string(respBody))
	// log.Println(resp.StatusCode)
	return nil, fmt.Errorf("failed to get customer subscriptions for %s", customer.CompanyProfile.Domain)
}

type AgreementsResponse struct {
	TotalCount int64        `json:"totalCount"`
	Items      []*Agreement `json:"items"`
}

type Agreement struct {
	TemplateID     string                  `json:"templateId"`
	DateAgreed     string                  `json:"dateAgreed"`
	Type           string                  `json:"type"`
	AgreementLink  string                  `json:"agreementLink"`
	PrimaryContact AgreementPrimaryContact `json:"primaryContact"`
}

type AgreementPrimaryContact struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Phone     string `json:"phoneNumber"`
}

func (a *AccessToken) signedCustomerAgreements(customer *Customer) (bool, error) {
	if err := a.Refresh(); err != nil {
		return false, err
	}

	client := http.Client{
		Timeout: 300 * time.Second,
	}
	urlStr := fmt.Sprintf("%s/v1/customers/%s/agreements?agreementType=MicrosoftCustomerAgreement", a.Resource, customer.ID)
	req, _ := http.NewRequest("GET", urlStr, nil)
	requestID, _ := uuid.NewRandom()
	correlationID, _ := uuid.NewRandom()

	req.Header.Set("Authorization", fmt.Sprintf("%s %s", a.TokenType, a.AccessToken))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("MS-RequestId", requestID.String())
	req.Header.Set("MS-CorrelationId", correlationID.String())

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}

	respBody, err := io.ReadAll(resp.Body)

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var agreementsResp AgreementsResponse
		if err := json.Unmarshal(stripBOM(respBody), &agreementsResp); err != nil {
			return false, err
		}

		return agreementsResp.TotalCount > 0, nil
	}

	return false, fmt.Errorf("failed to get signed customer agreements for %s", customer.CompanyProfile.Domain)
}
