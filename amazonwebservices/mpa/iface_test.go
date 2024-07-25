package mpa

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type fixture struct {
	data     []byte
	expected PolicyPermissions
}

func TestPolicyPermissions_UnmarshalJSON(t *testing.T) {
	testAccountID := randomAccountID()
	testCURBucket := randomCURBucketValue()

	tests := []struct {
		name    string
		data    []byte
		in      PolicyPermissions
		want    PolicyPermissions
		wantErr bool
	}{
		{
			name: "basic",
			data: fixturePolicyPermissionsUnMarshalJSONBasic(testAccountID, testCURBucket).data,
			in: PolicyPermissions{
				accountID: testAccountID,
				curBucket: testCURBucket,
			},
			want: fixturePolicyPermissionsUnMarshalJSONBasic(testAccountID, testCURBucket).expected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := tt.in
			if err := p.UnmarshalJSON(tt.data); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON(%s) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}

			if !cmp.Equal(p, tt.want, cmpopts.IgnoreUnexported(PolicyPermissions{})) {
				t.Errorf("UnmarshalJSON(%s): %s", tt.name, cmp.Diff(tt.want, p, cmpopts.IgnoreUnexported(PolicyPermissions{})))
			}
		})
	}
}

func fixturePolicyPermissionsUnMarshalJSONBasic(accountID, curBucket string) fixture {
	return fixture{
		data: []byte(fmt.Sprintf(`
{
  "Statement": [
    {
      "Sid": "OrganizationAccountAccessRole",
      "Effect": "Allow",
      "Action": ["sts:AssumeRole"],
      "Resource": "arn:aws:iam::*:role/OrganizationAccountAccessRole"
    },
    {
      "Sid": "BillingPipeline",
      "Effect": "Allow",
      "Action": [
        "iam:GetRole",
        "ec2:DescribeReservedInstances",
        "savingsplans:Describe*",
        "ce:List*",
        "ce:Describe*",
        "ce:Get*"
      ],
      "Resource": "*"
    },
    {
      "Sid": "S3Object",
      "Effect": "Allow",
      "Action": ["s3:GetObject"],
      "Resource": "arn:aws:s3:::%[2]v/*"
    },
    {
      "Action": ["iam:GetRole", "iam:GetPolicy*"],
      "Resource": ["arn:aws:iam::%[1]v:role/doitintl_cmp", "arn:aws:iam::%[1]v:policy/doitintl_cmp"],
      "Effect": "Allow",
      "Sid": "Onboarding"
    }
  ],
  "Version": "2012-10-17"
}
`, accountID, curBucket)),

		expected: PolicyPermissions{
			Version: "2012-10-17",
			Statement: []Statement{ // ordered statements slice is expected
				{
					Sid:    "BillingPipeline",
					Effect: "Allow",
					Action: []string{ // ordered action slices is expected
						"ce:Describe*",
						"ce:Get*",
						"ce:List*",
						"ec2:DescribeReservedInstances",
						"iam:GetRole",
						"savingsplans:Describe*",
					},
					Resource: []string{"*"},
				},
				{
					Sid:    "Onboarding",
					Effect: "Allow",
					Action: []string{
						"iam:GetPolicy*",
						"iam:GetRole",
					},
					Resource: []string{
						fmt.Sprintf("arn:aws:iam::%s:policy/doitintl_cmp", accountID),
						fmt.Sprintf("arn:aws:iam::%s:role/doitintl_cmp", accountID),
					},
				},
				{
					Sid:      "OrganizationAccountAccessRole",
					Effect:   "Allow",
					Action:   []string{"sts:AssumeRole"},
					Resource: []string{"arn:aws:iam::*:role/OrganizationAccountAccessRole"},
				},

				{
					Sid:      "S3Object",
					Effect:   "Allow",
					Action:   []string{"s3:GetObject"},
					Resource: []string{fmt.Sprintf("arn:aws:s3:::%s/*", curBucket)},
				},
			},
		},
	}
}
