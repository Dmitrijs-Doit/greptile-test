package pkg

type AzureAsset struct {
	BaseAsset
	Properties *AzureProperties `firestore:"properties"`
}

type AzureProperties struct {
	CustomerDomain string             `firestore:"customerDomain"`
	CustomerID     string             `firestore:"customerId"`
	Reseller       string             `firestore:"reseller"`
	Subscription   *AzureSubscription `firestore:"subscription"`
}

type AzureSubscription struct {
	BillingProfileDisplayName string `firestore:"billingProfileDisplayName"`
	BillingProfileID          string `firestore:"billingProfileId"`
	CustomerDisplayName       string `firestore:"customerDisplayName"`
	CustomerID                string `firestore:"customerId"`
	DisplayName               string `firestore:"displayName"`
	SKUDescription            string `firestore:"skuDescription"`
	SKUID                     string `firestore:"skuId"`
	SubscriptionBillingStatus string `firestore:"subscriptionBillingStatus"`
	SubscriptionID            string `firestore:"subscriptionId"`
}
