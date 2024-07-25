package partnersales

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"cloud.google.com/go/channel/apiv1/channelpb"
	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/googlecloud"
	"github.com/doitintl/hello/scheduled-tasks/slice"
)

// CreateBillingAccountRequestData - provided by create billing account http request
type CreateBillingAccountRequestData struct {
	CustomerID     string
	EntityID       string
	Email          string
	ReqBody        *googlecloud.CreateBillingAccountBody
	bindingMembers []string
	displayName    string
}

// ResponseData - returned by create billing account http request
type ResponseData struct {
	DisplayName string `json:"displayName"`
	ID          string `json:"id"`
}

// GCP billing roles
const (
	RoleBillingViewer = "roles/billing.viewer"
	RoleBillingUser   = "roles/billing.user"
	RoleBillingAdmin  = "roles/billing.admin"
)

type BindingsMember string

const (
	userMember   = "user"
	groupMember  = "group"
	domainMember = "domain"
)

const (
	googleCloudPlatform  = "google_cloud_platform"
	displayNameParamName = "display_name"
)

var (
	ErrInternalServerError = errors.New("internal server error")
	ErrBadRequest          = errors.New("bad request")
	ErrUnknownArgument     = errors.New("unkown argument")
)

var gcpNewBillingAccountCCs = []string{"adamkh@google.com", "gcp-ops@doit.com"}

func newMember(identifier string, memberType string) BindingsMember {
	return BindingsMember(memberType + ":" + identifier)
}

// CreateBillingAccount creates new billing account in Partner Sales Console using Channel API
// Input: create billing account request data
// Output: new account (asset) display name and ID
func (s *GoogleChannelService) CreateBillingAccount(ctx *gin.Context, reqData *CreateBillingAccountRequestData) (*ResponseData, error) {
	l := s.loggerProvider(ctx)

	l.Info("ChannelServices - Create Billing Account")

	channelBillingAccount, err := s.newChannelBillingAccount()
	if err != nil {
		l.Errorf("failed creating new channel billing account")
		return nil, ErrInternalServerError
	}

	if valid, err := channelBillingAccount.validateRequest(ctx, reqData); err != nil {
		l.Errorf("failed validating input")
		return nil, err
	} else if !valid {
		return nil, ErrBadRequest
	}

	channelCustomer, err := s.CreateCustomer(ctx, reqData.CustomerID)
	if err != nil {
		l.Errorf("Failed creating new channel customer for %s: %s", reqData.CustomerID, err)
		return nil, ErrInternalServerError
	}

	channelBillingAccount.channelCustomer = channelCustomer

	offer, err := s.selectGCPOffer(ctx)
	if err != nil {
		l.Errorf("Failed obtaining GCP offer: %s", err)
		return nil, ErrInternalServerError
	}

	channelBillingAccount.offer = offer

	if err := channelBillingAccount.createEntitlement(ctx, reqData); err != nil {
		l.Errorf("Failed creating entitlement: %s", err)
		return nil, ErrInternalServerError
	}

	if err := channelBillingAccount.setIamPolicy(ctx, reqData); err != nil {
		l.Errorf("Failed creating IAM policy: %s", err)
		return nil, ErrInternalServerError
	}

	// try to set primary domain with billing account user
	// this might fail, but we do not abort the request in this case
	channelBillingAccount.addDomainToIamPolicy(ctx, reqData)

	if err := channelBillingAccount.createAsset(ctx, reqData); err != nil {
		l.Errorf("Failed creating asset: %s", err)
		return nil, ErrInternalServerError
	}

	channelBillingAccount.sendNotifications(ctx, reqData)

	respData := &ResponseData{
		ID:          channelBillingAccount.getBillingAccountID(),
		DisplayName: reqData.displayName,
	}

	return respData, nil
}

func (s *GoogleChannelService) selectGCPOffer(ctx context.Context) (*channelpb.Offer, error) {
	l := s.loggerProvider(ctx)
	client := s.cloudChannel

	l.Info("Selecting GCP Offer")

	var selectedOffer *channelpb.Offer

	it := client.ListOffers(ctx, &channelpb.ListOffersRequest{
		Parent: partnerAccountName,
		Filter: "sku.name=products/UkRBE2yBJ0Q9K5/skus/UR6k9lRk4vy4XR",
	})

	for {
		offer, err := it.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return nil, err
		}

		if strings.Contains(offer.GetSku().GetMarketingInfo().GetDisplayName(), "Google Cloud") {
			selectedOffer = offer
			break
		}
	}

	if selectedOffer == nil {
		return nil, ErrFailedToFetchGCPOffer
	}

	l.Info("GCP offer")
	l.Info(protojson.Format(selectedOffer))

	return selectedOffer, nil
}

type channelBillingAccount struct {
	service         *GoogleChannelService
	customer        *common.Customer
	channelCustomer *channelpb.Customer
	offer           *channelpb.Offer
	entitlement     *channelpb.Entitlement
	policy          *cloudbilling.Policy
}

func (s *GoogleChannelService) newChannelBillingAccount(args ...interface{}) (*channelBillingAccount, error) {
	ba := channelBillingAccount{
		service: s,
	}

	for _, arg := range args {
		switch t := arg.(type) {
		case *common.Customer:
			ba.customer = t
		case *channelpb.Customer:
			ba.channelCustomer = t
		case *channelpb.Offer:
			ba.offer = t
		case *channelpb.Entitlement:
			ba.entitlement = t
		case *cloudbilling.Policy:
			ba.policy = t
		default:
			return nil, ErrUnknownArgument
		}
	}

	return &ba, nil
}

// ToGCPBillingAccount converts channel API entitelment object to gcp billing account object
func (ba *channelBillingAccount) ToGCPBillingAccount() (*googlecloud.BillingAccount, error) {
	entitlement := ba.entitlement

	var displayName string

	for _, p := range entitlement.GetParameters() {
		if p.GetName() == displayNameParamName {
			displayName = p.GetValue().GetStringValue()
		}
	}

	if displayName == "" {
		return nil, ErrDisplayNameMissing
	}

	return &googlecloud.BillingAccount{
		DisplayName: displayName,
		Name:        ba.getFullBillingAccountID(),
		ID:          ba.getBillingAccountID(),
	}, nil
}

// validateRequest validates the requester is allowed to create a GCP billing account
// and that the request body is legal.
func (ba *channelBillingAccount) validateRequest(ctx *gin.Context, reqData *CreateBillingAccountRequestData) (bool, error) {
	l := ba.service.loggerProvider(ctx)
	fs := ba.service.conn.Firestore(ctx)

	// account name has prefix doit.
	// account admin(s) was not provided
	if strings.HasPrefix(reqData.ReqBody.Name, "doit.") {
		return false, ErrBadRequest
	}

	customerRef := fs.Collection("customers").Doc(reqData.CustomerID)
	entityRef := fs.Collection("entities").Doc(reqData.EntityID)

	refs := []*firestore.DocumentRef{
		customerRef,
		entityRef,
	}

	// fetch provided customer and billing profile data
	docSnaps, err := fs.GetAll(ctx, refs)
	if err != nil || len(docSnaps) != 2 {
		l.Errorf("Failed reading customer & entity data: %s", err)
		return false, ErrInternalServerError
	}

	var customer common.Customer

	var entity common.Entity

	if err := docSnaps[0].DataTo(&customer); err != nil {
		l.Errorf("Failed reading customer data: %s", err)
		return false, ErrInternalServerError
	}

	if err := docSnaps[1].DataTo(&entity); err != nil {
		l.Errorf("Failed reading entity data: %s", err)
		return false, ErrInternalServerError
	}
	// billing profile is not active or belongs to different customer
	if !entity.Active || entity.Customer.ID != docSnaps[0].Ref.ID {
		return false, errors.New("invalid billing profile")
	}

	var displayName string
	if reqData.ReqBody.Name != "" {
		displayName = strings.ToLower(fmt.Sprintf("doit.%s.%s", reqData.ReqBody.Name, customer.PrimaryDomain))
	} else {
		displayName = strings.ToLower(fmt.Sprintf("doit.%s", customer.PrimaryDomain))
	}

	// account display name contains customer domain or has invalid format
	if strings.Contains(reqData.ReqBody.Name, customer.PrimaryDomain) || !NamePattern.MatchString(displayName) {
		return false, errors.New("invalid billing account display name")
	}

	reqData.displayName = displayName

	members, err := ba.validateAdmins(ctx, reqData.ReqBody.Admins)
	if err != nil {
		return false, err
	}

	reqData.bindingMembers = members

	ba.customer = &customer

	return true, nil
}

func (ba *channelBillingAccount) createEntitlement(ctx context.Context, reqData *CreateBillingAccountRequestData) error {
	l := ba.service.loggerProvider(ctx)
	client := ba.service.cloudChannel

	l.Info("Creating entitlement")

	request := &channelpb.CreateEntitlementRequest{
		Parent: ba.channelCustomer.Name,
		Entitlement: &channelpb.Entitlement{
			Offer: ba.offer.Name,
			Parameters: []*channelpb.Parameter{
				{
					Name: displayNameParamName,
					Value: &channelpb.Value{
						Kind: &channelpb.Value_StringValue{StringValue: reqData.displayName},
					},
				},
			},
			PurchaseOrderId: reqData.CustomerID + "-" + reqData.EntityID,
		},
	}

	op, err := client.CreateEntitlement(ctx, request)
	if err != nil {
		return err
	}

	entitlement, err := op.Wait(ctx)
	if err != nil {
		return err
	}

	l.Info("Created entitlement")
	l.Info(protojson.Format(entitlement))

	ba.entitlement = entitlement

	return nil
}

func (ba *channelBillingAccount) setIamPolicy(ctx context.Context, reqData *CreateBillingAccountRequestData) error {
	l := ba.service.loggerProvider(ctx)
	client := ba.service.cloudBilling

	l.Info("Setting IAM Policy")

	billingAccountID := ba.getFullBillingAccountID()

	policy, err := client.BillingAccounts.SetIamPolicy(billingAccountID, &cloudbilling.SetIamPolicyRequest{
		Policy: &cloudbilling.Policy{
			Bindings: []*cloudbilling.Binding{
				{
					Role:    RoleBillingAdmin,
					Members: reqData.bindingMembers,
				},
			}},
	}).Do()
	if err != nil {
		if herr, ok := err.(*googleapi.Error); ok {
			l.Warning(herr.Body)
		} else {
			l.Warning(err)
		}

		return err
	}

	l.Info("Created policy")
	l.Info(policy)

	ba.policy = policy

	return nil
}

func (ba *channelBillingAccount) addDomainToIamPolicy(ctx context.Context, reqData *CreateBillingAccountRequestData) {
	l := ba.service.loggerProvider(ctx)
	client := ba.service.cloudBilling

	l.Info("adding domain to IAM policy")

	isContractSigned, err := common.IsThereActiveSignedContract(ctx, reqData.CustomerID, reqData.EntityID, []string{"google-cloud", "looker", "google-geolocation-services"})
	if err != nil {
		l.Error(err)
		return
	}

	bindings := make([]*cloudbilling.Binding, len(ba.policy.Bindings))
	copy(bindings, ba.policy.Bindings)

	noDomainIsListed := true

	for _, binding := range ba.policy.Bindings {
		for _, member := range binding.Members {
			if strings.HasPrefix(member, "domain:") {
				noDomainIsListed = false
				break
			}
		}

		binding.Role = RoleBillingUser
	}

	if isContractSigned && noDomainIsListed {
		bindings = append(bindings, &cloudbilling.Binding{
			Role:    RoleBillingUser,
			Members: []string{string(newMember(ba.customer.PrimaryDomain, domainMember))},
		})
	}

	if v, err := client.BillingAccounts.SetIamPolicy(ba.getFullBillingAccountID(), &cloudbilling.SetIamPolicyRequest{
		Policy: &cloudbilling.Policy{Bindings: bindings, Etag: ba.policy.Etag},
	}).Do(); err != nil {
		if herr, ok := err.(*googleapi.Error); ok {
			l.Warning(herr.Body)
		} else {
			l.Warning(err)
		}
	} else {
		l.Info("domain added to IAM policy")

		ba.policy = v
	}
}

func (ba *channelBillingAccount) createAsset(ctx context.Context, reqData *CreateBillingAccountRequestData) error {
	l := ba.service.loggerProvider(ctx)
	fs := ba.service.conn.Firestore(ctx)

	l.Info("Creating asset")

	billingAccountID := ba.getBillingAccountID()

	docID := fmt.Sprintf("%s-%s", common.Assets.GoogleCloud, billingAccountID)
	assetRef := fs.Collection("assets").Doc(docID)
	assetSettingsRef := fs.Collection("assetSettings").Doc(docID)

	admins := make([]string, 0)

	for _, binding := range ba.policy.Bindings {
		if binding.Role == RoleBillingAdmin {
			for _, member := range binding.Members {
				if strings.HasPrefix(member, "user:") || strings.HasPrefix(member, "group:") {
					admins = append(admins, member)
				}
			}
		}
	}

	customerRef := fs.Collection("customers").Doc(reqData.CustomerID)
	entityRef := fs.Collection("entities").Doc(reqData.EntityID)

	props := &googlecloud.AssetProperties{
		BillingAccountID: billingAccountID,
		DisplayName:      reqData.displayName,
		Admins:           admins,
		Projects:         []string{},
		Etag:             ba.policy.Etag,
	}

	asset := googlecloud.Asset{
		BaseAsset: common.BaseAsset{
			AssetType: common.Assets.GoogleCloud,
			Customer:  customerRef,
			Entity:    entityRef,
		},
		Properties: props,
	}

	assetSettings := common.AssetSettings{
		BaseAsset: common.BaseAsset{
			AssetType: common.Assets.GoogleCloud,
			Customer:  customerRef,
			Entity:    entityRef,
		},
	}

	batch := fs.Batch()
	batch.Set(assetRef, asset)
	batch.Set(assetSettingsRef, assetSettings)

	if _, err := batch.Commit(ctx); err != nil {
		l.Error(err)
	}

	return nil
}

func (ba *channelBillingAccount) sendNotifications(ctx context.Context, reqData *CreateBillingAccountRequestData) {
	l := ba.service.loggerProvider(ctx)
	fs := ba.service.conn.Firestore(ctx)

	l.Info("Sending notifications")

	billingAccountID := ba.getBillingAccountID()

	ccs := ba.getBillingAccountCCs(ctx, reqData.Email)

	customerRef := fs.Collection("customers").Doc(reqData.CustomerID)

	go func() {
		if err := googlecloud.SendBillingAccountInstructions(
			reqData.CustomerID, billingAccountID, reqData.displayName, reqData.ReqBody.Admins, ccs); err != nil {
			l.Error("Failed sending billing account instruction")
		}
	}()
	go func() {
		if err := googlecloud.SendAccountCreateSlackNotification(
			billingAccountID, reqData.displayName, reqData.Email, customerRef, ba.customer); err != nil {
			l.Error("Failed sending billing account creation Slack notification")
		}
	}()
}

func (ba *channelBillingAccount) getBillingAccountCCs(ctx context.Context, email string) []string {
	l := ba.service.loggerProvider(ctx)

	ccs := []string{email}

	doitManagers, err := common.GetCustomerAccountManagers(ctx, ba.customer, common.AccountManagerCompanyDoit)
	if err != nil {
		l.Errorf("failed to get doit account managers with error %s", err)
	}

	gcpManagers, err := common.GetCustomerAccountManagers(ctx, ba.customer, common.AccountManagerCompanyGcp)
	if err != nil {
		l.Errorf("failed to get gcp account managers with error %s", err)
	}

	if common.Production {
		ccs = append(ccs, gcpNewBillingAccountCCs...)

		for _, am := range append(doitManagers, gcpManagers...) {
			// Email only FSR/AM/SAMs
			if am.Role != common.AccountManagerRoleFSR && am.Role != common.AccountManagerRoleSAM {
				continue
			}

			ccs = append(ccs, am.Email)
		}
	}

	return ccs
}

func (ba *channelBillingAccount) validateAdmins(ctx *gin.Context, admins []string) ([]string, error) {
	l := ba.service.loggerProvider(ctx)

	if len(admins) == 0 {
		return []string{}, nil
	}

	customerID := ctx.Param("customerID")

	fs := ba.service.conn.Firestore(ctx)
	customerRef := fs.Collection("customers").Doc(customerID)

	entityID := ctx.Param("entityID")

	allowedDomains, err := googlecloud.ValidateAllowedDomains(ctx, customerID, entityID, customerRef)
	if err != nil {
		return nil, ctx.AbortWithError(http.StatusInternalServerError, err)
	}

	members, err := ba.validateMembers(ctx, admins, allowedDomains)

	if err != nil {
		if gapiErr, ok := err.(*googleapi.Error); ok {
			l.Error(gapiErr.Body)

			if gapiErr.Code == http.StatusBadRequest {
				match := MissingUserErrorPattern.FindStringSubmatch(gapiErr.Message)
				if len(match) > 1 {
					return nil, fmt.Errorf("%s is not an active user Google Account", match[1])
				}
			}
		} else if strings.HasSuffix(err.Error(), unallowedDomainErrSuffix) {
			ctx.AbortWithStatusJSON(http.StatusBadRequest, err.Error())
		} else {
			l.Error(err)
			return nil, errors.New("invalid billing account admins")
		}
	}

	return members, nil
}

const unallowedDomainErrSuffix = " has an unallowed domain"

func validateEmailDomains(emails []string, allowedDomains []string) error {
	for _, email := range emails {
		lowCasedEmail := strings.ToLower(email)
		if !strings.Contains(lowCasedEmail, "@") || !slice.Contains(allowedDomains, strings.Split(lowCasedEmail, "@")[1]) {
			return errors.New("email:" + email + unallowedDomainErrSuffix)
		}
	}

	return nil
}

func (ba *channelBillingAccount) validateMembers(ctx context.Context, emails []string, allowedDomains []string) ([]string, error) {
	defer ba.resetValidationsBillingAccount(ctx)

	if err := validateEmailDomains(emails, allowedDomains); err != nil {
		return nil, err
	}

	switch len(emails) {
	case 1:
		// if only 1 email, try to add it as a group
		if members, err := ba.validateEmails(emails, groupMember); err == nil {
			return members, nil
		}

		fallthrough
	default:
		members, err := ba.validateEmails(emails, userMember)
		if err != nil {
			return nil, err
		}

		return members, nil
	}
}

func (ba *channelBillingAccount) validateEmails(emails []string, memberType string) ([]string, error) {
	client := ba.service.cloudBilling

	members := make([]string, len(emails))
	for i, email := range emails {
		members[i] = string(newMember(email, memberType))
	}

	_, err := client.BillingAccounts.SetIamPolicy(googlecloud.ValidationsBillingAccountResource,
		&cloudbilling.SetIamPolicyRequest{
			Policy: &cloudbilling.Policy{
				Bindings: []*cloudbilling.Binding{
					{
						Role:    RoleBillingViewer,
						Members: members,
					},
				},
			},
		}).Do()

	return members, err
}

func (ba *channelBillingAccount) resetValidationsBillingAccount(ctx context.Context) {
	ba.service.cloudBilling.BillingAccounts.SetIamPolicy(googlecloud.ValidationsBillingAccountResource,
		&cloudbilling.SetIamPolicyRequest{
			Policy: &cloudbilling.Policy{
				Bindings: []*cloudbilling.Binding{},
			},
		}).Do()
}

func (ba *channelBillingAccount) getFullBillingAccountID() string {
	return ba.entitlement.ProvisionedService.ProvisioningId
}

func (ba *channelBillingAccount) getBillingAccountID() string {
	fullBillingAccountID := ba.getFullBillingAccountID()
	splitID := strings.Split(fullBillingAccountID, "/")

	return splitID[len(splitID)-1]
}
