package partnersales

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/channel/apiv1/channelpb"
	"cloud.google.com/go/firestore"
	"google.golang.org/genproto/googleapis/type/postaladdress"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/priority"
	"github.com/doitintl/hello/scheduled-tasks/priority/domain"
)

type ChannelServicesCustomer struct {
	Customer       *firestore.DocumentRef `firestore:"customer"`
	Domain         string                 `firestore:"domain"`
	OrgDisplayName string                 `firestore:"orgDisplayName"`
	Timestamp      time.Time              `firestore:"timestamp,serverTimestamp"`
}

func getChannelCustomerFullID(customerID string) string {
	return partnerAccountName + "/customers/" + customerID
}

// CreateCustomer creates a customer in Partner Sales Console using Channel Services API
// If customer already exists in Partner Sales Console, pointer to existing customer
// object is returned
// Channel customer resource is required for creating new billing account
//
//	customerID - CMP customer ID
//
// Output:
//
//	*channelpb.Customer - pointer to new channel customer object
func (s *GoogleChannelService) CreateCustomer(ctx context.Context, customerID string) (*channelpb.Customer, error) {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)

	l.Info("ChannelServices - Customer Create")

	customerRef := fs.Collection("customers").Doc(customerID)
	if channelCustomer, err := s.getExistingCustomer(ctx, customerRef); err != nil {
		return nil, err
	} else if channelCustomer != nil {
		l.Info("Customer exists id " + channelCustomer.Name)
		l.Info(protojson.Format(channelCustomer))

		return channelCustomer, nil
	}

	request, err := s.buildCreateCustomerRequest(ctx, customerRef)
	if err != nil {
		l.Error("Failed creating channel request")
		return nil, err
	}

	channelCustomer, err := s.exhaustiveCreateCustomer(ctx, request)
	if err != nil {
		l.Error("Failed creating channel customer")
		return nil, err
	}

	l.Info("Created customer id " + channelCustomer.Name)
	l.Info(protojson.Format(channelCustomer))

	// Update firestore integrations db
	channelCustomerID := s.getChannelCustomerID(channelCustomer)

	if _, err := fs.Collection("integrations").
		Doc("google-cloud").
		Collection("googlePartnerSalesCustomers").
		Doc(channelCustomerID).
		Set(ctx, ChannelServicesCustomer{
			Customer:       customerRef,
			Domain:         request.Customer.Domain,
			OrgDisplayName: request.Customer.OrgDisplayName,
		}); err != nil {
		l.Error("Failed storing channel customer " + customerID + ", channel id: " + channelCustomerID)
		return nil, err
	}

	return channelCustomer, nil
}

func (s *GoogleChannelService) exhaustiveCreateCustomer(ctx context.Context, request *channelpb.CreateCustomerRequest) (*channelpb.Customer, error) {
	client := s.cloudChannel

	channelCustomer, err := client.CreateCustomer(ctx, request)
	if err != nil {
		if !s.isAddressError(err) {
			return nil, err
		}

		// if customer creation failed due to address error try using only customer country code if exists
		if request.Customer.OrgPostalAddress != nil &&
			request.Customer.OrgPostalAddress.RegionCode != "" {
			request.Customer.OrgPostalAddress = &postaladdress.PostalAddress{
				RegionCode: request.Customer.OrgPostalAddress.RegionCode,
			}

			channelCustomer, err = client.CreateCustomer(ctx, request)
			if err != nil && !s.isAddressError(err) {
				return nil, err
			} else if err == nil {
				return channelCustomer, nil
			}
		}
		// if no country code exists or second trial was unsuccessful
		// use default customer address (doit address)
		request.Customer.OrgPostalAddress = defaultCustomerAddress

		channelCustomer, err = client.CreateCustomer(ctx, request)
		if err != nil {
			return nil, err
		}
	}

	return channelCustomer, nil
}

func (s *GoogleChannelService) isAddressError(err error) bool {
	if status.Code(err) == codes.InvalidArgument {
		for _, f := range nonAddressFields {
			if strings.Contains(err.Error(), f) {
				return false
			}
		}

		return true
	}

	return false
}

func (s *GoogleChannelService) getExistingCustomer(ctx context.Context, customerRef *firestore.DocumentRef) (*channelpb.Customer, error) {
	l := s.loggerProvider(ctx)
	fs := s.conn.Firestore(ctx)
	client := s.cloudChannel

	docSnaps, err := fs.Collection("integrations").
		Doc("google-cloud").
		Collection("googlePartnerSalesCustomers").
		Where("customer", "==", customerRef).
		Limit(1).Documents(ctx).GetAll()
	if err != nil {
		l.Error("Failed reading customers")
		return nil, err
	}

	if len(docSnaps) == 0 {
		return nil, nil
	}

	customer, err := client.GetCustomer(ctx,
		&channelpb.GetCustomerRequest{
			Name: getChannelCustomerFullID(docSnaps[0].Ref.ID),
		},
	)
	if err != nil {
		l.Error("Failed reading existing channel customer with error: ", err)
		return nil, err
	}

	return customer, nil
}

func (s *GoogleChannelService) buildCreateCustomerRequest(ctx context.Context, customerRef *firestore.DocumentRef) (*channelpb.CreateCustomerRequest, error) {
	l := s.loggerProvider(ctx)

	customer, err := common.GetCustomer(ctx, customerRef)
	if err != nil {
		l.Error("Failed reading customer data")
		return nil, err
	}

	orgDisplayName := customer.Name
	// When creating customer from dev env, add DEV prefix
	if !common.Production {
		orgDisplayName = fmt.Sprintf("[DEV] %s", orgDisplayName)
	}

	// Create the Customer resource
	request := &channelpb.CreateCustomerRequest{
		Parent: partnerAccountName,
		Customer: &channelpb.Customer{
			OrgDisplayName: orgDisplayName,
			Domain:         customer.PrimaryDomain,
		},
	}

	request.Customer.OrgPostalAddress, err = s.getCustomerAddress(ctx, customer)
	if err != nil {
		l.Error("Failed fetching customer address with priority service")
		return nil, err
	}

	return request, nil
}

func (s *GoogleChannelService) getCustomerAddress(ctx context.Context, customer *common.Customer) (*postaladdress.PostalAddress, error) {
	fs := s.conn.Firestore(ctx)

	if len(customer.Entities) < 1 {
		return nil, errors.New("Customer has no billing profile info")
	}

	docSnap, err := customer.Entities[0].Get(ctx)
	if err != nil {
		return nil, err
	}

	var entity common.Entity
	if err := docSnap.DataTo(&entity); err != nil {
		return nil, err
	}

	var priorityAddress domain.CustomerAddress

	params := make(map[string][]string)
	params["$select"] = []string{"COUNTRYNAME,ADDRESS,ADDRESS2,STATE,STATEA,STATENAME,ZIP"}

	if body, err := priority.Client.Get(entity.PriorityCompany, "CUSTOMERS/"+entity.PriorityID, params); err != nil {
		return nil, err
	} else if err := json.Unmarshal(body, &priorityAddress); err != nil {
		return nil, err
	}

	// Get customer country code
	var countriesCodeMap common.CountriesInfo

	contriesDocSnaps, err := fs.Collection("app").Doc("countries").Get(ctx)
	if err != nil {
		return nil, err
	}

	if err := contriesDocSnaps.DataTo(&countriesCodeMap); err != nil {
		return nil, err
	}

	var countryCode string

	for code, country := range countriesCodeMap.Code {
		if country.Name == priorityAddress.CountryName {
			countryCode = code
			break
		}
	}

	address := postaladdress.PostalAddress{
		PostalCode: priorityAddress.Zip,
		RegionCode: countryCode,
		Locality:   priorityAddress.City,
	}

	// For US customers, take the STATENAME field from priority
	// Otherwise, take the STATEA field we use in the billing profile address form
	if entity.PriorityCompany == string(priority.CompanyCodeUSA) && priorityAddress.StateName != "" {
		address.Locality = priorityAddress.StateName
	} else if priorityAddress.StateA != "" {
		address.Locality = priorityAddress.StateA
	} else {
		address.Locality = priorityAddress.State
	}

	if priorityAddress.Address != "" {
		address.AddressLines = []string{priorityAddress.Address}
	}

	return &address, nil
}
