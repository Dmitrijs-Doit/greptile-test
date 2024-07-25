package domain

type AggregatedQueryData struct {
	Cloud                        string
	FullBillingDataTableFullName string
	BillingAccountID             string
	AdditionalFields             []string
	AggregationInterval          string
	IsCSP                        bool
}
