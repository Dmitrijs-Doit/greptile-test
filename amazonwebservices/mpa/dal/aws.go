package dal

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/costandusagereportservice"

	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices"
)

const (
	doitRole   = "doitintl_cmp"
	doitPolicy = "doitintl_cmp"
)

var (
	config = aws.Config{Region: aws.String(endpoints.UsEast1RegionID)}
)

type AWSDal struct {
	awsSession *session.Session
	accountID  string
}

func NewAWSDal() IAWSDal {
	return &AWSDal{nil, ""}
}

func (d *AWSDal) initSessionOnce(accountID string) error {
	if d.awsSession == nil || d.accountID != accountID {
		doitSession, err := amazonwebservices.GetAssumedSessionForAccount(accountID)
		if err != nil {
			return err
		}

		d.awsSession = doitSession
		d.accountID = accountID
	}

	return nil
}

func (d *AWSDal) ListObjectsV2(accountID, s3Bucket string) (*s3.ListObjectsV2Output, error) {
	if err := d.initSessionOnce(accountID); err != nil {
		return nil, err
	}

	s3Service := s3.New(d.awsSession, &config)

	return s3Service.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket: aws.String(s3Bucket),
	})
}

func (d *AWSDal) DescribeReportDefinitions(accountID string) (*costandusagereportservice.DescribeReportDefinitionsOutput, error) {
	if err := d.initSessionOnce(accountID); err != nil {
		return nil, err
	}

	curService := costandusagereportservice.New(d.awsSession, &config)

	return curService.DescribeReportDefinitions(nil)
}

func (d *AWSDal) GetRole(accountID, roleArn string) (*iam.GetRoleOutput, error) {
	if err := d.initSessionOnce(accountID); err != nil {
		return nil, err
	}

	iamService := iam.New(d.awsSession, &config)

	return iamService.GetRole(&iam.GetRoleInput{
		RoleName: aws.String(doitRole),
	})
}

func (d *AWSDal) GetPolicy(accountID, policyArn string) (*iam.GetPolicyOutput, error) {
	if err := d.initSessionOnce(accountID); err != nil {
		return nil, err
	}

	iamService := iam.New(d.awsSession, &config)

	return iamService.GetPolicy(&iam.GetPolicyInput{
		PolicyArn: aws.String(policyArn),
	})
}

func (d *AWSDal) GetPolicyVersion(accountID, policyArn, versionID string) (*iam.GetPolicyVersionOutput, error) {
	if err := d.initSessionOnce(accountID); err != nil {
		return nil, err
	}

	iamService := iam.New(d.awsSession, &config)

	return iamService.GetPolicyVersion(&iam.GetPolicyVersionInput{
		PolicyArn: &policyArn,
		VersionId: &versionID,
	})
}
