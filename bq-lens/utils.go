package bqlens

import (
	"context"
	"fmt"

	"github.com/doitintl/hello/scheduled-tasks/common"
	httpClient "github.com/doitintl/http"
	"github.com/doitintl/idtoken"
)

const (
	OnboardURLDev  = "https://scheduled-tasks-dot-doitintl-cmp-dev.appspot.com/tasks/bq-lens/onboarding"
	OnboardURLProd = "https://scheduled-tasks-dot-me-doit-intl-com.appspot.com/tasks/bq-lens/onboarding"
)

func TriggerBQLensProcess(ctx context.Context, clientID string, removeData bool) error {
	functionURL := OnboardURLDev
	if common.Production {
		functionURL = OnboardURLProd
	}

	tokenSource, err := idtoken.New().GetTokenSource(ctx, functionURL)
	if err != nil {
		return err
	}

	client, err := httpClient.NewClient(ctx, &httpClient.Config{
		BaseURL:     functionURL,
		TokenSource: tokenSource,
	})
	if err != nil {
		return err
	}

	payload := struct {
		HandleSpecificSink string `json:"handleSpecificSink"`
		RemoveData         bool   `json:"removeData"`
	}{
		HandleSpecificSink: fmt.Sprintf("google-cloud-%s", clientID),
		RemoveData:         removeData,
	}

	// call cloud function
	if _, err := client.Post(ctx, &httpClient.Request{
		URL:     "",
		Payload: payload,
	}); err != nil {
		return err
	}

	return nil
}
