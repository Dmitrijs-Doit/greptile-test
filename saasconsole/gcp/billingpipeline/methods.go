package billingpipeline

import (
	"context"
	"strings"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/http"
)

func (s *service) TestConnection(ctx context.Context, billingAccountID, serviceAccountEmail string, tables *pkg.BillingTablesLocation) error {
	if tables == nil {
		return ErrRequestBodyIsNil
	}

	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return err
	}

	req := TestRequestBody{
		ServiceAccountEmail: serviceAccountEmail,
		BillingAccountID:    billingAccountID,
		DetailedProjectID:   tables.DetailedProjectID,
		DetailedDataset:     tables.DetailedDatasetID,
	}

	_, err = s.apiClient.Post(ctx, &http.Request{
		URL:     "/standalone/billing-pipeline/test",
		Payload: req,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *service) Onboard(ctx context.Context, customerID, billingAccountID, serviceAccountEmail string, tables *pkg.BillingTablesLocation) error {
	if tables == nil {
		return ErrRequestBodyIsNil
	}

	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return err
	}

	req := OnboardRequestBody{
		CustomerID:          customerID,
		ServiceAccountEmail: serviceAccountEmail,
		BillingAccountID:    billingAccountID,
		DetailedProjectID:   tables.DetailedProjectID,
		DetailedDataset:     tables.DetailedDatasetID,
	}

	_, err = s.apiClient.Post(ctx, &http.Request{
		URL:     "/standalone/billing-pipeline/onboard",
		Payload: req,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *service) GetAccountBillingDataStatus(ctx context.Context, customerID, billingAccountID string) (AccountDataStatus, error) {
	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return AccountDataStatusMissing, err
	}

	req := AccountRequestBody{
		CustomerID:       customerID,
		BillingAccountID: billingAccountID,
	}

	res, err := s.apiClient.Post(ctx, &http.Request{
		URL:     "/standalone/billing-pipeline/status",
		Payload: req,
	})
	if err != nil {
		return AccountDataStatusMissing, err
	}

	status := strings.Replace(string(res.Body), "\"", "", -1) // remove quotes from response string

	return validateStatus(status), nil
}

func validateStatus(status string) AccountDataStatus {
	switch status {
	case string(AccountDataStatusImportRunning),
		string(AccountDataStatusImportFinished),
		string(AccountDataStatusHistoryFinished),
		string(AccountDataStatusImportPaused):
		return AccountDataStatus(status)
	default:
		return AccountDataStatusMissing
	}
}

func (s *service) PauseAccounts(ctx context.Context, billingAccountIDs []string) error {
	ctx, err := s.setBearerToken(ctx)
	if err != nil {
		return err
	}

	req := PauseAccountsRequestBody{
		BillingAccountIDs: billingAccountIDs,
	}

	_, err = s.apiClient.Post(ctx, &http.Request{
		URL:     "/standalone/billing-pipeline/pause-accounts",
		Payload: req,
	})

	return err
}
