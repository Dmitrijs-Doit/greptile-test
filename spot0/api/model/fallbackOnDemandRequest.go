package model

type FallbackOnDemandRequest struct {
	Action          string `json:"action,omitempty"`
	CustomerID      string `json:"customerID"`
	AccountID       string `json:"accountId,omitempty"`
	Region          string `json:"region,omitempty"`
	RoleToAssumeArn string `json:"roleToAssumeArn"`
	ExternalID      string `json:"externalID"`
	AsgName         string `json:"asgName,omitempty"`
}
