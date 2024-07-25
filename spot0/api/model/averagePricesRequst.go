package model

type AveragePricesRequest struct {
	AccountID     string   `json:"accountId,omitempty"`
	Region        string   `json:"region,omitempty"`
	ASGName       string   `json:"asgName,omitempty"`
	InstanceTypes []string `json:"instanceTypes,omitempty"`
	Subnets       []string `json:"subnets,omitempty"`
}
