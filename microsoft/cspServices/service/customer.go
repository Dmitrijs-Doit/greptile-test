package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/microsoft"
	httpClient "github.com/doitintl/http"
)

func NewCustomersService(s *CSPServiceClient) *CustomersService {
	rs := &CustomersService{s: s, Users: NewUsersService(s)}
	return rs
}

func (r *CustomersService) AgreementsMetadata(ctx context.Context) (*AgreementMetadata, error) {
	sfCtx, err := r.s.accessToken.GetAuthenticatedCtx(ctx)
	if err != nil {
		return nil, err
	}

	var a AgreementsMetadataResponse

	if _, err = r.s.httpClient.Get(sfCtx, &httpClient.Request{
		URL:          "/v1/agreements?agreementType=MicrosoftCustomerAgreement",
		ResponseType: &a,
	}); err != nil {
		return nil, err
	}

	if a.TotalCount > 0 {
		return a.Items[0], nil
	}

	return nil, errors.New("failed to get agreements metadata")
}

func (r *CustomersService) AcceptAgreement(ctx context.Context, customerID, email, name string) error {
	sfCtx, err := r.s.accessToken.GetAuthenticatedCtx(ctx)
	if err != nil {
		return err
	}

	metadata, err := r.AgreementsMetadata(ctx)
	if err != nil {
		return err
	}

	var lastName = "API"

	nameParts := strings.SplitN(name, " ", 2)

	firstName := nameParts[0]

	if len(nameParts) > 1 {
		lastName = nameParts[1]
	}

	body := Agreement{
		TemplateID: metadata.TemplateID,
		Type:       metadata.AgreementType,
		DateAgreed: time.Now().UTC().Format(time.RFC3339),
		UserID:     "fcf1461e-e77a-479f-9b75-8fa1cd473ca8",
		PrimaryContact: AgreementPrimaryContact{
			FirstName: firstName,
			LastName:  lastName,
			Email:     email,
		},
	}

	_, err = r.s.httpClient.Post(sfCtx, &httpClient.Request{
		URL:     fmt.Sprintf("/v1/customers/%s/agreements", customerID),
		Payload: body,
	})

	if err != nil {
		return err
	}

	return nil
}

func (r *CustomersService) Get(ctx context.Context, customerID string) (*microsoft.Customer, error) {
	if err := r.s.accessToken.Refresh(); err != nil {
		return nil, err
	}

	sfCtx := httpClient.WithBearerAuth(ctx, &httpClient.BearerAuthContextData{
		Token: fmt.Sprintf("%s %s", r.s.accessToken.GetTokenType(), r.s.accessToken.GetAccessToken()),
	})

	var c microsoft.Customer

	_, err := r.s.httpClient.Get(sfCtx, &httpClient.Request{
		URL:          fmt.Sprintf("/v1/customers/%s", customerID),
		ResponseType: &c,
	})

	if err != nil {
		return nil, err
	}

	return &c, nil
}
