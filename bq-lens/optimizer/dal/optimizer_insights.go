package dal

import (
	"context"

	"github.com/doitintl/hello/scheduled-tasks/common"
	httpClient "github.com/doitintl/http"
	"github.com/doitintl/idtoken"
	insightsSDK "github.com/doitintl/insights/sdk"
)

const (
	bqLensProviderID = "bq-lens"

	apiURL = "api/insights/results"
)

type Insights struct {
	client *httpClient.Client
}

func NewInsights(ctx context.Context) (*Insights, error) {
	var targetURL string

	if common.Production {
		targetURL = insightsSDK.InternalAPIURLProd
	} else {
		targetURL = insightsSDK.InternalAPIURLDev
	}

	tokenSource, err := idtoken.New().GetTokenSource(ctx, targetURL)
	if err != nil {
		return nil, err
	}

	client, err := httpClient.NewClient(ctx, &httpClient.Config{
		BaseURL:     targetURL,
		TokenSource: tokenSource,
	})
	if err != nil {
		return nil, err
	}

	return &Insights{
		client: client,
	}, nil
}

func (d *Insights) PostInsightResults(ctx context.Context, results []insightsSDK.InsightResponse) error {
	payload := insightsSDK.PostResultsBody{
		ProviderID: bqLensProviderID,
		Results:    results,
	}

	if _, err := d.client.Post(ctx, &httpClient.Request{
		URL:     apiURL,
		Payload: payload,
	}); err != nil {
		return err
	}

	return nil
}
