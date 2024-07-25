package common

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/doitintl/firestore/pkg"
)

type CustomerClassification string

// Customer classification types
const (
	CustomerClassificationBusiness               CustomerClassification = "business"
	CustomerClassificationStrategic              CustomerClassification = "strategic"
	CustomerClassificationTerminated             CustomerClassification = "terminated"
	CustomerClassificationInactive               CustomerClassification = "inactive"
	CustomerClassificationSuspendedForNonPayment CustomerClassification = "suspendedForNonPayment"
)

type CustomerSecurityMode string

// Customer security mode types
const (
	CustomerSecurityModeNone       CustomerSecurityMode = "none"
	CustomerSecurityModeRestricted CustomerSecurityMode = "restricted"
)

// PlatformAccountManager ..
type PlatformAccountManager struct {
	AccountManager1 *AccountManagerRef `firestore:"account_manager"`   // FSR
	AccountManager2 *AccountManagerRef `firestore:"customer_engineer"` // SAM
	AccountManager3 *AccountManagerRef `firestore:"partner_sales_engineer,omitempty"`
}

type AccountManagerRef struct {
	Ref          *firestore.DocumentRef `firestore:"ref"`
	Notification int64                  `firestore:"notification"`
}

// AccountTeam
type AccountTeamMember struct {
	Ref                      *firestore.DocumentRef `firestore:"ref"`
	Company                  AccountManagerCompany  `firestore:"company"`
	SupportNotificationLevel int64                  `firestore:"supportNotificationLevel"`
}

type MarketplaceSettings struct {
	AccountExists bool `firestore:"accountExists"`
}

type Marketplace struct {
	GCP *MarketplaceSettings `firestore:"GCP,omitempty"`
	AWS *MarketplaceSettings `firestore:"AWS,omitempty"`
}

type SegmentValue string

const (
	Invest     SegmentValue = "Invest"
	Incubate   SegmentValue = "Incubate"
	Accelerate SegmentValue = "Accelerate"
)

type CustomerSegment struct {
	CurrentSegment  SegmentValue `firestore:"currentSegment"`
	OverrideSegment SegmentValue `firestore:"overrideSegment"`
}

// Customer represents a firestore customer document
type Customer struct {
	Name                    string                             `firestore:"name"`
	LowerName               string                             `firestore:"_name"`
	PrimaryDomain           string                             `firestore:"primaryDomain"`
	Domains                 []string                           `firestore:"domains"`
	Assets                  []string                           `firestore:"assets"`
	AccountManager          *firestore.DocumentRef             `firestore:"accountManager"`
	AccountManagers         map[string]*PlatformAccountManager `firestore:"accountManagers"`
	AccountTeam             []*AccountTeamMember               `firestore:"accountTeam"`
	Entities                []*firestore.DocumentRef           `firestore:"entities"`
	SharedDriveFolderID     *string                            `firestore:"sharedDriveFolderId"`
	Classification          CustomerClassification             `firestore:"classification"`
	SecurityMode            *CustomerSecurityMode              `firestore:"securityMode"`
	TimeCreated             time.Time                          `firestore:"timeCreated"`
	Enrichment              *CustomerEnrichment                `firestore:"enrichment"`
	TrialEndDate            *time.Time                         `firestore:"trialEndDate"`
	Settings                *CustomerSettings                  `firestore:"settings"`
	Auth                    Auth                               `firestore:"auth"`
	Type                    *string                            `firestore:"type"`
	Subscribers             []string                           `firestore:"subscribers"`
	EnabledFlexsave         *CustomerEnabledFlexsave           `firestore:"enabledFlexsave,omitempty"`
	EnabledSaaSConsole      *CustomerEnabledSaaSConsole        `firestore:"enabledSaaSConsole,omitempty"`
	Marketplace             *Marketplace                       `firestore:"marketplace,omitempty"`
	EarlyAccessFeatures     []string                           `firestore:"earlyAccessFeatures,omitempty"`
	InvoiceAttributionGroup *firestore.DocumentRef             `firestore:"invoiceAttributionGroup"`
	CustomerSegment         *CustomerSegment                   `firestore:"customerSegment"`
	InvoicesOnHold          map[string]*CloudOnHoldDetails     `firestore:"invoicesOnHold"`
	PresentationMode        *PresentationMode                  `firestore:"presentationMode"`
	Tiers                   map[string]*pkg.CustomerTier       `firestore:"tiers"`

	Snapshot *firestore.DocumentSnapshot `firestore:"-"`
	ID       string                      `firestore:"-"`
}

type PresentationMode struct {
	IsPredefined bool   `firestore:"isPredefined"`
	Enabled      bool   `firestore:"enabled"`
	CustomerID   string `firestore:"customerId"`
}

type CloudOnHoldDetails struct {
	Note      string    `firestore:"note"`
	Timestamp time.Time `firestore:"timestamp"`
	Email     string    `firestore:"email"`
}

type CustomerEnabledFlexsave struct {
	AWS bool `firestore:"AWS"`
	GCP bool `firestore:"GCP"`
}

type CustomerEnabledSaaSConsole struct {
	AWS bool `firestore:"AWS"`
	GCP bool `firestore:"GCP"`
}

// Auth holding data requires for customer's users for authentication
// TODO
type Auth struct {
	Sso *CustomerAuthSso `firestore:"sso"`
}

type CustomerAuthSso struct {
	SAML *string `firestore:"saml"`
	OIDC *string `firestore:"oidc"`
}

type AWSSettings struct {
	IsRecalculated             bool `firestore:"isRecalculated"`
	UseAnalyticsDataForInvoice bool `firestore:"useAnalyticsDataForInvoice"`
}

type CustomerEnrichment struct {
	Logo      *string                `firestore:"logo"`
	HubspotID *string                `firestore:"hubspotId"`
	Geo       *CustomerEnrichmentGeo `firestore:"geo"`
}

type CustomerEnrichmentGeo struct {
	CountryCode string `firestore:"countryCode"`
	PostalCode  string `firestore:"postalCode"`
	StreetName  string `firestore:"streetName"`
	City        string `firestore:"city"`
	Country     string `firestore:"country"`
}

type CustomerSettings struct {
	Currency  string                    `firestore:"currency"`
	Timezone  string                    `firestore:"timezone"`
	Invoicing CustomerSettingsInvoicing `firestore:"invoicing"`
}

type CustomerSettingsInvoicing struct {
	MaxLineItems int `firestore:"maxLineItems"`
}

func (c *Customer) Terminated() bool {
	return c.Classification == CustomerClassificationTerminated
}

func (c *Customer) Inactive() bool {
	return c.Classification == CustomerClassificationInactive
}

func (c *Customer) SuspendedForNonPayment() bool {
	return c.Classification == CustomerClassificationSuspendedForNonPayment
}

func GetCustomer(ctx context.Context, ref *firestore.DocumentRef) (*Customer, error) {
	if ref == nil {
		return nil, errors.New("invalid nil customer ref")
	}

	docSnap, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}

	var customer Customer
	if err := docSnap.DataTo(&customer); err != nil {
		return nil, err
	}

	customer.Snapshot = docSnap

	return &customer, nil
}

func GetAccountManager(ctx context.Context, ref *firestore.DocumentRef) (*AccountManager, error) {
	if ref == nil {
		return nil, errors.New("invalid nil account manager ref")
	}

	docSnap, err := ref.Get(ctx)
	if err != nil {
		return nil, err
	}

	var accountManager AccountManager
	if err := docSnap.DataTo(&accountManager); err != nil {
		return nil, err
	}

	return &accountManager, nil
}

func GetCustomerAccountManagersForCustomerID(ctx context.Context, customerID string, company AccountManagerCompany) ([]*AccountManager, error) {
	customerRef := fs.Collection("customers").Doc(customerID)

	customer, err := GetCustomer(ctx, customerRef)
	if err != nil {
		return nil, err
	}

	return GetCustomerAccountManagers(ctx, customer, company)
}

// GetCustomerAccountManagers retrieves the customer 2 account managers for a vendor
func GetCustomerAccountManagers(ctx context.Context, customer *Customer, company AccountManagerCompany) ([]*AccountManager, error) {
	if company == "" {
		return nil, errors.New("missing param vendor")
	}

	var accountManagers []*AccountManager

	for _, accountManager := range customer.AccountTeam {
		if accountManager.Company != company {
			continue
		}

		if am, err := GetAccountManager(ctx, accountManager.Ref); err == nil {
			accountManagers = append(accountManagers, am)
		}
	}

	return accountManagers, nil
}

// GetCustomerAccountManagersEmails retrieves a list with the emails of the account managers assigned to a customer for a vendor
func GetCustomerAccountManagersEmails(ctx context.Context, customer *Customer, company AccountManagerCompany) ([]*mail.Email, error) {
	amEmails := make([]*mail.Email, 0)

	ams, err := GetCustomerAccountManagers(ctx, customer, company)
	if err != nil {
		return nil, err
	}

	for _, am := range ams {
		amEmails = append(amEmails, mail.NewEmail(am.Name, am.Email))
	}

	return amEmails, nil
}

// GetCustomerUsersWithPermissions returns a customer's users that have any of the permissions in the permissions slice
func GetCustomerUsersWithPermissions(ctx context.Context, fs *firestore.Client, customer *firestore.DocumentRef, permissions []string) ([]*User, error) {
	if customer == nil {
		return nil, errors.New("invalid nil customer reference")
	}

	if len(permissions) == 0 {
		return nil, errors.New("permissions must not be empty")
	}

	if len(permissions) > 10 {
		return nil, errors.New("maximum amount of permissions to search is 10") // firestore limitation for array-contains-any
	}

	uniques := make(map[string]struct{})
	users := make([]*User, 0)
	rolesRefs := make([]*firestore.DocumentRef, 0)

	// Create slice of references to the permissions to be used in the querie below
	permissionsRefs := make([]*firestore.DocumentRef, len(permissions))
	for i, permission := range permissions {
		permissionsRefs[i] = fs.Collection("permissions").Doc(permission)
	}

	// Fetch custom roles that include the required permissions
	customRoleDocSnaps, err := fs.Collection("roles").
		Where("type", "==", "custom").
		Where("customer", "==", customer).
		Where("permissions", "array-contains-any", permissionsRefs).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	// Fetch preset roles that include the required permissions
	presetRolesDocSnaps, err := fs.Collection("roles").
		Where("type", "==", "preset").
		Where("customer", "==", nil).
		Where("permissions", "array-contains-any", permissionsRefs).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	// Fetch users with the legacy permission
	// Remove this query when the legacy permissions are deprecated
	permissionsUserDocSnaps, err := fs.Collection("users").
		Where("customer.ref", "==", customer).
		Where("permissions", "array-contains-any", permissions).
		Documents(ctx).GetAll()
	if err != nil {
		return nil, err
	}

	for _, docSnap := range permissionsUserDocSnaps {
		var user User
		if err := docSnap.DataTo(&user); err != nil {
			return nil, err
		}

		// Don't include users that have roles but somehow still have legacy permissions
		if user.Roles != nil && len(user.Roles) > 0 {
			continue
		}

		users = append(users, &user)
		uniques[docSnap.Ref.ID] = struct{}{}
	}

	// Fill a slice of references to roles that include one of the required permissions
	for _, docSnap := range append(presetRolesDocSnaps, customRoleDocSnaps...) {
		rolesRefs = append(rolesRefs, docSnap.Ref)
	}

	if len(rolesRefs) > 0 {
		// Query users that have any of the roles that contains the required permissions
		// Make a query for each 10 roles because of array-contains-any limitations
		rolesRefsSize := len(rolesRefs)
		maxArrayContainsAnySize := 10

		for i, j := 0, 0; i < rolesRefsSize; i += maxArrayContainsAnySize {
			j += maxArrayContainsAnySize
			if j > rolesRefsSize {
				j = rolesRefsSize
			}

			subSlice := rolesRefs[i:j]

			rolesUserDocSnaps, err := fs.Collection("users").
				Where("customer.ref", "==", customer).
				Where("roles", "array-contains-any", subSlice).
				Documents(ctx).GetAll()
			if err != nil {
				return nil, err
			}

			for _, docSnap := range rolesUserDocSnaps {
				// Skip users we have already added before
				if _, prs := uniques[docSnap.Ref.ID]; prs {
					continue
				}

				var user User
				if err := docSnap.DataTo(&user); err != nil {
					return nil, err
				}

				users = append(users, &user)
			}
		}
	}

	return users, nil
}

func GetCustomerOrgs(
	ctx context.Context,
	fs *firestore.Client,
	customerRef *firestore.DocumentRef,
	orgID string,
) ([]*Organization, error) {
	if customerRef == nil {
		return nil, errors.New("invalid nil customer ref")
	}

	var orgs []*Organization

	orgsCollection := customerRef.Collection("customerOrgs")
	if orgID == "" {
		orgSnaps, err := orgsCollection.Documents(ctx).GetAll()
		if err != nil {
			return nil, err
		}

		presetOrgSnaps, err := fs.Collection("organizations").Documents(ctx).GetAll()
		if err != nil {
			return nil, err
		}

		for _, orgSnap := range append(orgSnaps, presetOrgSnaps...) {
			var org Organization
			if err := orgSnap.DataTo(&org); err != nil {
				return nil, err
			}

			org.Snapshot = orgSnap
			orgs = append(orgs, &org)
		}
	} else {
		var org Organization

		orgSnap, err := orgsCollection.Doc(orgID).Get(ctx)
		if err != nil {
			return nil, err
		}

		if err := orgSnap.DataTo(&org); err != nil {
			return nil, err
		}

		org.Snapshot = orgSnap
		orgs = append(orgs, &org)
	}

	return orgs, nil
}

// GetCustomerIsRecalculatedFlag returns whether a customer aws billing is recalculated
// note: the default IsRecalculated flag value is false
func GetCustomerIsRecalculatedFlag(ctx context.Context, customerRef *firestore.DocumentRef) (bool, error) {
	if customerRef == nil {
		return false, errors.New("customer reference is nil")
	}

	docSnap, err := customerRef.Collection("accountConfiguration").Doc(Assets.AmazonWebServices).Get(ctx)
	if err != nil {
		// If doc not found it means that the flag was never set to true. Don't throw an error and return the default value which is false
		return false, nil
	}

	var awsSettings *AWSSettings

	if err := docSnap.DataTo(&awsSettings); err != nil {
		// Doc format is not correct. Don't throw an error and return the default value which is false
		return false, nil
	}

	return awsSettings.IsRecalculated, nil
}

func GetCustomerCurrency(customer *Customer) string {
	if customer.Settings != nil {
		return customer.Settings.Currency
	}

	return "USD"
}

func GetCustomerRef(customerID string) *firestore.DocumentRef {
	return fs.Collection("customers").Doc(customerID)
}
