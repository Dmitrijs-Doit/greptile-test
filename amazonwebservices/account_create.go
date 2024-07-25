package amazonwebservices

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/organizations"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/dal"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/doitintl/hello/scheduled-tasks/assets/pkg"
	"github.com/doitintl/hello/scheduled-tasks/common"
	log "github.com/doitintl/hello/scheduled-tasks/logger"
	"github.com/doitintl/hello/scheduled-tasks/mailer"
	"github.com/doitintl/hello/scheduled-tasks/zerobounce"
)

type CreateAccountBody struct {
	Email                     string `json:"email"`
	Name                      string `json:"name"`
	PayerAccountID            string `json:"payerAccountId"`
	DeleteAwsOrganizationRole bool   `json:"deleteAwsOrganizationRole"`
}

const (
	EmailIsNotValid string = "the provided email is not valid"
)

var (
	ErrAccountAlreadyExist  = errors.New("an account already exists for this email")
	ErrEmailIsNotValid      = errors.New(EmailIsNotValid)
	ErrCustomerIDNotMatch   = errors.New("customer id not match")
	ErrPayerAccountNotValid = errors.New("payer account not valid")
	ErrCreateAccountFailed  = errors.New("create account operation failed")
)

const assumeRolePolicyDocument = `{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::454464851268:root"
      },
      "Action": "sts:AssumeRole",
      "Condition": {
        "StringEquals": {
          "sts:ExternalId": "6cda262029ad7b34a64ff537196ab4"
        }
      }
    }
  ]
}`

func (s *AWSService) CreateAccount(ctx context.Context, customerID, entityID, email string, body *CreateAccountBody) (string, error) {
	l := s.loggerProvider(ctx)

	doitEmployee := s.doitEmployeeService.IsDoitEmployee(ctx)

	// validate with zerobounce
	emailParams := zerobounce.ValidateParams{
		Email:     body.Email,
		IPAddress: "",
	}

	resp, err := s.zb.Validate(&emailParams)
	if err != nil {
		l.Errorf("failed to zerobounce validate email with error: %s", err)
		return "", err
	}

	l.Infof("zerobounce response: %+v", resp)

	// temporary: giveup on the check of resp.FreeEmail to allow creating accounts with @gmail.com addresses
	if valid, addSubStatus := resp.IsValidEmail(true); !valid {
		if addSubStatus {
			return "", fmt.Errorf("%s (%s)", EmailIsNotValid, resp.SubStatus)
		}

		return "", ErrEmailIsNotValid
	}

	accountID, err := s.createAccount(ctx, customerID, entityID, email, body, doitEmployee)
	if err != nil {
		// AWS API Error
		var aerr awserr.Error
		if errors.As(err, &aerr) {
			l.Error(err)

			switch aerr.Code() {
			case organizations.CreateAccountFailureReasonEmailAlreadyExists:
				return "", ErrAccountAlreadyExist
			case organizations.CreateAccountFailureReasonInvalidEmail:
				return "", ErrEmailIsNotValid
			default:
				return "", ErrCreateAccountFailed
			}
		}
		// Other kind of error
		return "", err
	}

	return accountID, nil
}

func (s *AWSService) createAccount(ctx context.Context, customerID, entityID, email string, body *CreateAccountBody, doitEmployee bool) (string, error) {
	fs := s.conn.Firestore(ctx)
	l := s.loggerProvider(ctx)

	customerRef := fs.Collection("customers").Doc(customerID)
	entityRef := fs.Collection("entities").Doc(entityID)

	var customer common.Customer

	var entity common.Entity

	refs := []*firestore.DocumentRef{customerRef, entityRef}

	docs, err := fs.GetAll(ctx, refs)
	if err != nil {
		return "", err
	}

	if err1, err2 := docs[0].DataTo(&customer), docs[1].DataTo(&entity); err1 != nil {
		l.Errorf("failed to get customer with error: %s", err1)
		return "", err1
	} else if err2 != nil {
		l.Errorf("failed to get entity with error: %s", err2)
		return "", err2
	}

	doitManagers, err := common.GetCustomerAccountManagers(ctx, &customer, common.AccountManagerCompanyDoit)
	if err != nil {
		return "", err
	}

	awsManagers, err := common.GetCustomerAccountManagers(ctx, &customer, common.AccountManagerCompanyAws)
	if err != nil {
		return "", err
	}

	ccs := make([]string, 0)

	for _, am := range append(doitManagers, awsManagers...) {
		// Email only FSR/AM/SAMs
		if am.Role != common.AccountManagerRoleFSR && am.Role != common.AccountManagerRoleSAM {
			continue
		}

		ccs = append(ccs, am.Email)
	}

	if customerRef.ID != entity.Customer.ID {
		return "", ErrCustomerIDNotMatch
	}

	payerAccount, err := isValidPayer(ctx, fs, doitEmployee, customerRef, body.PayerAccountID)
	if err != nil {
		return "", err
	}

	if payerAccount == nil {
		return "", ErrPayerAccountNotValid
	}

	masterPayerAccount, err := dal.ToMasterPayerAccount(ctx, payerAccount, fs)
	if err != nil {
		return "", err
	}

	creds, err := masterPayerAccount.NewCredentials("doit-navigator-create-account")
	if err != nil {
		return "", err
	}

	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(endpoints.UsEast1RegionID),
		Credentials: creds,
	})

	if err != nil {
		return "", err
	}

	organizationsSvc := organizations.New(sess)

	createAccountStatus, err := s.createAccountInOrganization(ctx, organizationsSvc, body.Name, body.Email)
	if err != nil {
		return "", err
	}

	// Create account failed, returns the failure reason as error
	if *createAccountStatus.State == organizations.CreateAccountStateFailed {
		err := awserr.New(*createAccountStatus.FailureReason, "Create account in organization failed.", nil)
		return "", err
	}

	l.Infof("create account status: %s", *createAccountStatus.State)
	accountID := *createAccountStatus.AccountId

	describeAccountOutput, err := organizationsSvc.DescribeAccount(&organizations.DescribeAccountInput{
		AccountId: aws.String(accountID),
	})
	if err != nil {
		return "", err
	}

	account := describeAccountOutput.Account

	go func() {
		if err := sendResetPasswordInstructions(account, email, ccs); err != nil {
			l.Errorf("failed to send instructions with error %s", err)
		}

		if err := publishAccountCreatedSlackNotification(account, email, payerAccount, customer, entity); err != nil {
			l.Errorf("failed to send slack notification with error %s", err)
		}
	}()

	if !masterPayerAccount.IsDedicatedPayer() {
		// Create a cloudhealth customer if needed
		if _, err := createCloudhealthResources(ctx, fs, customerRef, entityRef, customer, entity); err != nil {
			l.Warningf("failed to create cloudhealth resources for %s:\n%s", accountID, err.Error())
		}
	}

	if err := createAwsAccountAsset(ctx, fs, account, payerAccount, customerRef, entityRef); err != nil {
		l.Warningf("failed to create aws account asset for  %s:\n%s", accountID, err.Error())
	}

	// Create cloudhealth cross account IAM role
	go func(logger log.ILogger, sess *session.Session, account *organizations.Account) {
		// wait a few seconds after the account is created. for some reason sts AssumeRoles on
		// OrganizationAccountAccessRole fails immediately after account creation, but succeeds
		// after some time.
		time.Sleep(15 * time.Second)

		roleArn := fmt.Sprintf(organizationAccountAccessRoleArnFormat, *account.Id)
		assumeRoleConfig := &aws.Config{
			Region:      aws.String(endpoints.UsEast1RegionID),
			Credentials: stscreds.NewCredentials(sess, roleArn),
		}

		iamSvc := iam.New(sess, assumeRoleConfig)
		if !masterPayerAccount.IsDedicatedPayer() {
			if err := createCHTRole(ctx, iamSvc, account); err != nil {
				logger.Warningf("failed to create cloudhealth cross-account iam role for %s:\n%s", *account.Id, err.Error())
			}
		}

		if !body.DeleteAwsOrganizationRole {
			return
		}

		deleteRoleInput := &iam.DeleteRoleInput{RoleName: aws.String("OrganizationAccountAccessRole")}

		_, err := iamSvc.DeleteRole(deleteRoleInput)
		if err != nil {
			if _, err := iamSvc.DetachRolePolicy(&iam.DetachRolePolicyInput{
				RoleName:  aws.String("OrganizationAccountAccessRole"),
				PolicyArn: aws.String("arn:aws:iam::aws:policy/AdministratorAccess"),
			}); err != nil {
				logger.Warningf("failed to detach 'AdministratorAccess' role policy for %s with error: %s", *account.Id, err)
			}

			_, err := iamSvc.DeleteRole(deleteRoleInput)
			if err != nil {
				logger.Warningf("failed to delete OrganizationAccountAccessRole for %s with error: %s", *account.Id, err)
				return
			}
		}

		logger.Debug("successfully deleted OrganizationAccountAccessRole")
	}(l, sess, describeAccountOutput.Account)

	return accountID, nil
}

// createAwsAccountAsset creates an AWS account asset when a new account is created
// or when an AWS handshake status is changed to "accepted".
// We are manually creating the asset because CHT takes some time until it picks up
// the information of a new account.
func createAwsAccountAsset(
	ctx context.Context,
	fs *firestore.Client,
	account *organizations.Account,
	payerAccount *domain.PayerAccount,
	customerRef, entityRef *firestore.DocumentRef) error {
	if payerAccount == nil {
		return errors.New("payer account must not be nil")
	}

	if account == nil {
		return errors.New("account must not be nil")
	}

	if customerRef == nil || entityRef == nil {
		return errors.New("customer/entity must not be nil")
	}

	props := &pkg.AWSProperties{
		AccountID:    *account.Id,
		Name:         *account.Name,
		FriendlyName: *account.Id,
		OrganizationInfo: &pkg.OrganizationInfo{
			PayerAccount: payerAccount,
			Status:       *account.Status,
			Email:        *account.Email,
		},
		SauronRole: false,
		// It takes a few days until CHT picks up the account
		// we do not have this info until that is done
		CloudHealth: &pkg.CloudHealthAccountInfo{},
	}

	asset := Asset{
		AssetType:  common.Assets.AmazonWebServices,
		Contract:   nil,
		Bucket:     nil,
		Customer:   customerRef,
		Entity:     entityRef,
		Properties: props,
	}

	assetSettings := common.AssetSettings{
		BaseAsset: common.BaseAsset{
			Entity:    entityRef,
			AssetType: common.Assets.AmazonWebServices,
			Customer:  customerRef,
		},
	}

	docID := fmt.Sprintf("%s-%s", common.Assets.AmazonWebServices, *account.Id)
	batch := fs.Batch()
	assetRef := fs.Collection("assets").Doc(docID)
	batch.Set(assetRef, asset)
	batch.Set(assetRef.Collection("assetMetadata").Doc("metadata"), map[string]interface{}{
		// Assets are removed some time after they are stopped being updated.
		// because CHT takes a couple of days to pick up the new accounts,
		// we will allow the new asset to "live" in CMP for at least 7 days
		// by settings the lastUpdated field to 7 days in the future.
		"lastUpdated": time.Now().UTC().AddDate(0, 0, 7),
		"type":        asset.AssetType,
	})
	batch.Set(fs.Collection("assetSettings").Doc(docID), assetSettings)
	_, err := batch.Commit(ctx)

	return err
}

func (s *AWSService) createAccountInOrganization(ctx context.Context, svc *organizations.Organizations, accountName string, email string) (*organizations.CreateAccountStatus, error) {
	logger := s.loggerProvider(ctx)

	var createAccountInput organizations.CreateAccountInput

	createAccountInput.SetAccountName(accountName)
	createAccountInput.SetEmail(email)
	createAccountInput.SetIamUserAccessToBilling(organizations.IAMUserAccessToBillingDeny)

	err := createAccountInput.Validate()
	if err != nil {
		return nil, err
	}

	createAccountOutput, err := svc.CreateAccount(&createAccountInput)
	if err != nil {
		return nil, err
	}

	done := make(chan *organizations.DescribeCreateAccountStatusOutput, 1)

	var describeCreateAccountStatusInput organizations.DescribeCreateAccountStatusInput

	describeCreateAccountStatusInput.SetCreateAccountRequestId(*createAccountOutput.CreateAccountStatus.Id)

	// Wait for operation to complete
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			describeCreateAccountStatusOutput, err := svc.DescribeCreateAccountStatus(&describeCreateAccountStatusInput)
			if err != nil {
				done <- nil
			}

			logger.Debugf("%#v", describeCreateAccountStatusOutput.CreateAccountStatus)

			state := *describeCreateAccountStatusOutput.CreateAccountStatus.State
			if state == organizations.CreateAccountStateSucceeded || state == organizations.CreateAccountStateFailed {
				ticker.Stop()
				done <- describeCreateAccountStatusOutput
			}
		}
	}()

	if describeCreateAccountStatusOutput := <-done; describeCreateAccountStatusOutput != nil {
		return describeCreateAccountStatusOutput.CreateAccountStatus, nil
	}

	return nil, errors.New("failed to create account")
}

func createCHTRole(ctx context.Context, svc *iam.IAM, account *organizations.Account) error {
	gs, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}

	rc, err := gs.Bucket("hello-static-assets").Object("cloudhealth/iam-policy.json").NewReader(ctx)
	if err != nil {
		return err
	}
	defer rc.Close()

	policyDocument, err := io.ReadAll(rc)
	if err != nil {
		return err
	}

	createRoleOutput, err := svc.CreateRole(&iam.CreateRoleInput{
		RoleName:                 aws.String("CloudHealth"),
		AssumeRolePolicyDocument: aws.String(assumeRolePolicyDocument),
	})
	if err != nil {
		return err
	}

	createPolicyOutput, err := svc.CreatePolicy(&iam.CreatePolicyInput{
		PolicyName:     aws.String("CloudHealth"),
		PolicyDocument: aws.String(string(policyDocument)),
	})
	if err != nil {
		return err
	}

	if _, err := svc.AttachRolePolicy(&iam.AttachRolePolicyInput{
		PolicyArn: createPolicyOutput.Policy.Arn,
		RoleName:  createRoleOutput.Role.RoleName,
	}); err != nil {
		return err
	}

	return nil
}

func sendResetPasswordInstructions(account *organizations.Account, requesterEmail string, ccs []string) error {
	data := struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}{
		ID:    *account.Id,
		Name:  *account.Name,
		Email: *account.Email,
	}

	m := mail.NewV3Mail()
	m.SetTemplateID(mailer.Config.DynamicTemplates.AmazonWebServicesAccountCreated)
	m.SetFrom(mail.NewEmail(mailer.Config.NoReplyName, mailer.Config.NoReplyEmail))
	m.SetTrackingSettings(&mail.TrackingSettings{SubscriptionTracking: &mail.SubscriptionTrackingSetting{Enable: common.Bool(false)}})

	tos := []string{*account.Email, requesterEmail}
	bccs := []string{"awsops@doit-intl.com"}
	uniqEmails := make(map[string]bool)
	personalization := mail.NewPersonalization()

	for _, email := range tos {
		if _, prs := uniqEmails[email]; email == "" || prs {
			continue
		}

		personalization.AddTos(mail.NewEmail("", email))

		uniqEmails[email] = true
	}

	// Do not CC/BCC account managers in dev envs
	if common.Production {
		for _, email := range append(ccs, bccs...) {
			if _, prs := uniqEmails[email]; email == "" || prs {
				continue
			}

			personalization.AddCCs(mail.NewEmail("", email))

			uniqEmails[email] = true
		}
	}

	personalization.SetDynamicTemplateData("data", data)
	m.AddPersonalizations(personalization)

	request := sendgrid.GetRequest(mailer.Config.APIKey, mailer.Config.MailSendPath, mailer.Config.BaseURL)
	request.Method = "POST"
	request.Body = mail.GetRequestBody(m)

	if _, err := sendgrid.MakeRequestRetry(request); err != nil {
		return err
	}

	return nil
}

func publishAccountCreatedSlackNotification(
	account *organizations.Account,
	requesterEmail string,
	payerAccount *domain.PayerAccount,
	customer common.Customer,
	entity common.Entity,
) error {
	ctx := context.Background()
	fields := []map[string]interface{}{
		{
			"title": "Customer",
			"value": fmt.Sprintf("<https://console.doit.com/customers/%s|%s>", entity.Customer.ID, customer.Name),
			"short": true,
		},
		{
			"title": "Priority ID",
			"value": entity.PriorityID,
			"short": true,
		},
		{
			"title": "Payer Account",
			"value": payerAccount.DisplayName,
			"short": true,
		},
		{
			"title": "Account",
			"value": *account.Id,
			"short": true,
		},
		{
			"title": "Root Email",
			"value": *account.Email,
			"short": true,
		},
		{
			"title": "Account Name",
			"value": *account.Name,
			"short": true,
		},
	}

	message := map[string]interface{}{
		"attachments": []map[string]interface{}{
			{
				"ts":          time.Now().Unix(),
				"color":       "#4CAF50",
				"author_name": fmt.Sprintf("<mailto:%s|%s>", requesterEmail, requesterEmail),
				"title":       fmt.Sprintf("New AWS Account Created"),
				"title_link":  fmt.Sprintf("https://console.doit.com/customers/%s/assets/amazon-web-services", entity.Customer.ID),
				"thumb_url":   "https://storage.googleapis.com/hello-static-assets/logos/amazon-web-services.png",
				"fields":      fields,
			},
		},
	}

	_, err := common.PublishToSlack(ctx, message, slackChannel)

	return err
}
