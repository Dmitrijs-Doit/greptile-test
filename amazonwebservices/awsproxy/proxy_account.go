package awsproxy

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/doitintl/hello/scheduled-tasks/common"
	"github.com/doitintl/hello/scheduled-tasks/secretmanager"
)

const proxyAccountRoleDev string = "arn:aws:iam::068664126052:role/cmp-aws-payer-dev"
const proxyAccountRoleProd string = "arn:aws:iam::068664126052:role/cmp-aws-payer-prod"
const fiveMinutes time.Duration = time.Duration(5 * time.Minute)

// ProxyAccount is the CMP AWS account (Also used for "Cloud Connect")
// CMP Assumes a role in this account using WebIdentity credentials
// and then we assume an IAM role inside the MPAs which trusts this account
type proxyAccount struct {
	credentials      *sts.Credentials
	mutex            *sync.Mutex
	proxyAccountRole string
}

var account *proxyAccount

func init() {
	account = &proxyAccount{
		mutex: &sync.Mutex{},
	}
	account.setProxyAccountProps()
}

func (pa *proxyAccount) setProxyAccountProps() {
	if common.Production {
		pa.proxyAccountRole = proxyAccountRoleProd
		return
	}

	pa.proxyAccountRole = proxyAccountRoleDev
}

func (pa *proxyAccount) newCredentials() error {
	ctx := context.Background()

	secret, err := secretmanager.AccessSecretLatestVersion(ctx, secretmanager.SecretAWSPayerIntegration)
	if err != nil {
		return err
	}

	type Secret struct {
		ClientID string `json:"client_id"`
	}

	var s Secret
	if err := json.Unmarshal(secret, &s); err != nil {
		if err != nil {
			return err
		}
	}

	token, err := common.GetServiceAccountIDToken(ctx, s.ClientID, secret)
	if err != nil {
		return err
	}

	stsService := sts.New(session.Must(session.NewSession()))
	i := sts.AssumeRoleWithWebIdentityInput{
		DurationSeconds:  aws.Int64(3600),
		RoleArn:          aws.String(pa.proxyAccountRole),
		RoleSessionName:  aws.String(fmt.Sprintf("%s.%s.%s", common.ProjectID, common.GAEService, common.GAEVersion)),
		WebIdentityToken: &token.AccessToken,
	}

	proxyAssumedRole, err := stsService.AssumeRoleWithWebIdentity(&i)
	if err != nil {
		return err
	}

	pa.credentials = proxyAssumedRole.Credentials

	return nil
}

// NewCredentials  will return credentials based on the proxyAccount role.
// If no credentials or credentials expired it will generate new credentials
func NewCredentials() (*sts.Credentials, error) {
	account.mutex.Lock()
	defer account.mutex.Unlock()

	if account.credentials == nil || time.Until(*account.credentials.Expiration) < fiveMinutes {
		if err := account.newCredentials(); err != nil {
			return nil, err
		}
	}

	return account.credentials, nil
}
