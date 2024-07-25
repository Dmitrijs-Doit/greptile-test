package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/awsproxy"
)

// List of AWS regions supported for FlexSave for Dedicated Payer accounts
var DedicatedMasterPayerAccountRegions = []string{
	"us-east-1",      // N.Virginia
	"us-east-2",      // Ohio
	"us-west-1",      // N. California
	"us-west-2",      // Oregon
	"eu-central-1",   // Frankfurt
	"eu-west-1",      // Ireland
	"ap-northeast-1", // Tokyo
}

// List of AWS regions supported for FlexSave for Shared Payer accounts
var SharedMasterPayerAccountRegions = []string{
	"us-east-1",      // N.Virginia
	"us-east-2",      // Ohio
	"us-west-1",      // N. California
	"us-west-2",      // Oregon
	"eu-central-1",   // Frankfurt
	"eu-west-1",      // Ireland
	"eu-west-2",      // London
	"eu-west-3",      // Paris
	"ap-northeast-1", // Tokyo
	"ap-northeast-2", // Seoul
	"ap-southeast-1", // Singapore
	"ap-southeast-2", // Sydney
	"ap-south-1",     // Mumbai
	"ca-central-1",   // C. Canada
	"sa-east-1",      // Sao Paulo
}

var AllAwsRegions = []string{
	"us-east-1",      // N.Virginia
	"us-east-2",      // Ohio
	"us-west-1",      // N. California
	"us-west-2",      // Oregon
	"af-south-1",     // Cape Town --
	"ap-east-1",      // Hong Kong --
	"ap-south-1",     // Mumbai
	"ap-northeast-1", // Tokyo
	"ap-northeast-2", // Seoul
	"ap-northeast-3", // Osaka --
	"ap-southeast-1", // Singapore
	"ap-southeast-2", // Sydney
	"ap-southeast-3", // Jakarta
	"ap-southeast-4", // Melbourne
	"ca-central-1",   // Canada --
	"eu-central-1",   // Frankfurt
	"eu-west-1",      // Ireland
	"eu-west-2",      // London
	"eu-west-3",      // Paris
	"eu-south-1",     // Milan
	"eu-south-2",     // Spain
	"eu-north-1",     // Stockholm
	"eu-central-2",   // Zurich
	"me-south-1",     // Bahrain
	"me-central-1",   // UAE
	"sa-east-1",      // São Paulo
	"il-central-1",   // Israel
}

var GovRegions = []string{"us-gov-east-1", "us-gov-west-1"}

type PayerAccount struct {
	AccountID   string `firestore:"id"`
	DisplayName string `firestore:"displayName"`
}

type MasterPayerAccounts struct {
	Accounts map[string]*MasterPayerAccount `firestore:"accounts"`
}

type MasterPayerAccountStatus string

const (
	MasterPayerAccountStatusActive  MasterPayerAccountStatus = "active"
	MasterPayerAccountStatusRetired MasterPayerAccountStatus = "retired"
	MasterPayerAccountStatusPending MasterPayerAccountStatus = "pending"
)

type MasterPayerAccount struct {
	AccountNumber                  string                              `json:"accountNumber" firestore:"accountNumber"`
	CustomerID                     *string                             `json:"customerId" firestore:"customerId"`
	Name                           string                              `json:"name" firestore:"name"`
	FriendlyName                   string                              `json:"friendlyName" firestore:"friendlyName"`
	RoleARN                        string                              `json:"roleARN" firestore:"roleARN"`
	TenancyType                    string                              `json:"tenancyType" firestore:"tenancyType"`
	FlexSaveAllowed                bool                                `json:"flexSaveAllowed" firestore:"flexSaveAllowed"`
	FlexSaveRecalculationStartDate *time.Time                          `json:"flexSaveRecalculationStartDate" firestore:"flexSaveRecalculationStartDate"`
	DefaultAwsFlexSaveDiscountRate float64                             `json:"defaultAwsFlexSaveDiscountRate" firestore:"defaultAwsFlexSaveDiscountRate"`
	Regions                        []string                            `json:"regions" firestore:"regions"`
	credentials                    map[string]*credentials.Credentials `json:"-" firestore:"-"`
	Support                        *SupportConfiguration               `json:"support" firestore:"support"`
	Path                           *string                             `json:"cur_path" firestore:"cur_path"`
	Bucket                         *string                             `json:"cur_bucket" firestore:"cur_bucket"`
	LastUpdated                    *time.Time                          `json:"lastUpdated" firestore:"lastUpdated"`
	CreatedAt                      *time.Time                          `json:"createdAt" firestore:"createdAt"`
	RequestedBy                    *string                             `json:"requestedBy" firestore:"requestedBy"`
	Status                         MasterPayerAccountStatus            `json:"status" firestore:"status"`
	Features                       *Features                           `json:"features" firestore:"features"`
	RootEmail                      *string                             `json:"rootEmail" firestore:"rootEmail"`
	TimeCreated                    *time.Time                          `json:"timeCreated" firestore:"timeCreated"`
	OnboardingDate                 *time.Time                          `json:"expectedOnboardingDate" firestore:"expectedOnboardingDate"`
	Domain                         string                              `json:"domain" firestore:"domain"`
}

type Features struct {
	NRA                bool       `json:"no-root-access" firestore:"no-root-access"`
	OrganizationImport bool       `json:"import-org" firestore:"import-org"`
	EdpPpa             bool       `json:"edp-ppa" firestore:"edp-ppa"`
	NoImportOrgReason  string     `json:"no-import-org-reason" firestore:"no-import-org-reason"`
	BillingStartDate   *time.Time `json:"org-billing-start-month" firestore:"org-billing-start-month"`
}

type SupportConfiguration struct {
	Tier  string  `json:"tier" firestore:"tier"`
	Model *string `json:"model" firestore:"model"`
}

func (pa *MasterPayerAccount) IsSharedPayer() bool {
	return pa.TenancyType == "shared"
}

func (pa *MasterPayerAccount) IsDedicatedPayer() bool {
	return pa.TenancyType == "dedicated"
}

// IsFlexsaveAllowed Check if this account is on the list of the allowed
// accounts for using FlexSave.
// Note that FlexSave support is not tied to it being shared payer account, as
// we allow certain dedicated payer as well.
// Not all accounts in the platform are connected to our payer accounts, which is
// fine, they are just not supported.
func (pa *MasterPayerAccounts) IsFlexsaveAllowed(accountID string) bool {
	if mpa, ok := pa.Accounts[accountID]; ok {
		return mpa.FlexSaveAllowed
	}

	return false
}

// GetTenancyType returns the tenancy type for a master payer account with the given account ID
func (pa *MasterPayerAccounts) GetTenancyType(accountID string) string {
	if mpa, ok := pa.Accounts[accountID]; ok {
		return mpa.TenancyType
	}

	return ""
}

func (pa *MasterPayerAccounts) IsRetiredMPA(accountID string) bool {
	if mpa, ok := pa.Accounts[accountID]; ok {
		return mpa.Status == MasterPayerAccountStatusRetired
	}

	return false
}

func (pa *MasterPayerAccount) newCredentials(roleSessionName string) error {
	if pa.RoleARN == "" {
		return errors.New("master payer account missing RoleARN")
	}

	creds, err := awsproxy.NewCredentials()
	if err != nil {
		return err
	}

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(endpoints.UsEast1RegionID),
		Credentials: credentials.NewStaticCredentials(
			*creds.AccessKeyId,
			*creds.SecretAccessKey,
			*creds.SessionToken,
		),
	})
	if err != nil {
		return err
	}

	if roleSessionName == "" {
		roleSessionName = "doitintl-cmp"
	}

	assumeRoleCreds := stscreds.NewCredentials(sess, pa.RoleARN, func(arp *stscreds.AssumeRoleProvider) {
		arp.RoleSessionName = roleSessionName
		arp.Duration = 60 * time.Minute
	})

	if pa.credentials == nil {
		pa.credentials = make(map[string]*credentials.Credentials)
	}

	pa.credentials[roleSessionName] = assumeRoleCreds

	return nil
}

// NewCredentials will return credentials per roleSessionName. If no credentials or
// credentials expired it will generate new credentials using proxyAccount credentials
func (pa *MasterPayerAccount) NewCredentials(roleSessionName string) (*credentials.Credentials, error) {
	if roleSessionName == "" {
		roleSessionName = "doitintl-cmp"
	}

	if pa.credentials == nil || pa.credentials[roleSessionName] == nil || pa.credentials[roleSessionName].IsExpired() {
		if err := pa.newCredentials(roleSessionName); err != nil {
			return nil, err
		}
	}

	return pa.credentials[roleSessionName], nil
}

func trimAndLowerCase(orig string) string {
	return strings.ToLower(strings.TrimSpace(orig))
}

func RegionsArrayContainsAllValue(regions []string) bool {
	for _, region := range regions {
		if trimAndLowerCase(region) == "all" {
			return true
		}
	}

	return false
}

func deduplicateRegions(srcArr []string) []string {
	tempSet := make(map[string]struct{})

	for _, each := range srcArr {
		tempSet[each] = struct{}{}
	}

	destArr := make([]string, 0, len(tempSet))

	for each := range tempSet {
		destArr = append(destArr, each)
	}

	return destArr
}

func (pa *MasterPayerAccount) GetMasterPayerAccountRegions() []string {
	var regions []string
	if pa.TenancyType == "shared" {
		regions = append(pa.Regions, SharedMasterPayerAccountRegions...)
	} else {
		regions = append(pa.Regions, DedicatedMasterPayerAccountRegions...)
	}

	return deduplicateRegions(regions)
}

// IsValidRegion checks if given region is valid for MPA id
func (pa *MasterPayerAccounts) IsValidRegion(accountID string, region string) bool {
	if mpa, ok := pa.Accounts[accountID]; ok {
		return mpa.IsValidRegion(region)
	}

	return false
}

// IsValidRegion checks if given region is valid for MPA
func (pa *MasterPayerAccount) IsValidRegion(region string) bool {
	var regions []string

	if RegionsArrayContainsAllValue(pa.Regions) {
		regions = AllAwsRegions
	} else {
		regions = pa.GetMasterPayerAccountRegions()
	}

	regionToCheck := trimAndLowerCase(region)

	for _, region := range regions {
		if region == regionToCheck {
			return true
		}
	}

	return false
}
