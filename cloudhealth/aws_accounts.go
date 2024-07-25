package cloudhealth

import (
	"encoding/json"
	"strconv"
	"time"
)

type AwsAccounts struct {
	AwsAccounts []*AwsAccount `json:"aws_accounts"`
}

type Authentication struct {
	Protocol             string `json:"protocol,omitempty"`
	AssumeRoleArn        string `json:"assume_role_arn,omitempty"`
	AssumeRoleExternalID string `json:"assume_role_external_id,omitempty"`
}

type AwsAccount struct {
	ID               int64          `json:"id,omitempty"`
	OwnerID          string         `json:"owner_id,omitempty"`
	Name             string         `json:"name,omitempty"`
	AmazonName       string         `json:"amazon_name,omitempty"`
	Authentication   Authentication `json:"authentication,omitempty"`
	HidePublicFields bool           `json:"hide_public_fields"`
	Region           string         `json:"region"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	AccountType      string         `json:"account_type"`
	VpcOnly          bool           `json:"vpc_only,omitempty"`
	ClusterName      string         `json:"cluster_name,omitempty"`
	Status           struct {
		Level      string    `json:"level"`
		LastUpdate time.Time `json:"last_update"`
	} `json:"status,omitempty"`
	Billing struct {
		IsConsolidated bool `json:"is_consolidated"`
	} `json:"billing,omitempty"`
	Cloudtrail struct {
		Enabled bool `json:"enabled"`
	} `json:"cloudtrail"`
	Ecs struct {
		Enabled bool `json:"enabled"`
	} `json:"ecs"`
	AwsConfig struct {
		Enabled bool `json:"enabled"`
	} `json:"aws_config"`
	Cloudwatch struct {
		Enabled bool `json:"enabled"`
	} `json:"cloudwatch"`
	CostAndUsageReport struct {
		Path string `json:"path"`
	} `json:"cost_and_usage_report,omitempty"`
	Tags   []interface{} `json:"tags"`
	Groups []struct {
		Name  string `json:"name"`
		Group string `json:"group"`
	} `json:"groups"`
	Links struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"_links"`
}

func ListAccounts(page int64, m map[int64]*AwsAccount, customer *Customer) error {
	path := "/v1/aws_accounts"

	params := make(map[string][]string)
	if customer != nil {
		params["client_api_id"] = []string{strconv.FormatInt(customer.ID, 10)}
	}

	params["per_page"] = []string{"100"}
	params["page"] = []string{strconv.FormatInt(page, 10)}

	body, err := Client.Get(path, params)
	if err != nil {
		return err
	}

	var response AwsAccounts
	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	if len(response.AwsAccounts) > 0 {
		for _, awsAccount := range response.AwsAccounts {
			m[awsAccount.ID] = awsAccount
		}

		return ListAccounts(page+1, m, customer)
	}

	return nil
}
