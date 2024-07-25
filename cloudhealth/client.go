package cloudhealth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
)

type client struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

type Report struct {
	Dimensions []map[string][]Dimension `json:"dimensions"`
	Data       [][][]float64            `json:"data"`
	Interval   string                   `json:"interval"`
	Status     string                   `json:"status"`
	Measures   []Measure                `json:"measures"`
}

type SingleDimensionReport struct {
	Dimensions []map[string][]Dimension `json:"dimensions"`
	Data       [][]float64              `json:"data"`
	Interval   string                   `json:"interval"`
	Status     string                   `json:"status"`
	Measures   []Measure                `json:"measures"`
}

type ErrorPayload struct {
	Error string `json:"error"`
}

type ErrorReport struct {
	ErrorMessage ErrorPayload `json:"data"`
}

type CostHistoryReport struct {
	Dimensions []ReportDimensions `json:"dimensions"`
	Data       [][][]float64      `json:"data"`
	Interval   string             `json:"interval"`
	Status     string             `json:"status"`
}

type CostHistoryReport3Dim struct {
	Dimensions []ReportDimensions `json:"dimensions"`
	Data       [][][][]float64    `json:"data"`
	Interval   string             `json:"interval"`
	Status     string             `json:"status"`
}

type CostHistoryReport4Dim struct {
	Dimensions []ReportDimensions `json:"dimensions"`
	Data       [][][][][]float64  `json:"data"`
	Interval   string             `json:"interval"`
	Status     string             `json:"status"`
}

// On Demand Cost Data
type OnDemandData struct {
	Data     [][]float64 `json:"data"`
	Interval string      `json:"interval"`
	Status   string      `json:"status"`
	Forecast float64     `json:"forecast"`
}
type DailyOnDemandData struct {
	Dimensions []ReportDimensions `json:"dimensions"`
	Data       [][]float64        `json:"data"`
	Interval   string             `json:"interval"`
	Status     string             `json:"status"`
}

type DailyOnDemand struct {
	Dimensions []ReportDimensions `json:"dimensions"`
	Data       []float64          `json:"data"`
}

type ReportDimensions struct {
	Time                  []Dimension `json:"time,omitempty"`
	AWSAccount            []Dimension `json:"AWS-Account,omitempty"`
	AWSBillingAccount     []Dimension `json:"AWS-Billing-Account,omitempty"`
	AWSServiceCategory    []Dimension `json:"AWS-Service-Category,omitempty"`
	AWSRegions            []Dimension `json:"AWS-Regions,omitempty"`
	EC2InstanceTypes      []Dimension `json:"EC2-Instance-Types,omitempty"`
	EC2OperatingSystems   []Dimension `json:"EC2-Operating-Systems,omitempty"`
	EC2InstanceTypeFamily []Dimension `json:"EC2-Instance-Type-Family,omitempty"`
}

type Dimension struct {
	Name      string `json:"name"`
	Label     string `json:"label"`
	Populated bool   `json:"populated"`
	Excluded  bool   `json:"excluded"`
	Direct    bool   `json:"direct"`
	Parent    int    `json:"parent"`
}

type Measure struct {
	Name  string   `json:"name"`
	Label string   `json:"label"`
	Meta  Metadata `json:"metadata"`
}

type Metadata struct {
	Units string `json:"units"`
	Scale string `json:"scale"`
}

const (
	PartnerExternalID = "6cda262029ad7b34a64ff537196ab4"

	ProtocolAssumeRole = "assume_role"

	ClassificationManagedWithAccess = "managed_with_access"

	ErrEmptyReportMessage = "This filter combination has no members to find at this time interval (i.e. empty set). Please try a different filter set or time interval."
)

var Client = &client{}

func init() {
	ctx := context.Background()

	data, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretCloudHealth)
	if err != nil {
		log.Fatalln(err)
	}

	if err = json.Unmarshal(data, Client); err != nil {
		log.Fatalln(err)
	}
}

func (c *client) Get(path string, params map[string][]string) ([]byte, error) {
	url := c.BaseURL + path
	client := http.DefaultClient

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("api_key", c.APIKey)

	if params != nil {
		for key, values := range params {
			for _, value := range values {
				if value != "" {
					q.Add(key, value)
				}
			}
		}
	}

	req.URL.RawQuery = q.Encode()
	req.Header.Set("Accept", "application/json")

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

	return nil, fmt.Errorf("%s: %s", resp.Status, string(respBody))
}

func (c *client) Post(path string, params map[string][]string, data []byte) ([]byte, error) {
	url := c.BaseURL + path
	client := http.DefaultClient

	var body *bytes.Buffer
	if len(data) > 0 {
		body = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("api_key", c.APIKey)

	if params != nil {
		for key, values := range params {
			for _, value := range values {
				if value != "" {
					q.Add(key, value)
				}
			}
		}
	}

	req.URL.RawQuery = q.Encode()
	req.Header.Set("Content-Type", "application/json")

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

	return nil, fmt.Errorf("%s: %s", resp.Status, string(respBody))
}

func (c *client) Put(path string, params map[string][]string, data []byte) ([]byte, error) {
	url := c.BaseURL + path
	client := http.DefaultClient

	var body *bytes.Buffer
	if len(data) > 0 {
		body = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest("PUT", url, body)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("api_key", c.APIKey)

	if params != nil {
		for key, values := range params {
			for _, value := range values {
				if value != "" {
					q.Add(key, value)
				}
			}
		}
	}

	req.URL.RawQuery = q.Encode()
	req.Header.Set("Content-Type", "application/json")

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

	return nil, fmt.Errorf("%s: %s", resp.Status, string(respBody))
}

func (c *client) Delete(path string, params map[string][]string) ([]byte, error) {
	url := c.BaseURL + path
	client := http.DefaultClient

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("api_key", c.APIKey)

	if params != nil {
		for key, values := range params {
			for _, value := range values {
				if value != "" {
					q.Add(key, value)
				}
			}
		}
	}

	req.URL.RawQuery = q.Encode()

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

	return nil, fmt.Errorf("%s: %s", resp.Status, string(respBody))
}

func (c *client) GetSingleDimensionReport(path string, params map[string][]string) (SingleDimensionReport, error) {
	body, err := c.Get(path, params)
	if err != nil {
		return SingleDimensionReport{}, err
	}

	var report SingleDimensionReport
	if err := json.Unmarshal(body, &report); err != nil {
		if reportError := AttemptUnmarshalErrorFromReport(body); reportError != nil {
			return SingleDimensionReport{}, reportError
		}

		return SingleDimensionReport{}, err
	}

	return report, nil
}

func (c *client) GetCostHistoryReport4Dim(path string, params map[string][]string) (*CostHistoryReport4Dim, error) {
	body, err := c.Get(path, params)
	if err != nil {
		return nil, err
	}

	var report CostHistoryReport4Dim
	if err := json.Unmarshal(body, &report); err != nil {
		if reportError := AttemptUnmarshalErrorFromReport(body); reportError != nil {
			return nil, reportError
		}

		return nil, err
	}

	return &report, nil
}

// AttemptUnmarshalErrorFromReport When you request a valid report with for which they have data, you will receive a field with `"data": [ [ ....`
// When you request a valid report, for which there is no data (i.e. no instances running in that time period)
// you instead get a field with `"data": { "error": "This filter combination has no members......`
func AttemptUnmarshalErrorFromReport(body []byte) error {
	var errorReport ErrorReport
	if err := json.Unmarshal(body, &errorReport); err != nil {
		return nil
	}

	return errors.New(errorReport.ErrorMessage.Error)
}
