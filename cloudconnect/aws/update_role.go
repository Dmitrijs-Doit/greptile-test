package aws

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/iam"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/doitintl/hello/scheduled-tasks/cloudconnect/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

var (
	ErrAccountAlreadyExists = errors.New("already_exists")
	ErrArnNotValid          = errors.New("invalid_arn") // if aws returned an error when we try to get role details
	ErrArnUnauthorized      = errors.New("unauthorized")
	ErrNotExistsInAssets    = errors.New("not_in_assets")
	ErrInvalidExternalID    = errors.New("invalid_external_id") // if the user changed the external id inside the "create stack"
	ErrMissingStackName     = errors.New("missing_stack_name")
	ErrEmailAlreadySent     = errors.New("email_already_sent")
)

type RoleRequest struct {
	CustomerID string `json:"external_id"`
	Arn        string `json:"management_arn"`
	AccountID  string `json:"account_id"`
	StackID    string `json:"stack_id"`
	StackName  string `json:"stack_name"`
}

func (p *Permissions) AddRole(ctx context.Context, req *RoleRequest) error {
	splitArn := strings.Split(req.Arn, ":")
	if len(splitArn) <= 4 {
		return ErrArnNotValid
	}

	req.AccountID = splitArn[4]

	return p.CreateCloudConnectDocument(ctx, req)
}

func (p *Permissions) UpdateRoleAndSendNotification(ctx context.Context, req *RoleRequest) error {
	l := p.loggerProvider(ctx)

	if err := p.CreateCloudConnectDocument(ctx, req); err != nil {
		notificationErr := p.UpdateChannelNotification(ctx, req, err.Error())
		if notificationErr != nil {
			l.Errorf("could not update channels notification. error %s", notificationErr)
		}

		return err
	}

	if err := p.UpdateChannelNotification(ctx, req, ""); err != nil {
		return err
	}

	return nil
}

func (p *Permissions) DeleteRole(ctx context.Context, req *RoleRequest) error {
	customerRef := p.fs.Collection("customers").Doc(req.CustomerID)
	query := p.fs.CollectionGroup("cloudConnect").
		Where("cloudPlatform", "==", common.Assets.AmazonWebServices).
		Where("accountId", "==", req.AccountID).
		Where("customer", "==", customerRef)

	docSnaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil
		}

		return err
	}

	if len(docSnaps) == 0 {
		return nil
	}

	_, err = docSnaps[0].Ref.Delete(ctx) // assuming account id is unique.
	if err != nil {
		return fmt.Errorf("could not delete account from firestore. error %s", err)
	}

	return nil
}

func (p *Permissions) CreateCloudConnectDocument(ctx context.Context, req *RoleRequest) error {
	customerRef := p.fs.Collection("customers").Doc(req.CustomerID)

	_, err := customerRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return ErrInvalidExternalID
		}
	}

	inAssets, err := p.isAccountInAssets(ctx, req)
	if err != nil {
		return err
	}

	// in case of account not exist in assets - we are returning without updating.
	if !inAssets {
		return ErrNotExistsInAssets
	}

	shouldCreate, err := p.shouldCreateAccountRole(ctx, req)
	if err != nil {
		return err
	}

	// in case of account already exists - we are returning without updating.
	if !shouldCreate {
		return ErrAccountAlreadyExists
	}

	var roleName string

	xs := strings.Split(req.Arn, "/")
	if len(xs) == 2 {
		roleName = xs[1]
	}

	account := Account{
		Customer:      customerRef,
		Arn:           req.Arn,
		AccountID:     req.AccountID,
		CloudPlatform: common.Assets.AmazonWebServices,
	}
	account.SupportedFeatures = make([]SupportedFeature, 0)

	svc := p.initIAM(&account)
	roleID, timeLinked, err := p.getRoleDetails(svc, roleName)

	if err != nil {
		return err
	}

	account.RoleID = roleID
	account.RoleName = roleName
	account.TimeLinked = timeLinked

	_, err = account.Customer.Collection("cloudConnect").
		Doc(common.Assets.AmazonWebServices+"-"+account.RoleID).Set(ctx, account)
	if err != nil {
		return fmt.Errorf("could not update firestore with new account role. error %s", err)
	}

	return nil
}

func (p *Permissions) shouldCreateAccountRole(ctx context.Context, req *RoleRequest) (bool, error) {
	customerRef := p.fs.Collection("customers").Doc(req.CustomerID)
	query := p.fs.CollectionGroup("cloudConnect").
		Where("type", "in", []string{common.Assets.AmazonWebServices, common.Assets.AmazonWebServicesStandalone}).
		Where("accountId", "==", req.AccountID).
		Where("customer", "==", customerRef)

	docSnaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return true, nil
		}

		return false, fmt.Errorf("could not retrieve account id %s error %s", req.AccountID, err)
	}

	if len(docSnaps) == 0 {
		return true, nil
	}

	return false, nil
}

func (p *Permissions) isAccountInAssets(ctx context.Context, req *RoleRequest) (bool, error) {
	customerRef := p.fs.Collection("customers").Doc(req.CustomerID)

	docSnaps, err := p.fs.Collection("assets").
		Where("type", "in", []string{common.Assets.AmazonWebServices, common.Assets.AmazonWebServicesStandalone}).
		Where("customer", "==", customerRef).
		Where("properties.accountId", "==", req.AccountID).Documents(ctx).GetAll()
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return false, nil
		}

		return false, err
	}

	if len(docSnaps) == 0 {
		return false, nil
	}

	var asset Asset

	err = docSnaps[0].DataTo(&asset) // assuming account id is unique - we should only get one document.
	if err != nil {
		return false, err
	}

	if asset.Properties != nil &&
		asset.Properties.OrganizationInfo != nil &&
		asset.Properties.OrganizationInfo.PayerAccount != nil {
		return true, nil
	}

	return false, nil
}

func (p *Permissions) UpdateChannelNotification(ctx context.Context, req *RoleRequest, errMsg string) error {
	if req.StackName == "" && req.StackID != "" {
		parts := strings.Split(req.StackID, "/")
		if len(parts) >= 2 {
			req.StackName = parts[1]
		}
	}

	if req.StackName == "" {
		return ErrMissingStackName
	}

	query := p.fs.Collection("channels").
		Where("stackName", "==", req.StackName)

	docSnaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		return err
	}

	for i := range docSnaps {
		var channel Channel

		err = docSnaps[i].DataTo(&channel)
		if err != nil {
			return err
		}

		_, err = docSnaps[i].Ref.Update(ctx, []firestore.Update{
			{FieldPath: []string{"complete"}, Value: true},
			{FieldPath: []string{"error"}, Value: errMsg},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// getRoleDetails return roleID and creation time of the role
func (p *Permissions) getRoleDetails(svc *iam.IAM, roleName string) (string, *time.Time, error) {
	inputRole := &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	}

	roleOutput, err := svc.GetRole(inputRole)
	if err != nil {
		return "", nil, ErrArnUnauthorized
	}

	return aws.StringValue(roleOutput.Role.RoleId), roleOutput.Role.CreateDate, nil
}

func (p *Permissions) SendWelcomeEmail(ctx context.Context, req *RoleRequest) error {
	l := p.loggerProvider(ctx)

	spot0CustomerFlags, _ := p.dal.GetSpot0CustomerFlags(ctx, req.CustomerID)
	if spot0CustomerFlags != nil && spot0CustomerFlags.WelcomeEmailSent != nil {
		l.Warningf("email already sent for customer: %s", req.CustomerID)
		return ErrEmailAlreadySent
	}

	customer, err := p.dal.GetCustomer(ctx, req.CustomerID)
	if err != nil {
		l.Errorf("could not get customer: %s. %s", req.CustomerID, err)
		return err
	}

	companyName := customer.Name

	accountManagers, err := p.dal.GetCustomerAccountManagers(ctx, customer, common.AccountManagerCompanyDoit)
	if err != nil {
		l.Errorf("could not get account managers for account: %s, %s", req.AccountID, err)
		return err
	}

	admins, err := p.dal.GetCustomerAdmins(ctx, req.CustomerID)
	if err != nil || len(admins) == 0 {
		l.Errorf("could not get customers admins: %s, %s", req.CustomerID, err)
		return err
	}

	var mailRecipients []dal.MailRecipient

	for _, admin := range admins {
		if admin.Email != "" && admin.FirstName != "" {
			mailRecipients = append(mailRecipients, dal.MailRecipient{
				Email:     admin.Email,
				FirstName: cases.Title(language.English).String(admin.FirstName),
			})
		}
	}

	if len(mailRecipients) == 0 {
		return fmt.Errorf("missing admin details for customer: %s", req.CustomerID)
	}

	var bccs []string

	for _, am := range accountManagers {
		if am.Email != "" {
			bccs = append(bccs, am.Email)
		}
	}

	sendErr := p.dal.SendMail(ctx, mailRecipients, bccs, companyName)
	if sendErr == nil {
		if err := p.dal.SetSpot0CustomerFlags(ctx, req.CustomerID); err != nil {
			log.Printf("err setting WelcomeEmailSent timestamp customer: %s, %s\n", req.CustomerID, err)
			return err
		}

		return nil
	}

	return sendErr
}

// temporary. will be removed
type Asset struct {
	Properties *properties `firestore:"properties"`
}

type properties struct {
	OrganizationInfo *organization `firestore:"organization"`
}

type organization struct {
	PayerAccount interface{} `firestore:"payerAccount"`
}

type Channel struct {
	Complete   bool   `firestore:"complete"`
	CustomerID string `firestore:"customer"`
	Type       string `firestore:"type"`
	Error      string `firestore:"error"`
}
