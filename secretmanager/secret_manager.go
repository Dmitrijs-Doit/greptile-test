package secretmanager

import (
	"context"
	"fmt"
	"strings"
	"sync"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"google.golang.org/api/iterator"

	"github.com/doitintl/hello/scheduled-tasks/common"
)

type SecretName string

// List of configured secrets in Secret Manager
const (
	SecretAppEngine             SecretName = "appengine"
	SecretASM                   SecretName = "azure-service-management"
	SecretCloudBilling          SecretName = "cloud-billing"
	SecretGoogleChannelServices SecretName = "google-channel-services"
	SecretCloudHealth           SecretName = "cloudhealth"
	SecretFirebaseDemo          SecretName = "firebase-demo"
	SecretFixer                 SecretName = "fixer"
	SecretFullstory             SecretName = "fullstory"
	SecretGoogleDrive           SecretName = "google-drive"
	SecretGSuiteReseller        SecretName = "gsuite-reseller"
	SecretMPC                   SecretName = "microsoft-partner-center"
	SecretMixpanel              SecretName = "mixpanel"
	SecretPriority              SecretName = "priority"
	SecretSendgrid              SecretName = "sendgrid"
	SecretStripeAccounts        SecretName = "stripe-accounts"
	SecretSuperQuery            SecretName = "superquery"
	SecretHubSpot               SecretName = "hubspot"
	SecretSlackBot              SecretName = "slack-bot"
	SecretSlackApp              SecretName = "slack-app"
	SecretSlackSigning          SecretName = "slack-signing"
	SecretCmpApiSignKey         SecretName = "cmp-api-sign-key"
	SecretSauronAPIKey          SecretName = "sauron-api-key"
	SecretSpot0AwsCred          SecretName = "spot0-aws-cred"
	SecretZerobounce            SecretName = "zerobounce"
	SecretAWSPayerIntegration   SecretName = "aws-payer-integration"
	SecretRootTenantHashKey     SecretName = "rootTenantHashKey"
	SecretAnnouncekit           SecretName = "announcekit"
	SecretSalesforce            SecretName = "salesforce"
	SecretMPAVaultToken         SecretName = "one-password-aws-mpa-vault-token"
	AzureLighthouseAccess       SecretName = "azure-lighthouse-access"
	SecretSalesforceSync        SecretName = "salesforce-sync"
)

const (
	latestVersion = "latest"
)

var (
	state = make(map[string][]byte)
	mutex = &sync.Mutex{}

	preload = []SecretName{
		SecretAppEngine, SecretCloudBilling, SecretSendgrid,
		SecretPriority, SecretStripeAccounts, SecretMPC, SecretASM, SecretSuperQuery,
		SecretFullstory, SecretCloudHealth, SecretFixer, SecretGSuiteReseller, SecretGoogleDrive, SecretCmpApiSignKey, SecretSpot0AwsCred, SecretHubSpot,
	}
)

func init() {
	ctx := context.Background()

	// preload to state commonly used secrets concurrently
	wg := &sync.WaitGroup{}
	wg.Add(len(preload))

	for _, secret := range preload {
		go func(ctx context.Context, secret SecretName) {
			defer wg.Done()
			AccessSecretLatestVersion(ctx, secret)
		}(ctx, secret)
	}

	wg.Wait()
}

// AccessSecretLatestVersion utility function to fetch the latest version of a secret payload
func AccessSecretLatestVersion(ctx context.Context, secret SecretName) ([]byte, error) {
	return AccessSecretVersion(ctx, string(secret), latestVersion)
}

// AccessSecretVersion fetch payload of a secret's version
func AccessSecretVersion(ctx context.Context, secret, version string) ([]byte, error) {
	name := secretResourceName(common.ProjectID, secret, version)

	mutex.Lock()
	v, prs := state[name]
	mutex.Unlock()

	if prs {
		return v, nil
	}

	sm, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	defer sm.Close()

	accessSecretVersionRes, err := sm.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	})
	if err != nil {
		return nil, err
	}

	data := accessSecretVersionRes.Payload.GetData()

	mutex.Lock()
	state[name] = data
	mutex.Unlock()

	return data, nil
}

// GetSecretLatestVersion utility function to get the secret latest version
func GetSecretLatestVersion(ctx context.Context, secret SecretName) (*secretmanagerpb.SecretVersion, error) {
	return GetSecretVersion(ctx, string(secret), latestVersion)
}

// GetSecretVersion gets a secret version
func GetSecretVersion(ctx context.Context, secret, version string) (*secretmanagerpb.SecretVersion, error) {
	name := secretResourceName(common.ProjectID, secret, version)

	sm, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	defer sm.Close()

	getSecretVersionRes, err := sm.GetSecretVersion(ctx, &secretmanagerpb.GetSecretVersionRequest{
		Name: name,
	})
	if err != nil {
		return nil, err
	}

	return getSecretVersionRes, nil
}

// AddSecretVersion creates a new secret version
func AddSecretVersion(ctx context.Context, secret SecretName, data []byte) (*secretmanagerpb.SecretVersion, error) {
	name := fmt.Sprintf("projects/%s/secrets/%s", common.ProjectID, secret)

	sm, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	defer sm.Close()

	addSecretVersionRes, err := sm.AddSecretVersion(ctx, &secretmanagerpb.AddSecretVersionRequest{
		Parent:  name,
		Payload: &secretmanagerpb.SecretPayload{Data: data},
	})
	if err != nil {
		return nil, err
	}

	return addSecretVersionRes, nil
}

func secretResourceName(projectID, secret, version string) string {
	return fmt.Sprintf("projects/%s/secrets/%s/versions/%s", projectID, secret, version)
}

func listSecretsInProject(ctx context.Context, projectID string) ([]string, error) {
	sm, err := secretmanager.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	defer sm.Close()

	iter := sm.ListSecrets(ctx, &secretmanagerpb.ListSecretsRequest{
		Parent: fmt.Sprintf("projects/%s", projectID),
	})

	if err != nil {
		return nil, err
	}

	secrets := make([]string, 0)

	for {
		secret, err := iter.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		name := secret.GetName()
		secrets = append(secrets, name[strings.LastIndex(name, "/")+1:])
	}

	return secrets, nil
}

func createSecret(ctx context.Context, projectID, secretID string, payload []byte) error {
	sm, err := secretmanager.NewClient(ctx)
	if err != nil {
		return err
	}
	defer sm.Close()

	createSecretReq := secretmanagerpb.CreateSecretRequest{
		Parent:   fmt.Sprintf("projects/%s", projectID),
		SecretId: secretID,
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
		},
	}

	secret, err := sm.CreateSecret(ctx, &createSecretReq)
	if err != nil {
		return err
	}

	addVerReq := secretmanagerpb.AddSecretVersionRequest{
		Parent: secret.GetName(),
		Payload: &secretmanagerpb.SecretPayload{
			Data: payload,
		},
	}

	secretVersion, err := sm.AddSecretVersion(ctx, &addVerReq)
	if err != nil {
		return err
	}

	_ = secretVersion

	return nil
}
