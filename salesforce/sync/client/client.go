package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"github.com/doitintl/retry"
)

type Company struct {
	Name                             string                        `json:"Name__c"`
	AmEmail                          string                        `json:"AM_Email__c"`
	AeEmail                          string                        `json:"AE_Email__c"`
	AdditionalDomains                string                        `json:"Additional_Domains__c"`
	Advantage                        string                        `json:"Advantage__c"`
	AWSRevenue                       float64                       `json:"AWS_Revenue__c"`
	AzureRevenue                     float64                       `json:"Azure_Revenue__c"`
	Classification                   common.CustomerClassification `json:"Classification__c"`
	ConsoleLink                      string                        `json:"Console_Link__c"`
	FlexsaveSavings                  float64                       `json:"Flexsave_Savings__c"`
	GCPRevenue                       float64                       `json:"GCP_Revenue__c"`
	GSuiteRevenue                    float64                       `json:"GSuite_Revenue__c"`
	IsSolveFreeTrial                 string                        `json:"Is_Solve_Free_Trial__c"`
	NavigatorFreeTrialExpirationDate *string                       `json:"Navigator_Free_Trial_Expiration_Date__c"`
	NavigatorFreeTrialStartDate      *string                       `json:"Navigator_Free_Trial_Start_Date__c"`
	NavigatorFreeTrial               string                        `json:"Navigator_Free_Trial__c"`
	NavigatorRevenue                 float64                       `json:"Navigator_Revenue__c"`
	NavigatorTier                    string                        `json:"Navigator_Tier__c"`
	O365Revenue                      float64                       `json:"O365_Revenue__c"`
	PrimaryDomain                    string                        `json:"Primary_domain__c"`
	SolveFreeTrialExpirationDate     *string                       `json:"Solve_Free_Trial_Expiration_Date__c"`
	SolveFreeTrialStartDate          *string                       `json:"Solve_Free_Trial_Start_Date__c"`
	SolveRevenue                     float64                       `json:"Solve_Revenue__c"`
	SolveTier                        string                        `json:"Solve_Tier__c"`
}

type User struct {
	ConsoleCompanyID   string `json:"Console_Company_Id__c"`
	ConsoleFirstLogin  string `json:"console_First_Login__c"`
	ConsoleLastLogin   string `json:"Console_Last_Login__c"`
	ConsolePermissions string `json:"Console_Permissions__c"`
	ConsoleRoles       string `json:"Console_Roles__c"`
	Email              string `json:"Email__c"`
	FirstName          string `json:"First_Name__c"`
	LastName           string `json:"Last_Name__c"`
	Title              string `json:"Title__c"`
}

type Contract struct {
	ConsoleCompanyID   string  `json:"Console_Company_Id__c"`
	Name               string  `json:"Name__c"`
	Assets             string  `json:"Assets__c"`
	ChargePerTerm      float64 `json:"Charge_per_Term__c"`
	CommitmentMonths   float64 `json:"Commitment_Months__c"`
	CommitmentPeriod   string  `json:"Commitment_Periods__c"`
	CommitmentRollover bool    `json:"Commitment_Rollover__c"`
	Customer           string  `json:"Customer__c"`
	Description        string  `json:"Description__c"`
	Discount           float64 `json:"Discount__c"`
	DiscountEndDate    string  `json:"Discount_End_Date__c"`
	DisplayName        string  `json:"Display_Name__c"`
	EndDate            string  `json:"End_Date__c"`
	Entity             string  `json:"Entity__c"`
	EstimatedValue     float64 `json:"Estimated_Value__c"`
	IsCommitment       bool    `json:"Is_Commitment__c"`
	IsRenewal          bool    `json:"Is_Renewal__c"`
	Notes              string  `json:"Notes__c"`
	PackageType        string  `json:"Package_Type__c"`
	PaymentTerm        string  `json:"Payment_Term__c"`
	Price              float64 `json:"Price__c"`
	ProductType        string  `json:"Product_Type__c"`
	Properties         string  `json:"Properties__c"`
	PurchaseOrder      string  `json:"Purchase_Order__c"`
	SKU                string  `json:"SKU__c"`
	StartDate          string  `json:"Start_Date__c"`
	Tier               string  `json:"Tier__c"`
	Timestamp          string  `json:"Timestamp__c"`
	Type               string  `json:"Type__c"`
	ContractType       string  `json:"Contract_Type__c"`
}

type Client interface {
	UpsertCompany(ctx context.Context, id string, company Company) error
	UpsertUser(ctx context.Context, id string, user User) error
	UpsertContract(ctx context.Context, id string, contract Contract) error
}

type client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient() (Client, error) {
	httpClient, baseURL, err := newHTTPClient(context.Background())
	if err != nil {
		return nil, err
	}

	return &client{
		httpClient: httpClient,
		baseURL:    baseURL,
	}, nil
}

func (c *client) UpsertCompany(ctx context.Context, id string, company Company) error {
	payload, err := json.Marshal(company)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", c.baseURL+"/services/data/v60.0/sobjects/CAccount__c/console_id__c/"+id, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return errors.New("failed to create company. Status: " + resp.Status + ". Response body: " + string(body))
	}

	return nil
}

func (c *client) UpsertUser(ctx context.Context, userID string, user User) error {
	payload, err := json.Marshal(user)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", c.baseURL+"/services/data/v60.0/sobjects/Ccontact__c/Console_Id__c/"+userID, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return errors.New("failed to create user. Status: " + resp.Status + ". Response body: " + string(body))
	}

	return nil
}

func (c *client) UpsertContract(ctx context.Context, contractID string, contract Contract) error {
	payload, err := json.Marshal(contract)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", c.baseURL+"/services/data/v60.0/sobjects/CContract__c/Console_Id__c/"+contractID, bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return errors.New("failed to create contract. Status: " + resp.Status + ". Response body: " + string(body))
	}

	return nil
}

func getBearerToken(_ context.Context, clientID, clientSecret, instanceURL string) (string, error) {
	payload := []byte("grant_type=client_credentials&client_id=" + clientID + "&client_secret=" + clientSecret)

	var body []byte

	if err := retry.Do(func() error {
		req, err := http.NewRequest("POST", instanceURL+"/services/oauth2/token", bytes.NewBuffer(payload))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		client := &http.Client{}

		resp, err := client.Do(req)

		if err != nil {
			return err
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to get bearer token, status code: %d, body: %s", resp.StatusCode, string(body))
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return nil

	}, 3, 10*time.Second, "exponential"); err != nil {
		return "", err
	}

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
	}

	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return "", err
	}

	return tokenResponse.AccessToken, nil
}

type SecretPayload struct {
	ClientID     string `json:"clientId" validate:"required"`
	ClientSecret string `json:"clientSecret" validate:"required"`
	InstanceURL  string `json:"instanceURL" validate:"required"`
}

type token struct {
	token       *string
	lastUpdated time.Time
}

func (t *token) getToken() *string {
	if t.token != nil && time.Since(t.lastUpdated) <= 5*time.Minute {
		return t.token
	}

	return nil
}

func (t *token) setToken(token string) {
	t.token = &token
	t.lastUpdated = time.Now()
}

var cachedToken = token{
	token:       nil,
	lastUpdated: time.Time{},
}

func newHTTPClient(ctx context.Context) (*http.Client, string, error) {
	secret, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretSalesforceSync)
	if err != nil {
		return nil, "", err
	}

	var config SecretPayload

	if err := json.Unmarshal(secret, &config); err != nil {
		return nil, "", err
	}

	var bearerToken string

	existingToken := cachedToken.getToken()

	if existingToken == nil {
		bearerToken, err = getBearerToken(ctx, config.ClientID, config.ClientSecret, config.InstanceURL)
		if err != nil {
			return nil, "", err
		}

		cachedToken.setToken(bearerToken)
	} else {
		bearerToken = *existingToken
	}

	client := &http.Client{}

	client.Transport = &BearerAuthTransport{
		Token: bearerToken,
	}

	return client, config.InstanceURL, nil
}

type BearerAuthTransport struct {
	Token string
}

func (t *BearerAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.Token)
	return http.DefaultTransport.RoundTrip(req)
}
