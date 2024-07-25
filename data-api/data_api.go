package dataapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
)

const (
	dataAPIProduction  = "https://api4prod.doit.com/cmp-data-api"
	dataAPIDevelopment = "https://api4dev.doit.com/cmp-data-api"
)

var idToken string
var serviceURL string

type LogItem struct {
	Operation    string                 `json:"operation"`
	Context      string                 `json:"context"`
	UserEmail    string                 `json:"user_email,omitempty"`
	CustomerID   string                 `json:"customer_id,omitempty"`
	CustomerName string                 `json:"customer_name,omitempty"`
	SubContext   string                 `json:"sub_context,omitempty"`
	Category     string                 `json:"category,omitempty"`
	Action       string                 `json:"action,omitempty"`
	Status       string                 `json:"status,omitempty"`
	ProjectID    string                 `json:"project_id,omitempty"`
	JobID        string                 `json:"job_id,omitempty"`
	Severity     string                 `json:"severity,omitempty"`
	LogName      string                 `json:"log_name,omitempty"`
	TotalMs      int64                  `json:"total_ms,omitempty"`
	Cost         float64                `json:"cost,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

func init() {
	// Specify service URL
	serviceURL = dataAPIDevelopment
	if common.Env == "production" {
		serviceURL = dataAPIProduction
	}

	// Create idToken if does not exist
	if idToken == "" {
		token, err := getIDToken()
		if err != nil {
			fmt.Println(err)
		}

		idToken = token
	}
}

func SendLogToCloudLogging(logObj *LogItem) error {
	// Get request body
	bodyB, err := buildRequestBody(logObj)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// Send request
	err = makePostRequest(bodyB)
	if err != nil {
		// Try to get a new token and send the request again
		idToken, err = getIDToken()
		if err != nil {
			fmt.Println(err)
			return err
		}

		err = makePostRequest(bodyB)
	}

	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func buildRequestBody(logObj *LogItem) ([]byte, error) {
	if logObj.Operation == "" || logObj.Context == "" {
		return nil, fmt.Errorf(`CMPDataApi.logEvent ERROR: at least one of the following params is not specified: operation, context`)
	}

	body, err := json.Marshal(logObj)

	return body, err
}

func getIDToken() (string, error) {
	ctx := context.Background()

	secret, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretAppEngine)
	if err != nil {
		return "", err
	}

	token, err := common.GetServiceAccountIDToken(ctx, serviceURL, secret)
	if err != nil {
		return "", err
	}

	return token.AccessToken, nil
}

func makePostRequest(body []byte) error {
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", serviceURL, "log"), bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", idToken))
	req.Header.Add("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)

	statusCode := res.StatusCode
	if statusCode != 200 {
		return fmt.Errorf("Error: status code %v", statusCode)
	}

	return nil
}
