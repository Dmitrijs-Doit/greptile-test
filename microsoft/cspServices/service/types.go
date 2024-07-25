package service

import (
	"net/http"

	"github.com/doitintl/cloudtasks/iface"

	"github.com/doitintl/hello/scheduled-tasks/microsoft"
	httpClient "github.com/doitintl/http"
)

type SubscriptionsService struct {
	s *CSPServiceClient
}

type CustomersService struct {
	s *CSPServiceClient

	Users *UsersService
}

type CSPServiceClient struct {
	//client will be removed in the future
	client           *http.Client
	httpClient       httpClient.IClient
	cloudTaskService iface.CloudTaskClient
	accessToken      microsoft.IAccessToken
}

type CSPService struct {
	Customers     ICustomersService
	Subscriptions ISubscriptionsService
}

type Agreement struct {
	TemplateID     string                  `json:"templateId"`
	UserID         string                  `json:"userId"`
	DateAgreed     string                  `json:"dateAgreed"`
	Type           string                  `json:"type"`
	AgreementLink  string                  `json:"agreementLink"`
	PrimaryContact AgreementPrimaryContact `json:"primaryContact"`
}

type AgreementPrimaryContact struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Phone     string `json:"phoneNumber"`
}

type AgreementMetadata struct {
	TemplateID    string `json:"templateId"`
	AgreementType string `json:"agreementType"`
}

type AgreementsMetadataResponse struct {
	TotalCount int64                `json:"totalCount"`
	Items      []*AgreementMetadata `json:"items"`
}
