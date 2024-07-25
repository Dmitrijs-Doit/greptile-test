package dal

import "time"

type BillingDataConfig struct {
	CustomerID     string    `json:"customerId" firestore:"customerId"`
	Container      string    `json:"container" firestore:"container"`
	Account        string    `json:"account" firestore:"account"`
	Directory      string    `json:"directory" firestore:"directory"`
	ResourceGroup  string    `json:"resourceGroup" firestore:"resourceGroup"`
	SubscriptionID string    `json:"subscriptionId" firestore:"subscriptionId"`
	CreatedAt      time.Time `json:"createdAt" firestore:"createdAt"`
}
