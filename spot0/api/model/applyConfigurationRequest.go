package model

type ApplyConfigurationRequest struct {
	AccountID        string         `json:"accountId,omitempty"`
	Region           string         `json:"region,omitempty"`
	ASGName          string         `json:"asgName,omitempty"`
	Configuration    *Configuration `json:"configuration,omitempty"`
	CustomerID       string         `json:"-"`
	ForceManagedMode bool           `json:"-"`
	Scope            string         `json:"scope,omitempty"`
}
