package microsoft

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/common"
	fb "github.com/doitintl/hello/scheduled-tasks/firebase"
)

type AzureCustomersResult struct {
	NextLink   string           `json:"nextLink"`
	TotalCount int              `json:"totalCount"`
	Value      []*AzureCustomer `json:"value"`
}

type AzureSubscriptions struct {
	Value []*AzureSubscription `json:"value"`
}

type AzureCustomer struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Properties struct {
		BillingProfileDisplayName string `json:"billingProfileDisplayName"`
		BillingProfileID          string `json:"billingProfileId"`
		DisplayName               string `json:"displayName"`
	} `json:"properties"`
}

type AzureSubscription struct {
	ID         string                       `json:"id"`
	Name       string                       `json:"name"`
	Type       string                       `json:"type"`
	Properties *AzureSubscriptionProperties `json:"properties" firestore:"properties"`
}

type AzureSubscriptionProperties struct {
	BillingProfileDisplayName string `json:"billingProfileDisplayName" firestore:"billingProfileDisplayName"`
	BillingProfileID          string `json:"billingProfileId" firestore:"billingProfileId"`
	CustomerDisplayName       string `json:"customerDisplayName" firestore:"customerDisplayName"`
	CustomerID                string `json:"customerId" firestore:"customerId"`
	SubscriptionID            string `json:"subscriptionId" firestore:"subscriptionId"`
	DisplayName               string `json:"displayName" firestore:"displayName"`
	SubscriptionBillingStatus string `json:"subscriptionBillingStatus" firestore:"subscriptionBillingStatus"`
	SkuID                     string `json:"skuId" firestore:"skuId"`
	SkuDescription            string `json:"skuDescription" firestore:"skuDescription"`
}

type AzureAsset struct {
	AssetType  string                 `firestore:"type"`
	Properties *AzureAssetProperties  `firestore:"properties"`
	Bucket     *firestore.DocumentRef `firestore:"bucket"`
	Contract   *firestore.DocumentRef `firestore:"contract"`
	Customer   *firestore.DocumentRef `firestore:"customer"`
	Entity     *firestore.DocumentRef `firestore:"entity"`
	Tags       []string               `firestore:"tags"`
}

type AzureAssetProperties struct {
	CustomerDomain string                       `firestore:"customerDomain"`
	CustomerID     string                       `firestore:"customerId"`
	Reseller       CSPDomain                    `firestore:"reseller"`
	Subscription   *AzureSubscriptionProperties `firestore:"subscription"`
}

type handleSubscriptionInputParams struct {
	fs           *firestore.Client
	batch        *firestore.WriteBatch
	ctx          *gin.Context
	subscription *AzureSubscription
	domain       string
	customer     *Customer
	reseller     CSPDomain
	customerRef  *firestore.DocumentRef
	batchSize    int
}

const (
	AzureAPIVersion = "2019-10-01-preview"

	AzureQueryAPIVersion = "2019-11-01"
)

func findCustomer(customers []*Customer, customerID string) *Customer {
	for _, customer := range customers {
		if customer.ID == customerID {
			return customer
		}
	}

	return nil
}

func handleMicrosoftAzureAssets(ctx *gin.Context, asmAccessToken *AccessToken, billingAccount AzureBillingAccount, customers []*Customer) {
	fs := common.GetFirestoreClient(ctx)

	azureCustomers, err := asmAccessToken.listAllAzureCustomers(billingAccount)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	reseller := asmAccessToken.GetDomain()
	batch := fs.Batch()
	batchSize := 0

	for _, azureCustomer := range azureCustomers {
		customer := findCustomer(customers, azureCustomer.Name)
		if customer == nil {
			err := fmt.Errorf("azure customer not found (%s)", azureCustomer.Properties.DisplayName)
			ctx.Error(err)

			continue
		}

		domain := strings.ToLower(strings.TrimSpace(customer.CompanyProfile.Domain))

		customerRef, err := getCustomerRefByDomain(ctx, fs, domain)
		if err != nil {
			ctx.Error(err)
			continue
		}

		subscriptions, err := asmAccessToken.listCustomerAzureSubscriptions(azureCustomer)
		if err != nil {
			err := fmt.Errorf("%s (%s)", err.Error(), domain)
			ctx.Error(err)

			continue
		}

		for _, subscription := range subscriptions {
			// handleSubscriptionParams := handleSubscriptionInputParams{
			// 	fs, batch, ctx, subscription, *domain, customer, reseller, customerRef, batchSize,
			// }
			// if err := handleSubscription(handleSubscriptionParams); err != nil {
			// 	ctx.Error(err)
			// 	continue
			// }
			assetType := common.Assets.MicrosoftAzure
			docID := fmt.Sprintf("%s-%s", assetType, subscription.Name)
			assetRef := fs.Collection("assets").Doc(docID)
			assetSettingsRef := fs.Collection("assetSettings").Doc(docID)

			if subscription.Properties.SubscriptionBillingStatus == "Active" {
				props := &AzureAssetProperties{
					CustomerDomain: domain,
					CustomerID:     customer.ID,
					Subscription:   subscription.Properties,
					Reseller:       reseller,
				}
				asset := AzureAsset{
					AssetType:  assetType,
					Customer:   customerRef,
					Properties: props,
				}

				if customerRef.ID != fb.Orphan.ID {
					docSnap, err := assetSettingsRef.Get(ctx)
					if err != nil {
						if status.Code(err) != codes.NotFound {
							ctx.Error(err)
							continue
						}
					}

					if docSnap.Exists() {
						var as common.AssetSettings

						var contractRef *firestore.DocumentRef

						if err := docSnap.DataTo(&as); err != nil {
							fmt.Println("[ERROR]", err)
							continue
						} else {
							asset.Bucket = as.Bucket
							asset.Tags = as.Tags

							if entityRef, update := common.GetAssetEntity(ctx, fs, as.Customer, as.Entity, customerRef, nil); update {
								batch.Set(assetSettingsRef, map[string]interface{}{
									"customer": customerRef,
									"entity":   entityRef,
									"contract": contractRef,
								}, firestore.MergeAll)

								asset.Entity = entityRef
								asset.Contract = contractRef
							} else if as.Customer != nil && as.Entity != nil {
								batch.Set(assetSettingsRef, map[string]interface{}{
									"contract": contractRef,
								}, firestore.MergeAll)

								asset.Entity = as.Entity
								asset.Contract = contractRef
							}
						}
					} else {
						batch.Set(assetSettingsRef, map[string]interface{}{
							"customer": customerRef,
							"entity":   nil,
							"contract": nil,
							"bucket":   nil,
							"type":     assetType,
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
					batchSize++
				}

				batch.Set(assetRef, asset)
				batch.Set(assetRef.Collection("assetMetadata").Doc("metadata"), map[string]interface{}{
					"lastUpdated": firestore.ServerTimestamp,
					"type":        asset.AssetType,
				})

				batchSize += 2
			} else {
				batch.Delete(assetRef)

				batchSize++
			}
		}

		if batchSize >= 100 {
			if _, err := batch.Commit(ctx); err != nil {
				ctx.Error(err)
			}

			batch = fs.Batch()
			batchSize = 0
		}
	}

	if batchSize > 0 {
		if _, err := batch.Commit(ctx); err != nil {
			ctx.Error(err)
		}
	}
}

func handleSubscription(params handleSubscriptionInputParams) error {
	assetType := common.Assets.MicrosoftAzure
	docID := fmt.Sprintf("%s-%s", assetType, params.subscription.Name)
	assetRef := params.fs.Collection("assets").Doc(docID)
	assetSettingsRef := params.fs.Collection("assetSettings").Doc(docID)

	if params.subscription.Properties.SubscriptionBillingStatus == "Active" {
		props := &AzureAssetProperties{
			CustomerDomain: params.domain,
			CustomerID:     params.customer.ID,
			Subscription:   params.subscription.Properties,
			Reseller:       params.reseller,
		}
		asset := AzureAsset{
			AssetType:  assetType,
			Customer:   params.customerRef,
			Properties: props,
		}

		if params.customerRef.ID != fb.Orphan.ID {
			if err := setAssetsSettings(params.ctx, assetSettingsRef, &asset, params.batch, params.customerRef, params.fs, assetType); err != nil {
				return err
			}
		} else {
			// Could not find customer, update settings to reference orphan customer and reset entity, contract
			params.batch.Set(assetSettingsRef, map[string]interface{}{
				"customer": params.customerRef,
				"entity":   nil,
				"contract": nil,
				"bucket":   nil,
				"type":     assetType,
			}, firestore.MergeAll)

			asset.Entity = nil
			asset.Contract = nil
			asset.Bucket = nil
			params.batchSize++
		}

		params.batch.Set(assetRef, asset)
		params.batch.Set(assetRef.Collection("assetMetadata").Doc("metadata"), map[string]interface{}{
			"lastUpdated": firestore.ServerTimestamp,
			"type":        asset.AssetType,
		})

		params.batchSize += 2
	} else {
		params.batch.Delete(assetRef)

		params.batchSize++
	}

	return nil
}

func setAssetsSettings(ctx *gin.Context, assetSettingsRef *firestore.DocumentRef, asset *AzureAsset, batch *firestore.WriteBatch, customerRef *firestore.DocumentRef, fs *firestore.Client, assetType string) error {
	docSnap, err := assetSettingsRef.Get(ctx)
	if err != nil {
		if status.Code(err) != codes.NotFound {
			return err
		}
	}

	if docSnap.Exists() {
		var as common.AssetSettings

		var contractRef *firestore.DocumentRef

		if err := docSnap.DataTo(&as); err != nil {
			return err
		}

		asset.Bucket = as.Bucket
		asset.Tags = as.Tags

		if entityRef, update := common.GetAssetEntity(ctx, fs, as.Customer, as.Entity, customerRef, nil); update {
			batch.Set(assetSettingsRef, map[string]interface{}{
				"customer": customerRef,
				"entity":   entityRef,
				"contract": contractRef,
			}, firestore.MergeAll)

			asset.Entity = entityRef
			asset.Contract = contractRef
		} else if as.Customer != nil && as.Entity != nil {
			batch.Set(assetSettingsRef, map[string]interface{}{
				"contract": contractRef,
			}, firestore.MergeAll)

			asset.Entity = as.Entity
			asset.Contract = contractRef
		}

		return nil
	}

	batch.Set(assetSettingsRef, map[string]interface{}{
		"customer": customerRef,
		"entity":   nil,
		"contract": nil,
		"bucket":   nil,
		"type":     assetType,
	}, firestore.MergeAll)

	return nil
}

func (a *AccessToken) listAllAzureCustomers(billingAccount AzureBillingAccount) ([]*AzureCustomer, error) {
	if err := a.Refresh(); err != nil {
		return nil, err
	}

	result, err := a.listAzureCustomersPage(billingAccount, "")
	if err != nil {
		return nil, err
	}

	customers := make([]*AzureCustomer, 0)
	customers = append(customers, result.Value...)
	nextLink := result.NextLink

	// Keep fetching customers until there are no more next links and append to the customers slice
	for nextLink != "" {
		result, err = a.listAzureCustomersPage(billingAccount, nextLink)
		if err != nil {
			return nil, err
		}

		customers = append(customers, result.Value...)
		nextLink = result.NextLink
	}

	return customers, nil
}

func (a *AccessToken) listAzureCustomersPage(billingAccount AzureBillingAccount, nextLinkURL string) (*AzureCustomersResult, error) {
	var url string

	if nextLinkURL == "" {
		url = fmt.Sprintf("%s/providers/Microsoft.Billing/billingAccounts/%s/customers", a.Resource, billingAccount)
	} else {
		url = nextLinkURL
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add api version to the query string if it's not a next link (those arleady include the api-version from previous request)
	if nextLinkURL == "" {
		q := req.URL.Query()
		q.Add("api-version", AzureAPIVersion)
		req.URL.RawQuery = q.Encode()
	}

	requestID, _ := uuid.NewRandom()
	correlationID, _ := uuid.NewRandom()

	req.Header.Set("Authorization", fmt.Sprintf("%s %s", a.TokenType, a.AccessToken))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("MS-RequestId", requestID.String())
	req.Header.Set("MS-CorrelationId", correlationID.String())

	client := http.DefaultClient

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	fmt.Println(string(respBody))

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to get azure customers")
	}

	var result AzureCustomersResult

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (a *AccessToken) listCustomerAzureSubscriptions(azureCustomer *AzureCustomer) ([]*AzureSubscription, error) {
	if err := a.Refresh(); err != nil {
		return nil, err
	}

	client := http.DefaultClient
	urlStr := a.Resource + azureCustomer.ID + "/billingSubscriptions"
	req, _ := http.NewRequest("GET", urlStr, nil)
	q := req.URL.Query()
	q.Add("api-version", AzureAPIVersion)
	req.URL.RawQuery = q.Encode()
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var result AzureSubscriptions
		if err := json.Unmarshal(respBody, &result); err != nil {
			return nil, err
		}

		return result.Value, nil
	}

	return nil, errors.New("failed to get azure customer subscriptions")
}

type CostManagementQueryError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type CostManagementQueryOutput struct {
	ID         string                        `json:"id"`
	Name       string                        `json:"name"`
	Type       string                        `json:"type"`
	Properties CostManagementQueryProperties `json:"properties"`

	CostManagementQueryError *CostManagementQueryError `json:"error,omitempty"`
}

type CostManagementQueryProperties struct {
	NextLink *string `json:"nextLink"`
	Columns  []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	} `json:"columns"`
	Rows [][]interface{} `json:"rows"`
}

type CostManagementQueryInput struct {
	Type       string                         `json:"type"`
	Timeframe  string                         `json:"timeframe"`
	TimePeriod *CostManagementQueryTimePeriod `json:"timePeriod,omitempty"`
	Dataset    *CostManagementQueryDataset    `json:"dataset"`
}

type CostManagementQueryTimePeriod struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type CostManagementQueryDataset struct {
	Granularity string                   `json:"granularity"`
	Aggregation map[string]interface{}   `json:"aggregation"`
	Grouping    []map[string]interface{} `json:"grouping"`
}

func CostManagementQuery(a *AccessToken, billingAccount AzureBillingAccount, input *CostManagementQueryInput) (*CostManagementQueryOutput, error) {
	if err := a.Refresh(); err != nil {
		return nil, err
	}

	data, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	var body *bytes.Buffer
	if len(data) > 0 {
		body = bytes.NewBuffer(data)
	}

	client := http.DefaultClient
	scope := fmt.Sprintf("providers/Microsoft.Billing/billingAccounts/%s", billingAccount)
	urlStr := fmt.Sprintf("%s/%s/providers/Microsoft.CostManagement/query", a.Resource, scope)
	req, _ := http.NewRequest("POST", urlStr, body)
	q := req.URL.Query()
	q.Add("api-version", AzureQueryAPIVersion)
	req.URL.RawQuery = q.Encode()
	requestID, _ := uuid.NewRandom()
	correlationID, _ := uuid.NewRandom()

	req.Header.Set("Authorization", fmt.Sprintf("%s %s", a.TokenType, a.AccessToken))
	req.Header.Set("Content-Type", "application/json")
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

	var result CostManagementQueryOutput

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusOK {
		return &result, nil
	}

	if result.CostManagementQueryError == nil {
		return nil, fmt.Errorf("status %d, body [%s]", resp.StatusCode, string(respBody))
	}

	return nil, fmt.Errorf("code: %s, message: %s", result.CostManagementQueryError.Code, result.CostManagementQueryError.Message)
}
