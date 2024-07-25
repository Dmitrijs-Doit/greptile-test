package aws

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/awsproxy"
	"github.com/doitintl/hello/scheduled-tasks/cloudconnect/aws/internal"
	"github.com/doitintl/hello/scheduled-tasks/cloudconnect/dal"
	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/logger"
)

type Permissions struct {
	loggerProvider logger.Provider
	fs             *internal.Firestore
	bq             *internal.BigQuery
	session        *session.Session
	dal            dal.IAwsConnect
}

type FeaturePermissions struct {
	FeatureName string   `firestore:"name"`
	Permissions []string `firestore:"permissions"`
	Policies    []string `firestore:"policies"`
}

const (
	securityAuditPolicy = "arn:aws:iam::aws:policy/SecurityAudit"
	billingPolicy       = "arn:aws:iam::aws:policy/job-function/Billing"
	savingPlansPolicy   = "arn:aws:iam::aws:policy/AWSSavingsPlansReadOnlyAccess"
)

var (
	ErrRoleNotValid = errors.New("role is not valid")
	ErrUnauthorized = errors.New("unauthorized permission request")
)

// NewPermissions initializes db connections and constructs them on AWSPermissions.
func NewPermissions(ctx context.Context) (*Permissions, error) {
	fs, err := internal.NewFirestoreClient(ctx)
	if err != nil {
		return nil, err
	}

	bq, err := internal.NewBigQueryClient(ctx)
	if err != nil {
		return nil, err
	}

	se, err := NewAWSSession(ctx)
	if err != nil {
		return nil, err
	}

	dal, err := dal.NewAwsConnect(ctx)
	if err != nil {
		return nil, err
	}

	loggerProvider := logger.FromContext

	return &Permissions{
		loggerProvider,
		fs,
		bq,
		se,
		dal,
	}, nil
}

func NewAWSSession(ctx context.Context) (*session.Session, error) {
	awsCred, err := awsproxy.NewCredentials()
	if err != nil {
		return nil, err
	}

	se, err := session.NewSession(&aws.Config{
		Credentials: credentials.NewStaticCredentials(
			*awsCred.AccessKeyId,
			*awsCred.SecretAccessKey,
			*awsCred.SessionToken,
		),
		MaxRetries: aws.Int(2),
	})
	if err != nil {
		return nil, fmt.Errorf("could not initialize aws session. error %s", err)
	}

	return se, err
}

// Close closes related permissions db connections.
func (p *Permissions) Close() {
	p.fs.Close()
	p.bq.Close()
}

// UpdateAWSPermissions scans accounts with their current roles, and checks required permissions for updating supported features statuses.
func (p *Permissions) UpdateAWSPermissions(ctx context.Context, optionalCustomerID, optionalAccountID string) error {
	l := p.loggerProvider(ctx)

	requiredPermissions, err := p.requiredPermissions(ctx)
	if err != nil {
		return fmt.Errorf("could not retrieve required permissions from firestore. error %s", err)
	}

	query := p.fs.CollectionGroup("cloudConnect").Where("cloudPlatform", "==", common.Assets.AmazonWebServices)

	if len(optionalCustomerID) > 0 {
		customerRef := p.fs.Collection("customers").Doc(optionalCustomerID)
		query = query.Where("customer", "==", customerRef)
	}

	if len(optionalAccountID) > 0 {
		query = query.Where("accountId", "==", optionalAccountID)
	}

	docSnaps, err := query.Documents(ctx).GetAll()
	if err != nil {
		return fmt.Errorf("could not retrieve accounts from firestore. error %s", err)
	}

	var wg sync.WaitGroup

	done := make(chan bool)
	errs := make(chan error)

	for i := range docSnaps {
		var account Account

		err = docSnaps[i].DataTo(&account)
		if err != nil {
			return fmt.Errorf("could not unmarsal data into credential. error %s", err)
		}

		// initializing the state of the account before checking.
		account.SupportedFeatures = []SupportedFeature{}
		account.ErrorStatus = ""

		wg.Add(1)

		go func(acc *Account, errs chan<- error) {
			svc := p.initIAM(&account)

			// update time linked if not exist (for accounts that were created long time ago)
			if account.TimeLinked == nil {
				_, timeLinked, _ := p.getRoleDetails(svc, acc.RoleName)
				account.TimeLinked = timeLinked
			}

			// retrieves permissions details using iam instance.
			accountPermissions, hasCorePolicies, err := p.getPermissions(ctx, svc, acc)
			if err != nil {
				if err == ErrUnauthorized {
					account.ErrorStatus = ErrUnauthorized.Error()
				} else {
					account.ErrorStatus = ErrRoleNotValid.Error()
				}
			}

			if acc.ErrorStatus == "" {
				for _, featurePermissions := range requiredPermissions {
					supportedFeature := SupportedFeature{
						Name:                   featurePermissions.FeatureName,
						HasRequiredPermissions: p.hasRequiredFeaturePermissions(ctx, featurePermissions, accountPermissions, hasCorePolicies),
					}

					acc.SupportedFeatures = append(acc.SupportedFeatures, supportedFeature)
				}
			}

			_, err = acc.Customer.Collection("cloudConnect").
				Doc(common.Assets.AmazonWebServices+"-"+account.RoleID).Set(ctx, account)
			if err != nil {
				// Use a select statement to safely send on the channel
				select {
				case errs <- fmt.Errorf("could not update firestore with new permissions status. error %s", err):
					break
				default:
					l.Warningf("channel closed, unable to send")
				}
			}

			wg.Done()
		}(&account, errs)
	}

	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		break
	case er := <-errs:
		close(errs)
		return er
	}

	return nil
}

func (p *Permissions) requiredPermissions(ctx context.Context) ([]FeaturePermissions, error) {
	docSnap, err := p.fs.Collection("app").Doc("cloud-connect").Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("could not get required permissions from firestore. error %s", err)
	}

	var resp struct {
		AWSFeaturePermissions []FeaturePermissions `firestore:"awsFeaturePermissions"`
	}

	err = docSnap.DataTo(&resp)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal data permissions into struct. error %s", err)
	}

	return resp.AWSFeaturePermissions, nil
}

// hasRequiredFeaturePermissions checks if the account permissions are valid with our required feature permissions.
// it returns bool for the validation status.
func (p *Permissions) hasRequiredFeaturePermissions(ctx context.Context, featurePermissions FeaturePermissions, accountPermissions []string, hasCorePolicies bool) bool {
	l := p.loggerProvider(ctx)

	if featurePermissions.FeatureName == "core" && hasCorePolicies {
		return true
	}

	// checking diff between the our required permissions and the grouped permissions.
	missingPermissions := difference(featurePermissions.Permissions, accountPermissions)

	if len(missingPermissions) > 0 {
		// if there are missing permissions, it's not valid state - hence we return false.
		l.Infof("%s: missing permissions: %s", featurePermissions.FeatureName, missingPermissions)
		return false
	}

	return true
}

// initIAM initializes IAM instance with a customer's session.
func (p *Permissions) initIAM(account *Account) *iam.IAM {
	conf := aws.NewConfig().WithCredentials(stscreds.NewCredentials(p.session, account.Arn, func(arp *stscreds.AssumeRoleProvider) {
		arp.RoleSessionName = "role-session-name"
		arp.Duration = 60 * time.Minute
		arp.ExternalID = aws.String(account.Customer.ID)
	}))

	return iam.New(p.session, conf)
}

// getPermissions retrieves the permissions details from account using the iam instance.
func (p *Permissions) getPermissions(ctx context.Context, svc *iam.IAM, account *Account) ([]string, bool, error) {
	l := p.loggerProvider(ctx)

	// list attached role policies
	listAttachedRolePoliciesOutput, err := svc.ListAttachedRolePolicies(&iam.ListAttachedRolePoliciesInput{
		RoleName: aws.String(account.RoleName),
	})
	if err != nil {
		l.Errorf("account is not authorized. %s", err)
		return nil, false, ErrUnauthorized
	}

	var (
		policyPermissions                                           []string
		isSecurityAuditPolicy, isBillingPolicy, isSavingPlansPolicy bool
	)

	// gets permissions details and grouping it inside policyPermissions slice.
	for _, policy := range listAttachedRolePoliciesOutput.AttachedPolicies {
		policyArn := aws.StringValue(policy.PolicyArn)
		if policyArn == securityAuditPolicy {
			isSecurityAuditPolicy = true
		}

		if policyArn == billingPolicy {
			isBillingPolicy = true
		}

		if policyArn == savingPlansPolicy {
			isSavingPlansPolicy = true
		}

		policyOutput, err := svc.GetPolicy(&iam.GetPolicyInput{
			PolicyArn: policy.PolicyArn,
		})
		if err != nil {
			l.Errorf("could not get policy. error %s", err)
			return nil, false, ErrUnauthorized
		}

		policyVersionOutput, err := svc.GetPolicyVersion(&iam.GetPolicyVersionInput{
			VersionId: policyOutput.Policy.DefaultVersionId,
			PolicyArn: policy.PolicyArn,
		})
		if err != nil {
			return nil, false, fmt.Errorf("could not get policy version. error %s", err)
		}

		document := aws.StringValue(policyVersionOutput.PolicyVersion.Document)

		decodedDocument, err := url.PathUnescape(document)
		if err != nil {
			return nil, false, fmt.Errorf("could not decode policy document. error %s", err)
		}

		var policyVersions PolicyPermissions

		err = json.Unmarshal([]byte(decodedDocument), &policyVersions)
		if err != nil {
			return nil, false, fmt.Errorf("could not unmarshal decoded policy document into struct. error %s", err)
		}

		// grouping policy permissions within policy versions.
		for i := range policyVersions.Statement {
			policyPermissions = append(policyPermissions, policyVersions.Statement[i].Action...)
		}
	}

	hasCorePolicies := isSecurityAuditPolicy && isBillingPolicy && isSavingPlansPolicy

	return policyPermissions, hasCorePolicies, nil
}

// difference returns the elements in `a` that aren't in `b`.
func difference(a, b []string) []string {
	mb := make(map[string]bool, len(b))
	for _, x := range b {
		mb[x] = true
	}

	var diff []string

	for _, x := range a {
		if _, ok := mb[x]; !ok {
			diff = append(diff, x)
		}
	}

	return diff
}
