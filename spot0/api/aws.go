package api

import "github.com/doitintl/hello/scheduled-tasks/spot0/api/model"

// GetAccountsEvent - The main event to trigger the flow
type GetAccountsEvent struct {
	Scope            string // all, onePage, account
	PageSize         int
	NextToken        *string // hold the last processed account
	AccountID        string
	ExecID           string
	CustomerID       string
	Region           string
	AsgName          string
	ForceManagedMode bool
	Configuration    *model.Configuration
}

// AwsCred ...
type AwsCred struct {
	AwsAccessKey       string `json:"aws_access_key_id"`
	AwsSecretAccessKey string `json:"aws_secret_access_key"`
}
