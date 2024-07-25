package model

type UpdateAsgConfigRequest struct {
	CustomerID    string         `json:"customerID,omitempty"`
	AccountID     string         `json:"accountId,omitempty"`
	Region        string         `json:"region,omitempty"`
	AsgName       string         `json:"asgName,omitempty"`
	Configuration *Configuration `json:"configuration,omitempty"`
}
