package amazonwebservices

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/doitintl/hello/scheduled-tasks/cloudconnect"
)

type AssumedSession interface {
	GetAssumedSessionForAccount(accountID string) (*session.Session, error)
}

type Service struct{}

func NewService() AssumedSession {
	return Service{}
}

func (s Service) GetAssumedSessionForAccount(accountID string) (*session.Session, error) {
	return GetAssumedSessionForAccount(accountID)
}

func GetAssumedSessionForAccount(accountID string) (*session.Session, error) {
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

	doitSTSService := sts.New(doitSession)

	assumedARN := fmt.Sprintf("arn:aws:iam::%s:role/doitintl_cmp", accountID)
	roleSessionName := fmt.Sprintf("%sProxy%s", "GetSavingsPlansPurchaseRecommendation", accountID)

	assumedRole, err := doitSTSService.AssumeRole(&sts.AssumeRoleInput{
		RoleArn:         &assumedARN,
		RoleSessionName: &roleSessionName,
	})
	if err != nil {
		return nil, err
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
