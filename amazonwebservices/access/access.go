package access

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/doitintl/hello/scheduled-tasks/cloudconnect"
)

const (
	standaloneRole = "doitintl_cmp"
)

func (s *Access) GetAWSSession(accountID, functionName string) (*session.Session, error) {
	doitAWSCreds, err := cloudconnect.GetAWSCredentials()
	if err != nil {
		return nil, err
	}

	doitSession, err := session.NewSession(&aws.Config{
		Region:      aws.String(endpoints.UsEast1RegionID),
		Credentials: doitAWSCreds,
	})
	if err != nil {
		return nil, err
	}

	stsService := sts.New(doitSession)

	roleArn := s.getRoleArn(standaloneRole, accountID)
	roleSessionName := fmt.Sprintf("%sProxy%s", functionName, accountID)
	assumeRoleInput := &sts.AssumeRoleInput{
		RoleArn:         &roleArn,
		RoleSessionName: &roleSessionName,
	}

	assumedRole, err := stsService.AssumeRole(assumeRoleInput)
	if err != nil {
		return nil, err
	}

	if assumedRole == nil || assumedRole.Credentials == nil {
		return nil, errorEmptyAssumeRole
	}

	accessKeyID := *assumedRole.Credentials.AccessKeyId
	secretAccessKey := *assumedRole.Credentials.SecretAccessKey
	sessionToken := *assumedRole.Credentials.SessionToken

	assumedRoleSessionConfig := aws.Config{
		Credentials: credentials.NewStaticCredentials(accessKeyID, secretAccessKey, sessionToken),
	}

	assumedRoleSession, err := session.NewSession(&assumedRoleSessionConfig)
	if err != nil {
		return nil, err
	}

	return assumedRoleSession, nil
}

func (s *Access) getRoleArn(role, accountID string) string {
	return fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, role)
}
