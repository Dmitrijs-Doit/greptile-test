package microsoft

import (
	"time"

	"github.com/doitintl/hello/scheduled-tasks/assets"
	baseAssets "github.com/doitintl/hello/scheduled-tasks/assets/pkg"
)

type CSPDomain string

type Customers struct {
	TotalCount int64       `json:"totalCount"`
	Items      []*Customer `json:"items"`
}

type Customer struct {
	ID             string `json:"id"`
	CompanyProfile struct {
		TenantID string `json:"tenantId"`
		Domain   string `json:"domain"`
		Name     string `json:"companyName"`
	} `json:"companyProfile"`
}

type Subscriptions struct {
	TotalCount int64           `json:"totalCount"`
	Items      []*Subscription `json:"items"`
}

type Availabilities struct {
	TotalCount int64           `json:"totalCount"`
	Items      []*Availability `json:"items"`
	//do we need to check if billingCycle include monthly billing?
}

type Availability struct {
	ID            string `json:"id"`
	ProductID     string `json:"productId"`
	SkuID         string `json:"skuId"`
	CatalogItemID string `json:"catalogItemId"`
	Sku           SKU    `json:"sku"`
	//do we need to check if billingCycle include monthly billing?
}

type Cart struct {
	ID                    string    `json:"id"`
	CreationTimestamp     time.Time `json:"creationTimestamp"`
	LastModifiedTimestamp time.Time `json:"lastModifiedTimestamp"`
	ExpirationTimestamp   time.Time `json:"expirationTimestamp"`
	LastModifiedUser      string    `json:"lastModifiedUser"`
	Status                string    `json:"status"`
	LineItems             []struct {
		ID            int    `json:"id"`
		CatalogItemID string `json:"catalogItemId"`
		Quantity      int64  `json:"quantity"`
		CurrencyCode  string `json:"currencyCode"`
		BillingCycle  string `json:"billingCycle"`
		TermDuration  string `json:"termDuration"`
	}
}

type CheckedOutCart struct {
	Orders []Order
}

type Order struct {
	ID                  string    `json:"id"`
	AlternateID         string    `json:"alternateId"`
	ReferenceCustomerID string    `json:"referenceCustomerId"`
	BillingCycle        string    `json:"billingCycle"`
	CurrencyCode        string    `json:"currencyCode"`
	CreationDate        time.Time `json:"creationDate"`
	LineItems           []OrderLineItem
}

type OrderLineItem struct {
	LineItemNumber  int    `json:"lineItemNumber"`
	OfferID         string `json:"offerId"`
	SubscriptionID  string `json:"subscriptionId"`
	TermDuration    string `json:"termDuration"`
	TransactionType string `json:"transactionType"`
	FriendlyName    string `json:"friendlyName"`
	Quantity        int64  `json:"quantity"`
}

type Subscription struct {
	ID                  string       `json:"id" firestore:"id"`
	OfferID             string       `json:"offerId" firestore:"offerId"`
	OrderID             string       `json:"orderId" firestore:"orderId"`
	FriendlyName        string       `json:"friendlyName" firestore:"friendlyName"`
	OfferName           string       `json:"offerName" firestore:"offerName"`
	Quantity            int64        `json:"quantity" firestore:"quantity"`
	CreationDate        string       `json:"creationDate" firestore:"creationDate"`
	StartDate           string       `json:"effectiveStartDate" firestore:"effectiveStartDate"`
	EndDate             string       `json:"commitmentEndDate" firestore:"commitmentEndDate"`
	Status              string       `json:"status" firestore:"status"`
	AutoRenew           bool         `json:"autoRenewEnabled" firestore:"autoRenewEnabled"`
	BillingType         string       `json:"billingType" firestore:"billingType"`
	BillingCycle        string       `json:"billingCycle" firestore:"billingCycle"`
	TermDuration        string       `json:"termDuration" firestore:"termDuration"`
	RenewalTermDuration string       `json:"renewalTermDuration" firestore:"renewalTermDuration"`
	IsTrial             bool         `json:"isTrial" firestore:"isTrial"`
	ProductType         *ProductType `json:"productType" firestore:"productType"`
}

type SubscriptionWithStatus struct {
	Subscription *Subscription
	Syncing      bool
}

type SubscriptionSyncRequest struct {
	Quantity          int64     `json:"quantity"`
	Reseller          CSPDomain `json:"reseller"`
	LicenseCustomerID string    `json:"customer"`
	SubscriptionID    string    `json:"subscription"`
}

// ProductType exists for New Commerce Experience (NCE) subscriptions and Azure,
// but does not exist for old legacy subscription
type ProductType struct {
	ID          string `json:"id" firestore:"id"`
	DisplayName string `json:"displayName" firestore:"displayName"`
}

type Asset struct {
	baseAssets.BaseAsset
	Properties *AssetProperties `firestore:"properties"`
}

type AssetProperties struct {
	Subscription   *Subscription    `json:"subscription" firestore:"subscription"`
	CustomerDomain string           `json:"customerDomain" firestore:"customerDomain"`
	CustomerID     string           `json:"customerID" firestore:"customerId"`
	Reseller       CSPDomain        `json:"-" firestore:"reseller"`
	Syncing        bool             `json:"syncing" firestore:"syncing"`
	Settings       *assets.Settings `json:"-" firestore:"settings"`
}

type Skus struct {
	TotalCount int   `json:"totalCount"`
	Items      []SKU `json:"items"`
}

type SKU struct {
	ID                     string   `json:"id"`
	ProductID              string   `json:"productId"`
	Title                  string   `json:"title"`
	Description            string   `json:"description"`
	MinimumQuantity        int      `json:"minimumQuantity"`
	MaximumQuantity        int      `json:"maximumQuantity"`
	IsTrial                bool     `json:"isTrial"`
	SupportedBillingCycles []string `json:"supportedBillingCycles"`
	PurchasePrerequisites  []string `json:"purchasePrerequisites"`
	DynamicAttributes      struct {
		IsAddon          bool     `json:"isAddon"`
		PrerequisiteSkus []string `json:"prerequisiteSkus"`
		ProvisioningID   string   `json:"provisioningId"`
	} `json:"dynamicAttributes"`
}

type CreateAssetProps struct {
	CustomerID            string
	EntityID              string
	LicenseCustomerDomain string
	LicenseCustomerID     string
	Reseller              CSPDomain
}
