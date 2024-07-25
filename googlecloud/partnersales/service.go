package partnersales

import (
	"context"
	"errors"
	"regexp"
	"strings"

	channel "cloud.google.com/go/channel/apiv1"
	"cloud.google.com/go/channel/apiv1/channelpb"
	"google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/option"
	"google.golang.org/genproto/googleapis/type/postaladdress"

	"github.com/doitintl/hello/scheduled-tasks/framework/connection"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
)

const (
	partnerAccountID   string = "C03rw2ty2"
	partnerAccountName string = "accounts/" + partnerAccountID
	authScope          string = "https://www.googleapis.com/auth/apps.order"
)

type GoogleChannelService struct {
	loggerProvider logger.Provider
	conn           *connection.Connection
	cloudChannel   *channel.CloudChannelClient
	cloudBilling   *cloudbilling.APIService
}

var (
	ErrDisplayNameMissing    = errors.New("missing billing account display name")
	ErrFailedToFetchGCPOffer = errors.New("failed to fetch google cloud offer")
)

var (
	// NamePattern - Billing Account display name pattern
	NamePattern             *regexp.Regexp
	MissingUserErrorPattern *regexp.Regexp

	// nonAddressFields - customer create request non adress required fields
	nonAddressFields = []string{"org_display_name", "domain"}
)

// DefaultAddress - default customer address
var defaultCustomerAddress = &postaladdress.PostalAddress{
	RegionCode:   "IL",
	Locality:     "Tel Aviv",
	AddressLines: []string{"Rav Aluf David Elazar 12"},
	PostalCode:   "6107408",
}

func init() {
	NamePattern = regexp.MustCompile("^[a-z0-9]+(?:-[a-z0-9]+)*(?:\\.[a-z0-9]+(?:-[a-z0-9]+)*)*$")
	MissingUserErrorPattern = regexp.MustCompile("^User ([^\\s]+) does not exist.$")
}

func NewGoogleChannelService(loggerProvider logger.Provider, conn *connection.Connection) (*GoogleChannelService, error) {
	ctx := context.Background()

	cloudChannel, err := newCloudChannel(ctx)
	if err != nil {
		return nil, err
	}

	cloudBilling, err := newCloudBilling(ctx)
	if err != nil {
		return nil, err
	}

	return &GoogleChannelService{
		loggerProvider,
		conn,
		cloudChannel,
		cloudBilling,
	}, nil
}

func newCloudChannel(ctx context.Context) (*channel.CloudChannelClient, error) {
	secret, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretGoogleChannelServices)
	if err != nil {
		return nil, err
	}

	creds := option.WithCredentialsJSON(secret)

	client, err := channel.NewCloudChannelClient(ctx, creds)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func newCloudBilling(ctx context.Context) (*cloudbilling.APIService, error) {
	secret, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretCloudBilling)
	if err != nil {
		return nil, err
	}

	creds := option.WithCredentialsJSON(secret)

	client, err := cloudbilling.NewService(ctx, creds)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (s *GoogleChannelService) getChannelCustomerID(customer *channelpb.Customer) string {
	nameSplit := strings.Split(customer.Name, "/")
	return nameSplit[len(nameSplit)-1]
}
