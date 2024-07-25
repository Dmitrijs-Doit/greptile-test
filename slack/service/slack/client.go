package slack

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/doitintl/hello/scheduled-tasks/errorreporting"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
)

type slack struct {
	BaseURL string `json:"base_url"`
	Token   string `json:"token"`
}

const baseURL = "https://slack.com/api"

// Client - for Doitsy Slack bot (A012TR3MK5E)
var Client *slack

func init() {
	ctx := context.Background()

	data, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretSlackBot)
	if err != nil {
		log.Fatalln(err)
	}

	token := string(data)
	Client = &slack{
		Token:   token,
		BaseURL: baseURL,
	}
}

func (s *slack) Get(name string, params map[string][]string) ([]byte, error) {
	url := fmt.Sprintf("%s/%s?token=%s", s.BaseURL, name, s.Token)
	client := http.DefaultClient

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

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

	return respBody, nil
}

func (s *slack) Post(ctx *gin.Context, name string, params map[string][]string, body []byte) ([]byte, error) {
	url := fmt.Sprintf("%s/%s?token=%s", s.BaseURL, name, s.Token)
	client := http.DefaultClient

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return nil, err
	}

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

	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		errorreporting.AbortWithErrorReport(ctx, http.StatusInternalServerError, err)
		return nil, err
	}

	return respBody, nil
}
