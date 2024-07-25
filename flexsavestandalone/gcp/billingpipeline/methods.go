package billingpipeline

import (
	"context"

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
		StandardProjectID:   tables.StandardProjectID,
		StandardDataset:     tables.StandardDatasetID,
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
		StandardProjectID:   tables.StandardProjectID,
		StandardDataset:     tables.StandardDatasetID,
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
