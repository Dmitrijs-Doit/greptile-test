package mpa

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/doitintl/firestore/pkg"
	"github.com/doitintl/hello/scheduled-tasks/logger"
	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
)

const (
	testSaasDoitPolicyTemplateFile                       = "../../scripts/data/saas_doitintl_cmp_policy.json"
	testSaasDoitPolicyConditionalPermissionsTemplateFile = "../../scripts/data/saas_doitintl_cmp_conditional_permissions.json"
)

func TestMasterPayerAccountService_validateSaaSPolicyHasRequiredPermissions(t *testing.T) {
	var (
		testAccountID = randomAccountID()
		testCURBucket = randomCURBucketValue()
	)

	ctx := context.Background()
	timePast := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	timeNow := time.Now()
	emptyCloudConnect := pkg.AWSCloudConnect{}

	type args struct { //nolint:wsl
		accountID      string
		curBucket      string
		policyDocument string
		timeCreated    *time.Time
	}
	tests := []struct { //nolint:wsl
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "valid-policy",
			args: args{
				accountID:      testAccountID,
				curBucket:      testCURBucket,
				policyDocument: fixtureSaasPolicyOnboardingV1(testAccountID, testCURBucket),
				timeCreated:    &timePast,
			},
			want: []string{},
		},
		{
			name: "incomplete-policy-before-condition",
			args: args{
				accountID:      testAccountID,
				curBucket:      testCURBucket,
				policyDocument: fixtureSaasPolicyMissingStatementActions(testAccountID, testCURBucket),
				timeCreated:    &timePast,
			},
			want: []string{
				fmt.Sprintf("iam:GetPolicy~iam::%s:policy/doitintl_cmp", testAccountID),
				fmt.Sprintf("iam:GetPolicy~iam::%s:role/doitintl_cmp", testAccountID),
			},
		},
		{
			name: "incomplete-policy-after-condition",
			args: args{
				accountID:      testAccountID,
				curBucket:      testCURBucket,
				policyDocument: fixtureSaasPolicyMissingStatementActions(testAccountID, testCURBucket),
				timeCreated:    &timeNow,
			},
			want: []string{
				fmt.Sprintf("iam:GetPolicy~iam::%s:policy/doitintl_cmp", testAccountID),
				fmt.Sprintf("iam:GetPolicy~iam::%s:role/doitintl_cmp", testAccountID),
				"organizations:ListParents~*",
			},
		},
		{
			name: "incomplete-policy-no-time-created",
			args: args{
				accountID:      testAccountID,
				curBucket:      testCURBucket,
				policyDocument: fixtureSaasPolicyMissingStatementActions(testAccountID, testCURBucket),
				timeCreated:    &emptyCloudConnect.TimeCreated,
			},
			want: []string{
				fmt.Sprintf("iam:GetPolicy~iam::%s:policy/doitintl_cmp", testAccountID),
				fmt.Sprintf("iam:GetPolicy~iam::%s:role/doitintl_cmp", testAccountID),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := testSaasDoitPolicyTemplateFile
			cf := testSaasDoitPolicyConditionalPermissionsTemplateFile
			loggerProvider := func(_ context.Context) logger.ILogger {
				return &loggerMocks.ILogger{}
			}

			s := &MasterPayerAccountService{
				loggerProvider:             loggerProvider,
				saasDoitPolicyTemplateFile: &f,
				saasDoitPolicyConditionalPermissionsTemplateFile: &cf,
			}

			got, err := s.validateSaaSPolicyHasRequiredPermissions(ctx, tt.args.accountID, tt.args.curBucket, tt.args.policyDocument, tt.args.timeCreated)
			slices.Sort(got)

			if !cmp.Equal(got, tt.want) {
				t.Errorf("validateSaaSPolicyHasRequiredPermissions(%s) = %s", tt.name, cmp.Diff(tt.want, got))
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("validateSaaSPolicyHasRequiredPermissions(%s) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func fixtureSaasPolicyOnboardingV1(accountID, curBucket string) string {
	b, _ := os.ReadFile(testSaasDoitPolicyTemplateFile)
	res := string(b)
	res = strings.ReplaceAll(res, templateMPAAccountIDPlaceholderKey, accountID)
	res = strings.ReplaceAll(res, templateMPAS3CURBucketPlaceholderKey, curBucket)

	return res
}

func fixtureSaasPolicyMissingStatementActions(accountID, curBucket string) string {
	return fmt.Sprintf(`
{
	"Statement": [
	{
		"Sid": "Organizations",
		"Effect": "Allow",
		"Action": [
		"organizations:DescribeAccount",
		"organizations:DescribeHandshake",
		"organizations:DescribeOrganization",
		"organizations:ListAccounts*",
		"organizations:ListHandshakes*",
        "organizations:ListTagsForResource"
		],
		"Resource": "*"
	},
	{
		"Effect": "Allow",
		"Sid": "HealthKnownIssues",
		"Action": [
		"health:EnableHealthServiceAccessForOrganization",
		"health:DescribeEventsForOrganization",
		"health:DescribeEvents"
		],
		"Resource": "*"
	},
	{
		"Sid": "Finops",
		"Effect": "Allow",
		"Action": [
		"ec2:DescribeReservedInstances",
		"savingsplans:DescribeSavingsPlans",
		"ce:Get*",
		"ce:List*",
		"ce:Describe*",
		"cur:DescribeReportDefinitions",
		"ce:UpdateCostAllocationTagsStatus"
		],
		"Resource": "*"
	},
	{
		"Sid": "Onboarding",
		"Effect": "Allow",
		"Action": ["iam:GetRole", "iam:GetPolicyVersion"],
		"Resource": ["arn:aws:iam::%[1]v:role/doitintl_cmp", "arn:aws:iam::%[1]v:policy/doitintl_cmp"]
	},
	{
		"Sid": "BillingBucket",
		"Effect": "Allow",
		"Action": ["s3:ListBucket"],
		"Resource": "arn:aws:s3:::%[2]v"
	},
	{
		"Sid": "BillingObject",
		"Effect": "Allow",
		"Action": ["s3:GetObject"],
		"Resource": "arn:aws:s3:::%[2]v/*"
	}
	],
	"Version": "2012-10-17"
}
`, accountID, curBucket)
}
