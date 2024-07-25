package mpa

import (
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"
)

const (
	testDoitPolicyTemplateFile       = "../../scripts/data/doitintl_cmp.json"
	testingAWSAccountIDMinOffset     = 100000000000 // guarantees the computed id will not be less than 12 digits
	testingAWSAccountIDMaxOuterRange = 899999999999 // guarantees the computed id will not be more than 12 digits
)

func TestMasterPayerAccountService_validatePolicyHasRequiredPermissions(t *testing.T) {
	var (
		testAccountID = randomAccountID()
		testCURBucket = randomCURBucketValue()
	)

	type args struct { //nolint:wsl
		accountID      string
		curBucket      string
		policyDocument string
	}
	tests := []struct { //nolint:wsl
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "onboarding-v1",
			args: args{
				accountID:      testAccountID,
				curBucket:      testCURBucket,
				policyDocument: fixturePolicyOnboardingV1(testAccountID, testCURBucket),
			},
		},
		{
			name: "onboarding-v2",
			args: args{
				accountID:      testAccountID,
				curBucket:      testCURBucket,
				policyDocument: fixturePolicyOnboardingV2(testAccountID, testCURBucket),
			},
		},
		{
			name: "random-statement-order",
			args: args{
				accountID:      testAccountID,
				curBucket:      testCURBucket,
				policyDocument: fixturePolicyRandomStatementOrder(testAccountID, testCURBucket),
			},
		},
		{
			name: "missing-statement",
			args: args{
				accountID:      testAccountID,
				curBucket:      testCURBucket,
				policyDocument: fixturePolicyMissingStatements(testCURBucket),
			},
			wantErr: true,
		},
		{
			name: "missing-statement-actions",
			args: args{
				accountID:      testAccountID,
				curBucket:      testCURBucket,
				policyDocument: fixturePolicyMissingStatementActions(testAccountID, testCURBucket),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			f := testDoitPolicyTemplateFile
			s := &MasterPayerAccountService{doitPolicyTemplateFile: &f}

			if err := s.validatePolicyHasRequiredPermissions(tt.args.accountID, tt.args.curBucket, tt.args.policyDocument); (err != nil) != tt.wantErr {
				t.Errorf("validatePolicyHasRequiredPermissions(%s) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
		})
	}
}

func randomAccountID() string {
	return strconv.Itoa(int(testingAWSAccountIDMinOffset + rand.Int63n(testingAWSAccountIDMaxOuterRange)))
}

func randomCURBucketValue() string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

	length := max(rand.Int63n(50), 10)
	b := make([]rune, length)

	for i := range b {
		b[i] = letters[rand.Int63n(int64(len(letters)))]
	}

	return string(b)
}

func fixturePolicyOnboardingV1(accountID, curBucket string) string {
	b, _ := os.ReadFile(testDoitPolicyTemplateFile)
	res := string(b)
	res = strings.ReplaceAll(res, templateMPAAccountIDPlaceholderKey, accountID)
	res = strings.ReplaceAll(res, templateMPAS3CURBucketPlaceholderKey, curBucket)

	return res
}

func fixturePolicyOnboardingV2(accountID, curBucket string) string {
	return fmt.Sprintf(`
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Action": "sts:AssumeRole",
			"Resource": "arn:aws:iam::*:role/OrganizationAccountAccessRole",
			"Effect": "Allow",
			"Sid": "OrganizationAccountAccessRole"
		},
		{
			"Action": [
				"organizations:CreateAccount",
				"organizations:DescribeAccount",
				"organizations:DescribeCreateAccountStatus",
				"organizations:DescribeHandshake",
				"organizations:DescribeOrganization",
				"organizations:InviteAccountToOrganization",
				"organizations:ListAccounts*",
				"organizations:ListHandshakes*",
				"organizations:ListParents",
				"organizations:ListTagsForResource"
			],
			"Resource": "*",
			"Effect": "Allow",
			"Sid": "Organizations"
		},
		{
			"Action": [
				"iam:GetPolicy*",
				"iam:GetRole"
			],
			"Resource": [
				"arn:aws:iam::%[1]v:policy/doitintl_cmp",
				"arn:aws:iam::%[1]v:role/doitintl_cmp"
			],
			"Effect": "Allow",
			"Sid": "Onboarding"
		},
		{
			"Action": [
				"ce:UpdateCostAllocationTagsStatus",
				"cur:DescribeReportDefinitions"
			],
			"Resource": "*",
			"Effect": "Allow",
			"Sid": "BillingCMP"
		},
		{
			"Action": [
				"health:DescribeEvents",
				"health:DescribeEventsForOrganization",
				"health:EnableHealthServiceAccessForOrganization"
			],
			"Resource": "*",
			"Effect": "Allow",
			"Sid": "HealthKnownIssues"
		},
		{
			"Action": [
				"ce:Describe*",
				"ce:Get*",
				"ce:List*",
				"ec2:DescribeReservedInstances",
				"iam:GetRole",
				"savingsplans:Describe*"
			],
			"Resource": "*",
			"Effect": "Allow",
			"Sid": "BillingPipeline"
		},
		{
			"Action": "s3:ListBucket",
			"Resource": "arn:aws:s3:::%[2]v",
			"Effect": "Allow",
			"Sid": "S3Bucket"
		},
		{
			"Action": "s3:GetObject",
			"Resource": "arn:aws:s3:::%[2]v/*",
			"Effect": "Allow",
			"Sid": "S3Object"
		}
	]
}
`, accountID, curBucket)
}

func fixturePolicyRandomStatementOrder(accountID, curBucket string) string {
	return fmt.Sprintf(`
{
	"Version": "2012-10-17",
	"Statement": [

		{
			"Action": [
				"organizations:CreateAccount",
				"organizations:DescribeAccount",
				"organizations:DescribeCreateAccountStatus",
				"organizations:DescribeHandshake",
				"organizations:DescribeOrganization",
				"organizations:InviteAccountToOrganization",
				"organizations:ListAccounts*",
				"organizations:ListHandshakes*",
				"organizations:ListParents",
				"organizations:ListTagsForResource"
			],
			"Resource": "*",
			"Effect": "Allow",
			"Sid": "Organizations"
		},
		{
			"Action": "sts:AssumeRole",
			"Resource": "arn:aws:iam::*:role/OrganizationAccountAccessRole",
			"Effect": "Allow",
			"Sid": "OrganizationAccountAccessRole"
		},
		{
			"Action": [
				"iam:GetPolicy*",
				"iam:GetRole"
			],
			"Resource": [
				"arn:aws:iam::%[1]v:policy/doitintl_cmp",
				"arn:aws:iam::%[1]v:role/doitintl_cmp"
			],
			"Effect": "Allow",
			"Sid": "Onboarding"
		},
		{
			"Action": [
				"health:DescribeEvents",
				"health:DescribeEventsForOrganization",
				"health:EnableHealthServiceAccessForOrganization"
			],
			"Resource": "*",
			"Effect": "Allow",
			"Sid": "HealthKnownIssues"
		},
		{
			"Action": "s3:ListBucket",
			"Resource": "arn:aws:s3:::%[2]v",
			"Effect": "Allow",
			"Sid": "S3Bucket"
		},
		{
			"Action": [
				"ce:Describe*",
				"ce:Get*",
				"ce:List*",
				"ec2:DescribeReservedInstances",
				"iam:GetRole",
				"savingsplans:Describe*"
			],
			"Resource": "*",
			"Effect": "Allow",
			"Sid": "BillingPipeline"
		},

		{
			"Action": "s3:GetObject",
			"Resource": "arn:aws:s3:::%[2]v/*",
			"Effect": "Allow",
			"Sid": "S3Object"
		},
		{
			"Action": [
				"ce:UpdateCostAllocationTagsStatus",
				"cur:DescribeReportDefinitions"
			],
			"Resource": "*",
			"Effect": "Allow",
			"Sid": "BillingCMP"
		}
	]
}
`, accountID, curBucket)
}

func fixturePolicyMissingStatements(curBucket string) string {
	return fmt.Sprintf(`
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Action": "sts:AssumeRole",
			"Resource": "arn:aws:iam::*:role/OrganizationAccountAccessRole",
			"Effect": "Allow",
			"Sid": "OrganizationAccountAccessRole"
		},
		{
			"Action": [
				"organizations:CreateAccount",
				"organizations:DescribeAccount",
				"organizations:DescribeCreateAccountStatus",
				"organizations:DescribeHandshake",
				"organizations:DescribeOrganization",
				"organizations:InviteAccountToOrganization",
				"organizations:ListAccounts*",
				"organizations:ListHandshakes*"
			],
			"Resource": "*",
			"Effect": "Allow",
			"Sid": "Organizations
		},
		{
			"Action": "s3:ListBucket",
			"Resource": "arn:aws:s3:::%[1]v",
			"Effect": "Allow",
			"Sid": "S3Bucket"
		},
		{
			"Action": "s3:GetObject",
			"Resource": "arn:aws:s3:::%[1]v/*",
			"Effect": "Allow",
			"Sid": "S3Object"
		}
	]
}
`, curBucket)
}

func fixturePolicyMissingStatementActions(accountID, curBucket string) string {
	return fmt.Sprintf(`
{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Action": "sts:AssumeRole",
			"Resource": "arn:aws:iam::*:role/OrganizationAccountAccessRole",
			"Effect": "Allow",
			"Sid": "OrganizationAccountAccessRole"
		},
		{
			"Action": [
				"organizations:CreateAccount",
				"organizations:DescribeCreateAccountStatus",
				"organizations:DescribeHandshake",
				"organizations:ListAccounts*",
				"organizations:ListHandshakes*"
			],
			"Resource": "*",
			"Effect": "Allow",
			"Sid": "Organizations"
		},
		{
			"Action": [
				"iam:GetPolicy*",
				"iam:GetRole"
			],
			"Resource": [
				"arn:aws:iam::%[1]v:policy/doitintl_cmp",
				"arn:aws:iam::%[1]v:role/doitintl_cmp"
			],
			"Effect": "Allow",
			"Sid": "Onboarding"
		},
		{
			"Action": [
				"ce:UpdateCostAllocationTagsStatus",
				"cur:DescribeReportDefinitions"
			],
			"Resource": "*",
			"Effect": "Allow",
			"Sid": "BillingCMP"
		},
		{
			"Action": [
				"health:DescribeEvents",
				"health:DescribeEventsForOrganization"
			],
			"Resource": "*",
			"Effect": "Allow",
			"Sid": "HealthKnownIssues"
		},
		{
			"Action": [
				"ce:Describe*",
				"iam:GetRole",
				"savingsplans:Describe*"
			],
			"Resource": "*",
			"Effect": "Allow",
			"Sid": "BillingPipeline"
		},
		{
			"Action": "s3:ListBucket",
			"Resource": "arn:aws:s3:::%[2]v",
			"Effect": "Allow",
			"Sid": "S3Bucket"
		},
		{
			"Action": "s3:GetObject",
			"Resource": "arn:aws:s3:::%[2]v/*",
			"Effect": "Allow",
			"Sid": "S3Object"
		}
	]
}`, accountID, curBucket)
}
