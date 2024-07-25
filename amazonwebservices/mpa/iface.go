package mpa

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/mpa/accounts/domain"
	"github.com/goccy/go-json"
	"google.golang.org/api/groupssettings/v1"
)

const (
	templateMPAAccountIDPlaceholderKey   = "MPA_ACCOUNT_ID"
	templateMPAS3CURBucketPlaceholderKey = "MPA_S3_CUR_BUCKET"
)

var (
	doitPolicyTemplateFile                           = "./scripts/data/doitintl_cmp.json"
	saasDoitRoleTemplateFile                         = "./scripts/data/saas_doitintl_cmp.json"
	saasDoitPolicyTemplateFile                       = "./scripts/data/saas_doitintl_cmp_policy.json"
	doitPolicyOptionalPermissionsMaskTemplateFile    = "./scripts/data/doitintl_cmp_optional_permissions_mask.json"
	saasDoitPolicyConditionalPermissionsTemplateFile = "./scripts/data/saas_doitintl_cmp_conditional_permissions.json"
)

type ValidateMPARequest struct {
	AccountID string `json:"accountId"`
	RoleArn   string `json:"roleArn"`
	CURBucket string `json:"curBucket"`
	CURPath   string `json:"curPath"`
}

type ValidateSaaSRequest struct {
	CustomerID string `json:"customerId"`
	AccountID  string `json:"accountId"`
	RoleArn    string `json:"roleArn"`
	CURBucket  string `json:"curBucket"`
	CURPath    string `json:"curPath"`
}

type MPAGoogleGroup struct {
	Domain    string `json:"domain"`
	RootEmail string `json:"rootEmail"`
}

type MPAGoogleGroupUpdate struct {
	MPAGoogleGroup
	CurrentRootEmail string `json:"currentRootEmail"`
}

type Statement struct {
	Sid      string      `json:"Sid"`
	Effect   string      `json:"Effect"`
	Action   interface{} `json:"Action"`
	Resource interface{} `json:"Resource"`
}

type ProcessedStatementsMap map[string]ProcessedStatement

type ProcessedStatement struct {
	Effect   string   `json:"Effect"`
	Action   []string `json:"Action"`
	Resource []string `json:"Resource"`
}

type PolicyPermissions struct {
	accountID string
	curBucket string

	Version   string      `json:"Version"`
	Statement []Statement `json:"Statement"`
}

type RolePermissions struct {
	Version   string `json:"Version"`
	Statement []struct {
		Sid       string      `json:"Sid"`
		Effect    string      `json:"Effect"`
		Action    interface{} `json:"Action"`
		Principal interface{} `json:"Principal"`
	} `json:"Statement"`
}

type DatedPolicyPermissions struct {
	Version        string                 `json:"Version"`
	DatedStatement map[string][]Statement `json:"DatedStatement"`
}

type LinkMpaToSauronData struct {
	AccountNumber string `json:"account_number"`
	Name          string `json:"name"`
	Email         string `json:"email"`
}

type clientKeys struct {
	SauronApiKey string
}

// IMPAService interface for the MPA service
type IMPAService interface {
	ValidateMPA(ctx context.Context, req *ValidateMPARequest) error
	LinkMpaToSauron(ctx context.Context, data *LinkMpaToSauronData) error
	CreateGoogleGroup(ctx context.Context, req *MPAGoogleGroup) error
	CreateGoogleGroupCloudTask(ctx context.Context, req *MPAGoogleGroup) error
	AdjustGoogleGroupSettings(email string) (*groupssettings.Groups, error)
	UpdateGoogleGroup(ctx context.Context, req *MPAGoogleGroupUpdate) error
	DeleteGoogleGroup(ctx context.Context, req *MPAGoogleGroup) error
	GetMasterPayerAccountByAccountNumber(ctx context.Context, accountNumber string) (*domain.MasterPayerAccount, error)
	ValidateSaaS(ctx context.Context, req *ValidateSaaSRequest) error
	RetireMPA(ctx context.Context, payerID string) error
}

// UnmarshalJSON overrides PolicyPermissions JSON unmarshalling applying processing during the decoding phase.
// The processing makes it so that the Statement ordering and inner Statement.Actions and Statement.Resource ordering are deterministic.
// This is needed to reliably compare policies between each other further on.
func (p *PolicyPermissions) UnmarshalJSON(data []byte) error {
	type T PolicyPermissions

	d := string(data)
	t := new(T)

	if p.accountID != "" {
		// if accountID template placeholder is present in data, replace it with the specified value
		d = strings.ReplaceAll(d, templateMPAAccountIDPlaceholderKey, p.accountID)
	}

	if p.curBucket != "" {
		// if CUR bucket template placeholder is present in data, replace it with the specified value
		d = strings.ReplaceAll(d, templateMPAS3CURBucketPlaceholderKey, p.curBucket)
	}

	if err := json.Unmarshal([]byte(d), t); err != nil {
		return err
	}
	// deterministically sort the actions and resources within the statements
	for i, s := range t.Statement {
		action := toStringSlice(s.Action)
		resource := toStringSlice(s.Resource)

		slices.Sort(action)
		slices.Sort(resource)

		t.Statement[i].Action = action
		t.Statement[i].Resource = resource
	}
	// deterministically sort the statements
	slices.SortStableFunc(t.Statement, func(a, b Statement) int {
		return cmp.Compare(a.Sid, b.Sid)
	})

	p.Statement = t.Statement
	p.Version = t.Version

	return nil
}

// ToProcessedStatementsMap returns a map where the keys are the PolicyPermissions statement Sids, and values the processed statements.
func (p *PolicyPermissions) ToProcessedStatementsMap() ProcessedStatementsMap {
	policyPermissionsMap := make(ProcessedStatementsMap)

	for _, s := range p.Statement {
		policyPermissionsMap[s.Sid] = ProcessedStatement{
			Effect:   s.Effect,
			Action:   toStringSlice(s.Action),
			Resource: toStringSlice(s.Resource),
		}
	}

	return policyPermissionsMap
}
