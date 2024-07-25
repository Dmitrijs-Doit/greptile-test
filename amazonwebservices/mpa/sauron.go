package mpa

import (
	"context"
	"fmt"

	"github.com/doitintl/http"
)

func (s *MasterPayerAccountService) LinkMpaToSauron(ctx context.Context, data *LinkMpaToSauronData) error {
	headers := map[string]string{
		http.ContentType:         http.ApplicationJSON,
		http.AuthorizationHeader: s.clientKeys.SauronApiKey,
	}
	req := &http.Request{URL: "cmp_payer/", CustomHeaders: headers, Payload: data}

	resp, err := s.sauronClient.Post(ctx, req)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("error linking MPA to Sauron. Sauron request failed with status %d and message %s", resp.StatusCode, string(resp.Body))
	}

	return nil
}
