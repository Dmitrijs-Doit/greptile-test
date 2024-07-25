package cloudconnect

import (
	"time"

	"cloud.google.com/go/firestore"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/doitintl/hello/scheduled-tasks/amazonwebservices/awsproxy"
	"github.com/doitintl/hello/scheduled-tasks/common"
)

type RequestARN struct {
	Arn string `form:"arn"`
}

type PolicyPermissions struct {
	Version   string `json:"Version"`
	Statement []struct {
		Sid      string   `json:"Sid"`
		Effect   string   `json:"Effect"`
		Action   []string `json:"Action"`
		Resource string   `json:"Resource"`
	} `json:"Statement"`
}

type AmazonWebServicesCredential struct {
	Customer          *firestore.DocumentRef        `firestore:"customer"`
	RoleID            string                        `firestore:"roleId"`
	RoleName          string                        `firestore:"roleName"`
	Arn               string                        `firestore:"arn"`
	AccountID         string                        `firestore:"accountId"`
	Status            common.CloudConnectStatusType `firestore:"status"`
	CloudPlatform     string                        `firestore:"cloudPlatform"`
	SupportedFeatures []SupportedFeature            `firestore:"supportedFeatures"`
	ErrorStatus       string                        `firestore:"error"`
	TimeLinked        *time.Time                    `firestore:"timeLinked"`
}

type SupportedFeature struct {
	Name                   string `firestore:"name"`
	HasRequiredPermissions bool   `firestore:"hasRequiredPermissions"`
}

func GetAWSCredentials() (*credentials.Credentials, error) {
	awsCred, err := awsproxy.NewCredentials()
	if err != nil {
		return nil, err
	}

	return credentials.NewStaticCredentials(
		*awsCred.AccessKeyId,
		*awsCred.SecretAccessKey,
		*awsCred.SessionToken,
	), nil
}
