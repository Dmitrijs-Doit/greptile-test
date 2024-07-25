package googlecloud

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/gin-gonic/gin"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"google.golang.org/api/cloudbilling/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/mailer"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
	"github.com/doitintl/hello/scheduled-tasks/slice"
	"github.com/doitintl/retry"
)

type BindingsMember string

type CreateBillingAccountBody struct {
	Name   string   `json:"name"`
	Admins []string `json:"admins"`
}

type SendBillingAccountInstructionsBody struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
}

type SetBillingAccountAdminsBody struct {
	AddMemebers    []string `json:"add_members"`
	RemoveMemebers []string `json:"remove_members"`
}

// GCP billing roles
const (
	RoleBillingViewer = "roles/billing.viewer"
	RoleBillingUser   = "roles/billing.user"
	RoleBillingAdmin  = "roles/billing.admin"
)

const (
	slackChannel        = "#sales-ops"
	googleCloudPlatform = "google_cloud_platform"
)

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

func validateMembers(cb *cloudbilling.APIService, emails []string, allowedDomains []string) ([]string, error) {
	defer resetValidationsBillingAccount()

	if err := validateEmailDomains(emails, allowedDomains); err != nil {
		return nil, err
	}

	switch len(emails) {
	case 0:
		return nil, fmt.Errorf("invalid email list of size 0")
	case 1:
		// if only 1 email, try to add it as a group
		members := make([]string, len(emails))
		for i, email := range emails {
			members[i] = string(NewGroupMember(email))
		}

		if _, err := cb.BillingAccounts.SetIamPolicy(ValidationsBillingAccountResource,
			&cloudbilling.SetIamPolicyRequest{
				Policy: &cloudbilling.Policy{
					Bindings: []*cloudbilling.Binding{
						{
							Role:    RoleBillingViewer,
							Members: members,
						},
					},
				},
			}).Do(); err == nil {
			return members, nil
		}

		fallthrough
	default:
		members := make([]string, len(emails))
		for i, email := range emails {
			members[i] = string(NewUserMember(email))
		}

		if _, err := cb.BillingAccounts.SetIamPolicy(ValidationsBillingAccountResource,
			&cloudbilling.SetIamPolicyRequest{
				Policy: &cloudbilling.Policy{
					Bindings: []*cloudbilling.Binding{
						{
							Role:    RoleBillingViewer,
							Members: members,
						},
					},
				},
			}).Do(); err != nil {
			return nil, err
		}

		return members, nil
	}
}

// remove members just from the policy struct, without calling the API
func removeAdminMembers(policy *cloudbilling.Policy, membersToOmit []string) {
	for _, binding := range policy.Bindings {
		if binding.Role == RoleBillingAdmin {
			modifiedMembers := make([]string, 0)

			for _, member := range binding.Members {
				if !strings.HasPrefix(member, "user:") || slice.FindIndex(membersToOmit, member) == -1 {
					modifiedMembers = append(modifiedMembers, member)
				}
			}

			binding.Members = modifiedMembers

			break
		}
	}
}

func getDoitAdminMembers(policy *cloudbilling.Policy) []string {
	doitUserMembers := make([]string, 0)

	for _, binding := range policy.Bindings {
		if binding.Role == RoleBillingAdmin {
			for _, member := range binding.Members {
				memberLower := strings.ToLower(member)
				if strings.HasPrefix(memberLower, "user:") && (strings.HasSuffix(memberLower, "@doit.com") || strings.HasSuffix(memberLower, "@doit-intl.com")) {
					doitUserMembers = append(doitUserMembers, member)
				}
			}
		}
	}

	return doitUserMembers
}

func getCustomerDomains(ctx *gin.Context, customerRef *firestore.DocumentRef) ([]string, error) {
	customerSnap, err := customerRef.Get(ctx)
	if err != nil {
		return nil, err
	}

	customerDomains, err := customerSnap.DataAt("domains")
	if err != nil {
		return nil, err
	}

	retVal := make([]string, 0)

	if customerDomains != nil {
		if customerDomains, ok := customerDomains.([]interface{}); ok {
			for _, domain := range customerDomains {
				retVal = append(retVal, domain.(string))
			}
		}
	}

	return retVal, nil
}

func ValidateAllowedDomains(ctx *gin.Context, customerID string, entityID string, customerRef *firestore.DocumentRef) ([]string, error) {
	isContractSigned, err := common.IsThereActiveSignedContract(ctx, customerID, entityID, []string{"google-cloud", "looker", "google-geolocation-services"})
	if err != nil {
		return nil, ctx.AbortWithError(http.StatusInternalServerError, err)
	}

	allowedDomains := []string{}

	if !isContractSigned {
		allowedDomains = append(allowedDomains, "doit.com", "doit-intl.com")
	} else {
		allowedDomains, err = getCustomerDomains(ctx, customerRef)
		if err != nil {
			return nil, ctx.AbortWithError(http.StatusInternalServerError, err)
		}
	}

	return allowedDomains, nil
}

func SetBillingAccountAdmins(ctx *gin.Context) {
	l := logger.FromContext(ctx)

	var body SetBillingAccountAdminsBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	l.Info(body)

	customerID, billingAccountID, entityID := ctx.Param("customerID"), ctx.Param("billingAccountID"), ctx.Param("entityID")
	email := ctx.GetString("email")

	isContractSigned, err := common.IsThereActiveSignedContract(ctx, customerID, entityID, []string{"google-cloud", "looker", "google-geolocation-services"})
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	secret, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretCloudBilling)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	creds := option.WithCredentialsJSON(secret)

	cb, err := cloudbilling.NewService(ctx, creds)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	fs := common.GetFirestoreClient(ctx)

	docID := fmt.Sprintf("%s-%s", common.Assets.GoogleCloud, billingAccountID)
	assetRef := fs.Collection("assets").Doc(docID)

	docSnap, err := assetRef.Get(ctx)
	if err != nil {
		ctx.AbortWithError(http.StatusUnauthorized, err)
		return
	}

	var asset Asset
	if err := docSnap.DataTo(&asset); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if asset.Customer == nil || asset.Customer.ID != customerID {
		ctx.AbortWithStatus(http.StatusForbidden)
		return
	}

	resource := fmt.Sprintf("billingAccounts/%s", asset.Properties.BillingAccountID)

	policy, err := cb.BillingAccounts.GetIamPolicy(resource).Do()
	if err != nil {
		ctx.AbortWithError(http.StatusUnauthorized, err)
		return
	}

	l.Info(policy)

	if policy.Etag != asset.Properties.Etag {
		admins := make([]string, 0)

		for _, binding := range policy.Bindings {
			if binding.Role == RoleBillingAdmin {
				admins = binding.Members
			}
		}

		_, err = assetRef.Update(ctx, []firestore.Update{
			{FieldPath: []string{"properties", "admins"}, Value: admins},
			{FieldPath: []string{"properties", "etag"}, Value: policy.Etag},
		})

		ctx.AbortWithStatus(http.StatusConflict)

		return
	}

	if !ctx.GetBool(common.CtxKeys.DoitEmployee) {
		authz := false

		for _, binding := range policy.Bindings {
			if binding.Role == RoleBillingAdmin {
				for _, member := range binding.Members {
					if strings.HasPrefix(member, "user:") {
						memberEmail := strings.ToLower(member[5:])
						if memberEmail == email {
							authz = true
							break
						}
					}
				}
			}
		}

		if !authz {
			ctx.AbortWithStatusJSON(http.StatusForbidden, "You must be a billing account admin")
			return
		}
	}

	// Remove members
	if len(body.RemoveMemebers) > 0 {
		removeAdminMembers(policy, body.RemoveMemebers)
	}

	customerRef := fs.Collection("customers").Doc(customerID)

	customer, err := common.GetCustomer(ctx, customerRef)
	if err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	// Add members
	if len(body.AddMemebers) > 0 {
		allowedDomains, err := ValidateAllowedDomains(ctx, customerID, asset.Entity.ID, asset.Customer)
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		members, err := validateMembers(cb, body.AddMemebers, allowedDomains)

		if err != nil {
			if gapiErr, ok := err.(*googleapi.Error); ok {
				l.Error(gapiErr.Body)

				if gapiErr.Code == http.StatusBadRequest {
					errorPattern := regexp.MustCompile("^User ([^\\s]+) does not exist.$")

					match := errorPattern.FindStringSubmatch(gapiErr.Message)
					if len(match) > 1 {
						ctx.AbortWithStatusJSON(gapiErr.Code, fmt.Sprintf("%s is not an active user Google Account", match[1]))
						return
					}
				}

				ctx.AbortWithStatusJSON(gapiErr.Code, gapiErr.Message)
			} else if strings.HasSuffix(err.Error(), unallowedDomainErrSuffix) {
				ctx.AbortWithStatusJSON(http.StatusBadRequest, err.Error())
			} else {
				l.Error(err)
				ctx.AbortWithError(http.StatusInternalServerError, err)
			}

			return
		}

		policy.Bindings = append(policy.Bindings, &cloudbilling.Binding{
			Role:    RoleBillingAdmin,
			Members: members,
		})
	}

	_, code, err := UpdatePolicy(ctx, resource, policy, l, assetRef, cb)
	if err != nil {
		ctx.AbortWithError(code, err)
		return
	}

	// try to add domain to the policy but without failing the reqeust
	if isContractSigned {
		backoffDuration, err := time.ParseDuration("300ms")
		if err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if err := retry.BackOffDelay(func() error {
			policyAfterUpdate, err := cb.BillingAccounts.GetIamPolicy(resource).Do()
			if err != nil {
				l.Error(err)
				return err
			}

			policyAfterUpdate.Bindings = append(policyAfterUpdate.Bindings, &cloudbilling.Binding{
				Role:    RoleBillingUser,
				Members: []string{string(NewDomainMember(customer.PrimaryDomain))},
			})

			_, _, err = UpdatePolicy(ctx, resource, policyAfterUpdate, l, assetRef, cb)
			if err != nil {
				l.Error(err)
				return err
			}
			return nil
		}, 5, backoffDuration); err != nil {
			ctx.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}
}

func UpdatePolicy(
	ctx context.Context,
	resource string,
	policy *cloudbilling.Policy,
	l logger.ILogger,
	assetRef *firestore.DocumentRef,
	cb *cloudbilling.APIService,
) (*cloudbilling.Policy, int, error) {
	updatedPolicy, err := cb.BillingAccounts.SetIamPolicy(resource, &cloudbilling.SetIamPolicyRequest{
		Policy: &cloudbilling.Policy{Bindings: policy.Bindings, Etag: policy.Etag},
	}).Do()
	if err != nil {
		if gapiErr, ok := err.(*googleapi.Error); ok {
			l.Error(gapiErr.Body)

			if gapiErr.Code == http.StatusBadRequest {
				errorPattern := regexp.MustCompile("^User ([^\\s]+) does not exist.$")

				match := errorPattern.FindStringSubmatch(gapiErr.Message)
				if len(match) > 1 {
					return nil, gapiErr.Code, fmt.Errorf("%s is not an active user Google Account", match[1])
				}
			}

			return nil, gapiErr.Code, gapiErr
		} else {
			l.Error(err)
			return nil, http.StatusInternalServerError, err
		}
	}

	l.Info(updatedPolicy)

	admins := make([]string, 0)

	for _, binding := range updatedPolicy.Bindings {
		if binding.Role == RoleBillingAdmin {
			admins = binding.Members
		}
	}

	_, _ = assetRef.Update(ctx, []firestore.Update{
		{FieldPath: []string{"properties", "admins"}, Value: admins},
		{FieldPath: []string{"properties", "etag"}, Value: updatedPolicy.Etag},
	})

	return updatedPolicy, http.StatusOK, nil
}

func SendBillingAccountInstructionsHandler(ctx *gin.Context) {
	l := logger.FromContext(ctx)

	var body SendBillingAccountInstructionsBody
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	l.Info(body)

	if body.Email == "" {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	customerID := ctx.Param("customerID")
	billingAccountID := ctx.Param("billingAccountID")

	if !ctx.GetBool("doitEmployee") {
		ctx.AbortWithStatus(http.StatusForbidden)
		return
	}

	emails := []string{body.Email}
	if err := SendBillingAccountInstructions(customerID, billingAccountID, body.DisplayName, emails, nil); err != nil {
		ctx.AbortWithError(http.StatusInternalServerError, err)
		return
	}
}

func NewUserMember(email string) BindingsMember {
	return BindingsMember("user:" + email)
}

func NewGroupMember(email string) BindingsMember {
	return BindingsMember("group:" + email)
}

func NewDomainMember(domain string) BindingsMember {
	return BindingsMember("domain:" + domain)
}

func resetValidationsBillingAccount() {
	ctx := context.Background()

	secret, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretCloudBilling)
	if err != nil {
		return
	}

	creds := option.WithCredentialsJSON(secret)

	cb, err := cloudbilling.NewService(ctx, creds)
	if err != nil {
		return
	}

	cb.BillingAccounts.SetIamPolicy(ValidationsBillingAccountResource,
		&cloudbilling.SetIamPolicyRequest{
			Policy: &cloudbilling.Policy{
				Bindings: []*cloudbilling.Binding{},
			},
		}).Do()
}

func SendBillingAccountInstructions(customerID, billingAccountID, displayName string, emails []string, ccs []string) error {
	if len(emails) == 0 {
		return nil
	}

	data := struct {
		DisplayName string `json:"displayName"`
		ID          string `json:"id"`
	}{
		DisplayName: displayName,
		ID:          billingAccountID,
	}

	m := mail.NewV3Mail()
	m.SetTemplateID(mailer.Config.DynamicTemplates.GoogleCloudNewBillingAccount)
	m.SetFrom(mail.NewEmail(mailer.Config.NoReplyName, mailer.Config.NoReplyEmail))
	m.SetTrackingSettings(&mail.TrackingSettings{SubscriptionTracking: &mail.SubscriptionTrackingSetting{Enable: common.Bool(false)}})

	// Sendgrid will fail if you use the same email more than once
	// Also make sure we don't send email to the same user twice
	uniqueEmails := make(map[string]bool)

	for i, email := range emails {
		if _, ok := uniqueEmails[email]; email == "" || ok {
			continue
		}

		uniqueEmails[email] = true

		personalization := mail.NewPersonalization()
		personalization.AddTos(mail.NewEmail("", email))
		personalization.SetDynamicTemplateData("email", email)
		personalization.SetDynamicTemplateData("data", data)
		personalization.SetDynamicTemplateData("customerId", customerID)

		// Adds CCs (google and doit account managers), only on production
		if i == 0 && common.Production {
			for _, cc := range ccs {
				if _, ok := uniqueEmails[cc]; !ok {
					personalization.AddCCs(mail.NewEmail("", cc))

					uniqueEmails[cc] = true
				}
			}
		}

		m.AddPersonalizations(personalization)
	}

	request := sendgrid.GetRequest(mailer.Config.APIKey, mailer.Config.MailSendPath, mailer.Config.BaseURL)
	request.Method = "POST"
	request.Body = mail.GetRequestBody(m)

	if _, err := sendgrid.MakeRequestRetry(request); err != nil {
		return err
	}

	return nil
}

func SendAccountCreateSlackNotification(billingAccountID, displayName string, email string, customerRef *firestore.DocumentRef, customer *common.Customer) error {
	ctx := context.Background()
	message := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"ts":          time.Now().Unix(),
				"color":       "#4CAF50",
				"author_name": fmt.Sprintf("<mailto:%s|%s>", email, email),
				"title":       "Google Cloud Billing Account Created",
				"title_link":  fmt.Sprintf("https://console.doit.com/customers/%s/assets/google-cloud/google-cloud-%s", customerRef.ID, billingAccountID),
				"thumb_url":   "https://storage.googleapis.com/hello-static-assets/logos/google-cloud.png",
				"fields": []map[string]interface{}{
					{
						"title": "Customer",
						"value": fmt.Sprintf("<https://console.doit.com/customers/%s|%s>", customerRef.ID, customer.Name),
						"short": false,
					},
					{
						"title": "Billing Account ID",
						"value": billingAccountID,
						"short": false,
					},
					{
						"title": "Display Name",
						"value": displayName,
						"short": false,
					},
				},
			},
		},
	}

	if _, err := common.PublishToSlack(ctx, message, slackChannel); err != nil {
		return err
	}

	return nil
}
