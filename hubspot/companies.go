package hubspot

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold"
	"github.com/doitintl/hello/scheduled-tasks/flexsaveresold/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cloud.google.com/go/firestore"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/dashboard/invoices"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

// CompaniesSearchRes companies search response
type CompaniesSearchRes struct {
	Total   int64              `json:"total"`
	Results []*ReturnedCompany `json:"results"`
}

type ReturnedCompany struct {
	ID         string                   `json:"id"`
	Properties *StringCompanyProperties `json:"properties"`
}

// Company HubSpot Company
type Company struct {
	ID         string             `json:"id"`
	Properties *CompanyProperties `json:"properties"`
}

// CompanyProperties Hubspost company properties
type CompanyProperties struct {
	Domain         string                        `json:"domain"`
	CustomerID     string                        `json:"cmp_external_id"`
	Classification common.CustomerClassification `json:"cmp_classification"`
	FlexSaveMode   string                        `json:"cmp_flexsave_mode"`

	AmazonWebServices bool `json:"cmp_amazon_web_services"`
	GoogleCloud       bool `json:"cmp_google_cloud"`
	GSuite            bool `json:"cmp_g_suite"`
	MicrosoftAzure    bool `json:"cmp_microsoft_azure"`
	Office365         bool `json:"cmp_office_365"`

	GCPServices string `json:"cmp_gcp_services"`
	AWSServices string `json:"cmp_aws_services"`

	GCPRevenue float64 `json:"cmp_gcp_revenue"`
	AWSRevenue float64 `json:"cmp_aws_revenue"`

	FlexSaveAutopilotPotential float64 `json:"cmp_flexsaveautopilotpotential"`
	CustomerType               string  `json:"customer_type"`
}

type StringCompanyProperties struct {
	Domain         string                        `json:"domain"`
	CustomerID     string                        `json:"cmp_external_id"`
	Classification common.CustomerClassification `json:"cmp_classification"`
	FlexSaveMode   string                        `json:"cmp_flexsave_mode"`

	AmazonWebServices string `json:"cmp_amazon_web_services"`
	GoogleCloud       string `json:"cmp_google_cloud"`
	GSuite            string `json:"cmp_g_suite"`
	Office365         string `json:"cmp_office_365"`
	MicrosoftAzure    string `json:"cmp_microsoft_azure"`

	GCPServices string `json:"cmp_gcp_services"`
	AWSServices string `json:"cmp_aws_services"`

	GCPRevenue string `json:"cmp_gcp_revenue"`
	AWSRevenue string `json:"cmp_aws_revenue"`

	FlexSaveAutopilotPotential string `json:"cmp_flexsaveautopilotpotential"`
}

// Hubspot company update payload
type updateHsCompany struct {
	Properties CompanyProperties `json:"properties"`
}

var defaultCompanyRet = []string{
	"domain",
	"hs_additional_domains",
	"cmp_external_id",
	"cmp_classification",
	"cmp_flexsave_mode",

	// Assets
	"cmp_amazon_web_services",
	"cmp_google_cloud",
	"cmp_g_suite",
	"cmp_microsoft_azure",
	"cmp_office_365",

	"cmp_aws_services",
	"cmp_gcp_services",

	"cmp_gcp_revenue",
	"cmp_aws_revenue",

	"cmp_flexsaveautopilotpotential",
}

func (s *HubspotService) SyncCompanyWorker(ctx context.Context, customerID string) error {
	logger := s.Logger(ctx)

	hubspotService, err := NewService(ctx)
	if err != nil {
		return err
	}

	customerRef := s.Firestore(ctx).Collection("customers").Doc(customerID)

	docSnap, err := customerRef.Get(ctx)
	if err != nil {
		return err
	}

	var customer common.Customer
	if err := docSnap.DataTo(&customer); err != nil {
		return err
	}

	customer.Snapshot = docSnap

	// logger.Debug(customer.PrimaryDomain)
	company, err := searchCompany(ctx, hubspotService, customer)
	if err != nil {
		logger.Debugf("searchCompany: %s", err.Error())
		return err
	}

	fullCMPCustomer, err := s.preparePayload(ctx, &customer)
	if err != nil {
		logger.Debugf("preparePayload: %s", err.Error())
		return err
	}

	if company == nil {
		// Company not found, create a new company
		logger.Debugf("creating hubspot company %s", customer.PrimaryDomain)

		if err := hubspotService.Companies.Create(ctx, fullCMPCustomer); err != nil {
			logger.Debugf("Companies.Create: %s", err.Error())
			return err
		}
	} else if !compareCustomerToHS(&fullCMPCustomer.Properties, company.Properties) {
		// Company found but properties needs to be updated
		logger.Debugf("updating hubspot company %s (%s)", customer.PrimaryDomain, company.ID)

		if err := hubspotService.Companies.Update(ctx, fullCMPCustomer, company.ID); err != nil {
			logger.Debugf("Companies.Update: %s", err.Error())
			return err
		}
	}

	// Update customer document with hubspotID
	if company != nil && customer.Enrichment != nil && customer.Enrichment.HubspotID == nil {
		if _, err := customer.Snapshot.Ref.Update(ctx, []firestore.Update{
			{Path: "enrichment.hubspotId", Value: company.ID},
		}); err != nil {
			return err
		}
	}

	return nil
}

// Search firebase and then search hubspot for properties retrieved
func searchCompany(ctx context.Context, hubspotService *Service, customer common.Customer) (*Company, error) {
	company, err := compareFirstPass(ctx, customer, hubspotService)
	if err != nil {
		return nil, err
	}

	if company == nil {
		company, err = compareSecondPass(ctx, customer, hubspotService)
		if err != nil {
			return nil, err
		}
	}

	return company, nil
}

// compareByDomain: compares domain on customer to hubspot domain to check if exists and returns hubspot customer/company if exists
func compareFirstPass(ctx context.Context, customer common.Customer, hubspotService *Service) (*Company, error) {
	filterGroups := []Filters{
		// Check if CMP customer ID equals the HS CMP external ID property
		{
			Filters: []Filter{
				{
					PropertyName: "cmp_external_id",
					Operator:     FilterOperatorEquals,
					Value:        customer.Snapshot.Ref.ID,
				},
			},
		},
		// Or if the CMP primary domain equals an HS primary domain
		{
			Filters: []Filter{
				{
					PropertyName: "domain",
					Operator:     FilterOperatorEquals,
					Value:        customer.PrimaryDomain,
				},
			},
		},
		// Or if the CMP primary domains equals an HS secondary domain
		{
			Filters: []Filter{
				{
					PropertyName: "hs_additional_domains",
					Operator:     FilterOperatorContainsToken,
					Value:        customer.PrimaryDomain,
				},
			},
		},
	}

	company, err := queryHS(ctx, hubspotService, hsReq{
		Properties:   defaultCompanyRet,
		FilterGroups: filterGroups,
		Sorts:        sorts,
	}, customer.Snapshot.Ref.ID)
	if err != nil {
		return nil, err
	}

	return company, nil
}

func compareSecondPass(ctx context.Context, customer common.Customer, hubspotService *Service) (*Company, error) {
	for _, domain := range customer.Domains {
		// skip the primary domain which was already checked
		if domain == customer.PrimaryDomain {
			continue
		}

		company, err := queryHS(ctx, hubspotService, hsReq{
			Properties: defaultCompanyRet,
			FilterGroups: []Filters{
				// check if CMP secondary domain equals an HS primary domain
				{
					Filters: []Filter{
						{
							PropertyName: "domain",
							Operator:     FilterOperatorEquals,
							Value:        domain,
						},
					},
				},
				// Or check if CMP secondary domain equals to an HS secondary domain
				{
					Filters: []Filter{
						{
							PropertyName: "hs_additional_domains",
							Operator:     FilterOperatorContainsToken,
							Value:        domain,
						},
					},
				},
			},
		}, customer.Snapshot.Ref.ID)
		if err != nil {
			return nil, err
		}

		if company != nil {
			return company, nil
		}
	}

	return nil, nil
}

// queryHS: search hubspot based on a property (e.g. domain, address etc.)
func queryHS(ctx context.Context, hubspotService *Service, body hsReq, cmpCustomerID string) (*Company, error) {
	res, err := hubspotService.Search(ctx, body, "company")
	if err != nil {
		return nil, err
	}

	var data CompaniesSearchRes
	if err := json.Unmarshal(res, &data); err != nil {
		return nil, err
	}

	// No companies found
	if data.Total <= 0 {
		return nil, nil
	}

	var (
		foundMatch     bool
		hsCompanyID    string
		hsCompanyProps *StringCompanyProperties
	)

	for _, v := range data.Results {
		// Check if HS company have cmp_external_id
		if v.Properties.CustomerID == cmpCustomerID {
			hsCompanyID = v.ID
			hsCompanyProps = v.Properties
			foundMatch = true

			break
		}
	}

	// If no company have the correct cmp_external_id
	// select the first company without a cmp_external_id
	if !foundMatch {
		for _, v := range data.Results {
			if v.Properties.CustomerID == "" {
				hsCompanyID = v.ID
				hsCompanyProps = v.Properties
				foundMatch = true

				break
			}
		}

		// If match is still not found
		if !foundMatch {
			return nil, nil
		}
	}

	var convertedCompany CompanyProperties
	convertedCompany.CustomerID = hsCompanyProps.CustomerID
	convertedCompany.Classification = hsCompanyProps.Classification
	convertedCompany.FlexSaveMode = hsCompanyProps.FlexSaveMode

	// Assets
	convertedCompany.AmazonWebServices, _ = strconv.ParseBool(hsCompanyProps.AmazonWebServices)
	convertedCompany.GoogleCloud, _ = strconv.ParseBool(hsCompanyProps.GoogleCloud)
	convertedCompany.GSuite, _ = strconv.ParseBool(hsCompanyProps.GSuite)
	convertedCompany.Office365, _ = strconv.ParseBool(hsCompanyProps.Office365)
	convertedCompany.MicrosoftAzure, _ = strconv.ParseBool(hsCompanyProps.MicrosoftAzure)

	// Services
	convertedCompany.GCPServices = hsCompanyProps.GCPServices
	convertedCompany.AWSServices = hsCompanyProps.AWSServices

	// Revenue
	convertedCompany.AWSRevenue, _ = strconv.ParseFloat(hsCompanyProps.AWSRevenue, 64)
	convertedCompany.GCPRevenue, _ = strconv.ParseFloat(hsCompanyProps.GCPRevenue, 64)

	// FlexSaveAutopilotPotential (flexSaveAutopilotSavings)
	convertedCompany.FlexSaveAutopilotPotential, _ = strconv.ParseFloat(hsCompanyProps.FlexSaveAutopilotPotential, 64)

	return &Company{
		ID:         hsCompanyID,
		Properties: &convertedCompany,
	}, nil
}

func (s *HubspotService) preparePayload(ctx context.Context, customer *common.Customer) (*updateHsCompany, error) {
	docSnaps, err := s.Firestore(ctx).Collection("assets").Where("customer", "==", customer.Snapshot.Ref).Where("type", "in", []string{common.Assets.GoogleCloud, common.Assets.AmazonWebServices, common.Assets.MicrosoftAzure, common.Assets.Office365, common.Assets.GSuite}).Select("type").Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	var props CompanyProperties

	for _, assetDocSnap := range docSnaps {
		for _, field := range assetDocSnap.Data() {
			switch field {
			case common.Assets.AmazonWebServices:
				if !props.AmazonWebServices {
					props.AmazonWebServices = true
				}
			case common.Assets.GSuite:
				if !props.GSuite {
					props.GSuite = true
				}
			case common.Assets.GoogleCloud:
				if !props.GoogleCloud {
					props.GoogleCloud = true
				}
			case common.Assets.MicrosoftAzure:
				if !props.MicrosoftAzure {
					props.MicrosoftAzure = true
				}
			case common.Assets.Office365:
				if !props.Office365 {
					props.Office365 = true
				}
			}
		}
	}

	props.Domain = customer.PrimaryDomain
	props.Classification = customer.Classification
	props.CustomerID = customer.Snapshot.Ref.ID
	props.FlexSaveMode = flexsaveresold.OrderExecUnmanaged

	flexsaveData, err := s.Firestore(ctx).Collection("integrations").Doc("flexsave").Collection("configuration").Doc(customer.Snapshot.Ref.ID).Get(ctx)
	if err != nil && status.Code(err) != codes.NotFound {
		return nil, err
	}

	if flexsaveData.Exists() {
		var config types.ConfigData
		if err := flexsaveData.DataTo(&config); err != nil {
			return nil, err
		}

		if config.AWS.Enabled && config.AWS.SavingsSummary != nil {
			props.FlexSaveMode = flexsaveresold.OrderExecAutopilot
			props.FlexSaveAutopilotPotential = config.AWS.SavingsSummary.NextMonth.Savings
		}
	}

	cloudServicesDocSnaps, err := s.Firestore(ctx).Collection("customers").
		Doc(props.CustomerID).
		Collection("cloudServices").
		Select("serviceName", "type").
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	awsServicesMap := awsServices()
	gcpServicesSlice := make([]string, 0)
	awsServicesSlice := make([]string, 0)

	for _, serviceDocSnap := range cloudServicesDocSnaps {
		serviceType := serviceDocSnap.Data()["type"].(string)
		serviceName := serviceDocSnap.Data()["serviceName"].(string)

		if serviceType == common.Assets.GoogleCloud {
			if _, prs := gcpServices[serviceName]; prs {
				gcpServicesSlice = append(gcpServicesSlice, serviceName)
			}
		}

		if serviceType == common.Assets.AmazonWebServices {
			trimmedServiceName := strings.TrimSpace(strings.TrimSuffix(serviceName, " - Direct"))
			if _, prs := awsServicesMap[trimmedServiceName]; prs {
				awsServicesSlice = append(awsServicesSlice, trimmedServiceName)
			}
		}
	}

	props.GCPServices = strings.Join(gcpServicesSlice, HubspotArraySeparator)
	props.AWSServices = strings.Join(awsServicesSlice, HubspotArraySeparator)
	props.CustomerType = "Resold"

	if productOnly, err := s.customerTypeDal.IsProductOnlyCustomerType(ctx, customer.Snapshot.Ref.ID); err != nil {
		return nil, err
	} else if productOnly {
		props.CustomerType = "SaaS"
	}

	now := time.Now().UTC()
	currentYear, currentMonth, currentDay := now.Date()

	var (
		startDate time.Time
		endDate   time.Time
	)

	startDate = time.Date(currentYear, currentMonth-1, 1, 0, 0, 0, 0, time.UTC)
	endDate = now

	if currentDay < 11 {
		startDate = time.Date(currentYear, currentMonth-2, 1, 0, 0, 0, 0, time.UTC)
		endDate = time.Date(currentYear, currentMonth-1, 0, 0, 0, 0, 0, time.UTC)
	}

	query := s.Firestore(ctx).Collection("invoices").Where("customer", "==", customer.Snapshot.Ref).
		Where("CANCELED", "==", false).
		Where("PRODUCTS", "array-contains-any", []string{common.Assets.GoogleCloud, common.Assets.AmazonWebServices}).
		Where("IVDATE", ">=", startDate).
		Where("IVDATE", "<=", endDate)

	fullInvoicesDocSnaps, err := query.OrderBy("IVDATE", firestore.Desc).Select("USDTOTAL", "PRODUCTS", "IVDATE").Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	for _, fullInvoiceDocSnap := range fullInvoicesDocSnaps {
		var invoice invoices.FullInvoice
		if err := fullInvoiceDocSnap.DataTo(&invoice); err != nil {
			return nil, err
		}

		products := invoice.Products

		for _, product := range products {
			if product == common.Assets.GoogleCloud {
				props.GCPRevenue += invoice.USDTotal
			}

			if product == common.Assets.AmazonWebServices {
				props.AWSRevenue += invoice.USDTotal
			}
		}
	}

	payload := &updateHsCompany{
		Properties: props,
	}

	return payload, nil
}

func compareCustomerToHS(customer *CompanyProperties, HS *CompanyProperties) bool {
	classificationEq := customer.Classification == HS.Classification
	flexSaveModeEq := customer.FlexSaveMode == HS.FlexSaveMode
	AWSEq := customer.AmazonWebServices == HS.AmazonWebServices
	googleCloudEq := customer.GoogleCloud == HS.GoogleCloud
	gSuiteEq := customer.GSuite == HS.GSuite
	office365Eq := customer.Office365 == HS.Office365
	microsoftAzureEq := customer.MicrosoftAzure == HS.MicrosoftAzure
	GCPRevenueEq := math.Abs(customer.GCPRevenue-HS.GCPRevenue) <= 0.01
	AWSRevenueEq := math.Abs(customer.AWSRevenue-HS.AWSRevenue) <= 0.01
	FlexSaveAutopilotPotentialEq := math.Abs(customer.FlexSaveAutopilotPotential-HS.FlexSaveAutopilotPotential) <= 0.01

	GCPServicesEq := slice.UnorderedSeparatedStringsComp(customer.GCPServices, HS.GCPServices, HubspotArraySeparator)
	AWSServicesEq := slice.UnorderedSeparatedStringsComp(customer.AWSServices, HS.AWSServices, HubspotArraySeparator)

	result := AWSEq && classificationEq && flexSaveModeEq && office365Eq && microsoftAzureEq && googleCloudEq && gSuiteEq && GCPRevenueEq && AWSRevenueEq && GCPServicesEq && AWSServicesEq && FlexSaveAutopilotPotentialEq

	return result
}
