package mpa

import (
	"testing"
	"time"

	loggerMocks "github.com/doitintl/hello/scheduled-tasks/logger/mocks"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/mock"
)

const testInvalidSaasDoitPolicyConditionalPermissionsTemplateFile = "../../scripts/data/file_that_does_not_exist.json"

func Test_applyPermissionMask(t *testing.T) {
	type args struct {
		permissions *PolicyPermissions
		mask        *PolicyPermissions
	}

	typicalMask := &PolicyPermissions{
		Statement: []Statement{
			{
				Sid:    "Organizations",
				Effect: "Allow",
				Action: []string{
					"organizations:CreateAccount",
					"organizations:InviteAccountToOrganization",
				},
				Resource: "*",
			},
			{
				Sid:    "OrganizationAccountAccessRole",
				Effect: "Allow",
				Action: []string{
					"sts:AssumeRole",
				},
				Resource: "arn:aws:iam::*:role/OrganizationAccountAccessRole",
			},
			{
				Sid:    "BillingCMP",
				Effect: "Allow",
				Action: []string{
					"ce:UpdateCostAllocationTagsStatus",
				},
				Resource: "*",
			},
		},
	}

	tests := []struct {
		name    string
		args    args
		want    *PolicyPermissions
		wantErr bool
	}{
		{
			name: "typical-use-case",
			args: args{
				permissions: &PolicyPermissions{
					Statement: []Statement{
						{
							Effect: "Allow",
							Sid:    "Organizations",
							Action: []string{
								"organizations:InviteAccountToOrganization",
								"organizations:ListAccounts*",
								"organizations:DescribeAccount",
								"organizations:ListHandshakes*",
								"organizations:DescribeCreateAccountStatus",
								"organizations:DescribeOrganization",
								"organizations:DescribeHandshake",
								"organizations:CreateAccount",
							},
							Resource: "*",
						},
						{
							Sid:    "BillingCMP",
							Effect: "Allow",
							Action: []string{
								"ce:UpdateCostAllocationTagsStatus",
								"cur:DescribeReportDefinitions",
							},
							Resource: "*",
						},
					},
				},
				mask: typicalMask,
			},
			want: &PolicyPermissions{
				Statement: []Statement{ // ordered slice statements are expected
					{
						Sid:    "BillingCMP",
						Effect: "Allow",
						Action: []string{
							"cur:DescribeReportDefinitions",
						},
						Resource: "*",
					},
					{
						Sid:    "Organizations",
						Effect: "Allow",
						Action: []string{ // ordered slice elements are expected
							"organizations:DescribeAccount",
							"organizations:DescribeCreateAccountStatus",
							"organizations:DescribeHandshake",
							"organizations:DescribeOrganization",
							"organizations:ListAccounts*",
							"organizations:ListHandshakes*",
						},
						Resource: "*",
					},
				},
			},
		},
		{
			name: "remove-statement-due-to-empty-resource",
			args: args{
				permissions: &PolicyPermissions{
					Statement: []Statement{
						{
							Sid:    "Organizations",
							Effect: "Allow",
							Action: []string{
								"organizations:CreateAccount",
								"organizations:InviteAccountToOrganization",
							},
							Resource: "*",
						},
						{
							Sid:    "BillingCMP",
							Effect: "Allow",
							Action: []string{
								"ce:UpdateCostAllocationTagsStatus",
							},
							Resource: "*",
						},
					},
				},
				mask: typicalMask,
			},
			want: &PolicyPermissions{
				Statement: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := applyPermissionMask(tt.args.permissions, tt.args.mask); !cmp.Equal(got, tt.want, cmpopts.IgnoreUnexported(PolicyPermissions{})) {
				t.Errorf("applyPermissionMask(%s) = %s", tt.name, cmp.Diff(tt.want, got, cmpopts.IgnoreUnexported(PolicyPermissions{})))
			}
		})
	}
}

func Test_getConditionalOptionalPermissionsFromTemplate(t *testing.T) {
	type args struct {
		templatePath      string
		policyTimeCreated time.Time
	}

	timePast := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		args    args
		want    *PolicyPermissions
		wantErr bool
	}{
		{
			name: "policy-created-before-condition",
			args: args{
				templatePath:      testSaasDoitPolicyConditionalPermissionsTemplateFile,
				policyTimeCreated: timePast,
			},
			want: &PolicyPermissions{
				Statement: []Statement{ // ordered slice statements are expected
					{
						Sid:      "Organizations",
						Effect:   "Allow",
						Action:   []any{string("organizations:ListParents")},
						Resource: "*",
					},
				},
			},
		},
		{
			name: "policy-created-after-condition",
			args: args{
				templatePath:      testSaasDoitPolicyConditionalPermissionsTemplateFile,
				policyTimeCreated: time.Now(),
			},
			want: nil,
		},
		{
			name: "invalid-template-file-name",
			args: args{
				templatePath:      testInvalidSaasDoitPolicyConditionalPermissionsTemplateFile,
				policyTimeCreated: timePast,
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		loggerMock := &loggerMocks.ILogger{}
		loggerMock.On("Errorf", mock.Anything, mock.Anything)

		t.Run(tt.name, func(t *testing.T) {
			if got := getConditionalOptionalPermissionsFromTemplate(loggerMock, tt.args.templatePath, tt.args.policyTimeCreated); !cmp.Equal(got, tt.want, cmpopts.IgnoreUnexported(PolicyPermissions{})) {
				t.Errorf("getConditionalOptionalPermissionsFromTemplate(%s) = %s", tt.name, cmp.Diff(tt.want, got, cmpopts.IgnoreUnexported(PolicyPermissions{})))
			}
		})
	}
}
