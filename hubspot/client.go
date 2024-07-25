package hubspot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/time/rate"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
)

type ClientConfig struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
	Token   string `json:"token"`
}

type Service struct {
	limiter   *rate.Limiter
	client    *http.Client
	config    *ClientConfig
	Companies *CompaniesService
	Contacts  *ContactsService
}

type CompaniesService struct {
	url     string
	service *Service
}

type ContactsService struct {
	url     string
	service *Service
}

// FilterOperator hubspot filter operators type
type FilterOperator string

const (
	FilterOperatorEquals        FilterOperator = "EQ"
	FilterOperatorContainsToken FilterOperator = "CONTAINS_TOKEN"
)

const (
	HubspotArraySeparator      string = ";"
	HubspotCustomRoleSeparator string = ", "
)

var sorts = []Sort{
	{
		PropertyName: "hs_lastmodifieddate",
		Direction:    "DESCENDING",
	},
}

func NewService(ctx context.Context) (*Service, error) {
	data, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretHubSpot)
	if err != nil {
		return nil, err
	}

	var conf ClientConfig
	if err = json.Unmarshal(data, &conf); err != nil {
		return nil, err
	}

	// hubspot rate limits are: 100req/10sec
	// however, it seems that anything higher than 2 requests/second
	// causes a "secondly rate limit" error...
	lim := rate.Every(1 * time.Second / 2)

	s := &Service{
		limiter: rate.NewLimiter(lim, 1),
		client:  http.DefaultClient,
		config:  &conf,
	}
	s.Companies = &CompaniesService{
		url:     "/crm/v3/objects/companies",
		service: s,
	}
	s.Contacts = &ContactsService{
		url:     "/crm/v3/objects/contacts",
		service: s,
	}

	return s, nil
}

type hsReq struct {
	Properties   []string  `json:"properties"`
	FilterGroups []Filters `json:"filterGroups"`
	Sorts        []Sort    `json:"sorts"`
}

type Filters struct {
	Filters []Filter `json:"filters"`
}

type Filter struct {
	PropertyName string         `json:"propertyName"`
	Operator     FilterOperator `json:"operator"`
	Value        string         `json:"value"`
}

type Sort struct {
	PropertyName string `json:"propertyName"`
	Direction    string `json:"direction"`
}

// Create a new company
func (s *CompaniesService) Create(ctx context.Context, updateReq *updateHsCompany) error {
	reqBody, err := json.Marshal(&updateReq)
	if err != nil {
		return err
	}

	if _, err = s.service.request(ctx, http.MethodPost, s.service.Companies.url, nil, reqBody); err != nil {
		return err
	}

	return nil
}

// Update an existing company
func (s *CompaniesService) Update(ctx context.Context, updateReq *updateHsCompany, id string) error {
	path := s.service.Companies.url + "/" + id

	reqBody, err := json.Marshal(&updateReq)
	if err != nil {
		return err
	}

	if _, err = s.service.request(ctx, http.MethodPatch, path, nil, reqBody); err != nil {
		return err
	}

	return nil
}

// Create a new contact
func (s *ContactsService) Create(ctx context.Context, updateReq updateHsContact) error {
	reqBody, err := json.Marshal(updateReq)
	if err != nil {
		return err
	}

	if _, err := s.service.request(ctx, http.MethodPost, s.service.Contacts.url, nil, reqBody); err != nil {
		return err
	}

	return nil
}

// Update an existing contact
func (s *ContactsService) Update(ctx context.Context, updateReq updateHsContact, id string) error {
	path := s.service.Contacts.url + "/" + id

	reqBody, err := json.Marshal(updateReq)
	if err != nil {
		return err
	}

	if _, err := s.service.request(ctx, http.MethodPatch, path, nil, reqBody); err != nil {
		return err
	}

	return nil
}

func (s *Service) request(ctx context.Context, method string, path string, params map[string][]string, data []byte) ([]byte, error) {
	url := s.config.BaseURL + path
	client := http.DefaultClient

	var body *bytes.Buffer
	if len(data) > 0 {
		body = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.config.Token))
	req.Header.Set("X-CMP-Project", common.ProjectID)
	req.Header.Set("X-CMP-Service", common.GAEService)
	req.Header.Set("X-CMP-Version", common.GAEVersion)

	q := req.URL.Query()

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

	if err := s.limiter.Wait(ctx); err != nil {
		return nil, err
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

	return nil, fmt.Errorf("%s %s (%d):\n%s", method, path, resp.StatusCode, string(respBody))
}

// Search a company or contact
func (s *Service) Search(ctx context.Context, body hsReq, entity string) ([]byte, error) {
	var path string
	if entity == "company" {
		path = s.Companies.url + "/search"
	} else {
		path = s.Contacts.url + "/search"
	}

	reqBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	res, err := s.request(ctx, http.MethodPost, path, nil, reqBody)
	if err != nil {
		return nil, err
	}

	return res, nil
}
