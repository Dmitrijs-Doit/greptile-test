package dal

import (
	"github.com/aws/aws-sdk-go/service/costandusagereportservice"

	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"
)

type IAWSDal interface {
	ListObjectsV2(accountID, s3Bucket string) (*s3.ListObjectsV2Output, error)
	DescribeReportDefinitions(accountID string) (*costandusagereportservice.DescribeReportDefinitionsOutput, error)
	GetRole(accountID, roleArn string) (*iam.GetRoleOutput, error)
	GetPolicy(accountID, policyArn string) (*iam.GetPolicyOutput, error)
	GetPolicyVersion(accountID, policyArn, versionID string) (*iam.GetPolicyVersionOutput, error)
}
