package priority

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
)

type priority struct {
	URL            string `json:"url"`
	BaseURL        string `json:"api_url"`
	Username       string `json:"username"`
	Password       string `json:"password"`
	Tabula         string `json:"tabulaini"`
	StorageSecret  string `json:"storage_hmac_secret"`
	StorageBaseURL string `json:"storage_base_url"`
}

type HTTPError struct {
	Status  int    `json:"status"`
	Message string `json:"error"`
}

// Deprecated: Use newer Priority service
var Client = &priority{}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP Error: [%d] %s", e.Status, e.Message)
}

func init() {
	ctx := context.Background()

	data, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretPriority)
	if err != nil {
		log.Fatalln(err)
	}

	if err := json.Unmarshal(data, Client); err != nil {
		log.Fatalln(err)
	}
}

func String(v string) *string {
	return &v
}

func (c *priority) Get(company, form string, params map[string][]string) ([]byte, error) {
	url := fmt.Sprintf("%s/%s/%s", c.BaseURL, company, form)
	client := http.DefaultClient
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(c.Username, c.Password)

	if params != nil {
		q := req.URL.Query()

		for key, values := range params {
			for _, value := range values {
				if value != "" {
					q.Add(key, value)
				}
			}
		}

		req.URL.RawQuery = q.Encode()
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return respBody, nil
	}

	httpErr := &HTTPError{resp.StatusCode, string(respBody)}

	return nil, httpErr
}

func (c *priority) Post(company, form string, params map[string][]string, data []byte) ([]byte, error) {
	url := fmt.Sprintf("%s/%s/%s", c.BaseURL, company, form)
	client := http.DefaultClient

	var body *bytes.Buffer
	if len(data) > 0 {
		body = bytes.NewBuffer(data)
	}

	req, _ := http.NewRequest("POST", url, body)
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.Username, c.Password)

	if params != nil {
		q := req.URL.Query()

		for key, values := range params {
			for _, value := range values {
				if value != "" {
					q.Add(key, value)
				}
			}
		}

		req.URL.RawQuery = q.Encode()
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return respBody, nil
	}

	httpErr := &HTTPError{resp.StatusCode, string(respBody)}

	return nil, httpErr
}

func (c *priority) Patch(company, form string, params map[string][]string, data []byte) ([]byte, error) {
	url := fmt.Sprintf("%s/%s/%s", c.BaseURL, company, form)
	client := http.DefaultClient

	var body *bytes.Buffer
	if len(data) > 0 {
		body = bytes.NewBuffer(data)
	}

	req, _ := http.NewRequest("PATCH", url, body)
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(c.Username, c.Password)

	if params != nil {
		q := req.URL.Query()

		for key, values := range params {
			for _, value := range values {
				if value != "" {
					q.Add(key, value)
				}
			}
		}

		req.URL.RawQuery = q.Encode()
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return respBody, nil
	}

	httpErr := &HTTPError{resp.StatusCode, string(respBody)}

	return nil, httpErr
}
