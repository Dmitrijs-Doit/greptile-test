package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/doitintl/hello/scheduled-tasks/cloudconnect"
)

// TODO: move this to separate package
type AWSAccess struct{}

func (s *AWSAccess) GetAWSSession(accountID, functionName string) (*session.Session, error) {
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

	roleArn := getRoleArn(standaloneRole, accountID)
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

func (s *AWSAccess) GetSavingsPlansPurchaseRecommendation(input costexplorer.GetSavingsPlansPurchaseRecommendationInput, accountID string) (*costexplorer.GetSavingsPlansPurchaseRecommendationOutput, error) {
	session, err := s.GetAWSSession(accountID, functionGetSavingsPlansPurchaseRecommendation)
	if err != nil {
		return nil, err
	}

	costexplorerService := costexplorer.New(session)

	return costexplorerService.GetSavingsPlansPurchaseRecommendation(&input)
}
