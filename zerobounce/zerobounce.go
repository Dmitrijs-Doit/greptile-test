package zerobounce

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

type ClientConfig struct {
	BaseURL string `json:"base_url"`
	APIKey  string `json:"api_key"`
}

type Service struct {
	client *http.Client
	config *ClientConfig
}

// Email validate statuses
const (
	StatusValid     = "valid"
	StatusInvalid   = "invalid"
	StatusCatchAll  = "catch-all"
	StatusDoNotMail = "do_not_mail"
	StatusUnknown   = "unknown"
)

// Email validate sub-statuses
// https://www.zerobounce.net/docs/email-validation-api-quickstart/v1-status-codes
const (
	SubStatusGreyListed        = "greylisted"
	SubStatusMailboxNotFound   = "mailbox_not_found"
	SubStatusRoleBased         = "role_based"
	SubStatusDisposable        = "disposable"
	SubStatusRoleBasedCatchAll = "role_based_catch_all"
	SubStatusPossibleTrap      = "possible_trap"
)

var allowedDoNotMailSubStatuses = []string{
	SubStatusRoleBased,
	SubStatusRoleBasedCatchAll,
	SubStatusPossibleTrap,
	SubStatusDisposable,
}

var conf ClientConfig

func init() {
	clientConfigB, err := secretmanager.AccessSecretLatestVersion(context.Background(), secretmanager.SecretZerobounce)
	if err != nil {
		log.Fatalln(err)
	}

	if err := json.Unmarshal(clientConfigB, &conf); err != nil {
		log.Fatalln(err)
	}
}

func New() *Service {
	return &Service{client: http.DefaultClient, config: &conf}
}

type ValidateResult struct {
	Address      string `json:"address"`
	Status       string `json:"status"`
	SubStatus    string `json:"sub_status"`
	SMTPProvider string `json:"smtp_provider"`
	FreeEmail    bool   `json:"free_email"`
}

type ValidateParams struct {
	Email     string `form:"email" json:"email"`
	IPAddress string `form:"ip_address" json:"ip_address"`
}

func (s *Service) Validate(params *ValidateParams) (*ValidateResult, error) {
	url := fmt.Sprintf("%s/validate", s.config.BaseURL)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/json")

	q := req.URL.Query()
	q.Add("api_key", s.config.APIKey)
	q.Add("email", params.Email)
	q.Add("ip_address", params.IPAddress)
	req.URL.RawQuery = q.Encode()

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == 200 {
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		var res ValidateResult
		if err := json.Unmarshal(respBody, &res); err != nil {
			return nil, err
		}

		return &res, nil
	}

	return nil, fmt.Errorf("error: %s", resp.Status)
}

func (r *ValidateResult) IsValidEmail(withFreeEmail bool) (bool, bool) {
	if !withFreeEmail && r.FreeEmail {
		return false, false
	}

	switch r.Status {
	case StatusValid, StatusCatchAll:
		return true, false
	case StatusDoNotMail:
		return slice.Contains(allowedDoNotMailSubStatuses, r.SubStatus), false
	case StatusUnknown:
		return r.SubStatus == SubStatusGreyListed, false
	case StatusInvalid:
		return false, r.SubStatus == SubStatusMailboxNotFound
	default:
		return false, false
	}
}
